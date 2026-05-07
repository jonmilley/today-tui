package api

import (
	"testing"
	"time"
)

func TestDegToDir(t *testing.T) {
	tests := []struct {
		deg  float64
		want string
	}{
		{0, "N"},
		{22.4, "N"},
		{22.5, "NE"},
		{45, "NE"},
		{67.4, "NE"},
		{67.5, "E"},
		{90, "E"},
		{112.5, "SE"},
		{135, "SE"},
		{157.5, "S"},
		{180, "S"},
		{202.5, "SW"},
		{225, "SW"},
		{247.5, "W"},
		{270, "W"},
		{292.5, "NW"},
		{315, "NW"},
		{337.4, "NW"},
		{337.5, "N"},
		{359.9, "N"},
		{382.5, "NE"}, // wrap around
	}

	for _, tt := range tests {
		if got := degToDir(tt.deg); got != tt.want {
			t.Errorf("degToDir(%v) = %v; want %v", tt.deg, got, tt.want)
		}
	}
}

func TestCToF(t *testing.T) {
	tests := []struct {
		c    float64
		want float64
	}{
		{0, 32},
		{100, 212},
		{-40, -40},
		{20, 68},
	}

	for _, tt := range tests {
		if got := cToF(tt.c); got != tt.want {
			t.Errorf("cToF(%v) = %v; want %v", tt.c, got, tt.want)
		}
	}
}

func TestWeatherIcon(t *testing.T) {
	tests := []struct {
		desc string
		want string
	}{
		{"clear sky", "  \\O/  "},
		{"few clouds", "  ~~~  "},
		{"scattered clouds", "  ~~~  "},
		{"broken clouds", "  ~~~  "},
		{"overcast clouds", "  ~~~  "},
		{"light rain", "  ///  "},
		{"moderate rain", "  ///  "},
		{"heavy intensity rain", "  ///  "},
		{"drizzle", "  ///  "},
		{"light snow", "  ***  "},
		{"snow", "  ***  "},
		{"thunderstorm", "  !!!  "},
		{"mist", "  ---  "},
		{"fog", "  ---  "},
	}

	for _, tt := range tests {
		if got := weatherIcon(tt.desc); got != tt.want {
			t.Errorf("weatherIcon(%v) = %v; want %v", tt.desc, got, tt.want)
		}
	}
}

// forecastEntry is shorthand for building one of the inner List entries
// in owmForecastResponse for testing.
func forecastEntry(t time.Time, minC, maxC float64, desc string, pop float64) struct {
	Dt   int64 `json:"dt"`
	Main struct {
		TempMin float64 `json:"temp_min"`
		TempMax float64 `json:"temp_max"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
	Pop float64 `json:"pop"`
} {
	var entry struct {
		Dt   int64 `json:"dt"`
		Main struct {
			TempMin float64 `json:"temp_min"`
			TempMax float64 `json:"temp_max"`
		} `json:"main"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Pop float64 `json:"pop"`
	}
	entry.Dt = t.Unix()
	entry.Main.TempMin = minC
	entry.Main.TempMax = maxC
	if desc != "" {
		entry.Weather = append(entry.Weather, struct {
			Description string `json:"description"`
		}{Description: desc})
	}
	entry.Pop = pop
	return entry
}

func TestParseForecastPicksTomorrowWhenAvailable(t *testing.T) {
	utc := time.FixedZone("utc0", 0)
	now := time.Date(2026, 5, 7, 9, 0, 0, 0, utc) // Thu 09:00 UTC

	r := owmForecastResponse{}
	r.City.Timezone = 0
	// Thursday entries (today): should be ignored
	r.List = append(r.List,
		forecastEntry(time.Date(2026, 5, 7, 12, 0, 0, 0, utc), 18, 22, "clear sky", 0.0),
		forecastEntry(time.Date(2026, 5, 7, 21, 0, 0, 0, utc), 16, 20, "clear sky", 0.0),
	)
	// Friday entries (tomorrow): should aggregate
	r.List = append(r.List,
		forecastEntry(time.Date(2026, 5, 8, 6, 0, 0, 0, utc), 12, 14, "light rain", 0.4),
		forecastEntry(time.Date(2026, 5, 8, 12, 0, 0, 0, utc), 16, 21, "light rain", 0.6),
		forecastEntry(time.Date(2026, 5, 8, 18, 0, 0, 0, utc), 14, 18, "broken clouds", 0.2),
	)

	got := parseForecast(r, now)
	if got == nil {
		t.Fatal("expected forecast for Friday, got nil")
	}
	if got.TempMinC != 12 || got.TempMaxC != 21 {
		t.Errorf("min/max = %v/%v; want 12/21", got.TempMinC, got.TempMaxC)
	}
	if got.Desc != "light rain" {
		t.Errorf("desc = %q; want %q (most frequent)", got.Desc, "light rain")
	}
	if got.PrecipPct != 60 {
		t.Errorf("precip = %d; want 60 (max pop)", got.PrecipPct)
	}
}

func TestParseForecastFallsBackWhenTomorrowEmpty(t *testing.T) {
	utc := time.FixedZone("utc0", 0)
	now := time.Date(2026, 5, 7, 23, 30, 0, 0, utc) // late Thu UTC

	r := owmForecastResponse{}
	r.City.Timezone = 0
	// Today only — no Friday entries at all (simulating end of window).
	r.List = append(r.List,
		forecastEntry(time.Date(2026, 5, 7, 23, 0, 0, 0, utc), 8, 10, "clear sky", 0.0),
	)
	// Saturday entries — the fallback target.
	r.List = append(r.List,
		forecastEntry(time.Date(2026, 5, 9, 9, 0, 0, 0, utc), 14, 18, "few clouds", 0.1),
		forecastEntry(time.Date(2026, 5, 9, 15, 0, 0, 0, utc), 17, 22, "few clouds", 0.1),
	)

	got := parseForecast(r, now)
	if got == nil {
		t.Fatal("expected fallback to Saturday, got nil")
	}
	if got.TempMinC != 14 || got.TempMaxC != 22 {
		t.Errorf("min/max = %v/%v; want 14/22 (Saturday)", got.TempMinC, got.TempMaxC)
	}
}

func TestParseForecastReturnsNilWhenNoFutureEntries(t *testing.T) {
	utc := time.FixedZone("utc0", 0)
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, utc)

	r := owmForecastResponse{}
	r.City.Timezone = 0
	r.List = append(r.List,
		forecastEntry(time.Date(2026, 5, 7, 18, 0, 0, 0, utc), 18, 22, "clear sky", 0.0),
	)

	if got := parseForecast(r, now); got != nil {
		t.Errorf("expected nil for today-only entries, got %+v", got)
	}
}

func TestParseForecastUsesCityTimezoneForDateBoundary(t *testing.T) {
	// Tokyo is UTC+9. At 16:00 UTC on May 7, Tokyo wall clock is 01:00 May 8.
	// Entries timestamped during May 8 UTC (12:00, 18:00 UTC = 21:00, 03:00
	// Tokyo time on May 8/9) should map correctly: 12:00 UTC is May 8 21:00
	// in Tokyo (still May 8); 18:00 UTC is May 9 03:00 Tokyo (May 9).
	tokyoOffset := 9 * 3600
	utc := time.FixedZone("utc0", 0)
	now := time.Date(2026, 5, 7, 16, 0, 0, 0, utc) // Tokyo: May 8 01:00

	r := owmForecastResponse{}
	r.City.Timezone = tokyoOffset
	r.List = append(r.List,
		// Tokyo-local May 8 (today in Tokyo): should be skipped
		forecastEntry(time.Date(2026, 5, 8, 3, 0, 0, 0, utc), 5, 5, "clear sky", 0),
		// Tokyo-local May 9 (tomorrow in Tokyo): should be picked
		forecastEntry(time.Date(2026, 5, 8, 21, 0, 0, 0, utc), 11, 15, "light rain", 0.5),
	)

	got := parseForecast(r, now)
	if got == nil {
		t.Fatal("expected forecast for Tokyo-local tomorrow, got nil")
	}
	if got.TempMinC != 11 || got.TempMaxC != 15 {
		t.Errorf("min/max = %v/%v; want 11/15 (Tokyo May 9)", got.TempMinC, got.TempMaxC)
	}
}
