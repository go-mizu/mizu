package ui

import "github.com/charmbracelet/lipgloss"

// Colors - matching s3-benchmark output
var (
	ColorGreen   = lipgloss.Color("#10B981") // Green for throughput
	ColorYellow  = lipgloss.Color("#F59E0B") // Yellow for warnings
	ColorRed     = lipgloss.Color("#EF4444") // Red for errors
	ColorBlue    = lipgloss.Color("#3B82F6") // Blue for info
	ColorMuted   = lipgloss.Color("#6B7280") // Gray for muted text
	ColorCyan    = lipgloss.Color("#06B6D4") // Cyan for headers
	ColorMagenta = lipgloss.Color("#8B5CF6") // Purple for driver names
)

// Styles
var (
	// Header style for section headers (--- SETUP ---)
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan)

	// Phase divider style
	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Success/OK style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed)

	// Warning style
	WarnStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	// Info/muted style
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Driver name style
	DriverStyle = lipgloss.NewStyle().
			Foreground(ColorMagenta).
			Bold(true)

	// Throughput value style (colored green like s3-benchmark)
	ThroughputStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	// Table header style
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorCyan)

	// Table border style
	TableBorderStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// Progress bar styles
	ProgressCompleteStyle = lipgloss.NewStyle().
				Foreground(ColorGreen)

	ProgressIncompleteStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			MarginBottom(1)

	// Subtitle style
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// RenderDivider renders a section divider line.
func RenderDivider(title string, width int) string {
	if width < 20 {
		width = 80
	}

	prefix := "--- "
	suffix := " "
	dashes := width - len(prefix) - len(title) - len(suffix)
	if dashes < 10 {
		dashes = 10
	}

	line := prefix + title + suffix
	for i := 0; i < dashes; i++ {
		line += "-"
	}

	return DividerStyle.Render(line)
}
