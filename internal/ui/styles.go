package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	titleStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	errStyle   = lipgloss.NewStyle().Foreground(colorErr)
	dimStyle   = lipgloss.NewStyle().Foreground(colorDim)
	boldStyle  = lipgloss.NewStyle().Bold(true)
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

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
