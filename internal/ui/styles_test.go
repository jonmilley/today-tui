package ui

import (
	"testing"

	"github.com/mattn/go-runewidth"
)

const (
	helloFits   = "hello"
	saoPaulo    = "São Paulo"
	cjkTokyo    = "東京都"
	cjkOverflow = "東京都市計画"
)

func TestTruncateWidth(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		max   int
		want  string
		wantW int
	}{
		{"ascii fits", helloFits, 10, helloFits, 5},
		{"ascii truncated with ellipsis", "abcdefghij", 6, "abc...", 6},
		{"ascii max <=3 no ellipsis", "abcdef", 2, "ab", 2},
		{"max zero returns empty", "anything", 0, "", 0},
		{"multi-byte rune fits", saoPaulo, 10, saoPaulo, 9},
		{"multi-byte rune truncated does not split mid-rune", saoPaulo, 5, "Sã...", 5},
		{"double-width CJK fits", cjkTokyo, 6, cjkTokyo, 6},
		{"double-width CJK truncated respects cell width", cjkOverflow, 6, "東...", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q; want %q", tt.s, tt.max, got, tt.want)
			}
			if w := runewidth.StringWidth(got); w > tt.max && tt.max > 0 {
				t.Errorf("truncate(%q, %d) width %d > max", tt.s, tt.max, w)
			}
		})
	}
}

func TestPadRightWidth(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"ascii needs padding", "hi", 5, "hi   "},
		{"ascii already at width", helloFits, 5, helloFits},
		{"ascii longer than width", "toolong", 4, "toolong"},
		{"multi-byte rune padded by visible width not bytes", "São", 5, "São  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padRight(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("padRight(%q, %d) = %q; want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}
