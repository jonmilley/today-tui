# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

- `make build` — build the `today` binary at the repo root (uses `-ldflags="-s -w" -trimpath`).
- `make install` — build and install to `~/bin/today`.
- `go run main.go` — run during development. Add `-- --reconfigure` (or just `go run . --reconfigure`) to re-run the setup wizard against an existing config.
- `go test ./...` — run all tests. Test a single package with `go test ./internal/api`, a single test with `go test ./internal/config -run TestLoad`.
- `golangci-lint run` — lint. Config in `.golangci.yml` (v2 schema, `goimports` local prefix `github.com/jonmilley/today-tui`, `lll` line length 120, `gocyclo` complexity 15). CI runs lint + tests on Go 1.24.
- The compiled artifacts (`today`, `today_debug`, `today-tui`) are checked into the working tree but are build outputs, not source. Don't edit them.

## Architecture

A Bubble Tea (charmbracelet) TUI dashboard. The whole program is one `tea.Program` running a single root model (`ui.App`). State machine, layout, and pane dispatch all live in `internal/ui/app.go`.

### Three packages

- `internal/config` — `Config` struct (loaded from `~/.config/today-tui/config.json`), with defaults applied on `Load` (default stocks list, units default `F`, all panels default to enabled when the `panels` key is absent).
- `internal/api` — interfaces and HTTP clients for external services. Each pane talks only to an interface, never a concrete type. Key interfaces: `TodoBackend` (GitHub issues *or* a local JSON store at `~/.config/today-tui/todos.json`), `Weather`, `Stocks`, `News`. Concrete clients: `GitHubClient`, `LocalTodoClient`, `WeatherClient`, `YahooClient` (cookie/crumb auth, no API key), `NewsClient` (RSS/Atom via `gofeed`).
- `internal/ui` — one file per pane (`todo.go`, `weather.go`, `stocks.go`, `stats.go`, `news.go`) plus `app.go` (root model), `splash.go`, `wizard.go` (first-run setup), `config_editor.go` (runtime panel toggle), `styles.go` (lipgloss styles, `boldStyle` is package-level).

### App modes (state machine)

`App.mode` cycles through `modeSplash` → `modeSetup` (only if no config) → `modeDash` ⇄ `modeConfig` (`,` opens the panel toggle editor). `Update` routes messages by mode; in dash mode, key messages go through `dispatchKey` to the focused pane, while non-key messages broadcast to all panes via `dispatchToPanes`. A `refreshTickMsg` fires every 60s to re-fetch weather/stocks/news; the stats pane self-refreshes every 3s. The todo pane refreshes only on startup or when the user presses `r`.

### Dependency injection

`main.go` builds a `ui.Deps` struct of API interfaces and passes it to `NewApp`/`NewReconfigureApp`. When config changes (setup wizard finishes, todo backend toggled), `deps.Refresh(cfg)` rebuilds the affected clients — in particular it swaps `Todo` between `GitHubClient` and `LocalTodoClient` based on `cfg.TodoBackend`. The constants `todoBackendLocal` and `todoBackendGitHub` are defined in `app.go`; use them rather than string literals to satisfy `goconst`.

### Layout

The dashboard is a fixed 5-pane grid: Todo on the left at 2/5 width, the right column splits into a top row (first 1–2 visible right panes) and bottom row (remaining 0–2). `PanelConfig` lets users hide any pane; `visiblePanes` / `visibleRightPanes` drive both rendering (`View`) and sizing (`resizePanes`). Focus only cycles through visible panes — when toggling visibility off, `ensureFocusVisible` snaps focus to the first visible one.

### Pane input capture

A pane can declare it's "capturing" all keys (e.g. the todo create-issue inline form). When `todo.IsCapturing()` is true, every key — including `Tab`, `q`, `,` — is forwarded to that pane and global handlers are skipped. If you add a pane with modal input, follow the same pattern.

### Setup wizard step indexing

`wizard.go` defines steps as an enum starting at `stepTodoBackend = 0`, but `inputs` is a slice of text inputs that only covers steps 1–6. Use the `textIdx()` helper (`int(step) - 1`) to map the current step to its `inputs[]` slot, rather than hardcoding offsets.

### Issue model used by both todo backends

`api.Issue` is the GitHub-shaped struct (`Number`, `Title`, `State`, `HTMLURL`, `Labels`, `CreatedAt`). The local backend reuses this struct and leaves `HTMLURL` empty for local todos; the todo pane skips `openBrowser` when `HTMLURL == ""`.
