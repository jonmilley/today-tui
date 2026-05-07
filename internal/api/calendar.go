package api

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/teambition/rrule-go"
)

type Event struct {
	Summary  string
	Location string
	Start    time.Time
	End      time.Time
	AllDay   bool
}

type Calendar interface {
	FetchEvents(window time.Duration) ([]Event, error)
}

// ICSClient reads events from an iCalendar source. Source can be either an
// http(s):// URL (typical for Google's "Secret iCal address", iCloud
// subscription URLs, Outlook published feeds) or a local file path.
type ICSClient struct {
	source string
	client *http.Client
}

var _ Calendar = (*ICSClient)(nil)

func NewICSClient(source string) *ICSClient {
	return &ICSClient{
		source: source,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchEvents returns events from now through now+window, expanding RRULE
// recurrences within the window. Events are sorted by start time.
func (c *ICSClient) FetchEvents(window time.Duration) ([]Event, error) {
	if c.source == "" {
		return nil, fmt.Errorf("no calendar source configured")
	}
	cal, err := c.parseSource()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	end := now.Add(window)

	var events []Event
	for _, ev := range cal.Events() {
		expanded := expandEvent(ev, now, end)
		events = append(events, expanded...)
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Start.Before(events[j].Start)
	})
	return events, nil
}

func (c *ICSClient) parseSource() (*ics.Calendar, error) {
	if strings.HasPrefix(c.source, "http://") || strings.HasPrefix(c.source, "https://") {
		req, err := http.NewRequest("GET", c.source, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "today-tui/1.0")
		req.Header.Set("Accept", "text/calendar, */*;q=0.5")
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("calendar fetch: %s", resp.Status)
		}
		return ics.ParseCalendar(resp.Body)
	}
	f, err := os.Open(c.source)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ics.ParseCalendar(f)
}

// expandEvent yields one Event for a non-recurring event (if it overlaps the
// window) or many Events for a recurring one (one per occurrence in the
// window). Returns nil for events that fall outside.
func expandEvent(ev *ics.VEvent, windowStart, windowEnd time.Time) []Event {
	summary := propValue(ev, ics.ComponentPropertySummary)
	location := propValue(ev, ics.ComponentPropertyLocation)
	start, err := ev.GetStartAt()
	if err != nil {
		return nil
	}
	endTime, endErr := ev.GetEndAt()
	if endErr != nil {
		// All-day events with only DTSTART;VALUE=DATE often lack DTEND.
		// Treat as same-day all-day event.
		endTime = start.Add(24 * time.Hour)
	}
	allDay := isAllDay(ev)
	duration := endTime.Sub(start)

	rruleProp := ev.GetProperty(ics.ComponentPropertyRrule)
	if rruleProp == nil {
		// Non-recurring: include if it overlaps the window.
		if endTime.Before(windowStart) || start.After(windowEnd) {
			return nil
		}
		return []Event{{
			Summary:  summary,
			Location: location,
			Start:    start,
			End:      endTime,
			AllDay:   allDay,
		}}
	}

	// Recurring: expand RRULE between windowStart and windowEnd.
	roption, err := rrule.StrToROption(rruleProp.Value)
	if err != nil {
		return nil // malformed rule — skip
	}
	roption.Dtstart = start
	rr, err := rrule.NewRRule(*roption)
	if err != nil {
		return nil
	}
	// Pull occurrences whose start is in [windowStart-duration, windowEnd]
	// so a meeting that started before the window but is still ongoing
	// still surfaces.
	occurrences := rr.Between(windowStart.Add(-duration), windowEnd, true)
	out := make([]Event, 0, len(occurrences))
	for _, occStart := range occurrences {
		out = append(out, Event{
			Summary:  summary,
			Location: location,
			Start:    occStart,
			End:      occStart.Add(duration),
			AllDay:   allDay,
		})
	}
	return out
}

// propValue returns the value of a component property or empty string when
// the property is absent.
func propValue(ev *ics.VEvent, name ics.ComponentProperty) string {
	p := ev.GetProperty(name)
	if p == nil {
		return ""
	}
	return p.Value
}

// isAllDay reports whether the event's DTSTART is a DATE (not DATE-TIME),
// indicating an all-day event per RFC 5545.
func isAllDay(ev *ics.VEvent) bool {
	p := ev.GetProperty(ics.ComponentPropertyDtStart)
	if p == nil {
		return false
	}
	for _, ical := range p.ICalParameters["VALUE"] {
		if strings.EqualFold(ical, "DATE") {
			return true
		}
	}
	return false
}
