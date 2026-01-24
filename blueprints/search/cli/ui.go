package cli

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#4285F4")
	secondaryColor = lipgloss.Color("#34A853")
	errorColor     = lipgloss.Color("#EA4335")
	warningColor   = lipgloss.Color("#FBBC05")
	mutedColor     = lipgloss.Color("#9AA0A6")

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

	warningStyle = lipgloss.NewStyle().
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
   ____                      _
  / ___|  ___  __ _ _ __ ___| |__
  \___ \ / _ \/ _` + "`" + ` | '__/ __| '_ \
   ___) |  __/ (_| | | | (__| | | |
  |____/ \___|\__,_|_|  \___|_| |_|
`
	return titleStyle.Render(banner)
}
