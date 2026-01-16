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
	Throughput     float64       // Current MB/s
	PrevThroughput float64       // Previous for trend
	TTFB           time.Duration // Time to first byte
	TTLB           time.Duration // Time to last byte
	Samples        int
	Errors         int
	LastUpdate     time.Time
}

// StatsPanel displays live benchmark statistics.
type StatsPanel struct {
	stats          map[string]*DriverLiveStats
	currentDriver  string
	objectSize     int
	threads        int
	totalTarget    int
	completed      int
	startTime      time.Time
	width          int
	statusMessage  string
	operation      string // "upload", "download", "cleanup"
	currentTTFB    time.Duration
	currentTTLB    time.Duration
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

// SetCurrentDriver sets the current driver being tested.
func (s *StatsPanel) SetCurrentDriver(driver string) {
	s.currentDriver = driver
}

// SetStatus sets the current status message.
func (s *StatsPanel) SetStatus(message string) {
	s.statusMessage = message
}

// SetOperation sets the current operation (upload, download, cleanup).
func (s *StatsPanel) SetOperation(operation string) {
	s.operation = operation
}

// SetLatency sets the current latency values.
func (s *StatsPanel) SetLatency(ttfb, ttlb time.Duration) {
	s.currentTTFB = ttfb
	s.currentTTLB = ttlb
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

// UpdateDriverLatency updates latency for a driver.
func (s *StatsPanel) UpdateDriverLatency(driver string, ttfb, ttlb time.Duration) {
	stat, ok := s.stats[driver]
	if !ok {
		s.stats[driver] = &DriverLiveStats{
			Driver: driver,
		}
		stat = s.stats[driver]
	}
	stat.TTFB = ttfb
	stat.TTLB = ttlb
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

	// Title with current driver
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	sb.WriteString(titleStyle.Render("Current Test"))
	sb.WriteString("\n\n")

	// Status message (prominent if set)
	if s.statusMessage != "" {
		// Check if it's an error message
		isError := len(s.statusMessage) > 6 && (s.statusMessage[:6] == "[ERROR" || strings.Contains(s.statusMessage, "FAILED"))
		if isError {
			sb.WriteString(ErrorStyle.Render("  " + s.statusMessage))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(ColorYellow).Render("  " + s.statusMessage))
		}
		sb.WriteString("\n\n")
	}

	// Current driver being tested (big and clear)
	if s.currentDriver != "" {
		color := getDriverColor(s.currentDriver)
		driverStyle := lipgloss.NewStyle().Bold(true).Foreground(color)
		sb.WriteString("  Driver: ")
		sb.WriteString(driverStyle.Render(s.currentDriver))
		sb.WriteString("\n")
	}

	// Current operation
	if s.operation != "" {
		opStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		sb.WriteString("  ")
		sb.WriteString(opStyle.Render(s.operation))
		sb.WriteString("\n")
	}

	// Current config
	if s.objectSize > 0 || s.threads > 0 {
		sb.WriteString("  Config: ")
		if s.objectSize > 0 {
			sb.WriteString(fmt.Sprintf("%s", formatSizeCompact(s.objectSize)))
		}
		if s.threads > 0 {
			sb.WriteString(fmt.Sprintf(" × %d threads", s.threads))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Live metrics for current driver
	if s.currentDriver != "" {
		stat := s.stats[s.currentDriver]
		if stat != nil && stat.Throughput > 0 {
			// Throughput
			throughputStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorGreen)
			sb.WriteString("  Throughput: ")
			sb.WriteString(throughputStyle.Render(fmt.Sprintf("%.0f MB/s", stat.Throughput)))
			sb.WriteString(s.getTrend(stat))
			sb.WriteString("\n")

			// Latency (TTFB)
			if stat.TTFB > 0 {
				ttfbStyle := lipgloss.NewStyle().Foreground(ColorCyan)
				sb.WriteString("  TTFB:       ")
				sb.WriteString(ttfbStyle.Render(formatLatency(stat.TTFB)))
				sb.WriteString("\n")
			}

			// Latency (TTLB)
			if stat.TTLB > 0 {
				ttlbStyle := lipgloss.NewStyle().Foreground(ColorBlue)
				sb.WriteString("  TTLB:       ")
				sb.WriteString(ttlbStyle.Render(formatLatency(stat.TTLB)))
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString(MutedStyle.Render("  Measuring...\n"))
		}
	}

	sb.WriteString("\n")

	// Progress
	if s.totalTarget > 0 {
		percent := float64(s.completed) / float64(s.totalTarget) * 100
		sb.WriteString(fmt.Sprintf("  Progress: %d/%d (%.0f%%)\n", s.completed, s.totalTarget, percent))
	}

	// Elapsed time
	elapsed := time.Since(s.startTime)
	sb.WriteString(MutedStyle.Render(fmt.Sprintf("  Elapsed: %s\n", elapsed.Round(time.Second))))

	// Errors
	totalErrors := 0
	for _, stat := range s.stats {
		totalErrors += stat.Errors
	}
	if totalErrors > 0 {
		sb.WriteString(ErrorStyle.Render(fmt.Sprintf("  Errors: %d\n", totalErrors)))
	}

	return sb.String()
}

// RenderLeaderboard returns a comparison of all drivers tested.
func (s *StatsPanel) RenderLeaderboard() string {
	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	sb.WriteString(titleStyle.Render("Results"))
	sb.WriteString("\n\n")

	// Sort by throughput
	var driverStats []*DriverLiveStats
	for _, stat := range s.stats {
		if stat.Throughput > 0 {
			driverStats = append(driverStats, stat)
		}
	}

	if len(driverStats) == 0 {
		sb.WriteString(MutedStyle.Render("  No results yet\n"))
		return sb.String()
	}

	sort.Slice(driverStats, func(i, j int) bool {
		return driverStats[i].Throughput > driverStats[j].Throughput
	})

	for i, stat := range driverStats {
		color := getDriverColor(stat.Driver)
		driverStyle := lipgloss.NewStyle().Foreground(color)

		// Rank
		rank := ""
		switch i {
		case 0:
			rank = SuccessStyle.Render("1st")
		case 1:
			rank = lipgloss.NewStyle().Foreground(ColorCyan).Render("2nd")
		case 2:
			rank = WarnStyle.Render("3rd")
		default:
			rank = MutedStyle.Render(fmt.Sprintf("%dth", i+1))
		}

		sb.WriteString(fmt.Sprintf("  %s %s %s\n",
			rank,
			driverStyle.Render(fmt.Sprintf("%-10s", stat.Driver)),
			fmt.Sprintf("%.0f MB/s", stat.Throughput)))
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
	s.currentDriver = ""
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

// formatLatency formats a duration as latency.
func formatLatency(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
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
