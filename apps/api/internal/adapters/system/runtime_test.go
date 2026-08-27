package system

import (
	"context"
	"testing"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

func TestProjectIDGeneratorProducesConstructedUUID(t *testing.T) {
	t.Parallel()
	id, err := (ProjectIDGenerator{}).NewProjectID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NewProjectID(id.String()); err != nil {
		t.Fatalf("generated ID %q is invalid: %v", id.String(), err)
	}
}
