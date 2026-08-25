package consoletest

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// defaultWidth is the terminal a test gets. It is fixed rather than asked for,
// so that a table wraps the same way on a laptop and on a build machine.
const defaultWidth = 80

// settings are what the options add up to.
type settings struct {
	steps []step
	argv  []string
	parse bool
	input string
	stdin bool
	ctx   context.Context
	opts  console.Options
}

// An Option changes how a command is run. The scripted answers, [Answer],
// [Confirm], [Choose] and [ChooseAll], are options too, so a whole test reads
// as one call.
type Option func(*settings)

// Args is the command line to parse into the command before running it, without
// the command's own name.
//
//	consoletest.Run(t, &UsersPrune{}, consoletest.Args("--days", "7", "acme"))
//
// Without it the command runs on the fields it was built with and nothing is
// parsed, which is the shorter way to write a test that is not about the
// parsing. Note that this means defaults are not applied either: to run a
// command the way an empty command line would, pass [Args] with nothing in it.
//
// Global flags such as --json are not parsed here, because a command never sees
// one. Ask for what they would have done with [With].
func Args(argv ...string) Option {
	return func(s *settings) { s.argv, s.parse = argv, true }
}

// Input is text for a command that reads its stdin rather than asking
// questions, such as one taking a document to validate.
//
// It cannot be combined with scripted answers, since stdin is one stream and
// the two would be reading over each other.
func Input(text string) Option {
	return func(s *settings) { s.input, s.stdin = text, true }
}

// With sets the options the [console.IO] is built with, for a test about what a
// command says rather than what it does: --json, --quiet, a narrower terminal.
//
//	consoletest.Run(t, &UsersPrune{}, consoletest.With(console.Options{JSON: true}))
//
// Anything left at auto keeps what a test wants: prompts ask, colour is off,
// and the terminal is 80 columns.
func With(opts console.Options) Option {
	return func(s *settings) { s.opts = opts }
}

// Context is the context the command is run with, for a command that reads the
// clock, a deadline or a value from it. The default is
// [context.Background].
func Context(ctx context.Context) Option {
	return func(s *settings) { s.ctx = ctx }
}

// Run runs one command and returns what happened.
//
// The command's Run is an ordinary method taking a [console.IO], so this starts
// no process and opens nothing. The streams are buffers, the answers to any
// questions come from the script, and the exit code is the one a process would
// have exited with.
//
//	r := consoletest.Run(t, &UsersPrune{Days: 7, DryRun: true},
//		consoletest.Confirm("Delete 3 users?", true))
//	r.AssertSuccess()
//	r.AssertOutputContains("Pruned")
//
// A question the script has no answer for fails the test, and so does an answer
// nothing asked for. Both are reported once the command has finished, with
// everything it wrote, rather than from inside the read that found them.
func Run(tb testing.TB, cmd console.Command, opts ...Option) *Result {
	tb.Helper()

	s := settings{ctx: context.Background()}
	for _, opt := range opts {
		opt(&s)
	}
	// Auto means ask the terminal, and there is no terminal here. A test that
	// says something other than auto gets what it said.
	if s.opts.Interaction == console.InteractionAuto {
		s.opts.Interaction = console.InteractionAlways
	}
	if s.opts.Color == console.ColorAuto {
		s.opts.Color = console.ColorNever
	}
	if s.opts.Width == 0 {
		s.opts.Width = defaultWidth
	}
	if s.stdin && len(s.steps) > 0 {
		tb.Fatal("consoletest: stdin is either the text from Input or the scripted answers, not both")
		return nil
	}

	r := &Result{tb: tb, name: name(cmd)}
	in := io.Reader(strings.NewReader(s.input))
	if !s.stdin {
		r.script = &script{out: &r.stderr, steps: s.steps}
		in = r.script
	}
	c := console.New(in, &r.stdout, &r.stderr, s.opts)

	var err error
	if s.parse {
		spec := cmd.Spec()
		err = console.Parse(spec.Flags, spec.Args, s.argv)
	}
	if err == nil {
		err = cmd.Run(s.ctx, c)
	}
	r.err = err
	r.code = console.Report(c, err)

	r.check()
	return r
}

// name is what a failure calls the command.
func name(cmd console.Command) string {
	if n := cmd.Spec().Name; n != "" {
		return n
	}
	return "the command"
}
