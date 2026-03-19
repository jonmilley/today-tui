package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"today-tui/internal/config"
	"today-tui/internal/ui"
)

func main() {
	reconfigure := len(os.Args) > 1 && os.Args[1] == "--reconfigure"

	var app ui.App
	if reconfigure {
		existing, _ := config.Load() // best-effort; nil is fine if no config exists yet
		app = ui.NewReconfigureApp(existing)
	} else {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		app = ui.NewApp(cfg)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
