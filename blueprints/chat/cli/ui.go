package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Color palette (Discord-inspired)
var (
	primaryColor   = lipgloss.Color("#5865F2") // Discord blurple
	secondaryColor = lipgloss.Color("#99AAB5") // Gray
	accentColor    = lipgloss.Color("#57F287") // Green
	successColor   = lipgloss.Color("#57F287") // Green
	errorColor     = lipgloss.Color("#ED4245") // Red
	warnColor      = lipgloss.Color("#FEE75C") // Yellow
	dimColor       = lipgloss.Color("#72767D") // Dim gray
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
			Foreground(primaryColor)

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

	serverStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	channelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#99AAB5"))

	usernameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
)

// Icons
const (
	iconCheck   = "✓"
	iconCross   = "✗"
	iconServer  = "◎"
	iconChannel = "#"
	iconUser    = "◇"
	iconMessage = "▸"
	iconInfo    = "●"
	iconWarning = "▲"
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
	fmt.Printf("%s %s\n", warnStyle.Render(iconWarning), message)
}

// Hint prints a hint message
func (u *UI) Hint(message string) {
	fmt.Printf("  %s\n", hintStyle.Render(message))
}

// Divider prints a horizontal line
func (u *UI) Divider() {
	fmt.Println(subtitleStyle.Render(strings.Repeat("─", 50)))
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

// ServerRow prints a formatted server row
func (u *UI) ServerRow(name string, memberCount int, isPublic bool) {
	visibility := "private"
	if isPublic {
		visibility = "public"
	}
	meta := subtitleStyle.Render(fmt.Sprintf("[%d members] %s", memberCount, visibility))
	if len(name) > 30 {
		name = name[:30] + "..."
	}
	fmt.Printf("  %s %-35s %s\n",
		iconServer,
		serverStyle.Render(name),
		meta)
}

// ChannelRow prints a formatted channel row
func (u *UI) ChannelRow(name, channelType string) {
	fmt.Printf("    %s %s %s\n",
		iconChannel,
		channelStyle.Render(name),
		subtitleStyle.Render("("+channelType+")"))
}

// UserRow prints a formatted user row
func (u *UI) UserRow(username, displayName string, status string) {
	statusIcon := "○"
	switch status {
	case "online":
		statusIcon = successStyle.Render("●")
	case "idle":
		statusIcon = warnStyle.Render("◐")
	case "dnd":
		statusIcon = errorStyle.Render("●")
	}
	fmt.Printf("  %s %s %-20s %s\n",
		iconUser,
		statusIcon,
		usernameStyle.Render(username),
		subtitleStyle.Render(displayName))
}

// Progress prints a progress line
func (u *UI) Progress(icon, message string) {
	fmt.Printf("  %s %s\n", icon, message)
}

// Step prints a step message
func (u *UI) Step(message string) {
	fmt.Printf("  %s %s\n", progressStyle.Render("→"), message)
}

// Item prints a key-value item (alias for Info)
func (u *UI) Item(key, value string) {
	u.Info(key, value)
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
