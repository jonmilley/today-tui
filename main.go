package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jonmilley/today-tui/internal/api"
	"github.com/jonmilley/today-tui/internal/config"
	"github.com/jonmilley/today-tui/internal/ui"
)

func main() {
	reconfigure := len(os.Args) > 1 && os.Args[1] == "--reconfigure"

	cfg, _ := config.Load()

	// Best-effort client init. If some keys are missing, the UI will show
	// appropriate errors or the setup wizard will collect them.
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

	var app ui.App
	if reconfigure {
		app = ui.NewReconfigureApp(cfg, deps)
	} else {
		if cfg == nil {
			// Trigger setup wizard by passing nil config
			app = ui.NewApp(nil, deps)
		} else {
			app = ui.NewApp(cfg, deps)
		}
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
