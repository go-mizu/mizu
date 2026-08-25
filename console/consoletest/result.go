package consoletest

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// dumpLimit is how much of a stream a failure prints. A command that wrote a
// megabyte has a problem the first two kilobytes will show.
const dumpLimit = 2 << 10

// Result is what a command did, and the assertions about it.
//
// Every assertion returns the same Result, so they chain, and every one of them
// reports a failure rather than stopping the test. The first failure prints
// everything the command wrote and the ones after it print the assertion alone,
// because output that is wrong is usually wrong for one reason.
type Result struct {
	tb     testing.TB
	name   string
	script *script
	stdout bytes.Buffer
	stderr bytes.Buffer
	err    error
	code   int

	shown bool // the output has been printed once already
}

// Code is what a process running this command would have exited with.
func (r *Result) Code() int { return r.code }

// Err is the error the command returned, which is nil when it did what it was
// asked. It is the error itself, so a test can reach for [errors.Is] and
// [errors.As] rather than matching on the message.
func (r *Result) Err() error { return r.err }

// Stdout is the data the command wrote.
func (r *Result) Stdout() string { return r.stdout.String() }

// Stderr is everything else it wrote: status lines, warnings, questions, and
// the error that ended it.
func (r *Result) Stderr() string { return r.stderr.String() }

// Prompts are the questions the command asked, in order. It is empty for a
// command run with [Input], since there was nothing watching the stream to
// notice them.
func (r *Result) Prompts() []Prompt {
	if r.script == nil {
		return nil
	}
	return r.script.asked
}

// AssertSuccess asserts the command did what it was asked, which is to say it
// returned no error and would have exited zero.
func (r *Result) AssertSuccess() *Result {
	r.tb.Helper()
	if r.err != nil {
		r.fail("failed with %v, want it to succeed", r.err)
	}
	return r
}

// AssertFailure asserts the command failed, and returns the error it failed
// with so a test can go on and say more about it.
//
//	if err := r.AssertFailure(); !errors.Is(err, os.ErrNotExist) { ... }
func (r *Result) AssertFailure() error {
	r.tb.Helper()
	if r.err == nil {
		r.fail("succeeded, want it to fail")
	}
	return r.err
}

// AssertExitCode asserts an exact exit code. See the Code constants in the
// console package for what they mean.
func (r *Result) AssertExitCode(want int) *Result {
	r.tb.Helper()
	if r.code != want {
		r.fail("exits %d, want %d", r.code, want)
	}
	return r
}

// AssertOutput asserts stdout is exactly this, byte for byte.
func (r *Result) AssertOutput(want string) *Result {
	r.tb.Helper()
	if got := r.Stdout(); got != want {
		r.fail("the output is not what was expected\ngot:  %q\nwant: %q", got, want)
	}
	return r
}

// AssertOutputContains asserts stdout contains a string.
func (r *Result) AssertOutputContains(want string) *Result {
	r.tb.Helper()
	if !strings.Contains(r.Stdout(), want) {
		r.fail("the output does not contain %q", want)
	}
	return r
}

// AssertNoOutput asserts the command wrote no data at all, which is what a
// command that only did something should do.
func (r *Result) AssertNoOutput() *Result {
	r.tb.Helper()
	if got := r.Stdout(); got != "" {
		r.fail("the output is %q, want nothing", clip(got))
	}
	return r
}

// AssertErrorContains asserts stderr contains a string, which covers a warning
// and a status line as well as the error itself.
func (r *Result) AssertErrorContains(want string) *Result {
	r.tb.Helper()
	if !strings.Contains(r.Stderr(), want) {
		r.fail("nothing on stderr contains %q", want)
	}
	return r
}

// AssertNoErrorOutput asserts the command said nothing on stderr: no warning,
// no status line, no error.
func (r *Result) AssertNoErrorOutput() *Result {
	r.tb.Helper()
	if got := r.Stderr(); got != "" {
		r.fail("stderr is %q, want nothing", clip(got))
	}
	return r
}

// AssertAsked asserts the questions the command asked, in order and all of
// them.
//
// The comparison is against the question alone, without the default in brackets
// and without the colon after it, so a test says what it was asked rather than
// matching a substring of a terminal dump.
func (r *Result) AssertAsked(questions ...string) *Result {
	r.tb.Helper()
	prompts := r.Prompts()
	got := make([]string, 0, len(prompts))
	for _, p := range prompts {
		got = append(got, p.Question)
	}
	if !slices.Equal(got, questions) {
		r.fail("the questions asked are not the ones expected\ngot:%s\nwant:%s", lines(got), lines(questions))
	}
	return r
}

// check reports what went wrong with the script itself, which is a mistake in
// the test rather than in the command.
func (r *Result) check() {
	r.tb.Helper()
	if r.script == nil {
		return
	}
	if r.script.problem != nil {
		r.fail("%v", r.script.problem)
		return
	}
	if left := r.script.steps[r.script.at:]; len(left) > 0 {
		r.fail("the script answers %q and nothing asked it", left[0].want)
	}
}

// fail reports a failure, with everything the command wrote under the first
// one.
func (r *Result) fail(format string, args ...any) {
	r.tb.Helper()
	msg := r.name + ": " + fmt.Sprintf(format, args...)
	if r.shown {
		r.tb.Error(msg)
		return
	}
	r.shown = true
	r.tb.Errorf("%s\n%s", msg, indent(r.dump()))
}

// dump is what the command wrote, with each stream named, for reading under a
// failure.
func (r *Result) dump() string {
	var b strings.Builder
	fmt.Fprintf(&b, "exit %d", r.code)
	if r.err != nil {
		fmt.Fprintf(&b, ", error %v", r.err)
	}
	b.WriteByte('\n')
	stream(&b, "out", r.Stdout())
	stream(&b, "err", r.Stderr())
	return strings.TrimSuffix(b.String(), "\n")
}

// stream writes one of the two streams, a line at a time so that the name is on
// every line and a blank stream says so.
func stream(b *strings.Builder, name, text string) {
	if text == "" {
		fmt.Fprintf(b, "%s (nothing)\n", name)
		return
	}
	for line := range strings.Lines(clip(text)) {
		fmt.Fprintf(b, "%s %s\n", name, strings.TrimRight(line, "\r\n"))
	}
}

// clip cuts a stream down to what is worth printing.
func clip(s string) string {
	if len(s) <= dumpLimit {
		return s
	}
	return s[:dumpLimit] + fmt.Sprintf("\n... and %d bytes more", len(s)-dumpLimit)
}

// indent puts the dump under the failure, where the testing package's own
// output already is.
func indent(s string) string {
	return "    " + strings.ReplaceAll(s, "\n", "\n    ")
}

// lines writes a list one entry to a line, since a list of questions on one
// line is not something anybody can compare by eye.
func lines(items []string) string {
	if len(items) == 0 {
		return " nothing"
	}
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "\n  %q", item)
	}
	return b.String()
}
