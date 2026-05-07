package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/jonmilley/today-tui/internal/api"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// calendarWindow is how far ahead we ask the calendar source for events.
const calendarWindow = 7 * 24 * time.Hour

type fetchCalendarMsg struct{}
type gotCalendarMsg struct {
	events []api.Event
	err    error
}

type calendarPane struct {
	cal      api.Calendar
	source   string // ICS URL or file path; empty = pane not configured
	events   []api.Event
	loading  bool
	err      string
	lastSync time.Time
	viewport viewport.Model
	width    int
	height   int
	focused  bool
}

func newCalendarPane(cal api.Calendar, source string) calendarPane {
	return calendarPane{cal: cal, source: source, loading: source != ""}
}

func (p calendarPane) Init() tea.Cmd {
	if p.source == "" {
		return nil
	}
	return func() tea.Msg { return fetchCalendarMsg{} }
}

func doFetchCalendar(cal api.Calendar) tea.Cmd {
	return func() tea.Msg {
		events, err := cal.FetchEvents(calendarWindow)
		return gotCalendarMsg{events: events, err: err}
	}
}

func (p calendarPane) Update(msg tea.Msg) (calendarPane, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case fetchCalendarMsg:
		if p.source == "" {
			break
		}
		p.loading = true
		cmds = append(cmds, doFetchCalendar(p.cal))
	case gotCalendarMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
		} else {
			p.err = ""
			p.events = msg.events
			p.lastSync = time.Now()
		}
		p.viewport.SetContent(p.renderContent())
	case tea.KeyMsg:
		if p.focused && p.source != "" && msg.String() == "r" {
			p.loading = true
			cmds = append(cmds, doFetchCalendar(p.cal))
		}
	}
	var vpCmd tea.Cmd
	p.viewport, vpCmd = p.viewport.Update(msg)
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}
	return p, tea.Batch(cmds...)
}

func (p *calendarPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.viewport.Width = w - 4
	p.viewport.Height = h - 4
	p.viewport.SetContent(p.renderContent())
}

func (p *calendarPane) SetFocused(f bool) { p.focused = f }

func (p calendarPane) renderContent() string {
	if p.source == "" {
		return dimStyle.Render(
			"  No calendar configured.\n  Set calendar_url in\n  ~/.config/today-tui/config.json",
		)
	}
	if p.loading {
		return dimStyle.Render("  Fetching events...")
	}
	if p.err != "" {
		return errStyle.Render("  Error: " + truncate(p.err, p.width-6))
	}
	if len(p.events) == 0 {
		return dimStyle.Render("  No upcoming events")
	}

	contentWidth := p.width - 6
	now := time.Now()
	var sb strings.Builder
	currentDay := ""

	for _, ev := range p.events {
		day := dayLabel(ev.Start, now)
		if day != currentDay {
			if currentDay != "" {
				sb.WriteString("\n")
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(colorCalendar).Bold(true).
				Render("  "+day) + "\n")
			currentDay = day
		}
		timeStr := formatEventTime(ev)
		title := truncate(ev.Summary, contentWidth-len(timeStr)-3)
		fmt.Fprintf(&sb, "  %s  %s\n", dimStyle.Render(timeStr), title)
	}
	return sb.String()
}

// dayLabel returns a human-friendly bucket name for an event's start time:
// "Today", "Tomorrow", or a short weekday + date.
func dayLabel(t, now time.Time) string {
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	d := t.Format("2006-01-02")
	switch d {
	case today:
		return "Today"
	case tomorrow:
		return "Tomorrow"
	default:
		return t.Format("Mon Jan 2")
	}
}

// formatEventTime returns "all day" or "HH:MM" for the event's start.
func formatEventTime(ev api.Event) string {
	if ev.AllDay {
		return "all day"
	}
	return ev.Start.Format("3:04 PM")
}

func (p calendarPane) View() string {
	accent := lipgloss.NewStyle().Foreground(colorCalendar).Bold(true)
	title := accent.Render("CALENDAR")
	count := ""
	if !p.loading && p.err == "" && p.source != "" {
		count = dimStyle.Render(fmt.Sprintf("  %d upcoming", len(p.events)))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, count)
	sep := dimStyle.Render(strings.Repeat("─", p.width-4))

	parts := []string{header, sep, p.viewport.View()}
	if p.focused && p.source != "" {
		parts = append(parts, dimStyle.Render("  r: refresh"))
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return paneStyle(colorCalendar, p.focused, p.width, p.height).Render(inner)
}
