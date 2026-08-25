package validategen

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/gen"
)

// Name is what this generator is called on the command line and in a marker.
const Name = "validate"

// Version is bumped when the shape of the output changes, so a file written by
// an older toolchain is visible as one.
const Version = "1"

// outputFile is what every package's validators land in. One file rather than
// one per struct, for the same reason the binders share one: a package with six
// request types would otherwise grow six files whose names nobody chose.
const outputFile = "validate_gen.go"

// Generate writes the Validate method for every marked struct in the given
// packages.
//
// A struct asks for one with a //mizu:validate marker. The rules come off the
// same validate tags [validate.Struct] reads, so the two modes are the same
// rules and a struct can switch between them by adding or taking away the
// marker.
func Generate(pkgs ...*gen.Package) ([]gen.File, error) {
	var files []gen.File
	var errs []error

	for _, p := range pkgs {
		targets, scanErrs := gen.Scan(p)
		for _, e := range scanErrs {
			errs = append(errs, e)
		}

		plan := Analyze(p, targets)
		errs = append(errs, plan.Errors...)
		if len(plan.Errors) > 0 || len(plan.Structs) == 0 {
			continue
		}
		files = append(files, gen.File{
			Path: path.Join(plan.Dir, outputFile),
			Data: render(plan),
		})
	}
	return files, errors.Join(errs...)
}

func marked(t gen.Target) bool {
	for _, m := range t.Markers {
		if m.Name == Name {
			return true
		}
	}
	return false
}

// render writes the file. Everything it needs is in the plan, so this is the
// only place that knows what the output looks like.
//
// It is only ever called with a plan that has no errors in it, which is what
// lets it write rather than check.
func render(p *Plan) []byte {
	var b strings.Builder
	b.WriteString(gen.Header(Name, Version, p.Source))
	fmt.Fprintf(&b, "\npackage %s\n\n", p.Pkg)
	writeImports(&b, p.Imports())

	for i := range p.Structs {
		if i > 0 {
			b.WriteString("\n")
		}
		writeMethod(&b, p, &p.Structs[i])
	}
	for _, h := range p.Helpers {
		b.WriteString("\n")
		writeHelper(&b, p, h)
	}
	return []byte(b.String())
}

// writeImports writes the block. There is always at least context and the
// validate package in it, so there is no empty case to think about.
func writeImports(b *strings.Builder, lines []importLine) {
	if len(lines) == 1 && lines[0].Alias == "" {
		fmt.Fprintf(b, "import %q\n\n", lines[0].Path)
		return
	}
	b.WriteString("import (\n")
	std := true
	for _, l := range lines {
		if std && !isStd(l.Path) {
			b.WriteString("\n")
			std = false
		}
		if l.Alias != "" {
			b.WriteString("\t" + l.Alias + " " + strconv.Quote(l.Path) + "\n")
		} else {
			b.WriteString("\t" + strconv.Quote(l.Path) + "\n")
		}
	}
	b.WriteString(")\n\n")
}

// writeMethod writes one marked struct's Validate method.
//
// The method is the whole of what a caller sees, and the checking is in a
// function next to it. That is what lets a type holding a list of itself have
// something to call, and what lets a type reached from two marked structs be
// written down once.
func writeMethod(b *strings.Builder, p *Plan, s *Struct) {
	fmt.Fprintf(b, "// Validate checks %s %s and reports every field that was wrong.\n//\n", article(s.Type), s.Type)
	b.WriteString("// It is what web.Bind calls once it has filled the struct in, and what\n")
	fmt.Fprintf(b, "// %s.Struct would have interpreted the tags for. The error is nil when\n", p.Val)
	fmt.Fprintf(b, "// nothing failed, and otherwise what %s.Errors.OrNil returns.\n", p.Val)
	fmt.Fprintf(b, "func (v %s) Validate(ctx %s.Context) error {\n", s.Type, p.Ctx)
	fmt.Fprintf(b, "\tvar bad %s.Errors\n", p.Val)
	fmt.Fprintf(b, "\t%s(&bad, \"\", v)\n", s.Call)
	b.WriteString("\treturn bad.OrNil()\n}\n")
}

// writeHelper writes the function that checks one struct type.
//
// at is what the fields are named under, which is empty at the top and
// items.1. inside a list, so one function checks a type wherever it turned up.
func writeHelper(b *strings.Builder, p *Plan, h *Helper) {
	fmt.Fprintf(b, "// %s checks %s %s and adds what was wrong to bad, under at.\n", h.Name, article(h.Doc), h.Doc)
	fmt.Fprintf(b, "func %s(bad *%s.Errors, at string, v %s) {\n", h.Name, p.Val, h.Type)
	for line := range strings.SplitSeq(strings.TrimRight(h.Body, "\n"), "\n") {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString("\t" + line + "\n")
	}
	b.WriteString("}\n")
}

// article is a or an, for a sentence that names a type.
func article(s string) string {
	if s == "" {
		return "a"
	}
	switch s[0] {
	case 'A', 'E', 'I', 'O', 'U', 'a', 'e', 'i', 'o', 'u':
		return "an"
	}
	return "a"
}
