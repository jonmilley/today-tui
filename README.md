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
| **Todo** | Either GitHub Issues (navigate, create, close) or a local JSON file — toggle backends in the setup wizard |
| **Calendar** | Upcoming events from any iCalendar source (Google, iCloud, Outlook, local `.ics` file) over the next 7 days, with an embedded mini month view when the pane is tall enough |
| **Weather** | Current conditions and tomorrow's forecast via OpenWeatherMap — temperature, feels-like, humidity, wind. Imperial (°F, mph) or Metric (°C, kph) |
| **Stocks** | Live quotes via Yahoo Finance — price, change, % change, and company name when there's room |
| **System** | CPU usage + temperature, RAM, disk, and uptime — refreshes every 3 seconds |
| **News** | One or many RSS/Atom feeds, merged and sorted by timestamp. Multi-feed entries are prefixed with the source host (e.g. `[vocm.com]`). Inline article preview with HTML stripped to plain text |

## Install

### Homebrew (macOS)

```bash
brew install jonmilley/tap/today
```

> **macOS Security Warning:** Because this tool isn't signed with an Apple Developer certificate, macOS will block it on first run. To unblock:
> 1. Try to run `today`, then click **Cancel** on the warning.
> 2. Open **System Settings** > **Privacy & Security**.
> 3. Click **Open Anyway** next to the notice about `today`.
>
> **Or, run this command to whitelist it immediately:**
> ```bash
> sudo xattr -rd com.apple.quarantine $(which today) && codesign -s - --force $(which today)
> ```

### Build from source

Requires **Go 1.24+**.

```bash
git clone https://github.com/jonmilley/today-tui
cd today-tui
make build
```

Or install directly to `~/bin/today`:

```bash
make install
```

## Requirements

- A free [OpenWeatherMap API key](https://openweathermap.org/api) (only if the Weather pane is enabled)
- A GitHub personal access token with `repo` scope, **only if** you choose the GitHub-backed Todo pane. Pick the local JSON backend instead and no token is needed.
- An iCalendar URL or `.ics` file path (only if the Calendar pane is enabled — see [Calendar sources](#calendar-sources))

## First Run

On first launch the setup wizard collects your credentials and saves them to `~/.config/today-tui/config.json`. The wizard walks through:

1. **Todo backend** — `github` (uses GitHub Issues) or `local` (JSON file at `~/.config/today-tui/todos.json`). Local needs no token.
2. **GitHub repo** — `owner/repo` that backs your todo list *(GitHub backend only)*
3. **GitHub token** — personal access token with `repo` scope *(GitHub backend only — [create one here](https://github.com/settings/tokens))*
4. **OpenWeatherMap API key** — free at [openweathermap.org/api](https://openweathermap.org/api) *(new keys activate within 2 hours)*
5. **Weather city** — city name, e.g. `London` or `New York`
6. **Units** — `Imperial` (°F, mph) or `Metric` (°C, kph). Defaults to `Imperial`. `I` / `M` are accepted as shortcuts.
7. **RSS feeds** *(optional)* — one or more comma-separated RSS/Atom URLs. With multiple feeds, items are merged in time order and prefixed with `[host]`.
8. **Calendar URL** *(optional)* — an iCalendar URL or local `.ics` path. See [Calendar sources](#calendar-sources).
9. **Panels** — pick which panes to show on the dashboard.

## Configuration

Config is stored at `~/.config/today-tui/config.json`:

```json
{
  "todo_backend":    "github",
  "github_repo":     "owner/repo",
  "github_token":    "ghp_...",
  "weather_api_key": "...",
  "weather_city":    "London",
  "units":           "Imperial",
  "stocks":          ["SPY", "QQQ", "AAPL", "GOOGL", "META", "AMZN", "NFLX"],
  "rss_feed_urls":   ["https://hnrss.org/frontpage", "https://vocm.com/feed/"],
  "calendar_url":    "https://calendar.google.com/calendar/ical/.../basic.ics",
  "panels": {
    "todo":     true,
    "calendar": true,
    "weather":  true,
    "stocks":   true,
    "stats":    true,
    "news":     true
  }
}
```

Notes:

- `todo_backend` is either `"github"` or `"local"`. The local backend stores tasks at `~/.config/today-tui/todos.json`.
- `units` is `"Imperial"` or `"Metric"`. Older configs with `"F"` / `"C"` are migrated automatically on first launch.
- `rss_feed_urls` is a list. Single-URL configs that used the legacy `"rss_feed_url"` key are migrated automatically on first launch.
- `calendar_url` accepts a URL (Google secret iCal, iCloud, Outlook published, etc.) or an absolute local file path to a `.ics` file.
- Each entry in `panels` controls whether that pane is visible. All panels default to enabled if the `panels` key is absent.

You can edit any setting at runtime by pressing `,`, which opens a menu where text fields are edited inline (Enter), choice fields cycle on Enter, and panel toggles flip with Space.

To re-run the setup wizard at any time:

```bash
./today --reconfigure
```

### Calendar sources

The Calendar pane accepts any iCalendar (`.ics`) source. The most common ones:

- **Google Calendar (private)** — open Google Calendar in a browser → settings for the calendar you want → scroll to **Integrate calendar** → copy the **Secret address in iCal format** (the URL ending in `/basic.ics`). Treat this URL like a password: anyone with it can read your calendar.
- **Google Calendar (public)** — use the **Public address in iCal format** instead.
- **iCloud** — share a calendar publicly, then copy the `webcal://...` URL and paste it as `https://...` (replace `webcal://` with `https://`).
- **Outlook / Office 365** — calendar settings → **Publish a calendar** → copy the ICS link.
- **Local file** — point at any `.ics` file on disk, e.g. `/Users/you/Documents/work.ics`.

Events are fetched once per minute. Recurring events are expanded over a 7-day window starting from "now."

## Key Bindings

### Global

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle focus between panes |
| `,` | Open the runtime config editor |
| `q` / `Ctrl+C` | Quit |
| Mouse click | Focus the clicked pane |
| Mouse wheel | Scroll the focused pane |

### Todo pane

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate items |
| `n` | Create new item (inline form) |
| `c` | Close (complete) selected item |
| `Enter` | Open selected issue in browser *(GitHub backend only)* |
| `r` | Refresh |

### Calendar pane

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll events list |
| `r` | Refresh |

### News pane

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate items |
| `Enter` / `Space` | Toggle inline article preview |
| `Esc` | Close preview |
| `o` | Open article in browser |
| `r` | Refresh feeds |

## Refresh Intervals

| Pane | Interval |
|------|----------|
| System stats | Every 3 seconds |
| Weather, Stocks, News, Calendar | Every 60 seconds (staggered across the window so the four network panes don't all fetch at once) |
| Todo | On demand (`r`) or at startup |

## Data Sources

- **Weather** — [OpenWeatherMap](https://openweathermap.org) current conditions and 5-day forecast APIs
- **Stocks** — [Yahoo Finance](https://finance.yahoo.com) (cookie/crumb authenticated, no paid key required)
- **News** — Any RSS 2.0 or Atom feed via [gofeed](https://github.com/mmcdole/gofeed)
- **Todo** — [GitHub Issues API](https://docs.github.com/en/rest/issues), or a local JSON file
- **Calendar** — Any iCalendar (`.ics`) URL or local file via [golang-ical](https://github.com/arran4/golang-ical)
- **System** — [gopsutil](https://github.com/shirou/gopsutil) (CPU, memory, disk, temperature)
