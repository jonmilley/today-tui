package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type WeatherData struct {
	City    string
	Country string
	TempF   float64
	TempC   float64
	FeelsF  float64
	Desc    string
	Icon    string
	Humidity int
	WindMph float64
	WindDir string
}

type owmResponse struct {
	Name string `json:"name"`
	Sys  struct {
		Country string `json:"country"`
	} `json:"sys"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   float64 `json:"deg"`
	} `json:"wind"`
}

func degToDir(deg float64) string {
	dirs := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	idx := int((deg+22.5)/45) % 8
	return dirs[idx]
}

func cToF(c float64) float64 { return c*9/5 + 32 }

func weatherIcon(desc string) string {
	switch {
	case contains(desc, "clear"):
		return "  \\O/  "
	case contains(desc, "cloud"):
		return "  ~~~  "
	case contains(desc, "rain"), contains(desc, "drizzle"):
		return "  ///  "
	case contains(desc, "snow"):
		return "  ***  "
	case contains(desc, "thunder"):
		return "  !!!  "
	default:
		return "  ---  "
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func FetchWeather(apiKey, city string) (*WeatherData, error) {
	endpoint := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric",
		url.QueryEscape(city), apiKey,
	)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key (new keys take up to 2h to activate)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("city not found: %q", city)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API: %s", resp.Status)
	}

	var r owmResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	desc := ""
	if len(r.Weather) > 0 {
		desc = r.Weather[0].Description
	}

	return &WeatherData{
		City:     r.Name,
		Country:  r.Sys.Country,
		TempC:    r.Main.Temp,
		TempF:    cToF(r.Main.Temp),
		FeelsF:   cToF(r.Main.FeelsLike),
		Desc:     desc,
		Icon:     weatherIcon(desc),
		Humidity: r.Main.Humidity,
		WindMph:  r.Wind.Speed * 2.237, // m/s to mph
		WindDir:  degToDir(r.Wind.Deg),
	}, nil
}
