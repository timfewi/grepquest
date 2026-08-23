package learning

import (
	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/game"
	"testing"
	"time"
)

func TestDueReviewsComeBeforeUnseen(t *testing.T) {
	catalog := challenge.Core()[:3]
	profile := game.NewProfile()
	profile.Solved[catalog[1].ID] = true
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	profile.Reviews[catalog[1].ID] = game.ReviewState{DueAt: now.Add(-time.Hour).Format(time.RFC3339)}
	got := Queue(catalog, profile, now)
	if got[0].ID != catalog[1].ID || got[1].ID != catalog[0].ID {
		t.Fatalf("unexpected queue: %s, %s", got[0].ID, got[1].ID)
	}
}
