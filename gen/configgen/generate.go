package configgen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/gen"
)

// Name is what this generator is called on the command line and in a marker.
const Name = "config"

// Version is bumped when the shape of the output changes, so a file written by
// an older toolchain is visible as one.
const Version = "1"

// Generate writes the decoder for every configuration struct in the given
// packages.
//
// A struct asks for one with a //mizu:config marker. Two structs in one
// package are an error, since an application has one configuration and a
// second one means a mistake rather than a second application.
func Generate(pkgs ...*gen.Package) ([]gen.File, error) {
	var files []gen.File
	var errs []error

	for _, p := range pkgs {
		targets, scanErrs := gen.Scan(p)
		for _, e := range scanErrs {
			errs = append(errs, e)
		}

		var plans []*Plan
		docs := fieldDocs(p)
		for _, t := range targets {
			if !marked(t) {
				continue
			}
			plan := Analyze(t, docs)
			plan.Source = path.Join(dirOf(p), filepath.Base(t.Pos().Filename))
			errs = append(errs, plan.Errors...)
			if len(plan.Errors) == 0 {
				plans = append(plans, plan)
			}
		}
		switch {
		case len(plans) == 0:
			continue
		case len(plans) > 1:
			errs = append(errs, fmt.Errorf("%s has %d structs marked as configuration, and an application has one",
				p.PkgPath, len(plans)))
			continue
		}

		data, err := render(plans[0])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		files = append(files, gen.File{Path: path.Join(dirOf(p), "config_gen.go"), Data: data})
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
func render(p *Plan) ([]byte, error) {
	if len(p.Fields) == 0 {
		return nil, fmt.Errorf("%s has no settings in it", p.Type)
	}
	cfg := p.imps.name(configPkg)

	var b strings.Builder
	b.WriteString(gen.Header(Name, Version, p.Source))
	fmt.Fprintf(&b, "\npackage %s\n\n", p.Pkg)
	writeImports(&b, p.Imports())

	fmt.Fprintf(&b, "// %sFields are the settings %s has, in the order they are declared.\n",
		lower(p.Type), p.Type)
	fmt.Fprintf(&b, "var %sFields = []%s.FieldDoc{\n", lower(p.Type), cfg)
	for _, f := range p.Fields {
		fmt.Fprintf(&b, "\t{Field: %s.Field{Name: %q, Path: %q", cfg, f.Name, f.Path)
		if f.Env != "" {
			fmt.Fprintf(&b, ", Env: %q", f.Env)
		}
		if f.Default != "" {
			fmt.Fprintf(&b, ", Default: %q", f.Default)
		}
		if f.Secret {
			b.WriteString(", Secret: true")
		}
		fmt.Fprintf(&b, "}, Type: %q", f.Type)
		if f.Doc != "" {
			fmt.Fprintf(&b, ", Doc: %q", f.Doc)
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n\n")

	writeLoad(&b, p, cfg)
	writeValues(&b, p)
	writeDescribe(&b, p, cfg)
	writeDiff(&b, p, cfg)
	writeRedact(&b, p)

	return []byte(b.String()), nil
}

// writeImports writes the block. There is always at least the config package
// in it, so there is no empty case to think about.
func writeImports(b *strings.Builder, lines []importLine) {
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

func writeLoad(b *strings.Builder, p *Plan, cfg string) {
	fields := lower(p.Type) + "Fields"

	fmt.Fprintf(b, "// Load%s reads a %s from the places src names.\n//\n", p.Type, p.Type)
	fmt.Fprintf(b, "// Every setting that will not read is reported, rather than the first one,\n")
	fmt.Fprintf(b, "// and so is every setting in a file that this struct has no field for.\n")
	fmt.Fprintf(b, "func Load%s(src %s.Sources) (*%s, error) {\n", p.Type, cfg, p.Type)
	fmt.Fprintf(b, "\tl, err := %s.Open(src)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n", cfg)
	fmt.Fprintf(b, "\tc := new(%s)\n", p.Type)
	for i, f := range p.Fields {
		fmt.Fprintf(b, "\t%s.Get(l, &c.%s, %s[%d].Field, %s)\n", cfg, f.Name, fields, i, f.Parse)
	}
	fmt.Fprintf(b, "\n\tif err := l.Err(); err != nil {\n\t\treturn nil, err\n\t}\n")
	fmt.Fprintf(b, "\tif err := l.Check(); err != nil {\n\t\treturn nil, err\n\t}\n")
	b.WriteString("\treturn c, nil\n}\n\n")
}

func writeValues(b *strings.Builder, p *Plan) {
	fmt.Fprintf(b, "// %sValues is what a %s holds now, one entry per field, in the same\n",
		lower(p.Type), p.Type)
	fmt.Fprintf(b, "// order as %sFields.\n", lower(p.Type))
	fmt.Fprintf(b, "func (c *%s) %sValues() []string {\n\treturn []string{\n", p.Type, lower(p.Type))
	for _, f := range p.Fields {
		fmt.Fprintf(b, "\t\t%s(c.%s),\n", f.Show, f.Name)
	}
	b.WriteString("\t}\n}\n\n")
}

func writeDescribe(b *strings.Builder, p *Plan, cfg string) {
	fmt.Fprintf(b, "// Describe is every setting, what it is for, and what it is set to now,\n")
	fmt.Fprintf(b, "// with secrets hidden. It is what config:show and config:doc print.\n")
	fmt.Fprintf(b, "func (c *%s) Describe() []%s.FieldDoc {\n", p.Type, cfg)
	fmt.Fprintf(b, "\treturn %s.Describe(%sFields, c.%sValues())\n}\n\n", cfg, lower(p.Type), lower(p.Type))
}

func writeDiff(b *strings.Builder, p *Plan, cfg string) {
	fmt.Fprintf(b, "// Diff is every setting this and other disagree about, in the order the\n")
	fmt.Fprintf(b, "// fields are declared. A secret that changed says so without either value\n")
	fmt.Fprintf(b, "// being printed.\n")
	fmt.Fprintf(b, "func (c *%s) Diff(other *%s) []%s.Change {\n", p.Type, p.Type, cfg)
	fmt.Fprintf(b, "\treturn %s.Diff(%sFields, c.%sValues(), other.%sValues())\n}\n\n",
		cfg, lower(p.Type), lower(p.Type), lower(p.Type))
}

func writeRedact(b *strings.Builder, p *Plan) {
	fmt.Fprintf(b, "// Redact is a copy with every secret taken out of it, safe to print, log or\n")
	fmt.Fprintf(b, "// send somewhere. A secret that is text becomes three stars and a secret of\n")
	fmt.Fprintf(b, "// any other type becomes its zero value.\n//\n")
	fmt.Fprintf(b, "// Every secret is named here rather than found by walking the struct, so\n")
	fmt.Fprintf(b, "// adding one shows up as a line in the diff of this file and marking one by\n")
	fmt.Fprintf(b, "// mistake shows up the same way.\n")
	fmt.Fprintf(b, "func (c *%s) Redact() *%s {\n", p.Type, p.Type)
	fmt.Fprintf(b, "\tout := new(%s)\n\t*out = *c\n", p.Type)

	// A plain copy shares whatever the original points at, so anything that
	// holds a pointer is cloned. A secret is not, because the line below
	// replaces the whole field anyway.
	var cloned bool
	for _, f := range p.Fields {
		if !f.IsCopy || f.Secret {
			continue
		}
		if !cloned {
			b.WriteString("\n\t// A struct copy shares a slice or a map with the original, so these\n")
			b.WriteString("\t// are copied properly before anything is written over.\n")
			cloned = true
		}
		fmt.Fprintf(b, "\tout.%s = %s(c.%s)\n", f.Name, f.Clone, f.Name)
	}

	var secret bool
	for _, f := range p.Fields {
		if !f.Secret {
			continue
		}
		if !secret {
			b.WriteString("\n")
			secret = true
		}
		fmt.Fprintf(b, "\tout.%s = %s\n", f.Name, f.Zero)
	}
	if !secret {
		b.WriteString("\n\t// Nothing in this configuration is marked secret.\n")
	}
	b.WriteString("\n\treturn out\n}\n")
}

// fieldDocs is the doc comment of every struct field in a package, keyed by the
// object the field declares.
//
// go/types does not carry comments, so this walks the syntax once and pairs
// each field with what go/types made of it. A field with no comment of its own
// borrows the one on the line beside it, which is where a short explanation
// usually goes.
func fieldDocs(p *gen.Package) map[types.Object]string {
	out := map[types.Object]string{}
	if p.TypesInfo == nil {
		return out
	}
	for _, file := range p.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok {
				return true
			}
			for _, f := range st.Fields.List {
				text := oneLine(f.Doc.Text())
				if text == "" {
					text = oneLine(f.Comment.Text())
				}
				if text == "" {
					continue
				}
				for _, name := range f.Names {
					if obj, ok := p.TypesInfo.Defs[name]; ok && obj != nil {
						out[obj] = text
					}
				}
			}
			return true
		})
	}
	return out
}

// oneLine is a doc comment as a single line, which is what a table has room
// for. Everything after the first sentence is left for the source.
func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "\n\n"); i >= 0 {
		s = s[:i]
	}
	return strings.Join(strings.Fields(s), " ")
}

// dirOf is the package directory relative to its module, which is what a path
// in a generated file is relative to.
func dirOf(p *gen.Package) string {
	if p.Module == "" || p.PkgPath == p.Module {
		return ""
	}
	return strings.TrimPrefix(p.PkgPath, p.Module+"/")
}

// lower makes the first letter of a name lower case, so the generated
// identifiers that belong to this file rather than to the application are not
// exported.
func lower(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
