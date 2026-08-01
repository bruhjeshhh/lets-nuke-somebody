// Package tui contains all terminal rendering and input handling for the
// game. It deliberately knows nothing about game rules - it only calls
// into the game package's public API (game.Execute, game.State) and
// displays whatever comes back. This keeps simulation logic testable and
// swappable without touching presentation code.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"territoria/internal/game"
)

const maxLogLines = 200

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("24")).
			Padding(0, 1)

	promptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))
)

// Model is the Bubble Tea model driving the CLI. It wraps a game.State
// plus whatever is needed purely for display: scrollback log and the
// text input widget.
type Model struct {
	state    *game.State
	input    textinput.Model
	log      []string
	quitting bool
}

// New builds the initial TUI model around an already-loaded game state.
func New(s *game.State) Model {
	ti := textinput.New()
	ti.Placeholder = "type a command (try 'help')"
	ti.Focus()
	ti.CharLimit = 128
	ti.Width = 60

	m := Model{
		state: s,
		input: ti,
		log: []string{
			"Territoria - World Setup",
			"Type 'countries' to see the world, then 'select <country>' to begin.",
			"Each 'end' grows your population, gold, oil, and steel - try 'leaderboard' to see how you stack up.",
			"Use 'resources' to check your stockpile, and 'build <unit> <amount>' to buy units.",
			"",
		},
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			line := m.input.Value()
			m.input.Reset()
			if strings.TrimSpace(line) == "" {
				return m, nil
			}

			m.appendLine(promptStyle.Render("> ") + line)

			result := game.Execute(m.state, line)
			for _, l := range result.Lines {
				m.appendLine(l)
			}
			m.appendLine("")

			if result.Quit {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) appendLine(line string) {
	m.log = append(m.log, line)
	if len(m.log) > maxLogLines {
		m.log = m.log[len(m.log)-maxLogLines:]
	}
}

func (m Model) View() string {
	if m.quitting {
		return "Thanks for playing Territoria.\n"
	}

	var b strings.Builder

	owner := "none selected"
	if m.state.PlayerHasSelected() {
		owner = m.state.PlayerCountry
	}
	header := fmt.Sprintf("TERRITORIA  |  Turn %d  |  Playing as: %s", m.state.Turn, owner)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	b.WriteString(strings.Join(m.log, "\n"))
	b.WriteString("\n\n")

	b.WriteString(promptStyle.Render("> ") + m.input.View())
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("(esc or ctrl+c to quit)"))

	return b.String()
}

// errorLine is a small helper kept here so future commands can reuse
// consistent error styling without importing lipgloss into game logic.
func errorLine(s string) string {
	return errorStyle.Render(s)
}
