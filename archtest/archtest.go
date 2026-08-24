package archtest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os/exec"
	"slices"
	"strings"
)

// A Pattern matches package paths. The syntax is the go command's, so "std"
// is every standard library package, a trailing "/..." matches a path and
// everything under it, "..." on its own matches everything, and anything else
// matches one path exactly.
//
// "std" covers the copies of golang.org/x that the toolchain vendors into
// itself, which the go command reports under paths like
// vendor/golang.org/x/net/idna. Those ship with the compiler and never appear
// in a go.mod, so counting them as anything else would fail every rule for a
// package that imports net/http.
type Pattern string

// Package is one node of the graph.
type Package struct {
	ImportPath string
	Standard   bool
	Module     string   // module path, empty for the standard library
	Imports    []string // direct imports, in the order the go command reports them
}

// Graph is a module's import graph, as the go command reports it.
//
// The graph is the build graph and not the test graph, so a package imported
// only by a _test.go file is not in it. That is the right scope for a
// dependency policy, because a test dependency does not reach anybody who
// imports the module. It is the wrong scope for a go.mod policy, and the two
// are worth asserting separately.
type Graph struct {
	roots []string
	pkgs  map[string]*Package
}

// Load returns the import graph of the packages matching patterns, resolved
// from dir. Passing no patterns means "./...".
//
// It shells out to the go command, so the toolchain has to be on PATH, which
// inside a test it always is. Shelling out is the reason this package has no
// dependencies of its own: golang.org/x/tools/go/packages does the same job
// with a nicer API and a module graph attached, and a package that exists to
// keep the toolkit on the standard library should not be the one exception.
//
// Load takes a few hundred milliseconds on a cold cache, so call it once per
// test binary rather than once per test.
func Load(dir string, patterns ...string) (*Graph, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	// One call, not two. -deps flattens the roots in with everything they
	// reach, and the difference between the two is the whole question, but
	// the go command already answers it: DepOnly is set on a package that
	// nothing in the patterns matched.
	out, err := run(dir, append([]string{"list", "-deps", "-json"}, patterns...)...)
	if err != nil {
		return nil, err
	}

	g := &Graph{pkgs: make(map[string]*Package)}

	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var raw struct {
			ImportPath string
			Standard   bool
			DepOnly    bool
			Imports    []string
			Module     *struct{ Path string }
		}
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("archtest: decode go list output: %w", err)
		}
		p := &Package{
			ImportPath: raw.ImportPath,
			Standard:   raw.Standard,
			Imports:    raw.Imports,
		}
		if raw.Module != nil {
			p.Module = raw.Module.Path
		}
		g.pkgs[p.ImportPath] = p
		if !raw.DepOnly {
			g.roots = append(g.roots, p.ImportPath)
		}
	}
	if len(g.roots) == 0 {
		return nil, fmt.Errorf("archtest: %v matched no packages in %s", patterns, dir)
	}
	return g, nil
}

func run(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("archtest: go %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("archtest: go %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// Roots returns the packages the patterns matched, sorted. Everything else in
// the graph is there because a root reached it.
func (g *Graph) Roots() []string {
	return sorted(g.roots)
}

// Packages returns every package in the graph, roots and dependencies alike,
// sorted.
func (g *Graph) Packages() []string {
	return slices.Sorted(maps.Keys(g.pkgs))
}

// Lookup returns the package at path. The second result reports whether it is
// in the graph at all.
func (g *Graph) Lookup(path string) (*Package, bool) {
	p, ok := g.pkgs[path]
	return p, ok
}

// DepsOf returns everything pkg imports, directly or through another package,
// sorted. An unknown package has no dependencies rather than being an error,
// which keeps a caller from having to check twice.
func (g *Graph) DepsOf(pkg string) []string {
	seen := map[string]bool{}
	var walk func(string)
	walk = func(p string) {
		node, ok := g.pkgs[p]
		if !ok {
			return
		}
		for _, imp := range node.Imports {
			if seen[imp] {
				continue
			}
			seen[imp] = true
			walk(imp)
		}
	}
	walk(pkg)
	return slices.Sorted(maps.Keys(seen))
}

// Chain returns the shortest import chain from one package to another,
// starting at from and ending at to. It returns nil when from does not reach
// to, and a chain of length one when they are the same package.
//
// This is the part of a violation somebody can act on. Knowing that the router
// reaches a YAML parser is a fact; knowing which four imports got it there is
// a fix.
func (g *Graph) Chain(from, to string) []string {
	if from == to {
		if _, ok := g.pkgs[from]; ok {
			return []string{from}
		}
		return nil
	}
	if _, ok := g.pkgs[from]; !ok {
		return nil
	}

	prev := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		node, ok := g.pkgs[cur]
		if !ok {
			continue
		}
		for _, imp := range node.Imports {
			if _, seen := prev[imp]; seen {
				continue
			}
			prev[imp] = cur
			if imp == to {
				return unwind(prev, from, to)
			}
			queue = append(queue, imp)
		}
	}
	return nil
}

func unwind(prev map[string]string, from, to string) []string {
	var chain []string
	for at := to; at != ""; at = prev[at] {
		chain = append(chain, at)
		if at == from {
			break
		}
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

// A Violation is one package reaching something it was not allowed to reach.
type Violation struct {
	Package string   // the root package the rule was applied to
	Dep     string   // what it reached
	Chain   []string // how it got there, from Package to Dep
}

// Error renders the violation with its chain, which is the whole point of
// reporting one.
func (v Violation) Error() string {
	if len(v.Chain) < 2 {
		return fmt.Sprintf("%s depends on %s", v.Package, v.Dep)
	}
	return fmt.Sprintf("%s depends on %s\n\t%s", v.Package, v.Dep, strings.Join(v.Chain, "\n\t  -> "))
}

// AllowOnly reports every dependency of every root that no pattern allows.
//
// A root is always allowed to depend on another root, since a rule about what
// a module may reach is a rule about the outside world. Passing no patterns
// therefore asserts that the module depends on nothing at all, which is a
// real thing to want and a rare thing to achieve.
func (g *Graph) AllowOnly(patterns ...Pattern) []Violation {
	roots := map[string]bool{}
	for _, r := range g.roots {
		roots[r] = true
	}

	var out []Violation
	for _, root := range g.Roots() {
		for _, dep := range g.DepsOf(root) {
			if roots[dep] || g.allowed(dep, patterns) {
				continue
			}
			out = append(out, Violation{
				Package: root,
				Dep:     dep,
				Chain:   g.Chain(root, dep),
			})
		}
	}
	return out
}

// Forbid reports every package matching from that reaches a package matching
// to. It is the rule for "no package imports the composition root", and for
// the layering rules an application declares.
func (g *Graph) Forbid(from, to Pattern) []Violation {
	var out []Violation
	for _, root := range g.Roots() {
		if !g.allowed(root, []Pattern{from}) {
			continue
		}
		for _, dep := range g.DepsOf(root) {
			if !g.allowed(dep, []Pattern{to}) {
				continue
			}
			out = append(out, Violation{
				Package: root,
				Dep:     dep,
				Chain:   g.Chain(root, dep),
			})
		}
	}
	return out
}

func (g *Graph) allowed(path string, patterns []Pattern) bool {
	pkg, ok := g.pkgs[path]
	if !ok {
		pkg = &Package{ImportPath: path}
	}
	for _, p := range patterns {
		if p.match(pkg) {
			return true
		}
	}
	return false
}

func (p Pattern) match(pkg *Package) bool {
	s := string(p)
	switch {
	case s == "...":
		return true
	case s == "std":
		return pkg.Standard
	}
	if prefix, ok := strings.CutSuffix(s, "/..."); ok {
		return pkg.ImportPath == prefix || strings.HasPrefix(pkg.ImportPath, prefix+"/")
	}
	return pkg.ImportPath == s
}

func sorted(in []string) []string {
	out := slices.Clone(in)
	slices.Sort(out)
	return out
}
