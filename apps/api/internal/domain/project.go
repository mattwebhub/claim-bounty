package domain

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxProjectNameRunes = 120

// ProjectID is an opaque UUID-shaped project identifier. Its representation is
// private so every non-zero ID has passed the constructor.
type ProjectID struct {
	value string
}

func NewProjectID(raw string) (ProjectID, error) {
	if !isUUID(raw) {
		return ProjectID{}, NewValidationError(FieldIssue{
			Field:   "projectId",
			Code:    "invalid_format",
			Message: "must be a UUID",
		})
	}

	return ProjectID{value: strings.ToLower(raw)}, nil
}

func (id ProjectID) String() string { return id.value }

// Project is immutable outside this package. Construct or restore it through
// the functions below so adapters cannot bypass its invariants.
type Project struct {
	id        ProjectID
	name      string
	createdAt time.Time
	updatedAt time.Time
}

func NewProject(id ProjectID, name string, now time.Time) (Project, error) {
	return RestoreProject(id, name, now, now)
}

func RestoreProject(id ProjectID, name string, createdAt, updatedAt time.Time) (Project, error) {
	var issues []FieldIssue
	if !isUUID(id.String()) {
		issues = append(issues, FieldIssue{Field: "projectId", Code: "invalid_format", Message: "must be a UUID"})
	}

	name = strings.TrimSpace(name)
	if name == "" {
		issues = append(issues, FieldIssue{Field: "name", Code: "required", Message: "must not be empty"})
	} else {
		if !utf8.ValidString(name) {
			issues = append(issues, FieldIssue{Field: "name", Code: "invalid_encoding", Message: "must be valid UTF-8"})
		}
		if utf8.RuneCountInString(name) > maxProjectNameRunes {
			issues = append(issues, FieldIssue{Field: "name", Code: "too_long", Message: "must be at most 120 characters"})
		}
		if containsControl(name) {
			issues = append(issues, FieldIssue{Field: "name", Code: "invalid_characters", Message: "must not contain control characters"})
		}
	}

	createdAt = createdAt.UTC()
	updatedAt = updatedAt.UTC()
	if createdAt.IsZero() {
		issues = append(issues, FieldIssue{Field: "createdAt", Code: "required", Message: "must not be zero"})
	}
	if updatedAt.IsZero() {
		issues = append(issues, FieldIssue{Field: "updatedAt", Code: "required", Message: "must not be zero"})
	}
	if !createdAt.IsZero() && !updatedAt.IsZero() && updatedAt.Before(createdAt) {
		issues = append(issues, FieldIssue{Field: "updatedAt", Code: "before_created_at", Message: "must not be before createdAt"})
	}
	if err := NewValidationError(issues...); err != nil {
		return Project{}, err
	}

	return Project{id: id, name: name, createdAt: createdAt, updatedAt: updatedAt}, nil
}

func (p Project) ID() ProjectID        { return p.id }
func (p Project) Name() string         { return p.name }
func (p Project) CreatedAt() time.Time { return p.createdAt }
func (p Project) UpdatedAt() time.Time { return p.updatedAt }

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func isUUID(value string) bool {
	if len(value) != 36 || value == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	for index := range value {
		switch index {
		case 8, 13, 18, 23:
			if value[index] != '-' {
				return false
			}
		default:
			character := value[index]
			if !((character >= '0' && character <= '9') ||
				(character >= 'a' && character <= 'f') ||
				(character >= 'A' && character <= 'F')) {
				return false
			}
		}
	}
	return true
}
