package ui

import (
	"fmt"
	"strings"

	"github.com/jonmilley/today-tui/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SetupDoneMsg struct{ Cfg *config.Config }

func normalizeUnits(s string) string {
	if strings.ToUpper(strings.TrimSpace(s)) == "C" {
		return "C"
	}
	return "F"
}

type wizardStep int

const (
	stepTodoBackend wizardStep = iota // 0 — new first step
	stepGitHubRepo                    // 1
	stepGitHubToken                   // 2
	stepWeatherKey                    // 3
	stepWeatherCity                   // 4
	stepUnits                         // 5
	stepRSSURL                        // 6
	stepPanels                        // 7
	stepDone                          // 8
)

type wizardModel struct {
	step        wizardStep
	todoBackend string // "github" or "local"
	inputs      []textinput.Model
	err         string
	width       int
	height      int
	panels      config.PanelConfig
	panelCursor int
}

// textIdx returns the inputs[] index for the current text-input step.
// stepGitHubRepo(1) → 0, stepGitHubToken(2) → 1, …, stepRSSURL(6) → 5.
func (m wizardModel) textIdx() int { return int(m.step) - 1 }

var wizardPrompts = []struct {
	title       string
	hint        string
	placeholder string
	password    bool
}{
	{"GitHub Repository", "Format: owner/repo (e.g. acme/tasks)", "owner/repo", false},
	{"GitHub Token", "Personal access token with 'repo' scope", "ghp_...", true},
	{"OpenWeatherMap API Key", "Free at openweathermap.org/api", "your-api-key", true},
	{"Weather City", "City name for weather (e.g. London, New York)", "City Name", false},
	{"Temperature Units", "Enter F for Fahrenheit or C for Celsius", "F", false},
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
	return wizardModel{
		step:        stepTodoBackend,
		todoBackend: todoBackendGitHub,
		inputs:      inputs,
		panels:      config.PanelConfig{Todo: true, Weather: true, Stocks: true, Stats: true, News: true},
	}
}

// newWizardFrom creates a wizard pre-populated from an existing config.
// Used by --reconfigure so the user can edit rather than re-enter everything.
func newWizardFrom(cfg *config.Config) wizardModel {
	m := newWizard()
	if cfg == nil {
		return m
	}
	if cfg.TodoBackend == todoBackendLocal {
		m.todoBackend = todoBackendLocal
	} else {
		m.todoBackend = todoBackendGitHub
	}
	m.inputs[0].SetValue(cfg.GitHubRepo)
	m.inputs[1].SetValue(cfg.GitHubToken)
	m.inputs[2].SetValue(cfg.WeatherAPIKey)
	m.inputs[3].SetValue(cfg.WeatherCity)
	m.inputs[4].SetValue(cfg.Units)
	m.inputs[5].SetValue(cfg.RSSFeedURL)
	m.panels = cfg.Panels
	return m
}

func (m wizardModel) isPanelEnabled(idx int) bool {
	switch panelToggles[idx].key {
	case "todo":
		return m.panels.Todo
	case "weather":
		return m.panels.Weather
	case "stocks":
		return m.panels.Stocks
	case "stats":
		return m.panels.Stats
	case "news":
		return m.panels.News
	}
	return false
}

func (m *wizardModel) togglePanel(idx int) {
	switch panelToggles[idx].key {
	case "todo":
		m.panels.Todo = !m.panels.Todo
	case "weather":
		m.panels.Weather = !m.panels.Weather
	case "stocks":
		m.panels.Stocks = !m.panels.Stocks
	case "stats":
		m.panels.Stats = !m.panels.Stats
	case "news":
		m.panels.News = !m.panels.News
	}
}

func (m wizardModel) buildConfig() *config.Config {
	return &config.Config{
		TodoBackend:   m.todoBackend,
		GitHubRepo:    m.inputs[0].Value(),
		GitHubToken:   m.inputs[1].Value(),
		WeatherAPIKey: m.inputs[2].Value(),
		WeatherCity:   m.inputs[3].Value(),
		Units:         normalizeUnits(m.inputs[4].Value()),
		RSSFeedURL:    m.inputs[5].Value(),
		Stocks:        config.DefaultStocks(),
		Panels:        m.panels,
	}
}

func (m wizardModel) Init() tea.Cmd { return textinput.Blink }

func (m wizardModel) validate() string {
	val := strings.TrimSpace(m.inputs[m.textIdx()].Value())
	switch m.step {
	case stepGitHubRepo:
		if val == "" {
			return "Repository is required"
		}
		parts := strings.Split(val, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "Format must be owner/repo"
		}
	case stepGitHubToken:
		if val == "" {
			return "Token is required"
		}
	case stepWeatherKey:
		if val == "" {
			return "API key is required"
		}
	case stepWeatherCity:
		if val == "" {
			return "City is required"
		}
	case stepUnits:
		u := strings.ToUpper(strings.TrimSpace(val))
		if u != "F" && u != "C" {
			return "Enter 'F' or 'C'"
		}
	}
	return ""
}

func (m wizardModel) Update(msg tea.Msg) (wizardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	var cmd tea.Cmd
	if m.step > stepTodoBackend && m.textIdx() < len(m.inputs) {
		m.inputs[m.textIdx()], cmd = m.inputs[m.textIdx()].Update(msg)
	}
	return m, cmd
}

func (m wizardModel) handleKeyMsg(msg tea.KeyMsg) (wizardModel, tea.Cmd) {
	if m.step == stepTodoBackend {
		return m.handleBackendStepKey(msg)
	}
	if m.step == stepPanels {
		return m.handlePanelStepKey(msg)
	}
	return m.handleTextInputKey(msg)
}

func (m wizardModel) handleBackendStepKey(msg tea.KeyMsg) (wizardModel, tea.Cmd) {
	switch msg.String() {
	case "j", keyDown:
		m.todoBackend = todoBackendLocal
	case "k", keyUp:
		m.todoBackend = todoBackendGitHub
	case "enter":
		m.err = ""
		if m.todoBackend == todoBackendLocal {
			m.step = stepWeatherKey
		} else {
			m.step = stepGitHubRepo
		}
		m.inputs[m.textIdx()].Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m wizardModel) handlePanelStepKey(msg tea.KeyMsg) (wizardModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		cfg := m.buildConfig()
		if err := cfg.Save(); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.step = stepDone
		return m, func() tea.Msg { return SetupDoneMsg{Cfg: cfg} }
	case tea.KeyEsc:
		m.step = stepRSSURL
		m.inputs[m.textIdx()].Focus()
		return m, textinput.Blink
	}

	switch msg.String() {
	case "j", keyDown:
		if m.panelCursor < len(panelToggles)-1 {
			m.panelCursor++
		}
	case "k", keyUp:
		if m.panelCursor > 0 {
			m.panelCursor--
		}
	case " ":
		m.togglePanel(m.panelCursor)
	}
	return m, nil
}

func (m wizardModel) handleTextInputKey(msg tea.KeyMsg) (wizardModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.Type {
	case tea.KeyEnter:
		if m.step < stepPanels {
			if err := m.validate(); err != "" {
				m.err = err
				return m, nil
			}
			m.err = ""
			m.inputs[m.textIdx()].Blur()
			m.step++
			if m.step == stepPanels {
				return m, nil
			}
			m.inputs[m.textIdx()].Focus()
			return m, textinput.Blink
		}
	case tea.KeyEsc:
		m.err = ""
		m.inputs[m.textIdx()].Blur()
		m.step--
		// If local backend, skip GitHub steps when going backward
		if m.todoBackend == todoBackendLocal && (m.step == stepGitHubRepo || m.step == stepGitHubToken) {
			m.step = stepTodoBackend
			return m, nil
		}
		if m.step == stepTodoBackend {
			return m, nil
		}
		m.inputs[m.textIdx()].Focus()
		return m, textinput.Blink
	default:
		if m.textIdx() < len(m.inputs) {
			m.inputs[m.textIdx()], cmd = m.inputs[m.textIdx()].Update(msg)
		}
	}
	return m, cmd
}

func (m wizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	if m.step == stepDone {
		return "Saving configuration..."
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69")).
		MarginBottom(1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(60)

	// --- Backend selection step ---
	if m.step == stepTodoBackend {
		progress := fmt.Sprintf("Step 1 / %d", int(stepDone))

		githubLine := "    github  — GitHub Issues (requires token)"
		localLine := "    local   — Local file (~/.config/today-tui/todos.json)"
		boldStyle := lipgloss.NewStyle().Bold(true)
		if m.todoBackend == todoBackendGitHub {
			githubLine = "  ▶ " + boldStyle.Render(todoBackendGitHub) + "  — GitHub Issues (requires token)"
		} else {
			localLine = "  ▶ " + boldStyle.Render(todoBackendLocal) +
				"   — Local file (~/.config/today-tui/todos.json)"
		}

		lines := []string{
			headerStyle.Render("today-tui — First Launch Setup"),
			dimStyle.Render(progress),
			"",
			boldStyle.Render("Todo Backend"),
			dimStyle.Render("Where should todos be stored?"),
			"",
			githubLine,
			localLine,
		}
		if m.err != "" {
			lines = append(lines, "", errStyle.Render("Error: "+m.err))
		}
		footer := dimStyle.Render("j/k: select  •  Enter: confirm  •  Ctrl+C: quit")
		content := boxStyle.Render(strings.Join(lines, "\n"))
		full := lipgloss.JoinVertical(lipgloss.Left, content, "", footer)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, full)
	}

	// --- Panels step ---
	if m.step == stepPanels {
		progress := fmt.Sprintf("Step %d / %d", int(stepPanels)+1, int(stepDone))

		cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
		checkedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		uncheckedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		lines := []string{
			headerStyle.Render("today-tui — First Launch Setup"),
			dimStyle.Render(progress),
			"",
			boldStyle.Render("Enable Panels"),
			dimStyle.Render("Choose which panels to show on the dashboard"),
			"",
		}
		for i, toggle := range panelToggles {
			var cursor, checkbox, label string
			if i == m.panelCursor {
				cursor = cursorStyle.Render("> ")
			} else {
				cursor = "  "
			}
			if m.isPanelEnabled(i) {
				checkbox = checkedStyle.Render("[x]")
			} else {
				checkbox = uncheckedStyle.Render("[ ]")
			}
			if i == m.panelCursor {
				label = boldStyle.Render(toggle.label)
			} else {
				label = toggle.label
			}
			lines = append(lines, cursor+checkbox+" "+label)
		}
		if m.err != "" {
			lines = append(lines, "", errStyle.Render("Error: "+m.err))
		}

		footer := dimStyle.Render("Space: toggle  •  j/k: navigate  •  Enter: finish  •  Esc: back")
		content := boxStyle.Render(strings.Join(lines, "\n"))
		full := lipgloss.JoinVertical(lipgloss.Left, content, "", footer)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, full)
	}

	// --- Text input steps ---
	p := wizardPrompts[m.textIdx()]
	progress := fmt.Sprintf("Step %d / %d", int(m.step)+1, int(stepDone))

	lines := []string{
		headerStyle.Render("today-tui — First Launch Setup"),
		dimStyle.Render(progress),
		"",
		boldStyle.Render(p.title),
		dimStyle.Render(p.hint),
		"",
		m.inputs[m.textIdx()].View(),
	}
	if m.err != "" {
		lines = append(lines, "", errStyle.Render("Error: "+m.err))
	}

	footer := dimStyle.Render("Enter: confirm  •  Esc: back  •  Ctrl+C: quit")
	content := boxStyle.Render(strings.Join(lines, "\n"))
	full := lipgloss.JoinVertical(lipgloss.Left, content, "", footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, full)
}
