// Package rangerart renders the reviewed ASCII Motion Ranger export.
package rangerart

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type sourceCell struct {
	x, y  int
	char  rune
	color int
}

type motionMode uint8

const (
	idleMode motionMode = iota
	clearMode
)

type motionStep struct {
	glow  int
	delay time.Duration
}

var idleSteps = []motionStep{
	{glow: 0, delay: 2000 * time.Millisecond},
	{glow: 1, delay: 900 * time.Millisecond},
	{glow: 0, delay: 1200 * time.Millisecond},
	{glow: -1, delay: 1800 * time.Millisecond},
	{glow: 0, delay: 1200 * time.Millisecond},
}

var clearSteps = []motionStep{
	{glow: 2, delay: 220 * time.Millisecond},
	{glow: 3, delay: 320 * time.Millisecond},
	{glow: 2, delay: 460 * time.Millisecond},
	{glow: 1, delay: 700 * time.Millisecond},
}

type tickMsg time.Time

// Model owns cosmetic Ranger animation state. Gameplay never reads it.
type Model struct {
	animated bool
	mode     motionMode
	step     int
}

// New constructs the Ranger renderer for the current motion preference.
func New(animated bool) Model {
	return Model{animated: animated}
}

// Init starts the idle loop when motion is enabled.
func (m Model) Init() tea.Cmd {
	if !m.animated {
		return nil
	}
	return m.tick()
}

func (m Model) tick() tea.Cmd {
	delay := m.steps()[m.step].delay
	return tea.Tick(delay, func(now time.Time) tea.Msg { return tickMsg(now) })
}

// Update advances only messages owned by the Ranger renderer.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if _, ok := msg.(tickMsg); !ok || !m.animated {
		return m, nil
	}
	m.step++
	if m.step >= len(m.steps()) {
		m.step = 0
		if m.mode == clearMode {
			m.mode = idleMode
		}
	}
	return m, m.tick()
}

// Celebrate plays a finite CLEAR glow before returning to idle.
func (m *Model) Celebrate() tea.Cmd {
	m.mode = clearMode
	m.step = 0
	if !m.animated {
		m.step = len(clearSteps) - 1
		return nil
	}
	return m.tick()
}

// Animated reports whether animation ticks are enabled.
func (m Model) Animated() bool { return m.animated }

// SetAnimated applies a runtime motion preference and returns the next tick
// when animation has just been enabled.
func (m *Model) SetAnimated(animated bool) tea.Cmd {
	if m.animated == animated {
		return nil
	}
	m.animated = animated
	if !animated {
		if m.mode == clearMode {
			m.step = len(clearSteps) - 1
		} else {
			m.step = 0
		}
		return nil
	}
	m.step = 0
	return m.tick()
}

// CurrentStep exposes deterministic animation state for tests and diagnostics.
func (m Model) CurrentStep() int { return m.step }

// Celebrating reports whether the finite CLEAR sequence is active.
func (m Model) Celebrating() bool { return m.mode == clearMode }

func (m Model) steps() []motionStep {
	if m.mode == clearMode {
		return clearSteps
	}
	return idleSteps
}

// Render draws only cells from the reviewed 60x60 Bubble Tea export. The
// source is fitted without cropping; terminal cell proportions reduce the
// natural display height to 30 rows at full width.
func (m Model) Render(maxWidth, maxHeight int) string {
	width, height := fittedSize(maxWidth, maxHeight)
	if width == 0 || height == 0 {
		return ""
	}
	step := m.steps()[m.step]
	return renderExport(width, height, step.glow)
}

type sampledCell struct {
	char  rune
	color string
}

func fittedSize(maxWidth, maxHeight int) (width, height int) {
	if maxWidth < 1 || maxHeight < 1 {
		return 0, 0
	}
	naturalHeight := (sourceHeight + 1) / 2
	if maxWidth*naturalHeight <= maxHeight*sourceWidth {
		width = min(maxWidth, sourceWidth)
		height = max(1, (naturalHeight*width+sourceWidth-1)/sourceWidth)
		return width, height
	}
	height = min(maxHeight, naturalHeight)
	width = max(1, sourceWidth*height/naturalHeight)
	return width, height
}

func renderExport(width, height, glow int) string {
	occupied := make(map[[2]int]sourceCell, len(sourceCells))
	for _, cell := range sourceCells {
		occupied[[2]int{cell.x, cell.y}] = cell
	}

	rows := make([]string, height)
	for outputY := range height {
		row := make([]sampledCell, width)
		for outputX := range width {
			sourceX0 := outputX * sourceWidth / width
			sourceX1 := max((outputX+1)*sourceWidth/width, sourceX0+1)
			sourceY0 := outputY * sourceHeight / height
			sourceY1 := max((outputY+1)*sourceHeight/height, sourceY0+1)
			cell, ok := representativeCell(occupied, sourceX0, sourceX1, sourceY0, sourceY1)
			if !ok {
				row[outputX].char = ' '
				continue
			}
			color := sourcePalette[cell.color]
			if isSignalGreen(color) {
				color = signalGlow(glow)
			}
			row[outputX] = sampledCell{char: cell.char, color: color}
		}
		rows[outputY] = renderRow(row)
	}
	return strings.Join(rows, "\n")
}

func representativeCell(occupied map[[2]int]sourceCell, minX, maxX, minY, maxY int) (sourceCell, bool) {
	var best sourceCell
	bestScore := -1
	centerX, centerY := minX+maxX-1, minY+maxY-1
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			cell, ok := occupied[[2]int{x, y}]
			if !ok {
				continue
			}
			distance := abs(2*x-centerX) + abs(2*y-centerY)
			score := glyphWeight(cell.char)*100 - distance
			if score > bestScore {
				best, bestScore = cell, score
			}
		}
	}
	return best, bestScore >= 0
}

func renderRow(cells []sampledCell) string {
	var out, run strings.Builder
	currentColor := ""
	flush := func() {
		if run.Len() == 0 {
			return
		}
		if currentColor == "" {
			out.WriteString(run.String())
		} else {
			out.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(currentColor)).Render(run.String()))
		}
		run.Reset()
	}
	for _, cell := range cells {
		if cell.color != currentColor {
			flush()
			currentColor = cell.color
		}
		run.WriteRune(cell.char)
	}
	flush()
	return out.String()
}

func glyphWeight(char rune) int {
	switch char {
	case '█', '▓', '▄', '▀', '▌', '▐', '@':
		return 3
	case '░', '▒':
		return 2
	default:
		return 1
	}
}

func signalGlow(level int) string {
	colors := [...]string{"#4FAD52", "#68D95E", "#78F46A", "#A4FF96", "#C5FFBA"}
	return colors[min(max(level+1, 0), len(colors)-1)]
}

func isSignalGreen(color string) bool {
	if len(color) != 7 {
		return false
	}
	value, err := strconv.ParseUint(color[1:], 16, 24)
	if err != nil {
		return false
	}
	r := int(value >> 16)
	g := int(value>>8) & 0xff
	b := int(value) & 0xff
	return g > r+24 && g > b+24
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// DebugSummary makes the authored source dimensions visible without exposing
// the generated cell table.
func DebugSummary() string {
	return fmt.Sprintf("%dx%d, %d colored cells", sourceWidth, sourceHeight, len(sourceCells))
}
