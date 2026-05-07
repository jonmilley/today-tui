package ui

import (
	"github.com/jonmilley/today-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type configClosedMsg struct {
	panels config.PanelConfig
}

type panelToggle struct {
	label string
	key   string
}

var panelToggles = []panelToggle{
	{"Todo", panelTodo},
	{"Calendar", panelCalendar},
	{"Weather", panelWeather},
	{"Stocks", panelStocks},
	{"Stats", panelStats},
	{"News", panelNews},
}

// panelByIndex returns a pointer to the bool field in p that backs the
// panelToggles entry at idx, or nil for an unknown index. Used by both the
// runtime config editor and the setup wizard to toggle/inspect panels.
func panelByIndex(p *config.PanelConfig, idx int) *bool {
	if idx < 0 || idx >= len(panelToggles) {
		return nil
	}
	switch panelToggles[idx].key {
	case panelTodo:
		return &p.Todo
	case panelCalendar:
		return &p.Calendar
	case panelWeather:
		return &p.Weather
	case panelStocks:
		return &p.Stocks
	case panelStats:
		return &p.Stats
	case panelNews:
		return &p.News
	}
	return nil
}

type configEditor struct {
	panels config.PanelConfig
	cursor int
	width  int
	height int
}

func newConfigEditor(panels config.PanelConfig) configEditor {
	return configEditor{panels: panels}
}

func (e configEditor) isEnabled(idx int) bool {
	if p := panelByIndex(&e.panels, idx); p != nil {
		return *p
	}
	return false
}

func (e *configEditor) toggle(idx int) {
	if p := panelByIndex(&e.panels, idx); p != nil {
		*p = !*p
	}
}

func (e configEditor) Update(msg tea.Msg) (configEditor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", keyDown:
			if e.cursor < len(panelToggles)-1 {
				e.cursor++
			}
		case "k", keyUp:
			if e.cursor > 0 {
				e.cursor--
			}
		case " ", keyEnter:
			e.toggle(e.cursor)
		case keyEsc, "q", ",":
			return e, func() tea.Msg { return configClosedMsg{panels: e.panels} }
		}
	}
	return e, nil
}

func (e configEditor) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69"))

	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("69")).
		Bold(true)

	checkedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	uncheckedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	labelActiveStyle := lipgloss.NewStyle().Bold(true)

	rows := []string{
		titleStyle.Render("Panel Visibility"),
		"",
	}

	for i, toggle := range panelToggles {
		var cursor, checkbox, label string
		if i == e.cursor {
			cursor = cursorStyle.Render("> ")
		} else {
			cursor = "  "
		}
		if e.isEnabled(i) {
			checkbox = checkedStyle.Render("[x]")
		} else {
			checkbox = uncheckedStyle.Render("[ ]")
		}
		if i == e.cursor {
			label = labelActiveStyle.Render(toggle.label)
		} else {
			label = toggle.label
		}
		rows = append(rows, cursor+checkbox+" "+label)
	}

	rows = append(rows, "")
	rows = append(rows, dimStyle.Render("j/k: navigate   space/enter: toggle   esc: save & close"))

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 3).
		Render(content)

	return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, box)
}
