package lint

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag/diagtest"
	"github.com/go-mizu/mizu/gen"
)

// corpus is where the golden message entries live, under the test module so
// that the packages in it are packages rather than strings. A check reads
// types, and a string has none.
//
// The underscore keeps them out of everybody's way. The go command does not
// match a directory whose name starts with one, so ./... in that module skips
// every case here, and a case that breaks a rule on purpose stays broken
// without anything else tripping over it.
const corpus = "testdata/_diag"

// TestDiagnostics runs the golden message corpus for the checks.
//
// Each directory under testdata/_diag holds a src.go that breaks one rule, and
// a want.txt, which is what somebody running mizu lint over it sees. A case
// may also hold a checks file naming which checks to run, one per line, which
// is how the case for a name that is not a check says so.
//
// Run it with -update to rewrite the want.txt files, then read the diff. That
// diff is user-facing text and the five rules in doc 36 section 2.1 are the
// review checklist for it.
func TestDiagnostics(t *testing.T) {
	dir, pkgs := loadCorpus(t)

	diagtest.Run(t, dir, func(tb testing.TB, c diagtest.Case) error {
		p, ok := pkgs[c.Name]
		if !ok {
			tb.Fatalf("%s did not come back from the load, so the go command did not see it as a package", c.Name)
			return nil
		}
		found, err := Run([]*gen.Package{p}, checksFor(tb, c)...)
		if err != nil {
			return err
		}
		return found.Err()
	})
}

// TestEveryMessageHasACase is what the package comment promises: a check with
// a message no case produces is a message nobody has read.
//
// Two halves, because the messages come from two places. What a check reports
// is a literal handed to report, and what Run refuses is an ordinary error, so
// [diagtest.Cover] finds the second and reportedMessages finds the first.
func TestEveryMessageHasACase(t *testing.T) {
	diagtest.Cover(t, corpus, ".")

	golden := goldenText(t, corpus)
	for _, m := range reportedMessages(t, ".") {
		if !strings.Contains(golden, m.text) {
			t.Errorf("%s: nothing under %s prints %q.\nAdd a case whose source produces it, then run this test with -update and read what it wrote.", m.where, corpus, m.text)
		}
	}
}

// checksFor is the checks a case runs, which is all of them unless it says
// otherwise.
func checksFor(tb testing.TB, c diagtest.Case) []string {
	tb.Helper()

	if _, err := os.Stat(c.Path("checks")); err != nil {
		return nil
	}
	return c.Lines(tb, "checks")
}

// loadCorpus loads every case as a package, in one go, and returns the
// directory the entries are in along with the packages by case name.
//
// One load rather than one per case is worth it twice over. The go command is
// what a corpus of this size spends its time in, and a load that holds every
// case at once is closer to the run somebody has in front of them than a load
// that holds one.
//
// The directory is absolute because the go command reports absolute paths, and
// a golden file with the path to somebody's checkout in it is a golden file
// that only passes on their machine.
func loadCorpus(t *testing.T) (string, map[string]*gen.Package) {
	t.Helper()

	module, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(module, "_diag")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var patterns []string
	for _, e := range entries {
		if e.IsDir() {
			patterns = append(patterns, "./_diag/"+e.Name())
		}
	}
	if len(patterns) == 0 {
		t.Fatalf("%s holds no cases", dir)
	}

	pkgs, err := gen.Load(gen.Config{Dir: module}, patterns...)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]*gen.Package, len(pkgs))
	for _, p := range pkgs {
		if err := p.Err(); err != nil {
			t.Fatalf("%s did not load: %v", p.PkgPath, err)
		}
		out[path.Base(p.PkgPath)] = p
	}
	return dir, out
}
