package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PanelConfig struct {
	Todo    bool `json:"todo"`
	Weather bool `json:"weather"`
	Stocks  bool `json:"stocks"`
	Stats   bool `json:"stats"`
	News    bool `json:"news"`
}

type Config struct {
	GitHubRepo    string      `json:"github_repo"`
	GitHubToken   string      `json:"github_token"`
	TodoBackend   string      `json:"todo_backend"` // "github" or "local"
	WeatherAPIKey string      `json:"weather_api_key"`
	WeatherCity   string      `json:"weather_city"`
	Units         string      `json:"units"` // "F" or "C"
	Stocks        []string    `json:"stocks"`
	RSSFeedURL    string      `json:"rss_feed_url"`
	Panels        PanelConfig `json:"panels"`
}

func DefaultStocks() []string {
	return []string{"SPY", "QQQ", "AAPL", "GOOGL", "META", "AMZN", "NFLX"}
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "today-tui", "config.json"), nil
}

func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil // needs setup
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.Stocks) == 0 {
		cfg.Stocks = DefaultStocks()
	}
	if cfg.Units != "C" {
		cfg.Units = "F" // default to Fahrenheit
	}
	// Default all panels to enabled if none are set
	if !cfg.Panels.Todo && !cfg.Panels.Weather && !cfg.Panels.Stocks && !cfg.Panels.Stats && !cfg.Panels.News {
		cfg.Panels = PanelConfig{Todo: true, Weather: true, Stocks: true, Stats: true, News: true}
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o700); mkErr != nil {
		return mkErr
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
