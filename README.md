# today-tui

A terminal dashboard that puts everything you need to start your day in one place.

```
╭────────────────────────────╮╭─────────────────╮╭─────────────────╮
│ TODO              5 open   ││ WEATHER         ││ STOCKS          │
│ ──────────────────────────── ──────────────────── ────────────────│
│▶ #12 Fix CI pipeline  [bug]││ St. John's, CA  ││ SYM    PRICE    │
│  #11 Add dark mode [feature]││                 ││ ─────────────── │
│  #10 Update deps    [chore]││  23°F / -5°C    ││ SPY   512.34    │
│  #9  Write tests    [docs] ││  Feels like 18°F││ QQQ   445.67    │
│  #8  Refactor auth         ││                 ││ AAPL  178.23    │
│                            ││  Light snow      ││ GOOGL 142.56    │
│                            ││  Humidity: 82%  ││ META  478.90    │
│                            ││  Wind: 14mph NE ││ AMZN  185.34    │
│                            ││                 ││ NFLX  612.45    │
│                            ││  Updated: 9:12AM││                 │
│                            │╰─────────────────╯╰─────────────────╯
│                            │╭─────────────────╮╭─────────────────╮
│                            ││ SYSTEM          ││ NEWS            │
│                            ││ ────────────────││ ──────────────── │
│                            ││ CPU  [████░] 42%││▶ Go 1.24 releas…│
│                            ││ TEMP 51°C/124°F ││  3m ago         │
│                            ││ MEM  [████░] 61%││                 │
│                            ││      9.8/16.0 GB││  New CVE discov…│
│                            ││ DISK [██░░░] 38%││  1h ago         │
│                            ││                 ││                 │
│                            ││ Uptime: 3d 2h5m ││  Linux 6.9 rele…│
│  n: new  j/k  c  Enter  r  │╰─────────────────╯│  2h ago         │
╰────────────────────────────╯╰─────────────────────────────────────╯
```

## Features

| Pane | Description |
|------|-------------|
| **Todo** | GitHub Issues as a todo list — navigate, create, and close issues without leaving the terminal |
| **Weather** | Current conditions via OpenWeatherMap — temperature, feels-like, humidity, wind |
| **Stocks** | Live quotes via Yahoo Finance — price, change, % change, and company name when there's room |
| **System** | CPU usage + temperature, RAM, disk, and uptime — refreshes every 3 seconds |
| **News** | Any RSS/Atom feed — inline article preview with HTML stripped to plain text |

## Requirements

- **Go 1.24+**
- A free [OpenWeatherMap API key](https://openweathermap.org/api)
- A GitHub personal access token with `repo` scope (for the Todo pane)

## Build

```bash
git clone https://github.com/jonmilley/today-tui
cd today-tui
go build -o today-tui .
```

Or install directly:

```bash
go install github.com/jonmilley/today-tui@latest
```

## First Run

On first launch the setup wizard collects your credentials and saves them to `~/.config/today-tui/config.json`:

```
$ ./today-tui

  today-tui — First Launch Setup          Step 1 / 6

  GitHub Repository
  Format: owner/repo (e.g. acme/tasks)

  > jonmilley/todos

  Enter: confirm  •  Esc: back  •  Ctrl+C: quit
```

The wizard asks for:

1. **GitHub repo** — the `owner/repo` that backs your todo list
2. **GitHub token** — personal access token with `repo` scope ([create one here](https://github.com/settings/tokens))
3. **OpenWeatherMap API key** — free at [openweathermap.org/api](https://openweathermap.org/api) *(new keys activate within 2 hours)*
4. **Weather city** — city name, e.g. `London` or `New York`
5. **RSS feed URL** *(optional)* — any RSS or Atom feed, e.g. `https://hnrss.org/frontpage`

## Configuration

Config is stored at `~/.config/today-tui/config.json`:

```json
{
  "github_repo":      "owner/repo",
  "github_token":     "ghp_...",
  "weather_api_key":  "...",
  "weather_city":     "London",
  "stocks": ["SPY", "QQQ", "AAPL", "GOOGL", "META", "AMZN", "NFLX"],
  "rss_feed_url":     "https://hnrss.org/frontpage"
}
```

To re-run the setup wizard at any time:

```bash
./today-tui --reconfigure
```

## Key Bindings

### Global

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle focus between panes |
| `q` / `Ctrl+C` | Quit |

### Todo pane

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate issues |
| `n` | Create new issue (inline form) |
| `c` | Close (complete) selected issue |
| `Enter` | Open selected issue in browser |
| `r` | Refresh |

### News pane

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate items |
| `Enter` / `Space` | Toggle inline article preview |
| `Esc` | Close preview |
| `o` | Open article in browser |
| `r` | Refresh feed |

## Refresh Intervals

| Pane | Interval |
|------|----------|
| System stats | Every 3 seconds |
| Weather, Stocks, News | Every 60 seconds |
| Todo | On demand (`r`) or at startup |

## Data Sources

- **Weather** — [OpenWeatherMap](https://openweathermap.org) current conditions API
- **Stocks** — [Yahoo Finance](https://finance.yahoo.com) (cookie/crumb authenticated, no paid key required)
- **News** — Any RSS 2.0 or Atom feed via [gofeed](https://github.com/mmcdole/gofeed)
- **Todo** — [GitHub Issues API](https://docs.github.com/en/rest/issues)
- **System** — [gopsutil](https://github.com/shirou/gopsutil) (CPU, memory, disk, temperature)
