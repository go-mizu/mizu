package console

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Verbosity is how much a command says about what it is doing.
//
// It is the count of -v flags, with [Quiet] for --quiet, so a caller can hold
// it as the number it came from rather than translating twice.
type Verbosity int

const (
	// Quiet is --quiet. Warnings, errors, and the data the command was asked
	// for. Nothing else.
	Quiet Verbosity = -1

	// Normal is the default and the zero value.
	Normal Verbosity = 0

	// Verbose is -v. Debug lines appear, and an error prints its cause chain.
	Verbose Verbosity = 1

	// Trace is -vv. An error prints a stack as well. Anything above this is
	// more of the same, and is left to whoever finds a use for it.
	Trace Verbosity = 2
)

// Options configure an [IO]. The zero value is the sensible one: normal
// verbosity, human output, and colour decided by looking at the terminal.
type Options struct {
	// Verbosity is how much the command says. See [Verbosity].
	Verbosity Verbosity

	// JSON turns on machine readable output. See [IO.Table] and [IO.JSON].
	JSON bool

	// Color decides whether output carries ANSI escapes. The zero value,
	// [ColorAuto], asks the terminal and the environment.
	Color Color

	// Interaction decides whether prompts ask or take their default. The zero
	// value, [InteractionAuto], asks when there is somebody there to answer.
	Interaction Interaction

	// Width is the number of columns to assume, overriding what the terminal
	// reports. Zero means ask the terminal, which is what a real program
	// wants and what a test does not.
	Width int
}

// An IO is where a command reads from and writes to.
//
// It is not safe for concurrent use. A command that fans out gathers its
// output and writes it from one place, which is also the only way the result
// is readable.
type IO struct {
	in  io.Reader
	out io.Writer
	err io.Writer

	verbosity Verbosity
	jsonMode  bool

	colorOut bool
	colorErr bool
	width    int

	interactive bool
	// reader is built by the first prompt and kept, because it buffers. See
	// [IO.readLine].
	reader *bufio.Reader
}

// New returns an IO on the given streams.
//
// Colour is decided per stream, by looking at whether it is a terminal, so
// passing a buffer for one of them does not change the other. Whether prompts
// ask is decided once here, for the same reason and from the same question.
func New(in io.Reader, out, err io.Writer, opts Options) *IO {
	return &IO{
		in:          in,
		out:         out,
		err:         err,
		verbosity:   opts.Verbosity,
		jsonMode:    opts.JSON,
		colorOut:    colorEnabled(out, opts.Color, os.Getenv),
		colorErr:    colorEnabled(err, opts.Color, os.Getenv),
		width:       terminalWidth(out, opts.Width),
		interactive: canPrompt(in, err, opts.Interaction),
	}
}

// Stdio returns an IO on the process's own streams.
func Stdio(opts Options) *IO {
	return New(os.Stdin, os.Stdout, os.Stderr, opts)
}

// In returns the stream the command reads from.
func (c *IO) In() io.Reader { return c.in }

// Out returns the stream data goes to.
func (c *IO) Out() io.Writer { return c.out }

// Err returns the stream everything else goes to.
func (c *IO) Err() io.Writer { return c.err }

// Verbosity returns how much the command was asked to say.
func (c *IO) Verbosity() Verbosity { return c.verbosity }

// JSONMode reports whether output should be machine readable.
//
// Most commands do not need to ask, because [IO.Table] and [IO.JSON] already
// know. A command whose two renderings are genuinely different shapes does.
func (c *IO) JSONMode() bool { return c.jsonMode }

// Width returns the number of columns available on stdout, or 0 when stdout is
// not a terminal.
//
// Zero is the answer to give a pipe, not a guess at one. A caller that needs a
// number either way picks its own default and says so.
func (c *IO) Width() int { return c.width }

// Print writes to stdout. It adds no newline, so the caller controls the line.
func (c *IO) Print(format string, a ...any) {
	fmt.Fprintf(c.out, format, a...)
}

// Line writes s to stdout, followed by a newline.
//
// It takes a string rather than a format, so a line containing a percent sign
// arrives intact.
func (c *IO) Line(s string) {
	fmt.Fprintln(c.out, s)
}

// Info writes a line to stderr. It is silent when the command was asked to be
// quiet or to speak JSON.
func (c *IO) Info(format string, a ...any) {
	if c.decorated() {
		c.say(styleNone, "", format, a...)
	}
}

// Success writes a line to stderr in green. It is silent under the same
// conditions as [IO.Info].
func (c *IO) Success(format string, a ...any) {
	if c.decorated() {
		c.say(styleGreen, "", format, a...)
	}
}

// Warn writes a line to stderr, prefixed with "warning: ".
//
// It prints in every mode. --quiet is a request for less output, not for a
// problem to go unmentioned, and the prefix is there because colour is gone by
// the time somebody reads the log this ends up in.
func (c *IO) Warn(format string, a ...any) {
	c.say(styleYellow, "warning: ", format, a...)
}

// Error writes a line to stderr, prefixed with "error: ". Like [IO.Warn] it
// prints in every mode.
//
// It reports a problem. It does not decide the exit code, which belongs to
// whoever is running the command.
func (c *IO) Error(format string, a ...any) {
	c.say(styleRed, "error: ", format, a...)
}

// Debug writes a line to stderr, prefixed with "debug: ", when the command was
// run with -v or more.
func (c *IO) Debug(format string, a ...any) {
	if c.verbosity >= Verbose && !c.jsonMode {
		c.say(styleDim, "debug: ", format, a...)
	}
}

// JSON writes v to stdout as indented JSON, followed by a newline.
//
// It writes in every mode. A caller reaching for it has already decided what
// the output is, unlike [IO.Table], which renders the same data either way.
//
// HTML metacharacters are left alone. Escaping them is a habit from writing
// JSON into a script tag, and terminal output is not that.
func (c *IO) JSON(v any) error {
	enc := json.NewEncoder(c.out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// decorated reports whether output meant for a person should be written at all.
func (c *IO) decorated() bool { return c.verbosity > Quiet && !c.jsonMode }

// say writes one line to stderr.
func (c *IO) say(s style, prefix, format string, a ...any) {
	fmt.Fprintln(c.err, s.wrap(prefix+fmt.Sprintf(format, a...), c.colorErr))
}
