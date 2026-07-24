package rollback

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const defaultTimeout = 30 * time.Second

type step struct {
	description string
	run         func(context.Context) error
}

// Manager coordinates compensating actions for operations that span multiple
// external systems. Cleanup is detached from request cancellation but remains
// bounded by a timeout so a disconnected client cannot prevent compensation.
type Manager struct {
	timeout time.Duration
	steps   []step
}

func New() *Manager {
	return &Manager{timeout: defaultTimeout}
}

func NewWithTimeout(timeout time.Duration) *Manager {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Manager{timeout: timeout}
}

func (m *Manager) Add(description string, run func(context.Context) error) {
	m.steps = append(m.steps, step{description: description, run: run})
}

func (m *Manager) Execute(parent context.Context) error {
	if parent == nil {
		parent = context.Background()
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), m.timeout)
	defer cancel()

	var cleanupErrors []error
	for index := len(m.steps) - 1; index >= 0; index-- {
		current := m.steps[index]
		if err := current.run(cleanupCtx); err != nil {
			cleanupErrors = append(
				cleanupErrors,
				fmt.Errorf("%s: %w", current.description, err),
			)
		}
	}
	return errors.Join(cleanupErrors...)
}
