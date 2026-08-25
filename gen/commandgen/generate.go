package commandgen

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/gen"
)

// Name is what this generator is called on the command line and in a marker.
const Name = "command"

// Version is bumped when the shape of the output changes, so a file written by
// an older toolchain is visible as one.
const Version = "1"

// consolePkg is what the output imports. It is written here rather than read
// off a type, because a package with no commands in it imports nothing and a
// package with one imports exactly this.
const consolePkg = "github.com/go-mizu/mizu/console"

// outputFile is what every package's commands land in. It is one file rather
// than one per command, because Commands has to list them all and a list split
// across files is a list that goes out of date.
const outputFile = "commands_gen.go"

// Generate writes the Spec method for every command in the given packages.
//
// A struct asks for one with a //mizu:command marker. Each package gets one
// file holding every command in it, plus the list to hand to console.App.
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
		if len(plan.Errors) > 0 || len(plan.Commands) == 0 {
			continue
		}
		files = append(files, gen.File{
			Path: path.Join(plan.Dir, outputFile),
			Data: render(plan),
		})
	}
	return files, errors.Join(errs...)
}

// render writes the file. Everything it needs is in the plan, so this is the
// only place that knows what the output looks like.
func render(p *Plan) []byte {
	var b strings.Builder
	b.WriteString(gen.Header(Name, Version, p.Source))
	fmt.Fprintf(&b, "\npackage %s\n\n", p.Pkg)

	if p.Console == "console" {
		fmt.Fprintf(&b, "import %q\n\n", consolePkg)
	} else {
		fmt.Fprintf(&b, "import %s %q\n\n", p.Console, consolePkg)
	}

	writeCommands(&b, p)
	for _, cmd := range p.Commands {
		writeSpec(&b, p, cmd)
	}
	return []byte(b.String())
}

// writeCommands writes the list to hand to console.App, so that adding a
// command is a marker and nothing else.
func writeCommands(b *strings.Builder, p *Plan) {
	b.WriteString("// Commands is every command in this package, in the order they are declared.\n")
	b.WriteString("//\n")
	b.WriteString("//\tapp.Add(Commands()...)\n")
	fmt.Fprintf(b, "func Commands() []%s.Command {\n", p.Console)
	fmt.Fprintf(b, "\treturn []%s.Command{\n", p.Console)
	for _, cmd := range p.Commands {
		fmt.Fprintf(b, "\t\t&%s{},\n", cmd.Type)
	}
	b.WriteString("\t}\n}\n")
}

func writeSpec(b *strings.Builder, p *Plan, cmd Command) {
	c := p.Console

	fmt.Fprintf(b, "\n// Spec is what %s is called and what it takes.\n", cmd.Name)
	fmt.Fprintf(b, "func (c *%s) Spec() %s.Spec {\n", cmd.Type, c)
	fmt.Fprintf(b, "\treturn %s.Spec{\n", c)
	fmt.Fprintf(b, "\t\tName: %q,\n", cmd.Name)
	if cmd.Desc != "" {
		fmt.Fprintf(b, "\t\tDesc: %q,\n", cmd.Desc)
	}
	if cmd.Long != "" {
		fmt.Fprintf(b, "\t\tLong: %q,\n", cmd.Long)
	}
	if cmd.Hidden {
		b.WriteString("\t\tHidden: true,\n")
	}

	if len(cmd.Flags) > 0 {
		fmt.Fprintf(b, "\t\tFlags: []%s.Flag{\n", c)
		for _, f := range cmd.Flags {
			b.WriteString("\t\t\t{")
			writeFields(b, []field{
				{"Name", strconv.Quote(f.Name)},
				{"Short", quoteRune(f.Short)},
				{"Desc", strconv.Quote(f.Desc)},
				{"Env", strconv.Quote(f.Env)},
				{"Default", strconv.Quote(f.Default)},
				{"Required", boolOf(f.Required)},
				{"Hidden", boolOf(f.Hidden)},
				{"Value", f.Value},
			})
			b.WriteString("},\n")
		}
		b.WriteString("\t\t},\n")
	}

	if len(cmd.Args) > 0 {
		fmt.Fprintf(b, "\t\tArgs: []%s.Arg{\n", c)
		for _, a := range cmd.Args {
			b.WriteString("\t\t\t{")
			writeFields(b, []field{
				{"Name", strconv.Quote(a.Name)},
				{"Desc", strconv.Quote(a.Desc)},
				{"Default", strconv.Quote(a.Default)},
				{"Required", boolOf(a.Required)},
				{"Rest", boolOf(a.Rest)},
				{"Value", a.Value},
			})
			b.WriteString("},\n")
		}
		b.WriteString("\t\t},\n")
	}

	b.WriteString("\t}\n}\n")
}

// A field is one entry of a composite literal, left out when it holds nothing.
type field struct{ name, value string }

// writeFields writes the entries that say something. A struct literal carrying
// Desc: "" and Required: false for every flag is longer than the declaration it
// came from and harder to read than either.
func writeFields(b *strings.Builder, fields []field) {
	first := true
	for _, f := range fields {
		if f.value == "" || f.value == `""` {
			continue
		}
		if !first {
			b.WriteString(", ")
		}
		first = false
		b.WriteString(f.name + ": " + f.value)
	}
}

func boolOf(b bool) string {
	if b {
		return "true"
	}
	return ""
}

// quoteRune writes a short flag as a rune literal, and nothing for a flag that
// has none.
func quoteRune(r rune) string {
	if r == 0 {
		return ""
	}
	return strconv.QuoteRune(r)
}
