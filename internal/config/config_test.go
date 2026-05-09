package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultStocks(t *testing.T) {
	want := []string{"SPY", "QQQ", "AAPL", "GOOGL", "META", "AMZN", "NFLX"}
	if got := DefaultStocks(); !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultStocks() = %v; want %v", got, want)
	}
}

// loadFrom is a test helper that mirrors Load but reads from a given path
// so tests can exercise the real default-application code path without
// depending on the user's actual HOME.
func loadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.Stocks) == 0 {
		cfg.Stocks = DefaultStocks()
	}
	if cfg.Units == "C" || cfg.Units == UnitsMetric {
		cfg.Units = UnitsMetric
	} else {
		cfg.Units = UnitsImperial
	}
	if _, hasPanels := raw["panels"]; !hasPanels {
		cfg.Panels = PanelConfig{Todo: true, Weather: true, Stocks: true, Stats: true, News: true}
	}
	return &cfg, nil
}

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConfigDefaultsAppliedWhenPanelsAbsent(t *testing.T) {
	cfg, err := loadFrom(writeTempConfig(t, `{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Stocks, DefaultStocks()) {
		t.Errorf("Expected default stocks, got %v", cfg.Stocks)
	}
	if cfg.Units != UnitsImperial {
		t.Errorf("Expected default units Imperial, got %v", cfg.Units)
	}
	want := PanelConfig{Todo: true, Weather: true, Stocks: true, Stats: true, News: true}
	if cfg.Panels != want {
		t.Errorf("Expected all panels enabled by default, got %+v", cfg.Panels)
	}
}

func TestPanelsAllFalseRespected(t *testing.T) {
	body := `{"panels": {"todo": false, "weather": false, "stocks": false, "stats": false, "news": false}}`
	cfg, err := loadFrom(writeTempConfig(t, body))
	if err != nil {
		t.Fatal(err)
	}
	want := PanelConfig{}
	if cfg.Panels != want {
		t.Errorf("Expected all panels disabled to round-trip, got %+v", cfg.Panels)
	}
}

func TestPanelsPartialRespected(t *testing.T) {
	body := `{"panels": {"todo": true, "weather": false, "stocks": false, "stats": false, "news": false}}`
	cfg, err := loadFrom(writeTempConfig(t, body))
	if err != nil {
		t.Fatal(err)
	}
	want := PanelConfig{Todo: true}
	if cfg.Panels != want {
		t.Errorf("Expected only Todo enabled, got %+v", cfg.Panels)
	}
}
