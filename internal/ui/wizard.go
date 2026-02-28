package ui

import (
	"fmt"
	"strings"

	"today-tui/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SetupDoneMsg struct{ Cfg *config.Config }

type wizardStep int

const (
	stepGitHubRepo wizardStep = iota
	stepGitHubToken
	stepWeatherKey
	stepWeatherCity
	stepRSSURL
	stepDone
)

type wizardModel struct {
	step    wizardStep
	inputs  []textinput.Model
	err     string
	width   int
	height  int
}

var wizardPrompts = []struct {
	title    string
	hint     string
	placeholder string
	password bool
}{
	{"GitHub Repository", "Format: owner/repo (e.g. acme/tasks)", "owner/repo", false},
	{"GitHub Token", "Personal access token with 'repo' scope", "ghp_...", true},
	{"OpenWeatherMap API Key", "Free at openweathermap.org/api", "your-api-key", true},
	{"Weather City", "City name for weather (e.g. London, New York)", "City Name", false},
	{"RSS Feed URL (optional)", "Full RSS/Atom URL — press Enter to skip", "https://example.com/rss", false},
}

func newWizard() wizardModel {
	inputs := make([]textinput.Model, len(wizardPrompts))
	for i, p := range wizardPrompts {
		ti := textinput.New()
		ti.Placeholder = p.placeholder
		ti.CharLimit = 256
		ti.Width = 50
		if p.password {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		inputs[i] = ti
	}
	inputs[0].Focus()
	return wizardModel{inputs: inputs}
}

func (m wizardModel) Init() tea.Cmd { return textinput.Blink }

func (m wizardModel) Update(msg tea.Msg) (wizardModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.step < stepDone {
				m.inputs[m.step].Blur()
				m.step++
				if int(m.step) < len(m.inputs) {
					m.inputs[m.step].Focus()
					return m, textinput.Blink
				}
				// All done
				cfg := &config.Config{
					GitHubRepo:    m.inputs[0].Value(),
					GitHubToken:   m.inputs[1].Value(),
					WeatherAPIKey: m.inputs[2].Value(),
					WeatherCity:   m.inputs[3].Value(),
					RSSFeedURL:    m.inputs[4].Value(),
					Stocks:        config.DefaultStocks(),
				}
				if err := cfg.Save(); err != nil {
					m.err = err.Error()
					return m, nil
				}
				return m, func() tea.Msg { return SetupDoneMsg{Cfg: cfg} }
			}
		case tea.KeyEsc:
			if m.step > 0 {
				m.inputs[m.step].Blur()
				m.step--
				m.inputs[m.step].Focus()
				return m, textinput.Blink
			}
		}
	}
	if int(m.step) < len(m.inputs) {
		m.inputs[m.step], cmd = m.inputs[m.step].Update(msg)
	}
	return m, cmd
}

func (m wizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	step := int(m.step)
	if step >= len(wizardPrompts) {
		return "Saving configuration..."
	}

	p := wizardPrompts[step]

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69")).
		MarginBottom(1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(60)

	progress := fmt.Sprintf("Step %d / %d", step+1, len(wizardPrompts))

	lines := []string{
		headerStyle.Render("today-tui — First Launch Setup"),
		dimStyle.Render(progress),
		"",
		boldStyle.Render(p.title),
		dimStyle.Render(p.hint),
		"",
		m.inputs[step].View(),
	}

	if m.err != "" {
		lines = append(lines, "", errStyle.Render("Error: "+m.err))
	}

	footer := dimStyle.Render("Enter: confirm  •  Esc: back  •  Ctrl+C: quit")

	content := boxStyle.Render(strings.Join(lines, "\n"))
	full := lipgloss.JoinVertical(lipgloss.Left,
		content,
		"",
		footer,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, full)
}
