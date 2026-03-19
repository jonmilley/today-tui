package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type splashTickMsg time.Time
type splashDoneMsg struct{}

type splashModel struct {
	width     int
	height    int
	remaining int
	now       time.Time
}

func newSplash() splashModel {
	return splashModel{
		remaining: 5,
		now:       time.Now(),
	}
}

func (s splashModel) Init() tea.Cmd {
	return splashTick()
}

func splashTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return splashTickMsg(t)
	})
}

func (s splashModel) Update(msg tea.Msg) (splashModel, tea.Cmd) {
	switch msg := msg.(type) {
	case splashTickMsg:
		s.now = time.Time(msg)
		s.remaining--
		if s.remaining <= 0 {
			return s, func() tea.Msg { return splashDoneMsg{} }
		}
		return s, splashTick()
	}
	return s, nil
}

func (s splashModel) View() string {
	now := s.now
	if now.IsZero() {
		now = time.Now()
	}

	greeting := timeGreeting(now)
	date := now.Format("Monday, January 2, 2006")
	clock := now.Format("3:04:05 PM")
	dismiss := fmt.Sprintf("Dismissing in %ds...", s.remaining)

	appNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69")).
		MarginBottom(1)

	greetStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255"))

	dateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	clockStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("51"))

	dismissStyle := lipgloss.NewStyle().
		Foreground(colorDim)

	content := lipgloss.JoinVertical(lipgloss.Center,
		appNameStyle.Render("today"),
		greetStyle.Render(greeting),
		dateStyle.Render(date),
		clockStyle.Render(clock),
		"",
		dismissStyle.Render(dismiss),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(2, 6).
		Render(content)

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, box)
}

func timeGreeting(t time.Time) string {
	switch h := t.Hour(); {
	case h >= 5 && h < 12:
		return "Good morning."
	case h >= 12 && h < 17:
		return "Good afternoon."
	case h >= 17 && h < 21:
		return "Good evening."
	default:
		return "Good night."
	}
}
