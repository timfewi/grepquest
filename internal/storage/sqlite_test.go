package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/timfewi/grepquest/internal/game"
)

func TestProfileRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "state", "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	p := game.NewProfile()
	p.XP = 420
	p.Level = 2
	p.Solved["literal-001"] = true
	p.Mastery["literal"] = 88
	p.Reviews["literal-001"] = game.ReviewState{Stage: 1, DueAt: "2026-08-23T00:00:00Z"}
	if err = s.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.XP != p.XP || got.Mastery["literal"] != 88 || !got.Solved["literal-001"] || got.Reviews["literal-001"].Stage != 1 {
		t.Fatalf("roundtrip mismatch: %#v", got)
	}
}

func TestAICacheAndUsage(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	if err = s.PutAI(ctx, "key", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	raw, ok, err := s.GetAI(ctx, "key")
	if err != nil || !ok || string(raw) != `{"ok":true}` {
		t.Fatalf("cache: %s %v %v", raw, ok, err)
	}
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	if err = s.RecordAIRequest(ctx, now, 0.01); err != nil {
		t.Fatal(err)
	}
	count, err := s.AIRequestsToday(ctx, now)
	if err != nil || count != 1 {
		t.Fatalf("usage: %d %v", count, err)
	}
}
