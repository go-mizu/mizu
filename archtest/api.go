package archtest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
)

// An API is the exported surface of a module's packages, as the type checker
// sees it.
//
// The import graph answers what a package reaches. This answers what a caller
// has to name in order to use it, which is the other half of whether a package
// can be picked up on its own. A package can import nothing at all and still be
// unusable alone, because its constructor takes a type that only exists three
// packages away.
type API struct {
	roots []string
	pkgs  map[string]*types.Package
	std   map[string]bool
}

// LoadAPI type checks the packages matching patterns, resolved from dir.
// Passing no patterns means "./...".
//
// It reads the export data the compiler already writes, which means the
// packages are built first. That is slower than [Load] on a cold cache and
// about the same afterwards, and it is the only way to ask what a signature
// says without a type checker of one's own.
//
// The toolchain that runs the test has to be the one that wrote the export
// data, which inside go test it is.
func LoadAPI(dir string, patterns ...string) (*API, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	out, err := run(dir, append([]string{"list", "-deps", "-export", "-json"}, patterns...)...)
	if err != nil {
		return nil, err
	}

	a := &API{pkgs: map[string]*types.Package{}, std: map[string]bool{}}
	exports := map[string]string{}

	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var raw struct {
			ImportPath string
			Export     string
			DepOnly    bool
			Standard   bool
		}
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("archtest: decode go list output: %w", err)
		}
		if raw.Export != "" {
			exports[raw.ImportPath] = raw.Export
		}
		if raw.Standard {
			a.std[raw.ImportPath] = true
		}
		if !raw.DepOnly {
			a.roots = append(a.roots, raw.ImportPath)
		}
	}
	if len(a.roots) == 0 {
		return nil, fmt.Errorf("archtest: %v matched no packages in %s", patterns, dir)
	}

	// go list -deps reports every package the roots reach, so the map has an
	// entry for whatever the type checker asks for. A path that is somehow not
	// in it opens nothing, and the type checker says which import it was.
	imp := importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		return os.Open(exports[path])
	})
	for _, root := range a.roots {
		pkg, err := imp.Import(root)
		if err != nil {
			return nil, fmt.Errorf("archtest: type check %s: %w", root, err)
		}
		a.pkgs[root] = pkg
	}
	return a, nil
}

// Roots returns the packages the patterns matched, sorted.
func (a *API) Roots() []string { return sorted(a.roots) }

// Package returns the type checked package at path. The second result reports
// whether the patterns matched it.
//
// A command is in here like anything else, with an empty package scope, since
// nothing in a main package is callable from outside it. A rule over a whole
// module then reads a command as a package that asks nothing of anybody, which
// is what it is.
func (a *API) Package(path string) (*types.Package, bool) {
	p, ok := a.pkgs[path]
	return p, ok
}

// A Func is one exported function or method, and the packages a caller has to
// name to call it.
type Func struct {
	Package string   // where it is declared
	Name    string   // Add for a function, Server.Add for a method
	Needs   []string // the packages named in its parameters, sorted
}

// String is the function as somebody would refer to it, which is the last
// element of the import path and the name.
func (f Func) String() string {
	name := f.Package
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name + "." + f.Name
}

// Funcs returns the exported functions and methods of the package at path,
// sorted by name.
//
// Parameters are what Needs is computed from, and results are not. A caller
// that writes l, err := log.New(...) has named nothing, so a returned type
// costs them no import. A parameter is different: it has to be built before the
// call, out of a package that has to be imported to build it.
//
// The walk stops at a named type. A parameter of type log.Options needs log,
// and what the fields of Options are made of is log's business rather than the
// caller's. It does not stop at a type spelled out in the signature, so a
// []store.Record, a map[string]store.Record and a func(store.Record) all need
// store, since a caller cannot write any of them without it.
func (a *API) Funcs(path string) []Func {
	pkg, ok := a.pkgs[path]
	if !ok {
		return nil
	}

	// Export data is not only the exported surface. An unexported function
	// small enough to inline is in there too, with its body, so that the
	// packages importing this one can inline it, and so is an unexported
	// method, because a method set is part of what a type satisfies. Neither is
	// something a caller can write, so neither is part of the API.
	var out []Func
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch x := obj.(type) {
		case *types.Func:
			out = append(out, a.fn(pkg, x.Name(), x.Signature()))
		case *types.TypeName:
			named, ok := types.Unalias(x.Type()).(*types.Named)
			// An alias to somebody else's type brings that type's methods with
			// it, and they belong to the package that declared them.
			if !ok || named.Obj().Pkg() != pkg {
				continue
			}
			for i := range named.NumMethods() {
				m := named.Method(i)
				if !m.Exported() {
					continue
				}
				out = append(out, a.fn(pkg, x.Name()+"."+m.Name(), m.Signature()))
			}
		}
	}
	slices.SortFunc(out, func(x, y Func) int { return strings.Compare(x.Name, y.Name) })
	return out
}

func (a *API) fn(pkg *types.Package, name string, sig *types.Signature) Func {
	needs := map[string]bool{}
	for i := range sig.Params().Len() {
		walk(sig.Params().At(i).Type(), pkg, needs)
	}
	return Func{
		Package: pkg.Path(),
		Name:    name,
		Needs:   slices.Sorted(maps.Keys(needs)),
	}
}

// walk records the package of every named type reachable in t without going
// inside one.
//
// It ends, because a named type stops it and everything else is written out at
// the call, and a type somebody wrote out is finite.
func walk(t types.Type, own *types.Package, needs map[string]bool) {
	switch x := types.Unalias(t).(type) {
	case *types.Named:
		// A type parameter's constraint is not part of what a caller writes at
		// the call, and a built-in like error has no package at all.
		if pkg := x.Obj().Pkg(); pkg != nil && pkg != own {
			needs[pkg.Path()] = true
		}
		for i := range x.TypeArgs().Len() {
			walk(x.TypeArgs().At(i), own, needs)
		}
	case *types.Pointer:
		walk(x.Elem(), own, needs)
	case *types.Slice:
		walk(x.Elem(), own, needs)
	case *types.Array:
		walk(x.Elem(), own, needs)
	case *types.Chan:
		walk(x.Elem(), own, needs)
	case *types.Map:
		walk(x.Key(), own, needs)
		walk(x.Elem(), own, needs)
	case *types.Struct:
		// Anonymous, since a named one stopped above. A caller writing it out
		// names everything in it.
		for i := range x.NumFields() {
			walk(x.Field(i).Type(), own, needs)
		}
	case *types.Interface:
		// Also anonymous, and the same reasoning: whatever satisfies it has to
		// be written against these signatures.
		for i := range x.NumMethods() {
			walk(x.Method(i).Signature(), own, needs)
		}
	case *types.Signature:
		for i := range x.Params().Len() {
			walk(x.Params().At(i).Type(), own, needs)
		}
		for i := range x.Results().Len() {
			walk(x.Results().At(i).Type(), own, needs)
		}
	}
}

// A Requirement is one exported function that cannot be called without naming
// a package a rule did not allow.
type Requirement struct {
	Func Func
	Need string
}

// Error names the function and what calling it costs.
func (r Requirement) Error() string {
	return fmt.Sprintf("%s cannot be called without %s", r.Func, r.Need)
}

// AllowOnly reports every exported function in the package at path whose
// parameters name a package no pattern allows.
//
// The package's own types are always allowed, since a package a caller cannot
// name is not a package. Everything else has to be listed, including a sibling
// in the same module, which is the whole question: a package that can only be
// used with three of its neighbours is three packages rather than one.
//
//	for _, r := range a.AllowOnly("github.com/go-mizu/mizu/log", "std") {
//		t.Error(r)
//	}
func (a *API) AllowOnly(path string, patterns ...Pattern) []Requirement {
	var out []Requirement
	for _, f := range a.Funcs(path) {
		for _, need := range f.Needs {
			if a.allowed(need, patterns) {
				continue
			}
			out = append(out, Requirement{Func: f, Need: need})
		}
	}
	return out
}

func (a *API) allowed(path string, patterns []Pattern) bool {
	pkg := &Package{ImportPath: path, Standard: a.std[path]}
	for _, p := range patterns {
		if p.match(pkg) {
			return true
		}
	}
	return false
}
