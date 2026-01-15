package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// DriverProgressState holds the progress state for a single driver.
type DriverProgressState struct {
	Driver       string
	Completed    int
	Total        int
	Throughput   float64 // Current throughput MB/s
	PrevThroughput float64 // Previous throughput for trend
	StartTime    time.Time
	Rank         int  // Current rank (1 = best)
	Active       bool // Whether driver is actively running
}

// MultiDriverProgress manages progress bars for multiple drivers.
type MultiDriverProgress struct {
	drivers   map[string]*DriverProgressState
	barWidth  int
	startTime time.Time
}

// NewMultiDriverProgress creates a new multi-driver progress manager.
func NewMultiDriverProgress() *MultiDriverProgress {
	return &MultiDriverProgress{
		drivers:   make(map[string]*DriverProgressState),
		barWidth:  40,
		startTime: time.Now(),
	}
}

// InitDriver initializes a driver with total samples.
func (m *MultiDriverProgress) InitDriver(driver string, total int) {
	m.drivers[driver] = &DriverProgressState{
		Driver:    driver,
		Total:     total,
		StartTime: time.Now(),
		Active:    true,
	}
}

// Update updates the progress for a driver.
func (m *MultiDriverProgress) Update(driver string, completed int, throughput float64) {
	state, ok := m.drivers[driver]
	if !ok {
		m.drivers[driver] = &DriverProgressState{
			Driver:    driver,
			StartTime: time.Now(),
			Active:    true,
		}
		state = m.drivers[driver]
	}

	state.PrevThroughput = state.Throughput
	state.Completed = completed
	state.Throughput = throughput
	state.Active = true

	// Recalculate ranks
	m.updateRanks()
}

// SetTotal sets the total for a driver.
func (m *MultiDriverProgress) SetTotal(driver string, total int) {
	if state, ok := m.drivers[driver]; ok {
		state.Total = total
	} else {
		m.drivers[driver] = &DriverProgressState{
			Driver:    driver,
			Total:     total,
			StartTime: time.Now(),
		}
	}
}

// Complete marks a driver as completed.
func (m *MultiDriverProgress) Complete(driver string) {
	if state, ok := m.drivers[driver]; ok {
		state.Active = false
		state.Completed = state.Total
	}
}

// updateRanks recalculates the ranking based on throughput.
func (m *MultiDriverProgress) updateRanks() {
	// Collect active drivers with throughput > 0
	var active []*DriverProgressState
	for _, state := range m.drivers {
		if state.Throughput > 0 {
			active = append(active, state)
		}
	}

	// Sort by throughput descending
	sort.Slice(active, func(i, j int) bool {
		return active[i].Throughput > active[j].Throughput
	})

	// Assign ranks
	for i, state := range active {
		state.Rank = i + 1
	}
}

// SetBarWidth sets the progress bar width.
func (m *MultiDriverProgress) SetBarWidth(width int) {
	m.barWidth = width
}

// Render returns the multi-driver progress display.
func (m *MultiDriverProgress) Render() string {
	if len(m.drivers) == 0 {
		return MutedStyle.Render("No drivers active")
	}

	var sb strings.Builder

	// Sort drivers by name for consistent ordering
	var names []string
	for name := range m.drivers {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, name := range names {
		state := m.drivers[name]
		sb.WriteString(m.renderDriverProgress(state))
		if i < len(names)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderDriverProgress renders a single driver's progress bar.
func (m *MultiDriverProgress) renderDriverProgress(state *DriverProgressState) string {
	var sb strings.Builder

	// Driver name with color
	color := getDriverColor(state.Driver)
	driverStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	sb.WriteString(driverStyle.Render(fmt.Sprintf("%-12s", state.Driver)))
	sb.WriteString(" ")

	// Progress bar
	percent := 0.0
	if state.Total > 0 {
		percent = float64(state.Completed) / float64(state.Total)
	}
	if percent > 1 {
		percent = 1
	}

	filled := int(float64(m.barWidth) * percent)
	if filled > m.barWidth {
		filled = m.barWidth
	}

	// Color bar based on rank
	barColor := m.getRankColor(state.Rank)
	barStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	bar := strings.Repeat("█", filled) + strings.Repeat("░", m.barWidth-filled)
	sb.WriteString("[")
	sb.WriteString(barStyle.Render(bar[:filled]))
	sb.WriteString(emptyStyle.Render(bar[filled:]))
	sb.WriteString("] ")

	// Percentage
	sb.WriteString(fmt.Sprintf("%3.0f%% ", percent*100))

	// Throughput with trend
	if state.Throughput > 0 {
		trend := m.getTrend(state)
		throughputStyle := lipgloss.NewStyle().Foreground(ColorGreen)
		sb.WriteString(throughputStyle.Render(fmt.Sprintf("%6.0f MB/s", state.Throughput)))
		sb.WriteString(" ")
		sb.WriteString(trend)
	} else {
		sb.WriteString(MutedStyle.Render("       -    "))
		sb.WriteString("  ")
	}

	// ETA
	if state.Active && state.Completed > 0 && state.Completed < state.Total {
		elapsed := time.Since(state.StartTime)
		remaining := time.Duration(float64(elapsed) / percent * (1 - percent))
		sb.WriteString(MutedStyle.Render(fmt.Sprintf(" ETA %s", formatETA(remaining))))
	}

	// Rank medal
	if state.Rank > 0 && state.Active {
		sb.WriteString(" ")
		sb.WriteString(m.getRankMedal(state.Rank))
	}

	return sb.String()
}

// getRankColor returns the color for a given rank.
func (m *MultiDriverProgress) getRankColor(rank int) lipgloss.Color {
	switch rank {
	case 1:
		return ColorGreen // Gold -> Green for 1st
	case 2:
		return ColorCyan // Silver -> Cyan for 2nd
	case 3:
		return ColorYellow // Bronze -> Yellow for 3rd
	default:
		return ColorMuted
	}
}

// getRankMedal returns the medal emoji for a given rank.
func (m *MultiDriverProgress) getRankMedal(rank int) string {
	switch rank {
	case 1:
		return SuccessStyle.Render("1st")
	case 2:
		return lipgloss.NewStyle().Foreground(ColorCyan).Render("2nd")
	case 3:
		return WarnStyle.Render("3rd")
	default:
		return ""
	}
}

// getTrend returns the trend indicator.
func (m *MultiDriverProgress) getTrend(state *DriverProgressState) string {
	if state.PrevThroughput == 0 {
		return MutedStyle.Render("─")
	}

	diff := state.Throughput - state.PrevThroughput
	threshold := state.PrevThroughput * 0.05 // 5% threshold

	if diff > threshold {
		return SuccessStyle.Render("▲")
	} else if diff < -threshold {
		return ErrorStyle.Render("▼")
	}
	return MutedStyle.Render("─")
}

// formatETA formats a duration as a short ETA string.
func formatETA(d time.Duration) string {
	if d < 0 {
		return "0s"
	}
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

// Reset clears all driver progress.
func (m *MultiDriverProgress) Reset() {
	m.drivers = make(map[string]*DriverProgressState)
	m.startTime = time.Now()
}

// GetDrivers returns all driver names.
func (m *MultiDriverProgress) GetDrivers() []string {
	names := make([]string, 0, len(m.drivers))
	for name := range m.drivers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetState returns the state for a driver.
func (m *MultiDriverProgress) GetState(driver string) *DriverProgressState {
	return m.drivers[driver]
}
