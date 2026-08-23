package game

import (
	"context"
	"testing"
	"time"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/judge"
	"github.com/timfewi/grepquest/internal/sandbox"
	"github.com/timfewi/grepquest/internal/sandbox/mock"
)

func TestSpacedReviewProgression(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	c := challenge.Core()[10]
	backend := &mock.Backend{Results: []sandbox.ExecutionResult{
		{Stdout: c.WantOutput, ExitCode: c.WantExit, Args: []string{"grep"}},
		{Stdout: c.WantOutput, ExitCode: c.WantExit, Args: []string{"grep"}},
		{Stdout: "wrong\n", ExitCode: 0, Args: []string{"grep"}},
	}}
	engine := Engine{Judge: judge.Judge{Sandbox: backend}, Now: func() time.Time { return now }}
	profile := NewProfile()

	first, err := engine.Submit(context.Background(), &profile, c, "grep", 0)
	if err != nil || !first.Correct {
		t.Fatalf("initial: %#v %v", first, err)
	}
	if profile.Reviews[c.ID].DueAt != now.Add(24*time.Hour).Format(time.RFC3339) {
		t.Fatalf("wrong first interval: %#v", profile.Reviews[c.ID])
	}

	now = now.Add(24 * time.Hour)
	review, err := engine.Submit(context.Background(), &profile, c, "grep", 0)
	if err != nil || !review.Correct || !review.Review || review.XPGained != 30 {
		t.Fatalf("review: %#v %v", review, err)
	}
	state := profile.Reviews[c.ID]
	if state.Stage != 1 || state.DueAt != now.Add(3*24*time.Hour).Format(time.RFC3339) {
		t.Fatalf("wrong review state: %#v", state)
	}

	now = now.Add(3 * 24 * time.Hour)
	failed, err := engine.Submit(context.Background(), &profile, c, "grep", 0)
	if err != nil || failed.Correct {
		t.Fatalf("failed review: %#v %v", failed, err)
	}
	state = profile.Reviews[c.ID]
	if state.Stage != 0 || state.Attempts != 2 {
		t.Fatalf("failure did not reschedule: %#v", state)
	}
}
