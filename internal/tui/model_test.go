package tui

import (
	"strings"
	"testing"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/game"
)

func TestThreePanelViewSnapshot(t *testing.T) {
	t.Parallel()
	profile := game.NewProfile()
	profile.XP = 320
	profile.Level = 2
	profile.Mastery["literal"] = 80
	profile.Loadout[game.Head] = game.BuiltinItems[0].ID
	model := New(game.Engine{}, challenge.Core()[:1], profile, nil)
	content := model.View().Content

	for _, expected := range []string{
		"ROOM 1/1 · First Signal",
		"Print lines containing ERROR from vault.log.",
		"TERMINAL",
		"RANGER · LV 2",
		"XP 320",
		"literal       80",
		"head  [o] Brass Pattern",
		"Lens",
		"tool  —",
		"body  —",
		"charm —",
		"relic —",
		"enter submit · ctrl+h hint · q quit",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("view snapshot is missing %q\n%s", expected, content)
		}
	}
	if !model.View().AltScreen {
		t.Fatal("game view must use the alternate screen")
	}
}
