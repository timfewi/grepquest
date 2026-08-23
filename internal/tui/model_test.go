package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/game"
)

func TestImmersiveViewSnapshot(t *testing.T) {
	t.Parallel()
	profile := game.NewProfile()
	profile.XP = 320
	profile.Level = 2
	profile.Mastery["literal"] = 80
	profile.Loadout[game.Head] = game.BuiltinItems[0].ID
	model := New(game.Engine{}, challenge.Core()[:1], profile, nil)
	model.width = 160
	model.height = 58
	content := model.View().Content

	for _, expected := range []string{
		"GREPQUEST",
		"SIGNAL RUN",
		"ACT 01",
		"ROOM 01/01",
		"DETERMINISTIC JUDGE",
		"MISSION BRIEF",
		"ROOM 01  ·  FIRST SIGNAL",
		"Print lines containing ERROR from vault.log.",
		"SIGNAL CELLAR / ROOM 01",
		"@---+---+---[Q]",
		"AWAITING COMMAND",
		"THREAT",
		"TARGET",
		"vault.log",
		"EXPECTED EXIT",
		"DUNGEON PROGRESS",
		"VISIBLE INPUT",
		"JUDGE CONTRACT",
		"NO-HINT REWARD",
		"RUN SEQUENCE",
		"GNU GREP",
		"HINT CACHE",
		"ACTIVE CONTRACT",
		"SIGNAL TRACE",
		"RANGER",
		"LEVEL 02",
		"PATTERN SEEKER",
		"CLEARS",
		"literal",
		"LOADOUT",
		"HEAD",
		"[o] Brass Pattern",
		"Lens",
		"COMMAND BUFFER",
		"READY",
		"EXECUTE",
		"HELP / SETTINGS",
		"QUIT",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("view snapshot is missing %q\n%s", expected, content)
		}
	}
	if !model.View().AltScreen {
		t.Fatal("game view must use the alternate screen")
	}
	if got := lipgloss.Width(content); got > model.width {
		t.Fatalf("immersive view width = %d, want at most %d", got, model.width)
	}
	if got := lipgloss.Height(content); got > model.height {
		t.Fatalf("immersive view height = %d, want at most %d", got, model.height)
	}
}

func TestReducedMotionKeepsStaticRanger(t *testing.T) {
	t.Setenv("GREPQUEST_REDUCED_MOTION", "1")
	model := New(game.Engine{}, challenge.Core()[:1], game.NewProfile(), nil)
	if model.ranger.Animated() {
		t.Fatal("reduced motion must disable ranger animation")
	}
}

func TestHelpOverlayExplainsControlsAndChangesSettings(t *testing.T) {
	t.Parallel()
	model := New(game.Engine{}, challenge.Core()[:1], game.NewProfile(), nil)
	model.width = 160
	model.height = 58

	updated, _ := model.Update(keyPress("?", '?'))
	got := updated.(Model)
	if !got.settingsOpen || got.input.Focused() {
		t.Fatalf("help key did not open the overlay and blur input: open=%v focused=%v", got.settingsOpen, got.input.Focused())
	}
	content := got.View().Content
	for _, expected := range []string{"HELP & SETTINGS", "QUICK HELP", "GREP GUIDE", "HOW TO PLAY", "HUD panels are read-only", "quit when buffer is empty"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("settings overlay is missing %q", expected)
		}
	}
	if lipgloss.Width(content) > got.width || lipgloss.Height(content) > got.height {
		t.Fatalf("settings overlay exceeds viewport: %dx%d", lipgloss.Width(content), lipgloss.Height(content))
	}

	updated, _ = got.Update(keyPress("3", '3'))
	got = updated.(Model)
	updated, _ = got.Update(keyPress("down", tea.KeyDown))
	got = updated.(Model)
	if got.settingsIndex != hudSetting {
		t.Fatalf("settings index = %d, want HUD setting", got.settingsIndex)
	}
	updated, _ = got.Update(keyPress("enter", tea.KeyEnter))
	got = updated.(Model)
	if !got.compactHUD {
		t.Fatal("enter did not toggle compact HUD")
	}
	updated, _ = got.Update(keyPress("esc", tea.KeyEscape))
	got = updated.(Model)
	if got.settingsOpen || !got.input.Focused() {
		t.Fatalf("escape did not close overlay and restore input: open=%v focused=%v", got.settingsOpen, got.input.Focused())
	}
}

func TestQCanBeTypedInsideCommand(t *testing.T) {
	t.Parallel()
	model := New(game.Engine{}, challenge.Core()[:1], game.NewProfile(), nil)
	model.input.SetValue("grep ")
	updated, _ := model.Update(keyPress("q", 'q'))
	got := updated.(Model)
	if value := got.input.Value(); value != "grep q" {
		t.Fatalf("q key changed command to %q, want %q", value, "grep q")
	}
}

func keyPress(text string, code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: code})
}

func TestStandardViewFitsCommonTerminal(t *testing.T) {
	t.Parallel()
	model := New(game.Engine{}, challenge.Core()[:1], game.NewProfile(), nil)
	model.width = 100
	model.height = 30
	content := model.View().Content
	if got := lipgloss.Width(content); got > model.width {
		t.Fatalf("standard view width = %d, want at most %d", got, model.width)
	}
	if got := lipgloss.Height(content); got > model.height {
		t.Fatalf("standard view height = %d, want at most %d", got, model.height)
	}
	for _, expected := range []string{"MISSION BRIEF", "RANGER", "COMMAND BUFFER", "DUNGEON PROGRESS"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("standard view is missing %q", expected)
		}
	}
}

func TestCompactHelpOverlayFitsCommonTerminal(t *testing.T) {
	t.Parallel()
	model := New(game.Engine{}, challenge.Core()[:1], game.NewProfile(), nil)
	model.width = 100
	model.height = 30
	model.settingsOpen = true
	content := model.View().Content
	if got := lipgloss.Width(content); got > model.width {
		t.Fatalf("compact overlay width = %d, want at most %d", got, model.width)
	}
	if got := lipgloss.Height(content); got > model.height {
		t.Fatalf("compact overlay height = %d, want at most %d", got, model.height)
	}
	for _, expected := range []string{"HELP & SETTINGS", "QUICK HELP", "HUD panels are read-only"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("compact overlay is missing %q", expected)
		}
	}
}

func TestNewbieGuideIsAvailableInsideHelp(t *testing.T) {
	t.Parallel()
	model := New(game.Engine{}, challenge.Core()[:1], game.NewProfile(), nil)
	model.width = 100
	model.height = 30

	updated, _ := model.Update(keyPress("?", '?'))
	got := updated.(Model)
	updated, _ = got.Update(keyPress("2", '2'))
	got = updated.(Model)
	if !got.settingsOpen || got.helpPage != grepGuidePage || got.input.Focused() {
		t.Fatalf("help menu did not show guide: open=%v page=%d focused=%v", got.settingsOpen, got.helpPage, got.input.Focused())
	}
	content := got.View().Content
	for _, expected := range []string{"GREP FOR FIRST-TIMERS", "grep 'PATTERN' FILE", "grep -i 'warning' app.log", "-w", "Exit 0 = match"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("newbie guide is missing %q", expected)
		}
	}
	if lipgloss.Width(content) > got.width || lipgloss.Height(content) > got.height {
		t.Fatalf("newbie guide exceeds viewport: %dx%d", lipgloss.Width(content), lipgloss.Height(content))
	}
}

func TestGStartsGrepInsteadOfOpeningGuide(t *testing.T) {
	t.Parallel()
	model := New(game.Engine{}, challenge.Core()[:1], game.NewProfile(), nil)

	updated, _ := model.Update(keyPress("g", 'g'))
	got := updated.(Model)
	if got.settingsOpen {
		t.Fatal("g opened help while typing a command")
	}
	if value := got.input.Value(); value != "g" {
		t.Fatalf("input value = %q, want %q", value, "g")
	}
}

func TestRoomScenesCoverBothActs(t *testing.T) {
	t.Parallel()
	core := challenge.Core()
	if got := renderRoomScene(core[0], 0, len(core)); !strings.Contains(got, "SIGNAL CELLAR") {
		t.Fatalf("act 1 scene missing cellar identity:\n%s", got)
	}
	if got := renderRoomScene(core[6], 6, len(core)); !strings.Contains(got, "REGEX FOUNDRY") || !strings.Contains(got, "@=={ }==[ ]==[Q]") {
		t.Fatalf("act 2 scene missing foundry identity:\n%s", got)
	}
	for _, scene := range []string{renderRoomScene(core[0], 0, len(core)), renderRoomScene(core[6], 6, len(core))} {
		for lineNumber, line := range strings.Split(scene, "\n") {
			if len(line) != 31 {
				t.Fatalf("room scene line %d is %d columns wide, want 31: %q", lineNumber, len(line), line)
			}
		}
	}
}

func TestFixturePreviewNeutralizesTerminalControls(t *testing.T) {
	t.Parallel()
	if got := sanitizeTerminalText("safe\x1b[31m\ttext"); got != "safe�[31m text" {
		t.Fatalf("sanitizeTerminalText() = %q", got)
	}
}
