package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	quickHelpPage = iota
	grepGuidePage
	sessionSettingsPage
	helpPageCount
)

const (
	motionSetting = iota
	hudSetting
	settingsCount
)

var keyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#071008")).
	Background(lipgloss.Color(signalBright)).
	Bold(true).
	Padding(0, 1)

func renderSettingsOverlay(base string, m Model, width, height int) string {
	modalWidth := min(78, max(42, width-8))
	compact := height < 34 || modalWidth < 64
	modal := settingsModal(m, modalWidth, compact)
	modalHeight := lipgloss.Height(modal)
	x := max(1, (width-modalWidth)/2)
	y := max(1, (height-modalHeight)/2)

	base = lipgloss.NewStyle().Width(width).Height(height).Render(base)
	shadow := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Background(lipgloss.Color("#050806")).
		Render("")
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(base).X(0).Y(0).Z(0),
		lipgloss.NewLayer(shadow).X(min(width-modalWidth, x+1)).Y(min(height-modalHeight, y+1)).Z(1),
		lipgloss.NewLayer(modal).X(x).Y(y).Z(2),
	}
	return lipgloss.NewCompositor(layers...).Render()
}

func settingsModal(m Model, width int, compact bool) string {
	var body strings.Builder
	body.WriteString(brightStyle.Render("GREPQUEST") + softStyle.Render("  //  HELP & SETTINGS"))
	body.WriteString("\n")
	body.WriteString(helpTabs(m.helpPage))
	body.WriteString("\n\n")
	switch m.helpPage {
	case grepGuidePage:
		body.WriteString(grepGuide(compact))
	case sessionSettingsPage:
		body.WriteString(sessionSettings(m, compact))
	default:
		body.WriteString(quickHelp(compact))
	}
	body.WriteString("\n\n")
	body.WriteString(mutedStyle.Render("Switch page  ") + keyStyle.Render("TAB") + mutedStyle.Render(" / ") + keyStyle.Render("1–3") + mutedStyle.Render("   Close  ") + keyStyle.Render("ESC"))

	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(signalGreen)).
		Background(lipgloss.Color("#0B100D")).
		Padding(1, 2).
		Render(body.String())
}

func helpTabs(active int) string {
	labels := []string{"1 QUICK HELP", "2 GREP GUIDE", "3 SETTINGS"}
	parts := make([]string, len(labels))
	for index, label := range labels {
		if index == active {
			parts[index] = keyStyle.Render(label)
		} else {
			parts[index] = mutedStyle.Render("[ " + label + " ]")
		}
	}
	return strings.Join(parts, "  ")
}

func quickHelp(compact bool) string {
	var body strings.Builder
	body.WriteString(accentStyle.Render("HOW TO PLAY"))
	body.WriteString("\n")
	body.WriteString(softStyle.Render("Read the objective and target file, then type a complete grep command in the command buffer. HUD panels are read-only."))
	if compact {
		body.WriteString("\n")
		body.WriteString(softStyle.Render("HUD panels are read-only. A wrong answer is safe: inspect the feedback, edit, and retry."))
		body.WriteString("\n\n")
		body.WriteString(keyStyle.Render("ENTER") + softStyle.Render(" execute  ") + keyStyle.Render("CTRL+H") + softStyle.Render(" hint"))
		body.WriteString("\n")
		body.WriteString(keyStyle.Render("? / F1") + softStyle.Render(" help  ") + keyStyle.Render("ESC") + softStyle.Render(" close"))
		body.WriteString("\n")
		body.WriteString(keyStyle.Render("Q") + softStyle.Render(" quit with empty buffer  ") + keyStyle.Render("CTRL+C") + softStyle.Render(" quit now"))
	} else {
		body.WriteString("\n\n")
		body.WriteString(mutedStyle.Render("1 READ  ") + softStyle.Render("Find the requested lines and note the target filename."))
		body.WriteString("\n")
		body.WriteString(mutedStyle.Render("2 BUILD ") + softStyle.Render("Enter a complete command such as ") + accentStyle.Render("grep ERROR vault.log"))
		body.WriteString("\n")
		body.WriteString(mutedStyle.Render("3 CHECK ") + softStyle.Render("Press Enter. The local judge compares output and exit status."))
		body.WriteString("\n")
		body.WriteString(mutedStyle.Render("4 RETRY ") + softStyle.Render("Wrong answers are safe. Use the feedback or reveal a hint."))
		body.WriteString("\n\n")
		body.WriteString(controlRow("ENTER", "execute the command", "CTRL+H", "decrypt next hint"))
		body.WriteString("\n")
		body.WriteString(controlRow("← / →", "move input cursor", "? / F1", "open or close this overlay"))
		body.WriteString("\n")
		body.WriteString(controlRow("Q", "quit when buffer is empty", "CTRL+C", "quit immediately"))
		body.WriteString("\n")
		body.WriteString(controlRow("BACKSPACE", "delete character", "CTRL+W", "delete previous word"))
	}
	return body.String()
}

func grepGuide(compact bool) string {
	var body strings.Builder
	body.WriteString(accentStyle.Render("GREP FOR FIRST-TIMERS"))
	body.WriteString("\n")
	body.WriteString(softStyle.Render("grep prints lines that match a pattern. Start with:"))
	body.WriteString("\n")
	body.WriteString(brightStyle.Render("  grep 'PATTERN' FILE"))
	body.WriteString("\n\n")
	body.WriteString(mutedStyle.Render("EXAMPLES"))
	body.WriteString("\n")
	body.WriteString(accentStyle.Render("grep 'ERROR' app.log") + softStyle.Render("       lines containing ERROR"))
	body.WriteString("\n")
	body.WriteString(accentStyle.Render("grep -i 'warning' app.log") + softStyle.Render("  ignore upper/lower case"))
	body.WriteString("\n")
	body.WriteString(accentStyle.Render("grep -n '^TODO' notes.txt") + softStyle.Render(" show numbers; ^ means line start"))
	if compact {
		body.WriteString("\n\n")
		body.WriteString(softStyle.Render("Useful: ") + brightStyle.Render("-v") + softStyle.Render(" invert  ") + brightStyle.Render("-w") + softStyle.Render(" whole word  ") + brightStyle.Render("-E") + softStyle.Render(" extended regex"))
		body.WriteString("\n")
		body.WriteString(mutedStyle.Render("Quote patterns. Exit 0 = match, 1 = no match, 2 = error."))
		return body.String()
	}
	body.WriteString("\n\n")
	body.WriteString(mutedStyle.Render("USEFUL OPTIONS"))
	body.WriteString("\n")
	body.WriteString(optionRow("-i", "ignore case", "-v", "show non-matching lines", "-n", "show line numbers"))
	body.WriteString("\n")
	body.WriteString(optionRow("-w", "match whole words", "-F", "literal text", "-E", "extended regex"))
	body.WriteString("\n\n")
	body.WriteString(mutedStyle.Render("PATTERN BUILDING"))
	body.WriteString("\n")
	body.WriteString(softStyle.Render("^ start of line   $ end of line   . any character   [abc] one listed character"))
	body.WriteString("\n")
	body.WriteString(softStyle.Render("With -E:  + one or more   ? optional   | either   ( ) group"))
	body.WriteString("\n\n")
	body.WriteString(goldStyle.Render("TIP  ") + softStyle.Render("Put patterns in single quotes so the shell cannot reinterpret special characters."))
	body.WriteString("\n")
	body.WriteString(mutedStyle.Render("EXIT STATUS  ") + brightStyle.Render("0") + softStyle.Render(" match  ") + brightStyle.Render("1") + softStyle.Render(" no match  ") + brightStyle.Render("2") + softStyle.Render(" error"))
	return body.String()
}

func optionRow(firstKey, firstText, secondKey, secondText, thirdKey, thirdText string) string {
	return brightStyle.Render(firstKey) + softStyle.Render(" "+firstText+"   ") + brightStyle.Render(secondKey) + softStyle.Render(" "+secondText+"   ") + brightStyle.Render(thirdKey) + softStyle.Render(" "+thirdText)
}

func sessionSettings(m Model, compact bool) string {
	var body strings.Builder
	body.WriteString(accentStyle.Render("SESSION SETTINGS"))
	body.WriteString("\n")
	body.WriteString(settingRow(m.settingsIndex == motionSetting, "MOTION", onOff(m.ranger.Animated()), "Ranger, judge pulse, and signal trace"))
	body.WriteString("\n")
	hudValue := "AUTO"
	if m.compactHUD {
		hudValue = "COMPACT"
	}
	body.WriteString(settingRow(m.settingsIndex == hudSetting, "HUD DETAIL", hudValue, "Auto expands when the terminal has room"))
	body.WriteString("\n\n")
	if compact {
		body.WriteString(keyStyle.Render("↑/↓") + mutedStyle.Render(" select  ") + keyStyle.Render("←/→") + mutedStyle.Render(" change"))
	} else {
		body.WriteString(mutedStyle.Render("Navigate  ") + keyStyle.Render("↑/↓") + mutedStyle.Render(" or ") + keyStyle.Render("J/K") + mutedStyle.Render("   Change  ") + keyStyle.Render("←/→") + mutedStyle.Render(" or ") + keyStyle.Render("SPACE"))
	}
	return body.String()
}

func controlRow(leftKey, leftText, rightKey, rightText string) string {
	left := keyStyle.Render(leftKey) + " " + softStyle.Render(leftText)
	right := keyStyle.Render(rightKey) + " " + softStyle.Render(rightText)
	return left + strings.Repeat(" ", max(2, 36-lipgloss.Width(left))) + right
}

func settingRow(selected bool, label, value, description string) string {
	marker := mutedStyle.Render("  ")
	if selected {
		marker = brightStyle.Render("❯ ")
	}
	return marker + softStyle.Render(fmt.Sprintf("%-12s", label)) + brightStyle.Render(fmt.Sprintf("[ %-7s ]", value)) + mutedStyle.Render("  "+description)
}

func onOff(enabled bool) string {
	if enabled {
		return "ON"
	}
	return "OFF"
}
