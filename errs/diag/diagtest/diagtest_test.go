package diagtest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

// A recorder is a [testing.TB] that remembers what a helper said about a test
// instead of failing one.
//
// It is the only way to test the half of a harness that runs when a case goes
// wrong. testing.TB cannot be implemented outside the testing package, so the
// real one is embedded and the four methods that matter are replaced. Anything
// else a helper reaches for, Name and TempDir among them, goes to the real T
// and behaves.
type recorder struct {
	testing.TB
	errs  []string
	fatal string
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, a ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, a...))
}

func (r *recorder) Error(a ...any) { r.errs = append(r.errs, fmt.Sprint(a...)) }

// stopped is what a Fatalf panics with, so that the helper under test stops
// where the real one would have called runtime.Goexit.
var stopped = errors.New("the helper called Fatal")

func (r *recorder) Fatalf(format string, a ...any) {
	r.fatal = fmt.Sprintf(format, a...)
	panic(stopped)
}

func (r *recorder) Fatal(a ...any) {
	r.fatal = fmt.Sprint(a...)
	panic(stopped)
}

// watch runs fn against a recorder and returns what it complained about.
func watch(t *testing.T, fn func(tb testing.TB)) *recorder {
	t.Helper()

	r := &recorder{TB: t}
	func() {
		defer func() {
			if p := recover(); p != nil && p != stopped {
				panic(p)
			}
		}()
		fn(r)
	}()
	return r
}

// only returns the one complaint, or fails saying how many there were.
func (r *recorder) only(t *testing.T) string {
	t.Helper()

	switch {
	case r.fatal != "" && len(r.errs) == 0:
		return r.fatal
	case r.fatal == "" && len(r.errs) == 1:
		return r.errs[0]
	}
	t.Fatalf("wanted one complaint, got fatal %q and %d errors: %q", r.fatal, len(r.errs), r.errs)
	return ""
}

// corpus writes a corpus of one case and returns it.
func corpus(t *testing.T, files map[string]string) (dir string, c Case) {
	t.Helper()

	dir = t.TempDir()
	c = Case{Name: "a-case", Dir: filepath.Join(dir, "a-case")}
	if err := os.MkdirAll(c.Dir, 0o777); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(c.Path(name), []byte(body), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	return dir, c
}

// fails is a producer that always returns err.
func fails(err error) func(testing.TB, Case) error {
	return func(testing.TB, Case) error { return err }
}

func TestACaseThatMatchesItsGoldenFilePasses(t *testing.T) {
	_, c := corpus(t, map[string]string{"want.txt": "error: --days: \"soon\" is not a number\n"})

	r := watch(t, func(tb testing.TB) {
		c.verify(tb, fails(errors.New(`--days: "soon" is not a number`)))
	})
	if r.fatal != "" || len(r.errs) > 0 {
		t.Errorf("a matching case complained: %q %q", r.fatal, r.errs)
	}
}

func TestACaseThatDoesNotMatchItsGoldenFileFails(t *testing.T) {
	_, c := corpus(t, map[string]string{"want.txt": "error: --days: \"soon\" is not a number\n"})

	r := watch(t, func(tb testing.TB) {
		c.verify(tb, fails(errors.New("--days: nope")))
	})
	if got := r.only(t); !strings.Contains(got, "does not match the golden file") {
		t.Errorf("says %q, want it to name the golden file", got)
	}
}

// A corpus entry is a deliberately broken input, so an entry that produced
// nothing has stopped testing anything and is worth more than a pass.
func TestACaseWhereNothingWentWrongFails(t *testing.T) {
	_, c := corpus(t, map[string]string{"want.txt": ""})

	r := watch(t, func(tb testing.TB) { c.verify(tb, fails(nil)) })
	if got := r.only(t); !strings.Contains(got, "nothing went wrong") {
		t.Errorf("says %q, want it to say that nothing went wrong", got)
	}
}

// A report built by walking a map comes out in a different order on every run.
// The golden file is where that surfaces, and it surfaces as one confusing
// failure at a time unless the harness looks for it.
func TestAReportThatChangesBetweenRunsFails(t *testing.T) {
	_, c := corpus(t, map[string]string{"want.txt": ""})

	n := 0
	r := watch(t, func(tb testing.TB) {
		c.verify(tb, func(testing.TB, Case) error {
			n++
			return fmt.Errorf("run %d", n)
		})
	})
	if got := r.only(t); !strings.Contains(got, "changed between two runs") {
		t.Errorf("says %q, want it to say the report changed", got)
	}
}

// The path to the case directory is different on every machine and on Windows
// it is spelled with backslashes, so it has no business in a checked in file.
func TestTheCaseDirectoryIsNotInTheReport(t *testing.T) {
	_, c := corpus(t, nil)

	got := c.report(t, fails(fmt.Errorf("%s: no such table", c.Path("app.toml"))))
	if strings.Contains(got, c.Dir) {
		t.Errorf("the report holds the case directory:\n%s", got)
	}
	if want := "error: app.toml: no such table\n"; got != want {
		t.Errorf("report is %q, want %q", got, want)
	}
}

// The rules run on every entry, so a corpus is a rule check as well as a set of
// golden files.
func TestACaseIsHeldToTheRules(t *testing.T) {
	_, c := corpus(t, map[string]string{"want.txt": ""})

	r := watch(t, func(tb testing.TB) {
		c.verify(tb, fails(errors.New("Something went wrong.")))
	})
	if len(r.errs) == 0 {
		t.Fatalf("a message breaking three rules passed: %q", r.fatal)
	}
}

// A report quoting a line of the input reads it out of the case's own
// directory, so an entry works wherever the repository is checked out.
func TestTheSourceIsReadFromTheCase(t *testing.T) {
	_, c := corpus(t, map[string]string{"app.toml": "[app]\nname = 42\n"})

	err := diag.List{{
		Severity: diag.Error,
		Message:  "want a string, got integer",
		File:     "app.toml",
		Range:    diag.Span(2, 8, 2),
	}}.Err()

	if got := c.report(t, fails(err)); !strings.Contains(got, "name = 42") {
		t.Errorf("the report does not quote the input:\n%s", got)
	}
}

// A producer that opened a file by its full path names it that way in the
// diagnostic, and the quoted line has to come out of that file all the same.
func TestTheSourceCanBeAnAbsolutePath(t *testing.T) {
	_, c := corpus(t, map[string]string{"app.toml": "[app]\nname = 42\n"})

	err := diag.List{{
		Severity: diag.Error,
		Message:  "want a string, got integer",
		File:     c.Path("app.toml"),
		Range:    diag.Span(2, 8, 2),
	}}.Err()

	got := c.report(t, fails(err))
	if !strings.Contains(got, "name = 42") {
		t.Errorf("the report does not quote the input:\n%s", got)
	}
	if strings.Contains(got, c.Dir) {
		t.Errorf("the report holds the case directory:\n%s", got)
	}
}

// Run is what a package calls, and what it has to do is visit every case rather
// than the first one.
func TestRunVisitsEveryCase(t *testing.T) {
	dir := t.TempDir()
	want := []string{"first", "second", "third"}
	for _, name := range want {
		c := Case{Name: name, Dir: filepath.Join(dir, name)}
		if err := os.MkdirAll(c.Dir, 0o777); err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf("error: %s went wrong\n", name)
		if err := os.WriteFile(c.Path("want.txt"), []byte(body), 0o666); err != nil {
			t.Fatal(err)
		}
	}

	// Twice each, because a case is rendered twice and the two are compared.
	var seen []string
	Run(t, dir, func(tb testing.TB, c Case) error {
		if len(seen) == 0 || seen[len(seen)-1] != c.Name {
			seen = append(seen, c.Name)
		}
		return fmt.Errorf("%s went wrong", c.Name)
	})

	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Errorf("ran %v, want %v", seen, want)
	}
}

func TestRunFindsEveryCaseDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"first", "second", "third"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o777); err != nil {
			t.Fatal(err)
		}
	}
	// A stray file beside the cases, which a corpus picks up sooner or later.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), nil, 0o666); err != nil {
		t.Fatal(err)
	}

	got, err := cases(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("found %d cases, want %d: %v", len(got), len(want), got)
	}
	for i, c := range got {
		if c.Name != want[i] {
			t.Errorf("case %d is %s, want %s", i, c.Name, want[i])
		}
		if c.Dir != filepath.Join(dir, want[i]) {
			t.Errorf("case %s is at %s", c.Name, c.Dir)
		}
	}
}

// An empty corpus passes every test in a package and tests nothing, which is
// why [Run] fails on one. What it needs to see to do that is no cases at all,
// even in a directory that has files in it.
func TestADirectoryWithNoCaseDirectoriesIsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), nil, 0o666); err != nil {
		t.Fatal(err)
	}

	got, err := cases(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("found %d cases where there are none: %v", len(got), got)
	}
}

func TestLinesDropsBlanksAndComments(t *testing.T) {
	_, c := corpus(t, map[string]string{
		"args": "# what this checks\nusers:prune\n\n--days\nsoon\r\n\n# and nothing else\n",
	})

	got := c.Lines(t, "args")
	want := []string{"users:prune", "--days", "soon"}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %q", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d is %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadingAFileThatIsNotThere(t *testing.T) {
	_, c := corpus(t, nil)

	r := watch(t, func(tb testing.TB) { c.Read(tb, "args") })
	if !strings.Contains(r.fatal, c.Name) {
		t.Errorf("says %q, want the case named", r.fatal)
	}
}
