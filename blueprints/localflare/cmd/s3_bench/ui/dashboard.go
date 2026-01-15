package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Dashboard composes all UI components into a unified view.
type Dashboard struct {
	// Components
	chart          *ThroughputChart
	driverProgress *MultiDriverProgress
	statsPanel     *StatsPanel
	resultsTable   *ResultsTable

	// State
	phase      Phase
	viewMode   ViewMode
	paused     bool
	startTime  time.Time
	width      int
	height     int
	objectSize int
	threads    int

	// Logs buffer for log view
	logs []string
}

// NewDashboard creates a new dashboard.
func NewDashboard() *Dashboard {
	return &Dashboard{
		chart:          NewThroughputChart(),
		driverProgress: NewMultiDriverProgress(),
		statsPanel:     NewStatsPanel(),
		viewMode:       ViewDashboard,
		startTime:      time.Now(),
		width:          120,
		height:         40,
		logs:           make([]string, 0, 100),
	}
}

// SetSize updates the dashboard dimensions.
func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height

	// Update component widths
	chartWidth := width/2 - 4
	if chartWidth < 40 {
		chartWidth = 40
	}
	d.chart.SetSize(chartWidth, 10)
	d.driverProgress.SetBarWidth(width - 50)
	d.statsPanel.SetWidth(width/2 - 4)
}

// SetPhase updates the current phase.
func (d *Dashboard) SetPhase(phase Phase) {
	d.phase = phase
}

// SetConfig updates the current configuration.
func (d *Dashboard) SetConfig(objectSize, threads int) {
	d.objectSize = objectSize
	d.threads = threads
	d.statsPanel.SetConfig(objectSize, threads, 0)
}

// SetViewMode changes the view mode.
func (d *Dashboard) SetViewMode(mode ViewMode) {
	d.viewMode = mode
}

// TogglePause toggles pause state.
func (d *Dashboard) TogglePause() {
	d.paused = !d.paused
}

// AddLog adds a log message.
func (d *Dashboard) AddLog(msg string) {
	d.logs = append(d.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	if len(d.logs) > 100 {
		d.logs = d.logs[1:]
	}
}

// UpdateThroughput updates throughput chart.
func (d *Dashboard) UpdateThroughput(driver string, throughput float64) {
	d.chart.AddSample(driver, throughput, time.Now())
	d.statsPanel.UpdateDriver(driver, throughput, 0, 0)
}

// UpdateDriverProgress updates a driver's progress.
func (d *Dashboard) UpdateDriverProgress(driver string, completed, total int, throughput float64) {
	d.driverProgress.Update(driver, completed, throughput)
	d.driverProgress.SetTotal(driver, total)
}

// InitDriver initializes a driver.
func (d *Dashboard) InitDriver(driver string, total int) {
	d.driverProgress.InitDriver(driver, total)
}

// SetResultsTable sets the current results table.
func (d *Dashboard) SetResultsTable(table *ResultsTable) {
	d.resultsTable = table
}

// GetChart returns the chart component.
func (d *Dashboard) GetChart() *ThroughputChart {
	return d.chart
}

// GetDriverProgress returns the driver progress component.
func (d *Dashboard) GetDriverProgress() *MultiDriverProgress {
	return d.driverProgress
}

// GetStatsPanel returns the stats panel component.
func (d *Dashboard) GetStatsPanel() *StatsPanel {
	return d.statsPanel
}

// Render returns the complete dashboard view.
func (d *Dashboard) Render() string {
	switch d.viewMode {
	case ViewDashboard:
		return d.renderDashboard()
	case ViewDetails:
		return d.renderDetails()
	case ViewLogs:
		return d.renderLogs()
	case ViewHelp:
		return d.renderHelp()
	default:
		return d.renderDashboard()
	}
}

// renderDashboard renders the main dashboard view.
func (d *Dashboard) renderDashboard() string {
	var sb strings.Builder

	// Header
	sb.WriteString(d.renderHeader())
	sb.WriteString("\n")

	// Main content area - chart and stats side by side
	chartContent := d.chart.Render()
	statsContent := d.statsPanel.Render()

	// Create side-by-side layout
	chartLines := strings.Split(chartContent, "\n")
	statsLines := strings.Split(statsContent, "\n")

	chartWidth := d.width/2 - 2
	statsWidth := d.width/2 - 2

	maxLines := len(chartLines)
	if len(statsLines) > maxLines {
		maxLines = len(statsLines)
	}

	// Box styles
	chartBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Width(chartWidth).
		Padding(0, 1)

	statsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Width(statsWidth).
		Padding(0, 1)

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		chartBox.Render(chartContent),
		statsBox.Render(statsContent),
	))
	sb.WriteString("\n")

	// Progress section
	sb.WriteString(d.renderProgressSection())
	sb.WriteString("\n")

	// Results table (compact)
	if d.resultsTable != nil && len(d.resultsTable.Rows) > 0 {
		sb.WriteString(d.renderCompactResults())
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString(d.renderFooter())

	return sb.String()
}

// renderHeader renders the dashboard header.
func (d *Dashboard) renderHeader() string {
	// Title and phase
	title := TitleStyle.Render("S3 Benchmark Dashboard")

	// Phase indicator
	phaseStyle := d.getPhaseStyle()
	phase := phaseStyle.Render(fmt.Sprintf(" %s ", d.phase.String()))

	// Elapsed time
	elapsed := time.Since(d.startTime)
	elapsedStr := MutedStyle.Render(formatElapsed(elapsed))

	// Pause indicator
	pauseStr := ""
	if d.paused {
		pauseStr = WarnStyle.Render(" [PAUSED] ")
	}

	// Build header line
	leftPart := title + "  " + phase + pauseStr
	rightPart := elapsedStr

	// Calculate spacing
	spacing := d.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if spacing < 0 {
		spacing = 0
	}

	return leftPart + strings.Repeat(" ", spacing) + rightPart
}

// getPhaseStyle returns the style for the current phase.
func (d *Dashboard) getPhaseStyle() lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true).Padding(0, 1)

	switch d.phase {
	case PhaseSetup:
		return base.Background(ColorYellow).Foreground(lipgloss.Color("#000000"))
	case PhaseBenchmark:
		return base.Background(ColorGreen).Foreground(lipgloss.Color("#000000"))
	case PhaseCleanup:
		return base.Background(ColorBlue).Foreground(lipgloss.Color("#FFFFFF"))
	case PhaseDone:
		return base.Background(ColorCyan).Foreground(lipgloss.Color("#000000"))
	default:
		return base.Background(ColorMuted).Foreground(lipgloss.Color("#FFFFFF"))
	}
}

// renderProgressSection renders the driver progress bars.
func (d *Dashboard) renderProgressSection() string {
	var sb strings.Builder

	// Section title
	sb.WriteString(TableHeaderStyle.Render("Driver Progress"))
	sb.WriteString("\n\n")

	// Progress bars
	sb.WriteString(d.driverProgress.Render())

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Width(d.width - 4).
		Padding(0, 1).
		Render(sb.String())
}

// renderCompactResults renders a compact results summary.
func (d *Dashboard) renderCompactResults() string {
	if d.resultsTable == nil || len(d.resultsTable.Rows) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf("Latest Results (%s objects)", FormatSizeCompact(d.objectSize))))
	sb.WriteString("\n\n")

	// Compact table header
	sb.WriteString(TableBorderStyle.Render("| "))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%-10s", "Driver")))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%7s", "Threads")))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%12s", "Throughput")))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%10s", "TTFB p50")))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%10s", "TTFB p99")))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%6s", "Rank")))
	sb.WriteString(TableBorderStyle.Render(" |\n"))

	// Separator
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(strings.Repeat("-", 12))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(strings.Repeat("-", 9))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(strings.Repeat("-", 14))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(strings.Repeat("-", 12))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(strings.Repeat("-", 12))
	sb.WriteString(TableBorderStyle.Render("|"))
	sb.WriteString(strings.Repeat("-", 8))
	sb.WriteString(TableBorderStyle.Render("|\n"))

	// Sort rows by throughput for ranking
	rows := make([]TableRow, len(d.resultsTable.Rows))
	copy(rows, d.resultsTable.Rows)

	// Show last few rows (most recent results)
	maxRows := 6
	if len(rows) > maxRows {
		rows = rows[len(rows)-maxRows:]
	}

	for i, row := range rows {
		rank := len(rows) - i // Simplistic ranking by order
		sb.WriteString(d.renderCompactRow(row, rank))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Width(d.width - 4).
		Padding(0, 1).
		Render(sb.String())
}

// renderCompactRow renders a single compact result row.
func (d *Dashboard) renderCompactRow(row TableRow, rank int) string {
	var sb strings.Builder

	color := getDriverColor(row.Driver)
	driverStyle := lipgloss.NewStyle().Foreground(color)

	sb.WriteString(TableBorderStyle.Render("| "))
	sb.WriteString(driverStyle.Render(fmt.Sprintf("%-10s", row.Driver)))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(fmt.Sprintf("%7d", row.Threads))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(ThroughputStyle.Render(fmt.Sprintf("%8.1f MB/s", row.Throughput)))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(fmt.Sprintf("%7d ms", row.TTFBP50))
	sb.WriteString(TableBorderStyle.Render(" | "))
	sb.WriteString(fmt.Sprintf("%7d ms", row.TTFBP99))
	sb.WriteString(TableBorderStyle.Render(" | "))

	// Rank
	rankStr := ""
	switch rank {
	case 1:
		rankStr = SuccessStyle.Render(" 1st ")
	case 2:
		rankStr = lipgloss.NewStyle().Foreground(ColorCyan).Render(" 2nd ")
	case 3:
		rankStr = WarnStyle.Render(" 3rd ")
	default:
		rankStr = MutedStyle.Render(fmt.Sprintf(" %dth ", rank))
	}
	sb.WriteString(rankStr)
	sb.WriteString(TableBorderStyle.Render(" |\n"))

	return sb.String()
}

// renderFooter renders the footer with keyboard shortcuts.
func (d *Dashboard) renderFooter() string {
	shortcuts := []string{
		"[q] Quit",
		"[d] Details",
		"[l] Logs",
		"[?] Help",
	}

	return MutedStyle.Render(strings.Join(shortcuts, "  "))
}

// renderDetails renders the detailed view.
func (d *Dashboard) renderDetails() string {
	var sb strings.Builder

	sb.WriteString(d.renderHeader())
	sb.WriteString("\n\n")

	sb.WriteString(TableHeaderStyle.Render("Detailed Results"))
	sb.WriteString("\n\n")

	if d.resultsTable != nil {
		sb.WriteString(d.resultsTable.Render())
	} else {
		sb.WriteString(MutedStyle.Render("No results yet"))
	}

	sb.WriteString("\n\n")
	sb.WriteString(d.renderFooter())

	return sb.String()
}

// renderLogs renders the log view.
func (d *Dashboard) renderLogs() string {
	var sb strings.Builder

	sb.WriteString(d.renderHeader())
	sb.WriteString("\n\n")

	sb.WriteString(TableHeaderStyle.Render("Logs"))
	sb.WriteString("\n\n")

	// Show last N logs
	maxLogs := d.height - 10
	if maxLogs < 5 {
		maxLogs = 5
	}

	start := 0
	if len(d.logs) > maxLogs {
		start = len(d.logs) - maxLogs
	}

	for _, log := range d.logs[start:] {
		sb.WriteString(MutedStyle.Render(log))
		sb.WriteString("\n")
	}

	if len(d.logs) == 0 {
		sb.WriteString(MutedStyle.Render("No logs yet"))
	}

	sb.WriteString("\n")
	sb.WriteString(d.renderFooter())

	return sb.String()
}

// renderHelp renders the help view.
func (d *Dashboard) renderHelp() string {
	var sb strings.Builder

	sb.WriteString(d.renderHeader())
	sb.WriteString("\n\n")

	sb.WriteString(TableHeaderStyle.Render("Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	shortcuts := [][2]string{
		{"q", "Quit the benchmark"},
		{"d", "Toggle dashboard/details view"},
		{"l", "View logs"},
		{"?", "Show this help"},
		{"Esc", "Return to dashboard"},
	}

	for _, s := range shortcuts {
		key := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan).Render(fmt.Sprintf("%-8s", s[0]))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", key, s[1]))
	}

	sb.WriteString("\n\n")

	sb.WriteString(TableHeaderStyle.Render("Views"))
	sb.WriteString("\n\n")

	views := [][2]string{
		{"Dashboard", "Real-time throughput chart, progress bars, and live stats"},
		{"Details", "Full results table with all metrics"},
		{"Logs", "Debug log messages"},
	}

	for _, v := range views {
		name := lipgloss.NewStyle().Bold(true).Foreground(ColorGreen).Render(fmt.Sprintf("%-12s", v[0]))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", name, v[1]))
	}

	sb.WriteString("\n")
	sb.WriteString(MutedStyle.Render("Press Esc or ? to return to dashboard"))

	return sb.String()
}

// formatElapsed formats elapsed time.
func formatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// Reset resets the dashboard state.
func (d *Dashboard) Reset() {
	d.chart.Clear()
	d.driverProgress.Reset()
	d.statsPanel.Reset()
	d.resultsTable = nil
	d.logs = d.logs[:0]
	d.startTime = time.Now()
	d.phase = PhaseInit
}
