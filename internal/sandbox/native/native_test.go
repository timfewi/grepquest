package native

import (
	"context"
	"strings"
	"testing"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/commandparse"
)

func TestRunSingleGrep(t *testing.T) {
	b := New()
	c := challenge.Challenge{ID: "x", Act: 1, Title: "x", Prompt: "x", Difficulty: 1, Fixtures: map[string]string{"input.txt": "alpha\nbeta\n"}, WantExit: 0}
	if err := b.Boot(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close(context.Background()) }()
	got, err := b.Run(context.Background(), "grep '^a' input.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got.Stdout != "alpha\n" || got.ExitCode != 0 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestRejectsShellAndPathEscape(t *testing.T) {
	b := New()
	c := challenge.Challenge{ID: "x", Act: 1, Title: "x", Prompt: "x", Difficulty: 1, Fixtures: map[string]string{"input.txt": "x\n"}, WantExit: 0}
	if err := b.Boot(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close(context.Background()) }()
	for _, command := range []string{"grep x input.txt; id", "grep x /etc/passwd", "grep x ../secret", "sh -c grep"} {
		if _, err := b.Run(context.Background(), command); err == nil {
			t.Errorf("expected %q to be rejected", command)
		}
	}
}

func TestSplitCommandQuotes(t *testing.T) {
	got, err := commandparse.Split(`grep -E 'sword|shield' loot.txt`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"grep", "-E", "sword|shield", "loot.txt"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %#v", got)
	}
}
