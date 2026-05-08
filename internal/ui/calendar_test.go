package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonmilley/today-tui/internal/api"
)

// stripANSI removes lipgloss/ANSI color escape sequences so test assertions
// can compare against the visible characters of the mini calendar.
func stripANSI(s string) string {
	var out strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			in = true
		case in && r == 'm':
			in = false
		case !in:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func TestRenderMonthMiniLayoutMay2026(t *testing.T) {
	// May 1 2026 is a Friday → 5 leading blank cells before "1".
	utc := time.FixedZone("utc0", 0)
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, utc) // Today: Thu May 7
	p := calendarPane{
		events: []api.Event{
			// One event on May 8 → that day should be marked.
			{Start: time.Date(2026, 5, 8, 9, 0, 0, 0, utc), Summary: "Lunch"},
			// One event in a different month — must NOT mark anything.
			{Start: time.Date(2026, 6, 3, 9, 0, 0, 0, utc), Summary: "Other"},
		},
	}

	got := stripANSI(p.renderMonthMini(now))
	lines := strings.Split(got, "\n")

	if len(lines) != 8 {
		t.Fatalf("want 8 lines (header + weekday + 6 date rows), got %d:\n%s", len(lines), got)
	}

	// Header row: month name + year, centered in 21-col field with 2-space indent.
	if !strings.Contains(lines[0], "May 2026") {
		t.Errorf("header line %q does not contain %q", lines[0], "May 2026")
	}
	// Weekday row.
	if want := "  Su Mo Tu We Th Fr Sa"; lines[1] != want {
		t.Errorf("weekday row = %q; want %q", lines[1], want)
	}
	// First date row: 2-char indent + 5 blank cells joined by spaces +
	// " 1" + " " + " 2" → 18 visible spaces before the "1".
	if want := "                  1  2"; lines[2] != want {
		t.Errorf("first date row = %q; want %q", lines[2], want)
	}
	// Second date row: 3, 4, 5, 6, 7 (today), 8 (event), 9.
	if want := "   3  4  5  6  7  8  9"; lines[3] != want {
		t.Errorf("second date row = %q; want %q", lines[3], want)
	}
	// Last visible row contains 31 alone.
	if !strings.Contains(lines[7], "31") {
		t.Errorf("last row should contain day 31, got %q", lines[7])
	}
}

func TestCollectEventDaysFiltersByMonth(t *testing.T) {
	utc := time.FixedZone("utc0", 0)
	p := calendarPane{
		events: []api.Event{
			{Start: time.Date(2026, 5, 7, 9, 0, 0, 0, utc)},
			{Start: time.Date(2026, 5, 7, 14, 0, 0, 0, utc)}, // same day, different time
			{Start: time.Date(2026, 5, 14, 9, 0, 0, 0, utc)},
			{Start: time.Date(2026, 6, 1, 9, 0, 0, 0, utc)}, // different month
			{Start: time.Date(2025, 5, 7, 9, 0, 0, 0, utc)}, // different year
		},
	}
	got := p.collectEventDays(2026, time.May)
	if len(got) != 2 || !got[7] || !got[14] {
		t.Errorf("collectEventDays = %v; want {7:true, 14:true}", got)
	}
}

func TestShouldShowMiniThreshold(t *testing.T) {
	tests := []struct {
		w, h int
		want bool
	}{
		{40, 30, true},
		{25, 14, true},
		{24, 30, false}, // too narrow
		{40, 13, false}, // too short
		{0, 0, false},
	}
	for _, tt := range tests {
		p := calendarPane{width: tt.w, height: tt.h}
		if got := p.shouldShowMini(); got != tt.want {
			t.Errorf("shouldShowMini(%dx%d) = %v; want %v", tt.w, tt.h, got, tt.want)
		}
	}
}

func TestRenderMonthMiniIncludesAllDaysOfMonth(t *testing.T) {
	// Pick a 31-day month and verify every day-of-month appears in the
	// stripped output. (Lipgloss may suppress ANSI styling in non-TTY
	// test environments, so we can't reliably assert escape sequences;
	// content correctness is enough.)
	utc := time.FixedZone("utc0", 0)
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, utc)
	got := stripANSI(calendarPane{}.renderMonthMini(now))
	for d := 1; d <= 31; d++ {
		// Match " 1" (with leading space) — works for 1-9 and 10-31.
		needle := miniCellPlain(d)
		if !strings.Contains(got, needle) {
			t.Errorf("day %d not found in stripped output:\n%s", d, got)
		}
	}
}

// miniCellPlain returns the unstyled 2-char cell for day d (matching the
// %2d formatting in miniCell).
func miniCellPlain(d int) string {
	return fmt.Sprintf("%2d", d)
}
