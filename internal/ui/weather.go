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
	units    string // "F" or "C"
	data     *api.WeatherData
	forecast *api.ForecastDay
	loading  bool
	err      string
	lastSync time.Time
	width    int
	height   int
	focused  bool
}

func newWeatherPane(apiKey, city, units string) weatherPane {
	return weatherPane{apiKey: apiKey, city: city, units: units, loading: true}
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
		bodyH := p.height - 4 // border(2) + title(1) + sep(1)
		switch {
		case bodyH >= 18:
			body = p.renderCurrent() + "\n" + p.renderForecast()
		case bodyH >= 11:
			body = p.renderCurrent()
		default:
			body = p.renderCondensed(bodyH)
		}
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, title, sep, body)
	return paneStyle(colorWeather, p.focused, p.width, p.height).Render(inner)
}

// tempStr formats a temperature pair with the configured unit shown first.
func (p weatherPane) tempStr(f, c float64) string {
	if p.units == "C" {
		return fmt.Sprintf("%.0f°C / %.0f°F", c, f)
	}
	return fmt.Sprintf("%.0f°F / %.0f°C", f, c)
}

// feelsStr formats a feels-like temperature in the configured unit only.
func (p weatherPane) feelsStr(f, c float64) string {
	if p.units == "C" {
		return fmt.Sprintf("%.0f°C", c)
	}
	return fmt.Sprintf("%.0f°F", f)
}

func (p weatherPane) renderCurrent() string {
	d := p.data
	divider := dimStyle.Render("  " + strings.Repeat("─", p.width-6))
	syncLine := dimStyle.Render(fmt.Sprintf("  Updated: %s", p.lastSync.Format("3:04 PM")))

	feelsC := (d.FeelsF - 32) * 5 / 9
	lines := []string{
		fmt.Sprintf("  %s, %s", d.City, d.Country),
		"",
		lipgloss.NewStyle().Bold(true).Render("  " + p.tempStr(d.TempF, d.TempC)),
		fmt.Sprintf("  Feels like: %s", p.feelsStr(d.FeelsF, feelsC)),
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

// renderCondensed squeezes weather into fewer lines for short panes.
// bodyH is the number of lines available for body content.
func (p weatherPane) renderCondensed(bodyH int) string {
	d := p.data
	feelsC := (d.FeelsF - 32) * 5 / 9
	syncLine := dimStyle.Render(fmt.Sprintf("  Updated: %s", p.lastSync.Format("3:04 PM")))

	lines := []string{
		fmt.Sprintf("  %s, %s", d.City, d.Country),
		fmt.Sprintf("  %s  ·  Feels %s", p.tempStr(d.TempF, d.TempC), p.feelsStr(d.FeelsF, feelsC)),
		"  " + strings.Title(d.Desc),
		fmt.Sprintf("  Hum %d%%  ·  Wind %.0f mph %s", d.Humidity, d.WindMph, d.WindDir),
		syncLine,
	}

	if p.forecast != nil && bodyH >= 7 {
		f := p.forecast
		tomorrowLine := truncate(
			fmt.Sprintf("  Tomorrow: Hi %s  Lo %s  %s",
				p.feelsStr(f.TempMaxF, f.TempMaxC),
				p.feelsStr(f.TempMinF, f.TempMinC),
				strings.Title(f.Desc),
			), p.width-4)
		lines = append(lines, "", tomorrowLine)
		if f.PrecipPct > 0 && bodyH >= 8 {
			lines = append(lines, fmt.Sprintf("  Precip: %d%%", f.PrecipPct))
		}
	}

	return strings.Join(lines, "\n")
}

func (p weatherPane) renderForecast() string {
	accentStyle := lipgloss.NewStyle().Foreground(colorWeather).Bold(true)
	divider := dimStyle.Render("  " + strings.Repeat("─", p.width-6))
	header := accentStyle.Render("  TOMORROW")

	if p.forecast == nil {
		return "\n" + header + "\n" + divider + "\n" +
			dimStyle.Render("  No forecast data")
	}

	f := p.forecast
	lines := []string{
		"",
		header,
		divider,
		fmt.Sprintf("  High:   %s", p.tempStr(f.TempMaxF, f.TempMaxC)),
		fmt.Sprintf("  Low:    %s", p.tempStr(f.TempMinF, f.TempMinC)),
		"  " + strings.Title(f.Desc),
	}
	if f.PrecipPct > 0 {
		lines = append(lines, fmt.Sprintf("  Precip: %d%%", f.PrecipPct))
	}
	return strings.Join(lines, "\n")
}
