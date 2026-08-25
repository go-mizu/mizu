package bindgen

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/gen"
)

// Name is what this generator is called on the command line and in a marker.
const Name = "bind"

// Version is bumped when the shape of the output changes, so a file written by
// an older toolchain is visible as one.
const Version = "1"

// outputFile is what every package's binders land in. One file rather than one
// per struct, because a package with six request types would otherwise grow six
// files whose names nobody chose.
const outputFile = "bind_gen.go"

// Generate writes the BindRequest method for every marked struct in the given
// packages.
//
// A struct asks for one with a //mizu:bind marker. Each package gets one file
// holding every binder in it.
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
		writeBinder(&b, p, &p.Structs[i])
	}
	return []byte(b.String())
}

// writeImports writes the block. There is always at least the web package in
// it, so there is no empty case to think about.
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

// writeBinder writes one struct's method.
func writeBinder(b *strings.Builder, p *Plan, s *Struct) {
	fmt.Fprintf(b, "// BindRequest fills %s %s from the request.\n//\n", article(s.Type), s.Type)
	fmt.Fprintf(b, "// It is the method %s.Bind calls for this type. Each field is read from the\n", p.Web)
	b.WriteString("// place its tags name, and every value that will not decode is reported rather\n")
	b.WriteString("// than the first one.\n")
	fmt.Fprintf(b, "func (v *%s) BindRequest(c *%s.Ctx) error {\n", s.Type, p.Web)
	b.WriteString("\tb := c.Binding()\n")

	if len(s.Vars) > 0 {
		b.WriteString("\n")
		writeVars(b, s.Vars)
	}
	if s.Values {
		b.WriteString("\n")
		writeLoop(b, p, s)
		for i := range s.Fields {
			if f := &s.Fields[i]; f.Src == FromValues && f.List {
				b.WriteString("\n")
				writeList(b, f)
			}
		}
	}

	first := true
	for i := range s.Fields {
		f := &s.Fields[i]
		if f.Src == FromValues {
			continue
		}
		if first {
			b.WriteString("\n")
			first = false
		}
		writeOther(b, p, f)
	}

	b.WriteString("\n")
	if s.Lax {
		b.WriteString("\tb.BodyAllowUnknown(v)\n")
	} else {
		b.WriteString("\tb.Body(v)\n")
	}
	b.WriteString("\treturn b.Err()\n}\n")
}

// writeVars declares the accumulators. A list field is built up as the pairs
// arrive and written into the struct at the end, so that a name the request
// never sent leaves its field alone.
func writeVars(b *strings.Builder, vars []Var) {
	if len(vars) == 1 {
		fmt.Fprintf(b, "\tvar %s %s\n", vars[0].Name, vars[0].Type)
		return
	}
	b.WriteString("\tvar (\n")
	for _, v := range vars {
		fmt.Fprintf(b, "\t\t%s %s\n", v.Name, v.Type)
	}
	b.WriteString("\t)\n")
}

// writeLoop writes the one pass over the request's values.
func writeLoop(b *strings.Builder, p *Plan, s *Struct) {
	b.WriteString("\tfor name, value := range b.Values() {\n\t\tswitch name {\n")
	for i := range s.Fields {
		f := &s.Fields[i]
		if f.Src != FromValues {
			continue
		}
		fmt.Fprintf(b, "\t\tcase %s:\n", strconv.Quote(f.Name))
		if !f.List {
			set(b, p, f, "\t\t\t", "name", "value")
			continue
		}
		// The accumulator starts empty rather than nil the first time the name
		// turns up, so that a field is set to an empty slice when every value
		// sent under its name was one the field does not take.
		fmt.Fprintf(b, "\t\t\tif %s == nil {\n\t\t\t\t%s = %s{}\n\t\t\t}\n", f.Var, f.Var, f.Slice)
		appendTo(b, p, f, "\t\t\t", f.Var, "name", "value")
	}
	b.WriteString("\t\t}\n\t}\n")
}

// writeList writes one accumulator into the struct, which happens only when the
// request used the name at all.
func writeList(b *strings.Builder, f *Field) {
	fmt.Fprintf(b, "\tif %s != nil {\n", f.Var)
	writeLines(b, "\t\t", f.Prep)
	fmt.Fprintf(b, "\t\t%s = %s\n\t}\n", f.Go, f.Var)
}

// writeOther writes one field that comes from somewhere other than the values,
// which is a lookup rather than a pass.
func writeOther(b *strings.Builder, p *Plan, f *Field) {
	name := strconv.Quote(f.Name)

	switch f.Src {
	case FromFile:
		fmt.Fprintf(b, "\tif files := b.Files(%s); len(files) > 0 {\n", name)
		writeLines(b, "\t\t", f.Prep)
		if f.List {
			fmt.Fprintf(b, "\t\t%s = %s\n", f.Go, uploads(p, f))
		} else {
			fmt.Fprintf(b, "\t\t%s = files[0]\n", f.Go)
		}
		b.WriteString("\t}\n")

	case FromHeader:
		if !f.List {
			fmt.Fprintf(b, "\tif s, ok := b.Header(%s); ok {\n", name)
			set(b, p, f, "\t\t", name, "s")
			b.WriteString("\t}\n")
			return
		}
		fmt.Fprintf(b, "\tif got := b.Headers(%s); len(got) > 0 {\n", name)
		writeLines(b, "\t\t", f.Prep)
		fmt.Fprintf(b, "\t\tlist := make(%s, 0, len(got))\n", f.Slice)
		b.WriteString("\t\tfor _, s := range got {\n")
		appendTo(b, p, f, "\t\t\t", "list", name, "s")
		b.WriteString("\t\t}\n")
		fmt.Fprintf(b, "\t\t%s = list\n\t}\n", f.Go)

	default:
		read := "Path"
		if f.Src == FromCookie {
			read = "Cookie"
		}
		fmt.Fprintf(b, "\tif s, ok := b.%s(%s); ok {\n", read, name)
		if !f.List {
			set(b, p, f, "\t\t", name, "s")
			b.WriteString("\t}\n")
			return
		}
		writeLines(b, "\t\t", f.Prep)
		fmt.Fprintf(b, "\t\tlist := make(%s, 0, 1)\n", f.Slice)
		appendTo(b, p, f, "\t\t", "list", name, "s")
		fmt.Fprintf(b, "\t\t%s = list\n\t}\n", f.Go)
	}
}

// set writes the statements that put one value into one field.
//
// An empty value is a field nobody filled in rather than a mistake, which is
// what the helpers do with one and what the guard in front of a pointer field
// is for: a blank number input posts an empty value, and allocating for it
// would say the field arrived when it did not. A string is the other way round,
// since an empty string is a value somebody can mean.
func set(b *strings.Builder, p *Plan, f *Field, tab, name, value string) {
	writeLines(b, tab, f.Prep)

	switch {
	case f.Ptr && f.Kind != Assign:
		fmt.Fprintf(b, "%sif %s != \"\" {\n", tab, value)
		in := tab + "\t"
		alloc(b, f, in)
		fmt.Fprintf(b, "%s%s\n", in, call(p, f, f.Go, name, value))
		fmt.Fprintf(b, "%s}\n", tab)

	case f.Ptr:
		alloc(b, f, tab)
		fmt.Fprintf(b, "%s*%s = %s\n", tab, f.Go, conv(f, value))

	case f.Kind == Assign:
		fmt.Fprintf(b, "%s%s = %s\n", tab, f.Go, conv(f, value))

	default:
		fmt.Fprintf(b, "%s%s\n", tab, call(p, f, "&"+f.Go, name, value))
	}
}

// appendTo writes the statements that add one value to a list field's
// accumulator. A value the field does not take is left out of it, and the
// helper that said so has already recorded which name it arrived under.
func appendTo(b *strings.Builder, p *Plan, f *Field, tab, list, name, value string) {
	switch {
	case f.Ptr && f.Kind != Assign:
		fmt.Fprintf(b, "%sif %s != \"\" {\n", tab, value)
		in := tab + "\t"
		fmt.Fprintf(b, "%se := new(%s)\n", in, f.Type)
		fmt.Fprintf(b, "%sif %s {\n%s\t%s = append(%s, e)\n%s}\n",
			in, call(p, f, "e", name, value), in, list, list, in)
		fmt.Fprintf(b, "%s}\n", tab)

	case f.Ptr:
		fmt.Fprintf(b, "%se := new(%s)\n", tab, f.Type)
		fmt.Fprintf(b, "%s*e = %s\n", tab, conv(f, value))
		fmt.Fprintf(b, "%s%s = append(%s, e)\n", tab, list, list)

	case f.Kind == Assign:
		fmt.Fprintf(b, "%s%s = append(%s, %s)\n", tab, list, list, conv(f, value))

	default:
		fmt.Fprintf(b, "%svar e %s\n", tab, f.Type)
		fmt.Fprintf(b, "%sif %s {\n%s\t%s = append(%s, e)\n%s}\n",
			tab, call(p, f, "&e", name, value), tab, list, list, tab)
	}
}

// alloc writes the pointer a field holds, filled in on the way to writing
// through it.
func alloc(b *strings.Builder, f *Field, tab string) {
	fmt.Fprintf(b, "%sif %s == nil {\n%s\t%s = new(%s)\n%s}\n", tab, f.Go, tab, f.Go, f.Type, tab)
}

// call is the web helper that reads this field's type.
func call(p *Plan, f *Field, dst, name, value string) string {
	return fmt.Sprintf("%s.%s(b, %s, %s, %s)", p.Web, helpers[f.Kind], dst, name, value)
}

var helpers = map[Kind]string{
	Int:      "Int",
	Uint:     "Uint",
	Float:    "Float",
	Bool:     "Bool",
	Time:     "Time",
	Duration: "Duration",
	Text:     "Text",
}

// conv is a string on its way into a field that takes one, converted when the
// field is a type of its own and written as it is when it is not.
func conv(f *Field, value string) string {
	if f.Type == "" || f.Type == "string" {
		return value
	}
	return f.Type + "(" + value + ")"
}

// uploads is the files a form carried on their way into a field, converted when
// the field is a slice type of its own.
func uploads(p *Plan, f *Field) string {
	if f.Slice == "[]*"+p.Web+".Upload" {
		return "files"
	}
	return f.Slice + "(files)"
}

func writeLines(b *strings.Builder, tab string, lines []string) {
	for _, l := range lines {
		b.WriteString(tab)
		b.WriteString(l)
		b.WriteString("\n")
	}
}

// article is the word in front of a type's name in a sentence, which is a
// guess off the first letter and right for every name anybody writes.
func article(name string) string {
	if name == "" {
		return "a"
	}
	switch name[0] {
	case 'A', 'E', 'I', 'O', 'U', 'a', 'e', 'i', 'o', 'u':
		return "an"
	}
	return "a"
}
