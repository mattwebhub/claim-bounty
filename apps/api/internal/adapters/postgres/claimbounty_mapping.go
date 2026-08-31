package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

type claimQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *Store) loadClaimOrder(ctx context.Context, query claimQuerier, orderID string, subjectID string) (domain.Order, error) {
	statement := `SELECT id::text,subject_id::text,submitter_email_ciphertext,public_reference,status,version,title,purpose,target_claim_text,target_claim_location,execute_supplied_code,external_search,uploads_authorized,analysis_use_authorized,external_redistribution_authorized,customer_authorized_at,contains_participant_data,contains_direct_identifiers,COALESCE(terms_version,''),created_at,updated_at,submitted_at,retention_policy_version,retention_disposition,source_retention_expires_at,retention_expires_at FROM claimbounty_orders WHERE id=$1`
	arguments := []any{orderID}
	if subjectID != "" {
		statement += ` AND subject_id=$2`
		arguments = append(arguments, subjectID)
	}
	var rawID, rawSubject string
	var order domain.Order
	var version int64
	var emailCiphertext []byte
	var submittedAt *time.Time
	err := query.QueryRow(ctx, statement, arguments...).Scan(&rawID, &rawSubject, &emailCiphertext, &order.PublicReference, &order.Status, &version, &order.Title, &order.Purpose, &order.TargetClaim.Text, &order.TargetClaim.SourceLocation, &order.Permissions.ExecuteSuppliedCode, &order.Permissions.ExternalSearch, &order.Authorizations.UploadsAuthorized, &order.Authorizations.AnalysisUseAuthorized, &order.Authorizations.ExternalRedistributionAuthorized, &order.Authorizations.ConfirmedAt, &order.Privacy.ContainsParticipantLevelData, &order.Privacy.ContainsDirectIdentifiers, &order.TermsVersion, &order.CreatedAt, &order.UpdatedAt, &submittedAt, &order.PIIRetention.PolicyVersion, &order.PIIRetention.Disposition, &order.PIIRetention.SourceDeleteAfter, &order.PIIRetention.ApplyAfter)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}
	order.SubmitterEmail, err = s.revealEmail(ctx, emailCiphertext)
	if err != nil {
		return domain.Order{}, err
	}
	id, err := domain.NewIdentifier(rawID)
	if err != nil {
		return domain.Order{}, err
	}
	subject, err := domain.NewIdentifier(rawSubject)
	if err != nil {
		return domain.Order{}, err
	}
	if version <= 0 || version > int64(domain.MaxOrderVersion) {
		return domain.Order{}, errors.New("invalid order version")
	}
	restoredVersion, err := storedClaimVersion(version)
	if err != nil {
		return domain.Order{}, err
	}
	order.ID, order.SubjectID, order.Version, order.SubmittedAt = id, subject, restoredVersion, submittedAt
	rows, err := query.Query(ctx, `SELECT id::text,role,original_display_name,size_bytes,sha256,declared_media_type,COALESCE(detected_media_type,''),status,COALESCE(rejection_code,''),storage_key,COALESCE(storage_etag,''),COALESCE(object_generation,''),scanned_at,created_at,updated_at FROM claimbounty_files WHERE order_id=$1 ORDER BY created_at,id`, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var rawFileID string
		var file domain.OrderFile
		if err := rows.Scan(&rawFileID, &file.Role, &file.OriginalDisplayName, &file.SizeBytes, &file.SHA256, &file.DeclaredMediaType, &file.DetectedMediaType, &file.Status, &file.RejectionCode, &file.StorageKey, &file.StorageETag, &file.ObjectGeneration, &file.ScannedAt, &file.CreatedAt, &file.UpdatedAt); err != nil {
			return domain.Order{}, err
		}
		file.ID, err = domain.NewIdentifier(rawFileID)
		if err != nil {
			return domain.Order{}, err
		}
		order.Files = append(order.Files, file)
	}
	if err := rows.Err(); err != nil {
		return domain.Order{}, err
	}
	eventRows, err := query.Query(ctx, `SELECT id::text,actor_kind,actor_id,event_type,metadata,created_at FROM claimbounty_order_events WHERE order_id=$1 ORDER BY created_at,id`, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	defer eventRows.Close()
	for eventRows.Next() {
		var rawEventID string
		var event domain.OrderEvent
		if err := eventRows.Scan(&rawEventID, &event.ActorKind, &event.ActorID, &event.Type, &event.Metadata, &event.CreatedAt); err != nil {
			return domain.Order{}, err
		}
		event.ID, err = domain.NewIdentifier(rawEventID)
		if err != nil {
			return domain.Order{}, err
		}
		order.Events = append(order.Events, event)
	}
	if err := eventRows.Err(); err != nil {
		return domain.Order{}, err
	}
	var audit, scientific, execution []byte
	var intake domain.AdminIntake
	err = query.QueryRow(ctx, `SELECT audit_request,scientific_policy,execution_policy,routine_revision,routine_validated_at,routine_evidence_sha256,frozen_by::text,frozen_at FROM claimbounty_intakes WHERE order_id=$1`, orderID).Scan(&audit, &scientific, &execution, &intake.RoutineRevision, &intake.RoutineValidatedAt, &intake.RoutineEvidenceSHA, &rawID, &intake.FrozenAt)
	if err == nil {
		intake.AuditRequest, intake.ScientificPolicy, intake.ExecutionPolicy = audit, scientific, execution
		intake.FrozenBy, err = domain.NewIdentifier(rawID)
		if err != nil {
			return domain.Order{}, err
		}
		order.Intake = &intake
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, err
	}
	return domain.RestoreOrder(order)
}

func loadExports(ctx context.Context, query claimQuerier, orderID string) ([]domain.Export, error) {
	statement := `SELECT id::text,order_id::text,status,routine_id,routine_revision,routine_validated_at,routine_evidence_sha256,retention_policy_version,preserve_run_outputs,COALESCE(sha256,''),COALESCE(size_bytes,0),storage_key,COALESCE(object_generation,''),COALESCE(failure_code,''),created_at,completed_at FROM claimbounty_exports`
	args := []any{}
	if orderID != "" {
		statement += ` WHERE order_id=$1`
		args = append(args, orderID)
	}
	statement += ` ORDER BY created_at,id`
	rows, err := query.Query(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var exports []domain.Export
	for rows.Next() {
		var item domain.Export
		var id, rawOrderID string
		if err := rows.Scan(&id, &rawOrderID, &item.Status, &item.RoutineID, &item.RoutineRevision, &item.RoutineValidatedAt, &item.RoutineEvidenceSHA, &item.RetentionPolicy, &item.PreserveRunOutputs, &item.SHA256, &item.SizeBytes, &item.StorageKey, &item.ObjectGeneration, &item.FailureCode, &item.CreatedAt, &item.CompletedAt); err != nil {
			return nil, err
		}
		item.ID, err = domain.NewIdentifier(id)
		if err != nil {
			return nil, err
		}
		item.OrderID, err = domain.NewIdentifier(rawOrderID)
		if err != nil {
			return nil, err
		}
		exports = append(exports, item)
	}
	return exports, rows.Err()
}

type claimCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

func encodeClaimCursor(createdAt time.Time, id domain.Identifier) (string, error) {
	raw, err := json.Marshal(claimCursor{CreatedAt: createdAt.UTC(), ID: id.String()})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func decodeClaimCursor(raw string) (claimCursor, error) {
	value, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return claimCursor{}, err
	}
	var cursor claimCursor
	if err := json.Unmarshal(value, &cursor); err != nil {
		return claimCursor{}, err
	}
	if cursor.CreatedAt.IsZero() {
		return claimCursor{}, errors.New("missing cursor time")
	}
	if _, err := domain.NewIdentifier(cursor.ID); err != nil {
		return claimCursor{}, err
	}
	return cursor, nil
}

func saveOrderRow(ctx context.Context, query claimQuerier, order domain.Order, expected uint64) error {
	expectedVersion, err := claimVersionToInt64(expected)
	if err != nil {
		return err
	}
	newVersion, err := claimVersionToInt64(order.Version)
	if err != nil {
		return err
	}
	row := query.QueryRow(ctx, `UPDATE claimbounty_orders SET status=$3,version=$4,terms_version=NULLIF($5,''),updated_at=$6,submitted_at=$7,retention_policy_version=$8,retention_disposition=$9,source_retention_expires_at=$10,retention_expires_at=$11,uploads_authorized=$12,analysis_use_authorized=$13,external_redistribution_authorized=$14,customer_authorized_at=$15 WHERE id=$1 AND version=$2 RETURNING version`, order.ID.String(), expectedVersion, order.Status, newVersion, order.TermsVersion, order.UpdatedAt.UTC(), order.SubmittedAt, order.PIIRetention.PolicyVersion, order.PIIRetention.Disposition, order.PIIRetention.SourceDeleteAfter, order.PIIRetention.ApplyAfter, order.Authorizations.UploadsAuthorized, order.Authorizations.AnalysisUseAuthorized, order.Authorizations.ExternalRedistributionAuthorized, order.Authorizations.ConfirmedAt)
	var version int64
	if err := row.Scan(&version); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrVersionConflict
	} else if err != nil {
		return err
	}
	persistedVersion, err := storedClaimVersion(version)
	if err != nil {
		return err
	}
	if persistedVersion != order.Version {
		return fmt.Errorf("persisted unexpected version")
	}
	return nil
}

func claimVersionToInt64(value uint64) (int64, error) {
	if value == 0 || value > domain.MaxOrderVersion {
		return 0, errors.New("invalid order version")
	}
	return int64(value), nil
}

func storedClaimVersion(value int64) (uint64, error) {
	if value <= 0 || value > int64(domain.MaxOrderVersion) {
		return 0, errors.New("invalid persisted order version")
	}
	return uint64(value), nil
}
