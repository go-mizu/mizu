package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// output is the legacy output type for backward compatibility.
// New code should use Output instead.
type output struct {
	stdout  io.Writer
	stderr  io.Writer
	json    bool
	quiet   bool
	verbose int
	noColor bool
}

func newOutput(json, quiet, noColor bool, verbose int) *output {
	return &output{
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		json:    json,
		quiet:   quiet,
		verbose: verbose,
		noColor: noColor || !isTerminal(os.Stdout),
	}
}

func (o *output) print(format string, args ...any) {
	if o.quiet && !o.json {
		return
	}
	fmt.Fprintf(o.stdout, format, args...)
}

func (o *output) errorf(format string, args ...any) {
	fmt.Fprintf(o.stderr, format, args...)
}

func (o *output) verbosef(level int, format string, args ...any) {
	if o.verbose >= level && !o.quiet {
		fmt.Fprintf(o.stderr, format, args...)
	}
}

// ANSI color codes for legacy output.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func (o *output) color(c, text string) string {
	if o.noColor {
		return text
	}
	return c + text + colorReset
}

func (o *output) green(text string) string  { return o.color(colorGreen, text) }
func (o *output) yellow(text string) string { return o.color(colorYellow, text) }
func (o *output) cyan(text string) string   { return o.color(colorCyan, text) }
func (o *output) gray(text string) string   { return o.color(colorGray, text) }
func (o *output) bold(text string) string   { return o.color(colorBold, text) }

// padRight pads s to width with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
