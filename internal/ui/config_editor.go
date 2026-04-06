package ui

import (
	"github.com/jonmilley/today-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	panelTodo    = "todo"
	panelWeather = "weather"
	panelStocks  = "stocks"
	panelStats   = "stats"
	panelNews    = "news"

	keyUp    = "up"
	keyDown  = "down"
	keyEnter = "enter"
	keyEsc   = "esc"
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
	{"Weather", panelWeather},
	{"Stocks", panelStocks},
	{"Stats", panelStats},
	{"News", panelNews},
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
	switch panelToggles[idx].key {
	case panelTodo:
		return e.panels.Todo
	case panelWeather:
		return e.panels.Weather
	case panelStocks:
		return e.panels.Stocks
	case panelStats:
		return e.panels.Stats
	case panelNews:
		return e.panels.News
	}
	return false
}

func (e *configEditor) toggle(idx int) {
	switch panelToggles[idx].key {
	case panelTodo:
		e.panels.Todo = !e.panels.Todo
	case panelWeather:
		e.panels.Weather = !e.panels.Weather
	case panelStocks:
		e.panels.Stocks = !e.panels.Stocks
	case panelStats:
		e.panels.Stats = !e.panels.Stats
	case panelNews:
		e.panels.News = !e.panels.News
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
