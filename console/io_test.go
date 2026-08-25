package console

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

// streams is the three buffers a test writes through, plus the IO on them.
type streams struct {
	in       *strings.Reader
	out, err *bytes.Buffer
	io       *IO
}

func newStreams(t *testing.T, opts Options) *streams {
	t.Helper()

	s := &streams{
		in:  strings.NewReader(""),
		out: &bytes.Buffer{},
		err: &bytes.Buffer{},
	}
	s.io = New(s.in, s.out, s.err, opts)
	return s
}

// TestDataGoesToStdout is the rule the whole package rests on. A command that
// writes its answer to stderr cannot be the left-hand side of a pipe, and
// nobody finds out until a script breaks.
func TestDataGoesToStdout(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Line("ada@example.com")
	s.io.Print("%d users\n", 3)

	if got := s.out.String(); got != "ada@example.com\n3 users\n" {
		t.Errorf("stdout has %q", got)
	}
	if got := s.err.String(); got != "" {
		t.Errorf("stderr has %q, want the data to have gone to stdout", got)
	}
}

// TestEverythingElseGoesToStderr is the other half of it.
func TestEverythingElseGoesToStderr(t *testing.T) {
	s := newStreams(t, Options{Verbosity: Verbose})

	s.io.Info("loading the config")
	s.io.Success("done")
	s.io.Warn("--old is deprecated")
	s.io.Error("could not reach the database")
	s.io.Debug("took 4ms")

	if got := s.out.String(); got != "" {
		t.Errorf("stdout has %q, want nothing but data on it", got)
	}
	want := []string{
		"loading the config\n",
		"done\n",
		"warning: --old is deprecated\n",
		"error: could not reach the database\n",
		"debug: took 4ms\n",
	}
	if got := s.err.String(); got != strings.Join(want, "") {
		t.Errorf("stderr has %q, want %q", got, strings.Join(want, ""))
	}
}

// TestLineDoesNotFormat is why Line takes a string. A subject line, a file
// path, or anything else a user typed can contain a percent sign.
func TestLineDoesNotFormat(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Line("50% off")

	if got := s.out.String(); got != "50% off\n" {
		t.Errorf("Line wrote %q", got)
	}
}

func TestDebugNeedsAFlag(t *testing.T) {
	tests := map[Verbosity]bool{
		Quiet:   false,
		Normal:  false,
		Verbose: true,
		Trace:   true,
	}
	for v, want := range tests {
		s := newStreams(t, Options{Verbosity: v})
		s.io.Debug("took 4ms")

		if got := s.err.Len() > 0; got != want {
			t.Errorf("at verbosity %d Debug printed %v, want %v", v, got, want)
		}
	}
}

// TestQuietKeepsTheProblems is the part of --quiet that is a decision rather
// than an implementation. Silencing a warning is how the flag gets blamed for
// something it did not do.
func TestQuietKeepsTheProblems(t *testing.T) {
	s := newStreams(t, Options{Verbosity: Quiet})

	s.io.Info("loading the config")
	s.io.Success("done")
	s.io.Warn("--old is deprecated")
	s.io.Error("could not reach the database")

	want := "warning: --old is deprecated\nerror: could not reach the database\n"
	if got := s.err.String(); got != want {
		t.Errorf("under --quiet stderr has %q, want %q", got, want)
	}
}

// TestQuietStillAnswersTheQuestion covers the other thing --quiet must not do.
func TestQuietStillAnswersTheQuestion(t *testing.T) {
	s := newStreams(t, Options{Verbosity: Quiet})

	s.io.Line("ada@example.com")

	if got := s.out.String(); got != "ada@example.com\n" {
		t.Errorf("under --quiet stdout has %q, want the data", got)
	}
}

// TestJSONModeDropsTheDecoration keeps status messages out of the way of a
// parser, without dropping the two that report a problem. Those are on stderr,
// where they cannot corrupt anything.
func TestJSONModeDropsTheDecoration(t *testing.T) {
	s := newStreams(t, Options{JSON: true, Verbosity: Verbose})

	s.io.Info("loading the config")
	s.io.Success("done")
	s.io.Debug("took 4ms")
	s.io.Warn("--old is deprecated")

	if got := s.err.String(); got != "warning: --old is deprecated\n" {
		t.Errorf("in JSON mode stderr has %q", got)
	}
}

func TestJSON(t *testing.T) {
	s := newStreams(t, Options{})

	if err := s.io.JSON(map[string]int{"users": 3}); err != nil {
		t.Fatal(err)
	}

	const want = "{\n  \"users\": 3\n}\n"
	if got := s.out.String(); got != want {
		t.Errorf("JSON wrote %q, want %q", got, want)
	}
}

// TestJSONLeavesHTMLAlone is about the output being readable. The escaping the
// standard library does by default is for JSON inside a script tag, and a
// terminal is not that.
func TestJSONLeavesHTMLAlone(t *testing.T) {
	s := newStreams(t, Options{})

	if err := s.io.JSON(map[string]string{"q": "a&b<c"}); err != nil {
		t.Fatal(err)
	}

	if got := s.out.String(); !strings.Contains(got, "a&b<c") {
		t.Errorf("JSON wrote %q, want the ampersand left alone", got)
	}
}

// TestJSONWritesInEveryMode is the difference between JSON and Table. A caller
// reaching for JSON has decided what the output is.
func TestJSONWritesInEveryMode(t *testing.T) {
	s := newStreams(t, Options{Verbosity: Quiet})

	if err := s.io.JSON([]int{1}); err != nil {
		t.Fatal(err)
	}

	if s.out.Len() == 0 {
		t.Error("JSON wrote nothing under --quiet")
	}
}

func TestAccessors(t *testing.T) {
	s := newStreams(t, Options{Verbosity: Trace, JSON: true, Width: 100})

	if s.io.In() != s.in || s.io.Out() != s.out || s.io.Err() != s.err {
		t.Error("the streams came back changed")
	}
	if s.io.Verbosity() != Trace {
		t.Errorf("Verbosity is %d, want %d", s.io.Verbosity(), Trace)
	}
	if !s.io.JSONMode() {
		t.Error("JSONMode is false")
	}
	if s.io.Width() != 100 {
		t.Errorf("Width is %d, want the override", s.io.Width())
	}
}

// TestWidthOfAPipeIsZero is the answer being an answer rather than a guess. A
// caller that needs a number when there is no terminal picks its own.
func TestWidthOfAPipeIsZero(t *testing.T) {
	s := newStreams(t, Options{})

	if got := s.io.Width(); got != 0 {
		t.Errorf("the width of a buffer is %d, want 0", got)
	}
}

func TestStdioUsesTheProcessStreams(t *testing.T) {
	io := Stdio(Options{})

	if io.In() == nil || io.Out() == nil || io.Err() == nil {
		t.Error("Stdio left a stream nil")
	}
}

// TestDiagWritesAReportSomebodyCanRead is the shape a command's findings take
// when a person is looking at them.
func TestDiagWritesAReportSomebodyCanRead(t *testing.T) {
	s := newStreams(t, Options{})

	list := diag.List{{
		Code:     "MZ1042",
		Severity: diag.Error,
		Message:  "a setting is written down that nothing asked for",
		File:     "app.toml",
		Range:    diag.Span(3, 1, 9),
	}}
	if err := s.io.Diag(list.Err()); err != nil {
		t.Fatal(err)
	}

	got := s.out.String()
	for _, want := range []string{"error[MZ1042]", "app.toml:3:1", "mizu explain MZ1042"} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not hold %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b") {
		t.Error("the report carries escapes and nothing here is a terminal")
	}
	if s.err.Len() != 0 {
		t.Errorf("stderr has %q, and the findings are the answer", s.err.String())
	}
}

// TestDiagUnderJSONIsTheDocument is the same call answering a program.
func TestDiagUnderJSONIsTheDocument(t *testing.T) {
	s := newStreams(t, Options{JSON: true})

	list := diag.List{{Code: "MZ1042", Severity: diag.Error, Message: "nothing asked for it"}}
	if err := s.io.Diag(list.Err()); err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(s.out.Bytes(), &doc); err != nil {
		t.Fatalf("stdout is not a document: %v\n%s", err, s.out)
	}
	if len(doc.Diagnostics) != 1 || doc.Diagnostics[0].Code != "MZ1042" {
		t.Errorf("the document holds %+v", doc.Diagnostics)
	}
}

// TestDiagWithNothingToSayIsStillAnAnswer. A run that found nothing is not the
// same as a run that never happened, and a program reading the output should
// not have to tell them apart by whether there was any.
func TestDiagWithNothingToSayIsStillAnAnswer(t *testing.T) {
	s := newStreams(t, Options{JSON: true})

	if err := s.io.Diag(nil); err != nil {
		t.Fatal(err)
	}
	if s.out.Len() == 0 {
		t.Error("nothing was written, so an empty run reads as no run")
	}

	plain := newStreams(t, Options{})
	if err := plain.io.Diag(nil); err != nil {
		t.Fatal(err)
	}
	if plain.out.Len() != 0 {
		t.Errorf("a report with nothing in it printed %q", plain.out)
	}
}
