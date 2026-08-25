package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

// A reported is one piece of text a check can put in front of somebody, and
// where it was written.
type reported struct {
	text  string
	where string
}

// reportedMessages is every message, detail and fix the checks in src can
// print.
//
// A check does not build its text with a format string, so [diagtest.Cover],
// which reads formats, does not see any of it. What it hands report is written
// out in full, which makes this the easier half: find the calls, take the
// string literals, and leave out the one that is a code.
//
// It reads the source rather than running the checks because the point is the
// messages no case reaches. A message nothing produces is not in a list built
// by producing things.
func reportedMessages(t testing.TB, src string) []reported {
	t.Helper()

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}

	var out []reported
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(src, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !reports(call.Fun) {
				return true
			}
			where := fset.Position(call.Pos()).String()
			for _, s := range literals(call.Args) {
				if diag.Code(s).Valid() {
					continue
				}
				out = append(out, reported{text: s, where: where})
			}
			return true
		})
	}
	if len(out) == 0 {
		t.Fatalf("%s reports nothing, so either the checks moved or this is looking in the wrong place", src)
	}
	return out
}

// reports reports whether a call is one that puts text in front of somebody.
//
// Both of these take the same four strings in the same order, since fields is
// report with the walk over a field list in front of it.
func reports(fun ast.Expr) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "report" || sel.Sel.Name == "fields"
}

// literals is every string literal in an argument list, unquoted.
func literals(args []ast.Expr) []string {
	var out []string
	for _, a := range args {
		lit, ok := a.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		if s, err := strconv.Unquote(lit.Value); err == nil {
			out = append(out, s)
		}
	}
	return out
}

// goldenText is every golden file under dir, joined, which is what a message
// is looked for in.
//
// Which case prints a message does not matter here. What matters is that some
// case does, because that is the one somebody read when they reviewed it.
func goldenText(t testing.TB, dir string) string {
	t.Helper()

	var b strings.Builder
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "want.txt" {
			return err
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b.Write(text)
		b.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
