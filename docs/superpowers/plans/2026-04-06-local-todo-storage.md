# Local Todo Storage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a local JSON file as an alternative Todo backend to GitHub Issues, selectable during the setup wizard.

**Architecture:** Rename the `api.GitHub` interface to `api.TodoBackend` and add a `LocalTodoClient` that implements it, storing todos at `~/.config/today-tui/todos.json`. A new first wizard step lets the user pick "github" or "local"; the config gains a `TodoBackend` field that drives which client is instantiated.

**Tech Stack:** Go, encoding/json, os/filepath (stdlib only — no new dependencies)

---

## File Map

| Action | File | What changes |
|--------|------|-------------|
| Modify | `internal/api/github.go` | Rename `GitHub` interface → `TodoBackend`; update compile-check |
| Create | `internal/api/local_todos.go` | New `LocalTodoClient` implementing `TodoBackend` |
| Modify | `internal/config/config.go` | Add `TodoBackend string` field |
| Modify | `internal/ui/app.go` | Rename `Deps.GitHub` → `Deps.Todo`; update `Deps.Refresh` |
| Modify | `internal/ui/todo.go` | Update `gh` field and function param types |
| Modify | `internal/ui/wizard.go` | Add `stepTodoBackend` step, skip GitHub steps for local, update `buildConfig`/`newWizardFrom`/`View` |
| Modify | `main.go` | Update `Deps` literal |

---

## Task 1: Rename `GitHub` interface to `TodoBackend`

**Files:**
- Modify: `internal/api/github.go`

- [ ] **Step 1: Replace the interface name and compile check**

In `internal/api/github.go`, make these exact changes:

```go
// line 23: was "type GitHub interface {"
type TodoBackend interface {
    GetOpenIssues() ([]Issue, error)
    CreateIssue(title string) (*Issue, error)
    CloseIssue(number int) error
}

// line 35: was "var _ GitHub = (*GitHubClient)(nil)"
var _ TodoBackend = (*GitHubClient)(nil)
```

- [ ] **Step 2: Verify build fails on other files (expected)**

Run: `go build ./internal/api/...`
Expected: PASS (only this package so far; consumers will break next)

- [ ] **Step 3: Commit**

```bash
git add internal/api/github.go
git commit -m "refactor: rename api.GitHub interface to TodoBackend"
```

---

## Task 2: Create `LocalTodoClient`

**Files:**
- Create: `internal/api/local_todos.go`

- [ ] **Step 1: Create the file**

```go
package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type localTodoStore struct {
	NextID int     `json:"next_id"`
	Todos  []Issue `json:"todos"`
}

// LocalTodoClient implements TodoBackend using a JSON file on disk.
type LocalTodoClient struct {
	path string
}

var _ TodoBackend = (*LocalTodoClient)(nil)

func NewLocalTodoClient() *LocalTodoClient {
	home, _ := os.UserHomeDir()
	return &LocalTodoClient{
		path: filepath.Join(home, ".config", "today-tui", "todos.json"),
	}
}

func (c *LocalTodoClient) load() (localTodoStore, error) {
	data, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return localTodoStore{NextID: 1}, nil
	}
	if err != nil {
		return localTodoStore{}, err
	}
	var store localTodoStore
	if err := json.Unmarshal(data, &store); err != nil {
		return localTodoStore{}, err
	}
	if store.NextID < 1 {
		store.NextID = 1
	}
	return store, nil
}

func (c *LocalTodoClient) save(store localTodoStore) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o600)
}

func (c *LocalTodoClient) GetOpenIssues() ([]Issue, error) {
	store, err := c.load()
	if err != nil {
		return nil, err
	}
	return store.Todos, nil
}

func (c *LocalTodoClient) CreateIssue(title string) (*Issue, error) {
	store, err := c.load()
	if err != nil {
		return nil, err
	}
	issue := Issue{
		Number:    store.NextID,
		Title:     title,
		State:     "open",
		CreatedAt: time.Now(),
	}
	store.Todos = append([]Issue{issue}, store.Todos...)
	store.NextID++
	if err := c.save(store); err != nil {
		return nil, err
	}
	return &issue, nil
}

func (c *LocalTodoClient) CloseIssue(number int) error {
	store, err := c.load()
	if err != nil {
		return err
	}
	for i, t := range store.Todos {
		if t.Number == number {
			store.Todos = append(store.Todos[:i], store.Todos[i+1:]...)
			break
		}
	}
	return c.save(store)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/api/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/api/local_todos.go
git commit -m "feat: add LocalTodoClient backed by JSON file"
```

---

## Task 3: Add `TodoBackend` field to config

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add the field to `Config`**

In `internal/config/config.go`, add `TodoBackend` to the `Config` struct:

```go
type Config struct {
	GitHubRepo    string      `json:"github_repo"`
	GitHubToken   string      `json:"github_token"`
	TodoBackend   string      `json:"todo_backend"` // "github" or "local"
	WeatherAPIKey string      `json:"weather_api_key"`
	WeatherCity   string      `json:"weather_city"`
	Units         string      `json:"units"`
	Stocks        []string    `json:"stocks"`
	RSSFeedURL    string      `json:"rss_feed_url"`
	Panels        PanelConfig `json:"panels"`
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/config/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add TodoBackend field to config"
```

---

## Task 4: Update `Deps` and wiring in `app.go` and `main.go`

**Files:**
- Modify: `internal/ui/app.go`
- Modify: `main.go`

- [ ] **Step 1: Rename `Deps.GitHub` to `Deps.Todo` in `app.go`**

In `internal/ui/app.go`, change the `Deps` struct and its usages:

```go
type Deps struct {
	Todo    api.TodoBackend  // was: GitHub api.GitHub
	Weather api.Weather
	Stocks  api.Stocks
	News    api.News
}
```

- [ ] **Step 2: Update `Deps.Refresh` to instantiate the right client**

Replace the body of `Deps.Refresh` in `internal/ui/app.go`:

```go
func (d *Deps) Refresh(cfg *config.Config) {
	if cfg.TodoBackend == "local" {
		d.Todo = api.NewLocalTodoClient()
	} else {
		d.Todo = api.NewGitHubClient(cfg.GitHubToken, cfg.GitHubRepo)
	}
	d.Weather = api.NewWeatherClient(cfg.WeatherAPIKey)
	if d.Stocks == nil {
		d.Stocks = api.NewYahooClient()
	}
	if d.News == nil {
		d.News = api.NewNewsClient()
	}
}
```

- [ ] **Step 3: Update `buildDash` call in `app.go`**

```go
func buildDash(cfg *config.Config, deps Deps) App {
	return App{
		mode:    modeDash,
		cfg:     cfg,
		deps:    deps,
		todo:    newTodoPane(deps.Todo),  // was: deps.GitHub
		weather: newWeatherPane(deps.Weather, cfg.WeatherCity, cfg.Units),
		stocks:  newStocksPane(deps.Stocks, cfg.Stocks),
		stats:   newStatsPane(),
		news:    newNewsPane(deps.News, cfg.RSSFeedURL),
		focused: paneTodo,
	}
}
```

- [ ] **Step 4: Update `main.go`**

Replace the deps initialization block in `main.go`:

```go
var deps ui.Deps
if cfg != nil {
	var todo api.TodoBackend
	if cfg.TodoBackend == "local" {
		todo = api.NewLocalTodoClient()
	} else {
		todo = api.NewGitHubClient(cfg.GitHubToken, cfg.GitHubRepo)
	}
	deps = ui.Deps{
		Todo:    todo,
		Weather: api.NewWeatherClient(cfg.WeatherAPIKey),
		Stocks:  api.NewYahooClient(),
		News:    api.NewNewsClient(),
	}
} else {
	deps = ui.Deps{
		Stocks: api.NewYahooClient(),
		News:   api.NewNewsClient(),
	}
}
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: FAIL only on `internal/ui/todo.go` (field type mismatch) — that's fixed in the next task.

- [ ] **Step 6: Commit (after todo.go is fixed in next task)**

Hold this commit until Task 5 completes.

---

## Task 5: Update `todo.go` field and function types

**Files:**
- Modify: `internal/ui/todo.go`

- [ ] **Step 1: Update the `gh` field type and all function signatures**

In `internal/ui/todo.go`, make these replacements:

```go
// struct field (line ~33)
type todoPane struct {
	gh         api.TodoBackend  // was: api.GitHub
	// ... rest unchanged
}

// constructor
func newTodoPane(gh api.TodoBackend) todoPane {  // was: api.GitHub
	ti := textinput.New()
	ti.Placeholder = "Issue title…"
	ti.CharLimit = 256
	return todoPane{gh: gh, loading: true, titleInput: ti}
}

// helper functions
func fetchIssues(gh api.TodoBackend) tea.Cmd {  // was: api.GitHub
	return func() tea.Msg {
		issues, err := gh.GetOpenIssues()
		return gotTodosMsg{issues: issues, err: err}
	}
}

func closeIssue(gh api.TodoBackend, number int) tea.Cmd {  // was: api.GitHub
	return func() tea.Msg {
		err := gh.CloseIssue(number)
		return closedIssueMsg{number: number, err: err}
	}
}

func submitCreateIssue(gh api.TodoBackend, title string) tea.Cmd {  // was: api.GitHub
	return func() tea.Msg {
		issue, err := gh.CreateIssue(title)
		if err != nil {
			return createdIssueMsg{err: err}
		}
		return createdIssueMsg{issue: *issue}
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 3: Commit Tasks 4 and 5 together**

```bash
git add internal/ui/app.go internal/ui/todo.go main.go
git commit -m "refactor: rename Deps.GitHub to Deps.Todo, wire TodoBackend by config"
```

---

## Task 6: Update wizard with backend selection step

**Files:**
- Modify: `internal/ui/wizard.go`

This task rewrites the wizard in several coordinated steps. All changes are in one file.

- [ ] **Step 1: Update step constants and add `todoBackend` field**

Replace the `const` block and `wizardModel` struct:

```go
type wizardStep int

const (
	stepTodoBackend wizardStep = iota // 0 — NEW first step
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
	step         wizardStep
	todoBackend  string // "github" or "local"
	inputs       []textinput.Model
	err          string
	width        int
	height       int
	panels       config.PanelConfig
	panelCursor  int
}
```

- [ ] **Step 2: Add `textIdx()` helper and update `newWizard` / `newWizardFrom`**

Add the helper right after the struct definition:

```go
// textIdx returns the inputs[] index for the current text-input step.
// stepGitHubRepo(1) → 0, stepGitHubToken(2) → 1, …, stepRSSURL(6) → 5.
func (m wizardModel) textIdx() int { return int(m.step) - 1 }
```

Update `newWizard` — the inputs array and default focus are unchanged (inputs[0] is still GitHub Repo):

```go
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
	// inputs[0] is focused when the user chooses "github" from the backend step
	return wizardModel{
		step:        stepTodoBackend,
		todoBackend: "github",
		inputs:      inputs,
		panels:      config.PanelConfig{Todo: true, Weather: true, Stocks: true, Stats: true, News: true},
	}
}
```

Update `newWizardFrom`:

```go
func newWizardFrom(cfg *config.Config) wizardModel {
	m := newWizard()
	if cfg == nil {
		return m
	}
	if cfg.TodoBackend == "local" {
		m.todoBackend = "local"
	} else {
		m.todoBackend = "github"
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
```

- [ ] **Step 3: Update `buildConfig` to include `TodoBackend`**

```go
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
```

- [ ] **Step 4: Update `validate()` to use `textIdx()`**

```go
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
```

- [ ] **Step 5: Update `Update()` to use `textIdx()` for input dispatch**

```go
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
```

- [ ] **Step 6: Update `handleKeyMsg` to dispatch backend step**

```go
func (m wizardModel) handleKeyMsg(msg tea.KeyMsg) (wizardModel, tea.Cmd) {
	if m.step == stepTodoBackend {
		return m.handleBackendStepKey(msg)
	}
	if m.step == stepPanels {
		return m.handlePanelStepKey(msg)
	}
	return m.handleTextInputKey(msg)
}
```

- [ ] **Step 7: Add `handleBackendStepKey`**

```go
func (m wizardModel) handleBackendStepKey(msg tea.KeyMsg) (wizardModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.err = ""
		if m.todoBackend == "local" {
			m.step = stepWeatherKey
		} else {
			m.step = stepGitHubRepo
		}
		m.inputs[m.textIdx()].Focus()
		return m, textinput.Blink
	}
	switch msg.String() {
	case "j", keyDown:
		m.todoBackend = "local"
	case "k", keyUp:
		m.todoBackend = "github"
	}
	return m, nil
}
```

- [ ] **Step 8: Update `handleTextInputKey` to use `textIdx()` and handle skip logic**

```go
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
		if m.todoBackend == "local" && (m.step == stepGitHubRepo || m.step == stepGitHubToken) {
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
```

- [ ] **Step 9: Update `handlePanelStepKey` backward navigation**

```go
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
		m.inputs[int(stepRSSURL)-1].Focus()  // inputs[5] = RSS URL
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
```

- [ ] **Step 10: Update `View()` to render the backend step and fix input indexing**

Replace the full `View()` method:

```go
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
		if m.todoBackend == "github" {
			githubLine = "  ▶ " + lipgloss.NewStyle().Bold(true).Render("github") + "  — GitHub Issues (requires token)"
		} else {
			localLine = "  ▶ " + lipgloss.NewStyle().Bold(true).Render("local") + "   — Local file (~/.config/today-tui/todos.json)"
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
```

- [ ] **Step 11: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 12: Run existing tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 13: Commit**

```bash
git add internal/ui/wizard.go
git commit -m "feat: add todo backend selection step to setup wizard"
```

---

## Self-Review

**Spec coverage check:**
- [x] Rename `GitHub` → `TodoBackend` — Task 1
- [x] `LocalTodoClient` with JSON at `~/.config/today-tui/todos.json` — Task 2
- [x] Missing file treated as empty list — Task 2, `load()` uses `os.IsNotExist`
- [x] Corrupt file returns error — Task 2, `json.Unmarshal` error propagated
- [x] `CloseIssue` unknown number is no-op — Task 2, loop finds nothing, saves unchanged
- [x] `HTMLURL` empty (open browser does nothing) — Task 2, `Issue.HTMLURL` left as zero value
- [x] `TodoBackend` config field — Task 3
- [x] `Deps.GitHub` → `Deps.Todo` — Task 4
- [x] `Deps.Refresh` instantiates correct client — Task 4
- [x] `todo.go` field/param types updated — Task 5
- [x] Wizard new first step, local skips GitHub steps — Task 6
- [x] `newWizardFrom` pre-populates `todoBackend` — Task 6, Step 2
- [x] `buildConfig` includes `TodoBackend` — Task 6, Step 3

**No placeholders:** Confirmed — all steps contain complete code.

**Type consistency:** `api.TodoBackend` used consistently across Tasks 1, 4, 5. `LocalTodoClient` implements `TodoBackend` verified by `var _ TodoBackend = (*LocalTodoClient)(nil)`. `textIdx()` used consistently across `validate`, `Update`, `handleTextInputKey`, `handlePanelStepKey`, `View`.
