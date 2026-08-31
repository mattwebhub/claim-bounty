package postgres

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

func (s *Store) GetIdempotentFile(ctx context.Context, actorID domain.Identifier, key string, requestHash [32]byte) (domain.Order, domain.OrderFile, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var storedHash []byte
	var orderID, fileID string
	err := s.pool.QueryRow(ctx, `SELECT request_hash,order_id::text,file_id::text FROM claimbounty_idempotency WHERE actor_id=$1 AND operation='reserve_upload' AND idempotency_key=$2`, actorID.String(), key).Scan(&storedHash, &orderID, &fileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.OrderFile{}, false, nil
	}
	if err != nil {
		return domain.Order{}, domain.OrderFile{}, false, databaseFailure("get idempotent upload", err)
	}
	if !bytes.Equal(storedHash, requestHash[:]) {
		return domain.Order{}, domain.OrderFile{}, false, domain.ErrIdempotency
	}
	order, err := s.loadClaimOrder(ctx, s.pool, orderID, actorID.String())
	if err != nil {
		return domain.Order{}, domain.OrderFile{}, false, databaseFailure("get idempotent upload order", err)
	}
	for _, file := range order.Files {
		if file.ID.String() == fileID {
			return order, file, true, nil
		}
	}
	return domain.Order{}, domain.OrderFile{}, false, domain.ErrFileNotFound
}

func (s *Store) SaveUploadedFile(ctx context.Context, request ports.UploadedFileWrite) (domain.Order, domain.OrderFile, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	resultOrder, resultFile := request.Write.Order, request.File
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		replay, orderID, err := claimIdempotency(ctx, tx, request.Write)
		if err != nil {
			return err
		}
		if replay {
			resultOrder, err = s.loadClaimOrder(ctx, tx, orderID, request.Write.ActorID.String())
			if err != nil {
				return err
			}
			var persistedFileID string
			if err := tx.QueryRow(ctx, `SELECT file_id::text FROM claimbounty_idempotency WHERE actor_id=$1 AND operation=$2 AND idempotency_key=$3`, request.Write.ActorID.String(), request.Write.Operation, request.Write.IdempotencyKey).Scan(&persistedFileID); err != nil {
				return err
			}
			for _, file := range resultOrder.Files {
				if file.ID.String() == persistedFileID {
					resultFile = file
					return nil
				}
			}
			return domain.ErrFileNotFound
		}
		if err := saveOrderRow(ctx, tx, request.Write.Order, request.Write.ExpectedVersion); err != nil {
			return err
		}
		f := request.File
		_, err = tx.Exec(ctx, `INSERT INTO claimbounty_files(id,order_id,role,original_display_name,size_bytes,sha256,declared_media_type,detected_media_type,status,rejection_code,storage_key,storage_etag,object_generation,scanned_at,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,NULLIF($10,''),$11,NULLIF($12,''),NULLIF($13,''),$14,$15,$16)`, f.ID.String(), request.Write.Order.ID.String(), f.Role, f.OriginalDisplayName, f.SizeBytes, f.SHA256, f.DeclaredMediaType, f.DetectedMediaType, f.Status, f.RejectionCode, f.StorageKey, f.StorageETag, f.ObjectGeneration, f.ScannedAt, f.CreatedAt, f.UpdatedAt)
		if err != nil {
			return err
		}
		if err := insertClaimIdempotency(ctx, tx, request.Write, request.Write.Order.ID.String(), f.ID.String(), ""); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO claimbounty_outbox(kind,order_id,file_id,available_at) VALUES('inspect_file',$1,$2,$3)`, request.Write.Order.ID.String(), f.ID.String(), f.UpdatedAt.UTC()); err != nil {
			return err
		}
		return insertEvent(ctx, tx, request.Write.Order.ID, "submitter", request.Write.ActorID.String(), "file_uploaded", request.Write.Order.UpdatedAt)
	})
	if errors.Is(err, domain.ErrVersionConflict) || errors.Is(err, domain.ErrIdempotency) {
		return domain.Order{}, domain.OrderFile{}, err
	}
	if err != nil {
		return domain.Order{}, domain.OrderFile{}, databaseFailure("save uploaded file", err)
	}
	return resultOrder, resultFile, nil
}

func (s *Store) RemoveFile(ctx context.Context, order domain.Order, removed domain.OrderFile, expected uint64, actorID domain.Identifier) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := saveOrderRow(ctx, tx, order, expected); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM claimbounty_files WHERE id=$1 AND order_id=$2`, removed.ID.String(), order.ID.String())
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return domain.ErrFileNotFound
		}
		if removed.ObjectGeneration != "" {
			if _, err := tx.Exec(ctx, `INSERT INTO claimbounty_outbox(kind,storage_key,object_generation,available_at) VALUES('delete_object',$1,$2,$3)`, removed.StorageKey, removed.ObjectGeneration, order.UpdatedAt.UTC()); err != nil {
				return err
			}
		}
		return insertEvent(ctx, tx, order.ID, "submitter", actorID.String(), "file_removed", order.UpdatedAt)
	})
	if errors.Is(err, domain.ErrVersionConflict) || errors.Is(err, domain.ErrFileNotFound) {
		return err
	}
	if err != nil {
		return databaseFailure("remove file", err)
	}
	return nil
}

func (s *Store) QueueExport(ctx context.Context, request ports.ExportQueueRequest) (domain.Export, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	result := request.Export
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		write := ports.IdempotentOrderWrite{Order: request.Order, ExpectedVersion: request.Order.Version - 1, ActorID: request.ActorID, Operation: "create_export", IdempotencyKey: request.IdempotencyKey, RequestHash: request.RequestHash}
		replay, _, err := claimIdempotency(ctx, tx, write)
		if err != nil {
			return err
		}
		if replay {
			var exportID string
			if err := tx.QueryRow(ctx, `SELECT export_id::text FROM claimbounty_idempotency WHERE actor_id=$1 AND operation=$2 AND idempotency_key=$3`, request.ActorID.String(), "create_export", request.IdempotencyKey).Scan(&exportID); err != nil {
				return err
			}
			exports, err := loadExports(ctx, tx, request.Order.ID.String())
			if err != nil {
				return err
			}
			for _, item := range exports {
				if item.ID.String() == exportID {
					result = item
					return nil
				}
			}
			return domain.ErrExportNotFound
		}
		var active bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM claimbounty_exports WHERE order_id=$1 AND status IN ('queued','building','ready'))`, request.Order.ID.String()).Scan(&active); err != nil {
			return err
		}
		if active {
			return domain.ErrStateConflict
		}
		if err := saveOrderRow(ctx, tx, request.Order, request.Order.Version-1); err != nil {
			return err
		}
		e := request.Export
		_, err = tx.Exec(ctx, `INSERT INTO claimbounty_exports(id,order_id,status,routine_id,routine_revision,routine_validated_at,routine_evidence_sha256,retention_policy_version,preserve_run_outputs,storage_key,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, e.ID.String(), e.OrderID.String(), e.Status, e.RoutineID, e.RoutineRevision, e.RoutineValidatedAt, e.RoutineEvidenceSHA, e.RetentionPolicy, e.PreserveRunOutputs, e.StorageKey, e.CreatedAt)
		if err != nil {
			if constraint(err, "claimbounty_exports_one_active_idx") {
				return domain.ErrStateConflict
			}
			return err
		}
		if err := insertClaimIdempotency(ctx, tx, write, e.OrderID.String(), "", e.ID.String()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO claimbounty_outbox(kind,order_id,export_id,available_at) VALUES('build_export',$1,$2,$3)`, e.OrderID.String(), e.ID.String(), e.CreatedAt.UTC()); err != nil {
			return err
		}
		return insertEvent(ctx, tx, e.OrderID, "administrator", request.ActorID.String(), "export_queued", e.CreatedAt)
	})
	if errors.Is(err, domain.ErrVersionConflict) || errors.Is(err, domain.ErrIdempotency) || errors.Is(err, domain.ErrStateConflict) {
		return domain.Export{}, err
	}
	if err != nil {
		return domain.Export{}, databaseFailure("queue export", err)
	}
	return result, nil
}

func (s *Store) GetIdempotentExport(ctx context.Context, actorID domain.Identifier, key string, requestHash [32]byte) (domain.Export, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var storedHash []byte
	var exportID string
	err := s.pool.QueryRow(ctx, `SELECT request_hash,export_id::text FROM claimbounty_idempotency WHERE actor_id=$1 AND operation='create_export' AND idempotency_key=$2`, actorID.String(), key).Scan(&storedHash, &exportID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Export{}, false, nil
	}
	if err != nil {
		return domain.Export{}, false, databaseFailure("get idempotent export", err)
	}
	if !bytes.Equal(storedHash, requestHash[:]) {
		return domain.Export{}, false, domain.ErrIdempotency
	}
	exportIDValue, err := domain.NewIdentifier(exportID)
	if err != nil {
		return domain.Export{}, false, databaseFailure("restore idempotent export", err)
	}
	export, err := s.GetExport(ctx, "", exportIDValue)
	if err != nil {
		return domain.Export{}, false, err
	}
	return export, true, nil
}

func (s *Store) GetExport(ctx context.Context, orderID, exportID domain.Identifier) (domain.Export, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var resolvedOrderID string
	if orderID != "" {
		resolvedOrderID = orderID.String()
	} else {
		err := s.pool.QueryRow(ctx, `SELECT order_id::text FROM claimbounty_exports WHERE id=$1`, exportID.String()).Scan(&resolvedOrderID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Export{}, domain.ErrExportNotFound
		}
		if err != nil {
			return domain.Export{}, databaseFailure("get export", err)
		}
	}
	exports, err := loadExports(ctx, s.pool, resolvedOrderID)
	if err != nil {
		return domain.Export{}, databaseFailure("get export", err)
	}
	for _, item := range exports {
		if item.ID == exportID {
			return item, nil
		}
	}
	return domain.Export{}, domain.ErrExportNotFound
}

func (s *Store) ClaimDeletion(ctx context.Context, now time.Time) (int64, ports.ExpiredObject, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var id int64
	var object ports.ExpiredObject
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT id,storage_key,object_generation FROM claimbounty_outbox WHERE kind='delete_object' AND (status='pending' OR (status='processing' AND locked_at < ($1::timestamptz - interval '5 minutes'))) AND available_at <= $1::timestamptz ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`, now.UTC()).Scan(&id, &object.Key, &object.Generation)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE claimbounty_outbox SET status='processing',locked_at=$2 WHERE id=$1`, id, now.UTC())
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ports.ExpiredObject{}, false, nil
	}
	if err != nil {
		return 0, ports.ExpiredObject{}, false, databaseFailure("claim object deletion", err)
	}
	return id, object, true, nil
}

func (s *Store) FinishDeletion(ctx context.Context, id int64, completedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var retentionOrderID string
		err := tx.QueryRow(ctx, `UPDATE claimbounty_outbox SET status='done' WHERE id=$1 AND status='processing' RETURNING COALESCE(retention_order_id::text,'')`, id).Scan(&retentionOrderID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrStateConflict
		}
		if err != nil || retentionOrderID == "" {
			return err
		}
		var remaining bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM claimbounty_outbox WHERE kind='delete_object' AND retention_order_id=$1 AND status<>'done')`, retentionOrderID).Scan(&remaining); err != nil {
			return err
		}
		if !remaining {
			_, err = tx.Exec(ctx, `UPDATE claimbounty_orders SET source_deleted_at=$2 WHERE id=$1 AND source_deleted_at IS NULL`, retentionOrderID, completedAt.UTC())
		}
		return err
	})
	if errors.Is(err, domain.ErrStateConflict) {
		return err
	}
	if err != nil {
		return databaseFailure("finish object deletion", err)
	}
	return nil
}
