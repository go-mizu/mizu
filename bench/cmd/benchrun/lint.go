package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// lint applies the rules that make two runs comparable.
//
// None of them is about a benchmark being correct. A benchmark that reads the
// clock inside its loop measures whatever the machine was doing at the time,
// and it does that without failing, without looking wrong in review, and
// without any sign in the output beyond a number that moves. These are the ways
// that happens, written down.
func lint(root string, w io.Writer) error {
	problems, err := lintSource(root)
	if err != nil {
		return err
	}
	corpus, err := lintCorpus(root)
	if err != nil {
		return err
	}
	problems = append(problems, corpus...)

	for i, p := range problems {
		problems[i].Pos = relative(root, p.Pos)
	}
	slices.SortFunc(problems, func(a, b problem) int { return strings.Compare(a.String(), b.String()) })
	for _, p := range problems {
		fmt.Fprintln(w, p)
	}
	if len(problems) == 0 {
		fmt.Fprintln(w, "no problems")
		return nil
	}
	return errors.New(plural(len(problems), "problem", "problems"))
}

// A problem is one broken rule, printed the way a compiler prints one so that
// an editor can jump to it.
type problem struct {
	Pos  string
	Rule string
	Msg  string
}

func (p problem) String() string { return fmt.Sprintf("%s: %s (%s)", p.Pos, p.Msg, p.Rule) }

// relative rewrites a position against the module root, so that a problem reads
// as micro/log_test.go:9:3 rather than as the whole path from the root of the
// disk. A position that is somehow outside the module is left alone.
func relative(root, pos string) string {
	rel, err := filepath.Rel(root, pos)
	if err != nil || strings.HasPrefix(rel, "..") {
		return pos
	}
	return filepath.ToSlash(rel)
}

// lintSource loads the benchmark module and checks every file in it, tests
// included, since the benchmarks are test files.
func lintSource(root string) ([]problem, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles,
		Dir:   root,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading the benchmark module: %w", err)
	}
	if n := packages.PrintErrors(pkgs); n > 0 {
		return nil, fmt.Errorf("the benchmark module does not build")
	}

	var out []problem
	// The same file arrives more than once, because loading with tests gives
	// both the package and its test variant.
	seen := map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			path := pkg.Fset.Position(file.Pos()).Filename
			if seen[path] {
				continue
			}
			seen[path] = true

			c := &checker{fset: pkg.Fset, info: pkg.TypesInfo}
			c.file(file)
			out = append(out, c.out...)
		}
	}
	return out, nil
}

// A checker walks one file.
//
// info may be nil, in which case the rule about ranging over a map is skipped,
// because that is the one rule that cannot be decided from the syntax alone.
// The tests use a nil one so that a rule can be exercised against a few lines
// of source rather than against a module.
type checker struct {
	fset *token.FileSet
	info *types.Info
	out  []problem
}

func (c *checker) report(pos token.Pos, rule, format string, a ...any) {
	c.out = append(c.out, problem{
		Pos:  c.fset.Position(pos).String(),
		Rule: rule,
		Msg:  fmt.Sprintf(format, a...),
	})
}

func (c *checker) file(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			c.randCall(call)
		}
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if b, ok := benchParam(fn.Type); ok {
				c.benchmark(fn.Name.Name, b, fn.Body)
			}
		case *ast.FuncLit:
			if b, ok := benchParam(fn.Type); ok {
				c.benchmark("the benchmark here", b, fn.Body)
			}
		}
		return true
	})
}

// benchmark checks one function that takes a *testing.B. name is what to call
// it in a message and b is what the parameter was called, since nothing makes
// it b.
func (c *checker) benchmark(name, b string, body *ast.BlockStmt) {
	if body == nil {
		return
	}

	var loops []*ast.ForStmt
	reportsAllocs := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ForStmt:
			if isCall(n.Cond, b, "Loop") {
				loops = append(loops, n)
			}
		case *ast.SelectorExpr:
			if isIdent(n.X, b) && n.Sel.Name == "N" {
				c.report(n.Pos(), "b-loop",
					"%s drives its loop with %s.N, and b.Loop is what keeps the compiler from deleting work nobody reads", name, b)
			}
		case *ast.CallExpr:
			if isCall(n, b, "ReportAllocs") {
				reportsAllocs = true
			}
		}
		return true
	})

	if len(loops) == 0 {
		// A benchmark with no loop of its own is a parent that only calls
		// b.Run, and it measures nothing itself.
		return
	}
	if !reportsAllocs {
		c.report(body.Pos(), "report-allocs",
			"%s does not call %s.ReportAllocs, and an allocation count is half of what a benchmark is for", name, b)
	}
	for _, loop := range loops {
		c.measured(name, loop.Body)
	}
}

// measured checks the body of a b.Loop loop, which is the part that is timed.
func (c *checker) measured(name string, body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if isCall(n, "time", "Now") || isCall(n, "time", "Since") {
				c.report(n.Pos(), "clock",
					"%s reads the clock inside the measured region, which measures the machine rather than the code", name)
			}
		case *ast.RangeStmt:
			if c.isMap(n.X) {
				c.report(n.Pos(), "map-order",
					"%s ranges over a map inside the measured region, and the order is different on every run", name)
			}
		}
		return true
	})
}

// isMap reports whether e is a map. It answers no when there is no type
// information, which is the case the tests run in.
func (c *checker) isMap(e ast.Expr) bool {
	if c.info == nil {
		return false
	}
	t := c.info.TypeOf(e)
	if t == nil {
		return false
	}
	_, ok := t.Underlying().(*types.Map)
	return ok
}

// randConstructors are the math/rand entry points that take a seed or a source,
// so a benchmark using one has said what its input is. Everything else in the
// package draws from a global source that the runtime seeds differently on
// every run.
var randConstructors = []string{"New", "NewSource", "NewPCG", "NewChaCha8", "NewZipf"}

func (c *checker) randCall(call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !isIdent(sel.X, "rand") || slices.Contains(randConstructors, sel.Sel.Name) {
		return
	}
	c.report(call.Pos(), "rand-seed",
		"rand.%s draws from a source seeded differently on every run, so build one from a fixed seed with rand.New", sel.Sel.Name)
}

// benchParam returns the name of the *testing.B parameter, if there is one.
func benchParam(t *ast.FuncType) (string, bool) {
	if t.Params == nil {
		return "", false
	}
	for _, f := range t.Params.List {
		star, ok := f.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok || !isIdent(sel.X, "testing") || sel.Sel.Name != "B" {
			continue
		}
		if len(f.Names) == 0 {
			return "", false // func(*testing.B) as a type, with nothing to call it
		}
		return f.Names[0].Name, true
	}
	return "", false
}

// isCall reports whether e is a call of x.sel.
func isCall(e ast.Expr, x, sel string) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	s, ok := call.Fun.(*ast.SelectorExpr)
	return ok && isIdent(s.X, x) && s.Sel.Name == sel
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

// corpusIndex is the file that says what every corpus is and what changing one
// costs. It is one file rather than a header comment in each, because a corpus
// is often a format with nowhere to put a comment.
const corpusIndex = "README.md"

// lintCorpus checks that every corpus file is accounted for. A file nobody
// wrote down is one nobody knows the provenance of, which is the same problem
// as an input that changes: the numbers stop meaning anything and there is
// nothing in the repository that says so.
func lintCorpus(root string) ([]problem, error) {
	dir := filepath.Join(root, "testdata")
	index, err := os.ReadFile(filepath.Join(dir, corpusIndex))
	if err != nil {
		return nil, fmt.Errorf("reading the corpus index: %w", err)
	}

	var out []problem
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		if name == corpusIndex || strings.Contains(string(index), name) {
			return nil
		}
		out = append(out, problem{
			Pos:  filepath.Join("testdata", name),
			Rule: "corpus",
			Msg:  "is not listed in testdata/" + corpusIndex + ", so nothing says what it is or what changing it costs",
		})
		return nil
	})
	return out, err
}
