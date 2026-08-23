package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/game"
)

type SaveFunc func(game.Profile) error

type Model struct {
	engine                      game.Engine
	challenges                  []challenge.Challenge
	profile                     game.Profile
	save                        SaveFunc
	input                       textinput.Model
	index, hints, width, height int
	feedback                    string
	done                        bool
}

func New(engine game.Engine, challenges []challenge.Challenge, profile game.Profile, save SaveFunc) Model {
	in := textinput.New()
	in.Prompt = "$ "
	in.Placeholder = "grep PATTERN FILE"
	in.CharLimit = 512
	in.SetWidth(70)
	in.Focus()
	return Model{engine: engine, challenges: challenges, profile: profile, save: save, input: in, width: 100, height: 30}
}

func (m Model) Init() tea.Cmd { return m.input.Focus() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+h":
			if !m.done && m.hints < len(m.current().Hints) {
				m.feedback = "HINT: " + m.current().Hints[m.hints]
				m.hints++
			}
			return m, nil
		case "enter":
			if m.done {
				return m, tea.Quit
			}
			command := strings.TrimSpace(m.input.Value())
			if command == "" {
				return m, nil
			}
			result, err := m.engine.Submit(context.Background(), &m.profile, m.current(), command, m.hints)
			if err != nil {
				m.feedback = "Cannot judge: " + err.Error()
			} else if !result.Correct {
				m.feedback = fmt.Sprintf("Not yet. exit %d (wanted %d)\nOutput:\n%s\nConcept: %s", result.ActualExit, result.ExpectedExit, result.ActualOutput, result.Explanation)
				if result.HiddenTotal > 0 {
					m.feedback += fmt.Sprintf("\nVisible example passed, but %d/%d hidden variants held. Check whether the command generalizes.", result.HiddenPassed, result.HiddenTotal)
				}
				if m.save != nil {
					if saveErr := m.save(m.profile); saveErr != nil {
						m.feedback += "\nSave failed: " + saveErr.Error()
					}
				}
			} else {
				m.feedback = fmt.Sprintf("CLEAR +%d XP — %s", result.XPGained, result.Explanation)
				if result.Drop != nil {
					name, sprite := result.Drop.Name, result.Drop.Sprite
					if result.Generated != nil {
						name = result.Generated.Name
						sprite = result.Generated.Sprite
					}
					m.feedback += fmt.Sprintf("\nDROP [%s] %s %s", result.Drop.Rarity, sprite, name)
					if result.Generated != nil {
						m.feedback += "\nLORE: " + result.Generated.Flavor + "\nINSIGHT: " + result.Generated.Explanation
					}
				}
				if m.save != nil {
					if err = m.save(m.profile); err != nil {
						m.feedback += "\nSave failed: " + err.Error()
					}
				}
				m.index++
				m.hints = 0
				m.input.Reset()
				if m.index >= len(m.challenges) {
					m.done = true
					m.feedback = "ACT COMPLETE — the Signal Cellar and Regex Foundry await your return. Press Enter."
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	if len(m.challenges) == 0 {
		return tea.NewView("No challenges installed.\n")
	}
	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#65D46E")).Bold(true)
	mission := ""
	terminal := ""
	if m.done {
		mission = "All tutorial rooms cleared."
		terminal = m.feedback
	} else {
		c := m.current()
		mission = accent.Render(fmt.Sprintf("ROOM %d/%d · %s", m.index+1, len(m.challenges), c.Title)) + "\n\n" + c.Prompt + "\n\nTags: " + strings.Join(c.Tags, ", ")
		terminal = accent.Render("TERMINAL") + "\n\n" + m.input.View() + "\n\n" + m.feedback
	}
	mastery := make([]string, 0, len(m.profile.Mastery))
	for k, v := range m.profile.Mastery {
		mastery = append(mastery, fmt.Sprintf("%-12s %3d", k, v))
	}
	sort.Strings(mastery)
	character := accent.Render(fmt.Sprintf("RANGER · LV %d", m.profile.Level)) + fmt.Sprintf("\n\n    /\\\n   /  \\\n  | () |\n   \\__/\n    /|\n\nXP %d\nClears %d\n\n%s\n\n%s", m.profile.XP, len(m.profile.Solved), strings.Join(mastery, "\n"), renderLoadout(m.profile))
	w := m.width
	if w < 60 {
		w = 60
	}
	left := w*2/3 - 2
	right := w - left - 3
	missionBox := panel.Width(left).Render(mission)
	charBox := panel.Width(right).Render(character)
	terminalBox := panel.Width(w - 4).Render(terminal)
	footer := "enter submit · ctrl+h hint · q quit"
	v := tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top, missionBox, charBox) + "\n" + terminalBox + "\n" + footer + "\n")
	v.AltScreen = true
	return v
}

func renderLoadout(p game.Profile) string {
	lines := []string{}
	for _, slot := range []game.Slot{game.Head, game.Tool, game.Body, game.Charm, game.Relic} {
		name := "—"
		if id := p.Loadout[slot]; id != "" {
			for _, item := range game.BuiltinItems {
				if item.ID == id {
					name = item.Sprite + " " + item.Name
					break
				}
			}
		}
		lines = append(lines, fmt.Sprintf("%-5s %s", slot, name))
	}
	return strings.Join(lines, "\n")
}
func (m Model) current() challenge.Challenge {
	if m.index >= len(m.challenges) {
		return m.challenges[len(m.challenges)-1]
	}
	return m.challenges[m.index]
}
