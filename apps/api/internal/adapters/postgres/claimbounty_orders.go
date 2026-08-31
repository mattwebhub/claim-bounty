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

func (s *Store) CreateOrder(ctx context.Context, write ports.IdempotentOrderWrite) (domain.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	storedVersion, err := claimVersionToInt64(write.Order.Version)
	if err != nil {
		return domain.Order{}, err
	}
	emailCiphertext, emailLookup, err := s.protectEmail(ctx, write.Order.SubmitterEmail)
	if err != nil {
		return domain.Order{}, err
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		replay, orderID, err := claimIdempotency(ctx, tx, write)
		if err != nil {
			return err
		}
		if replay {
			existing, err := s.loadClaimOrder(ctx, tx, orderID, write.ActorID.String())
			if err != nil {
				return err
			}
			write.Order = existing
			return nil
		}
		o := write.Order
		_, err = tx.Exec(ctx, `INSERT INTO claimbounty_orders(id,subject_id,submitter_email_ciphertext,submitter_email_lookup_hash,public_reference,status,version,title,purpose,target_claim_text,target_claim_location,execute_supplied_code,external_search,uploads_authorized,analysis_use_authorized,external_redistribution_authorized,customer_authorized_at,contains_participant_data,contains_direct_identifiers,terms_version,created_at,updated_at,submitted_at,retention_policy_version,retention_disposition,source_retention_expires_at,retention_expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,NULLIF($20,''),$21,$22,$23,$24,$25,$26,$27)`, o.ID.String(), o.SubjectID.String(), emailCiphertext, emailLookup[:], o.PublicReference, o.Status, storedVersion, o.Title, o.Purpose, o.TargetClaim.Text, o.TargetClaim.SourceLocation, o.Permissions.ExecuteSuppliedCode, o.Permissions.ExternalSearch, o.Authorizations.UploadsAuthorized, o.Authorizations.AnalysisUseAuthorized, o.Authorizations.ExternalRedistributionAuthorized, o.Authorizations.ConfirmedAt, o.Privacy.ContainsParticipantLevelData, o.Privacy.ContainsDirectIdentifiers, o.TermsVersion, o.CreatedAt, o.UpdatedAt, o.SubmittedAt, o.PIIRetention.PolicyVersion, o.PIIRetention.Disposition, o.PIIRetention.SourceDeleteAfter, o.PIIRetention.ApplyAfter)
		if err != nil {
			return err
		}
		if err := insertClaimIdempotency(ctx, tx, write, o.ID.String(), "", ""); err != nil {
			return err
		}
		return insertEvent(ctx, tx, o.ID, "submitter", write.ActorID.String(), "order_created", o.CreatedAt)
	})
	if errors.Is(err, domain.ErrIdempotency) {
		return domain.Order{}, err
	}
	if err != nil {
		return domain.Order{}, databaseFailure("create claimbounty order", err)
	}
	return write.Order, nil
}

func (s *Store) GetOwnedOrder(ctx context.Context, subjectID, orderID domain.Identifier) (domain.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	order, err := s.loadClaimOrder(ctx, s.pool, orderID.String(), subjectID.String())
	if errors.Is(err, domain.ErrOrderNotFound) {
		return domain.Order{}, err
	}
	if err != nil {
		return domain.Order{}, databaseFailure("get owned order", err)
	}
	return order, nil
}

func (s *Store) GetOrder(ctx context.Context, orderID domain.Identifier) (domain.Order, []domain.Export, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	order, err := s.loadClaimOrder(ctx, s.pool, orderID.String(), "")
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			return domain.Order{}, nil, err
		}
		return domain.Order{}, nil, databaseFailure("get admin order", err)
	}
	exports, err := loadExports(ctx, s.pool, orderID.String())
	if err != nil {
		return domain.Order{}, nil, databaseFailure("get order exports", err)
	}
	return order, exports, nil
}

func (s *Store) ListOrders(ctx context.Context, request ports.OrderPageRequest) (ports.OrderPage, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	if request.Limit == 0 || request.Limit > 100 {
		return ports.OrderPage{}, domain.NewValidationError(domain.FieldIssue{Field: "limit", Code: "out_of_range", Message: "must be between 1 and 100"})
	}
	var cursor claimCursor
	var err error
	if request.Cursor != "" {
		cursor, err = decodeClaimCursor(request.Cursor)
		if err != nil {
			return ports.OrderPage{}, domain.NewValidationError(domain.FieldIssue{Field: "cursor", Code: "invalid", Message: "must be a cursor returned by this endpoint"})
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,created_at FROM claimbounty_orders WHERE ($1='' OR status=$1) AND ($2::timestamptz IS NULL OR created_at>$2) AND ($3::timestamptz IS NULL OR created_at<$3) AND ($4='' OR public_reference=$4) AND ($5::timestamptz IS NULL OR (created_at,id)<($5,$6::uuid)) ORDER BY created_at DESC,id DESC LIMIT $7`, request.Status, request.CreatedAfter, request.CreatedBefore, request.PublicReference, nullableCursorTime(request.Cursor, cursor.CreatedAt), nullableCursorID(request.Cursor, cursor.ID), int(request.Limit)+1)
	if err != nil {
		return ports.OrderPage{}, databaseFailure("list orders", err)
	}
	defer rows.Close()
	type item struct {
		id        string
		createdAt time.Time
	}
	var ids []item
	for rows.Next() {
		var value item
		if err := rows.Scan(&value.id, &value.createdAt); err != nil {
			return ports.OrderPage{}, databaseFailure("list orders", err)
		}
		ids = append(ids, value)
	}
	if err := rows.Err(); err != nil {
		return ports.OrderPage{}, databaseFailure("list orders", err)
	}
	page := ports.OrderPage{}
	for index, value := range ids {
		if index == int(request.Limit) {
			break
		}
		order, err := s.loadClaimOrder(ctx, s.pool, value.id, "")
		if err != nil {
			return ports.OrderPage{}, databaseFailure("load listed order", err)
		}
		page.Orders = append(page.Orders, order)
	}
	if len(ids) > int(request.Limit) && len(page.Orders) > 0 {
		last := page.Orders[len(page.Orders)-1]
		page.NextCursor, err = encodeClaimCursor(last.CreatedAt, last.ID)
		if err != nil {
			return ports.OrderPage{}, databaseFailure("encode order cursor", err)
		}
	}
	return page, nil
}

func (s *Store) SaveOrder(ctx context.Context, order domain.Order, expected uint64) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := saveOrderRow(ctx, tx, order, expected); err != nil {
			return err
		}
		if order.Intake != nil {
			i := order.Intake
			_, err := tx.Exec(ctx, `INSERT INTO claimbounty_intakes(order_id,audit_request,scientific_policy,execution_policy,routine_revision,routine_validated_at,routine_evidence_sha256,frozen_by,frozen_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(order_id) DO UPDATE SET audit_request=EXCLUDED.audit_request,scientific_policy=EXCLUDED.scientific_policy,execution_policy=EXCLUDED.execution_policy,routine_revision=EXCLUDED.routine_revision,routine_validated_at=EXCLUDED.routine_validated_at,routine_evidence_sha256=EXCLUDED.routine_evidence_sha256,frozen_by=EXCLUDED.frozen_by,frozen_at=EXCLUDED.frozen_at`, order.ID.String(), i.AuditRequest, i.ScientificPolicy, i.ExecutionPolicy, i.RoutineRevision, i.RoutineValidatedAt, i.RoutineEvidenceSHA, i.FrozenBy.String(), i.FrozenAt)
			if err != nil {
				return err
			}
			return insertEvent(ctx, tx, order.ID, "administrator", i.FrozenBy.String(), "intake_frozen", order.UpdatedAt)
		}
		return nil
	})
	if errors.Is(err, domain.ErrVersionConflict) {
		return err
	}
	if err != nil {
		return databaseFailure("save order", err)
	}
	return nil
}

func (s *Store) SaveOrderIdempotent(ctx context.Context, write ports.IdempotentOrderWrite) (domain.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		replay, orderID, err := claimIdempotency(ctx, tx, write)
		if err != nil {
			return err
		}
		if replay {
			write.Order, err = s.loadClaimOrder(ctx, tx, orderID, write.ActorID.String())
			return err
		}
		if err := saveOrderRow(ctx, tx, write.Order, write.ExpectedVersion); err != nil {
			return err
		}
		if err := insertClaimIdempotency(ctx, tx, write, write.Order.ID.String(), "", ""); err != nil {
			return err
		}
		return insertEvent(ctx, tx, write.Order.ID, "submitter", write.ActorID.String(), write.Operation, write.Order.UpdatedAt)
	})
	if errors.Is(err, domain.ErrVersionConflict) || errors.Is(err, domain.ErrIdempotency) {
		return domain.Order{}, err
	}
	if err != nil {
		return domain.Order{}, databaseFailure("save idempotent order", err)
	}
	return write.Order, nil
}

func (s *Store) GetIdempotentOrder(ctx context.Context, actorID domain.Identifier, operation, key string, requestHash [32]byte) (domain.Order, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	var storedHash []byte
	var orderID string
	err := s.pool.QueryRow(ctx, `SELECT request_hash,order_id::text FROM claimbounty_idempotency WHERE actor_id=$1 AND operation=$2 AND idempotency_key=$3`, actorID.String(), operation, key).Scan(&storedHash, &orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, false, nil
	}
	if err != nil {
		return domain.Order{}, false, databaseFailure("get idempotent order", err)
	}
	if !bytes.Equal(storedHash, requestHash[:]) {
		return domain.Order{}, false, domain.ErrIdempotency
	}
	order, err := s.loadClaimOrder(ctx, s.pool, orderID, actorID.String())
	if err != nil {
		return domain.Order{}, false, databaseFailure("load idempotent order", err)
	}
	return order, true, nil
}

func claimIdempotency(ctx context.Context, tx pgx.Tx, write ports.IdempotentOrderWrite) (bool, string, error) {
	tag, err := tx.Exec(ctx, `INSERT INTO claimbounty_idempotency(actor_id,operation,idempotency_key,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT(actor_id,operation,idempotency_key) DO NOTHING`, write.ActorID.String(), write.Operation, write.IdempotencyKey, write.RequestHash[:])
	if err != nil {
		return false, "", err
	}
	var existingHash []byte
	var existingOrderID string
	err = tx.QueryRow(ctx, `SELECT request_hash,COALESCE(order_id::text,'') FROM claimbounty_idempotency WHERE actor_id=$1 AND operation=$2 AND idempotency_key=$3 FOR UPDATE`, write.ActorID.String(), write.Operation, write.IdempotencyKey).Scan(&existingHash, &existingOrderID)
	if err == nil {
		if !bytes.Equal(existingHash, write.RequestHash[:]) {
			return false, "", domain.ErrIdempotency
		}
		return tag.RowsAffected() == 0, existingOrderID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, "", err
	}
	return false, "", nil
}

func insertClaimIdempotency(ctx context.Context, tx pgx.Tx, write ports.IdempotentOrderWrite, orderID, fileID, exportID string) error {
	tag, err := tx.Exec(ctx, `UPDATE claimbounty_idempotency SET order_id=NULLIF($5,'')::uuid,file_id=NULLIF($6,'')::uuid,export_id=NULLIF($7,'')::uuid WHERE actor_id=$1 AND operation=$2 AND idempotency_key=$3 AND request_hash=$4`, write.ActorID.String(), write.Operation, write.IdempotencyKey, write.RequestHash[:], orderID, fileID, exportID)
	if err == nil && tag.RowsAffected() != 1 {
		return domain.ErrIdempotency
	}
	return err
}

func insertEvent(ctx context.Context, tx pgx.Tx, orderID domain.Identifier, actorKind, actorID, eventType string, createdAt time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO claimbounty_order_events(order_id,actor_kind,actor_id,event_type,created_at) VALUES($1,$2,$3,$4,$5)`, orderID.String(), actorKind, actorID, eventType, createdAt.UTC())
	return err
}
func nullableCursorTime(raw string, value time.Time) any {
	if raw == "" {
		return nil
	}
	return value
}
func nullableCursorID(raw, value string) any {
	if raw == "" {
		return nil
	}
	return value
}
