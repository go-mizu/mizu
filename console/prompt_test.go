package console

import (
	"bytes"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

// answers is a scripted conversation: what the user typed, and the streams the
// prompt used.
//
// The interaction mode is always set, because a buffer is not a terminal and
// the whole point of the automatic mode is that it can tell.
func answers(t *testing.T, typed string, opts Options) *streams {
	t.Helper()

	if opts.Interaction == InteractionAuto {
		opts.Interaction = InteractionAlways
	}
	s := &streams{
		in:  strings.NewReader(typed),
		out: &bytes.Buffer{},
		err: &bytes.Buffer{},
	}
	s.io = New(s.in, s.out, s.err, opts)
	return s
}

func TestAsk(t *testing.T) {
	s := answers(t, "Ada\n", Options{})

	got, err := s.io.Ask("Name", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada" {
		t.Errorf("Ask returned %q", got)
	}
	if want := "Name: "; s.err.String() != want {
		t.Errorf("the prompt was %q, want %q", s.err.String(), want)
	}
}

// TestPromptsGoToStderr is the rule that lets a command ask a question and
// still be the left-hand side of a pipe.
func TestPromptsGoToStderr(t *testing.T) {
	s := answers(t, "Ada\n2\ny\n", Options{})

	s.io.Ask("Name", "")
	s.io.Choice("Environment", []string{"dev", "prod"}, -1)
	s.io.Confirm("Sure?", false)

	if got := s.out.String(); got != "" {
		t.Errorf("stdout has %q, and a prompt is not data", got)
	}
	if got := s.err.String(); !strings.Contains(got, "Name: ") {
		t.Errorf("stderr has %q", got)
	}
}

func TestAskTakesTheDefault(t *testing.T) {
	s := answers(t, "\n", Options{})

	got, err := s.io.Ask("Name", "Ada")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada" {
		t.Errorf("an empty answer returned %q, want the default", got)
	}
	if want := "Name [Ada]: "; s.err.String() != want {
		t.Errorf("the prompt was %q, want %q", s.err.String(), want)
	}
}

func TestAskTrimsSpaces(t *testing.T) {
	s := answers(t, "   Ada Lovelace  \n", Options{})

	got, err := s.io.Ask("Name", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada Lovelace" {
		t.Errorf("Ask returned %q", got)
	}
}

// TestAnswerWithNoNewline covers a stream that ends right after the answer,
// which is what an echo without -n and a here-string both produce.
func TestAnswerWithNoNewline(t *testing.T) {
	s := answers(t, "Ada", Options{})

	got, err := s.io.Ask("Name", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada" {
		t.Errorf("Ask returned %q", got)
	}
}

func TestAnswerFromWindows(t *testing.T) {
	s := answers(t, "Ada\r\n", Options{})

	got, err := s.io.Ask("Name", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada" {
		t.Errorf("Ask returned %q, want the carriage return gone", got)
	}
}

// TestPromptsShareTheReader is why the reader lives on the IO. A new buffered
// reader per prompt reads ahead and eats the next answer, which shows up as a
// question the user never got to see.
func TestPromptsShareTheReader(t *testing.T) {
	s := answers(t, "Ada\nBo\n", Options{})

	first, err := s.io.Ask("First", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.io.Ask("Second", "")
	if err != nil {
		t.Fatal(err)
	}
	if first != "Ada" || second != "Bo" {
		t.Errorf("the answers were %q and %q", first, second)
	}
}

// TestEndOfInputIsAnAbort is Ctrl-D, and a closed pipe, and a test that ran out
// of scripted answers.
func TestEndOfInputIsAnAbort(t *testing.T) {
	s := answers(t, "", Options{})

	if _, err := s.io.Ask("Name", ""); !errors.Is(err, ErrAborted) {
		t.Errorf("the end of the input returned %v, want an abort", err)
	}
}

// TestNoTerminalAndNoDefault is the rule that keeps a command from hanging in
// CI until somebody kills the runner.
func TestNoTerminalAndNoDefault(t *testing.T) {
	s := answers(t, "Ada\n", Options{Interaction: InteractionNever})

	_, err := s.io.Ask("Database URL", "")
	if !errors.Is(err, ErrNoInput) {
		t.Fatalf("Ask returned %v, want no input", err)
	}
	if !strings.Contains(err.Error(), "Database URL") {
		t.Errorf("the error is %q, and does not say which question", err)
	}
	if got := s.err.String(); got != "" {
		t.Errorf("stderr has %q, and there was nobody to ask", got)
	}
}

func TestNoTerminalTakesTheDefault(t *testing.T) {
	s := answers(t, "Bo\n", Options{Interaction: InteractionNever})

	got, err := s.io.Ask("Name", "Ada")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada" {
		t.Errorf("Ask returned %q, want the default rather than what was piped in", got)
	}
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		typed string
		def   bool
		want  bool
	}{
		{"y\n", false, true},
		{"Y\n", false, true},
		{"yes\n", false, true},
		{"YES\n", false, true},
		{"n\n", true, false},
		{"no\n", true, false},
		{" y \n", false, true},
		{"\n", false, false},
		{"\n", true, true},
	}
	for _, tt := range tests {
		s := answers(t, tt.typed, Options{})

		got, err := s.io.Confirm("Delete 3 users?", tt.def)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("Confirm(%q) with default %v returned %v", tt.typed, tt.def, got)
		}
	}
}

func TestConfirmShowsWhichWayEnterGoes(t *testing.T) {
	for def, want := range map[bool]string{false: "Delete? [y/N]: ", true: "Delete? [Y/n]: "} {
		s := answers(t, "\n", Options{})

		s.io.Confirm("Delete?", def)

		if got := s.err.String(); got != want {
			t.Errorf("the prompt was %q, want %q", got, want)
		}
	}
}

func TestConfirmAsksAgain(t *testing.T) {
	s := answers(t, "maybe\ny\n", Options{})

	got, err := s.io.Confirm("Sure?", false)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Confirm returned false after the second answer said yes")
	}
	if !strings.Contains(s.err.String(), "answer y or n") {
		t.Errorf("stderr has %q, and does not say what went wrong", s.err.String())
	}
}

// TestConfirmWithNoTerminalTakesTheDefault is why Confirm has one. A command
// guarding something destructive passes false, and the same command in CI
// stops instead of proceeding.
func TestConfirmWithNoTerminalTakesTheDefault(t *testing.T) {
	for _, def := range []bool{true, false} {
		s := answers(t, "y\n", Options{Interaction: InteractionNever})

		got, err := s.io.Confirm("Delete everything?", def)
		if err != nil {
			t.Fatal(err)
		}
		if got != def {
			t.Errorf("Confirm with default %v returned %v", def, got)
		}
	}
}

var environments = []string{"development", "staging", "production"}

func TestChoice(t *testing.T) {
	s := answers(t, "3\n", Options{})

	got, err := s.io.Choice("Environment", environments, -1)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Errorf("Choice returned %d, want the third entry", got)
	}

	want := strings.Join([]string{
		"  1  development",
		"  2  staging",
		"  3  production",
		"Environment: ",
	}, "\n")
	if got := s.err.String(); got != want {
		t.Errorf("the prompt was\n%s\nwant\n%s", got, want)
	}
}

func TestChoiceTakesTheDefault(t *testing.T) {
	s := answers(t, "\n", Options{})

	got, err := s.io.Choice("Environment", environments, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Errorf("Choice returned %d, want the default", got)
	}
	if !strings.HasSuffix(s.err.String(), "Environment [2]: ") {
		t.Errorf("the prompt was %q, and does not show the default", s.err.String())
	}
}

func TestChoiceAsksAgain(t *testing.T) {
	s := answers(t, "0\n4\nlast\n2\n", Options{})

	got, err := s.io.Choice("Environment", environments, -1)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Errorf("Choice returned %d, want the second entry", got)
	}
	if n := strings.Count(s.err.String(), "answer with a number from 1 to 3"); n != 3 {
		t.Errorf("three bad answers produced %d complaints", n)
	}
}

// TestChoiceNumbersTheListEvenly keeps a list of ten from stepping left when it
// reaches double figures.
func TestChoiceNumbersTheListEvenly(t *testing.T) {
	options := make([]string, 10)
	for i := range options {
		options[i] = "option"
	}
	s := answers(t, "1\n", Options{})

	s.io.Choice("Pick", options, -1)

	got := s.err.String()
	if !strings.Contains(got, "   1  option\n") || !strings.Contains(got, "  10  option\n") {
		t.Errorf("the list is\n%s", got)
	}
}

func TestChoiceWithNoTerminal(t *testing.T) {
	s := answers(t, "", Options{Interaction: InteractionNever})

	got, err := s.io.Choice("Environment", environments, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Errorf("Choice returned %d, want the default", got)
	}

	if _, err := s.io.Choice("Environment", environments, -1); !errors.Is(err, ErrNoInput) {
		t.Errorf("a choice with no default returned %v, want no input", err)
	}
}

func TestChoiceWithNothingToChooseFrom(t *testing.T) {
	s := answers(t, "1\n", Options{})

	if _, err := s.io.Choice("Environment", nil, 0); err == nil {
		t.Error("a choice with no options was accepted")
	}
	if _, err := s.io.MultiChoice("Environment", nil); err == nil {
		t.Error("a multiple choice with no options was accepted")
	}
}

func TestMultiChoice(t *testing.T) {
	s := answers(t, "3, 1,1\n", Options{})

	got, err := s.io.MultiChoice("Include", environments)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{0, 2}; !slices.Equal(got, want) {
		t.Errorf("MultiChoice returned %v, want %v", got, want)
	}
}

// TestMultiChoiceTakesNothing covers the empty answer, which is a real answer
// and not an abort.
func TestMultiChoiceTakesNothing(t *testing.T) {
	s := answers(t, "\n", Options{})

	got, err := s.io.MultiChoice("Include", environments)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("MultiChoice returned %v, want nothing", got)
	}
}

func TestMultiChoiceAsksAgain(t *testing.T) {
	s := answers(t, "1,9\n2\n", Options{})

	got, err := s.io.MultiChoice("Include", environments)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{1}; !slices.Equal(got, want) {
		t.Errorf("MultiChoice returned %v, want %v", got, want)
	}
	if !strings.Contains(s.err.String(), "separated by commas") {
		t.Errorf("stderr has %q", s.err.String())
	}
}

// TestMultiChoiceWithNoTerminalFails is the one prompt with no default. A
// scaffolder in CI says which flag it wanted rather than generating a project
// with none of the parts in it.
func TestMultiChoiceWithNoTerminalFails(t *testing.T) {
	s := answers(t, "1\n", Options{Interaction: InteractionNever})

	if _, err := s.io.MultiChoice("Include", environments); !errors.Is(err, ErrNoInput) {
		t.Errorf("MultiChoice returned %v, want no input", err)
	}
}

// TestAskSecretKeepsTheSpaces is the difference between a secret and an answer.
// A password with a space on the end of it is still that password.
func TestAskSecretKeepsTheSpaces(t *testing.T) {
	s := answers(t, "  hunter2  \n", Options{})

	got, err := s.io.AskSecret("Password")
	if err != nil {
		t.Fatal(err)
	}
	if got != "  hunter2  " {
		t.Errorf("AskSecret returned %q", got)
	}
}

func TestAskSecretWithNoTerminalFails(t *testing.T) {
	s := answers(t, "hunter2\n", Options{Interaction: InteractionNever})

	if _, err := s.io.AskSecret("Password"); !errors.Is(err, ErrNoInput) {
		t.Errorf("AskSecret returned %v, want no input", err)
	}
}

func TestInteractive(t *testing.T) {
	tests := []struct {
		mode Interaction
		want bool
	}{
		{InteractionAuto, false}, // a buffer is not a terminal
		{InteractionNever, false},
		{InteractionAlways, true},
	}
	for _, tt := range tests {
		s := newStreams(t, Options{Interaction: tt.mode})
		if got := s.io.Interactive(); got != tt.want {
			t.Errorf("interaction %v gave Interactive() = %v", tt.mode, got)
		}
	}
}

// TestPromptsNeedBothStreams covers the half a conversation case: input from a
// pipe, output on a terminal. Nothing is going to answer, so nothing asks.
func TestPromptsNeedBothStreams(t *testing.T) {
	if !isTerminal(os.Stderr) {
		t.Skip("stderr is not a terminal here")
	}
	if canPrompt(strings.NewReader("Ada\n"), os.Stderr, InteractionAuto) {
		t.Error("a command reading from a pipe was told it could ask questions")
	}
}

// TestReadError separates the two ways an answer fails to arrive. The stream
// ending is a user who walked away, and anything else is a broken stream,
// which are different things to report.
func TestReadError(t *testing.T) {
	if got := readError(io.EOF); !errors.Is(got, ErrAborted) {
		t.Errorf("the end of the input became %v, want an abort", got)
	}
	broken := errors.New("read |0: file already closed")
	if got := readError(broken); !errors.Is(got, broken) {
		t.Errorf("a broken stream became %v", got)
	}
}

func TestInteractionString(t *testing.T) {
	tests := map[Interaction]string{
		InteractionAuto:   "auto",
		InteractionNever:  "never",
		InteractionAlways: "always",
		Interaction(9):    "auto",
	}
	for mode, want := range tests {
		if got := mode.String(); got != want {
			t.Errorf("Interaction(%d).String() = %q, want %q", mode, got, want)
		}
	}
}

func TestQuestionsAreBold(t *testing.T) {
	s := answers(t, "Ada\n", Options{Color: ColorAlways})

	s.io.Ask("Name", "")

	if want := "\x1b[1mName\x1b[0m: "; s.err.String() != want {
		t.Errorf("the prompt was %q, want %q", s.err.String(), want)
	}
}
