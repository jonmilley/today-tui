package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type WeatherData struct {
	City     string
	Country  string
	TempF    float64
	TempC    float64
	FeelsF   float64
	FeelsC   float64
	Desc     string
	Icon     string
	Humidity int
	WindMph  float64
	WindKph  float64
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

	return parseForecast(r, time.Now()), nil
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
		FeelsC:   r.Main.FeelsLike,
		FeelsF:   cToF(r.Main.FeelsLike),
		Desc:     desc,
		Icon:     weatherIcon(desc),
		Humidity: r.Main.Humidity,
		WindMph:  r.Wind.Speed * 2.237, // m/s to mph
		WindKph:  r.Wind.Speed * 3.6,   // m/s to kph
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

// dayBucket aggregates the 3-hour forecast entries that fall on a single
// calendar day in the city's local timezone.
type dayBucket struct {
	minC, maxC float64
	descCount  map[string]int
	maxPop     float64
}

// parseForecast picks the earliest calendar day strictly after `now` (in the
// city's timezone) that has at least one forecast block, and aggregates its
// 3-hour entries into a single ForecastDay. Usually that's tomorrow; if the
// API returns no entries for tomorrow (very late in the forecast window, or
// if blocks shifted), we fall back to the next available day rather than
// showing "no forecast data". Returns nil only when every entry is dated
// today or earlier in the city's local time.
//
// `now` is injected rather than read from time.Now() so callers can pin it
// in tests.
func parseForecast(r owmForecastResponse, now time.Time) *ForecastDay {
	loc := time.FixedZone("city", r.City.Timezone)
	today := now.In(loc).Format("2006-01-02")

	byDate := map[string]*dayBucket{}
	var dates []string

	for _, entry := range r.List {
		date := time.Unix(entry.Dt, 0).In(loc).Format("2006-01-02")
		if date <= today {
			continue
		}
		b, ok := byDate[date]
		if !ok {
			b = &dayBucket{
				minC:      math.Inf(1),
				maxC:      math.Inf(-1),
				descCount: map[string]int{},
			}
			byDate[date] = b
			dates = append(dates, date)
		}
		if entry.Main.TempMin < b.minC {
			b.minC = entry.Main.TempMin
		}
		if entry.Main.TempMax > b.maxC {
			b.maxC = entry.Main.TempMax
		}
		if len(entry.Weather) > 0 {
			b.descCount[entry.Weather[0].Description]++
		}
		if entry.Pop > b.maxPop {
			b.maxPop = entry.Pop
		}
	}

	if len(dates) == 0 {
		return nil
	}
	// YYYY-MM-DD strings sort chronologically. Defensive sort in case the
	// API ever returns entries out of order.
	sort.Strings(dates)
	b := byDate[dates[0]]

	desc := ""
	best := 0
	for d, n := range b.descCount {
		if n > best {
			best, desc = n, d
		}
	}

	return &ForecastDay{
		TempMinC:  b.minC,
		TempMaxC:  b.maxC,
		TempMinF:  cToF(b.minC),
		TempMaxF:  cToF(b.maxC),
		Desc:      desc,
		PrecipPct: int(b.maxPop * 100),
	}
}
