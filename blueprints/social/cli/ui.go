package cli

import (
	"fmt"
)

// UI provides styled terminal output.
type UI struct{}

// NewUI creates a new UI.
func NewUI() *UI {
	return &UI{}
}

// Header prints a styled header.
func (u *UI) Header(name, version string) {
	fmt.Printf("\n  %s v%s\n\n", name, version)
}

// Info prints an info message.
func (u *UI) Info(msg string) {
	fmt.Printf("  %s\n", msg)
}

// Success prints a success message.
func (u *UI) Success(msg string) {
	fmt.Printf("  [OK] %s\n", msg)
}

// Warning prints a warning message.
func (u *UI) Warning(msg string) {
	fmt.Printf("  [WARN] %s\n", msg)
}

// Error prints an error message.
func (u *UI) Error(msg string) {
	fmt.Printf("  [ERROR] %s\n", msg)
}

// Item prints a key-value item.
func (u *UI) Item(key, value string) {
	fmt.Printf("    %s: %s\n", key, value)
}
