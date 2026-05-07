package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	vEventBegin = "BEGIN:VEVENT"
	vEventEnd   = "END:VEVENT"
)

// minimalICS returns a syntactically-valid iCalendar payload with the
// specified body inserted between BEGIN:VCALENDAR and END:VCALENDAR.
func minimalICS(body string) string {
	return strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//test//EN",
		body,
		"END:VCALENDAR",
		"",
	}, "\r\n")
}

func TestICSClientReadsLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ics")
	now := time.Now().UTC()
	soon := now.Add(2 * time.Hour).UTC()
	body := strings.Join([]string{
		vEventBegin,
		"UID:abc@today-tui",
		"DTSTART:" + soon.Format("20060102T150405Z"),
		"DTEND:" + soon.Add(time.Hour).Format("20060102T150405Z"),
		"SUMMARY:Lunch with Pat",
		"LOCATION:Cafe",
		vEventEnd,
	}, "\r\n")
	if err := os.WriteFile(path, []byte(minimalICS(body)), 0o600); err != nil {
		t.Fatal(err)
	}

	c := NewICSClient(path)
	events, err := c.FetchEvents(24 * time.Hour)
	if err != nil {
		t.Fatalf("FetchEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	if events[0].Summary != "Lunch with Pat" {
		t.Errorf("summary = %q; want %q", events[0].Summary, "Lunch with Pat")
	}
	if events[0].Location != "Cafe" {
		t.Errorf("location = %q; want %q", events[0].Location, "Cafe")
	}
	if events[0].AllDay {
		t.Errorf("AllDay = true; want false (timed event)")
	}
}

func TestICSClientReadsHTTPSource(t *testing.T) {
	soon := time.Now().UTC().Add(time.Hour)
	body := strings.Join([]string{
		vEventBegin,
		"UID:http@today-tui",
		"DTSTART:" + soon.Format("20060102T150405Z"),
		"DTEND:" + soon.Add(30*time.Minute).Format("20060102T150405Z"),
		"SUMMARY:Standup",
		vEventEnd,
	}, "\r\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		_, _ = w.Write([]byte(minimalICS(body)))
	}))
	defer srv.Close()

	c := NewICSClient(srv.URL)
	events, err := c.FetchEvents(24 * time.Hour)
	if err != nil {
		t.Fatalf("FetchEvents: %v", err)
	}
	if len(events) != 1 || events[0].Summary != "Standup" {
		t.Fatalf("got %+v; want one Standup event", events)
	}
}

func TestICSClientFiltersPastAndFuture(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-3 * time.Hour)
	tooFar := now.Add(48 * time.Hour) // outside a 24h window
	soon := now.Add(2 * time.Hour)
	mkEvent := func(uid string, start time.Time, summary string) string {
		return strings.Join([]string{
			vEventBegin,
			"UID:" + uid,
			"DTSTART:" + start.Format("20060102T150405Z"),
			"DTEND:" + start.Add(time.Hour).Format("20060102T150405Z"),
			"SUMMARY:" + summary,
			vEventEnd,
		}, "\r\n")
	}
	body := strings.Join([]string{
		mkEvent("p@x", past, "Yesterday"),
		mkEvent("s@x", soon, "Today soon"),
		mkEvent("f@x", tooFar, "Day after"),
	}, "\r\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "f.ics")
	if err := os.WriteFile(path, []byte(minimalICS(body)), 0o600); err != nil {
		t.Fatal(err)
	}

	events, err := NewICSClient(path).FetchEvents(24 * time.Hour)
	if err != nil {
		t.Fatalf("FetchEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 event in window, got %d: %+v", len(events), events)
	}
	if events[0].Summary != "Today soon" {
		t.Errorf("summary = %q; want %q", events[0].Summary, "Today soon")
	}
}

func TestICSClientExpandsRecurringEvents(t *testing.T) {
	// Daily standup starting yesterday, recurring forever.
	yesterday := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour).Add(15 * time.Hour)
	body := strings.Join([]string{
		vEventBegin,
		"UID:r@today-tui",
		"DTSTART:" + yesterday.Format("20060102T150405Z"),
		"DTEND:" + yesterday.Add(30*time.Minute).Format("20060102T150405Z"),
		"RRULE:FREQ=DAILY",
		"SUMMARY:Daily Standup",
		vEventEnd,
	}, "\r\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "r.ics")
	if err := os.WriteFile(path, []byte(minimalICS(body)), 0o600); err != nil {
		t.Fatal(err)
	}

	// 7-day window should yield ~7-8 occurrences (today + 7 future-ish).
	events, err := NewICSClient(path).FetchEvents(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("FetchEvents: %v", err)
	}
	if len(events) < 5 {
		t.Errorf("expected at least 5 daily-standup occurrences in 7d window, got %d", len(events))
	}
	// All occurrences should share the summary.
	for _, ev := range events {
		if ev.Summary != "Daily Standup" {
			t.Errorf("unexpected summary %q in expansion", ev.Summary)
			break
		}
	}
}

func TestICSClientHandlesAllDayEvent(t *testing.T) {
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("20060102")
	body := strings.Join([]string{
		vEventBegin,
		"UID:ad@today-tui",
		"DTSTART;VALUE=DATE:" + tomorrow,
		"SUMMARY:Holiday",
		vEventEnd,
	}, "\r\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "ad.ics")
	if err := os.WriteFile(path, []byte(minimalICS(body)), 0o600); err != nil {
		t.Fatal(err)
	}

	events, err := NewICSClient(path).FetchEvents(48 * time.Hour)
	if err != nil {
		t.Fatalf("FetchEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 all-day event, got %d", len(events))
	}
	if !events[0].AllDay {
		t.Errorf("AllDay = false; want true (DTSTART;VALUE=DATE)")
	}
}

func TestICSClientEmptySource(t *testing.T) {
	if _, err := NewICSClient("").FetchEvents(time.Hour); err == nil {
		t.Error("expected error for empty source")
	}
}
