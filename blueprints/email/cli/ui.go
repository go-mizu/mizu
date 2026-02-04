package cli

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors - Gmail-inspired palette
	primaryColor   = lipgloss.Color("#EA4335")
	secondaryColor = lipgloss.Color("#34A853")
	errorColor     = lipgloss.Color("#EA4335")
	warningColor   = lipgloss.Color("#FBBC05")
	mutedColor     = lipgloss.Color("#5F6368")

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

// Ensure warningStyle is used (prevent lint errors)
var _ = warningStyle

// Banner returns the ASCII art banner
func Banner() string {
	banner := `
   _____                 _ _
  | ____|_ __ ___   __ _(_) |
  |  _| | '_ ` + "`" + ` _ \ / _` + "`" + ` | | |
  | |___| | | | | | (_| | | |
  |_____|_| |_| |_|\__,_|_|_|
`
	return titleStyle.Render(banner)
}
