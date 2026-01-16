package cli

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#3ECF8E") // Supabase green
	secondaryColor = lipgloss.Color("#1F2937")
	errorColor     = lipgloss.Color("#EF4444")
	warningColor   = lipgloss.Color("#F59E0B")
	successColor   = lipgloss.Color("#10B981")
	infoColor      = lipgloss.Color("#3B82F6")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(infoColor)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Width(20)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3F4F6"))

	urlStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Underline(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(1, 2)
)

// Banner returns the localbase ASCII banner
func Banner() string {
	banner := `
 _                    _ _
| |    ___   ___ __ _| | |__   __ _ ___  ___
| |   / _ \ / __/ _  | | '_ \ / _  / __|/ _ \
| |__| (_) | (_| (_| | | |_) | (_| \__ \  __/
|_____\___/ \___\__,_|_|_.__/ \__,_|___/\___|
`
	return titleStyle.Render(banner)
}
