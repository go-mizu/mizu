package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// DriverLiveStats holds live statistics for a driver.
type DriverLiveStats struct {
	Driver         string
	Throughput     float64 // Current MB/s
	PrevThroughput float64 // Previous for trend
	Samples        int
	Errors         int
	LastUpdate     time.Time
}

// StatsPanel displays live benchmark statistics.
type StatsPanel struct {
	stats       map[string]*DriverLiveStats
	objectSize  int
	threads     int
	totalTarget int
	completed   int
	startTime   time.Time
	width       int
}

// NewStatsPanel creates a new stats panel.
func NewStatsPanel() *StatsPanel {
	return &StatsPanel{
		stats:     make(map[string]*DriverLiveStats),
		startTime: time.Now(),
		width:     30,
	}
}

// SetConfig sets the current benchmark configuration.
func (s *StatsPanel) SetConfig(objectSize, threads, totalTarget int) {
	s.objectSize = objectSize
	s.threads = threads
	s.totalTarget = totalTarget
}

// UpdateDriver updates statistics for a driver.
func (s *StatsPanel) UpdateDriver(driver string, throughput float64, samples, errors int) {
	stat, ok := s.stats[driver]
	if !ok {
		s.stats[driver] = &DriverLiveStats{
			Driver: driver,
		}
		stat = s.stats[driver]
	}

	stat.PrevThroughput = stat.Throughput
	stat.Throughput = throughput
	stat.Samples = samples
	stat.Errors = errors
	stat.LastUpdate = time.Now()
}

// AddCompleted adds to the completed count.
func (s *StatsPanel) AddCompleted(n int) {
	s.completed += n
}

// SetCompleted sets the completed count.
func (s *StatsPanel) SetCompleted(n int) {
	s.completed = n
}

// SetWidth sets the panel width.
func (s *StatsPanel) SetWidth(width int) {
	s.width = width
}

// Render returns the stats panel display.
func (s *StatsPanel) Render() string {
	var sb strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	sb.WriteString(titleStyle.Render("Live Stats"))
	sb.WriteString("\n\n")

	// Current config
	if s.objectSize > 0 && s.threads > 0 {
		sb.WriteString(MutedStyle.Render("Current: "))
		sb.WriteString(fmt.Sprintf("%s x %d threads\n\n", formatSizeCompact(s.objectSize), s.threads))
	}

	// Driver stats sorted by throughput
	var driverStats []*DriverLiveStats
	for _, stat := range s.stats {
		driverStats = append(driverStats, stat)
	}
	sort.Slice(driverStats, func(i, j int) bool {
		return driverStats[i].Throughput > driverStats[j].Throughput
	})

	for _, stat := range driverStats {
		sb.WriteString(s.renderDriverStat(stat))
		sb.WriteString("\n")
	}

	// Total progress
	sb.WriteString("\n")
	if s.totalTarget > 0 {
		sb.WriteString(MutedStyle.Render(fmt.Sprintf("Samples: %d/%d", s.completed, s.totalTarget)))
	}

	// Total errors
	totalErrors := 0
	for _, stat := range s.stats {
		totalErrors += stat.Errors
	}
	if totalErrors > 0 {
		sb.WriteString("\n")
		sb.WriteString(ErrorStyle.Render(fmt.Sprintf("Errors: %d", totalErrors)))
	}

	return sb.String()
}

// renderDriverStat renders a single driver's stats.
func (s *StatsPanel) renderDriverStat(stat *DriverLiveStats) string {
	var sb strings.Builder

	color := getDriverColor(stat.Driver)
	driverStyle := lipgloss.NewStyle().Foreground(color)

	// Bullet and name
	sb.WriteString(driverStyle.Render("● "))
	sb.WriteString(driverStyle.Render(fmt.Sprintf("%-10s", stat.Driver)))

	// Throughput
	if stat.Throughput > 0 {
		throughputStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
		sb.WriteString(throughputStyle.Render(fmt.Sprintf("%6.0f MB/s ", stat.Throughput)))
	} else {
		sb.WriteString(MutedStyle.Render("     -     "))
	}

	// Trend arrow
	trend := s.getTrend(stat)
	sb.WriteString(trend)

	return sb.String()
}

// getTrend returns the trend indicator.
func (s *StatsPanel) getTrend(stat *DriverLiveStats) string {
	if stat.PrevThroughput == 0 {
		return ""
	}

	diff := stat.Throughput - stat.PrevThroughput
	threshold := stat.PrevThroughput * 0.05 // 5% threshold

	if diff > threshold {
		return SuccessStyle.Render(" ▲")
	} else if diff < -threshold {
		return ErrorStyle.Render(" ▼")
	}
	return MutedStyle.Render(" ─")
}

// Reset clears all stats.
func (s *StatsPanel) Reset() {
	s.stats = make(map[string]*DriverLiveStats)
	s.completed = 0
	s.startTime = time.Now()
}

// formatSizeCompact returns a compact size string.
func formatSizeCompact(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%dGB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%dMB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%dKB", bytes/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// CompactStatsLine returns a single-line summary of stats.
func (s *StatsPanel) CompactStatsLine() string {
	if len(s.stats) == 0 {
		return MutedStyle.Render("No data")
	}

	var best *DriverLiveStats
	for _, stat := range s.stats {
		if best == nil || stat.Throughput > best.Throughput {
			best = stat
		}
	}

	if best == nil || best.Throughput == 0 {
		return MutedStyle.Render("Measuring...")
	}

	color := getDriverColor(best.Driver)
	driverStyle := lipgloss.NewStyle().Foreground(color)

	return fmt.Sprintf("Leader: %s at %.0f MB/s",
		driverStyle.Render(best.Driver),
		best.Throughput)
}
