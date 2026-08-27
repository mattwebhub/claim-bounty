// Package system implements process-local outbound capabilities.
package system

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

type Clock struct{}

func (Clock) Now() time.Time { return time.Now().UTC() }

type ProjectIDGenerator struct{}

func (ProjectIDGenerator) NewProjectID(ctx context.Context) (domain.ProjectID, error) {
	if err := ctx.Err(); err != nil {
		return domain.ProjectID{}, err
	}
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return domain.ProjectID{}, fmt.Errorf("system: read UUID randomness: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return domain.NewProjectID(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]))
}

var _ ports.Clock = Clock{}
var _ ports.ProjectIDGenerator = ProjectIDGenerator{}
