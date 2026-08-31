package domain

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	MaxOrderVersion uint64 = 9007199254740991
	MaxOrderFiles          = 20
	MaxFileBytes    int64  = 250 * 1024 * 1024
	MaxOrderBytes   int64  = 1 << 30
)

var (
	uuidPattern           = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	referencePattern      = regexp.MustCompile(`^CB-[A-Z0-9]{12}$`)
	sha256Pattern         = regexp.MustCompile(`^[a-f0-9]{64}$`)
	versionPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,99}$`)
	mediaTypePattern      = regexp.MustCompile(`^[A-Za-z0-9!#$&^_.+-]+/[A-Za-z0-9!#$&^_.+-]+(?:\s*;[^\r\n]*)?$`)
	allowedFileRoles      = set("primary_paper", "supplement", "preregistration", "data", "code", "environment", "data_dictionary", "other_evidence")
	allowedRoleExtensions = map[string]map[string]bool{
		"primary_paper":   set(".pdf"),
		"supplement":      set(".docx", ".md", ".pdf", ".txt"),
		"preregistration": set(".docx", ".md", ".pdf", ".txt"),
		"data":            set(".csv", ".dta", ".json", ".parquet", ".rdata", ".rds", ".sav", ".tsv", ".xlsx"),
		"code":            set(".do", ".ipynb", ".py", ".r", ".sh", ".sql", ".zip"),
		"environment":     set("", ".lock", ".toml", ".txt", ".yaml", ".yml"),
		"data_dictionary": set(".csv", ".md", ".pdf", ".txt", ".xlsx"),
		"other_evidence":  set(".md", ".pdf", ".txt", ".zip"),
	}
	allowedFileStates = set("upload_pending", "uploaded", "scanning", "clean", "rejected", "expired")
	allowedOrderState = set("draft", "awaiting_email_verification", "uploading", "submitted", "scanning", "needs_information", "ready_for_export", "exported", "rejected", "cancelled", "expired")
)

type Identifier string

func NewIdentifier(value string) (Identifier, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if !uuidPattern.MatchString(value) {
		return "", errors.New("identifier must be a UUID")
	}
	return Identifier(value), nil
}

func (id Identifier) String() string { return string(id) }

type Audience string

const (
	SubmitterAudience Audience = "submitter"
	AdminAudience     Audience = "administrator"
)

func NewAudience(value string) (Audience, error) {
	audience := Audience(value)
	if audience != SubmitterAudience && audience != AdminAudience {
		return "", NewValidationError(FieldIssue{Field: "audience", Code: "invalid", Message: "must be submitter or administrator"})
	}
	return audience, nil
}

type Session struct {
	ID                         Identifier
	SubjectID                  Identifier
	Email                      string
	Audience                   Audience
	AuthorizationPolicyVersion string
	ExpiresAt                  time.Time
}

func NewSession(id, subjectID Identifier, email string, audience Audience, policyVersion string, expiresAt time.Time) (Session, error) {
	if id == "" || subjectID == "" || expiresAt.IsZero() || !versionPattern.MatchString(policyVersion) {
		return Session{}, errors.New("session identity and expiry are required")
	}
	if _, err := NewAudience(string(audience)); err != nil {
		return Session{}, err
	}
	return Session{ID: id, SubjectID: subjectID, Email: email, Audience: audience, AuthorizationPolicyVersion: policyVersion, ExpiresAt: expiresAt.UTC()}, nil
}

type TargetClaim struct {
	Text           string
	SourceLocation string
}

type Permissions struct {
	ExecuteSuppliedCode bool
	ExternalSearch      bool
}

type Privacy struct {
	ContainsParticipantLevelData bool
	ContainsDirectIdentifiers    bool
}

type PIIRetention struct {
	PolicyVersion     string
	Disposition       string
	SourceDeleteAfter time.Time
	ApplyAfter        time.Time
}

type OrderFile struct {
	ID                  Identifier
	Role                string
	OriginalDisplayName string
	SizeBytes           int64
	SHA256              string
	DeclaredMediaType   string
	DetectedMediaType   string
	Status              string
	RejectionCode       string
	StorageKey          string
	StorageETag         string
	ObjectGeneration    string
	ScannedAt           *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func NewOrderFile(id Identifier, role, displayName string, size int64, sha256, declaredType, storageKey string, now time.Time) (OrderFile, error) {
	issues := validateFile(role, displayName, size, sha256, declaredType)
	if storageKey == "" {
		issues = append(issues, FieldIssue{Field: "storageKey", Code: "required", Message: "must be provided"})
	}
	if err := NewValidationError(issues...); err != nil {
		return OrderFile{}, err
	}
	return OrderFile{ID: id, Role: role, OriginalDisplayName: displayName, SizeBytes: size, SHA256: sha256, DeclaredMediaType: declaredType, Status: "upload_pending", StorageKey: storageKey, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}, nil
}

func RestoreOrderFile(file OrderFile) (OrderFile, error) {
	if file.ID == "" || !allowedFileStates[file.Status] || file.CreatedAt.IsZero() || file.UpdatedAt.Before(file.CreatedAt) {
		return OrderFile{}, errors.New("invalid persisted order file")
	}
	if err := NewValidationError(validateFile(file.Role, file.OriginalDisplayName, file.SizeBytes, file.SHA256, file.DeclaredMediaType)...); err != nil {
		return OrderFile{}, err
	}
	return file, nil
}

func (f OrderFile) ConfirmUpload(etag, generation string, now time.Time) (OrderFile, error) {
	if f.Status != "upload_pending" {
		return OrderFile{}, ErrStateConflict
	}
	if strings.TrimSpace(etag) == "" || len(etag) > 200 || strings.TrimSpace(generation) == "" || len(generation) > 200 {
		return OrderFile{}, NewValidationError(FieldIssue{Field: "storageETag", Code: "invalid", Message: "must identify the completed object"})
	}
	f.Status, f.StorageETag, f.ObjectGeneration, f.UpdatedAt = "uploaded", etag, generation, now.UTC()
	return f, nil
}

func (f OrderFile) InspectionResult(clean bool, detectedType, rejection string, now time.Time) (OrderFile, error) {
	if f.Status != "uploaded" && f.Status != "scanning" {
		return OrderFile{}, ErrStateConflict
	}
	if clean {
		if !mediaTypePattern.MatchString(detectedType) {
			return OrderFile{}, NewValidationError(FieldIssue{Field: "detectedMediaType", Code: "invalid_format", Message: "must be a media type"})
		}
		f.Status, f.DetectedMediaType, f.RejectionCode = "clean", detectedType, ""
		scannedAt := now.UTC()
		f.ScannedAt = &scannedAt
	} else {
		if strings.TrimSpace(rejection) == "" || len(rejection) > 100 {
			return OrderFile{}, NewValidationError(FieldIssue{Field: "rejectionCode", Code: "invalid", Message: "must be a stable rejection code"})
		}
		f.Status, f.DetectedMediaType, f.RejectionCode = "rejected", detectedType, rejection
	}
	f.UpdatedAt = now.UTC()
	return f, nil
}

// AcceptsDetectedMediaType binds the scanner signature result to the file role
// and extension accepted at intake. It prevents a declared text or data file
// from becoming clean when its bytes identify an executable or unrelated type.
func (f OrderFile) AcceptsDetectedMediaType(detected string) bool {
	detected = strings.ToLower(strings.TrimSpace(strings.SplitN(detected, ";", 2)[0]))
	extension := strings.ToLower(path.Ext(f.OriginalDisplayName))
	accepted := map[string]map[string]bool{
		".pdf":     set("application/pdf"),
		".docx":    set("application/zip", "application/x-zip-compressed", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"),
		".xlsx":    set("application/zip", "application/x-zip-compressed", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"),
		".zip":     set("application/zip", "application/x-zip-compressed"),
		".gz":      set("application/gzip", "application/x-gzip"),
		".json":    set("application/json", "text/plain"),
		".jsonl":   set("application/json", "application/x-ndjson", "text/plain"),
		".ipynb":   set("application/json", "text/plain"),
		".csv":     set("text/csv", "text/plain", "application/csv"),
		".tsv":     set("text/tab-separated-values", "text/plain"),
		".md":      set("text/markdown", "text/plain"),
		".txt":     set("text/plain"),
		".r":       set("text/plain", "text/x-r-source"),
		".py":      set("text/plain", "text/x-python"),
		".do":      set("text/plain"),
		".sh":      set("text/plain", "text/x-shellscript", "application/x-sh"),
		".sql":     set("text/plain", "application/sql"),
		".yaml":    set("text/plain", "application/yaml", "text/yaml"),
		".yml":     set("text/plain", "application/yaml", "text/yaml"),
		".xml":     set("application/xml", "text/xml", "text/plain"),
		".xls":     set("application/octet-stream", "application/vnd.ms-excel"),
		".rdata":   set("application/octet-stream", "application/x-r-data"),
		".rds":     set("application/octet-stream", "application/gzip", "application/x-gzip", "application/x-r-data"),
		".dta":     set("application/octet-stream", "application/x-stata"),
		".sav":     set("application/octet-stream", "application/x-spss-sav"),
		".feather": set("application/octet-stream", "application/vnd.apache.arrow.file"),
		".parquet": set("application/octet-stream", "application/vnd.apache.parquet"),
		".lock":    set("text/plain", "application/json"),
		".toml":    set("text/plain", "application/toml"),
	}
	if !allowedFileName(f.Role, f.OriginalDisplayName) {
		return false
	}
	if types, ok := accepted[extension]; ok {
		return types[detected]
	}
	if !allowedRoleExtensions[f.Role][extension] {
		return false
	}
	switch f.Role {
	case "code", "environment":
		return detected == "text/plain" || detected == "application/json"
	case "data", "data_dictionary":
		return detected == "text/plain" || detected == "application/octet-stream"
	}
	return false
}

type Order struct {
	ID              Identifier
	SubjectID       Identifier
	SubmitterEmail  string
	PublicReference string
	Status          string
	Version         uint64
	Title           string
	Purpose         string
	TargetClaim     TargetClaim
	Permissions     Permissions
	Authorizations  CustomerAuthorizations
	Privacy         Privacy
	PIIRetention    PIIRetention
	Files           []OrderFile
	TermsVersion    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	SubmittedAt     *time.Time
	Intake          *AdminIntake
	Events          []OrderEvent
}

type OrderEvent struct {
	ID        Identifier
	ActorKind string
	ActorID   string
	Type      string
	Metadata  []byte
	CreatedAt time.Time
}

func NewOrder(id, subjectID Identifier, email, reference, title, purpose string, claim TargetClaim, permissions Permissions, privacy Privacy, now time.Time) (Order, error) {
	issues := validateOrder(reference, title, purpose, claim)
	if id == "" || subjectID == "" {
		issues = append(issues, FieldIssue{Field: "id", Code: "required", Message: "identity must be provided"})
	}
	if err := NewValidationError(issues...); err != nil {
		return Order{}, err
	}
	deadline := now.UTC().Add(30 * 24 * time.Hour)
	return Order{ID: id, SubjectID: subjectID, SubmitterEmail: email, PublicReference: reference, Status: "draft", Version: 1, Title: title, Purpose: purpose, TargetClaim: claim, Permissions: permissions, Privacy: privacy, Files: []OrderFile{}, PIIRetention: PIIRetention{PolicyVersion: "intake-30d-v1", Disposition: "hard_delete", SourceDeleteAfter: deadline, ApplyAfter: deadline}, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}, nil
}

func RestoreOrder(order Order) (Order, error) {
	if order.ID == "" || order.SubjectID == "" || !allowedOrderState[order.Status] || order.Version == 0 || order.Version > MaxOrderVersion || order.CreatedAt.IsZero() || order.UpdatedAt.Before(order.CreatedAt) || !versionPattern.MatchString(order.PIIRetention.PolicyVersion) || (order.PIIRetention.Disposition != "hard_delete" && order.PIIRetention.Disposition != "irreversible_anonymize") || order.PIIRetention.SourceDeleteAfter.IsZero() || order.PIIRetention.ApplyAfter.IsZero() {
		return Order{}, errors.New("invalid persisted order")
	}
	if err := NewValidationError(validateOrder(order.PublicReference, order.Title, order.Purpose, order.TargetClaim)...); err != nil {
		return Order{}, err
	}
	for _, file := range order.Files {
		if _, err := RestoreOrderFile(file); err != nil {
			return Order{}, fmt.Errorf("restore order file: %w", err)
		}
	}
	for _, event := range order.Events {
		if event.ID == "" || event.ActorID == "" || event.Type == "" || event.CreatedAt.IsZero() || (event.ActorKind != "submitter" && event.ActorKind != "administrator" && event.ActorKind != "system") {
			return Order{}, errors.New("invalid persisted order event")
		}
	}
	return order, nil
}

func (o Order) ReserveFile(file OrderFile, expected uint64, now time.Time) (Order, error) {
	if o.Version != expected {
		return Order{}, NewVersionConflictError(expected, o.Version)
	}
	if o.Status != "draft" && o.Status != "uploading" {
		return Order{}, ErrStateConflict
	}
	if len(o.Files) >= MaxOrderFiles {
		return Order{}, NewValidationError(FieldIssue{Field: "files", Code: "too_many", Message: "at most 20 files are allowed"})
	}
	o.Files = append(append([]OrderFile(nil), o.Files...), file)
	o.Status = "uploading"
	return o.bump(now)
}

func (o Order) ReplaceFile(file OrderFile, expected uint64, now time.Time) (Order, error) {
	if o.Version != expected {
		return Order{}, NewVersionConflictError(expected, o.Version)
	}
	found := false
	for index := range o.Files {
		if o.Files[index].ID == file.ID {
			o.Files[index], found = file, true
		}
	}
	if !found {
		return Order{}, ErrFileNotFound
	}
	return o.bump(now)
}

func (o Order) RemoveFile(fileID Identifier, expected uint64, now time.Time) (Order, OrderFile, error) {
	if o.Version != expected {
		return Order{}, OrderFile{}, NewVersionConflictError(expected, o.Version)
	}
	if o.Status != "draft" && o.Status != "uploading" {
		return Order{}, OrderFile{}, ErrStateConflict
	}
	files := make([]OrderFile, 0, len(o.Files))
	var removed OrderFile
	for _, file := range o.Files {
		if file.ID == fileID {
			removed = file
			continue
		}
		files = append(files, file)
	}
	if removed.ID == "" {
		return Order{}, OrderFile{}, ErrFileNotFound
	}
	o.Files = files
	if len(files) == 0 {
		o.Status = "draft"
	}
	updated, err := o.bump(now)
	return updated, removed, err
}

func (o Order) Submit(expected uint64, termsVersion string, accepted bool, now time.Time) (Order, error) {
	return o.SubmitWithAuthorizations(expected, termsVersion, accepted, true, true, false, now)
}

func (o Order) SubmitWithAuthorizations(expected uint64, termsVersion string, termsAccepted, uploadsAuthorized, analysisUseAuthorized, externalRedistributionAuthorized bool, now time.Time) (Order, error) {
	return o.SubmitWithRetention(expected, termsVersion, termsAccepted, uploadsAuthorized, analysisUseAuthorized, externalRedistributionAuthorized, o.PIIRetention, now)
}

func (o Order) SubmitWithRetention(expected uint64, termsVersion string, termsAccepted, uploadsAuthorized, analysisUseAuthorized, externalRedistributionAuthorized bool, retention PIIRetention, now time.Time) (Order, error) {
	if o.Version != expected {
		return Order{}, NewVersionConflictError(expected, o.Version)
	}
	if o.Status != "uploading" && o.Status != "draft" {
		return Order{}, ErrStateConflict
	}
	if !termsAccepted || !uploadsAuthorized || !analysisUseAuthorized || externalRedistributionAuthorized || !versionPattern.MatchString(termsVersion) {
		return Order{}, NewValidationError(FieldIssue{Field: "termsAccepted", Code: "invalid", Message: "current terms must be accepted"})
	}
	primary, pending := 0, false
	for _, file := range o.Files {
		if file.Role == "primary_paper" {
			primary++
		}
		if file.Status == "upload_pending" {
			pending = true
		}
	}
	if primary != 1 || pending {
		return Order{}, NewValidationError(FieldIssue{Field: "files", Code: "invalid", Message: "exactly one completed primary paper is required"})
	}
	timestamp := now.UTC()
	if !versionPattern.MatchString(retention.PolicyVersion) || retention.Disposition != "hard_delete" || !retention.SourceDeleteAfter.After(timestamp) || !retention.ApplyAfter.After(timestamp) || retention.SourceDeleteAfter.After(retention.ApplyAfter) {
		return Order{}, NewValidationError(FieldIssue{Field: "piiRetention", Code: "invalid", Message: "server retention policy must freeze ordered future source and PII deletion deadlines"})
	}
	o.Status, o.TermsVersion, o.SubmittedAt = "submitted", termsVersion, &timestamp
	o.PIIRetention = retention
	o.Authorizations = CustomerAuthorizations{UploadsAuthorized: true, AnalysisUseAuthorized: true, ExternalRedistributionAuthorized: false, ConfirmedAt: &timestamp}
	return o.bump(timestamp)
}

func (o Order) ReconcileInspection(now time.Time) (Order, error) {
	if o.SubmittedAt == nil {
		return Order{}, ErrStateConflict
	}
	status := "needs_information"
	for _, file := range o.Files {
		switch file.Status {
		case "rejected":
			status = "rejected"
		case "uploaded", "scanning", "upload_pending":
			if status != "rejected" {
				status = "scanning"
			}
		}
	}
	if o.Status == status {
		return o, nil
	}
	o.Status = status
	return o.bump(now)
}

func (o Order) QueueExport(expected uint64, now time.Time) (Order, error) {
	if o.Version != expected {
		return Order{}, NewVersionConflictError(expected, o.Version)
	}
	if o.Status != "ready_for_export" || o.Intake == nil {
		return Order{}, ErrStateConflict
	}
	for _, file := range o.Files {
		if file.Status != "clean" || file.DetectedMediaType == "" {
			return Order{}, ErrFileNotClean
		}
	}
	return o.bump(now)
}

func (o Order) ExportReady(now time.Time) (Order, error) {
	if o.Status != "ready_for_export" {
		return Order{}, ErrStateConflict
	}
	o.Status = "exported"
	return o.bump(now)
}

func (o Order) bump(now time.Time) (Order, error) {
	if o.Version >= MaxOrderVersion {
		return Order{}, ErrStateConflict
	}
	o.Version++
	o.UpdatedAt = now.UTC()
	return o, nil
}

type Export struct {
	ID                 Identifier
	OrderID            Identifier
	Status             string
	RoutineID          string
	RoutineRevision    string
	RoutineValidatedAt time.Time
	RoutineEvidenceSHA string
	RetentionPolicy    string
	PreserveRunOutputs bool
	SHA256             string
	SizeBytes          int64
	StorageKey         string
	ObjectGeneration   string
	FailureCode        string
	CreatedAt          time.Time
	CompletedAt        *time.Time
}

type AdminIntake struct {
	AuditRequest       []byte
	ScientificPolicy   []byte
	ExecutionPolicy    []byte
	RoutineRevision    string
	RoutineValidatedAt time.Time
	RoutineEvidenceSHA string
	FrozenBy           Identifier
	FrozenAt           time.Time
}

func (o Order) FreezeIntake(expected uint64, intake AdminIntake, now time.Time) (Order, error) {
	if o.Version != expected {
		return Order{}, NewVersionConflictError(expected, o.Version)
	}
	if o.Status != "scanning" && o.Status != "needs_information" && o.Status != "submitted" {
		return Order{}, ErrStateConflict
	}
	if !versionPattern.MatchString(o.PIIRetention.PolicyVersion) || o.PIIRetention.Disposition != "hard_delete" || !o.PIIRetention.SourceDeleteAfter.After(now) || !o.PIIRetention.ApplyAfter.After(now) || o.PIIRetention.SourceDeleteAfter.After(o.PIIRetention.ApplyAfter) {
		return Order{}, NewValidationError(FieldIssue{Field: "auditRequest.retention", Code: "invalid", Message: "must select a future PII erasure deadline"})
	}
	if !regexp.MustCompile(`^sha256:[a-f0-9]{64}$`).MatchString(intake.RoutineRevision) || !sha256Pattern.MatchString(intake.RoutineEvidenceSHA) || intake.RoutineValidatedAt.IsZero() || len(intake.AuditRequest) == 0 || len(intake.ScientificPolicy) == 0 || len(intake.ExecutionPolicy) == 0 || intake.FrozenBy == "" {
		return Order{}, NewValidationError(FieldIssue{Field: "intake", Code: "invalid", Message: "complete validated intake and pinned routine revision are required"})
	}
	for _, file := range o.Files {
		if file.Status != "clean" {
			return Order{}, ErrFileNotClean
		}
	}
	intake.FrozenAt = now.UTC()
	intake.AuditRequest = append([]byte(nil), intake.AuditRequest...)
	intake.ScientificPolicy = append([]byte(nil), intake.ScientificPolicy...)
	intake.ExecutionPolicy = append([]byte(nil), intake.ExecutionPolicy...)
	o.Intake, o.Status = &intake, "ready_for_export"
	return o.bump(now)
}
