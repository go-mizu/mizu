// Package diagtest runs the golden message corpus.
//
// The corpus is what makes the five rules in doc 36 section 2.1 enforceable
// rather than aspirational. Each entry is a directory holding an input that is
// deliberately broken and the report mizu produces for it, and the report is
// checked in beside the input. Changing a message changes a file, so it appears
// in a pull request as user-facing text and gets reviewed as user-facing text,
// which is the only mechanism that has ever kept error messages good.
//
// A corpus lives beside the package whose messages it holds, under
// testdata/diag, and each package runs it with one test:
//
//	func TestDiagnostics(t *testing.T) {
//		diagtest.Run(t, "testdata/diag", func(tb testing.TB, c diagtest.Case) error {
//			return load(c.Path("app.toml"))
//		})
//	}
//
// Beside the code rather than in one directory at the root, so that a new
// diagnostic and its corpus entry are the same change to the same package,
// which is the rule doc 36 asks for. The entries are still countable across the
// repository, since they are all under the same path.
//
// # What an entry looks like
//
//	console/testdata/diag/unknown-flag/
//	    args        the command line, one token per line
//	    want.txt    the report, written by -update and read by a person
//
// The input files are whatever the case needs and the producer knows what to
// ask for. It asks the case for them, with [Case.Path] or [Case.Read], and the
// path to the case's own directory is taken back out of the report before the
// golden file is compared. So an entry whose message quotes a file says
// app.toml on every platform rather than a path with the separator of whoever
// ran it, and moving the corpus does not rewrite every file in it.
//
// # What [Run] checks
//
// The golden file is the part a person reviews. Around it are the parts a
// machine can hold, which is [Check]: a message is one line, it starts the way
// a Go error message starts, it does not end in a full stop, it does not reach
// for the words that say nothing, and a code on it is a code the registry knows
// about. A report is also rendered twice and required to come out the same,
// because a list built from a map is a report that changes between runs and a
// golden file is the only place that shows up.
//
// An entry whose input produced no error at all fails. A corpus of deliberately
// broken inputs where nothing broke is a corpus that has stopped testing
// anything, and the usual cause is that the input drifted rather than that the
// message got better.
package diagtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/golden"
)

// A Case is one entry in a corpus.
type Case struct {
	// Name is the directory's name, which is also what the subtest is called.
	Name string

	// Dir is the path to the directory, which is where the input files are.
	Dir string
}

// Path is the path to a file in the case.
func (c Case) Path(name string) string { return filepath.Join(c.Dir, name) }

// Read returns the contents of a file in the case.
func (c Case) Read(tb testing.TB, name string) []byte {
	tb.Helper()
	b, err := os.ReadFile(c.Path(name))
	if err != nil {
		tb.Fatalf("%s: %v", c.Name, err)
	}
	return b
}

// Lines returns the lines of a file in the case, with blank lines and comments
// dropped.
//
// It is how a case holds a command line: one token per line, so a token with a
// space in it needs no quoting and no parser. A line starting with # is a note
// to whoever reads the entry.
func (c Case) Lines(tb testing.TB, name string) []string {
	tb.Helper()

	var out []string
	for line := range strings.Lines(string(c.Read(tb, name))) {
		line = strings.TrimRight(line, "\r\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// Run runs fn over every case directory under dir.
//
// fn returns whatever the code under test returned for that case's input. Run
// turns it into diagnostics with [diag.Of], renders them with [diag.Text], and
// compares the result with want.txt in the case's directory. Running the test
// with -update writes that file.
//
// The report is rendered without colour and with the limit a person gets, so
// what is checked in is what somebody would have seen on a terminal.
//
// An empty dir is a failure. A corpus that has lost its entries passes every
// test in the package and tests nothing, and the usual cause is a directory
// that got renamed rather than one that was meant to be empty.
func Run(t *testing.T, dir string, fn func(tb testing.TB, c Case) error) {
	t.Helper()

	for _, c := range plan(t, dir) {
		t.Run(c.Name, func(t *testing.T) { c.verify(t, fn) })
	}
}

// plan is the cases [Run] is about to run, and the place it gives up if there
// are none.
//
// It takes a [testing.TB] for the same reason [Case.verify] does, which is that
// a harness nobody can point at a broken corpus is a harness with a broken
// corpus in it.
func plan(tb testing.TB, dir string) []Case {
	tb.Helper()

	cases, err := cases(dir)
	if err != nil {
		tb.Fatal(err)
		return nil
	}
	if len(cases) == 0 {
		tb.Fatalf("%s holds no cases", dir)
		return nil
	}
	return cases
}

// cases lists the case directories under dir, in name order, which is what
// os.ReadDir already gives.
func cases(dir string) ([]Case, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Case
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, Case{Name: e.Name(), Dir: filepath.Join(dir, e.Name())})
		}
	}
	return out, nil
}

// verify runs one case and compares what came back with its golden file.
//
// It takes a [testing.TB] rather than the *testing.T it is called with so that
// this package can test what it does when a case goes wrong, which is the half
// of a test harness that nothing else ever exercises.
func (c Case) verify(tb testing.TB, fn func(tb testing.TB, c Case) error) {
	tb.Helper()

	report := c.report(tb, fn)

	// Twice, and the same both times. A list of unknown settings built by
	// walking a map comes out in a different order on every run, and a golden
	// file is where that surfaces, one confusing failure at a time, long after
	// the change that caused it.
	if again := c.report(tb, fn); again != report {
		tb.Fatalf("the report changed between two runs of the same input:\n%s\nthen\n%s", report, again)
	}
	golden.AssertString(tb, report, golden.At(c.Path("want.txt")))
}

// report runs one case and renders what came back.
func (c Case) report(tb testing.TB, fn func(tb testing.TB, c Case) error) string {
	tb.Helper()

	err := fn(tb, c)
	if err == nil {
		tb.Fatalf("%s is a corpus entry and nothing went wrong, so either the input stopped being broken or the code stopped noticing", c.Name)
		return ""
	}

	l := diag.Of(err)
	Check(tb, l)

	var b strings.Builder
	if err := diag.Text(&b, l, diag.WithSource(c.source), diag.WithColor(false)); err != nil {
		tb.Fatal(err)
		return ""
	}
	return c.trim(b.String())
}

// trim takes the path to the case's own directory back out of the report.
//
// A producer that opens a file has to name it as a path, and that path is
// testdata/diag/<case> on one machine and testdata\diag\<case> on another. What
// is left is the base name, which is the part of the message anybody was going
// to read, and it is the same everywhere.
func (c Case) trim(report string) string {
	report = strings.ReplaceAll(report, c.Dir+string(filepath.Separator), "")

	// Again for a producer that built the path itself, with a slash, rather
	// than asking the case for it. Where the separator is already a slash this
	// is the same replacement twice, which costs one pass over a short string
	// and saves a branch that only one platform would ever take.
	return strings.ReplaceAll(report, filepath.ToSlash(c.Dir)+"/", "")
}

// source is where [diag.Text] reads the lines it quotes.
//
// A case names its input by base name and this reads it from the case's own
// directory, so an entry works wherever the repository is checked out and the
// golden file holds no path from the machine that wrote it.
func (c Case) source(file string) ([]byte, error) {
	if filepath.IsAbs(file) {
		return os.ReadFile(file)
	}
	return os.ReadFile(c.Path(file))
}
