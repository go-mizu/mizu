package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Icons
const (
	iconCheck = "✓"
	iconCross = "✗"
	iconArrow = "→"
	iconDot   = "•"
)

var (
	// Colors - Metabase blue theme
	primaryColor   = lipgloss.Color("#509EE3") // Metabase Blue
	secondaryColor = lipgloss.Color("#88BF4D") // Green
	accentColor    = lipgloss.Color("#A989C5") // Purple
	errorColor     = lipgloss.Color("#EF8C8C") // Red
	warnColor      = lipgloss.Color("#F9CF48") // Yellow
	mutedColor     = lipgloss.Color("#949AAB") // Gray

	// Styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	successStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	warnStyle = lipgloss.NewStyle().
			Foreground(warnColor)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	keyStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(14)

	valueStyle = lipgloss.NewStyle()
)

// Header prints a styled header.
func Header(icon, text string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", icon, headerStyle.Render(text))
}

// Blank prints a blank line.
func Blank() {
	fmt.Fprintln(os.Stderr)
}

// Summary prints key-value pairs.
func Summary(pairs ...string) {
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		val := ""
		if i+1 < len(pairs) {
			val = pairs[i+1]
		}
		fmt.Fprintf(os.Stderr, "  %s %s\n", keyStyle.Render(key+":"), valueStyle.Render(val))
	}
}

// Success prints a success message.
func Success(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", successStyle.Render("[OK]"), msg)
}

// Error prints an error message.
func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", errorStyle.Render("[ERROR]"), msg)
}

// Warn prints a warning message.
func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", warnStyle.Render("[WARN]"), msg)
}

// Hint prints a hint message.
func Hint(msg string) {
	fmt.Fprintf(os.Stderr, "  %s\n", mutedStyle.Render(msg))
}

// Step prints a step message with optional duration.
func Step(icon, msg string, d ...time.Duration) {
	if len(d) > 0 {
		fmt.Fprintf(os.Stderr, "%s %s %s\n", icon, msg, mutedStyle.Render(fmt.Sprintf("(%s)", d[0].Round(time.Millisecond))))
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", icon, msg)
	}
}

// StartSpinner starts a simple spinner.
func StartSpinner(msg string) func() {
	done := make(chan struct{})
	go func() {
		frames := []string{"|", "/", "-", "\\"}
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(msg)+5))
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], msg)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	return func() {
		close(done)
		time.Sleep(50 * time.Millisecond)
	}
}

// modeString returns the mode display string.
func modeString(dev bool) string {
	if dev {
		return "development"
	}
	return "production"
}
