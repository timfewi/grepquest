package rangerart

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func TestGeneratedSourceSummary(t *testing.T) {
	t.Parallel()
	if got, want := DebugSummary(), "60x60, 1605 colored cells"; got != want {
		t.Fatalf("DebugSummary() = %q, want %q", got, want)
	}
}

func TestRenderRespectsResponsiveBounds(t *testing.T) {
	t.Parallel()
	for _, size := range []struct {
		width  int
		height int
	}{{23, 10}, {23, 20}, {45, 20}} {
		output := New(false).Render(size.width, size.height)
		if strings.TrimSpace(output) == "" {
			t.Fatalf("Render(%d, %d) returned no visible art", size.width, size.height)
		}
		lines := strings.Split(output, "\n")
		if len(lines) > size.height {
			t.Fatalf("Render(%d, %d) used %d rows", size.width, size.height, len(lines))
		}
		for lineNumber, line := range lines {
			if width := lipgloss.Width(line); width > size.width {
				t.Fatalf("Render(%d, %d) line %d used %d columns", size.width, size.height, lineNumber, width)
			}
		}
	}
}

func TestRenderUsesReviewedExportCells(t *testing.T) {
	t.Parallel()
	wantCells := map[[2]int]rune{
		{28, 1}: '╬',
		{27, 2}: '┘',
		{29, 2}: '╗',
		{26, 3}: '▄',
		{27, 3}: '@',
	}
	for coordinate, want := range wantCells {
		found := false
		for _, cell := range sourceCells {
			if cell.x == coordinate[0] && cell.y == coordinate[1] {
				found = true
				if cell.char != want {
					t.Fatalf("source cell %v = %q, want %q", coordinate, cell.char, want)
				}
				break
			}
		}
		if !found {
			t.Fatalf("source cell %v is missing", coordinate)
		}
	}

	output := New(false).Render(60, 30)
	for _, glyph := range []string{"╬", "@", "█", "▒", "┘"} {
		if !strings.Contains(output, glyph) {
			t.Fatalf("render is missing exported glyph %q:\n%s", glyph, output)
		}
	}
	if strings.Contains(output, "[Q]") {
		t.Fatalf("render still contains the discarded line-art Ranger:\n%s", output)
	}
}

func TestMotionUsesSlowIdleGlow(t *testing.T) {
	t.Parallel()
	var total time.Duration
	for _, step := range idleSteps {
		total += step.delay
	}
	if total < 7*time.Second {
		t.Fatalf("idle loop lasts %s, want at least 7s", total)
	}

	var clearTotal time.Duration
	for _, step := range clearSteps {
		clearTotal += step.delay
	}
	if clearTotal < 1500*time.Millisecond {
		t.Fatalf("CLEAR sequence lasts %s, want at least 1.5s", clearTotal)
	}
}

func TestRenderPreservesCanvasWhitespace(t *testing.T) {
	t.Parallel()
	output := New(false).Render(45, 20)
	lines := strings.Split(output, "\n")
	wantWidth := lipgloss.Width(lines[0])
	if wantWidth == 0 {
		t.Fatal("render returned an empty canvas")
	}
	for lineNumber, line := range lines {
		if got := lipgloss.Width(line); got != wantWidth {
			t.Fatalf("render line %d has width %d, want fixed canvas width %d", lineNumber, got, wantWidth)
		}
	}
	if !strings.HasPrefix(lines[0], " ") || !strings.HasSuffix(lines[0], " ") {
		t.Fatalf("render must retain whitespace on both sides of the silhouette: %q", lines[0])
	}
}

func TestCelebrateReturnsToIdle(t *testing.T) {
	t.Parallel()
	model := New(true)
	if cmd := model.Celebrate(); cmd == nil {
		t.Fatal("animated CLEAR sequence must schedule its first tick")
	}
	if !model.Celebrating() {
		t.Fatal("Celebrate did not enter CLEAR mode")
	}
	for range clearSteps {
		next, cmd := model.Update(tickMsg{})
		if cmd == nil {
			t.Fatal("animated Ranger stopped scheduling ticks")
		}
		model = next
	}
	if model.Celebrating() || model.CurrentStep() != 0 {
		t.Fatalf("CLEAR sequence did not return to idle: celebrating=%v step=%d", model.Celebrating(), model.CurrentStep())
	}
}

func TestReducedMotionUsesCompleteStaticClearFrame(t *testing.T) {
	t.Parallel()
	model := New(false)
	if model.Init() != nil {
		t.Fatal("reduced-motion Ranger scheduled an idle tick")
	}
	if cmd := model.Celebrate(); cmd != nil {
		t.Fatal("reduced-motion Ranger scheduled a CLEAR tick")
	}
	if !model.Celebrating() || model.CurrentStep() != len(clearSteps)-1 {
		t.Fatalf("static CLEAR frame not selected: celebrating=%v step=%d", model.Celebrating(), model.CurrentStep())
	}
	before := model.CurrentStep()
	model, cmd := model.Update(tickMsg{})
	if cmd != nil || model.CurrentStep() != before {
		t.Fatal("reduced-motion Ranger advanced after a tick")
	}
}

func TestMotionCanBeChangedAtRuntime(t *testing.T) {
	t.Parallel()
	model := New(false)
	if cmd := model.SetAnimated(true); cmd == nil || !model.Animated() {
		t.Fatal("enabling motion did not start animation")
	}
	if cmd := model.SetAnimated(false); cmd != nil || model.Animated() || model.CurrentStep() != 0 {
		t.Fatalf("disabling motion did not restore static idle: animated=%v step=%d", model.Animated(), model.CurrentStep())
	}
}
