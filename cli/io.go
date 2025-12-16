package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// output handles CLI output with color, verbosity, and format support.
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
	_, _ = fmt.Fprintf(o.stdout, format, args...)
}

func (o *output) errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(o.stderr, format, args...)
}

func (o *output) verbosef(level int, format string, args ...any) {
	if o.verbose >= level && !o.quiet {
		_, _ = fmt.Fprintf(o.stderr, format, args...)
	}
}

func (o *output) color(c, text string) string {
	if o.noColor {
		return text
	}
	return c + text + colorReset
}

func (o *output) green(text string) string  { return o.color(colorGreen, text) }
func (o *output) red(text string) string    { return o.color(colorRed, text) }
func (o *output) yellow(text string) string { return o.color(colorYellow, text) }
func (o *output) blue(text string) string   { return o.color(colorBlue, text) }
func (o *output) cyan(text string) string   { return o.color(colorCyan, text) }
func (o *output) gray(text string) string   { return o.color(colorGray, text) }
func (o *output) bold(text string) string   { return o.color(colorBold, text) }

func (o *output) ok(msg string) {
	o.print("  %s %s\n", o.green("[ok]"), msg)
}

func (o *output) warn(msg string) {
	o.print("  %s %s\n", o.yellow("[warn]"), msg)
}

func (o *output) fail(msg string) {
	o.print("  %s %s\n", o.red("[error]"), msg)
}

func (o *output) info(msg string) {
	o.print("  %s %s\n", o.blue("[info]"), msg)
}

// isTerminal checks if w is a terminal (simplified check).
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		info, err := f.Stat()
		if err != nil {
			return false
		}
		return (info.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// padRight pads s to width with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
