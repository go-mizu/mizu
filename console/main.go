package console

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Exit codes, from sysexits.h, which is what a shell script and a process
// supervisor already know how to read.
//
// A command returns an error and this file turns it into one of these. A
// command that needs a particular one wraps its error with [Exit].
const (
	// CodeOK is a command that did what it was asked.
	CodeOK = 0

	// CodeFailure is anything that went wrong while it was doing it. Most
	// errors are this one.
	CodeFailure = 1

	// CodeUsage is a command line that could not be understood, so nothing ran.
	// Every [UsageError] is this.
	CodeUsage = 2

	// CodeUnavailable is something the command depends on not being there: the
	// database is down, the queue is not answering.
	CodeUnavailable = 69

	// CodeInternal is the command's own bug rather than the caller's.
	CodeInternal = 70

	// CodeNoPermission is a file, a socket or an API that said no.
	CodeNoPermission = 77

	// CodeConfig is a configuration that does not make sense, which is the one
	// worth separating because it is fixed in a different place from the rest.
	CodeConfig = 78

	// CodeInterrupted is Ctrl-C, a SIGTERM, or a person answering no to a
	// question. 128 plus the signal number is what a shell reports for the
	// first two, and a refusal is the same event with a politer interface.
	CodeInterrupted = 130
)

// An ExitCoder is an error that knows what the process should exit with.
//
// This is the seam for the packages this one cannot import. An error type that
// already classifies itself, which is what the errs package is for, implements
// this method and the codes above come out without console knowing anything
// about it.
type ExitCoder interface {
	error
	ExitCode() int
}

// Exit wraps err so that it exits with code.
//
//	return console.Exit(console.CodeConfig, err)
func Exit(code int, err error) error {
	return &exitError{code: code, err: err}
}

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }
func (e *exitError) ExitCode() int { return e.code }

// Globals are the flags every command takes, whatever it does.
//
// They shape what the command says rather than what it does, which is why they
// live here and not on the commands. A program adds its own with [App.Globals],
// for the ones that are about the application rather than the terminal.
type Globals struct {
	Verbose       int
	Quiet         bool
	JSON          bool
	DiagFile      string
	Color         Color
	NoColor       bool
	NoInteraction bool
	Timeout       time.Duration
}

// Flags returns the flags that fill g.
func (g *Globals) Flags() []Flag {
	return []Flag{
		{Name: "verbose", Short: 'v', Desc: "Say more about what is happening, twice for more again", Value: Count(&g.Verbose)},
		{Name: "quiet", Short: 'q', Desc: "Warnings and errors only", Value: Bool(&g.Quiet)},
		{Name: "json", Desc: "Machine readable output", Value: Bool(&g.JSON)},
		{Name: "diag-file", Env: "MIZU_DIAG_FILE", Desc: "Also write diagnostics as JSON to this file", Value: String(&g.DiagFile)},
		{Name: "color", Default: "auto", Desc: "When to colour output: auto, always or never", Value: Var(&g.Color, ParseColor)},
		{Name: "no-color", Desc: "Never colour output", Value: Bool(&g.NoColor)},
		{Name: "no-interaction", Short: 'n', Desc: "Never ask a question, take the defaults", Value: Bool(&g.NoInteraction)},
		{Name: "timeout", Desc: "Give up after this long", Value: Duration(&g.Timeout)},
	}
}

// Options is what the flags add up to.
func (g *Globals) Options() Options {
	opts := Options{
		Verbosity: Verbosity(g.Verbose),
		JSON:      g.JSON,
		DiagFile:  g.DiagFile,
		Color:     g.Color,
	}
	// --quiet beats -v, because somebody who passed both means the one that
	// asks for less, and the alternative is arguing with a shell alias.
	if g.Quiet {
		opts.Verbosity = Quiet
	}
	if g.NoColor {
		opts.Color = ColorNever
	}
	if g.NoInteraction {
		opts.Interaction = InteractionNever
	}
	return opts
}

// Main runs a command line against the process's own streams and returns the
// code to exit with.
//
//	func main() {
//		os.Exit(app.Main(os.Args[1:]))
//	}
//
// SIGINT and SIGTERM cancel the command's context, so a command that honours it
// gets to close what it opened. A second signal exits at once, because somebody
// pressing Ctrl-C twice has stopped asking.
func (a *App) Main(argv []string) int {
	ctx, stop := interrupt(os.Stderr, func() { os.Exit(CodeInterrupted) })
	defer stop()

	return a.Start(ctx, os.Stdin, os.Stdout, os.Stderr, argv)
}

// interrupt returns a context that is cancelled by the first SIGINT or SIGTERM,
// and calls now on the second. The returned function undoes both.
//
// Two channels rather than one: the context carries the first signal, and sigs
// carries every signal, so the second one arrives whatever the command is doing
// about the first. That is what makes a command which ignores its context still
// stoppable, and a shutdown that has itself hung is exactly when somebody
// reaches for Ctrl-C again.
func interrupt(w io.Writer, now func()) (context.Context, func()) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		select {
		case <-sigs:
		case <-done:
			return
		}
		fmt.Fprintln(w, "shutting down, press Ctrl-C again to stop now")
		select {
		case <-sigs:
			now()
		case <-done:
		}
	}()

	return ctx, func() {
		close(done)
		signal.Stop(sigs)
		stop()
	}
}

// Start runs a command line on the given streams and returns the code to exit
// with. It is [App.Main] without the process: no signals, no os.Exit, and the
// streams are whatever a test hands it.
func (a *App) Start(ctx context.Context, in io.Reader, out, err io.Writer, argv []string) int {
	var g Globals
	globals := append(g.Flags(), a.Globals...)
	a.globals = globals

	// The global flags come out first, wherever they were written, so that
	// --json means the same thing before the command name and after it, and so
	// that what is left is the command's own business.
	kept, taken := strip(globals, argv)
	perr := Parse(globals, nil, taken)

	c := New(in, out, err, g.Options())
	if perr != nil {
		return Report(c, perr)
	}

	if g.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.Timeout)
		defer cancel()
	}
	return Report(c, a.Run(ctx, c, kept))
}

// strip takes the global flags out of a command line, wherever they appear, and
// returns what is left and what was taken.
//
// Pulling them out first is what lets a command be parsed against its own flags
// and nothing else. The alternative, parsing the globals twice, applies their
// defaults twice, and a flag that collects rather than replaces ends up with
// everything in it twice.
func strip(globals []Flag, argv []string) (kept, taken []string) {
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch {
		case arg == "--":
			// Everything past this is the command's, whatever it looks like.
			return append(kept, argv[i:]...), taken

		case !isGlobal(globals, arg):
			kept = append(kept, arg)

		default:
			taken = append(taken, arg)
			if wantsValue(globals, arg) && i+1 < len(argv) {
				i++
				taken = append(taken, argv[i])
			}
		}
	}
	return kept, taken
}

// isGlobal reports whether a token belongs to the global flags.
//
// A cluster is global when every letter in it is, so -vq is taken and -vf, with
// a command's own -f in it, is left alone for the command to make sense of. The
// alternative is splitting a cluster in half, which means a command line that
// half worked.
func isGlobal(globals []Flag, arg string) bool {
	switch {
	case strings.HasPrefix(arg, "--"):
		name, _, _ := strings.Cut(arg[2:], "=")
		if find(globals, func(f Flag) bool { return f.Name == name }) >= 0 {
			return true
		}
		off, ok := strings.CutPrefix(name, "no-")
		return ok && find(globals, func(f Flag) bool { return f.Name == off && isBool(f.Value) }) >= 0

	case len(arg) > 1 && arg[0] == '-' && !negative(arg):
		letters, _, _ := strings.Cut(arg[1:], "=")
		for _, r := range letters {
			if find(globals, func(f Flag) bool { return f.Short == r }) < 0 {
				return false
			}
		}
		return true
	}
	return false
}

// wantsValue reports whether a global flag token takes the word after it.
func wantsValue(globals []Flag, arg string) bool {
	if strings.Contains(arg, "=") {
		return false
	}
	if name, ok := strings.CutPrefix(arg, "--"); ok {
		i := find(globals, func(f Flag) bool { return f.Name == name })
		return i >= 0 && !isBool(globals[i].Value)
	}
	// In a cluster only the last letter can take one, because anything after it
	// in the token is the value.
	last := []rune(arg)[len([]rune(arg))-1]
	i := find(globals, func(f Flag) bool { return f.Short == last })
	return i >= 0 && !isBool(globals[i].Value)
}

// errTimedOut is what --timeout is reported as.
//
// An error rather than a string because [Report] hands it to the same path as
// every other failure, and that path wants something [diag.Of] can turn into a
// document. context.DeadlineExceeded says "context deadline exceeded", which is
// Go's phrasing of somebody else's problem.
var errTimedOut = errors.New("timed out")

// Report prints what a command returned, if anything went wrong, and returns
// the code to exit with.
//
// It is the last thing [App.Main] and [App.Start] do, exported for a program
// whose main runs one command without an App around it and for a test fixture
// that wants the code a process would have exited with. A command that returns
// nil prints nothing and gets [CodeOK].
//
// Under --json what it prints is the mizu.diag/1 document, on stderr, whatever
// kind of error came back. With --diag-file, or MIZU_DIAG_FILE, the same
// document also goes to a file, on every run rather than only a failing one.
func Report(c *IO, err error) int {
	// Last, so that a warning about the file itself lands after the report
	// rather than in the middle of it.
	if c.diagFile != "" {
		defer c.writeDiag(err)
	}

	switch {
	case err == nil:
		return CodeOK

	case errors.Is(err, ErrAborted), errors.Is(err, context.Canceled):
		// Whoever pressed Ctrl-C or typed n knows what happened, and a message
		// under it reads as though something else went wrong.
		return CodeInterrupted

	case errors.Is(err, context.DeadlineExceeded):
		c.fail(errTimedOut)
		return CodeFailure
	}

	c.fail(err)

	// The chain is where the answer usually is: a config error three wraps down
	// says which file, and the top of the chain says which step. It costs a
	// flag rather than four lines on every failure.
	//
	// Not under --json, where the document already carries the whole error and
	// a line beside it would not be part of any document.
	if c.Verbosity() >= Verbose && !c.jsonMode {
		for cause := errors.Unwrap(err); cause != nil; cause = errors.Unwrap(cause) {
			c.Debug("caused by: %v", cause)
		}
	}

	var usage *UsageError
	var coder ExitCoder
	switch {
	case errors.As(err, &usage):
		return CodeUsage
	case errors.As(err, &coder):
		return coder.ExitCode()
	}
	return CodeFailure
}
