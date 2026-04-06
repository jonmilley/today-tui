package api

import (
	"testing"
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
