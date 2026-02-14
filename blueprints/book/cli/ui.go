package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
  ║   Personal Library       ║
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

// Bar renders a block-character progress bar like ████████░░░░.
func Bar(filled, total, width int) string {
	if total <= 0 || width <= 0 {
		return strings.Repeat("░", width)
	}
	n := filled * width / total
	if n > width {
		n = width
	}
	return strings.Repeat("█", n) + strings.Repeat("░", width-n)
}

func printStep(step, total int, name string) {
	header := fmt.Sprintf("Step %d/%d · %s", step, total, name)
	fmt.Printf("\n  %s\n  %s\n", header, strings.Repeat("─", len(header)+2))
}

func printBar(label string, count, total int, width int) {
	pct := float64(0)
	if total > 0 {
		pct = float64(count) * 100.0 / float64(total)
	}
	fmt.Printf("    %-11s%12s  %5.1f%%  %s\n", label, formatNumberCompact(count), pct, Bar(count, total, width))
}

func printSummaryBox(lines ...string) {
	rule := strings.Repeat("═", 60)
	fmt.Printf("\n  %s\n", rule)
	for _, l := range lines {
		fmt.Printf("  %s\n", l)
	}
	fmt.Printf("  %s\n", rule)
}

func formatNumberCompact(n int) string {
	if n < 1_000_000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}
