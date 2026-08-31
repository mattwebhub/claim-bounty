package services

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

type CreateOrderCommand struct {
	Session        domain.Session
	Title          string
	Purpose        string
	TargetClaim    domain.TargetClaim
	Permissions    domain.Permissions
	Privacy        domain.Privacy
	IdempotencyKey string
	RequestHash    [32]byte
}

type UploadFileCommand struct {
	Session             domain.Session
	OrderID             domain.Identifier
	ExpectedVersion     uint64
	Role                string
	OriginalDisplayName string
	SizeBytes           int64
	SHA256              string
	DeclaredMediaType   string
	IdempotencyKey      string
	RequestHash         [32]byte
	Body                io.Reader
}

type UploadedFile struct {
	Order domain.Order
	File  domain.OrderFile
}

type SubmitOrderCommand struct {
	Session                          domain.Session
	OrderID                          domain.Identifier
	ExpectedVersion                  uint64
	TermsAccepted                    bool
	TermsVersion                     string
	UploadsAuthorized                bool
	AnalysisUseAuthorized            bool
	ExternalRedistributionAuthorized bool
	IdempotencyKey                   string
	RequestHash                      [32]byte
}

type DeleteFileCommand struct {
	Session         domain.Session
	OrderID         domain.Identifier
	FileID          domain.Identifier
	ExpectedVersion uint64
}

type RetentionContract struct {
	PolicyVersion  string
	SourceDuration time.Duration
	PIIDuration    time.Duration
}

type IntakeService struct {
	repository ports.IntakeRepository
	storage    ports.PrivateObjectStore
	values     ports.SecureValues
	clock      ports.Clock
	retention  RetentionContract
}

func NewIntakeService(repository ports.IntakeRepository, storage ports.PrivateObjectStore, values ports.SecureValues, clock ports.Clock, retention RetentionContract) (*IntakeService, error) {
	if repository == nil || storage == nil || values == nil || clock == nil {
		return nil, ErrInvalidDependencies
	}
	if !domain.ValidPolicyVersion(retention.PolicyVersion) || retention.SourceDuration <= 0 || retention.PIIDuration <= 0 || retention.SourceDuration > retention.PIIDuration {
		return nil, ErrInvalidDependencies
	}
	return &IntakeService{repository: repository, storage: storage, values: values, clock: clock, retention: retention}, nil
}

func (service *IntakeService) retentionAt(now time.Time) domain.PIIRetention {
	now = now.UTC()
	return domain.PIIRetention{
		PolicyVersion:     service.retention.PolicyVersion,
		Disposition:       "hard_delete",
		SourceDeleteAfter: now.Add(service.retention.SourceDuration),
		ApplyAfter:        now.Add(service.retention.PIIDuration),
	}
}

func (service *IntakeService) CreateOrder(ctx context.Context, command CreateOrderCommand) (domain.Order, error) {
	if command.Session.Audience != domain.SubmitterAudience {
		return domain.Order{}, domain.ErrForbidden
	}
	if err := validateIdempotency(command.IdempotencyKey); err != nil {
		return domain.Order{}, err
	}
	id, err := service.values.NewIdentifier(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	reference, err := service.values.NewPublicReference(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	now := service.clock.Now()
	order, err := domain.NewOrder(id, command.Session.SubjectID, command.Session.Email, reference, command.Title, command.Purpose, command.TargetClaim, command.Permissions, command.Privacy, now)
	if err != nil {
		return domain.Order{}, err
	}
	order.PIIRetention = service.retentionAt(now)
	return service.repository.CreateOrder(ctx, ports.IdempotentOrderWrite{Order: order, ActorID: command.Session.SubjectID, Operation: "create_order", IdempotencyKey: command.IdempotencyKey, RequestHash: command.RequestHash})
}

func (service *IntakeService) GetOwnedOrder(ctx context.Context, session domain.Session, orderID domain.Identifier) (domain.Order, error) {
	if session.Audience != domain.SubmitterAudience {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return service.repository.GetOwnedOrder(ctx, session.SubjectID, orderID)
}

func (service *IntakeService) UploadFile(ctx context.Context, command UploadFileCommand) (UploadedFile, error) {
	if command.Session.Audience != domain.SubmitterAudience {
		return UploadedFile{}, domain.ErrForbidden
	}
	if err := validateIdempotency(command.IdempotencyKey); err != nil {
		return UploadedFile{}, err
	}
	if command.Body == nil {
		return UploadedFile{}, domain.NewValidationError(domain.FieldIssue{Field: "file", Code: "required", Message: "must be provided"})
	}
	if replayOrder, replayFile, ok, err := service.repository.GetIdempotentFile(ctx, command.Session.SubjectID, command.IdempotencyKey, command.RequestHash); err != nil {
		return UploadedFile{}, err
	} else if ok {
		return UploadedFile{Order: replayOrder, File: replayFile}, nil
	}
	order, err := service.repository.GetOwnedOrder(ctx, command.Session.SubjectID, command.OrderID)
	if err != nil {
		return UploadedFile{}, err
	}
	var total int64
	for _, file := range order.Files {
		total += file.SizeBytes
	}
	if total+command.SizeBytes > 1<<30 {
		return UploadedFile{}, domain.NewValidationError(domain.FieldIssue{Field: "sizeBytes", Code: "out_of_range", Message: "order files may total at most 1 GiB"})
	}
	fileID, err := service.values.NewIdentifier(ctx)
	if err != nil {
		return UploadedFile{}, err
	}
	objectKey, err := service.values.NewObjectKey(ctx, "quarantine")
	if err != nil {
		return UploadedFile{}, err
	}
	now := service.clock.Now()
	file, err := domain.NewOrderFile(fileID, command.Role, command.OriginalDisplayName, command.SizeBytes, command.SHA256, command.DeclaredMediaType, objectKey, now)
	if err != nil {
		return UploadedFile{}, err
	}
	metadata, err := service.storage.PutWriteOnce(ctx, file.StorageKey, io.LimitReader(command.Body, file.SizeBytes+1), file.SizeBytes, file.DeclaredMediaType, file.SHA256)
	if err != nil {
		return UploadedFile{}, err
	}
	if metadata.SizeBytes != file.SizeBytes || metadata.SHA256 != file.SHA256 || metadata.Generation == "" {
		_ = service.storage.DeleteVersion(ctx, file.StorageKey, metadata.Generation)
		return UploadedFile{}, domain.NewValidationError(domain.FieldIssue{Field: "file", Code: "invalid", Message: "uploaded bytes do not match size and SHA-256"})
	}
	file, err = file.ConfirmUpload(metadata.ETag, metadata.Generation, now)
	if err != nil {
		_ = service.storage.DeleteVersion(ctx, file.StorageKey, metadata.Generation)
		return UploadedFile{}, err
	}
	updated, err := order.ReserveFile(file, command.ExpectedVersion, now)
	if err != nil {
		_ = service.storage.DeleteVersion(ctx, file.StorageKey, metadata.Generation)
		return UploadedFile{}, err
	}
	savedOrder, savedFile, err := service.repository.SaveUploadedFile(ctx, ports.UploadedFileWrite{
		Write: ports.IdempotentOrderWrite{Order: updated, ExpectedVersion: command.ExpectedVersion, ActorID: command.Session.SubjectID, Operation: "reserve_upload", IdempotencyKey: command.IdempotencyKey, RequestHash: command.RequestHash},
		File:  file,
	})
	if err != nil {
		_ = service.storage.DeleteVersion(ctx, file.StorageKey, metadata.Generation)
		return UploadedFile{}, err
	}
	if savedFile.StorageKey != file.StorageKey || savedFile.ObjectGeneration != file.ObjectGeneration {
		if err := service.storage.DeleteVersion(ctx, file.StorageKey, metadata.Generation); err != nil {
			return UploadedFile{}, err
		}
	}
	return UploadedFile{Order: savedOrder, File: savedFile}, nil
}

func (service *IntakeService) SubmitOrder(ctx context.Context, command SubmitOrderCommand) (domain.Order, error) {
	if command.Session.Audience != domain.SubmitterAudience {
		return domain.Order{}, domain.ErrForbidden
	}
	if err := validateIdempotency(command.IdempotencyKey); err != nil {
		return domain.Order{}, err
	}
	if replay, ok, err := service.repository.GetIdempotentOrder(ctx, command.Session.SubjectID, "submit_order", command.IdempotencyKey, command.RequestHash); err != nil {
		return domain.Order{}, err
	} else if ok {
		return replay, nil
	}
	order, err := service.repository.GetOwnedOrder(ctx, command.Session.SubjectID, command.OrderID)
	if err != nil {
		return domain.Order{}, err
	}
	now := service.clock.Now()
	updated, err := order.SubmitWithRetention(command.ExpectedVersion, command.TermsVersion, command.TermsAccepted, command.UploadsAuthorized, command.AnalysisUseAuthorized, command.ExternalRedistributionAuthorized, service.retentionAt(now), now)
	if err != nil {
		return domain.Order{}, err
	}
	return service.repository.SaveOrderIdempotent(ctx, ports.IdempotentOrderWrite{Order: updated, ExpectedVersion: command.ExpectedVersion, ActorID: command.Session.SubjectID, Operation: "submit_order", IdempotencyKey: command.IdempotencyKey, RequestHash: command.RequestHash})
}

func (service *IntakeService) DeleteFile(ctx context.Context, command DeleteFileCommand) error {
	if command.Session.Audience != domain.SubmitterAudience {
		return domain.ErrOrderNotFound
	}
	order, err := service.repository.GetOwnedOrder(ctx, command.Session.SubjectID, command.OrderID)
	if err != nil {
		return err
	}
	updated, removed, err := order.RemoveFile(command.FileID, command.ExpectedVersion, service.clock.Now())
	if err != nil {
		return err
	}
	return service.repository.RemoveFile(ctx, updated, removed, command.ExpectedVersion, command.Session.SubjectID)
}

func findFile(order domain.Order, id domain.Identifier) (domain.OrderFile, bool) {
	for _, file := range order.Files {
		if file.ID == id {
			return file, true
		}
	}
	return domain.OrderFile{}, false
}

func validateIdempotency(value string) error {
	if len(value) < 16 || len(value) > 128 || strings.ContainsAny(value, " \t\r\n") {
		return domain.NewValidationError(domain.FieldIssue{Field: "Idempotency-Key", Code: "invalid", Message: "must be 16 to 128 non-whitespace characters"})
	}
	return nil
}
