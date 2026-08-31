package services

import (
	"context"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/ports"
)

type RetentionCleanup struct {
	repository     ports.IntakeRepository
	clock          ports.Clock
	batchSize      int
	abandonedAfter time.Duration
}

func NewRetentionCleanup(repository ports.IntakeRepository, clock ports.Clock, batchSize int, abandonedAfter ...time.Duration) (*RetentionCleanup, error) {
	if repository == nil || clock == nil || batchSize < 1 || batchSize > 1000 {
		return nil, ErrInvalidDependencies
	}
	age := 7 * 24 * time.Hour
	if len(abandonedAfter) > 0 {
		age = abandonedAfter[0]
	}
	if age < 24*time.Hour {
		return nil, ErrInvalidDependencies
	}
	return &RetentionCleanup{repository: repository, clock: clock, batchSize: batchSize, abandonedAfter: age}, nil
}

func (cleanup *RetentionCleanup) Run(ctx context.Context) error {
	now := cleanup.clock.Now()
	if err := cleanup.repository.CleanupExpiredIdentityAndAbandoned(ctx, now, now.Add(-cleanup.abandonedAfter)); err != nil {
		return err
	}
	_, err := cleanup.repository.AnonymizeExpired(ctx, now, cleanup.batchSize)
	return err
}
