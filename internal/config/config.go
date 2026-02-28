package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	GitHubRepo    string   `json:"github_repo"`
	GitHubToken   string   `json:"github_token"`
	WeatherAPIKey string   `json:"weather_api_key"`
	WeatherCity   string   `json:"weather_city"`
	Stocks        []string `json:"stocks"`
	RSSFeedURL    string   `json:"rss_feed_url"`
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
	return &cfg, nil
}

func (c *Config) Save() error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}
