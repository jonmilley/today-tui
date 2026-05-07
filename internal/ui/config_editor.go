package ui

import (
	"strings"

	"github.com/jonmilley/today-tui/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// configClosedMsg is sent when the runtime config editor is dismissed.
// The full Config is included so the App can persist it, refresh API
// clients, and reseed pane state.
type configClosedMsg struct {
	cfg config.Config
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

// panelByID returns a pointer to the bool in p that backs the panel with
// the given string id (panelTodo, panelCalendar, ...). Returns nil for
// unknown ids.
func panelByID(p *config.PanelConfig, id string) *bool {
	switch id {
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

// panelByIndex returns a pointer to the bool in p backing the panelToggles
// entry at idx, or nil for an unknown index. Used by the wizard which
// drives by index.
func panelByIndex(p *config.PanelConfig, idx int) *bool {
	if idx < 0 || idx >= len(panelToggles) {
		return nil
	}
	return panelByID(p, panelToggles[idx].key)
}

// fieldKind differentiates how a configuration field is rendered and edited.
type fieldKind int

const (
	fieldHeader   fieldKind = iota // section divider; not navigable
	fieldText                      // free-form text via textinput
	fieldPassword                  // text rendered as •••• in the editor
	fieldChoice                    // cycles through a fixed value list
	fieldList                      // comma-separated text → []string
	fieldBool                      // checkbox toggle (panel visibility)
)

type field struct {
	kind    fieldKind
	label   string
	key     string   // identifier for get/set dispatch
	choices []string // populated for fieldChoice
}

// Field keys identify editor fields in the get/set switches. They live in
// a distinct namespace from panel ids — the keyStocks list-editor field
// and the panelStocks panel-toggle field share a label but are otherwise
// unrelated.
const (
	keyTodoBackend = "todo_backend"
	keyGitHubRepo  = "github_repo"
	keyGitHubToken = "github_token"
	keyCalendarURL = "calendar_url"
	keyWeatherKey  = "weather_key"
	keyWeatherCity = "weather_city"
	keyUnits       = "units"
	keyStocks      = "stocks_list"
	keyRSSURL      = "rss_url"
)

// configFields is the ordered list of editable settings shown in the
// runtime config editor. Headers split the list into sections; the cursor
// skips them automatically. Panel toggle rows are derived from
// panelToggles so the labels live in exactly one place.
var configFields = buildConfigFields()

func buildConfigFields() []field {
	out := []field{
		{kind: fieldHeader, label: "Settings"},
		{kind: fieldChoice, label: "Todo backend", key: keyTodoBackend,
			choices: []string{todoBackendGitHub, todoBackendLocal}},
		{kind: fieldText, label: "GitHub repo", key: keyGitHubRepo},
		{kind: fieldPassword, label: "GitHub token", key: keyGitHubToken},
		{kind: fieldText, label: "Calendar URL", key: keyCalendarURL},
		{kind: fieldPassword, label: "Weather API key", key: keyWeatherKey},
		{kind: fieldText, label: "Weather city", key: keyWeatherCity},
		{kind: fieldChoice, label: "Units", key: keyUnits, choices: []string{"F", "C"}},
		{kind: fieldList, label: "Stocks", key: keyStocks},
		{kind: fieldText, label: "RSS feed URL", key: keyRSSURL},
		{kind: fieldHeader, label: "Panels"},
	}
	for _, t := range panelToggles {
		out = append(out, field{kind: fieldBool, label: t.label, key: t.key})
	}
	return out
}

// getStringField reads the string-valued setting backing key from c. List
// fields (Stocks) are returned as a comma-separated string.
func getStringField(c *config.Config, key string) string {
	switch key {
	case keyTodoBackend:
		return c.TodoBackend
	case keyGitHubRepo:
		return c.GitHubRepo
	case keyGitHubToken:
		return c.GitHubToken
	case keyCalendarURL:
		return c.CalendarURL
	case keyWeatherKey:
		return c.WeatherAPIKey
	case keyWeatherCity:
		return c.WeatherCity
	case keyUnits:
		return c.Units
	case keyStocks:
		return strings.Join(c.Stocks, ", ")
	case keyRSSURL:
		return c.RSSFeedURL
	}
	return ""
}

// setStringField writes value into the setting backing key on c, applying
// any field-specific normalization (units uppercased; stocks split on
// commas, trimmed, uppercased).
func setStringField(c *config.Config, key, value string) {
	switch key {
	case keyTodoBackend:
		c.TodoBackend = value
	case keyGitHubRepo:
		c.GitHubRepo = strings.TrimSpace(value)
	case keyGitHubToken:
		c.GitHubToken = strings.TrimSpace(value)
	case keyCalendarURL:
		c.CalendarURL = strings.TrimSpace(value)
	case keyWeatherKey:
		c.WeatherAPIKey = strings.TrimSpace(value)
	case keyWeatherCity:
		c.WeatherCity = strings.TrimSpace(value)
	case keyUnits:
		c.Units = normalizeUnits(value)
	case keyStocks:
		c.Stocks = parseStocksList(value)
	case keyRSSURL:
		c.RSSFeedURL = strings.TrimSpace(value)
	}
}

// parseStocksList splits a comma-separated symbol list, trimming whitespace
// and uppercasing each entry. Empty entries are dropped.
func parseStocksList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToUpper(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// nextChoice returns the choice that follows cur in choices, wrapping at
// the end. If cur isn't in the list, returns the first choice.
func nextChoice(choices []string, cur string) string {
	for i, c := range choices {
		if c == cur {
			return choices[(i+1)%len(choices)]
		}
	}
	if len(choices) > 0 {
		return choices[0]
	}
	return cur
}

type configEditor struct {
	cfg     config.Config
	cursor  int
	editing bool
	input   textinput.Model
	err     string
	width   int
	height  int
}

func newConfigEditor(cfg config.Config) configEditor {
	e := configEditor{cfg: cfg}
	// Land the cursor on the first navigable (non-header) row.
	for i, f := range configFields {
		if f.kind != fieldHeader {
			e.cursor = i
			break
		}
	}
	return e
}

// nextNavIndex finds the next non-header field in the given direction,
// wrapping. Returns the current cursor when only headers exist.
func (e configEditor) nextNavIndex(direction int) int {
	n := len(configFields)
	idx := e.cursor
	for step := 0; step < n; step++ {
		idx = (idx + direction + n) % n
		if configFields[idx].kind != fieldHeader {
			return idx
		}
	}
	return e.cursor
}

func (e configEditor) Update(msg tea.Msg) (configEditor, tea.Cmd) {
	if e.editing {
		return e.handleEditingKey(msg)
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return e, nil
	}
	switch km.String() {
	case "j", keyDown:
		e.cursor = e.nextNavIndex(1)
	case "k", keyUp:
		e.cursor = e.nextNavIndex(-1)
	case keyEnter, " ":
		return e.activate()
	case keyEsc, ",":
		return e, func() tea.Msg { return configClosedMsg{cfg: e.cfg} }
	}
	return e, nil
}

// activate is invoked when the user presses Enter/Space on a field.
// Text-like fields enter inline edit mode; choice/bool fields toggle in
// place without a textinput.
func (e configEditor) activate() (configEditor, tea.Cmd) {
	f := configFields[e.cursor]
	switch f.kind {
	case fieldText, fieldPassword, fieldList:
		ti := textinput.New()
		if f.kind == fieldPassword {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		ti.SetValue(getStringField(&e.cfg, f.key))
		ti.CharLimit = 256
		ti.Width = 40
		ti.Focus()
		e.input = ti
		e.editing = true
		return e, textinput.Blink
	case fieldChoice:
		cur := getStringField(&e.cfg, f.key)
		setStringField(&e.cfg, f.key, nextChoice(f.choices, cur))
	case fieldBool:
		if p := panelByID(&e.cfg.Panels, f.key); p != nil {
			*p = !*p
		}
	}
	return e, nil
}

func (e configEditor) handleEditingKey(msg tea.Msg) (configEditor, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.Type {
		case tea.KeyEnter:
			f := configFields[e.cursor]
			setStringField(&e.cfg, f.key, e.input.Value())
			e.editing = false
			e.input.Blur()
			return e, nil
		case tea.KeyEsc:
			// Discard pending input; field keeps prior value.
			e.editing = false
			e.input.Blur()
			return e, nil
		}
	}
	var cmd tea.Cmd
	e.input, cmd = e.input.Update(msg)
	return e, cmd
}

// renderValue produces the display string for a field's current value:
// • fieldText/fieldList: the raw value
// • fieldPassword: bullet characters proportional to length, capped at 12
// • fieldChoice: "[value]"
// • fieldBool: "[x]" or "[ ]"
func (e configEditor) renderValue(f field) string {
	switch f.kind {
	case fieldText, fieldList:
		return getStringField(&e.cfg, f.key)
	case fieldPassword:
		v := getStringField(&e.cfg, f.key)
		n := len(v)
		if n > 12 {
			n = 12
		}
		return strings.Repeat("•", n)
	case fieldChoice:
		return "[" + getStringField(&e.cfg, f.key) + "]"
	case fieldBool:
		if p := panelByID(&e.cfg.Panels, f.key); p != nil && *p {
			return "[x]"
		}
		return "[ ]"
	}
	return ""
}

func (e configEditor) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	checkedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	uncheckedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	const labelW = 18

	rows := []string{titleStyle.Render("Configuration"), ""}

	for i, f := range configFields {
		if f.kind == fieldHeader {
			if len(rows) > 2 {
				rows = append(rows, "") // spacer above non-first headers
			}
			rows = append(rows, headerStyle.Render(f.label))
			continue
		}
		cursor := "  "
		if i == e.cursor {
			cursor = cursorStyle.Render("> ")
		}
		// Inline-editing mode replaces the value with a live textinput.
		if i == e.cursor && e.editing {
			rows = append(rows, cursor+padRight(f.label, labelW)+" "+e.input.View())
			continue
		}
		value := e.renderValue(f)
		switch f.kind {
		case fieldBool:
			if value == "[x]" {
				value = checkedStyle.Render(value)
			} else {
				value = uncheckedStyle.Render(value)
			}
		}
		row := cursor + padRight(f.label, labelW) + " " + value
		if i == e.cursor {
			row = lipgloss.NewStyle().Bold(true).Render(row)
		}
		rows = append(rows, row)
	}

	if e.err != "" {
		rows = append(rows, "", errStyle.Render(e.err))
	}

	var hint string
	if e.editing {
		hint = "Enter: save  Esc: cancel"
	} else {
		hint = "j/k: nav  Enter: edit/toggle  Esc: save & close"
	}
	rows = append(rows, "", dimStyle.Render(hint))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 3).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, box)
}
