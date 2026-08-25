package console

import (
	"fmt"
	"io"

	"golang.org/x/term"
)

// Color is what a caller asked for on the command line.
//
// It is not what happens. [ColorAuto] is a question, and the answer depends on
// the stream, so an IO decides once per stream when it is built.
type Color int

const (
	// ColorAuto looks at the stream and the environment. It is the zero value
	// and it is what a program should use unless a user said otherwise.
	ColorAuto Color = iota

	// ColorAlways writes escapes whatever the other end is. This is --color=always,
	// and it exists for a pipe that ends up somewhere that understands them.
	ColorAlways

	// ColorNever writes none. This is --no-color and --color=never.
	ColorNever
)

// String returns the name the flag uses.
func (c Color) String() string {
	switch c {
	case ColorAlways:
		return "always"
	case ColorNever:
		return "never"
	default:
		return "auto"
	}
}

// ParseColor reads the value of a --color flag.
func ParseColor(s string) (Color, error) {
	switch s {
	case "auto", "":
		return ColorAuto, nil
	case "always":
		return ColorAlways, nil
	case "never":
		return ColorNever, nil
	}
	return ColorAuto, fmt.Errorf("bad colour %q, want auto, always, or never", s)
}

// colorEnabled decides whether one stream gets escapes.
//
// getenv is a parameter so the rules can be tested without setting variables in
// the process the test is running in.
func colorEnabled(w io.Writer, mode Color, getenv func(string) string) bool {
	switch mode {
	case ColorNever:
		return false
	case ColorAlways:
		return true
	}
	// NO_COLOR is honoured for any value, including one that looks false.
	// https://no-color.org says the variable being present is the signal, and
	// arguing with it means a user who set it still sees escapes.
	if getenv("NO_COLOR") != "" {
		return false
	}
	if getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(w)
}

// isTerminal reports whether w is one.
//
// It asks for the file descriptor through an interface rather than a concrete
// type, so a wrapper that forwards Fd is still recognised and a buffer is
// still not.
func isTerminal(w io.Writer) bool {
	f, ok := w.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(int(f.Fd()))
}

// terminalWidth returns the columns of w, or override when it is positive, or
// 0 when w is not a terminal.
func terminalWidth(w io.Writer, override int) int {
	if override > 0 {
		return override
	}
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return 0
	}
	cols, _, err := term.GetSize(int(f.Fd()))
	if err != nil || cols <= 0 {
		return 0
	}
	return cols
}

// A style is one SGR parameter. There are five of them because five is what
// the messages need, and a palette nobody uses is a palette that drifts from
// what the terminal actually renders.
type style string

const (
	styleNone   style = ""
	styleBold   style = "1"
	styleDim    style = "2"
	styleRed    style = "31"
	styleGreen  style = "32"
	styleYellow style = "33"
)

// wrap returns text with the escapes around it, or text unchanged when colour
// is off or the style is [styleNone].
func (s style) wrap(text string, on bool) string {
	if !on || s == styleNone {
		return text
	}
	return "\x1b[" + string(s) + "m" + text + "\x1b[0m"
}
