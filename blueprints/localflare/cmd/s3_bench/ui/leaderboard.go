package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// LeaderboardEntry represents a single driver's result in the leaderboard.
type LeaderboardEntry struct {
	Driver     string
	Throughput float64
	TTFBP50    time.Duration
	TTFBP99    time.Duration
	Timestamp  time.Time
}

// configKey creates a unique key for a benchmark configuration.
func configKey(objectSize, threads int) string {
	return fmt.Sprintf("%d-%d", objectSize, threads)
}

// Leaderboard tracks benchmark results grouped by configuration.
type Leaderboard struct {
	// Results grouped by config key (objectSize-threads)
	results map[string]map[string]*LeaderboardEntry // configKey -> driver -> entry

	// Current config
	objectSize int
	threads    int

	// Display settings
	width int
}

// NewLeaderboard creates a new leaderboard.
func NewLeaderboard() *Leaderboard {
	return &Leaderboard{
		results: make(map[string]map[string]*LeaderboardEntry),
		width:   40,
	}
}

// SetConfig sets the current benchmark configuration.
func (l *Leaderboard) SetConfig(objectSize, threads int) {
	l.objectSize = objectSize
	l.threads = threads
}

// SetWidth sets the display width.
func (l *Leaderboard) SetWidth(width int) {
	l.width = width
}

// AddResult adds a benchmark result to the leaderboard.
func (l *Leaderboard) AddResult(driver string, objectSize, threads int, throughput float64, ttfbP50, ttfbP99 time.Duration) {
	key := configKey(objectSize, threads)

	if l.results[key] == nil {
		l.results[key] = make(map[string]*LeaderboardEntry)
	}

	l.results[key][driver] = &LeaderboardEntry{
		Driver:     driver,
		Throughput: throughput,
		TTFBP50:    ttfbP50,
		TTFBP99:    ttfbP99,
		Timestamp:  time.Now(),
	}
}

// GetCurrentEntries returns entries for the current configuration, sorted by throughput.
func (l *Leaderboard) GetCurrentEntries() []*LeaderboardEntry {
	key := configKey(l.objectSize, l.threads)
	configResults := l.results[key]

	if len(configResults) == 0 {
		return nil
	}

	entries := make([]*LeaderboardEntry, 0, len(configResults))
	for _, entry := range configResults {
		entries = append(entries, entry)
	}

	// Sort by throughput descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Throughput > entries[j].Throughput
	})

	return entries
}

// Render returns the leaderboard display for the current configuration.
func (l *Leaderboard) Render() string {
	var sb strings.Builder

	// Title with current config
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	if l.objectSize > 0 && l.threads > 0 {
		sb.WriteString(titleStyle.Render(fmt.Sprintf("Leaderboard (%s @ %d threads)", FormatSizeCompact(l.objectSize), l.threads)))
	} else {
		sb.WriteString(titleStyle.Render("Leaderboard"))
	}
	sb.WriteString("\n\n")

	entries := l.GetCurrentEntries()
	if len(entries) == 0 {
		sb.WriteString(MutedStyle.Render("  No results yet for this config\n"))
		return sb.String()
	}

	// Find max throughput for bar scaling
	maxThroughput := entries[0].Throughput

	for i, entry := range entries {
		sb.WriteString(l.renderEntry(entry, i+1, maxThroughput))
		if i < len(entries)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderEntry renders a single leaderboard entry.
func (l *Leaderboard) renderEntry(entry *LeaderboardEntry, rank int, maxThroughput float64) string {
	var sb strings.Builder

	// Rank with medal/styling
	rankStr := l.getRankString(rank)
	sb.WriteString(fmt.Sprintf("  %s ", rankStr))

	// Driver name with color
	color := getDriverColor(entry.Driver)
	driverStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	sb.WriteString(driverStyle.Render(fmt.Sprintf("%-12s", entry.Driver)))

	// Throughput bar
	barWidth := 20
	if l.width > 60 {
		barWidth = 30
	}
	percent := entry.Throughput / maxThroughput
	filled := int(float64(barWidth) * percent)
	if filled < 1 && entry.Throughput > 0 {
		filled = 1
	}

	barColor := l.getRankColor(rank)
	barStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	sb.WriteString(" ")
	if filled > 0 {
		sb.WriteString(barStyle.Render(strings.Repeat("█", filled)))
	}
	if barWidth-filled > 0 {
		sb.WriteString(emptyStyle.Render(strings.Repeat("░", barWidth-filled)))
	}

	// Throughput value
	throughputStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	sb.WriteString(" ")
	sb.WriteString(throughputStyle.Render(fmt.Sprintf("%6.0f MB/s", entry.Throughput)))

	// TTFB (p50)
	if entry.TTFBP50 > 0 {
		sb.WriteString(MutedStyle.Render(fmt.Sprintf(" (p50: %dms)", entry.TTFBP50.Milliseconds())))
	}

	return sb.String()
}

// getRankString returns a styled rank string.
func (l *Leaderboard) getRankString(rank int) string {
	switch rank {
	case 1:
		return SuccessStyle.Render("1st")
	case 2:
		return lipgloss.NewStyle().Foreground(ColorCyan).Render("2nd")
	case 3:
		return WarnStyle.Render("3rd")
	default:
		return MutedStyle.Render(fmt.Sprintf("%dth", rank))
	}
}

// getRankColor returns the color for a given rank.
func (l *Leaderboard) getRankColor(rank int) lipgloss.Color {
	switch rank {
	case 1:
		return ColorGreen
	case 2:
		return ColorCyan
	case 3:
		return ColorYellow
	default:
		return ColorMuted
	}
}

// Reset clears all leaderboard data.
func (l *Leaderboard) Reset() {
	l.results = make(map[string]map[string]*LeaderboardEntry)
}

// HasResults returns true if there are any results for the current config.
func (l *Leaderboard) HasResults() bool {
	key := configKey(l.objectSize, l.threads)
	return len(l.results[key]) > 0
}
