package mizu

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// modulePath is what every import path in this module starts with.
const modulePath = "github.com/go-mizu/mizu"

// A fuzz target that only ever sees its seed corpus is a table test with extra
// steps.
//
// The seed corpus is what runs in the ordinary suite, and it has to be, because
// nobody waits an hour for a pull request. The hour happens nightly, in
// .github/workflows/fuzz.yml, and that file names its targets one at a time so
// that each of them gets an hour rather than a share of one. A list written by
// hand is a list that falls behind, so this reads it back.
//
// What it cannot check is that the hour is being spent. GitHub disables a
// scheduled workflow on a repository nobody has pushed to for sixty days, and
// the only thing that notices is somebody looking.
func TestEveryFuzzTargetIsFuzzedInCI(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join(".github", "workflows", "fuzz.yml"))
	if err != nil {
		t.Fatal(err)
	}
	matrix := string(workflow)

	for _, p := range listPackages(t) {
		// The workflow names the package the way go test is given it, which is
		// the path from the module root rather than the directory name. The two
		// differ for a package that is not at the top, and errs/diag is one.
		dir := strings.TrimPrefix(strings.TrimPrefix(p.ImportPath, modulePath), "/")

		for _, name := range append(append([]string{}, p.TestGoFiles...), p.XTestGoFiles...) {
			for _, target := range fuzzTargets(t, filepath.Join(p.Dir, name)) {
				row := "{ pkg: " + dir + ", target: " + target + " }"
				if !strings.Contains(matrix, row) {
					t.Errorf("%s has %s and the nightly fuzz workflow does not run it, so it only ever sees the inputs somebody wrote down.\nAdd this row to the matrix in .github/workflows/fuzz.yml:\n          - %s", p.ImportPath, target, row)
				}
			}
		}
	}
}

// fuzzTargets is the names of the fuzz targets in one test file.
func fuzzTargets(tb testing.TB, path string) []string {
	tb.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		tb.Fatalf("parse %s: %v", path, err)
	}

	var found []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		// The naming rule go test uses, and the same one Example follows: the
		// prefix, and then nothing or something that does not begin with a
		// lower case letter.
		rest, ok := strings.CutPrefix(fn.Name.Name, "Fuzz")
		if !ok {
			continue
		}
		if r, _ := utf8.DecodeRuneInString(rest); rest == "" || !unicode.IsLower(r) {
			found = append(found, fn.Name.Name)
		}
	}
	return found
}
