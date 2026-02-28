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
	data     *api.WeatherData
	forecast *api.ForecastDay
	err      error
}

type weatherPane struct {
	apiKey   string
	city     string
	data     *api.WeatherData
	forecast *api.ForecastDay
	loading  bool
	err      string
	lastSync time.Time
	width    int
	height   int
	focused  bool
}

func newWeatherPane(apiKey, city string) weatherPane {
	return weatherPane{apiKey: apiKey, city: city, loading: true}
}

func (p weatherPane) Init() tea.Cmd {
	return func() tea.Msg { return fetchWeatherMsg{} }
}

func doFetchWeather(apiKey, city string) tea.Cmd {
	return func() tea.Msg {
		// Fetch current conditions and tomorrow's forecast in parallel.
		type weatherResult struct {
			data *api.WeatherData
			err  error
		}
		type forecastResult struct {
			data *api.ForecastDay
			err  error
		}

		wCh := make(chan weatherResult, 1)
		fCh := make(chan forecastResult, 1)

		go func() {
			d, err := api.FetchWeather(apiKey, city)
			wCh <- weatherResult{d, err}
		}()
		go func() {
			d, err := api.FetchForecast(apiKey, city)
			fCh <- forecastResult{d, err}
		}()

		wr := <-wCh
		fr := <-fCh
		// Forecast failure is non-fatal — show current weather without it.
		return gotWeatherMsg{data: wr.data, forecast: fr.data, err: wr.err}
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
			p.forecast = msg.forecast
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
		body = p.renderCurrent() + "\n" + p.renderForecast()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, title, sep, body)
	return paneStyle(colorWeather, p.focused, p.width, p.height).Render(inner)
}

func (p weatherPane) renderCurrent() string {
	d := p.data
	divider := dimStyle.Render("  " + strings.Repeat("─", p.width-6))
	syncLine := dimStyle.Render(fmt.Sprintf("  Updated: %s", p.lastSync.Format("3:04 PM")))

	lines := []string{
		fmt.Sprintf("  %s, %s", d.City, d.Country),
		"",
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("  %.0f°F / %.0f°C", d.TempF, d.TempC)),
		fmt.Sprintf("  Feels like: %.0f°F", d.FeelsF),
		divider,
		"  " + strings.Title(d.Desc),
		"",
		fmt.Sprintf("  Humidity:  %d%%", d.Humidity),
		fmt.Sprintf("  Wind:      %.0f mph %s", d.WindMph, d.WindDir),
		"",
		syncLine,
	}
	return strings.Join(lines, "\n")
}

func (p weatherPane) renderForecast() string {
	accentStyle := lipgloss.NewStyle().Foreground(colorWeather).Bold(true)
	divider := dimStyle.Render("  " + strings.Repeat("─", p.width-6))

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		accentStyle.Render("  TOMORROW"),
	)

	if p.forecast == nil {
		return "\n" + header + "\n" + divider + "\n" +
			dimStyle.Render("  No forecast data")
	}

	f := p.forecast
	lines := []string{
		"",
		header,
		divider,
		fmt.Sprintf("  High:   %.0f°F / %.0f°C", f.TempMaxF, f.TempMaxC),
		fmt.Sprintf("  Low:    %.0f°F / %.0f°C", f.TempMinF, f.TempMinC),
		"  " + strings.Title(f.Desc),
	}
	if f.PrecipPct > 0 {
		lines = append(lines, fmt.Sprintf("  Precip: %d%%", f.PrecipPct))
	}
	return strings.Join(lines, "\n")
}
