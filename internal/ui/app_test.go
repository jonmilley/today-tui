package ui

import (
	"testing"

	"github.com/jonmilley/today-tui/internal/config"
)

// Test layout dimensions. 100 wide makes the math convenient
// (leftW = 100*2/5 = 40, rightW = 60 → top-row halves are 30 each), and 30
// tall leaves availH = 29 (status bar takes 1 row).
const (
	testW = 100
	testH = 30
)

// newTestApp constructs an App at a fixed testW×testH size with the given
// panel visibility, runs the layout pass, and returns it ready for assertions.
func newTestApp(panels config.PanelConfig) App {
	a := App{
		mode:   modeDash,
		cfg:    &config.Config{Panels: panels},
		width:  testW,
		height: testH,
		ready:  true,
	}
	a.resizePanes()
	return a
}

func TestPaneAtAllVisible(t *testing.T) {
	a := newTestApp(config.PanelConfig{
		Todo: true, Calendar: true, Weather: true, Stocks: true, Stats: true, News: true,
	})

	tests := []struct {
		name string
		x, y int
		want int
	}{
		{"top-left → todo", 1, 1, paneTodo},
		{"upper-mid-left (still todo)", 5, 5, paneTodo},
		{"middle-left → calendar", 10, 20, paneCalendar},
		{"top-right-half-left → weather", 50, 5, paneWeather},
		{"top-right-far-right → stocks", 90, 5, paneStocks},
		{"bottom-right-left → stats", 50, 25, paneStats},
		{"bottom-right-right → news", 90, 25, paneNews},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := a.paneAt(tt.x, tt.y)
			if !ok {
				t.Fatalf("paneAt(%d, %d) returned !ok", tt.x, tt.y)
			}
			if got != tt.want {
				t.Errorf("paneAt(%d, %d) = %d; want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestPaneAtMissesStatusBar(t *testing.T) {
	a := newTestApp(config.PanelConfig{
		Todo: true, Calendar: true, Weather: true, Stocks: true, Stats: true, News: true,
	})
	// Status bar is the bottom row at y=29 (height=30, statusH=1).
	if _, ok := a.paneAt(50, 29); ok {
		t.Errorf("paneAt(50, 29) hit a pane; expected miss on status bar row")
	}
}

func TestPaneAtTodoOnly(t *testing.T) {
	// With only Todo visible, it should fill the whole left column AND
	// (since there's no right side) span the full width.
	a := newTestApp(config.PanelConfig{Todo: true})

	got, ok := a.paneAt(50, 15)
	if !ok || got != paneTodo {
		t.Errorf("paneAt center = (%d, %v); want paneTodo, true", got, ok)
	}
	got, ok = a.paneAt(95, 15)
	if !ok || got != paneTodo {
		t.Errorf("paneAt far-right = (%d, %v); want paneTodo, true", got, ok)
	}
}

func TestPaneAtNoLeftPanes(t *testing.T) {
	// Right-only configuration: right panes should fill full width.
	a := newTestApp(config.PanelConfig{
		Weather: true, Stocks: true, Stats: true, News: true,
	})
	got, ok := a.paneAt(5, 5)
	if !ok || got != paneWeather {
		t.Errorf("paneAt(5, 5) = (%d, %v); want paneWeather (no left column)", got, ok)
	}
}

func TestSplitColumns(t *testing.T) {
	tests := []struct {
		name  string
		total int
		left  bool
		right bool
		wantL int
		wantR int
	}{
		{"both → 2/5 left", 100, true, true, 40, 60},
		{"left only → full width", 100, true, false, 100, 0},
		{"right only → full width", 100, false, true, 0, 100},
		{"neither → zero", 100, false, false, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, r := splitColumns(tt.total, tt.left, tt.right)
			if l != tt.wantL || r != tt.wantR {
				t.Errorf("splitColumns(%d, %v, %v) = (%d, %d); want (%d, %d)",
					tt.total, tt.left, tt.right, l, r, tt.wantL, tt.wantR)
			}
		})
	}
}
