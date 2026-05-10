package ui

import (
	"reflect"
	"testing"

	"github.com/jonmilley/today-tui/internal/config"
)

func TestParseStocksList(t *testing.T) {
	const spy, qqq = "SPY", "QQQ"
	tests := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{spy, []string{spy}},
		{"spy, qqq, aapl", []string{spy, qqq, "AAPL"}},
		{"  SPY ,, QQQ ", []string{spy, qqq}},
		{"SPY,SPY,SPY", []string{spy, spy, spy}}, // dedup not done by design
	}
	for _, tt := range tests {
		got := parseStocksList(tt.in)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseStocksList(%q) = %v; want %v", tt.in, got, tt.want)
		}
	}
}

func TestNextChoiceWraps(t *testing.T) {
	choices := []string{todoBackendGitHub, todoBackendLocal}
	tests := []struct {
		cur, want string
	}{
		{todoBackendGitHub, todoBackendLocal},
		{todoBackendLocal, todoBackendGitHub},
		{"", todoBackendGitHub},        // unknown → first
		{"unknown", todoBackendGitHub}, // unknown → first
	}
	for _, tt := range tests {
		if got := nextChoice(choices, tt.cur); got != tt.want {
			t.Errorf("nextChoice(%v, %q) = %q; want %q", choices, tt.cur, got, tt.want)
		}
	}
}

func TestGetSetStringFieldRoundTrip(t *testing.T) {
	cfg := config.Config{
		TodoBackend:   todoBackendGitHub,
		GitHubRepo:    "acme/tasks",
		GitHubToken:   "ghp_xxx",
		CalendarURL:   "https://example.com/cal.ics",
		WeatherAPIKey: "abc",
		WeatherCity:   "London",
		Units:         config.UnitsImperial,
		Stocks:        []string{"SPY", "QQQ"},
		RSSFeedURLs:   []string{"https://example.com/feed", "https://other.com/rss"},
	}

	tests := []struct {
		key  string
		want string
	}{
		{keyTodoBackend, todoBackendGitHub},
		{keyGitHubRepo, "acme/tasks"},
		{keyGitHubToken, "ghp_xxx"},
		{keyCalendarURL, "https://example.com/cal.ics"},
		{keyWeatherKey, "abc"},
		{keyWeatherCity, "London"},
		{keyUnits, config.UnitsImperial},
		{keyStocks, "SPY, QQQ"},
		{keyRSSURL, "https://example.com/feed, https://other.com/rss"},
	}
	for _, tt := range tests {
		if got := getStringField(&cfg, tt.key); got != tt.want {
			t.Errorf("getStringField(%q) = %q; want %q", tt.key, got, tt.want)
		}
	}

	// Mutate via setStringField, then re-read.
	setStringField(&cfg, keyWeatherCity, "  Tokyo  ")
	if cfg.WeatherCity != "Tokyo" {
		t.Errorf("setStringField trimming failed: WeatherCity = %q", cfg.WeatherCity)
	}
	setStringField(&cfg, keyUnits, "c")
	if cfg.Units != config.UnitsMetric {
		t.Errorf("setStringField unit normalization failed: %q", cfg.Units)
	}
	setStringField(&cfg, keyStocks, "msft, googl, aapl")
	if !reflect.DeepEqual(cfg.Stocks, []string{"MSFT", "GOOGL", "AAPL"}) {
		t.Errorf("setStringField stocks parsing failed: %v", cfg.Stocks)
	}
}

func TestEditorCursorSkipsHeaders(t *testing.T) {
	e := newConfigEditor(config.Config{})
	// Initial cursor should be on the first non-header field.
	if configFields[e.cursor].kind == fieldHeader {
		t.Fatalf("initial cursor landed on a header at idx %d", e.cursor)
	}
	// Walking forward should never land on a header.
	for step := 0; step < len(configFields)*2; step++ {
		e.cursor = e.nextNavIndex(1)
		if configFields[e.cursor].kind == fieldHeader {
			t.Fatalf("nextNavIndex(+1) landed on header at idx %d", e.cursor)
		}
	}
	// Walking backward should also skip headers.
	for step := 0; step < len(configFields)*2; step++ {
		e.cursor = e.nextNavIndex(-1)
		if configFields[e.cursor].kind == fieldHeader {
			t.Fatalf("nextNavIndex(-1) landed on header at idx %d", e.cursor)
		}
	}
}
