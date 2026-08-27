package ports

import (
	"context"
	"time"

	"github.com/mattwebhub/micro1-go-template/internal/domain"
)

// ProjectIDGenerator keeps randomness outside the domain and makes project
// creation deterministic in tests.
type ProjectIDGenerator interface {
	NewProjectID(ctx context.Context) (domain.ProjectID, error)
}

// Clock keeps current-time acquisition outside domain entities.
type Clock interface {
	Now() time.Time
}
