package httpapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

type orderResponse struct {
	ID              string  `json:"id"`
	PublicReference string  `json:"publicReference"`
	Status          string  `json:"status"`
	Version         uint64  `json:"version"`
	Title           string  `json:"title"`
	Purpose         string  `json:"purpose"`
	TargetClaim     any     `json:"targetClaim"`
	Files           []any   `json:"files"`
	PIIRetention    any     `json:"piiRetention"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
	SubmittedAt     *string `json:"submittedAt"`
}

func orderDTO(order domain.Order) orderResponse {
	files := make([]any, 0, len(order.Files))
	for _, file := range order.Files {
		files = append(files, fileDTO(file))
	}
	var submitted *string
	if order.SubmittedAt != nil {
		value := formatHTTPTime(*order.SubmittedAt)
		submitted = &value
	}
	var location any = nil
	if order.TargetClaim.SourceLocation != "" {
		location = order.TargetClaim.SourceLocation
	}
	return orderResponse{ID: order.ID.String(), PublicReference: order.PublicReference, Status: order.Status, Version: order.Version, Title: order.Title, Purpose: order.Purpose, TargetClaim: struct {
		Text           string `json:"text"`
		SourceLocation any    `json:"sourceLocation"`
	}{order.TargetClaim.Text, location}, Files: files, PIIRetention: struct {
		PolicyVersion     string `json:"policyVersion"`
		Disposition       string `json:"disposition"`
		SourceDeleteAfter string `json:"sourceDeleteAfter"`
		PIIDeleteAfter    string `json:"piiDeleteAfter"`
	}{order.PIIRetention.PolicyVersion, order.PIIRetention.Disposition, formatHTTPTime(order.PIIRetention.SourceDeleteAfter), formatHTTPTime(order.PIIRetention.ApplyAfter)}, CreatedAt: formatHTTPTime(order.CreatedAt), UpdatedAt: formatHTTPTime(order.UpdatedAt), SubmittedAt: submitted}
}

func fileDTO(file domain.OrderFile) any {
	var detected, rejection any = nil, nil
	if file.DetectedMediaType != "" {
		detected = file.DetectedMediaType
	}
	if file.RejectionCode != "" {
		rejection = file.RejectionCode
	}
	return struct {
		ID                  string `json:"id"`
		Role                string `json:"role"`
		OriginalDisplayName string `json:"originalDisplayName"`
		SizeBytes           int64  `json:"sizeBytes"`
		SHA256              string `json:"sha256"`
		Storage             any    `json:"storage"`
		DeclaredMediaType   string `json:"declaredMediaType"`
		DetectedMediaType   any    `json:"detectedMediaType"`
		Status              string `json:"status"`
		RejectionCode       any    `json:"rejectionCode"`
		CreatedAt           string `json:"createdAt"`
		UpdatedAt           string `json:"updatedAt"`
	}{file.ID.String(), file.Role, file.OriginalDisplayName, file.SizeBytes, file.SHA256, struct {
		ObjectVersion string `json:"objectVersion"`
		SHA256        string `json:"sha256"`
		Immutability  string `json:"immutability"`
	}{file.ObjectGeneration, file.SHA256, "write_once"}, file.DeclaredMediaType, detected, file.Status, rejection, formatHTTPTime(file.CreatedAt), formatHTTPTime(file.UpdatedAt)}
}

func adminOrderDTO(order domain.Order, exports []domain.Export) any {
	base := orderDTO(order)
	issues := readinessIssues(order)
	exportValues := make([]any, 0, len(exports))
	for _, item := range exports {
		exportValues = append(exportValues, exportDTO(item, order.Files))
	}
	var intake any = nil
	if order.Intake != nil {
		intake = struct {
			AuditRequest     json.RawMessage `json:"auditRequest"`
			ScientificPolicy json.RawMessage `json:"scientificPolicy"`
			ExecutionPolicy  json.RawMessage `json:"executionPolicy"`
			RoutineContract  any             `json:"routineContract"`
		}{order.Intake.AuditRequest, order.Intake.ScientificPolicy, order.Intake.ExecutionPolicy, routineDTO(order.Intake.RoutineRevision, order.Intake.RoutineValidatedAt, order.Intake.RoutineEvidenceSHA)}
	}
	var email any = nil
	if order.SubmitterEmail != "" {
		email = order.SubmitterEmail
	}
	permissions := struct {
		ExecuteSuppliedCode bool `json:"executeSuppliedCode"`
		ExternalSearch      bool `json:"externalSearch"`
	}{order.Permissions.ExecuteSuppliedCode, order.Permissions.ExternalSearch}
	privacy := struct {
		ContainsParticipantLevelData bool `json:"containsParticipantLevelData"`
		ContainsDirectIdentifiers    bool `json:"containsDirectIdentifiers"`
	}{order.Privacy.ContainsParticipantLevelData, order.Privacy.ContainsDirectIdentifiers}
	events := make([]any, 0, len(order.Events))
	for _, event := range order.Events {
		var metadata map[string]any
		if len(event.Metadata) > 0 {
			_ = json.Unmarshal(event.Metadata, &metadata)
		}
		if metadata == nil {
			metadata = map[string]any{}
		}
		events = append(events, struct {
			ID        string         `json:"id"`
			ActorKind string         `json:"actorKind"`
			ActorID   string         `json:"actorId"`
			Type      string         `json:"type"`
			Metadata  map[string]any `json:"metadata"`
			CreatedAt string         `json:"createdAt"`
		}{event.ID.String(), event.ActorKind, event.ActorID, event.Type, metadata, formatHTTPTime(event.CreatedAt)})
	}
	return struct {
		orderResponse
		SubmitterEmail  any   `json:"submitterEmail"`
		Permissions     any   `json:"permissions"`
		Privacy         any   `json:"privacy"`
		FrozenIntake    any   `json:"frozenIntake"`
		ReadinessIssues []any `json:"readinessIssues"`
		Events          []any `json:"events"`
		Exports         []any `json:"exports"`
	}{base, email, permissions, privacy, intake, issues, events, exportValues}
}

func readinessIssues(order domain.Order) []any {
	var result []any
	primary := 0
	for index, file := range order.Files {
		if file.Role == "primary_paper" {
			primary++
		}
		if file.Status != "clean" {
			result = append(result, readinessIssue("file_not_clean", fmt.Sprintf("files[%d].status", index), "Every export input must pass inspection."))
		}
	}
	if primary != 1 {
		result = append(result, readinessIssue("primary_paper_required", "files", "Exactly one primary paper is required."))
	}
	if order.Intake == nil {
		result = append(result, readinessIssue("intake_required", "frozenIntake", "A validated frozen intake is required."))
	}
	if result == nil {
		return []any{}
	}
	return result
}
func readinessIssue(code, path, message string) any {
	return struct {
		Code    string `json:"code"`
		Path    string `json:"path"`
		Message string `json:"message"`
	}{code, path, message}
}

func exportDTO(export domain.Export, files []domain.OrderFile) any {
	inputs := make([]any, 0, len(files))
	for _, file := range files {
		inputs = append(inputs, struct {
			FileID        string `json:"fileId"`
			ObjectVersion string `json:"objectVersion"`
			SHA256        string `json:"sha256"`
		}{file.ID.String(), file.ObjectGeneration, file.SHA256})
	}
	var sha, size, content, completed, failure any = nil, nil, nil, nil, nil
	if export.SHA256 != "" {
		sha = export.SHA256
	}
	if export.SizeBytes > 0 {
		size = export.SizeBytes
	}
	if export.Status == "ready" {
		content = "/api/v1/admin/exports/" + export.ID.String() + "/download"
	}
	if export.CompletedAt != nil {
		completed = formatHTTPTime(*export.CompletedAt)
	}
	if export.FailureCode != "" {
		failure = export.FailureCode
	}
	return struct {
		ID              string `json:"id"`
		OrderID         string `json:"orderId"`
		Status          string `json:"status"`
		RoutineContract any    `json:"routineContract"`
		Inputs          []any  `json:"inputs"`
		SHA256          any    `json:"sha256"`
		SizeBytes       any    `json:"sizeBytes"`
		ContentPath     any    `json:"contentPath"`
		CreatedAt       string `json:"createdAt"`
		CompletedAt     any    `json:"completedAt"`
		FailureCode     any    `json:"failureCode"`
	}{export.ID.String(), export.OrderID.String(), export.Status, routineDTO(export.RoutineRevision, export.RoutineValidatedAt, export.RoutineEvidenceSHA), inputs, sha, size, content, formatHTTPTime(export.CreatedAt), completed, failure}
}
func routineDTO(revision string, validatedAt time.Time, evidence string) any {
	return struct {
		RoutineID  string `json:"routineId"`
		Revision   string `json:"revision"`
		Validation any    `json:"validation"`
	}{"claim-bounty-operations/run-claimbounty-scientific-audit", revision, struct {
		Status         string `json:"status"`
		ValidatedAt    string `json:"validatedAt"`
		EvidenceSHA256 string `json:"evidenceSha256"`
	}{"validated", formatHTTPTime(validatedAt), evidence}}
}
func formatHTTPTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
