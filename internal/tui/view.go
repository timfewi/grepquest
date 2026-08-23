package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/game"
)

const (
	signalGreen   = "#78F46A"
	signalBright  = "#A4FF96"
	panelEdge     = "#294037"
	mutedText     = "#829188"
	softText      = "#EAF2EC"
	warningGold   = "#F2C45E"
	technicalCyan = "#74D7E8"
	dangerRed     = "#FF8C84"
	panelFill     = "#060A09"
)

var (
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(signalGreen)).Bold(true)
	brightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(signalBright)).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(mutedText))
	softStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(softText))
	goldStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(warningGold)).Bold(true)
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(technicalCyan))
	dangerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(dangerRed)).Bold(true)
)

func renderGameView(m Model) tea.View {
	if len(m.challenges) == 0 {
		return tea.NewView("No challenges installed.\n")
	}
	w, h := max(44, m.width), max(24, m.height)
	detailed := !m.compactHUD && w >= 132 && h >= 54
	header := renderHeader(m, w)
	footer := renderFooter(m, w)

	var content string
	if detailed {
		upperHeight := clamp(h*2/3-2, 36, 40)
		terminalHeight := max(15, h-upperHeight-3)
		rightWidth := clamp(w*42/100, 62, 70)
		leftWidth := w - rightWidth - 1
		mission := renderMissionPanel(m, leftWidth, upperHeight, true)
		ranger := renderRangerPanel(m, rightWidth, upperHeight, true)
		upper := lipgloss.JoinHorizontal(lipgloss.Top, mission, ranger)
		terminal := renderTerminalPanel(m, w, terminalHeight, true)
		content = header + "\n" + upper + "\n" + terminal + "\n" + footer + "\n"
	} else if w >= 78 {
		rightWidth := clamp(w/3, 30, 38)
		leftWidth := w - rightWidth - 1
		mission := renderMissionPanel(m, leftWidth, 0, false)
		ranger := renderRangerPanel(m, rightWidth, 0, false)
		upper := lipgloss.JoinHorizontal(lipgloss.Top, mission, ranger)
		terminal := renderTerminalPanel(m, w, 0, false)
		content = header + "\n" + upper + "\n" + terminal + "\n" + footer + "\n"
	} else {
		mission := renderMissionPanel(m, w, 0, false)
		ranger := renderRangerPanel(m, w, 0, false)
		terminal := renderTerminalPanel(m, w, 0, false)
		content = header + "\n" + mission + "\n" + ranger + "\n" + terminal + "\n" + footer + "\n"
	}

	if m.settingsOpen {
		content = renderSettingsOverlay(content, m, w, h)
	}
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func renderHeader(m Model, width int) string {
	room, total, act := min(m.index+1, len(m.challenges)), len(m.challenges), m.current().Act
	left := brightStyle.Render(" GREPQUEST ") + softStyle.Render("// SIGNAL RUN")
	center := mutedStyle.Render(fmt.Sprintf("  ACT %02d  ·  ROOM %02d/%02d", act, room, total))
	pulseFrames := []string{"◇", "◈", "◆", "◈"}
	pulse := pulseFrames[m.ranger.CurrentStep()%len(pulseFrames)]
	right := accentStyle.Render(pulse + " DETERMINISTIC JUDGE ")
	space := max(1, width-lipgloss.Width(left)-lipgloss.Width(center)-lipgloss.Width(right))
	return lipgloss.NewStyle().Width(width).Background(lipgloss.Color("#0B100D")).Render(left + center + strings.Repeat(" ", space) + right)
}

func renderFooter(m Model, width int) string {
	motion := "MOTION ON"
	if !m.ranger.Animated() {
		motion = "MOTION OFF"
	}
	keys := brightStyle.Render(" ENTER ") + softStyle.Render("EXECUTE  ") + brightStyle.Render(" CTRL+H ") + softStyle.Render("HINT  ") + brightStyle.Render(" ? ") + softStyle.Render("HELP / SETTINGS  ") + brightStyle.Render(" Q ") + softStyle.Render("QUIT")
	if width < 110 {
		keys = brightStyle.Render(" ENTER ") + softStyle.Render("RUN  ") + brightStyle.Render(" ? ") + softStyle.Render("HELP  ") + brightStyle.Render(" Q ") + softStyle.Render("QUIT")
	}
	right := mutedStyle.Render(motion)
	space := max(1, width-lipgloss.Width(keys)-lipgloss.Width(right))
	return lipgloss.NewStyle().Width(width).Render(keys + strings.Repeat(" ", space) + right)
}

func renderMissionPanel(m Model, width, height int, detailed bool) string {
	c := m.current()
	status, statusStyle := missionStatus(m)
	title := panelHeading("MISSION BRIEF", cyanStyle.Render(fmt.Sprintf("ROOM %02d  ·  %s", m.index+1, strings.ToUpper(c.Title))), max(20, width-6))
	threat := renderThreat(c.Difficulty)
	target := firstFixture(c)
	progress := renderBar(m.index, len(m.challenges), max(12, min(42, width-38)), signalGreen)
	innerWidth := max(24, width-6)

	var body strings.Builder
	body.WriteString(title)
	body.WriteString("\n\n")
	if detailed {
		body.WriteString(renderMissionOverview(c, m.index, len(m.challenges), innerWidth))
		body.WriteString("\n\n")
	} else {
		body.WriteString(mutedStyle.Render("OBJECTIVE"))
		body.WriteString("\n")
		body.WriteString(brightStyle.Render("› ") + c.Prompt)
		body.WriteString("\n\n")
	}
	body.WriteString(statusStyle.Render(" " + status + " "))
	body.WriteString("  ")
	body.WriteString(softStyle.Render("THREAT ") + threat)
	if detailed {
		body.WriteString("  ")
		body.WriteString(mutedStyle.Render("JUDGE ") + accentStyle.Render("DETERMINISTIC"))
	}
	if detailed {
		body.WriteString("\n\n")
		body.WriteString(mutedStyle.Render("DUNGEON PROGRESS  "))
		body.WriteString(progress)
		body.WriteString(softStyle.Render(fmt.Sprintf("  %02d/%02d CLEARED", m.index, len(m.challenges))))
		body.WriteString("\n\n")
		body.WriteString(renderMissionDataGrid(c, m.hints, innerWidth))
		body.WriteString("\n")
		body.WriteString(mutedStyle.Render("HINTS ") + renderHintCache(m.hints, len(c.Hints)) + softStyle.Render("  CTRL+H DECRYPT") + mutedStyle.Render("    SIGNAL TRACE ") + renderSignalTrace(max(12, min(28, width-68)), m.ranger.CurrentStep()))
	} else {
		body.WriteString("\n")
		body.WriteString(infoRow("TARGET", target, "MODE", strings.Join(c.Tags, "+")))
		body.WriteString("\n")
		body.WriteString(infoRow("EXPECTED EXIT", fmt.Sprint(c.WantExit), "HINTS", fmt.Sprintf("%d/%d", m.hints, len(c.Hints))))
		body.WriteString("\n\n")
		body.WriteString(mutedStyle.Render("DUNGEON PROGRESS"))
		body.WriteString("\n")
		body.WriteString(progress + softStyle.Render(fmt.Sprintf("  %d/%d CLEARED", m.index, len(m.challenges))))
	}

	return panel(width, height, "#3E6654").Render(body.String())
}

func renderRangerPanel(m Model, width, height int, detailed bool) string {
	innerWidth := max(16, width-6)
	artWidth, artHeight := min(32, innerWidth), 7
	if detailed {
		// Give the original 60x60 export enough room to retain its full
		// silhouette. Render handles the terminal cell aspect ratio.
		artWidth = min(60, innerWidth)
		artHeight = clamp(height-16, 20, 30)
	}
	art := m.ranger.Render(artWidth, artHeight)
	art = lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(art)
	xpInLevel := m.profile.XP % 300
	xpBarWidth := max(8, min(20, innerWidth-12))

	var body strings.Builder
	rank := softStyle.Render(fmt.Sprintf("LEVEL %02d", m.profile.Level)) + mutedStyle.Render("  ·  ") + goldStyle.Render(fmt.Sprintf("XP %03d", m.profile.XP))
	body.WriteString(panelHeading("RANGER", rank, innerWidth))
	body.WriteString("\n")
	body.WriteString(mutedStyle.Render("PATTERN SEEKER  ·  LINK ") + accentStyle.Render("LOCAL"))
	body.WriteString("\n")
	body.WriteString(art)
	body.WriteString("\n")
	if detailed {
		body.WriteString(mutedStyle.Render("XP  "))
		body.WriteString(renderBar(xpInLevel, 300, xpBarWidth, signalGreen))
		body.WriteString(softStyle.Render(fmt.Sprintf("  %03d/300", xpInLevel)))
		body.WriteString("\n")
		body.WriteString(softStyle.Render(fmt.Sprintf("CLEARS %02d  ·  HINTS %02d", len(m.profile.Solved), m.profile.HintsUsed)))
		body.WriteString(mutedStyle.Render("    MASTERY  ") + renderMasteryInline(m.profile, m.current().Tags, innerWidth-38))
		body.WriteString("\n")
		body.WriteString(mutedStyle.Render("LOADOUT  ") + renderLoadoutSummary(m.profile, innerWidth-9))
	} else {
		body.WriteString(mutedStyle.Render("XP TO NEXT LEVEL"))
		body.WriteString("\n")
		body.WriteString(renderBar(xpInLevel, 300, xpBarWidth, signalGreen))
		body.WriteString(softStyle.Render(fmt.Sprintf("  %03d/300", xpInLevel)))
		body.WriteString("\n")
		body.WriteString(softStyle.Render(fmt.Sprintf("CLEARS %02d  ·  HINTS %02d", len(m.profile.Solved), m.profile.HintsUsed)))
	}
	return panel(width, height, panelEdge).Render(body.String())
}

func renderTerminalPanel(m Model, width, height int, detailed bool) string {
	status, statusStyle := terminalStatus(m.feedback)
	title := panelHeading("COMMAND BUFFER", statusStyle.Render(status), max(20, width-6))
	feedback := m.feedback
	if !detailed {
		body := title + "\n" + brightStyle.Render("❯ ") + m.input.View()
		if feedback != "" {
			body += "\n" + styleFeedback(feedback)
		}
		return panel(width, height, "#607067").Render(body)
	}
	if feedback == "" {
		feedback = mutedStyle.Render("Awaiting a GNU grep command. The local judge verifies output and exit status.")
		feedback += "\n" + softStyle.Render("FORM  ") + accentStyle.Render("grep PATTERN FILE") + mutedStyle.Render("  ·  submit when the signal is stable")
	} else {
		feedback = styleFeedback(feedback)
	}
	contract := mutedStyle.Render("ACTIVE CONTRACT  ") + cyanStyle.Render(firstFixture(m.current())) + mutedStyle.Render("  ·  EXIT ") + softStyle.Render(fmt.Sprint(m.current().WantExit)) + mutedStyle.Render("  ·  ") + softStyle.Render(strings.ToUpper(strings.Join(m.current().Tags, " / ")))
	telemetry := mutedStyle.Render("BUFFER ") + softStyle.Render(fmt.Sprintf("%03d/512", len([]rune(m.input.Value())))) + mutedStyle.Render("  ·  PARSER GNU GREP  ·  JUDGE LOCAL  ·  FOCUS COMMAND")
	judgeStatus := "AWAITING TRANSMISSION"
	if m.feedback != "" {
		judgeStatus = status
	}
	body := title + "\n\n" + contract + "\n\n" + brightStyle.Render("❯ ") + m.input.View() + "\n\n" + mutedStyle.Render("JUDGE OUTPUT  ·  ") + statusStyle.Render(judgeStatus) + "\n" + feedback + "\n\n" + mutedStyle.Render("RUN SEQUENCE") + "\n" + renderSequenceRail(m.ranger.CurrentStep()) + "\n\n" + telemetry
	return panel(width, height, panelEdge).Render(body)
}

func renderMissionOverview(c challenge.Challenge, index, total, width int) string {
	scene := renderRoomScene(c, index, total)
	gap := "   "
	briefWidth := max(24, width-lipgloss.Width(scene)-lipgloss.Width(gap))
	brief := mutedStyle.Render("OBJECTIVE") + "\n" + brightStyle.Render("› ") + softStyle.Render(wrapText(c.Prompt, max(12, briefWidth-2)))
	brief += "\n\n" + mutedStyle.Render("SIGNAL PROFILE") + "\n"
	brief += mutedStyle.Render("TARGET ") + cyanStyle.Render(firstFixture(c)) + mutedStyle.Render("  ·  CLASS ") + softStyle.Render(strings.ToUpper(strings.Join(c.Tags, "/"))) + "\n"
	brief += telemetryValue("FILES", len(c.Fixtures)) + "  " + telemetryValue("CHECKS", 1+len(c.HiddenTests))
	return lipgloss.JoinHorizontal(lipgloss.Top, scene, gap, brief)
}

func renderMissionDataGrid(c challenge.Challenge, hints, width int) string {
	gap := "    "
	leftWidth := max(28, width*55/100)
	rightWidth := max(24, width-leftWidth-lipgloss.Width(gap))
	target := firstFixture(c)
	left := mutedStyle.Render("VISIBLE INPUT  ") + cyanStyle.Render(target) + "\n" + renderFixturePreview(c, leftWidth, 4)

	noHintXP := 100 + c.Difficulty*20
	rightLines := []string{
		mutedStyle.Render("JUDGE CONTRACT"),
		contractRow("EXPECTED EXIT", fmt.Sprint(c.WantExit), rightWidth),
		contractRow("VISIBLE CHECK", "01", rightWidth),
		contractRow("HIDDEN VARIANTS", fmt.Sprintf("%02d", len(c.HiddenTests)), rightWidth),
		contractRow("NO-HINT REWARD", fmt.Sprintf("%d XP", noHintXP), rightWidth),
		contractRow("HINT CACHE", fmt.Sprintf("%d/%d", hints, len(c.Hints)), rightWidth),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, gap, strings.Join(rightLines, "\n"))
}

func contractRow(label, value string, width int) string {
	labelWidth := min(18, max(12, width-lipgloss.Width(value)-2))
	return mutedStyle.Render(fmt.Sprintf("%-*s", labelWidth, label)) + brightStyle.Render(value)
}

func renderFixturePreview(c challenge.Challenge, width, maxLines int) string {
	contents := c.Fixtures[firstFixture(c)]
	lines := strings.Split(strings.TrimSuffix(contents, "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{"(empty file)"}
	}
	visible := min(len(lines), maxLines)
	result := make([]string, 0, visible+1)
	for index := 0; index < visible; index++ {
		line := sanitizeTerminalText(lines[index])
		line = truncate(line, max(1, width-6))
		result = append(result, mutedStyle.Render(fmt.Sprintf("%02d │ ", index+1))+softStyle.Render(line))
	}
	if len(lines) > visible {
		result = append(result, mutedStyle.Render(fmt.Sprintf("   └─ %d more lines", len(lines)-visible)))
	}
	return strings.Join(result, "\n")
}

func sanitizeTerminalText(value string) string {
	return strings.Map(func(char rune) rune {
		switch {
		case char == '\t':
			return ' '
		case char < ' ' || char == 0x7f:
			return '�'
		default:
			return char
		}
	}, value)
}

func panel(width, height int, edge string) lipgloss.Style {
	style := lipgloss.NewStyle().Width(width).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(edge)).Background(lipgloss.Color(panelFill)).Padding(1, 2)
	if height > 0 {
		style = style.Height(height)
	}
	return style
}

func panelHeading(label, value string, width int) string {
	left := accentStyle.Render(label)
	space := max(2, width-lipgloss.Width(left)-lipgloss.Width(value))
	line := left + strings.Repeat(" ", space) + value
	return line + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(panelEdge)).Render(strings.Repeat("─", width))
}

func missionStatus(m Model) (string, lipgloss.Style) {
	if m.done || strings.HasPrefix(m.feedback, "CLEAR") || strings.HasPrefix(m.feedback, "ACT COMPLETE") {
		return "ROOM CLEARED", accentStyle.Background(lipgloss.Color("#163D22"))
	}
	if strings.HasPrefix(m.feedback, "Not yet") || strings.HasPrefix(m.feedback, "Cannot judge") {
		return "SIGNAL REJECTED", dangerStyle.Background(lipgloss.Color("#3A1818"))
	}
	if strings.HasPrefix(m.feedback, "HINT") {
		return "HINT DECRYPTED", goldStyle.Background(lipgloss.Color("#392F16"))
	}
	return "AWAITING COMMAND", accentStyle.Background(lipgloss.Color("#163D22"))
}

func terminalStatus(feedback string) (string, lipgloss.Style) {
	if strings.HasPrefix(feedback, "CLEAR") || strings.HasPrefix(feedback, "ACT COMPLETE") {
		return "CLEAR", accentStyle
	}
	if strings.HasPrefix(feedback, "Not yet") || strings.HasPrefix(feedback, "Cannot judge") {
		return "RETRY", dangerStyle
	}
	if strings.HasPrefix(feedback, "HINT") {
		return "HINT", goldStyle
	}
	return "READY", accentStyle
}

func styleFeedback(feedback string) string {
	switch {
	case strings.HasPrefix(feedback, "CLEAR"), strings.HasPrefix(feedback, "ACT COMPLETE"):
		return accentStyle.Render(feedback)
	case strings.HasPrefix(feedback, "Not yet"), strings.HasPrefix(feedback, "Cannot judge"):
		return dangerStyle.Render(feedback)
	case strings.HasPrefix(feedback, "HINT"):
		return goldStyle.Render(feedback)
	default:
		return feedback
	}
}

func renderThreat(level int) string {
	return dangerStyle.Render(strings.Repeat("◆", level)) + mutedStyle.Render(strings.Repeat("◇", max(0, 5-level)))
}

func telemetryValue(label string, value int) string {
	return mutedStyle.Render(label+" ") + brightStyle.Render(fmt.Sprintf("%02d", value))
}

func renderSequenceRail(step int) string {
	stages := []string{"ROOM", "COMMAND", "MATCH", "CLEAR", "DROP", "REST"}
	active := step % len(stages)
	parts := make([]string, 0, len(stages))
	for index, stage := range stages {
		label := fmt.Sprintf("%02d %s", index+1, stage)
		if index == active {
			parts = append(parts, brightStyle.Render(label))
		} else if stage == "DROP" {
			parts = append(parts, goldStyle.Render(label))
		} else {
			parts = append(parts, mutedStyle.Render(label))
		}
	}
	return strings.Join(parts, lipgloss.NewStyle().Foreground(lipgloss.Color(panelEdge)).Render(" ── "))
}

func renderHintCache(used, total int) string {
	used = clamp(used, 0, total)
	return goldStyle.Render(strings.Repeat("◆", used)) + mutedStyle.Render(strings.Repeat("◇", total-used))
}

func renderSignalTrace(width, step int) string {
	position := (step * 5) % width
	return mutedStyle.Render(strings.Repeat("·", position)) + brightStyle.Render("◆") + mutedStyle.Render(strings.Repeat("·", width-position-1))
}

func renderBar(value, total, width int, color string) string {
	if total < 1 {
		total = 1
	}
	value = clamp(value, 0, total)
	filled := value * width / total
	if value > 0 && filled == 0 {
		filled = 1
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.Repeat("█", filled)) + mutedStyle.Render(strings.Repeat("░", width-filled))
}

func infoRow(leftLabel, leftValue, rightLabel, rightValue string) string {
	return mutedStyle.Render(fmt.Sprintf("%-14s", leftLabel)) + softStyle.Render(fmt.Sprintf("%-20s", leftValue)) + mutedStyle.Render(fmt.Sprintf("%-14s", rightLabel)) + softStyle.Render(rightValue)
}

func firstFixture(c challenge.Challenge) string {
	keys := make([]string, 0, len(c.Fixtures))
	for name := range c.Fixtures {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "—"
	}
	return keys[0]
}

func renderMasteryInline(profile game.Profile, currentTags []string, width int) string {
	seen := make(map[string]bool)
	keys := make([]string, 0, len(profile.Mastery)+len(currentTags))
	for key := range profile.Mastery {
		seen[key] = true
		keys = append(keys, key)
	}
	for _, key := range currentTags {
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) > 3 {
		keys = keys[:3]
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", strings.ToUpper(key), profile.Mastery[key]))
	}
	if len(parts) == 0 {
		return mutedStyle.Render("NO DATA")
	}
	return softStyle.Render(truncate(strings.Join(parts, "  ·  "), width))
}

func renderLoadoutSummary(profile game.Profile, width int) string {
	labels := map[game.Slot]string{game.Head: "HEAD", game.Tool: "TOOL", game.Body: "BODY", game.Charm: "CHARM", game.Relic: "RELIC"}
	parts := make([]string, 0, 5)
	for _, slot := range []game.Slot{game.Head, game.Tool, game.Body, game.Charm, game.Relic} {
		id := profile.Loadout[slot]
		if id == "" {
			continue
		}
		for _, item := range game.BuiltinItems {
			if item.ID == id {
				parts = append(parts, labels[slot]+" "+item.Sprite+" "+item.Name)
				break
			}
		}
	}
	if len(parts) == 0 {
		return mutedStyle.Render("EMPTY  ·  CLEAR ROOMS TO FIND GEAR")
	}
	return goldStyle.Render(truncate(strings.Join(parts, "  ·  "), width))
}

func wrapText(value string, width int) string {
	words := strings.Fields(value)
	if len(words) == 0 || width < 1 {
		return ""
	}
	lines := []string{words[0]}
	for _, word := range words[1:] {
		last := len(lines) - 1
		if lipgloss.Width(lines[last])+1+lipgloss.Width(word) <= width {
			lines[last] += " " + word
			continue
		}
		lines = append(lines, word)
	}
	return strings.Join(lines, "\n")
}

func truncate(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func clamp(value, low, high int) int {
	return min(max(value, low), high)
}
