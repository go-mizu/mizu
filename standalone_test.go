package mizu

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/archtest"
)

// A package is standalone when somebody can take that one package and nothing
// else.
//
// It is the claim in tenet T1, it is the reason this is a toolkit rather than a
// framework, and it is the kind of claim that stops being true quietly. A
// constructor grows a second parameter, the parameter is a type from three
// packages away, and adopting one package now means adopting four. Nobody
// decided that. It happened over four pull requests that each looked fine.
//
// Doc 35 section 1.1 asks for two tests and this file is both. The first is the
// examples module: one program per package, written the way somebody writes
// their first one, in a module that requires the mizu module and nothing else.
// The second is a pair of rules, one over what a package reaches and one over
// what a caller has to name in order to call it. The second is the one that
// matters, because an example goes on compiling long after the package under it
// has grown a dependency that only bites the next person.
const (
	module   = "github.com/go-mizu/mizu"
	examples = "examples/standalone"
)

// standalone is every package doc 35 section 2 claims can be adopted on its
// own, mapped to what a caller of it may end up holding besides the standard
// library and golang.org/x.
//
// The empty list is the point. It says adopting the package costs one line in a
// go.mod and nothing else, and it is the answer for every package here today.
// A name in one of these lists is a dependency somebody argued for in writing,
// which is what doc 35 section 2.2 is for, and it is also a package the person
// adopting this one now has in their build.
var standalone = map[string][]archtest.Pattern{
	"clock":   nil,
	"conc":    nil,
	"config":  nil,
	"console": nil,
	"crypt":   nil,
	"ctxdata": nil,
	"errs":    nil,
	"hash":    nil,
	"log":     nil,
	"str":     nil,
	"try":     nil,
	"xm":      nil,
	"xs":      nil,
}

// upper is the composition root and everything at L4 and L5 in doc 02 section
// 10, which is what a standalone package must not reach.
//
// L4 is the application services and L5 is the composition root and the tools
// around it. A dependency running that way means the piece only works
// assembled, which is the thing being ruled out. Most of these do not exist
// yet, and they are listed anyway so the rule is waiting when they land rather
// than being written by whoever happens to notice.
var upper = []archtest.Pattern{
	module,
	module + "/ai/...",
	module + "/auth/...",
	module + "/broadcast/...",
	module + "/di/...",
	module + "/flags/...",
	module + "/gate/...",
	module + "/mail/...",
	module + "/mcp/...",
	module + "/mizutest/...",
	module + "/notify/...",
	module + "/pulse/...",
	module + "/schedule/...",
	module + "/search/...",
	module + "/telescope/...",
}

// longExamples are the packages whose example does not fit in the fifteen lines
// doc 35 section 1 asks for.
//
// One entry, and it is the shape of the package rather than the example. A
// command line program has a type with two methods on it before it has printed
// anything, so fifteen lines would mean an example that does not show a command
// being declared, which is the whole of what the package does. It is named here
// rather than matched by a pattern, so a second one is a line somebody has to
// write and defend.
var longExamples = map[string]int{
	"console": 40,
}

// TestEveryStandalonePackageHasAnExample also checks the harder half, which is
// that the example uses one mizu package.
//
// A program that imports two of them still compiles and still runs, and it
// stops being evidence for anything: the reader cannot tell which of the two
// they would have to adopt. One import, and it is the package the directory is
// named after.
func TestEveryStandalonePackageHasAnExample(t *testing.T) {
	g, err := archtest.Load(examples, "./...")
	if err != nil {
		t.Fatal(err)
	}

	found := map[string]bool{}
	for _, root := range g.Roots() {
		p, ok := g.Lookup(root)
		if !ok {
			t.Fatalf("%s is a root of the graph and not in it", root)
		}

		var imports []string
		for _, imp := range p.Imports {
			if imp == module || strings.HasPrefix(imp, module+"/") {
				imports = append(imports, imp)
			}
		}
		name := path.Base(root)
		if want := []string{module + "/" + name}; !slices.Equal(imports, want) {
			t.Errorf("%s imports %v, and an example is one program using %v", root, imports, want)
			continue
		}
		if _, ok := standalone[name]; !ok {
			t.Errorf("%s is an example for a package that is not in the standalone table", root)
			continue
		}
		found[name] = true
	}

	for _, pkg := range slices.Sorted(maps.Keys(standalone)) {
		if !found[pkg] {
			t.Errorf("%s has no example under %s, and doc 35 section 1 asks every standalone package for one", pkg, examples)
		}
	}
}

// An example is read more often than it is run, so it stays short enough to
// read at once. Doc 35 section 1 says fifteen lines and means fifteen lines of
// code, not counting the package clause, the imports, the comments or the
// blanks, which is what somebody skims past on the way to the part that does
// something.
func TestAnExampleIsShortEnoughToRead(t *testing.T) {
	for _, pkg := range slices.Sorted(maps.Keys(standalone)) {
		limit := 15
		if n, ok := longExamples[pkg]; ok {
			limit = n
		}

		file := filepath.Join(examples, pkg, "main.go")
		got, err := codeLines(file)
		if err != nil {
			t.Errorf("%s: %v", pkg, err)
			continue
		}
		if got > limit {
			t.Errorf("%s is %d lines of code, and an example is %d", file, got, limit)
		}
	}
}

// codeLines counts what is left of a file after the package clause, the
// imports, the comments and the blank lines.
func codeLines(file string) (int, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return 0, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.SkipObjectResolution)
	if err != nil {
		return 0, err
	}

	// Everything up to and including the last import is preamble. A file with
	// no imports has none of it past the package clause, so counting starts at
	// the top.
	preamble := 0
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.IMPORT {
			break
		}
		preamble = fset.Position(g.End()).Line
	}

	n := 0
	for i, line := range strings.Split(string(src), "\n") {
		s := strings.TrimSpace(line)
		if i+1 <= preamble || s == "" || strings.HasPrefix(s, "//") {
			continue
		}
		n++
	}
	return n, nil
}

// The example module requires the module under test and nothing else.
//
// This is the number a person weighing adoption actually looks at, and it is
// one. The indirect requirements are skipped because they are not a choice
// anybody makes: they are the golang.org/x repositories the toolkit itself
// pulls in, recorded here because the go command records what the build needs.
// The rule that keeps that list short is in deps_test.go.
func TestTheExampleModuleRequiresOneModule(t *testing.T) {
	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Dir = examples

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go mod edit -json in %s: %v", examples, err)
	}
	var mod struct {
		Require []struct {
			Path     string
			Indirect bool
		}
	}
	if err := json.Unmarshal(out, &mod); err != nil {
		t.Fatalf("decode %s/go.mod: %v", examples, err)
	}

	var direct []string
	for _, r := range mod.Require {
		if !r.Indirect {
			direct = append(direct, r.Path)
		}
	}
	if want := []string{module}; !slices.Equal(direct, want) {
		t.Errorf("%s/go.mod requires %v, and an example module requires %v", examples, direct, want)
	}
}

// What a standalone package reaches.
//
// Two ends of the graph are out of bounds. The composition root is one, because
// a package that reaches back up to it cannot be taken away from it. The upper
// layers are the other: a package at L1 that reaches an application service has
// inverted the dependency the layers exist to state, and whoever adopts the L1
// package gets the service whether they wanted it or not.
func TestAStandalonePackageDoesNotReachUpwards(t *testing.T) {
	g, err := archtest.Load(".", "./...")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range slices.Sorted(maps.Keys(standalone)) {
		from := archtest.Pattern(module + "/" + pkg)
		for _, to := range upper {
			if from == to {
				continue
			}
			for _, v := range g.Forbid(from, to) {
				t.Errorf("%s is meant to be usable on its own\n%s", pkg, v.Error())
			}
		}
	}
}

// What a caller of a standalone package has to name.
//
// This is the rule the import graph cannot state, because the imports run the
// other way. When log took a config.Log, log imported config and config
// imported nothing, so the graph was in order and the package was still not
// standalone: nobody could call log.New without building a value out of config.
// The signature is where that shows up, so the signature is where it is
// checked.
//
// Results do not count and parameters do. A caller writing l, err := log.New()
// has named nothing, and a caller passing a log.Options has to build one first.
func TestAStandaloneConstructorTakesNothingFromMizu(t *testing.T) {
	a, err := archtest.LoadAPI(".", "./...")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range slices.Sorted(maps.Keys(standalone)) {
		allow := append([]archtest.Pattern{"std", "golang.org/x/..."}, standalone[pkg]...)
		for _, r := range a.AllowOnly(module+"/"+pkg, allow...) {
			t.Errorf("%s is meant to be usable on its own\n%s", pkg, r.Error())
		}
	}
}
