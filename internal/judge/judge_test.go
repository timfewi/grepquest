package judge

import (
	"context"
	"testing"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/sandbox/native"
)

func TestEveryCoreChallengeHasWorkingReference(t *testing.T) {
	references := map[string]string{
		"literal-001": "grep ERROR vault.log", "case-001": "grep -i warning alarms.log", "fixed-001": "grep -F a.b runes.txt", "word-001": "grep -w cat beasts.txt", "line-001": "grep -x READY state.txt", "exit-001": "grep dragon empty.txt", "anchor-start-001": "grep '^TODO' tasks.txt", "anchor-end-001": `grep '\.conf$' files.txt`, "class-001": `grep 'seal-[[:digit:]]$' seals.txt`, "ere-001": `grep -E 'sword|shield' loot.txt`,
		"literal-002": "grep key east.txt west.txt", "case-word-001": "grep -iw ORC scouts.txt", "fixed-002": "grep -F '[vault]' marks.txt", "line-case-001": "grep -ix sealed doors.txt", "exit-error-001": "grep rune missing.txt", "bre-repeat-001": `grep '^[[:digit:]]\{2\}$' codes.txt`, "ere-plus-001": `grep -E '^ha!+$' horn.txt`, "ere-group-001": `grep -E '^(red|blue)-key$' keys.txt`, "ere-optional-001": `grep -E '^armou?r$' scroll.txt`, "quote-space-001": `grep 'silver key' notes.txt`, "ere-class-001": `grep -E '^[A-F0-9]+$' hex.txt`, "multiple-pattern-001": "grep -e goblin -e troll bounties.txt", "bre-group-001": `grep '^\(ab\)\{2,3\}$' chants.txt`, "ere-bounded-001": `grep -E '^knocko{2,4}$' knocks.txt`, "escape-plus-001": `grep '+1' charges.txt`, "ere-combined-001": `grep -E '^Q-[[:digit:]]{3}$' ids.txt`,
	}
	for _, c := range challenge.Core() {
		t.Run(c.ID, func(t *testing.T) {
			v, err := (Judge{Sandbox: native.New()}).Evaluate(context.Background(), c, references[c.ID])
			if err != nil {
				t.Fatal(err)
			}
			if !v.Correct {
				t.Fatalf("reference rejected: %#v", v)
			}
		})
	}
}

func TestWrongOutputFails(t *testing.T) {
	c := challenge.Core()[0]
	v, err := (Judge{Sandbox: native.New()}).Evaluate(context.Background(), c, "grep WARN vault.log")
	if err != nil {
		t.Fatal(err)
	}
	if v.Correct {
		t.Fatal("wrong output passed")
	}
}
