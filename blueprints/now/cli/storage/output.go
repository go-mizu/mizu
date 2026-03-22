package storage

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// Exit codes matching the bash CLI spec.
const (
	ExitOK         = 0
	ExitError      = 1
	ExitUsage      = 2
	ExitAuth       = 3
	ExitNotFound   = 4
	ExitConflict   = 5
	ExitPermission = 6
	ExitNetwork    = 7
)

// CLIError is a structured error with exit code and hint.
type CLIError struct {
	Code int
	Msg  string
	Hint string
}

func (e *CLIError) Error() string { return e.Msg }

// Output handles formatted output to stdout/stderr.
type Output struct {
	IsTTY   bool
	NoColor bool
	Quiet   bool
}

// NewOutput creates an Output that auto-detects TTY and color.
func NewOutput(noColor, quiet bool) *Output {
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		noColor = true
	}
	return &Output{IsTTY: isTTY, NoColor: noColor || !isTTY, Quiet: quiet}
}

func (o *Output) bold(s string) string {
	if o.NoColor {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

func (o *Output) dim(s string) string {
	if o.NoColor {
		return s
	}
	return "\033[2m" + s + "\033[0m"
}

func (o *Output) green(s string) string {
	if o.NoColor {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

func (o *Output) red(s string) string {
	if o.NoColor {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}

func (o *Output) yellow(s string) string {
	if o.NoColor {
		return s
	}
	return "\033[33m" + s + "\033[0m"
}

func (o *Output) cyan(s string) string {
	if o.NoColor {
		return s
	}
	return "\033[36m" + s + "\033[0m"
}

// Info prints a success-style message to stderr.
func (o *Output) Info(action, detail string) {
	if o.Quiet {
		return
	}
	fmt.Fprintf(os.Stderr, "  %s %s\n", o.green(action), detail)
}

// Warn prints a warning to stderr.
func (o *Output) Warn(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", o.yellow("warning:"), msg)
}

// PrintError prints a structured error to stderr.
func (o *Output) PrintError(msg, reason, hint string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", o.red("error:"), msg)
	if reason != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", reason)
	}
	if hint != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", hint)
	}
}

// HumanSize formats bytes into a human-readable string.
func HumanSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// RelativeTime formats an epoch-ms timestamp as relative time.
func RelativeTime(epochMs int64) string {
	if epochMs == 0 {
		return "-"
	}
	t := time.UnixMilli(epochMs)
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	}
}
