package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Output handles CLI output with color, verbosity, and format support.
type Output struct {
	Stdout  io.Writer
	Stderr  io.Writer
	noColor bool
}

// NewOutput creates a new styled output instance.
func NewOutput() *Output {
	noColor := Flags.NoColor || os.Getenv("NO_COLOR") != "" || !isTerminal(os.Stdout)
	return &Output{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		noColor: noColor,
	}
}

// Print writes to stdout unless quiet mode is enabled.
func (o *Output) Print(format string, args ...any) {
	if Flags.Quiet && !Flags.JSON {
		return
	}
	fmt.Fprintf(o.Stdout, format, args...)
}

// Errorf writes an error message to stderr.
func (o *Output) Errorf(format string, args ...any) {
	if o.noColor {
		fmt.Fprintf(o.Stderr, format, args...)
		return
	}
	fmt.Fprint(o.Stderr, errorStyle.Render(fmt.Sprintf(format, args...)))
}

// Verbosef writes to stderr if verbosity level is met.
func (o *Output) Verbosef(level int, format string, args ...any) {
	if Flags.Verbose >= level && !Flags.Quiet {
		fmt.Fprintf(o.Stderr, format, args...)
	}
}

// Title renders text as a title.
func (o *Output) Title(text string) string {
	if o.noColor {
		return text
	}
	return titleStyle.Render(text)
}

// Bold renders text in bold.
func (o *Output) Bold(text string) string {
	if o.noColor {
		return text
	}
	return boldStyle.Render(text)
}

// Cyan renders text in cyan.
func (o *Output) Cyan(text string) string {
	if o.noColor {
		return text
	}
	return cyanStyle.Render(text)
}

// Dim renders text in dim/gray.
func (o *Output) Dim(text string) string {
	if o.noColor {
		return text
	}
	return dimStyle.Render(text)
}

// Success renders text in green.
func (o *Output) Success(text string) string {
	if o.noColor {
		return text
	}
	return successStyle.Render(text)
}

// Warn renders text in yellow.
func (o *Output) Warn(text string) string {
	if o.noColor {
		return text
	}
	return warnStyle.Render(text)
}

// WriteJSON writes a JSON-encoded value to stdout.
func (o *Output) WriteJSON(v any) error {
	enc := json.NewEncoder(o.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteJSONError writes a JSON error to stdout.
func (o *Output) WriteJSONError(code, message string) {
	o.WriteJSON(map[string]string{
		"error":   code,
		"message": message,
	})
}

// isTerminal checks if w is a terminal.
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
