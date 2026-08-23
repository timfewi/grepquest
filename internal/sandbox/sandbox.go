package sandbox

import (
	"context"
	"time"

	"github.com/timfewi/grepquest/internal/challenge"
)

type ExecutionResult struct {
	Stdout, Stderr string
	Args           []string
	ExitCode       int
	Duration       time.Duration
	Truncated      bool
}

type Sandbox interface {
	Boot(context.Context, challenge.Challenge) error
	Run(context.Context, string) (ExecutionResult, error)
	Reset(context.Context) error
	Close(context.Context) error
}
