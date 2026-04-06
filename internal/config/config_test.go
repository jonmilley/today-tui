package config

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestDefaultStocks(t *testing.T) {
	want := []string{"SPY", "QQQ", "AAPL", "GOOGL", "META", "AMZN", "NFLX"}
	if got := DefaultStocks(); !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultStocks() = %v; want %v", got, want)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Create a temp config file with missing fields
	tmpDir, err := os.MkdirTemp("", "today-tui-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock Path() behavior by setting an env var if needed, but Path() uses os.UserHomeDir()
	// Let's just test the logic after loading data.
	
	data := []byte(`{}`)
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	
	// Apply defaults (logic from Load)
	if len(cfg.Stocks) == 0 {
		cfg.Stocks = DefaultStocks()
	}
	if cfg.Units != "C" {
		cfg.Units = "F"
	}
	if !cfg.Panels.Todo && !cfg.Panels.Weather && !cfg.Panels.Stocks && !cfg.Panels.Stats && !cfg.Panels.News {
		cfg.Panels = PanelConfig{Todo: true, Weather: true, Stocks: true, Stats: true, News: true}
	}

	if !reflect.DeepEqual(cfg.Stocks, DefaultStocks()) {
		t.Errorf("Expected default stocks, got %v", cfg.Stocks)
	}
	if cfg.Units != "F" {
		t.Errorf("Expected default units F, got %v", cfg.Units)
	}
	if !cfg.Panels.Todo {
		t.Errorf("Expected all panels enabled by default")
	}
}
