package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type WeatherData struct {
	City     string
	Country  string
	TempF    float64
	TempC    float64
	FeelsF   float64
	Desc     string
	Icon     string
	Humidity int
	WindMph  float64
	WindDir  string
}

// ForecastDay holds an aggregated daily forecast (derived from 3-hour blocks).
type ForecastDay struct {
	TempMinC  float64
	TempMaxC  float64
	TempMinF  float64
	TempMaxF  float64
	Desc      string
	PrecipPct int // 0–100
}

type Weather interface {
	FetchWeather(city string) (*WeatherData, error)
	FetchForecast(city string) (*ForecastDay, error)
}

type WeatherClient struct {
	apiKey string
	client *http.Client
}

var _ Weather = (*WeatherClient)(nil)

func NewWeatherClient(apiKey string) *WeatherClient {
	return &WeatherClient{
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
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

type owmForecastResponse struct {
	City struct {
		Timezone int `json:"timezone"` // UTC offset in seconds
	} `json:"city"`
	List []struct {
		Dt   int64 `json:"dt"`
		Main struct {
			TempMin float64 `json:"temp_min"`
			TempMax float64 `json:"temp_max"`
		} `json:"main"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Pop float64 `json:"pop"` // probability of precipitation, 0–1
	} `json:"list"`
}

func (c *WeatherClient) FetchForecast(city string) (*ForecastDay, error) {
	endpoint := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/forecast?q=%s&appid=%s&units=metric",
		url.QueryEscape(city), c.apiKey,
	)
	resp, err := c.client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecast API: %s", resp.Status)
	}

	var r owmForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return parseTomorrow(r), nil
}

func (c *WeatherClient) FetchWeather(city string) (*WeatherData, error) {
	endpoint := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric",
		url.QueryEscape(city), c.apiKey,
	)
	resp, err := c.client.Get(endpoint)
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

func degToDir(deg float64) string {
	dirs := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	idx := int((deg+22.5)/45) % 8
	return dirs[idx]
}

func cToF(c float64) float64 { return c*9/5 + 32 }

func weatherIcon(desc string) string {
	switch {
	case strings.Contains(desc, "clear"):
		return "  \\O/  "
	case strings.Contains(desc, "cloud"):
		return "  ~~~  "
	case strings.Contains(desc, "rain"), strings.Contains(desc, "drizzle"):
		return "  ///  "
	case strings.Contains(desc, "snow"):
		return "  ***  "
	case strings.Contains(desc, "thunder"):
		return "  !!!  "
	default:
		return "  ---  "
	}
}

// parseTomorrow aggregates 3-hour forecast blocks for the next calendar day
// using the city's UTC offset so the date boundary is local time.
func parseTomorrow(r owmForecastResponse) *ForecastDay {
	loc := time.FixedZone("city", r.City.Timezone)
	tomorrow := time.Now().In(loc).AddDate(0, 0, 1).Format("2006-01-02")

	minC := 999.0
	maxC := -999.0
	descCount := map[string]int{}
	maxPop := 0.0

	for _, entry := range r.List {
		if time.Unix(entry.Dt, 0).In(loc).Format("2006-01-02") != tomorrow {
			continue
		}
		if entry.Main.TempMin < minC {
			minC = entry.Main.TempMin
		}
		if entry.Main.TempMax > maxC {
			maxC = entry.Main.TempMax
		}
		if len(entry.Weather) > 0 {
			descCount[entry.Weather[0].Description]++
		}
		if entry.Pop > maxPop {
			maxPop = entry.Pop
		}
	}

	if minC == 999.0 {
		return nil // no data for tomorrow (e.g. near end of forecast window)
	}

	// Pick the most frequently occurring description.
	desc := ""
	best := 0
	for d, n := range descCount {
		if n > best {
			best, desc = n, d
		}
	}

	return &ForecastDay{
		TempMinC:  minC,
		TempMaxC:  maxC,
		TempMinF:  cToF(minC),
		TempMaxF:  cToF(maxC),
		Desc:      desc,
		PrecipPct: int(maxPop * 100),
	}
}
