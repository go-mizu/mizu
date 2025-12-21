package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	primaryColor   = lipgloss.Color("#10B981") // Emerald green
	secondaryColor = lipgloss.Color("#6B7280") // Gray
	accentColor    = lipgloss.Color("#3B82F6") // Blue
	successColor   = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	dimColor       = lipgloss.Color("#9CA3AF") // Dim gray
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	labelStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	progressStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)
)

// Icons
const (
	iconDownload = "↓"
	iconCheck    = "✓"
	iconCross    = "✗"
	iconDatabase = "◉"
	iconServer   = "◎"
)

// Spinner frames
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// UI handles formatted CLI output
type UI struct {
	mu       sync.Mutex
	spinning bool
	spinMsg  string
	spinDone chan struct{}
}

// NewUI creates a new UI instance
func NewUI() *UI {
	return &UI{}
}

// Header prints a styled header
func (u *UI) Header(icon, title string) {
	fmt.Println()
	fmt.Printf("%s %s\n", icon, titleStyle.Render(title))
}

// Info prints a key-value pair
func (u *UI) Info(label, value string) {
	fmt.Printf("  %s %s\n",
		labelStyle.Render(label+":"),
		valueStyle.Render(value))
}

// Blank prints an empty line
func (u *UI) Blank() {
	fmt.Println()
}

// StartSpinner starts an animated spinner
func (u *UI) StartSpinner(message string) {
	u.mu.Lock()
	if u.spinning {
		u.mu.Unlock()
		return
	}
	u.spinning = true
	u.spinMsg = message
	u.spinDone = make(chan struct{})
	u.mu.Unlock()

	go func() {
		i := 0
		for {
			select {
			case <-u.spinDone:
				fmt.Print("\r\033[K") // Clear line
				return
			default:
				u.mu.Lock()
				msg := u.spinMsg
				u.mu.Unlock()
				frame := progressStyle.Render(spinnerFrames[i])
				fmt.Printf("\r%s %s", frame, msg)
				i = (i + 1) % len(spinnerFrames)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// UpdateSpinner updates the spinner message
func (u *UI) UpdateSpinner(message string) {
	u.mu.Lock()
	u.spinMsg = message
	u.mu.Unlock()
}

// StopSpinner stops the spinner and shows a completion message
func (u *UI) StopSpinner(message string, duration time.Duration) {
	u.mu.Lock()
	if !u.spinning {
		u.mu.Unlock()
		return
	}
	close(u.spinDone)
	u.spinning = false
	u.mu.Unlock()

	time.Sleep(100 * time.Millisecond) // Let spinner goroutine exit

	durStr := subtitleStyle.Render(fmt.Sprintf("(%s)", duration.Round(time.Millisecond)))
	fmt.Printf("%s %s %s\n", successStyle.Render(iconCheck), message, durStr)
}

// StopSpinnerError stops the spinner and shows an error
func (u *UI) StopSpinnerError(message string) {
	u.mu.Lock()
	if !u.spinning {
		u.mu.Unlock()
		return
	}
	close(u.spinDone)
	u.spinning = false
	u.mu.Unlock()

	time.Sleep(100 * time.Millisecond)
	fmt.Printf("%s %s\n", errorStyle.Render(iconCross), message)
}

// Progress prints file download progress
func (u *UI) Progress(current, total int, filename, size string, skipped bool) {
	progress := progressStyle.Render(fmt.Sprintf("[%d/%d]", current, total))
	sizeStr := subtitleStyle.Render(fmt.Sprintf("(%s)", size))
	if skipped {
		skipStr := hintStyle.Render("skipped")
		fmt.Printf("%s %s %s %s\n", progress, filename, sizeStr, skipStr)
	} else {
		fmt.Printf("%s %s %s\n", progress, filename, sizeStr)
	}
}

// ProgressDone prints file download completion
func (u *UI) ProgressDone(current, total int, filename, size string, duration time.Duration) {
	progress := progressStyle.Render(fmt.Sprintf("[%d/%d]", current, total))
	sizeStr := subtitleStyle.Render(fmt.Sprintf("(%s)", size))
	durStr := subtitleStyle.Render(fmt.Sprintf("%s", duration.Round(100*time.Millisecond)))
	fmt.Printf("%s %s %s %s %s\n", successStyle.Render(iconCheck), progress, filename, sizeStr, durStr)
}

// Success prints a success message
func (u *UI) Success(message string) {
	fmt.Println()
	fmt.Printf("%s %s\n", successStyle.Render(iconCheck), message)
}

// Error prints an error message
func (u *UI) Error(message string) {
	fmt.Println()
	fmt.Printf("%s %s\n", errorStyle.Render(iconCross), message)
}

// Hint prints a hint message
func (u *UI) Hint(message string) {
	fmt.Printf("  %s\n", hintStyle.Render(message))
}

// Divider prints a horizontal line
func (u *UI) Divider() {
	fmt.Println(subtitleStyle.Render(strings.Repeat("─", 40)))
}

// Summary prints a summary section
func (u *UI) Summary(items [][2]string) {
	fmt.Println()
	u.Divider()
	for _, item := range items {
		u.Info(item[0], item[1])
	}
	u.Divider()
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
