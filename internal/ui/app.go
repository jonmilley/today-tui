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
	mode    appMode
	wizard  wizardModel
	cfg     *config.Config

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
		} else {
			a.resizePanes()
		}
		return a, nil

	case SetupDoneMsg:
		a = buildDash(msg.Cfg)
		return a, a.Init()

	case tea.KeyMsg:
		if a.mode == modeSetup {
			var wizCmd tea.Cmd
			a.wizard, wizCmd = a.wizard.Update(msg)
			return a, wizCmd
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
		case "tab":
			a.setFocus((a.focused + 1) % paneCount)
		case "shift+tab":
			a.setFocus((a.focused + paneCount - 1) % paneCount)
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

func (a *App) setFocus(p int) {
	a.focused = p
	a.todo.SetFocused(p == paneTodo)
	a.weather.SetFocused(p == paneWeather)
	a.stocks.SetFocused(p == paneStocks)
	a.stats.SetFocused(p == paneStats)
	a.news.SetFocused(p == paneNews)
}

func (a *App) resizePanes() {
	if a.width == 0 || a.height == 0 {
		return
	}
	statusH := 1
	availH := a.height - statusH

	todoW := a.width * 2 / 5
	rightW := a.width - todoW
	rightPaneW := rightW / 2
	rightPaneW2 := rightW - rightPaneW // handles odd widths

	topH := availH / 2
	botH := availH - topH

	a.todo.SetSize(todoW, availH)
	a.weather.SetSize(rightPaneW, topH)
	a.stocks.SetSize(rightPaneW2, topH)
	a.stats.SetSize(rightPaneW, botH)
	a.news.SetSize(rightPaneW2, botH)

	// sync focus state
	a.todo.SetFocused(a.focused == paneTodo)
	a.weather.SetFocused(a.focused == paneWeather)
	a.stocks.SetFocused(a.focused == paneStocks)
	a.stats.SetFocused(a.focused == paneStats)
	a.news.SetFocused(a.focused == paneNews)
}

func (a App) View() string {
	if !a.ready {
		return "Initializing..."
	}
	if a.mode == modeSetup {
		return a.wizard.View()
	}

	// Left: todo (full height)
	left := a.todo.View()

	// Right: 2x2 grid
	topRight := lipgloss.JoinHorizontal(lipgloss.Top,
		a.weather.View(),
		a.stocks.View(),
	)
	botRight := lipgloss.JoinHorizontal(lipgloss.Top,
		a.stats.View(),
		a.news.View(),
	)
	right := lipgloss.JoinVertical(lipgloss.Left, topRight, botRight)

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	statusBar := buildStatusBar(a.width, a.focused)

	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}

func buildStatusBar(w, focused int) string {
	paneNames := []string{"Todo", "Weather", "Stocks", "Stats", "News"}
	name := ""
	if focused >= 0 && focused < len(paneNames) {
		name = paneNames[focused]
	}
	left := dimStyle.Render("  Tab: next pane  q: quit  r: refresh")
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
