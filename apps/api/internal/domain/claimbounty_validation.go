package domain

import (
	"fmt"
	"strings"
)

func validateOrder(reference, title, purpose string, claim TargetClaim) []FieldIssue {
	var issues []FieldIssue
	issues = boundedText(issues, "title", title, 300)
	issues = boundedText(issues, "purpose", purpose, 1000)
	issues = boundedText(issues, "targetClaim.text", claim.Text, 5000)
	if claim.SourceLocation != "" {
		issues = boundedText(issues, "targetClaim.sourceLocation", claim.SourceLocation, 1000)
	}
	if !referencePattern.MatchString(reference) {
		issues = append(issues, FieldIssue{Field: "publicReference", Code: "invalid_format", Message: "must use the ClaimBounty reference format"})
	}
	return issues
}

func validateFile(role, displayName string, size int64, sha256, mediaType string) []FieldIssue {
	var issues []FieldIssue
	if !allowedFileRoles[role] {
		issues = append(issues, FieldIssue{Field: "role", Code: "unsupported", Message: "unsupported file role"})
	}
	if strings.TrimSpace(displayName) == "" || len(displayName) > 255 || strings.ContainsAny(displayName, "/\\\x00") {
		issues = append(issues, FieldIssue{Field: "originalDisplayName", Code: "invalid", Message: "must be a safe display filename"})
	}
	if _, knownRole := allowedRoleExtensions[role]; knownRole && !allowedFileName(role, displayName) {
		issues = append(issues, FieldIssue{Field: "originalDisplayName", Code: "unsupported", Message: "file extension is not accepted for scientific intake"})
	}
	if size < 1 || size > MaxFileBytes {
		issues = append(issues, FieldIssue{Field: "sizeBytes", Code: "out_of_range", Message: "must be between 1 byte and 250 MiB"})
	}
	if role == "primary_paper" && size > 50*1024*1024 {
		issues = append(issues, FieldIssue{Field: "sizeBytes", Code: "out_of_range", Message: "primary paper must be at most 50 MiB"})
	}
	if role == "primary_paper" && mediaType != "application/pdf" {
		issues = append(issues, FieldIssue{Field: "declaredMediaType", Code: "unsupported", Message: "primary paper must be application/pdf"})
	}
	if !sha256Pattern.MatchString(sha256) {
		issues = append(issues, FieldIssue{Field: "sha256", Code: "invalid_format", Message: "must be a lowercase SHA-256 digest"})
	}
	if len(mediaType) > 255 || !mediaTypePattern.MatchString(mediaType) {
		issues = append(issues, FieldIssue{Field: "declaredMediaType", Code: "invalid_format", Message: "must be a media type"})
	} else if allowedFileRoles[role] && allowedFileName(role, displayName) && !(OrderFile{Role: role, OriginalDisplayName: displayName}.AcceptsDeclaredMediaType(mediaType)) {
		issues = append(issues, FieldIssue{Field: "declaredMediaType", Code: "unsupported", Message: "media type does not match the allowed role and file form"})
	}
	return issues
}

func boundedText(issues []FieldIssue, field, value string, maximum int) []FieldIssue {
	if strings.TrimSpace(value) == "" {
		return append(issues, FieldIssue{Field: field, Code: "required", Message: "must be provided"})
	}
	if len(value) > maximum {
		return append(issues, FieldIssue{Field: field, Code: "too_long", Message: fmt.Sprintf("must be at most %d bytes", maximum)})
	}
	return issues
}

func set(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func ValidPolicyVersion(value string) bool   { return versionPattern.MatchString(value) }
func ValidOrderStatus(value string) bool     { return allowedOrderState[value] }
func ValidPublicReference(value string) bool { return referencePattern.MatchString(value) }
