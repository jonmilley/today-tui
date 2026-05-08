package ui

import (
	"strings"
	"time"

	"github.com/jonmilley/today-tui/internal/api"
	"github.com/jonmilley/today-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appMode int

const (
	modeSplash appMode = iota
	modeSetup
	modeDash
	modeConfig
)

const (
	paneTodo = iota
	paneCalendar
	paneWeather
	paneStocks
	paneStats
	paneNews
	paneCount
)

const (
	todoBackendLocal  = "local"
	todoBackendGitHub = "github"
)

// refreshTickMsg triggers periodic data refresh for slow-updating panes.
type refreshTickMsg struct{}

type Deps struct {
	Todo     api.TodoBackend
	Calendar api.Calendar
	Weather  api.Weather
	Stocks   api.Stocks
	News     api.News
}

func (d *Deps) Refresh(cfg *config.Config) {
	if cfg.TodoBackend == todoBackendLocal {
		d.Todo = api.NewLocalTodoClient()
	} else {
		d.Todo = api.NewGitHubClient(cfg.GitHubToken, cfg.GitHubRepo)
	}
	d.Weather = api.NewWeatherClient(cfg.WeatherAPIKey)
	d.Calendar = api.NewICSClient(cfg.CalendarURL)
	if d.Stocks == nil {
		d.Stocks = api.NewYahooClient()
	}
	if d.News == nil {
		d.News = api.NewNewsClient()
	}
}

type App struct {
	mode         appMode
	splash       splashModel
	wizard       wizardModel
	configEditor configEditor
	cfg          *config.Config
	deps         Deps

	todo     todoPane
	calendar calendarPane
	weather  weatherPane
	stocks   stocksPane
	stats    statsPane
	news     newsPane

	focused int
	width   int
	height  int
	ready   bool

	// rects records each visible pane's screen position so mouse events
	// can be hit-tested. Recomputed on every resize.
	rects []paneRect

	// status surfaces transient app-level messages (e.g. config save
	// failures) in the status bar. Cleared on the next pane-toggle or
	// config-close interaction.
	status string
}

// paneRect is the screen-coordinate bounding box of one rendered pane.
type paneRect struct {
	pane       int
	x, y, w, h int
}

// paneAt returns the pane id whose rendered rect contains the given screen
// coordinates. The bool is false when the click landed outside any pane
// (e.g. on the status bar).
func (a *App) paneAt(x, y int) (int, bool) {
	for _, r := range a.rects {
		if x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h {
			return r.pane, true
		}
	}
	return 0, false
}

func NewApp(cfg *config.Config, deps Deps) App {
	a := App{
		mode:   modeSplash,
		splash: newSplash(),
		wizard: newWizard(),
		cfg:    cfg,
		deps:   deps,
	}
	// When config is already present, pre-seed the dashboard panes so their
	// initial fetches can run during the splash rather than after it.
	if cfg != nil {
		a.deps.Refresh(cfg)
		a.seedDashPanes()
	}
	return a
}

// NewReconfigureApp launches the setup wizard pre-populated with values from
// an existing config so the user can edit rather than re-enter everything.
// Panes are not pre-seeded since the user is about to change credentials.
func NewReconfigureApp(existing *config.Config, deps Deps) App {
	return App{
		mode:   modeSplash,
		splash: newSplash(),
		wizard: newWizardFrom(existing),
		deps:   deps,
	}
}

// seedDashPanes builds the dashboard pane state in place. Used both when
// transitioning out of splash with a pre-existing config and when finishing
// the setup wizard. Caller must ensure a.cfg and a.deps are populated.
func (a *App) seedDashPanes() {
	a.todo = newTodoPane(a.deps.Todo)
	a.calendar = newCalendarPane(a.deps.Calendar, a.cfg.CalendarURL)
	a.weather = newWeatherPane(a.deps.Weather, a.cfg.WeatherCity, a.cfg.Units)
	a.stocks = newStocksPane(a.deps.Stocks, a.cfg.Stocks)
	a.stats = newStatsPane()
	a.news = newNewsPane(a.deps.News, a.cfg.RSSFeedURL)
	a.focused = paneTodo
}

func (a App) Init() tea.Cmd {
	// If panes were pre-seeded, kick off their initial fetches now so data
	// is loading in the background while the splash ticks.
	if a.cfg != nil {
		return tea.Batch(a.splash.Init(), a.initPanes())
	}
	return a.splash.Init()
}

func (a App) initPanes() tea.Cmd {
	return tea.Batch(
		a.todo.Init(),
		a.calendar.Init(),
		a.weather.Init(),
		a.stocks.Init(),
		a.stats.Init(),
		a.news.Init(),
		refreshTick(),
	)
}

// refreshIntervalSecs is the period of the global refresh tick. Per-pane
// fetches inside that period are staggered (see paneRefreshOffsets) so
// they don't all fire simultaneously and burst load on remote APIs.
const refreshIntervalSecs = 60

// paneRefreshOffsets distributes the four network-bound pane fetches
// evenly across the refresh window. Within each refreshInterval the
// staggered fetches fire at: weather (+0s), stocks (+15s), news (+30s),
// calendar (+45s). Before the first tick fires, every pane has already
// been seeded by its Init().
var paneRefreshOffsets = struct {
	weather, stocks, news, calendar time.Duration
}{
	weather:  0 * time.Second,
	stocks:   15 * time.Second,
	news:     30 * time.Second,
	calendar: 45 * time.Second,
}

func refreshTick() tea.Cmd {
	return tea.Tick(refreshIntervalSecs*time.Second, func(_ time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

// scheduleFetch returns a Cmd that fires msg after delay. delay==0 sends
// the message immediately on the next runtime turn.
func scheduleFetch(delay time.Duration, msg tea.Msg) tea.Cmd {
	if delay == 0 {
		return func() tea.Msg { return msg }
	}
	return tea.Tick(delay, func(_ time.Time) tea.Msg { return msg })
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleWindowSize(msg), nil

	case splashDoneMsg:
		return a.handleSplashDone()

	case SetupDoneMsg:
		return a.handleSetupDone(msg)

	case configClosedMsg:
		return a.handleConfigClosed(msg)

	case tea.MouseMsg:
		return a.handleMouseMsg(msg)

	case tea.KeyMsg:
		return a.handleKeyMsg(msg)
	}

	if a.mode == modeSplash {
		var splashCmd tea.Cmd
		a.splash, splashCmd = a.splash.Update(msg)
		// If panes were pre-seeded, also let them process background results
		// (fetch responses, refresh ticks) while the splash is showing.
		if a.cfg != nil {
			var paneCmd tea.Cmd
			a, paneCmd = a.dispatchToPanes(msg)
			return a, tea.Batch(splashCmd, paneCmd)
		}
		return a, splashCmd
	}

	if a.mode == modeSetup {
		var wizCmd tea.Cmd
		a.wizard, wizCmd = a.wizard.Update(msg)
		return a, wizCmd
	}

	return a.dispatchToPanes(msg)
}

func (a App) handleWindowSize(msg tea.WindowSizeMsg) App {
	a.width = msg.Width
	a.height = msg.Height
	a.ready = true
	switch a.mode {
	case modeSplash:
		a.splash.width = msg.Width
		a.splash.height = msg.Height
		// Pre-seeded panes need a size so background-rendered content is
		// laid out correctly when we flip to the dashboard.
		if a.cfg != nil {
			a.resizePanes()
		}
	case modeSetup:
		a.wizard.width = msg.Width
		a.wizard.height = msg.Height
	case modeConfig:
		a.configEditor.width = msg.Width
		a.configEditor.height = msg.Height
	default:
		a.resizePanes()
	}
	return a
}

func (a App) handleSplashDone() (App, tea.Cmd) {
	if a.cfg == nil {
		a.mode = modeSetup
		a.wizard.width = a.width
		a.wizard.height = a.height
		return a, a.wizard.Init()
	}
	// Panes were seeded and their fetches kicked off in NewApp; just flip
	// to the dashboard. resizePanes was already called on window-size.
	a.mode = modeDash
	if a.ready {
		a.resizePanes()
	}
	return a, nil
}

func (a App) handleSetupDone(msg SetupDoneMsg) (App, tea.Cmd) {
	a.cfg = msg.Cfg
	a.deps.Refresh(msg.Cfg)
	a.seedDashPanes()
	a.mode = modeDash
	if a.ready {
		a.resizePanes()
	}
	return a, a.initPanes()
}

func (a App) handleConfigClosed(msg configClosedMsg) (App, tea.Cmd) {
	*a.cfg = msg.cfg
	if err := a.cfg.Save(); err != nil {
		a.status = "Save failed: " + err.Error()
	} else {
		a.status = ""
	}
	// Settings (token, URLs, city, units, stocks, RSS) may have changed —
	// rebuild the API clients and the panes so the new values take effect
	// without requiring a restart, and re-fetch their data.
	a.deps.Refresh(a.cfg)
	a.seedDashPanes()
	a.mode = modeDash
	a.ensureFocusVisible()
	if a.ready {
		a.resizePanes()
	}
	return a, a.initPanes()
}

func (a App) handleKeyMsg(msg tea.KeyMsg) (App, tea.Cmd) {
	if a.mode == modeSplash {
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		return a, func() tea.Msg { return splashDoneMsg{} }
	}
	if a.mode == modeSetup {
		var wizCmd tea.Cmd
		a.wizard, wizCmd = a.wizard.Update(msg)
		return a, wizCmd
	}
	if a.mode == modeConfig {
		var cmd tea.Cmd
		a.configEditor, cmd = a.configEditor.Update(msg)
		return a, cmd
	}
	// While a pane is capturing input (e.g. create form), send ALL keys
	// there so Tab/q/etc don't trigger global actions.
	if a.focused == paneTodo && a.todo.IsCapturing() {
		return a, a.dispatchKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit
	case ",":
		a.configEditor = newConfigEditor(*a.cfg)
		a.configEditor.width = a.width
		a.configEditor.height = a.height
		a.mode = modeConfig
		return a, nil
	case "tab":
		a.cycleFocus(1)
		return a, nil
	case "shift+tab":
		a.cycleFocus(-1)
		return a, nil
	default:
		return a, a.dispatchKey(msg)
	}
}

func (a App) dispatchToPanes(msg tea.Msg) (App, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	a.todo, cmd = a.todo.Update(msg)
	cmds = append(cmds, cmd)

	a.calendar, cmd = a.calendar.Update(msg)
	cmds = append(cmds, cmd)

	a.weather, cmd = a.weather.Update(msg)
	cmds = append(cmds, cmd)

	a.stocks, cmd = a.stocks.Update(msg)
	cmds = append(cmds, cmd)

	a.stats, cmd = a.stats.Update(msg)
	cmds = append(cmds, cmd)

	a.news, cmd = a.news.Update(msg)
	cmds = append(cmds, cmd)

	// Periodic refresh — stagger fetches so they don't all hit the network
	// (and visibly redraw) at the same instant. Stats refreshes on its
	// own 3s cadence; todo only refreshes on demand or at startup.
	if _, ok := msg.(refreshTickMsg); ok {
		cmds = append(cmds,
			scheduleFetch(paneRefreshOffsets.weather, fetchWeatherMsg{}),
			scheduleFetch(paneRefreshOffsets.stocks, fetchStocksMsg{}),
			scheduleFetch(paneRefreshOffsets.news, fetchNewsMsg{}),
			scheduleFetch(paneRefreshOffsets.calendar, fetchCalendarMsg{}),
			refreshTick(),
		)
	}

	return a, tea.Batch(cmds...)
}

// handleMouseMsg routes mouse events to the pane under the cursor. Wheel
// events scroll that pane regardless of focus; left-button presses move
// focus to the clicked pane and forward the press through. All other
// mouse events (motion, release, other buttons) are ignored.
func (a App) handleMouseMsg(msg tea.MouseMsg) (App, tea.Cmd) {
	if a.mode != modeDash {
		return a, nil
	}
	pane, ok := a.paneAt(msg.X, msg.Y)
	if !ok {
		return a, nil
	}
	switch {
	case tea.MouseEvent(msg).IsWheel():
		return a.routeMouseToPane(pane, msg)
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		if a.focused != pane {
			a.setFocus(pane)
		}
		return a.routeMouseToPane(pane, msg)
	}
	return a, nil
}

// routeMouseToPane forwards a single mouse event to one specific pane's
// Update — distinct from dispatchToPanes which broadcasts to all panes —
// so wheel events scroll only the pane being hovered, not every viewport.
func (a App) routeMouseToPane(pane int, msg tea.MouseMsg) (App, tea.Cmd) {
	var cmd tea.Cmd
	switch pane {
	case paneTodo:
		a.todo, cmd = a.todo.Update(msg)
	case paneCalendar:
		a.calendar, cmd = a.calendar.Update(msg)
	case paneWeather:
		a.weather, cmd = a.weather.Update(msg)
	case paneStocks:
		a.stocks, cmd = a.stocks.Update(msg)
	case paneStats:
		a.stats, cmd = a.stats.Update(msg)
	case paneNews:
		a.news, cmd = a.news.Update(msg)
	}
	return a, cmd
}

func (a *App) dispatchKey(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	switch a.focused {
	case paneTodo:
		a.todo, cmd = a.todo.Update(msg)
	case paneCalendar:
		a.calendar, cmd = a.calendar.Update(msg)
	case paneWeather:
		a.weather, cmd = a.weather.Update(msg)
	case paneStocks:
		a.stocks, cmd = a.stocks.Update(msg)
	case paneStats:
		// stats has no keyboard interaction beyond tab
	case paneNews:
		a.news, cmd = a.news.Update(msg)
	}
	return cmd
}

func (a *App) visiblePanes() []int {
	var panes []int
	panes = append(panes, a.visibleLeftPanes()...)
	panes = append(panes, a.visibleRightPanes()...)
	return panes
}

// visibleLeftPanes returns the list-shaped panes that stack vertically on
// the left half of the dashboard.
func (a *App) visibleLeftPanes() []int {
	var panes []int
	if a.cfg.Panels.Todo {
		panes = append(panes, paneTodo)
	}
	if a.cfg.Panels.Calendar {
		panes = append(panes, paneCalendar)
	}
	return panes
}

func (a *App) visibleRightPanes() []int {
	var panes []int
	if a.cfg.Panels.Weather {
		panes = append(panes, paneWeather)
	}
	if a.cfg.Panels.Stocks {
		panes = append(panes, paneStocks)
	}
	if a.cfg.Panels.Stats {
		panes = append(panes, paneStats)
	}
	if a.cfg.Panels.News {
		panes = append(panes, paneNews)
	}
	return panes
}

func (a *App) cycleFocus(direction int) {
	visible := a.visiblePanes()
	if len(visible) == 0 {
		return
	}
	idx := -1
	for i, p := range visible {
		if p == a.focused {
			idx = i
			break
		}
	}
	if idx == -1 {
		a.setFocus(visible[0])
		return
	}
	next := (idx + direction + len(visible)) % len(visible)
	a.setFocus(visible[next])
}

func (a *App) ensureFocusVisible() {
	visible := a.visiblePanes()
	if len(visible) == 0 {
		return
	}
	for _, p := range visible {
		if p == a.focused {
			return
		}
	}
	a.setFocus(visible[0])
}

func (a *App) setFocus(p int) {
	a.focused = p
	a.syncFocus()
}

func (a *App) syncFocus() {
	a.todo.SetFocused(a.focused == paneTodo)
	a.calendar.SetFocused(a.focused == paneCalendar)
	a.weather.SetFocused(a.focused == paneWeather)
	a.stocks.SetFocused(a.focused == paneStocks)
	a.stats.SetFocused(a.focused == paneStats)
	a.news.SetFocused(a.focused == paneNews)
}

func (a *App) setPaneSize(pane, w, h int) {
	switch pane {
	case paneTodo:
		a.todo.SetSize(w, h)
	case paneCalendar:
		a.calendar.SetSize(w, h)
	case paneWeather:
		a.weather.SetSize(w, h)
	case paneStocks:
		a.stocks.SetSize(w, h)
	case paneStats:
		a.stats.SetSize(w, h)
	case paneNews:
		a.news.SetSize(w, h)
	}
}

// placePane sets a pane's size and records its screen rect for mouse
// hit-testing. Origin (x, y) is in cells from the top-left of the screen.
func (a *App) placePane(pane, x, y, w, h int) {
	a.rects = append(a.rects, paneRect{pane: pane, x: x, y: y, w: w, h: h})
	a.setPaneSize(pane, w, h)
}

func (a *App) layoutRow(panes []int, x, y, totalW, h int) {
	if len(panes) == 0 || h == 0 {
		return
	}
	if len(panes) == 1 {
		a.placePane(panes[0], x, y, totalW, h)
		return
	}
	w1 := totalW / 2
	w2 := totalW - w1
	a.placePane(panes[0], x, y, w1, h)
	a.placePane(panes[1], x+w1, y, w2, h)
}

func (a *App) resizePanes() {
	if a.width == 0 || a.height == 0 {
		return
	}
	a.rects = a.rects[:0]
	statusH := 1
	availH := a.height - statusH

	leftVisible := a.visibleLeftPanes()
	rightVisible := a.visibleRightPanes()

	leftW, rightW := splitColumns(a.width, len(leftVisible) > 0, len(rightVisible) > 0)
	a.layoutLeftColumn(leftVisible, 0, 0, leftW, availH)

	n := len(rightVisible)
	if n == 0 || rightW == 0 {
		a.syncFocus()
		return
	}

	topH := availH
	botH := 0
	if n > 2 {
		topH = availH / 2
		botH = availH - topH
	}

	r1 := rightVisible
	r2 := []int{}
	if n > 2 {
		r1 = rightVisible[:2]
		r2 = rightVisible[2:]
	}

	a.layoutRow(r1, leftW, 0, rightW, topH)
	if len(r2) > 0 {
		a.layoutRow(r2, leftW, topH, rightW, botH)
	}

	a.syncFocus()
}

// splitColumns returns (leftW, rightW) for the dashboard. When both columns
// are present the left takes 2/5 (matching the original Todo-only ratio).
// When only one side is present it takes the full width.
func splitColumns(total int, leftHasPanes, rightHasPanes bool) (int, int) {
	switch {
	case leftHasPanes && rightHasPanes:
		left := total * 2 / 5
		return left, total - left
	case leftHasPanes:
		return total, 0
	case rightHasPanes:
		return 0, total
	default:
		return 0, 0
	}
}

// layoutLeftColumn stacks the left panes vertically starting at (x, y),
// splitting availH evenly (last pane absorbs the remainder so the column
// fills exactly).
func (a *App) layoutLeftColumn(panes []int, x, y, w, availH int) {
	if len(panes) == 0 || w == 0 {
		return
	}
	if len(panes) == 1 {
		a.placePane(panes[0], x, y, w, availH)
		return
	}
	per := availH / len(panes)
	cy := y
	for i, p := range panes {
		h := per
		if i == len(panes)-1 {
			h = availH - per*(len(panes)-1)
		}
		a.placePane(p, x, cy, w, h)
		cy += h
	}
}

func (a App) paneView(pane int) string {
	switch pane {
	case paneTodo:
		return a.todo.View()
	case paneCalendar:
		return a.calendar.View()
	case paneWeather:
		return a.weather.View()
	case paneStocks:
		return a.stocks.View()
	case paneStats:
		return a.stats.View()
	case paneNews:
		return a.news.View()
	}
	return ""
}

func (a App) buildRow(panes []int) string {
	views := make([]string, len(panes))
	for i, p := range panes {
		views[i] = a.paneView(p)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}

func (a App) View() string {
	if !a.ready {
		return "Initializing..."
	}
	if a.mode == modeSplash {
		return a.splash.View()
	}
	if a.mode == modeSetup {
		return a.wizard.View()
	}
	if a.mode == modeConfig {
		return a.configEditor.View()
	}

	leftVisible := a.visibleLeftPanes()
	rightVisible := a.visibleRightPanes()

	var left string
	if len(leftVisible) > 0 {
		views := make([]string, len(leftVisible))
		for i, p := range leftVisible {
			views[i] = a.paneView(p)
		}
		left = lipgloss.JoinVertical(lipgloss.Left, views...)
	}

	var right string
	if len(rightVisible) > 0 {
		r1 := rightVisible
		r2 := []int{}
		if len(rightVisible) > 2 {
			r1 = rightVisible[:2]
			r2 = rightVisible[2:]
		}
		rows := []string{a.buildRow(r1)}
		if len(r2) > 0 {
			rows = append(rows, a.buildRow(r2))
		}
		right = lipgloss.JoinVertical(lipgloss.Left, rows...)
	}

	var main string
	switch {
	case left != "" && right != "":
		main = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	case left != "":
		main = left
	default:
		main = right
	}

	statusBar := buildStatusBar(a.width, a.focused, a.status)
	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}

func buildStatusBar(w, focused int, status string) string {
	// panelToggles is maintained in the same order as the pane* iota
	// (Todo, Calendar, Weather, Stocks, Stats, News), so its labels
	// double as the focused-pane name shown in the status bar.
	name := ""
	if focused >= 0 && focused < len(panelToggles) {
		name = panelToggles[focused].label
	}
	left := dimStyle.Render("  Tab: next pane  ,: config  q: quit  r: refresh")
	// Right side normally shows the focused pane name, but if there's a
	// transient app-level status message (e.g. config save failure), show
	// that instead in the error color.
	var right string
	if status != "" {
		right = errStyle.Render(truncate(status, w/2) + "  ")
	} else {
		right = dimStyle.Render("Focus: " + name + "  ")
	}
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Width(w).
		Render(left + strings.Repeat(" ", gap) + right)
}
