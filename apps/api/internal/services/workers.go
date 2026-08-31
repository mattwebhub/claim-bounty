package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

type Workers struct {
	repository ports.IntakeRepository
	storage    ports.PrivateObjectStore
	inspector  ports.FileInspector
	builder    ports.ExportBuilder
	clock      ports.Clock
	interval   time.Duration
	logger     *slog.Logger
}

func NewWorkers(repository ports.IntakeRepository, storage ports.PrivateObjectStore, inspector ports.FileInspector, builder ports.ExportBuilder, clock ports.Clock, interval time.Duration, loggers ...*slog.Logger) (*Workers, error) {
	if repository == nil || storage == nil || inspector == nil || builder == nil || clock == nil {
		return nil, ErrInvalidDependencies
	}
	if interval <= 0 {
		interval = time.Second
	}
	logger := slog.New(slog.DiscardHandler)
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &Workers{repository: repository, storage: storage, inspector: inspector, builder: builder, clock: clock, interval: interval, logger: logger}, nil
}

func (workers *Workers) Run(ctx context.Context) error {
	ticker := time.NewTicker(workers.interval)
	defer ticker.Stop()
	for {
		if err := workers.step(ctx); err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return nil
			}
			workers.logger.ErrorContext(ctx, "background worker step failed; retrying", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (workers *Workers) step(ctx context.Context) error {
	if err := workers.inspectOne(ctx); err != nil {
		return err
	}
	if err := workers.exportOne(ctx); err != nil {
		return err
	}
	if err := workers.deleteOne(ctx); err != nil {
		return err
	}
	objects, err := workers.repository.AnonymizeExpired(ctx, workers.clock.Now(), 25)
	if err != nil {
		return err
	}
	for _, object := range objects {
		if err := workers.storage.DeleteVersion(ctx, object.Key, object.Generation); err != nil {
			return err
		}
	}
	return nil
}

func (workers *Workers) deleteOne(ctx context.Context) error {
	id, object, ok, err := workers.repository.ClaimDeletion(ctx, workers.clock.Now())
	if err != nil || !ok {
		return err
	}
	if err := workers.storage.DeleteVersion(ctx, object.Key, object.Generation); err != nil {
		return err
	}
	return workers.repository.FinishDeletion(ctx, id, workers.clock.Now())
}

func (workers *Workers) inspectOne(ctx context.Context) error {
	order, file, ok, err := workers.repository.ClaimInspection(ctx, workers.clock.Now())
	if err != nil || !ok {
		return err
	}
	reader, metadata, err := workers.storage.Open(ctx, file.StorageKey, file.ObjectGeneration)
	if err != nil {
		return err
	}
	defer reader.Close()
	if metadata.Generation != file.ObjectGeneration || metadata.SizeBytes != file.SizeBytes || metadata.SHA256 != file.SHA256 {
		return workers.rejectFile(ctx, order, file, "object_integrity_mismatch")
	}
	detected, clean, rejection, err := workers.inspector.Inspect(ctx, io.LimitReader(reader, file.SizeBytes+1), file.SizeBytes, file.DeclaredMediaType)
	if err != nil {
		return err
	}
	if clean && !file.AcceptsDetectedMediaType(detected) {
		clean, rejection = false, "media_signature_mismatch"
	}
	source := ports.ExpiredObject{Key: file.StorageKey, Generation: file.ObjectGeneration}
	inspected, err := file.InspectionResult(clean, detected, rejection, workers.clock.Now())
	if err != nil {
		return err
	}
	if clean {
		reader, reopenedMetadata, err := workers.storage.Open(ctx, source.Key, source.Generation)
		if err != nil {
			return err
		}
		acceptedKey := "accepted/sha256/" + file.SHA256 + "/" + file.ID.String()
		promoted, promoteErr := workers.storage.PutWriteOnce(ctx, acceptedKey, io.LimitReader(reader, file.SizeBytes+1), file.SizeBytes, detected, file.SHA256)
		closeErr := reader.Close()
		if promoteErr != nil {
			return promoteErr
		}
		if closeErr != nil || reopenedMetadata.Generation != source.Generation || promoted.Generation == "" {
			return errors.New("service: object promotion integrity check failed")
		}
		inspected.StorageKey, inspected.StorageETag, inspected.ObjectGeneration = acceptedKey, promoted.ETag, promoted.Generation
	}
	updated, err := order.ReplaceFile(inspected, order.Version, workers.clock.Now())
	if err != nil {
		return err
	}
	if updated.SubmittedAt != nil {
		updated, err = updated.ReconcileInspection(workers.clock.Now())
		if err != nil {
			return err
		}
	}
	return workers.repository.FinishInspection(ctx, updated, inspected, order.Version, source)
}

func (workers *Workers) rejectFile(ctx context.Context, order domain.Order, file domain.OrderFile, code string) error {
	rejected, err := file.InspectionResult(false, "application/octet-stream", code, workers.clock.Now())
	if err != nil {
		return err
	}
	updated, err := order.ReplaceFile(rejected, order.Version, workers.clock.Now())
	if err != nil {
		return err
	}
	if updated.SubmittedAt != nil {
		updated, err = updated.ReconcileInspection(workers.clock.Now())
		if err != nil {
			return err
		}
	}
	return workers.repository.FinishInspection(ctx, updated, rejected, order.Version, ports.ExpiredObject{Key: file.StorageKey, Generation: file.ObjectGeneration})
}

func (workers *Workers) exportOne(ctx context.Context) error {
	order, export, ok, err := workers.repository.ClaimExport(ctx, workers.clock.Now())
	if err != nil || !ok {
		return err
	}
	metadata, err := workers.builder.Build(ctx, order, export)
	if err != nil {
		return workers.repository.FailExport(ctx, order.ID, export.ID, "build_failed", workers.clock.Now())
	}
	export.Status, export.SHA256, export.SizeBytes, export.ObjectGeneration = "ready", metadata.SHA256, metadata.SizeBytes, metadata.Generation
	completed := workers.clock.Now()
	export.CompletedAt = &completed
	updated, err := order.ExportReady(completed)
	if err != nil {
		return err
	}
	return workers.repository.FinishExport(ctx, updated, export, order.Version)
}
