package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/distatus/battery"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type batteryInfo struct {
	pct     float64
	state   battery.AgnosticState
	present bool
}

type statsTickMsg struct {
	cpu       float64
	memPct    float64
	memGB     float64
	memTot    float64
	disk      float64
	uptime    uint64
	cpuTemp   float64
	cpuTempOK bool
	battery   batteryInfo
}

type statsPane struct {
	last    statsTickMsg
	viewport viewport.Model
	width   int
	height  int
	focused bool
}

func newStatsPane() statsPane {
	return statsPane{}
}

func (p statsPane) Init() tea.Cmd {
	return tickStats()
}

func tickStats() tea.Cmd {
	return tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return gatherStats()
	})
}

func gatherStats() tea.Msg {
	var msg statsTickMsg

	if pcts, err := cpu.Percent(0, false); err == nil && len(pcts) > 0 {
		msg.cpu = pcts[0]
	}
	if v, err := mem.VirtualMemory(); err == nil {
		msg.memPct = v.UsedPercent
		msg.memGB = float64(v.Used) / 1024 / 1024 / 1024
		msg.memTot = float64(v.Total) / 1024 / 1024 / 1024
	}
	if d, err := disk.Usage("/"); err == nil {
		msg.disk = d.UsedPercent
	}
	if info, err := host.Info(); err == nil {
		msg.uptime = info.Uptime
	}
	if temps, err := host.SensorsTemperatures(); err == nil {
		msg.cpuTemp, msg.cpuTempOK = findCPUTemp(temps)
	}
	if batteries, err := battery.GetAll(); err == nil && len(batteries) > 0 {
		// Aggregate across all batteries
		var totalFull, totalCurrent float64
		state := batteries[0].State
		for _, b := range batteries {
			totalFull += b.Full
			totalCurrent += b.Current
		}
		if totalFull > 0 {
			msg.battery = batteryInfo{
				pct:     totalCurrent / totalFull * 100,
				state:   state.Raw,
				present: true,
			}
		}
	}
	return msg
}

// findCPUTemp picks the best available CPU temperature reading.
// Priority: Intel package → AMD Tctl → highest-numbered core → any core/cpu sensor.
func findCPUTemp(sensors []host.TemperatureStat) (float64, bool) {
	priority := []string{
		"coretemp_package_id_0",
		"k10temp_tctl",
		"k10temp_tccd1",
		"cpu_thermal_input",
		"cpu-thermal_input",
	}
	byKey := make(map[string]float64, len(sensors))
	for _, s := range sensors {
		byKey[s.SensorKey] = s.Temperature
	}
	for _, key := range priority {
		if t, ok := byKey[key]; ok && t > 0 {
			return t, true
		}
	}
	// Fall back: average of all "core" sensors (e.g. coretemp_core0_input …)
	var sum float64
	var n int
	for _, s := range sensors {
		k := strings.ToLower(s.SensorKey)
		if (strings.Contains(k, "core") || strings.Contains(k, "cpu")) && s.Temperature > 0 {
			sum += s.Temperature
			n++
		}
	}
	if n > 0 {
		return sum / float64(n), true
	}
	return 0, false
}

func (p statsPane) Update(msg tea.Msg) (statsPane, tea.Cmd) {
	switch msg := msg.(type) {
	case statsTickMsg:
		p.last = msg
		p.viewport.SetContent(p.renderContent())
		return p, tickStats()
	}
	return p, nil
}

func (p *statsPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.viewport.Width = w - 4
	p.viewport.Height = h - 4
	p.viewport.SetContent(p.renderContent())
}

func (p *statsPane) SetFocused(f bool) { p.focused = f }

func (p statsPane) renderContent() string {
	barW := p.width - 21
	if barW < 4 {
		barW = 4
	}

	cpuColor := colorPos
	if p.last.cpu > 80 {
		cpuColor = colorNeg
	}
	memColor := colorPos
	if p.last.memPct > 80 {
		memColor = colorNeg
	}
	diskColor := colorPos
	if p.last.disk > 80 {
		diskColor = colorNeg
	}

	cpuBar := lipgloss.NewStyle().Foreground(cpuColor).Render(progressBar(p.last.cpu, barW))
	memBar := lipgloss.NewStyle().Foreground(memColor).Render(progressBar(p.last.memPct, barW))
	diskBar := lipgloss.NewStyle().Foreground(diskColor).Render(progressBar(p.last.disk, barW))

	upSec := p.last.uptime
	days := upSec / 86400
	hours := (upSec % 86400) / 3600
	mins := (upSec % 3600) / 60

	uptimeLine := fmt.Sprintf("  Uptime:  %dd %dh %dm", days, hours, mins)

	lines := []string{
		fmt.Sprintf("  CPU   [%s] %.1f%%", cpuBar, p.last.cpu),
	}
	if p.last.cpuTempOK {
		lines = append(lines, cpuTempLine(p.last.cpuTemp))
	}
	lines = append(lines,
		fmt.Sprintf("  MEM   [%s] %.1f%%", memBar, p.last.memPct),
		fmt.Sprintf("         %.1f / %.1f GB", p.last.memGB, p.last.memTot),
		fmt.Sprintf("  DISK  [%s] %.1f%%", diskBar, p.last.disk),
		"",
	)
	if p.last.battery.present {
		lines = append(lines, batteryLine(p.last.battery, barW))
	}
	lines = append(lines, uptimeLine)
	return strings.Join(lines, "\n")
}

func cpuTempLine(temp float64) string {
	tempF := temp*9/5 + 32
	var color lipgloss.Color
	switch {
	case temp >= 80:
		color = colorNeg // red
	case temp >= 60:
		color = colorStats // yellow
	default:
		color = colorPos // green
	}
	val := fmt.Sprintf("%.0f°C / %.0f°F", temp, tempF)
	return "  TEMP  " + lipgloss.NewStyle().Foreground(color).Render(val)
}

func batteryLine(b batteryInfo, barW int) string {
	var color lipgloss.Color
	switch {
	case b.pct <= 20:
		color = colorNeg
	case b.pct <= 50:
		color = colorStats
	default:
		color = colorPos
	}

	bar := lipgloss.NewStyle().Foreground(color).Render(progressBar(b.pct, barW))

	var status string
	switch b.state {
	case battery.Charging:
		status = "charging"
	case battery.Full:
		status = "full"
	case battery.Discharging:
		status = "discharging"
	default:
		status = ""
	}

	line := fmt.Sprintf("  BAT   [%s] %.0f%%", bar, b.pct)
	if status != "" {
		line += "\n" + dimStyle.Render("         "+status)
	}
	return line
}

func (p statsPane) View() string {
	accentStyle := lipgloss.NewStyle().Foreground(colorStats).Bold(true)
	title := accentStyle.Render("SYSTEM")
	sep := dimStyle.Render(strings.Repeat("─", p.width-4))

	inner := lipgloss.JoinVertical(lipgloss.Left, title, sep, p.viewport.View())
	return paneStyle(colorStats, p.focused, p.width, p.height).Render(inner)
}
