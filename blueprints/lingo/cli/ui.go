package cli

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Lingo green color (Duolingo-inspired)
	lingoGreen = lipgloss.Color("#58CC02")
	lingoBlue  = lipgloss.Color("#1CB0F6")
	lingoGold  = lipgloss.Color("#FFC800")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lingoGreen)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	urlStyle = lipgloss.NewStyle().
			Foreground(lingoBlue).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lingoGreen)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4B4B"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lingoGreen).
			Padding(1, 2)
)

// Banner returns the ASCII art banner
func Banner() string {
	banner := `
    __    _
   / /   (_)___  ____ _____
  / /   / / __ \/ __ '/ __ \
 / /___/ / / / / /_/ / /_/ /
/_____/_/_/ /_/\__, /\____/
              /____/
`
	return titleStyle.Render(banner)
}
