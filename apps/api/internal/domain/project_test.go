package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

const validProjectID = "123e4567-e89b-12d3-a456-426614174000"

func TestNewProjectNormalizesNameAndTime(t *testing.T) {
	t.Parallel()

	id, err := domain.NewProjectID("123E4567-E89B-12D3-A456-426614174000")
	if err != nil {
		t.Fatalf("NewProjectID() error = %v", err)
	}
	localTime := time.Date(2026, time.August, 27, 10, 0, 0, 0, time.FixedZone("test", 3600))
	project, err := domain.NewProject(id, "  Example project  ", localTime)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}

	if got, want := project.ID().String(), validProjectID; got != want {
		t.Fatalf("ID() = %q, want %q", got, want)
	}
	if got, want := project.Name(), "Example project"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	if got, want := project.CreatedAt().Location(), time.UTC; got != want {
		t.Fatalf("CreatedAt().Location() = %v, want UTC", got)
	}
	if !project.CreatedAt().Equal(localTime) || !project.UpdatedAt().Equal(localTime) {
		t.Fatal("project timestamps do not preserve the supplied instant")
	}
}

func TestProjectRejectsInvalidInputWithFieldIssues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		id        domain.ProjectID
		project   string
		createdAt time.Time
		updatedAt time.Time
		field     string
	}{
		{"invalid ID", domain.ProjectID{}, "Valid", time.Now(), time.Now(), "projectId"},
		{"empty name", mustProjectID(t), "   ", time.Now(), time.Now(), "name"},
		{"control character", mustProjectID(t), "bad\nname", time.Now(), time.Now(), "name"},
		{"zero created time", mustProjectID(t), "Valid", time.Time{}, time.Now(), "createdAt"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.RestoreProject(test.id, test.project, test.createdAt, test.updatedAt)
			if !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("RestoreProject() error = %v, want validation error", err)
			}
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("RestoreProject() error type = %T, want *ValidationError", err)
			}
			if !hasIssue(validation.Issues(), test.field) {
				t.Fatalf("issues = %#v, want field %q", validation.Issues(), test.field)
			}
		})
	}
}

func mustProjectID(t *testing.T) domain.ProjectID {
	t.Helper()
	id, err := domain.NewProjectID(validProjectID)
	if err != nil {
		t.Fatalf("NewProjectID() error = %v", err)
	}
	return id
}

func TestNewProjectIDRejectsNonUUID(t *testing.T) {
	t.Parallel()

	if _, err := domain.NewProjectID("project-1"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("NewProjectID() error = %v, want validation error", err)
	}
}

func hasIssue(issues []domain.FieldIssue, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
