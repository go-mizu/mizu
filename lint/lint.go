package lint

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/gen"
)

// A Check is one rule, and the code that finds where it was broken.
type Check struct {
	// Name is what somebody types after mizu lint to run this one alone.
	Name string

	// Doc is one line saying what the check is for, shown in the help.
	Doc string

	// Run reports everything wrong in one package.
	Run func(pkg *gen.Package) diag.List
}

// checks is every check, in the order the help lists them and the order a run
// reports them.
var checks = []Check{
	{"ctx", "A *web.Ctx that outlives the handler it belongs to", checkCtx},
}

// Checks is every check there is.
func Checks() []Check { return slices.Clone(checks) }

// Run reports what the named checks found in pkgs, sorted by where it is.
//
// Naming no check runs all of them. Naming one that does not exist is an error,
// because a typo that quietly runs nothing is a lint that passes for a month
// before anybody looks.
func Run(pkgs []*gen.Package, names ...string) (diag.List, error) {
	list, err := pick(names)
	if err != nil {
		return nil, err
	}

	var found diag.List
	for _, pkg := range pkgs {
		if skip(pkg) {
			continue
		}
		for _, c := range list {
			found = append(found, c.Run(pkg)...)
		}
	}
	found.Sort()
	return found, nil
}

// pick is the checks named, or all of them.
func pick(names []string) ([]Check, error) {
	if len(names) == 0 {
		return checks, nil
	}

	var out []Check
	for _, name := range names {
		i := slices.IndexFunc(checks, func(c Check) bool { return c.Name == name })
		if i < 0 {
			return nil, fmt.Errorf("there is no check called %q, and the checks are %s", name, checkNames())
		}
		out = append(out, checks[i])
	}
	return out, nil
}

// checkNames is every check name, for the message above.
func checkNames() string {
	var out []string
	for _, c := range checks {
		out = append(out, c.Name)
	}
	return strings.Join(out, ", ")
}

// skip reports whether a package is one no check has anything to say about.
//
// A package the loader could not type check is skipped, because a check reading
// half a type graph reports things that are not there, and the compiler has
// already said what is actually wrong.
func skip(pkg *gen.Package) bool {
	return pkg.Types == nil || pkg.TypesInfo == nil || len(pkg.Syntax) == 0
}

// at is the file and range of a piece of syntax, in the shape a diagnostic
// wants it.
//
// The end is on the same line as the start or it is left off, since a caret run
// that wraps across lines is a report nobody can read and the first line is the
// one that names the thing anyway.
func at(fset *token.FileSet, n ast.Node) (string, diag.Range) {
	start := fset.Position(n.Pos())
	end := fset.Position(n.End())
	if end.Line != start.Line {
		return start.Filename, diag.At(start.Line, start.Column)
	}
	return start.Filename, diag.Span(start.Line, start.Column, end.Column-start.Column)
}
