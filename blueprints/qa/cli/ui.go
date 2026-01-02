package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Color palette (Stack Overflow theme)
var (
	primaryColor   = lipgloss.Color("#f48024")
	secondaryColor = lipgloss.Color("#6a737c")
	accentColor    = lipgloss.Color("#0a95ff")
	successColor   = lipgloss.Color("#2f6f44")
	errorColor     = lipgloss.Color("#c22e32")
	warnColor      = lipgloss.Color("#d77a00")
	dimColor       = lipgloss.Color("#9fa6ad")
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
			Foreground(lipgloss.Color("#e3e6e8"))

	progressStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(warnColor)

	hintStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)

	usernameStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	tagStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	badgeStyle = lipgloss.NewStyle().
			Foreground(successColor)
)

// Icons
const (
	iconCheck    = "[ok]"
	iconCross    = "[x]"
	iconDatabase = "[db]"
	iconServer   = "[srv]"
	iconUser     = "[user]"
	iconTag      = "[tag]"
	iconQuestion = "[q]"
	iconAnswer   = "[a]"
	iconComment  = "[c]"
	iconInfo     = "[i]"
	iconWarning  = "[!]"
)

// Spinner frames
var spinnerFrames = []string{"|", "/", "-", "\\"}

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
				fmt.Print("\r\033[K")
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

	time.Sleep(100 * time.Millisecond)

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

// Warn prints a warning message
func (u *UI) Warn(message string) {
	fmt.Println()
	fmt.Printf("%s %s\n", warnStyle.Render(iconWarning), message)
}

// Step prints a step message
func (u *UI) Step(message string) {
	fmt.Printf("  %s %s\n", subtitleStyle.Render(iconInfo), message)
}

// Hint prints a hint message
func (u *UI) Hint(message string) {
	fmt.Printf("  %s\n", hintStyle.Render(message))
}

// Summary prints a list of key-value pairs
func (u *UI) Summary(items [][2]string) {
	fmt.Println()
	for _, item := range items {
		fmt.Printf("  %s %s\n", labelStyle.Render(item[0]+":"), valueStyle.Render(item[1]))
	}
}

// List prints a list of items
func (u *UI) List(title string, items []string) {
	fmt.Println()
	fmt.Printf("%s\n", titleStyle.Render(title))
	for _, item := range items {
		fmt.Printf("  - %s\n", item)
	}
}

// KeyValueTable prints a table of key-value pairs
func (u *UI) KeyValueTable(rows [][2]string) {
	maxLen := 0
	for _, row := range rows {
		if len(row[0]) > maxLen {
			maxLen = len(row[0])
		}
	}

	for _, row := range rows {
		label := row[0] + ":"
		pad := strings.Repeat(" ", maxLen-len(row[0]))
		fmt.Printf("  %s%s %s\n", labelStyle.Render(label), pad, valueStyle.Render(row[1]))
	}
}

// Print displays a normal message
func (u *UI) Print(message string) {
	fmt.Println(message)
}

// Printf prints a formatted message
func (u *UI) Printf(format string, args ...any) {
	fmt.Printf(format, args...)
}

// Exit prints an error message and exits
func (u *UI) Exit(message string, code int) {
	fmt.Println(errorStyle.Render(message))
	os.Exit(code)
}
