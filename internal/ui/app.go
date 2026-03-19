package ui

import (
	"time"

	"today-tui/internal/api"
	"today-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appMode int

const (
	modeSetup appMode = iota
	modeDash
	modeConfig
)

const (
	paneTodo = iota
	paneWeather
	paneStocks
	paneStats
	paneNews
	paneCount
)

// refreshTickMsg triggers periodic data refresh for slow-updating panes.
type refreshTickMsg struct{}

type App struct {
	mode         appMode
	wizard       wizardModel
	configEditor configEditor
	cfg          *config.Config

	todo    todoPane
	weather weatherPane
	stocks  stocksPane
	stats   statsPane
	news    newsPane

	focused int
	width   int
	height  int
	ready   bool
}

func NewApp(cfg *config.Config) App {
	if cfg == nil {
		return App{mode: modeSetup, wizard: newWizard()}
	}
	return buildDash(cfg)
}

func buildDash(cfg *config.Config) App {
	gh := api.NewGitHubClient(cfg.GitHubToken, cfg.GitHubRepo)
	yc := api.NewYahooClient()
	return App{
		mode:    modeDash,
		cfg:     cfg,
		todo:    newTodoPane(gh),
		weather: newWeatherPane(cfg.WeatherAPIKey, cfg.WeatherCity, cfg.Units),
		stocks:  newStocksPane(yc, cfg.Stocks),
		stats:   newStatsPane(),
		news:    newNewsPane(cfg.RSSFeedURL),
		focused: paneTodo,
	}
}

func (a App) Init() tea.Cmd {
	if a.mode == modeSetup {
		return a.wizard.Init()
	}
	return tea.Batch(
		a.todo.Init(),
		a.weather.Init(),
		a.stocks.Init(),
		a.stats.Init(),
		a.news.Init(),
		refreshTick(),
	)
}

func refreshTick() tea.Cmd {
	return tea.Tick(60*time.Second, func(_ time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		if a.mode == modeSetup {
			a.wizard.width = msg.Width
			a.wizard.height = msg.Height
		} else if a.mode == modeConfig {
			a.configEditor.width = msg.Width
			a.configEditor.height = msg.Height
		} else {
			a.resizePanes()
		}
		return a, nil

	case SetupDoneMsg:
		w, h, ready := a.width, a.height, a.ready
		a = buildDash(msg.Cfg)
		a.width, a.height, a.ready = w, h, ready
		if ready {
			a.resizePanes()
		}
		return a, a.Init()

	case configClosedMsg:
		a.cfg.Panels = msg.panels
		_ = a.cfg.Save()
		a.mode = modeDash
		a.ensureFocusVisible()
		a.resizePanes()
		return a, nil

	case tea.KeyMsg:
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
			cmds = append(cmds, a.dispatchKey(msg))
			return a, tea.Batch(cmds...)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case ",":
			a.configEditor = newConfigEditor(a.cfg.Panels)
			a.configEditor.width = a.width
			a.configEditor.height = a.height
			a.mode = modeConfig
		case "tab":
			a.cycleFocus(1)
		case "shift+tab":
			a.cycleFocus(-1)
		default:
			cmds = append(cmds, a.dispatchKey(msg))
		}
		return a, tea.Batch(cmds...)
	}

	if a.mode == modeSetup {
		var wizCmd tea.Cmd
		a.wizard, wizCmd = a.wizard.Update(msg)
		return a, wizCmd
	}

	// Dispatch data messages to panes
	var cmd tea.Cmd

	a.todo, cmd = a.todo.Update(msg)
	cmds = append(cmds, cmd)

	a.weather, cmd = a.weather.Update(msg)
	cmds = append(cmds, cmd)

	a.stocks, cmd = a.stocks.Update(msg)
	cmds = append(cmds, cmd)

	a.stats, cmd = a.stats.Update(msg)
	cmds = append(cmds, cmd)

	a.news, cmd = a.news.Update(msg)
	cmds = append(cmds, cmd)

	// Periodic refresh
	if _, ok := msg.(refreshTickMsg); ok {
		cmds = append(cmds,
			func() tea.Msg { return fetchWeatherMsg{} },
			func() tea.Msg { return fetchStocksMsg{} },
			func() tea.Msg { return fetchNewsMsg{} },
			refreshTick(),
		)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) dispatchKey(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	switch a.focused {
	case paneTodo:
		a.todo, cmd = a.todo.Update(msg)
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
	if a.cfg.Panels.Todo {
		panes = append(panes, paneTodo)
	}
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
	a.todo.SetFocused(p == paneTodo)
	a.weather.SetFocused(p == paneWeather)
	a.stocks.SetFocused(p == paneStocks)
	a.stats.SetFocused(p == paneStats)
	a.news.SetFocused(p == paneNews)
}

func (a *App) syncFocus() {
	a.todo.SetFocused(a.focused == paneTodo)
	a.weather.SetFocused(a.focused == paneWeather)
	a.stocks.SetFocused(a.focused == paneStocks)
	a.stats.SetFocused(a.focused == paneStats)
	a.news.SetFocused(a.focused == paneNews)
}

func (a *App) setPaneSize(pane, w, h int) {
	switch pane {
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

func (a *App) layoutRow(panes []int, totalW, h int) {
	if len(panes) == 0 || h == 0 {
		return
	}
	if len(panes) == 1 {
		a.setPaneSize(panes[0], totalW, h)
		return
	}
	w1 := totalW / 2
	w2 := totalW - w1
	a.setPaneSize(panes[0], w1, h)
	a.setPaneSize(panes[1], w2, h)
}

func (a *App) resizePanes() {
	if a.width == 0 || a.height == 0 {
		return
	}
	statusH := 1
	availH := a.height - statusH

	rightVisible := a.visibleRightPanes()

	rightW := a.width
	if a.cfg.Panels.Todo {
		todoW := a.width
		if len(rightVisible) > 0 {
			todoW = a.width * 2 / 5
			rightW = a.width - todoW
		} else {
			rightW = 0
		}
		a.todo.SetSize(todoW, availH)
	}

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

	a.layoutRow(r1, rightW, topH)
	if len(r2) > 0 {
		a.layoutRow(r2, rightW, botH)
	}

	a.syncFocus()
}

func (a App) paneView(pane int) string {
	switch pane {
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
	if a.mode == modeSetup {
		return a.wizard.View()
	}
	if a.mode == modeConfig {
		return a.configEditor.View()
	}

	rightVisible := a.visibleRightPanes()

	var left string
	if a.cfg.Panels.Todo {
		left = a.todo.View()
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

	statusBar := buildStatusBar(a.width, a.focused)
	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}

func buildStatusBar(w, focused int) string {
	paneNames := []string{"Todo", "Weather", "Stocks", "Stats", "News"}
	name := ""
	if focused >= 0 && focused < len(paneNames) {
		name = paneNames[focused]
	}
	left := dimStyle.Render("  Tab: next pane  ,: panels  q: quit  r: refresh")
	right := dimStyle.Render("Focus: " + name + "  ")
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	spacer := ""
	for i := 0; i < gap; i++ {
		spacer += " "
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Width(w).
		Render(left + spacer + right)
}
