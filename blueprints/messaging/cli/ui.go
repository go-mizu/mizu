package cli

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Icons
const (
	iconServer   = "🚀"
	iconDatabase = "💾"
	iconSeed     = "🌱"
	iconCheck    = "✓"
	iconCross    = "✗"
	iconArrow    = "→"
	iconDot      = "•"
)

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Width(12)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))
)

// UI provides styled output.
type UI struct{}

// NewUI creates a new UI.
func NewUI() *UI {
	return &UI{}
}

// Header prints a styled header.
func (u *UI) Header(icon, text string) {
	fmt.Println(headerStyle.Render(icon + " " + text))
}

// Blank prints a blank line.
func (u *UI) Blank() {
	fmt.Println()
}

// Step prints a step.
func (u *UI) Step(text string) {
	fmt.Println(dimStyle.Render(iconArrow) + " " + text)
}

// Success prints a success message.
func (u *UI) Success(text string) {
	fmt.Println(successStyle.Render(iconCheck + " " + text))
}

// Error prints an error message.
func (u *UI) Error(text string) {
	fmt.Println(errorStyle.Render(iconCross + " " + text))
}

// Warn prints a warning message.
func (u *UI) Warn(text string) {
	fmt.Println(warnStyle.Render(iconDot + " " + text))
}

// Hint prints a hint.
func (u *UI) Hint(text string) {
	fmt.Println(dimStyle.Render(text))
}

// Summary prints a summary table.
func (u *UI) Summary(items [][2]string) {
	for _, item := range items {
		fmt.Println(labelStyle.Render(item[0]+":") + " " + valueStyle.Render(item[1]))
	}
}

// StartSpinner starts a spinner.
func (u *UI) StartSpinner(text string) {
	fmt.Print(dimStyle.Render(iconDot + " " + text))
}

// StopSpinner stops the spinner with success.
func (u *UI) StopSpinner(text string, duration time.Duration) {
	fmt.Printf("\r%s %s %s\n",
		successStyle.Render(iconCheck),
		text,
		dimStyle.Render(fmt.Sprintf("(%s)", duration.Round(time.Millisecond))),
	)
}

// StopSpinnerError stops the spinner with error.
func (u *UI) StopSpinnerError(text string) {
	fmt.Printf("\r%s\n", errorStyle.Render(iconCross+" "+text))
}
