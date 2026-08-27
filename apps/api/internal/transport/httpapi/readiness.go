package httpapi

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
)

var ErrNotAcceptingTraffic = errors.New("application is starting or draining")

type ReadinessCheck func(context.Context) error

// ReadinessRegistry holds bounded, technical dependency checks. Registration is
// a startup operation; checks may run concurrently while shutdown changes state.
type ReadinessRegistry struct {
	mu        sync.RWMutex
	checks    map[string]ReadinessCheck
	accepting atomic.Bool
}

func NewReadinessRegistry() *ReadinessRegistry {
	return &ReadinessRegistry{checks: make(map[string]ReadinessCheck)}
}

func (registry *ReadinessRegistry) Register(name string, check ReadinessCheck) error {
	if registry == nil || name == "" || check == nil {
		return errors.New("httpapi: readiness check requires a registry, name, and function")
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.checks[name]; exists {
		return fmt.Errorf("httpapi: duplicate readiness check %q", name)
	}
	registry.checks[name] = check
	return nil
}

func (registry *ReadinessRegistry) SetAccepting(accepting bool) {
	if registry != nil {
		registry.accepting.Store(accepting)
	}
}

func (registry *ReadinessRegistry) Check(ctx context.Context) error {
	if registry == nil || !registry.accepting.Load() {
		return ErrNotAcceptingTraffic
	}
	registry.mu.RLock()
	names := make([]string, 0, len(registry.checks))
	checks := make(map[string]ReadinessCheck, len(registry.checks))
	for name, check := range registry.checks {
		names = append(names, name)
		checks[name] = check
	}
	registry.mu.RUnlock()
	sort.Strings(names)

	results := make(chan error, len(names))
	for _, name := range names {
		name, check := name, checks[name]
		go func() {
			defer func() {
				if recover() != nil {
					results <- fmt.Errorf("%s: readiness check panicked", name)
				}
			}()
			if err := check(ctx); err != nil {
				results <- fmt.Errorf("%s: %w", name, err)
				return
			}
			results <- nil
		}()
	}
	for range names {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-results:
			if err != nil {
				return err
			}
		}
	}
	return nil
}
