package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	panelTodo    = "todo"
	panelWeather = "weather"
	panelStocks  = "stocks"
	panelStats   = "stats"
	panelNews    = "news"

	keyUp    = "up"
	keyDown  = "down"
	keyEnter = "enter"
	keyEsc   = "esc"
)

var (
	colorTodo    = lipgloss.Color("69")  // blue
	colorWeather = lipgloss.Color("51")  // cyan
	colorStocks  = lipgloss.Color("82")  // green
	colorStats   = lipgloss.Color("220") // yellow
	colorNews    = lipgloss.Color("213") // magenta
	colorDim     = lipgloss.Color("245")
	colorErr     = lipgloss.Color("196")
	colorPos     = lipgloss.Color("82")
	colorNeg     = lipgloss.Color("196")

	errStyle  = lipgloss.NewStyle().Foreground(colorErr)
	dimStyle  = lipgloss.NewStyle().Foreground(colorDim)
	boldStyle = lipgloss.NewStyle().Bold(true)
)

func paneStyle(accent lipgloss.Color, focused bool, w, h int) lipgloss.Style {
	border := colorDim
	if focused {
		border = accent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(w - 2).
		Height(h - 2)
}

func progressBar(pct float64, width int) string {
	if width <= 0 {
		return ""
	}
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// truncate shortens s to at most maxLen display columns, appending an ellipsis
// when content was dropped. Operates on visual width (runewidth) rather than
// bytes so multi-byte and double-width runes don't get sliced mid-character.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return runewidth.Truncate(s, maxLen, "")
	}
	return runewidth.Truncate(s, maxLen, "...")
}

// padRight pads s with spaces on the right to reach width display columns.
// No-op if s is already at least that wide.
func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
