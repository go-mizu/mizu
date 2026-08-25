package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

// fails is a command that returns whatever it was built with, and writes its
// own answer to stdout first when asked to. The second part is the case the
// stream choice is about: a command that had already printed a document before
// it failed.
type fails struct {
	err    error
	answer any
}

func (c *fails) Spec() Spec { return Spec{Name: "check", Desc: "Check something"} }

func (c *fails) Run(ctx context.Context, io *IO) error {
	if c.answer != nil {
		if err := io.JSON(c.answer); err != nil {
			return err
		}
	}
	return c.err
}

// document reads back what a run wrote, and fails the test if it is not one
// mizu.diag/1 document and nothing else.
func document(t *testing.T, s string) diag.Document {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	var doc diag.Document
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("not a JSON document: %v\n%s", err, s)
	}
	if doc.Schema != diag.Schema {
		t.Errorf("schema is %q, want %q", doc.Schema, diag.Schema)
	}
	if dec.More() {
		t.Errorf("there is more than one document in the stream:\n%s", s)
	}
	return doc
}

func TestJSONModeReportsTheFailureAsADocument(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&fails{err: errors.New("the disk is full")})

	code, _, errOut := start(a, "--json", "check")
	if code != CodeFailure {
		t.Errorf("exit %d, want %d", code, CodeFailure)
	}
	doc := document(t, errOut)
	if len(doc.Diagnostics) != 1 || doc.Diagnostics[0].Message != "the disk is full" {
		t.Fatalf("diagnostics are %+v", doc.Diagnostics)
	}
	if doc.Summary.Errors != 1 {
		t.Errorf("summary counts %d errors, want 1", doc.Summary.Errors)
	}
}

// An ordinary error becomes a document too. That is what makes --json true of
// every command rather than of the ones written with it in mind.
func TestJSONModeReportsAnErrorThatIsNotADiagnostic(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want string
	}{
		{"plain", errors.New("no database"), "no database"},
		{"wrapped", fmt.Errorf("boot: %w", errors.New("no database")), "boot: no database"},
		{"usage", usagef("unknown flag --forse"), "unknown flag --forse"},
		{"timed out", context.DeadlineExceeded, "timed out"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{Name: "mizu"}
			a.Add(&fails{err: tt.err})

			_, _, errOut := start(a, "--json", "check")
			doc := document(t, errOut)
			if len(doc.Diagnostics) == 0 || doc.Diagnostics[0].Message != tt.want {
				t.Errorf("diagnostics are %+v, want a first message of %q", doc.Diagnostics, tt.want)
			}
		})
	}
}

// A command that already had a real diagnostic keeps everything on it: the
// code, the file, the place in the file and the fix.
func TestJSONModeKeepsWhatTheDiagnosticCarried(t *testing.T) {
	d := diag.Diagnostic{
		Code:     "MZ1042",
		Severity: diag.Error,
		Message:  `unknown setting "database.pool_size"`,
		File:     "config/app.toml",
		Range:    diag.Span(14, 1, 9),
		Fix:      "mizu config check",
	}
	a := &App{Name: "mizu"}
	a.Add(&fails{err: fmt.Errorf("reading config: %w", d)})

	_, _, errOut := start(a, "--json", "check")
	doc := document(t, errOut)
	if len(doc.Diagnostics) != 1 {
		t.Fatalf("diagnostics are %+v", doc.Diagnostics)
	}
	got := doc.Diagnostics[0]
	if got.Code != d.Code || got.File != d.File || got.Range != d.Range || got.Fix != d.Fix {
		t.Errorf("the document lost part of the diagnostic: %+v", got)
	}
}

// stdout is the answer and stderr is what went wrong. Both are one document,
// and neither can turn the other into a stream that is not JSON at all.
func TestJSONModeLeavesStdoutToTheAnswer(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&fails{answer: map[string]int{"checked": 3}, err: errors.New("two of them failed")})

	_, out, errOut := start(a, "--json", "check")

	var answer map[string]int
	if err := json.Unmarshal([]byte(out), &answer); err != nil {
		t.Fatalf("stdout is not the answer it was asked for: %v\n%s", err, out)
	}
	if answer["checked"] != 3 {
		t.Errorf("stdout is %q", out)
	}
	document(t, errOut)
}

// -v prints the cause chain for a person to read. Under --json the document
// already carries the error and a line beside it would belong to nothing.
func TestTheCauseChainStaysOutOfTheDocument(t *testing.T) {
	err := fmt.Errorf("reading config/app.toml: %w", errors.New("no such file"))
	a := &App{Name: "mizu"}
	a.Add(&fails{err: err})

	_, _, errOut := start(a, "-v", "--json", "check")
	if strings.Contains(errOut, "caused by") {
		t.Errorf("the chain was printed beside the document:\n%s", errOut)
	}
	document(t, errOut)
}

func TestJSONModeSaysNothingWhenNothingWentWrong(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&fails{})

	code, _, errOut := start(a, "--json", "check")
	if code != CodeOK {
		t.Errorf("exit %d, want 0", code)
	}
	if errOut != "" {
		t.Errorf("a run that worked wrote %q to stderr", errOut)
	}
}

// The human form is untouched. Somebody who did not ask for JSON reads the
// same line they read before.
func TestWithoutJSONTheErrorIsALine(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&fails{err: errors.New("the disk is full")})

	_, _, errOut := start(a, "check")
	if errOut != "error: the disk is full\n" {
		t.Errorf("stderr is %q", errOut)
	}
}

func TestDiagFileIsWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diag.json")

	a := &App{Name: "mizu"}
	a.Add(&fails{err: errors.New("the disk is full")})

	code, _, errOut := start(a, "--diag-file", path, "check")
	if code != CodeFailure {
		t.Errorf("exit %d, want %d", code, CodeFailure)
	}
	// The file is on top of what the command prints, not instead of it.
	if !strings.Contains(errOut, "error: the disk is full") {
		t.Errorf("stderr is %q", errOut)
	}
	doc := document(t, read(t, path))
	if len(doc.Diagnostics) != 1 || doc.Diagnostics[0].Message != "the disk is full" {
		t.Errorf("the file holds %+v", doc.Diagnostics)
	}
}

// MIZU_DIAG_FILE is the same flag by another name, which is how a generator
// invoked through go generate reaches it. There is no command line to put a
// flag on out there.
func TestDiagFileComesFromTheEnvironmentToo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diag.json")
	t.Setenv("MIZU_DIAG_FILE", path)

	a := &App{Name: "mizu"}
	a.Add(&fails{err: errors.New("the disk is full")})

	if code, _, _ := start(a, "check"); code != CodeFailure {
		t.Errorf("exit %d", code)
	}
	if doc := document(t, read(t, path)); len(doc.Diagnostics) != 1 {
		t.Errorf("the file holds %+v", doc.Diagnostics)
	}
}

// A run that found nothing writes an empty list and a summary of zeroes.
// Telling an empty run from one that never started should not depend on
// whether there was any output.
func TestDiagFileIsWrittenByARunThatWorked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diag.json")

	a := &App{Name: "mizu"}
	a.Add(&fails{})

	if code, _, _ := start(a, "--diag-file", path, "check"); code != CodeOK {
		t.Errorf("exit %d, want 0", code)
	}
	doc := document(t, read(t, path))
	if len(doc.Diagnostics) != 0 {
		t.Errorf("the file holds %+v, want nothing", doc.Diagnostics)
	}
	if doc.Summary.Errors != 0 || doc.Summary.Warnings != 0 {
		t.Errorf("the summary is %+v, want zeroes", doc.Summary)
	}
}

// The file is a record of the run rather than a copy of what was printed, so
// an interrupted run leaves one behind even though it says nothing to the
// person who interrupted it.
func TestDiagFileRecordsAnInterruptedRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diag.json")

	a := &App{Name: "mizu"}
	a.Add(&fails{err: ErrAborted})

	code, _, errOut := start(a, "--diag-file", path, "check")
	if code != CodeInterrupted {
		t.Errorf("exit %d, want %d", code, CodeInterrupted)
	}
	if errOut != "" {
		t.Errorf("it said %q", errOut)
	}
	doc := document(t, read(t, path))
	if len(doc.Diagnostics) != 1 || doc.Diagnostics[0].Message != "aborted" {
		t.Errorf("the file holds %+v", doc.Diagnostics)
	}
}

// The command has already finished by the time the file is written, so a file
// that cannot be written is a warning. Taking the exit code away now would say
// the work did not happen when it did.
func TestADiagFileThatCannotBeWrittenIsAWarning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-directory", "diag.json")

	a := &App{Name: "mizu"}
	a.Add(&fails{})

	code, _, errOut := start(a, "--diag-file", path, "check")
	if code != CodeOK {
		t.Errorf("exit %d, want 0", code)
	}
	if !strings.Contains(errOut, "warning: --diag-file:") {
		t.Errorf("stderr is %q", errOut)
	}
}

// Both at once, which is a generator run under a command line that also asked
// for JSON. The document goes to each place once.
func TestJSONModeAndADiagFileTogether(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diag.json")

	a := &App{Name: "mizu"}
	a.Add(&fails{err: errors.New("the disk is full")})

	_, _, errOut := start(a, "--json", "--diag-file", path, "check")
	for where, s := range map[string]string{"stderr": errOut, "the file": read(t, path)} {
		if doc := document(t, s); len(doc.Diagnostics) != 1 {
			t.Errorf("%s holds %+v", where, doc.Diagnostics)
		}
	}
}

// fussy is a writer that will not take anything, which is a stderr that has
// gone away underneath a command that is still running.
type fussy struct{}

func (fussy) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

// A stderr that will not take a document will not take a second attempt at one
// either, so the fallback is the line a person would read, and the point of it
// is that nothing panics and the exit code still comes back.
func TestAStderrThatWillNotTakeADocument(t *testing.T) {
	c := New(strings.NewReader(""), &strings.Builder{}, fussy{}, Options{JSON: true})
	if code := Report(c, errors.New("the disk is full")); code != CodeFailure {
		t.Errorf("exit %d, want %d", code, CodeFailure)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
