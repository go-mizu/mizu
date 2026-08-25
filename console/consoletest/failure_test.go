package consoletest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// newBuffer is a stream with output already on it, for the parts of the script
// that read a question back off one.
func newBuffer(written string) *bytes.Buffer { return bytes.NewBufferString(written) }

// recorder stands in for a *testing.T so that the assertions in this package
// can be tested the way anything else is: run it, and read what came out.
//
// Embedding testing.TB keeps it a TB as the interface grows, and the methods
// this package uses are all here. A method it does not use panics on the nil,
// which is the report anybody would want if that changed.
type recorder struct {
	testing.TB
	msgs  []string
	fatal bool
}

func (r *recorder) Helper() {}

func (r *recorder) Error(args ...any) { r.msgs = append(r.msgs, fmt.Sprint(args...)) }

func (r *recorder) Errorf(format string, args ...any) {
	r.msgs = append(r.msgs, fmt.Sprintf(format, args...))
}

func (r *recorder) Fatal(args ...any) {
	r.fatal = true
	r.msgs = append(r.msgs, fmt.Sprint(args...))
}

// only returns the one failure that was reported.
func (r *recorder) only(t *testing.T) string {
	t.Helper()
	if len(r.msgs) != 1 {
		t.Fatalf("reported %d failures, want 1: %q", len(r.msgs), r.msgs)
	}
	return r.msgs[0]
}

// contains asserts the failure says all of these things, in no particular
// order, since what matters is that the message has the facts in it.
func contains(t *testing.T, msg string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(msg, w) {
			t.Errorf("the failure does not mention %q:\n%s", w, msg)
		}
	}
}

func TestTheFirstFailurePrintsEverything(t *testing.T) {
	tb := &recorder{}
	r := Run(tb, &fails{err: errors.New("no such tenant")})
	r.AssertSuccess()
	r.AssertNoOutput()

	if len(tb.msgs) != 1 {
		t.Fatalf("reported %d failures, want the one that failed: %q", len(tb.msgs), tb.msgs)
	}
	contains(t, tb.msgs[0],
		"fails: failed with no such tenant, want it to succeed",
		"exit 1, error no such tenant",
		"out (nothing)",
		"err trying",
		"err error: no such tenant",
	)
}

func TestLaterFailuresPrintTheAssertionAlone(t *testing.T) {
	tb := &recorder{}
	r := Run(tb, &greet{Name: "Ada"})
	r.AssertNoOutput()
	r.AssertNoErrorOutput()

	if len(tb.msgs) != 2 {
		t.Fatalf("reported %d failures, want 2: %q", len(tb.msgs), tb.msgs)
	}
	if strings.Contains(tb.msgs[1], "exit 0") {
		t.Errorf("the second failure repeats the output:\n%s", tb.msgs[1])
	}
	contains(t, tb.msgs[1], `stderr is "said hello\n", want nothing`)
}

func TestFailedAssertions(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Result)
		want []string
	}{
		{
			name: "a command that was meant to fail",
			run:  func(r *Result) { r.AssertFailure() },
			want: []string{"greet: succeeded, want it to fail"},
		},
		{
			name: "the wrong exit code",
			run:  func(r *Result) { r.AssertExitCode(console.CodeConfig) },
			want: []string{"exits 0, want 78"},
		},
		{
			name: "output that is not what was expected",
			run:  func(r *Result) { r.AssertOutput("Hi\n") },
			want: []string{`got:  "Hello, Ada\n"`, `want: "Hi\n"`},
		},
		{
			name: "output missing a string",
			run:  func(r *Result) { r.AssertOutputContains("Goodbye") },
			want: []string{`the output does not contain "Goodbye"`},
		},
		{
			name: "output where none was wanted",
			run:  func(r *Result) { r.AssertNoOutput() },
			want: []string{`the output is "Hello, Ada\n", want nothing`},
		},
		{
			name: "stderr missing a string",
			run:  func(r *Result) { r.AssertErrorContains("warning") },
			want: []string{`nothing on stderr contains "warning"`},
		},
		{
			name: "questions that were never asked",
			run:  func(r *Result) { r.AssertAsked("Name") },
			want: []string{"the questions asked are not the ones expected", "got: nothing", `want:`, `"Name"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &recorder{}
			tt.run(Run(tb, &greet{Name: "Ada"}))
			contains(t, tb.only(t), tt.want...)
		})
	}
}

func TestAQuestionWithNoAnswer(t *testing.T) {
	tb := &recorder{}
	r := Run(tb, &setup{}, Answer("Project name", "blog"))

	contains(t, tb.only(t), `the command asked "Database [1]" and the script has no answer for it`)
	if err := r.Err(); !errors.Is(err, console.ErrAborted) {
		t.Errorf("the command got %v, want it to unwind like a closed terminal", err)
	}
	r.AssertAsked("Project name", "Database")
}

func TestAnAnswerNothingAsked(t *testing.T) {
	tb := &recorder{}
	Run(tb, &greet{Name: "Ada"}, Answer("Project name", "blog"))

	contains(t, tb.only(t), `the script answers "Project name" and nothing asked it`)
}

func TestAnAnswerToTheWrongQuestion(t *testing.T) {
	tb := &recorder{}
	Run(tb, &setup{}, Answer("Database", "postgres"))

	contains(t, tb.only(t), `the script answers "Database" and the question was "Project name"`)
}

func TestConfirmingSomethingThatIsNotAYesOrNoQuestion(t *testing.T) {
	tb := &recorder{}
	Run(tb, &setup{}, Confirm("Project name", true))

	contains(t, tb.only(t), `the question "Project name" is not a yes or no question`)
}

func TestChoosingSomethingNotOnTheList(t *testing.T) {
	tb := &recorder{}
	Run(tb, &setup{},
		Answer("Project name", "blog"),
		Choose("Database", "oracle"),
	)

	contains(t, tb.only(t), `the question "Database" offers "sqlite", "postgres", "mysql", and not "oracle"`)
}

func TestChoosingFromSomethingThatIsNotAList(t *testing.T) {
	tb := &recorder{}
	Run(tb, &setup{}, Choose("Project name", "blog"))

	contains(t, tb.only(t), `is not a list to choose from, so there is no "blog" to pick`)
}

func TestChoosingSeveralWhenOneIsNotOnTheList(t *testing.T) {
	tb := &recorder{}
	Run(tb, &setup{},
		Answer("Project name", "blog"),
		Choose("Database", "sqlite"),
		ChooseAll("What else", "queue", "docs"),
	)

	contains(t, tb.only(t), `and not "docs"`)
}

// The first thing that went wrong is the one worth reporting. A command that
// carries on after the stream has ended asks again, and the second question is
// unanswerable for a reason that says nothing about the first.
func TestOnlyTheFirstProblemIsReported(t *testing.T) {
	tb := &recorder{}
	r := Run(tb, &deaf{})

	contains(t, tb.only(t), `the command asked "First" and the script has no answer for it`)
	r.AssertAsked("First", "Second")
}

func TestInputAndAnswersTogether(t *testing.T) {
	tb := &recorder{}
	if r := Run(tb, &echo{}, Input("x"), Answer("Name", "Ada")); r != nil {
		t.Error("a script that cannot work ran anyway")
	}
	if !tb.fatal {
		t.Error("the test was not stopped")
	}
	contains(t, tb.only(t), "stdin is either the text from Input or the scripted answers")
}

// A stream that a command reads in small pieces is still one answer per
// question, which is what a prompt reading a line at a time needs.
func TestAnAnswerIsHandedOverInPieces(t *testing.T) {
	s := &script{out: newBuffer("Name: "), steps: []step{{
		want:  "Name",
		reply: func(Prompt) (string, error) { return "Ada", nil },
	}}}

	got := make([]byte, 0, 4)
	buf := make([]byte, 2)
	for {
		n, err := s.Read(buf)
		got = append(got, buf[:n]...)
		if err != nil {
			break
		}
	}
	if string(got) != "Ada\n" {
		t.Errorf("read %q, want the answer and a newline", got)
	}
}

// A command's own output that starts with a number is not a list to choose
// from, however much the line looks like one.
func TestNumberedOutputIsNotAList(t *testing.T) {
	tests := []struct {
		name  string
		block string
		want  []string
	}{
		{"a list", "  1  one\n  2  two\nPick: ", []string{"one", "two"}},
		{"numbering that does not start at one", "  2  two\nPick: ", nil},
		{"a number with one space after it", "1 one\nPick: ", nil},
		{"a number too big to be one", "99999999999999999999  one\nPick: ", nil},
		{"no number at all", "one\nPick: ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &script{out: newBuffer(tt.block)}
			if got := s.pending().Options; !equal(got, tt.want) {
				t.Errorf("the options are %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAVeryLongStreamIsCutDown(t *testing.T) {
	tb := &recorder{}
	Run(tb, &loud{}).AssertNoOutput()

	msg := tb.only(t)
	contains(t, msg, "bytes more")
	if len(msg) > 8<<10 {
		t.Errorf("the failure is %d bytes long", len(msg))
	}
}

// loud writes more than a failure should print.
type loud struct{}

func (c *loud) Spec() console.Spec { return console.Spec{Name: "loud"} }

func (c *loud) Run(ctx context.Context, out *console.IO) error {
	out.Print("%s", strings.Repeat("data ", 2000))
	return nil
}
