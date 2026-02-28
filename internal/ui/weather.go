package ui

import (
	"fmt"
	"strings"
	"time"

	"today-tui/internal/api"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fetchWeatherMsg struct{}
type gotWeatherMsg struct {
	data *api.WeatherData
	err  error
}

type weatherPane struct {
	apiKey  string
	city    string
	data    *api.WeatherData
	loading bool
	err     string
	lastSync time.Time
	width   int
	height  int
	focused bool
}

func newWeatherPane(apiKey, city string) weatherPane {
	return weatherPane{apiKey: apiKey, city: city, loading: true}
}

func (p weatherPane) Init() tea.Cmd {
	return func() tea.Msg { return fetchWeatherMsg{} }
}

func doFetchWeather(apiKey, city string) tea.Cmd {
	return func() tea.Msg {
		data, err := api.FetchWeather(apiKey, city)
		return gotWeatherMsg{data: data, err: err}
	}
}

func (p weatherPane) Update(msg tea.Msg) (weatherPane, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchWeatherMsg:
		p.loading = true
		return p, doFetchWeather(p.apiKey, p.city)
	case gotWeatherMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
		} else {
			p.err = ""
			p.data = msg.data
			p.lastSync = time.Now()
		}
	case tea.KeyMsg:
		if p.focused && msg.String() == "r" {
			p.loading = true
			return p, doFetchWeather(p.apiKey, p.city)
		}
	}
	return p, nil
}

func (p *weatherPane) SetSize(w, h int) { p.width = w; p.height = h }
func (p *weatherPane) SetFocused(f bool) { p.focused = f }

func (p weatherPane) View() string {
	accentStyle := lipgloss.NewStyle().Foreground(colorWeather).Bold(true)
	title := accentStyle.Render("WEATHER")
	sep := dimStyle.Render(strings.Repeat("─", p.width-4))

	var body string
	switch {
	case p.loading:
		body = dimStyle.Render("  Fetching weather...")
	case p.err != "":
		body = errStyle.Render("  Error: "+truncate(p.err, p.width-6)) +
			"\n\n" + dimStyle.Render("  Run with --reconfigure to re-enter\n  your API key.")
	default:
		d := p.data
		tempLine := fmt.Sprintf("  %.0f°F / %.0f°C", d.TempF, d.TempC)
		feelsLine := fmt.Sprintf("  Feels like: %.0f°F", d.FeelsF)
		descLine := "  " + strings.Title(d.Desc)
		humLine := fmt.Sprintf("  Humidity:  %d%%", d.Humidity)
		windLine := fmt.Sprintf("  Wind:      %.0f mph %s", d.WindMph, d.WindDir)
		locLine := fmt.Sprintf("  %s, %s", d.City, d.Country)
		syncLine := dimStyle.Render(fmt.Sprintf("  Updated: %s", p.lastSync.Format("3:04 PM")))

		lines := []string{
			locLine,
			"",
			lipgloss.NewStyle().Bold(true).Render(tempLine),
			feelsLine,
			dimStyle.Render("  " + strings.Repeat("─", p.width-6)),
			descLine,
			"",
			humLine,
			windLine,
			"",
			syncLine,
		}
		body = strings.Join(lines, "\n")
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, title, sep, body)
	return paneStyle(colorWeather, p.focused, p.width, p.height).Render(inner)
}
