package cli

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	errorColor     = lipgloss.Color("#EF4444")
	warningColor   = lipgloss.Color("#F59E0B")
	mutedColor     = lipgloss.Color("#9CA3AF")

	// Text styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	successStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	_ = lipgloss.NewStyle().
		Foreground(warningColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	labelStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	urlStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Underline(true)

	// Box style
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)
)

// Banner returns the ASCII art banner
func Banner() string {
	banner := `
   ____        _
  | __ )  ___ | |_
  |  _ \ / _ \| __|
  | |_) | (_) | |_
  |____/ \___/ \__|
`
	return titleStyle.Render(banner)
}
