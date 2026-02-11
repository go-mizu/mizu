package cli

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#382110"))
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5f6368"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00635D"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#D93025"))
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#1a73e8"))
	urlStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00635D")).Underline(true)
	labelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#5f6368")).Width(14)
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("#382110"))
	starStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#E87400"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#999999"))
)

func Banner() string {
	return titleStyle.Render(`
  ╔══════════════════════════╗
  ║   📚  Book Manager  📚   ║
  ║   Goodreads-compatible   ║
  ╚══════════════════════════╝`)
}

func Stars(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "★"
	}
	for i := n; i < 5; i++ {
		s += "☆"
	}
	return starStyle.Render(s)
}
