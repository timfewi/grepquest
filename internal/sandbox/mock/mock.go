package mock

import (
	"context"
	"errors"
	"sync"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/sandbox"
)

// Backend is a deterministic test double for UI and engine tests.
type Backend struct {
	mu      sync.Mutex
	Results []sandbox.ExecutionResult
	Errors  []error
	Runs    []string
	Booted  []challenge.Challenge
	closed  bool
}

func (b *Backend) Boot(_ context.Context, c challenge.Challenge) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		b.closed = false
	}
	b.Booted = append(b.Booted, c)
	return nil
}
func (b *Backend) Run(_ context.Context, command string) (sandbox.ExecutionResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Runs = append(b.Runs, command)
	if len(b.Results) == 0 {
		return sandbox.ExecutionResult{}, errors.New("mock result queue is empty")
	}
	result := b.Results[0]
	b.Results = b.Results[1:]
	var err error
	if len(b.Errors) > 0 {
		err = b.Errors[0]
		b.Errors = b.Errors[1:]
	}
	return result, err
}
func (b *Backend) Reset(context.Context) error { return nil }
func (b *Backend) Close(context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}
