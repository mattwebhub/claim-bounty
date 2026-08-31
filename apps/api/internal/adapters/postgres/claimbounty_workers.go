package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

func (s *Store) ClaimInspection(ctx context.Context, now time.Time) (domain.Order, domain.OrderFile, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var order domain.Order
	var selected domain.OrderFile
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var orderID, fileID string
		err := tx.QueryRow(ctx, `SELECT o.order_id::text,o.file_id::text FROM claimbounty_outbox o JOIN claimbounty_orders r ON r.id=o.order_id WHERE o.kind='inspect_file' AND (o.status='pending' OR (o.status='processing' AND o.locked_at < ($1::timestamptz - interval '5 minutes'))) AND o.available_at <= $1::timestamptz AND r.status IN ('submitted','scanning') ORDER BY o.id LIMIT 1 FOR UPDATE OF o SKIP LOCKED`, now.UTC()).Scan(&orderID, &fileID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE claimbounty_outbox SET status='processing',locked_at=$3 WHERE kind='inspect_file' AND order_id=$1 AND file_id=$2`, orderID, fileID, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE claimbounty_files SET status='scanning',updated_at=$3 WHERE order_id=$1 AND id=$2 AND status IN ('uploaded','scanning')`, orderID, fileID, now.UTC()); err != nil {
			return err
		}
		order, err = s.loadClaimOrder(ctx, tx, orderID, "")
		if err != nil {
			return err
		}
		for _, file := range order.Files {
			if file.ID.String() == fileID {
				selected = file
				return nil
			}
		}
		return domain.ErrFileNotFound
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.OrderFile{}, false, nil
	}
	if err != nil {
		return domain.Order{}, domain.OrderFile{}, false, databaseFailure("claim file inspection", err)
	}
	return order, selected, true, nil
}

func (s *Store) FinishInspection(ctx context.Context, order domain.Order, file domain.OrderFile, expected uint64, source ports.ExpiredObject) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := saveOrderRow(ctx, tx, order, expected); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE claimbounty_files SET detected_media_type=NULLIF($3,''),status=$4,rejection_code=NULLIF($5,''),scanned_at=$6,updated_at=$7,storage_key=$8,storage_etag=$9,object_generation=$10 WHERE order_id=$1 AND id=$2 AND status='scanning'`, order.ID.String(), file.ID.String(), file.DetectedMediaType, file.Status, file.RejectionCode, file.ScannedAt, file.UpdatedAt, file.StorageKey, file.StorageETag, file.ObjectGeneration)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return domain.ErrStateConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE claimbounty_outbox SET status='done' WHERE kind='inspect_file' AND order_id=$1 AND file_id=$2 AND status='processing'`, order.ID.String(), file.ID.String()); err != nil {
			return err
		}
		if source.Key != "" && source.Generation != "" {
			if _, err := tx.Exec(ctx, `INSERT INTO claimbounty_outbox(kind,storage_key,object_generation,available_at) VALUES('delete_object',$1,$2,$3)`, source.Key, source.Generation, file.UpdatedAt.UTC()); err != nil {
				return err
			}
		}
		return insertEvent(ctx, tx, order.ID, "system", "inspection-worker", "file_"+file.Status, file.UpdatedAt)
	})
	if errors.Is(err, domain.ErrVersionConflict) || errors.Is(err, domain.ErrStateConflict) {
		return err
	}
	if err != nil {
		return databaseFailure("finish file inspection", err)
	}
	return nil
}

func (s *Store) ClaimExport(ctx context.Context, now time.Time) (domain.Order, domain.Export, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var order domain.Order
	var selected domain.Export
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var orderID, exportID string
		err := tx.QueryRow(ctx, `SELECT order_id::text,export_id::text FROM claimbounty_outbox WHERE kind='build_export' AND (status='pending' OR (status='processing' AND locked_at < ($1::timestamptz - interval '5 minutes'))) AND available_at <= $1::timestamptz ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`, now.UTC()).Scan(&orderID, &exportID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE claimbounty_outbox SET status='processing',locked_at=$3 WHERE kind='build_export' AND order_id=$1 AND export_id=$2`, orderID, exportID, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE claimbounty_exports SET status='building' WHERE id=$1 AND status IN ('queued','building')`, exportID); err != nil {
			return err
		}
		order, err = s.loadClaimOrder(ctx, tx, orderID, "")
		if err != nil {
			return err
		}
		exports, err := loadExports(ctx, tx, orderID)
		if err != nil {
			return err
		}
		for _, item := range exports {
			if item.ID.String() == exportID {
				selected = item
				return nil
			}
		}
		return domain.ErrExportNotFound
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.Export{}, false, nil
	}
	if err != nil {
		return domain.Order{}, domain.Export{}, false, databaseFailure("claim export", err)
	}
	return order, selected, true, nil
}

func (s *Store) FinishExport(ctx context.Context, order domain.Order, export domain.Export, expected uint64) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := saveOrderRow(ctx, tx, order, expected); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE claimbounty_exports SET status=$3,sha256=$4,size_bytes=$5,object_generation=$6,completed_at=$7 WHERE order_id=$1 AND id=$2 AND status='building'`, order.ID.String(), export.ID.String(), export.Status, export.SHA256, export.SizeBytes, export.ObjectGeneration, export.CompletedAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return domain.ErrStateConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE claimbounty_outbox SET status='done' WHERE kind='build_export' AND export_id=$1 AND status='processing'`, export.ID.String()); err != nil {
			return err
		}
		return insertEvent(ctx, tx, order.ID, "system", "export-worker", "export_ready", order.UpdatedAt)
	})
	if errors.Is(err, domain.ErrVersionConflict) || errors.Is(err, domain.ErrStateConflict) {
		return err
	}
	if err != nil {
		return databaseFailure("finish export", err)
	}
	return nil
}

func (s *Store) FailExport(ctx context.Context, orderID, exportID domain.Identifier, code string, now time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	_, err := s.pool.Exec(ctx, `WITH failed AS (UPDATE claimbounty_exports SET status='failed',failure_code=$3,completed_at=$4 WHERE order_id=$1 AND id=$2 AND status='building') UPDATE claimbounty_outbox SET status='failed',failure_code=$3 WHERE kind='build_export' AND export_id=$2 AND status='processing'`, orderID.String(), exportID.String(), code, now.UTC())
	if err != nil {
		return databaseFailure("fail export", err)
	}
	return nil
}

func (s *Store) CleanupExpiredIdentityAndAbandoned(ctx context.Context, now, abandonedBefore time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM email_challenges WHERE expires_at <= $1 OR used_at <= $1`, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM claimbounty_sessions WHERE expires_at <= $1 OR revoked_at <= $1`, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM claimbounty_rate_limits WHERE window_started_at <= $1::timestamptz - interval '24 hours'`, now.UTC()); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE claimbounty_orders SET status='expired',updated_at=$1,source_retention_expires_at=LEAST(source_retention_expires_at,$1),retention_expires_at=LEAST(retention_expires_at,$1) WHERE status IN ('draft','awaiting_email_verification','uploading') AND updated_at <= $2`, now.UTC(), abandonedBefore.UTC())
		return err
	})
	if err != nil {
		return databaseFailure("cleanup expired identity and abandoned orders", err)
	}
	return nil
}

func (s *Store) AnonymizeExpired(ctx context.Context, now time.Time, limit int) ([]ports.ExpiredObject, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	if limit < 1 {
		return nil, nil
	}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		sourceRows, err := tx.Query(ctx, `SELECT id::text FROM claimbounty_orders o WHERE source_retention_expires_at<=$1 AND source_deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM claimbounty_outbox d WHERE d.kind='delete_object' AND d.retention_order_id=o.id) ORDER BY source_retention_expires_at LIMIT $2 FOR UPDATE OF o SKIP LOCKED`, now.UTC(), limit)
		if err != nil {
			return err
		}
		var sourceIDs []string
		for sourceRows.Next() {
			var id string
			if err := sourceRows.Scan(&id); err != nil {
				sourceRows.Close()
				return err
			}
			sourceIDs = append(sourceIDs, id)
		}
		if err := sourceRows.Err(); err != nil {
			sourceRows.Close()
			return err
		}
		sourceRows.Close()
		for _, id := range sourceIDs {
			tag, err := tx.Exec(ctx, `INSERT INTO claimbounty_outbox(kind,storage_key,object_generation,retention_order_id,available_at) SELECT 'delete_object',storage_key,object_generation,$1::uuid,$2::timestamptz FROM claimbounty_files WHERE order_id=$1 AND object_generation IS NOT NULL UNION ALL SELECT 'delete_object',storage_key,object_generation,$1::uuid,$2::timestamptz FROM claimbounty_exports WHERE order_id=$1 AND object_generation IS NOT NULL`, id, now.UTC())
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				if _, err := tx.Exec(ctx, `UPDATE claimbounty_orders SET source_deleted_at=$2 WHERE id=$1 AND source_deleted_at IS NULL`, id, now.UTC()); err != nil {
					return err
				}
			}
		}
		rows, err := tx.Query(ctx, `SELECT id::text,subject_id::text,submitter_email_lookup_hash,status FROM claimbounty_orders WHERE retention_expires_at<=$1 AND status IN ('exported','rejected','cancelled','expired') ORDER BY retention_expires_at LIMIT $2 FOR UPDATE SKIP LOCKED`, now.UTC(), limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		type item struct {
			id, subjectID, status string
			emailLookup           []byte
		}
		var items []item
		for rows.Next() {
			var value item
			if err := rows.Scan(&value.id, &value.subjectID, &value.emailLookup, &value.status); err != nil {
				return err
			}
			items = append(items, value)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, value := range items {
			if _, err := tx.Exec(ctx, `INSERT INTO claimbounty_outbox(kind,storage_key,object_generation,retention_order_id,available_at) SELECT 'delete_object',objects.storage_key,objects.object_generation,$1::uuid,$2::timestamptz FROM (SELECT storage_key,object_generation FROM claimbounty_files WHERE order_id=$1 AND object_generation IS NOT NULL UNION ALL SELECT storage_key,object_generation FROM claimbounty_exports WHERE order_id=$1 AND object_generation IS NOT NULL) objects WHERE EXISTS(SELECT 1 FROM claimbounty_orders WHERE id=$1 AND source_deleted_at IS NULL) AND NOT EXISTS(SELECT 1 FROM claimbounty_outbox WHERE kind='delete_object' AND retention_order_id=$1)`, value.id, now.UTC()); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO claimbounty_tombstones(order_id,final_status,erased_at) VALUES($1,$2,$3)`, value.id, value.status, now.UTC()); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM claimbounty_sessions WHERE (subject_id=$1 OR email_lookup_hash=$2) AND NOT EXISTS (SELECT 1 FROM claimbounty_orders WHERE id<>$3 AND (subject_id=$1 OR submitter_email_lookup_hash=$2))`, value.subjectID, value.emailLookup, value.id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM email_challenges WHERE (subject_id=$1 OR email_lookup_hash=$2) AND NOT EXISTS (SELECT 1 FROM claimbounty_orders WHERE id<>$3 AND (subject_id=$1 OR submitter_email_lookup_hash=$2))`, value.subjectID, value.emailLookup, value.id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM claimbounty_orders WHERE id=$1`, value.id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, databaseFailure("anonymize expired orders", err)
	}
	return nil, nil
}
