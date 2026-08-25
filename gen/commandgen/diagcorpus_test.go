package commandgen

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/errs/diag/diagtest"
	"github.com/go-mizu/mizu/gen"
)

// corpus is where the golden message entries live, under the test module so
// that the packages in it are packages rather than strings.
//
// The underscore is what keeps them out of everybody's way. The go command
// does not match a directory whose name starts with one, so ./... in that
// module skips every case here, which is what lets a case be a package that
// does not work. mizu gen and go test both run in that module, in cmd/mizu's
// tests, and neither of them has to know this directory exists.
const corpus = "testdata/_diag"

// TestDiagnostics runs the golden message corpus for this generator.
//
// Each directory under testdata/_diag holds a commands.go that is wrong in one
// way, and a want.txt, which is what somebody running mizu gen over it sees.
// A generator that refuses code is talking to somebody who has to change that
// code, and the message is the whole of what they have to go on.
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
		_, err := Generate(p)
		return err
	})
}

// TestEveryMessageHasAnEntry is M0's tenth acceptance criterion for this
// generator: a message with no entry is a message nobody has read.
func TestEveryMessageHasAnEntry(t *testing.T) {
	// The two that are skipped are wrappers rather than messages. errf writes
	// the position in front of what it was handed, and field writes the name
	// of the field in front of what value said about its type. Both of those
	// messages are in the corpus under their own text.
	diagtest.Cover(t, corpus, ".", "%s: %s", "%s %v")
}

// loadCorpus loads every case as a package, in one go, and returns the
// directory the entries are in along with the packages by case name.
//
// One load rather than one per case is worth it twice over. The go command is
// what a corpus of this size spends its time in, and a load that holds every
// broken package at once is closer to the run somebody has in front of them
// than a load that holds one.
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
		out[path.Base(p.PkgPath)] = p
	}
	return dir, out
}
