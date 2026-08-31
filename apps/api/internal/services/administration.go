package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

const routineID = "claim-bounty-operations/run-claimbounty-scientific-audit"

type OrderPageRequest = ports.OrderPageRequest
type OrderPage = ports.OrderPage
type ObjectReader = io.ReadCloser

type AdminIntakeCommand struct {
	Session            domain.Session
	OrderID            domain.Identifier
	ExpectedVersion    uint64
	AuditRequest       []byte
	ScientificPolicy   []byte
	ExecutionPolicy    []byte
	RoutineRevision    string
	RoutineValidatedAt time.Time
	RoutineEvidenceSHA string
}

type TrustedRoutineContract struct {
	Revision       string
	ValidatedAt    time.Time
	EvidenceSHA256 string
}

type CreateExportCommand struct {
	Session                domain.Session
	OrderID                domain.Identifier
	ExpectedVersion        uint64
	RetentionPolicyVersion string
	PreserveRunOutputs     bool
	IdempotencyKey         string
	RequestHash            [32]byte
}

type AdministrationService struct {
	repository ports.IntakeRepository
	storage    ports.PrivateObjectStore
	validator  ports.IntakeValidator
	policy     ports.AdminPolicy
	values     ports.SecureValues
	clock      ports.Clock
	routine    TrustedRoutineContract
}

func NewAdministrationService(repository ports.IntakeRepository, storage ports.PrivateObjectStore, validator ports.IntakeValidator, policy ports.AdminPolicy, values ports.SecureValues, clock ports.Clock, routine TrustedRoutineContract) (*AdministrationService, error) {
	if repository == nil || storage == nil || validator == nil || policy == nil || values == nil || clock == nil {
		return nil, ErrInvalidDependencies
	}
	if len(routine.Revision) != len("sha256:")+64 || routine.ValidatedAt.IsZero() || len(routine.EvidenceSHA256) != 64 {
		return nil, ErrInvalidDependencies
	}
	return &AdministrationService{repository: repository, storage: storage, validator: validator, policy: policy, values: values, clock: clock, routine: routine}, nil
}

func (service *AdministrationService) authorize(ctx context.Context, session domain.Session, revision string) error {
	if session.Audience != domain.AdminAudience {
		return domain.ErrForbidden
	}
	if err := service.policy.Authorize(ctx, session.Email, session.AuthorizationPolicyVersion); err != nil {
		return err
	}
	if revision != "" && len(revision) != len("sha256:")+64 {
		return domain.ErrForbidden
	}
	return nil
}

func (service *AdministrationService) ListOrders(ctx context.Context, session domain.Session, request OrderPageRequest) (OrderPage, error) {
	if err := service.authorize(ctx, session, ""); err != nil {
		return ports.OrderPage{}, err
	}
	return service.repository.ListOrders(ctx, request)
}

func (service *AdministrationService) GetOrder(ctx context.Context, session domain.Session, orderID domain.Identifier) (domain.Order, []domain.Export, error) {
	if err := service.authorize(ctx, session, ""); err != nil {
		return domain.Order{}, nil, err
	}
	return service.repository.GetOrder(ctx, orderID)
}

func (service *AdministrationService) UpdateIntake(ctx context.Context, command AdminIntakeCommand) (domain.Order, []domain.Export, error) {
	if err := service.authorize(ctx, command.Session, command.RoutineRevision); err != nil {
		return domain.Order{}, nil, err
	}
	if command.RoutineRevision != service.routine.Revision || !command.RoutineValidatedAt.Equal(service.routine.ValidatedAt) || command.RoutineEvidenceSHA != service.routine.EvidenceSHA256 {
		return domain.Order{}, nil, domain.NewValidationError(domain.FieldIssue{Field: "routineContract", Code: "invalid", Message: "must exactly match the trusted configured routine registry entry"})
	}
	now := service.clock.Now()
	if command.RoutineValidatedAt.IsZero() || command.RoutineValidatedAt.After(now) {
		return domain.Order{}, nil, domain.NewValidationError(domain.FieldIssue{Field: "routineContract.validation.validatedAt", Code: "invalid", Message: "must be a completed validation timestamp"})
	}
	if err := service.validator.ValidateAuditRequest(command.AuditRequest); err != nil {
		return domain.Order{}, nil, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest", Code: "invalid", Message: "must match schema version 1.0.0"})
	}
	if err := service.validator.ValidateScientificPolicy(command.ScientificPolicy); err != nil {
		return domain.Order{}, nil, domain.NewValidationError(domain.FieldIssue{Field: "scientificPolicy", Code: "invalid", Message: "must match schema version 1.0.0"})
	}
	if err := service.validator.ValidateExecutionPolicy(command.ExecutionPolicy); err != nil {
		return domain.Order{}, nil, domain.NewValidationError(domain.FieldIssue{Field: "executionPolicy", Code: "invalid", Message: "must match schema version 1.0.0"})
	}
	order, exports, err := service.repository.GetOrder(ctx, command.OrderID)
	if err != nil {
		return domain.Order{}, nil, err
	}
	command.AuditRequest, err = freezeAuditAuthority(command.AuditRequest, order, command.Session, service.policy, now)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if err := service.validator.ValidateAuditRequest(command.AuditRequest); err != nil {
		return domain.Order{}, nil, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest", Code: "invalid", Message: "server-owned authority fields produced an invalid audit request"})
	}
	if err := validatePolicyAuthority(command.AuditRequest, command.ExecutionPolicy, order); err != nil {
		return domain.Order{}, nil, err
	}
	retention, err := auditRetention(command.AuditRequest, order)
	if err != nil {
		return domain.Order{}, nil, err
	}
	order.PIIRetention = retention
	updated, err := order.FreezeIntake(command.ExpectedVersion, domain.AdminIntake{
		AuditRequest: command.AuditRequest, ScientificPolicy: command.ScientificPolicy,
		ExecutionPolicy: command.ExecutionPolicy, RoutineRevision: command.RoutineRevision,
		RoutineValidatedAt: command.RoutineValidatedAt, RoutineEvidenceSHA: command.RoutineEvidenceSHA,
		FrozenBy: command.Session.SubjectID,
	}, now)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if err := service.repository.SaveOrder(ctx, updated, command.ExpectedVersion); err != nil {
		return domain.Order{}, nil, err
	}
	return updated, exports, nil
}

func (service *AdministrationService) CreateExport(ctx context.Context, command CreateExportCommand) (domain.Export, error) {
	if err := validateIdempotency(command.IdempotencyKey); err != nil {
		return domain.Export{}, err
	}
	if err := service.authorize(ctx, command.Session, ""); err != nil {
		return domain.Export{}, err
	}
	if replay, ok, err := service.repository.GetIdempotentExport(ctx, command.Session.SubjectID, command.IdempotencyKey, command.RequestHash); err != nil {
		return domain.Export{}, err
	} else if ok {
		return replay, nil
	}
	order, _, err := service.repository.GetOrder(ctx, command.OrderID)
	if err != nil {
		return domain.Export{}, err
	}
	if order.Intake == nil {
		return domain.Export{}, domain.ErrStateConflict
	}
	if err := service.authorize(ctx, command.Session, order.Intake.RoutineRevision); err != nil {
		return domain.Export{}, err
	}
	if !domain.ValidPolicyVersion(command.RetentionPolicyVersion) {
		return domain.Export{}, domain.NewValidationError(domain.FieldIssue{Field: "retentionPolicyVersion", Code: "invalid_format", Message: "must be a version identifier"})
	}
	if err := validateExportPolicy(order.Intake.AuditRequest, command.RetentionPolicyVersion, command.PreserveRunOutputs); err != nil {
		return domain.Export{}, err
	}
	updated, err := order.QueueExport(command.ExpectedVersion, service.clock.Now())
	if err != nil {
		return domain.Export{}, err
	}
	id, err := service.values.NewIdentifier(ctx)
	if err != nil {
		return domain.Export{}, err
	}
	key, err := service.values.NewObjectKey(ctx, "exports")
	if err != nil {
		return domain.Export{}, err
	}
	export := domain.Export{ID: id, OrderID: order.ID, Status: "queued", RoutineID: routineID, RoutineRevision: order.Intake.RoutineRevision, RoutineValidatedAt: order.Intake.RoutineValidatedAt, RoutineEvidenceSHA: order.Intake.RoutineEvidenceSHA, RetentionPolicy: command.RetentionPolicyVersion, PreserveRunOutputs: command.PreserveRunOutputs, StorageKey: key, CreatedAt: service.clock.Now()}
	return service.repository.QueueExport(ctx, ports.ExportQueueRequest{Order: updated, Export: export, ActorID: command.Session.SubjectID, IdempotencyKey: command.IdempotencyKey, RequestHash: command.RequestHash})
}

func (service *AdministrationService) GetExport(ctx context.Context, session domain.Session, orderID, exportID domain.Identifier) (domain.Export, error) {
	if err := service.authorize(ctx, session, ""); err != nil {
		return domain.Export{}, err
	}
	return service.repository.GetExport(ctx, orderID, exportID)
}

func (service *AdministrationService) OpenFile(ctx context.Context, session domain.Session, orderID, fileID domain.Identifier) (ObjectReader, domain.OrderFile, error) {
	if err := service.authorize(ctx, session, ""); err != nil {
		return nil, domain.OrderFile{}, err
	}
	order, _, err := service.repository.GetOrder(ctx, orderID)
	if err != nil {
		return nil, domain.OrderFile{}, err
	}
	file, ok := findFile(order, fileID)
	if !ok {
		return nil, domain.OrderFile{}, domain.ErrFileNotFound
	}
	if file.Status != "clean" || file.ObjectGeneration == "" {
		return nil, domain.OrderFile{}, domain.ErrFileNotClean
	}
	reader, metadata, err := service.storage.Open(ctx, file.StorageKey, file.ObjectGeneration)
	if err != nil {
		return nil, domain.OrderFile{}, err
	}
	if metadata.SizeBytes != file.SizeBytes || metadata.SHA256 != file.SHA256 || metadata.Generation != file.ObjectGeneration {
		_ = reader.Close()
		return nil, domain.OrderFile{}, domain.ErrFileNotClean
	}
	return reader, file, nil
}

func (service *AdministrationService) OpenExport(ctx context.Context, session domain.Session, exportID domain.Identifier) (ObjectReader, domain.Export, error) {
	if err := service.authorize(ctx, session, ""); err != nil {
		return nil, domain.Export{}, err
	}
	export, err := service.repository.GetExport(ctx, "", exportID)
	if err != nil {
		return nil, domain.Export{}, err
	}
	if export.Status != "ready" || export.ObjectGeneration == "" {
		return nil, domain.Export{}, domain.ErrExportNotReady
	}
	reader, metadata, err := service.storage.Open(ctx, export.StorageKey, export.ObjectGeneration)
	if err != nil {
		return nil, domain.Export{}, err
	}
	if metadata.Generation != export.ObjectGeneration || metadata.SHA256 != export.SHA256 || metadata.SizeBytes != export.SizeBytes {
		_ = reader.Close()
		return nil, domain.Export{}, domain.ErrExportNotReady
	}
	return reader, export, nil
}

func auditRetention(document []byte, order domain.Order) (domain.PIIRetention, error) {
	var audit struct {
		SchemaVersion string `json:"schemaVersion"`
		CaseID        string `json:"caseId"`
		Purpose       string `json:"purpose"`
		TargetClaim   struct {
			Text   string `json:"text"`
			Source struct {
				Location string `json:"location"`
			} `json:"source"`
		} `json:"targetClaim"`
		Authority struct {
			TermsVersion string `json:"termsVersion"`
		} `json:"authority"`
		Retention struct {
			PolicyVersion     string    `json:"policyVersion"`
			SourceDeleteAfter time.Time `json:"sourceDeleteAfter"`
			PIIDeleteAfter    time.Time `json:"piiDeleteAfter"`
			PIIDisposition    string    `json:"piiDisposition"`
		} `json:"retention"`
	}
	if err := json.Unmarshal(document, &audit); err != nil || audit.CaseID != order.ID.String() || audit.Purpose != order.Purpose || audit.TargetClaim.Text != order.TargetClaim.Text || (order.TargetClaim.SourceLocation != "" && audit.TargetClaim.Source.Location != order.TargetClaim.SourceLocation) || audit.Authority.TermsVersion != order.TermsVersion {
		return domain.PIIRetention{}, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest", Code: "invalid", Message: "must match the frozen customer order"})
	}
	if audit.Retention.SourceDeleteAfter.IsZero() || audit.Retention.PIIDeleteAfter.IsZero() || audit.Retention.SourceDeleteAfter.After(audit.Retention.PIIDeleteAfter) {
		return domain.PIIRetention{}, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest.retention", Code: "invalid", Message: "source deletion must occur no later than PII deletion"})
	}
	if audit.Retention.PolicyVersion != order.PIIRetention.PolicyVersion {
		return domain.PIIRetention{}, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest.retention.policyVersion", Code: "snapshot_mismatch", Message: "must preserve the server policy version frozen at customer submission"})
	}
	if audit.Retention.PIIDisposition != order.PIIRetention.Disposition {
		return domain.PIIRetention{}, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest.retention.piiDisposition", Code: "snapshot_mismatch", Message: "must preserve the server disposition frozen at customer submission"})
	}
	if audit.Retention.SourceDeleteAfter.After(order.PIIRetention.SourceDeleteAfter) || audit.Retention.PIIDeleteAfter.After(order.PIIRetention.ApplyAfter) {
		return domain.PIIRetention{}, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest.retention", Code: "exceeds_server_ceiling", Message: "administrator deadlines may preserve or shorten, but never extend, the customer-frozen server deadlines"})
	}
	return domain.PIIRetention{PolicyVersion: audit.Retention.PolicyVersion, Disposition: audit.Retention.PIIDisposition, SourceDeleteAfter: audit.Retention.SourceDeleteAfter, ApplyAfter: audit.Retention.PIIDeleteAfter}, nil
}

func freezeAuditAuthority(raw []byte, order domain.Order, session domain.Session, policy ports.AdminPolicy, now time.Time) ([]byte, error) {
	var snapshot struct {
		CaseID      string `json:"caseId"`
		Purpose     string `json:"purpose"`
		TargetClaim struct {
			Text   string `json:"text"`
			Source struct {
				Artifact string `json:"artifact"`
				Location string `json:"location"`
			} `json:"source"`
		} `json:"targetClaim"`
		Permissions struct {
			ExecuteSuppliedCode bool `json:"executeSuppliedCode"`
			ExternalSearch      bool `json:"externalSearch"`
		} `json:"permissions"`
		Privacy struct {
			ContainsParticipantLevelData bool `json:"containsParticipantLevelData"`
			ContainsDirectIdentifiers    bool `json:"containsDirectIdentifiers"`
		} `json:"privacy"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest", Code: "invalid", Message: "must be valid JSON"})
	}
	primaryPath := ""
	for _, file := range order.Files {
		if file.Role == "primary_paper" {
			primaryPath = file.CaseBundlePath()
		}
	}
	if snapshot.CaseID != order.ID.String() || snapshot.Purpose != order.Purpose || snapshot.TargetClaim.Text != order.TargetClaim.Text || snapshot.TargetClaim.Source.Artifact != primaryPath || (order.TargetClaim.SourceLocation != "" && snapshot.TargetClaim.Source.Location != order.TargetClaim.SourceLocation) || snapshot.Permissions.ExecuteSuppliedCode != order.Permissions.ExecuteSuppliedCode || snapshot.Permissions.ExternalSearch != order.Permissions.ExternalSearch || snapshot.Privacy.ContainsParticipantLevelData != order.Privacy.ContainsParticipantLevelData || snapshot.Privacy.ContainsDirectIdentifiers != order.Privacy.ContainsDirectIdentifiers || order.SubmittedAt == nil {
		return nil, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest", Code: "snapshot_mismatch", Message: "must exactly preserve the submitted order and primary file"})
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	authority, ok := document["authority"].(map[string]any)
	if !ok {
		return nil, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest.authority", Code: "required", Message: "must be provided"})
	}
	authority["uploadsAuthorized"] = order.Authorizations.UploadsAuthorized
	authority["analysisUseAuthorized"] = order.Authorizations.AnalysisUseAuthorized
	authority["externalRedistributionAuthorized"] = false
	authority["termsVersion"] = order.TermsVersion
	if order.Authorizations.ConfirmedAt == nil {
		return nil, domain.NewValidationError(domain.FieldIssue{Field: "authority", Code: "missing_customer_grant", Message: "customer authorizations must be persisted before intake"})
	}
	authority["customerConfirmedAt"] = order.Authorizations.ConfirmedAt.UTC().Format(time.RFC3339Nano)
	authority["frozenBy"] = session.SubjectID.String()
	authority["frozenAt"] = now.UTC().Format(time.RFC3339Nano)
	authority["authorizationPolicyVersion"] = policy.Version()
	authority["adminAllowlistVersion"] = policy.AllowlistVersion()
	permissions, ok := document["permissions"].(map[string]any)
	if !ok {
		return nil, domain.NewValidationError(domain.FieldIssue{Field: "auditRequest.permissions", Code: "required", Message: "must be provided"})
	}
	permissions["externalRedistributionAuthorized"] = false
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, errors.New("service: audit authority encoding failed")
	}
	return encoded, nil
}

func validatePolicyAuthority(auditRaw, executionRaw []byte, order domain.Order) error {
	var audit struct {
		SchemaVersion string `json:"schemaVersion"`
		Permissions   struct {
			ReadUploadedFiles                bool `json:"readUploadedFiles"`
			ExecuteSuppliedCode              bool `json:"executeSuppliedCode"`
			CreateDerivedFiles               bool `json:"createDerivedFiles"`
			ExternalSearch                   bool `json:"externalSearch"`
			OpenAccessSourcesOnly            bool `json:"openAccessSourcesOnly"`
			ExternalRedistributionAuthorized bool `json:"externalRedistributionAuthorized"`
		} `json:"permissions"`
	}
	var execution struct {
		SchemaVersion string `json:"schemaVersion"`
		Resources     struct {
			MaximumCPUCores          int `json:"maximumCpuCores"`
			MaximumMemoryMiB         int `json:"maximumMemoryMiB"`
			MaximumWorkingStorageMiB int `json:"maximumWorkingStorageMiB"`
		} `json:"resources"`
		SourceAccess struct {
			ExternalSearch bool `json:"externalSearch"`
			OpenAccessOnly bool `json:"openAccessOnly"`
		} `json:"sourceAccess"`
	}
	if json.Unmarshal(auditRaw, &audit) != nil || json.Unmarshal(executionRaw, &execution) != nil {
		return domain.NewValidationError(domain.FieldIssue{Field: "policies", Code: "invalid", Message: "must be valid JSON"})
	}
	if audit.SchemaVersion == "" || execution.SchemaVersion == "" {
		return nil
	}
	permissions := audit.Permissions
	if !permissions.ReadUploadedFiles || !permissions.CreateDerivedFiles || permissions.ExecuteSuppliedCode != order.Permissions.ExecuteSuppliedCode || permissions.ExternalSearch != order.Permissions.ExternalSearch || !permissions.OpenAccessSourcesOnly || permissions.ExternalRedistributionAuthorized {
		return domain.NewValidationError(domain.FieldIssue{Field: "auditRequest.permissions", Code: "exceeds_customer_grant", Message: "must match customer grants and P0 redistribution restrictions"})
	}
	if execution.SourceAccess.ExternalSearch != order.Permissions.ExternalSearch || !execution.SourceAccess.OpenAccessOnly || execution.Resources.MaximumCPUCores > 4 || execution.Resources.MaximumMemoryMiB > 8192 || execution.Resources.MaximumWorkingStorageMiB > 5120 {
		return domain.NewValidationError(domain.FieldIssue{Field: "executionPolicy", Code: "exceeds_server_ceiling", Message: "must match customer grants and server resource ceilings"})
	}
	return nil
}

func validateExportPolicy(raw []byte, version string, preserve bool) error {
	var document struct {
		Retention struct {
			PolicyVersion      string `json:"policyVersion"`
			PreserveRunOutputs bool   `json:"preserveRunOutputs"`
		} `json:"retention"`
	}
	if err := json.Unmarshal(raw, &document); err != nil || document.Retention.PolicyVersion != version || document.Retention.PreserveRunOutputs != preserve {
		return domain.NewValidationError(domain.FieldIssue{Field: "retentionPolicyVersion", Code: "snapshot_mismatch", Message: "must match the frozen audit request retention policy"})
	}
	return nil
}

var _ = errors.Is
