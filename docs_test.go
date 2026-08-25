package mizu

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// Documentation is part of the package, so it is held to rules the way the rest
// of the package is.
//
// Doc 03 asks that every package ship a doc.go with an overview and a runnable
// example. Written down and nothing else, that lasts until the first package
// somebody is in a hurry with. The tests below are the same sentence, said
// where a missing doc.go fails a build.
//
// They are deliberately shallow. Whether a comment is any good is a question
// for a reader, and no test is going to answer it. What a test can answer is
// whether there is one at all, whether it is where go doc and pkg.go.dev look
// for it, and whether anybody has shown the package being used.

// pkg is what go list reports about one package, cut down to the fields these
// tests read.
type pkg struct {
	ImportPath   string
	Name         string
	Dir          string
	GoFiles      []string
	TestGoFiles  []string
	XTestGoFiles []string
}

// listPackages asks the go command what is in the module, which is the only
// answer that stays right as packages are added and removed.
func listPackages(tb testing.TB) []pkg {
	tb.Helper()

	out, err := exec.Command("go", "list", "-json", "./...").Output()
	if err != nil {
		tb.Fatalf("go list: %v", err)
	}

	var pkgs []pkg
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p pkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			tb.Fatalf("decode go list output: %v", err)
		}
		pkgs = append(pkgs, p)
	}
	if len(pkgs) == 0 {
		tb.Fatal("go list reported no packages")
	}
	return pkgs
}

// comment is the package comment in one file, or the empty string.
func comment(tb testing.TB, path string) string {
	tb.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		tb.Fatalf("parse %s: %v", path, err)
	}
	if f.Doc == nil {
		return ""
	}
	return f.Doc.Text()
}

// The comment goes in doc.go and nowhere else.
//
// Where it lives makes no difference to go doc, which is exactly the problem: a
// comment on top of whichever file the package started as is a comment that
// gets scrolled past, edited around, and eventually describes something the
// package used to do. A file with nothing in it but the overview is a file
// somebody opens on purpose.
func TestEveryPackageCommentIsInDocGo(t *testing.T) {
	for _, p := range listPackages(t) {
		var found []string
		for _, name := range p.GoFiles {
			if comment(t, filepath.Join(p.Dir, name)) != "" {
				found = append(found, name)
			}
		}
		switch {
		case len(found) == 0:
			t.Errorf("%s has no package comment, and it owes one in doc.go", p.ImportPath)
		case len(found) > 1:
			t.Errorf("%s has a package comment in %s, and go doc runs them together", p.ImportPath, strings.Join(found, " and "))
		case found[0] != "doc.go":
			t.Errorf("%s keeps its package comment in %s rather than doc.go", p.ImportPath, found[0])
		}
	}
}

// A comment that does not name what it is describing reads as a fragment
// wherever it is quoted, and the first line is quoted everywhere: in go doc, in
// the package list on pkg.go.dev, and in the search results that get somebody
// here in the first place.
func TestEveryPackageCommentStartsWithItsName(t *testing.T) {
	for _, p := range listPackages(t) {
		doc := comment(t, filepath.Join(p.Dir, "doc.go"))
		if doc == "" {
			continue // the test above has this one
		}

		want := "Package " + p.Name + " "
		if p.Name == "main" {
			// A command is named after its directory rather than its package
			// clause, and go doc says Command rather than Package for one.
			want = "Command " + filepath.Base(p.Dir) + " "
		}
		if !strings.HasPrefix(doc, want) {
			first, _, _ := strings.Cut(doc, "\n")
			t.Errorf("%s opens with %q, and a package comment starts with %q", p.ImportPath, first, strings.TrimSpace(want))
		}
	}
}

// Every package shows itself being used.
//
// An example is the part of the documentation that cannot be wrong, because it
// is compiled, and the one with an Output comment is run as well. It is also
// what most people read instead of the prose, so a package without one is a
// package whose documentation is a paragraph and a list of signatures.
//
// Not every example can assert. A fixture for tests is called with a *testing.T
// the surrounding test was handed, and an example has no test to fail, so the
// examples in golden, mizutest, consoletest and diagtest are compiled and shown
// rather than run. That is a real distinction and it is stated in each of those
// packages. It is not one this test can make, so what is required here is an
// example rather than a runnable one.
func TestEveryPackageHasAnExample(t *testing.T) {
	for _, p := range listPackages(t) {
		// pkg.go.dev has no page for a command's examples, so an example in one
		// is shown to nobody. The commands document themselves in doc.go, which
		// is what mizu --help prints a shorter version of.
		if p.Name == "main" {
			continue
		}

		var count int
		for _, name := range append(append([]string{}, p.TestGoFiles...), p.XTestGoFiles...) {
			count += exampleCount(t, filepath.Join(p.Dir, name))
		}
		if count == 0 {
			t.Errorf("%s has no Example function, so nothing in its documentation shows it being used", p.ImportPath)
		}
	}
}

// exampleCount counts the Example functions in one test file.
func exampleCount(tb testing.TB, path string) int {
	tb.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		tb.Fatalf("parse %s: %v", path, err)
	}

	var n int
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		// The naming rule go test uses: Example, or Example followed by
		// something that does not start with a lower case letter.
		name, ok := strings.CutPrefix(fn.Name.Name, "Example")
		if !ok {
			continue
		}
		if r, _ := utf8.DecodeRuneInString(name); name == "" || !unicode.IsLower(r) {
			n++
		}
	}
	return n
}
