package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/game"
	"github.com/timfewi/grepquest/internal/tui/rangerart"
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
	ranger                      rangerart.Model
	settingsOpen                bool
	helpPage                    int
	settingsIndex               int
	compactHUD                  bool
}

func New(engine game.Engine, challenges []challenge.Challenge, profile game.Profile, save SaveFunc) Model {
	in := textinput.New()
	in.Prompt = "$ "
	in.Placeholder = "grep PATTERN FILE"
	in.CharLimit = 512
	in.SetWidth(70)
	in.Focus()
	animated := os.Getenv("GREPQUEST_REDUCED_MOTION") != "1" && os.Getenv("TERM") != "dumb"
	return Model{
		engine: engine, challenges: challenges, profile: profile, save: save,
		input: in, width: 100, height: 30,
		ranger: rangerart.New(animated),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.ranger.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		if m.settingsOpen {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "?", "f1", "esc", "q":
				m.settingsOpen = false
				return m, m.input.Focus()
			case "tab":
				m.helpPage = (m.helpPage + 1) % helpPageCount
				return m, nil
			case "shift+tab":
				m.helpPage = (m.helpPage + helpPageCount - 1) % helpPageCount
				return m, nil
			case "1", "h":
				m.helpPage = quickHelpPage
				return m, nil
			case "2", "g":
				m.helpPage = grepGuidePage
				return m, nil
			case "3", "s":
				m.helpPage = sessionSettingsPage
				return m, nil
			case "up", "k":
				if m.helpPage != sessionSettingsPage {
					return m, nil
				}
				m.settingsIndex = (m.settingsIndex + settingsCount - 1) % settingsCount
				return m, nil
			case "down", "j":
				if m.helpPage != sessionSettingsPage {
					return m, nil
				}
				m.settingsIndex = (m.settingsIndex + 1) % settingsCount
				return m, nil
			case "left", "right", " ", "space", "enter":
				if m.helpPage != sessionSettingsPage {
					return m, nil
				}
				if m.settingsIndex == motionSetting {
					return m, m.ranger.SetAnimated(!m.ranger.Animated())
				}
				m.compactHUD = !m.compactHUD
				return m, nil
			}
			return m, nil
		}
		if msg.String() == "f1" || msg.String() == "?" && strings.TrimSpace(m.input.Value()) == "" {
			m.settingsOpen = true
			m.helpPage = quickHelpPage
			m.input.Blur()
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if strings.TrimSpace(m.input.Value()) == "" {
				return m, tea.Quit
			}
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
			var rangerCmd tea.Cmd
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
				rangerCmd = m.ranger.Celebrate()
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
			return m, rangerCmd
		}
	}
	var inputCmd, rangerCmd tea.Cmd
	m.ranger, rangerCmd = m.ranger.Update(msg)
	m.input, inputCmd = m.input.Update(msg)
	return m, tea.Batch(inputCmd, rangerCmd)
}

func (m Model) View() tea.View {
	return renderGameView(m)
}

func (m Model) current() challenge.Challenge {
	if m.index >= len(m.challenges) {
		return m.challenges[len(m.challenges)-1]
	}
	return m.challenges[m.index]
}
