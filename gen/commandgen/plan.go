package commandgen

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-mizu/mizu/gen"
)

// A Plan is every command in one package, in the order they are declared.
type Plan struct {
	Pkg      string    // the package name, for the package clause
	Dir      string    // the package directory, relative to the module
	Source   string    // the file the first command was declared in
	Console  string    // what the output calls the console package
	Commands []Command // in declaration order
	Errors   []error   // everything wrong, in the order found

	types *types.Package // for naming a type in a message the way the reader would
}

// A Command is one struct that carries the marker.
type Command struct {
	Type   string // the Go type, as in UsersPrune
	Name   string // what somebody types, as in users:prune
	Desc   string
	Long   string
	Hidden bool
	Flags  []Flag
	Args   []Arg
}

// A Flag is one field with a flag tag on it.
type Flag struct {
	Field    string // the Go field, as in DryRun
	Name     string // as in dry-run
	Short    rune   // zero for none
	Desc     string
	Env      string
	Default  string
	Required bool
	Hidden   bool
	Value    string // the expression that builds the console.Value
}

// An Arg is one field with an arg tag on it.
type Arg struct {
	Field    string
	Name     string
	Desc     string
	Default  string
	Required bool
	Rest     bool
	Value    string

	at   int  // the index written on the tag, for ordering and the gap check
	list bool // the field is a slice, which is what an argument taking the rest needs
}

// Analyze works out what every command in a package needs, without writing
// anything. It returns a plan even when there are errors in it, so a caller
// can report all of them at once.
func Analyze(pkg *gen.Package, targets []gen.Target) *Plan {
	p := &Plan{
		Pkg:     pkg.Name,
		Dir:     dirOf(pkg),
		Console: consoleName(pkg),
		types:   pkg.Types,
	}
	docs := fieldDocs(pkg)

	for _, t := range targets {
		m, ok := marker(t)
		if !ok {
			continue
		}
		if p.Source == "" {
			p.Source = path.Join(p.Dir, filepath.Base(t.Pos().Filename))
		}
		if cmd := p.command(t, m, docs); cmd != nil {
			p.Commands = append(p.Commands, *cmd)
		}
	}
	return p
}

func (p *Plan) errf(pos token.Position, format string, args ...any) {
	p.Errors = append(p.Errors, fmt.Errorf("%s: %s", pos, fmt.Sprintf(format, args...)))
}

// command reads one marked struct. It returns nil when there is nothing worth
// writing, having recorded why.
func (p *Plan) command(t gen.Target, m gen.Marker, docs map[types.Object]string) *Command {
	pos := t.Pos()
	if t.Object == nil {
		p.errf(pos, "the command marker is on something with no name")
		return nil
	}

	st, ok := t.Object.Type().Underlying().(*types.Struct)
	if !ok {
		p.errf(pos, "%s is marked as a command and is not a struct", t.Name())
		return nil
	}

	cmd := &Command{Type: t.Name(), Hidden: m.Flag("hidden")}
	cmd.Name, _ = m.Get("name")
	cmd.Desc, _ = m.Get("desc")
	cmd.Long, _ = m.Get("long")

	switch {
	case cmd.Name == "":
		p.errf(pos, "%s has no name, which is what somebody types to run it: //mizu:command name=%q",
			cmd.Type, suggestName(cmd.Type))
		return nil
	case strings.ContainsAny(cmd.Name, " \t"):
		p.errf(pos, "the name %q has a space in it, and a command is one word", cmd.Name)
		return nil
	}
	if cmd.Desc == "" {
		cmd.Desc = unprefix(docs[t.Object], cmd.Type)
	}

	p.checkMethods(t, cmd, pos)

	for i := range st.NumFields() {
		p.field(cmd, st.Field(i), reflect.StructTag(st.Tag(i)), docs, t.Pkg.Fset)
	}
	p.checkFlags(cmd, pos)
	p.checkArgs(cmd, pos)
	return cmd
}

// checkFlags reports two flags that would answer to the same thing. console.Parse
// panics on the pair, and it is a mistake that survives review easily, since the
// two tags are usually a long way apart in the struct.
func (p *Plan) checkFlags(cmd *Command, pos token.Position) {
	names := map[string]string{}
	shorts := map[rune]string{}
	for _, f := range cmd.Flags {
		if first, ok := names[f.Name]; ok {
			p.errf(pos, "%s.%s and %s.%s are both --%s", cmd.Type, first, cmd.Type, f.Field, f.Name)
		}
		names[f.Name] = f.Field

		if f.Short == 0 {
			continue
		}
		if first, ok := shorts[f.Short]; ok {
			p.errf(pos, "%s.%s and %s.%s are both -%c", cmd.Type, first, cmd.Type, f.Field, f.Short)
		}
		shorts[f.Short] = f.Field
	}
}

// checkMethods reports a struct that will not satisfy console.Command, since
// the generated Spec is half of the interface and the half nobody wrote is the
// half that breaks the build with a message about an interface.
func (p *Plan) checkMethods(t gen.Target, cmd *Command, pos token.Position) {
	ptr := types.NewPointer(t.Object.Type())

	// A Spec in the file this generator writes is the last run's, not somebody
	// else's, and the whole file is about to be replaced.
	if m, _, _ := types.LookupFieldOrMethod(ptr, true, t.Object.Pkg(), "Spec"); m != nil &&
		filepath.Base(t.Pkg.Fset.Position(m.Pos()).Filename) != outputFile {
		p.errf(pos, "%s already has a Spec method, so there is nothing here to generate", cmd.Type)
	}

	m, _, _ := types.LookupFieldOrMethod(ptr, true, t.Object.Pkg(), "Run")
	fn, ok := m.(*types.Func)
	if !ok {
		p.errf(pos, "%s has no Run method, and a command is a struct that says what it takes and does it", cmd.Type)
		return
	}
	if got := signature(fn); got != runSignature {
		p.errf(pos, "%s.Run is %s, and a command's Run is %s", cmd.Type, got, runSignature)
	}
}

// runSignature is what console.Command asks for.
const runSignature = "func(context.Context, *console.IO) error"

// signature writes a method's shape without the parameter names, since what
// somebody called their context is not the mistake being looked for.
func signature(fn *types.Func) string {
	sig := fn.Signature()
	name := func(pkg *types.Package) string { return pkg.Name() }

	var b strings.Builder
	b.WriteString("func(")
	for i := range sig.Params().Len() {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(types.TypeString(sig.Params().At(i).Type(), name))
	}
	b.WriteString(")")
	for i := range sig.Results().Len() {
		switch i {
		case 0:
			b.WriteString(" ")
			if sig.Results().Len() > 1 {
				b.WriteString("(")
			}
		default:
			b.WriteString(", ")
		}
		b.WriteString(types.TypeString(sig.Results().At(i).Type(), name))
	}
	if sig.Results().Len() > 1 {
		b.WriteString(")")
	}
	return b.String()
}

// field reads one struct field. A field with no flag and no arg tag is not part
// of the command line, which is how a command holds a field of its own.
func (p *Plan) field(cmd *Command, f *types.Var, tag reflect.StructTag, docs map[types.Object]string, fset *token.FileSet) {
	pos := fset.Position(f.Pos())
	flagTag, isFlag := tag.Lookup("flag")
	argTag, isArg := tag.Lookup("arg")

	switch {
	case isFlag && isArg:
		p.errf(pos, "%s is tagged as both a flag and an argument", f.Name())
		return
	case !isFlag && !isArg:
		if other := strayTag(tag); other != "" {
			p.errf(pos, "%s has a %s tag and no flag or arg tag, so nothing reads it", f.Name(), other)
		}
		return
	}

	value, err := p.value(f.Type(), "&c."+f.Name(), tag)
	if err != nil {
		// No colon after the name. Everything value reports carries on from
		// it, so the two read as the one sentence the rest of this file
		// writes: Ch is a chan int, which no console.Value reads.
		p.errf(pos, "%s %v", f.Name(), err)
		return
	}

	desc := tag.Get("desc")
	if desc == "" {
		desc = docs[f]
	}

	if isArg {
		p.arg(cmd, f, tag, argTag, desc, value, pos)
		return
	}
	p.flag(cmd, f, tag, flagTag, desc, value, pos)
}

func (p *Plan) flag(cmd *Command, f *types.Var, tag reflect.StructTag, spec, desc, value string, pos token.Position) {
	name, short, _ := strings.Cut(spec, ",")
	if name == "" {
		name = kebab(f.Name())
	}

	flag := Flag{
		Field:    f.Name(),
		Name:     name,
		Desc:     desc,
		Env:      tag.Get("env"),
		Default:  tag.Get("default"),
		Required: tag.Get("required") == "true",
		Hidden:   tag.Get("hidden") == "true",
		Value:    value,
	}

	switch r, size := utf8.DecodeRuneInString(short); {
	case short == "":
	case size != len(short):
		p.errf(pos, "%s has a short flag of %q, and a short flag is one letter", f.Name(), short)
	default:
		flag.Short = r
	}

	// console.Parse panics on this pair, since a required flag with a default
	// can never be missing. Saying so here points at the tag rather than at a
	// binary that dies on its first line.
	if flag.Required && flag.Default != "" {
		p.errf(pos, "%s is required and has a default, and a flag with a default is never missing", f.Name())
	}
	cmd.Flags = append(cmd.Flags, flag)
}

func (p *Plan) arg(cmd *Command, f *types.Var, tag reflect.StructTag, spec, desc, value string, pos token.Position) {
	rest := strings.HasSuffix(spec, "...")
	at, err := strconv.Atoi(strings.TrimSuffix(spec, "..."))
	if err != nil || at < 0 {
		p.errf(pos, "%s has an arg tag of %q, and an argument is its place on the line: arg:\"0\" or arg:\"1...\"", f.Name(), spec)
		return
	}

	_, isList := f.Type().Underlying().(*types.Slice)
	arg := Arg{
		Field:   f.Name(),
		Name:    kebab(f.Name()),
		Desc:    desc,
		Default: tag.Get("default"),
		Rest:    rest,
		Value:   value,
		at:      at,
		list:    isList,
	}

	// An argument is required unless it says otherwise, because that is what
	// writing one on the line means. A default is what says otherwise, and
	// required:"false" says it for an argument that has no useful default.
	arg.Required = arg.Default == "" && tag.Get("required") != "false"
	cmd.Args = append(cmd.Args, arg)
}

// checkArgs puts the arguments in the order they are typed and reports the
// mistakes that only show up once they are all in.
func (p *Plan) checkArgs(cmd *Command, pos token.Position) {
	args := cmd.Args
	for i := range args {
		for j := i + 1; j < len(args); j++ {
			if args[i].at > args[j].at {
				args[i], args[j] = args[j], args[i]
			}
		}
	}

	for i, a := range args {
		switch {
		case a.at != i:
			p.errf(pos, "%s.%s is argument %d and the line has no argument %d, so nothing can reach it",
				cmd.Type, a.Field, a.at, i)
		case a.Rest && i != len(args)-1:
			p.errf(pos, "%s.%s takes the rest of the line and is not the last argument", cmd.Type, a.Field)
		case a.Rest && !a.list:
			// Every word is Set on the same value, so anything that replaces
			// rather than appends keeps the last one and drops the rest.
			p.errf(pos, "%s.%s takes the rest of the line and is not a list, so it would only keep the last word",
				cmd.Type, a.Field)
		case i > 0 && a.Required && !args[i-1].Required:
			p.errf(pos, "%s.%s is required and comes after %s, which is not, so it can never be reached",
				cmd.Type, a.Field, args[i-1].Field)
		}
	}
}

// strayTag is a tag that only means something next to a flag or an arg tag,
// which is worth reporting because the field it is on does nothing.
func strayTag(tag reflect.StructTag) string {
	for _, name := range []string{"default", "desc", "env", "required", "hidden", "enum", "sep", "count"} {
		if _, ok := tag.Lookup(name); ok {
			return name
		}
	}
	return ""
}

// marker finds the command marker on a target.
func marker(t gen.Target) (gen.Marker, bool) {
	for _, m := range t.Markers {
		if m.Name == Name {
			return m, true
		}
	}
	return gen.Marker{}, false
}

// suggestName is what a type would be called if a name had to be guessed, used
// in the message about a marker that has none. It is a suggestion rather than a
// default: UsersPrune could be users:prune or user:sprune, and a command name
// is what people type, so it is written down rather than derived.
func suggestName(typeName string) string {
	words := split(typeName)
	if len(words) < 2 {
		return strings.ToLower(typeName)
	}
	return strings.ToLower(words[0]) + ":" + strings.ToLower(strings.Join(words[1:], ""))
}

// kebab is a Go field name as it is written on a command line, so DryRun is
// dry-run and MaxOpenConns is max-open-conns.
func kebab(name string) string {
	return strings.ToLower(strings.Join(split(name), "-"))
}

// split breaks a Go name into its words, keeping an acronym together, so
// DryRun is Dry Run and ParseHTTPHeader is Parse HTTP Header.
func split(name string) []string {
	var words []string
	runes := []rune(name)
	start := 0
	for i := 1; i < len(runes); i++ {
		prev, cur := runes[i-1], runes[i]
		next := ' '
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if (unicode.IsUpper(cur) && !unicode.IsUpper(prev)) ||
			(unicode.IsUpper(prev) && unicode.IsUpper(cur) && unicode.IsLower(next)) {
			words = append(words, string(runes[start:i]))
			start = i
		}
	}
	return append(words, string(runes[start:]))
}

// dirOf is the package directory relative to its module, which is what a path
// in a generated file is relative to.
func dirOf(p *gen.Package) string {
	if p.Module == "" || p.PkgPath == p.Module {
		return ""
	}
	return strings.TrimPrefix(p.PkgPath, p.Module+"/")
}

// consoleName is what the output calls the console package. It is console
// unless the package being generated into is itself called that, which happens
// exactly once, in this repository, and is worth one line to get right.
func consoleName(p *gen.Package) string {
	if p.Name == "console" {
		return "mizuconsole"
	}
	return "console"
}

// fieldDocs is the doc comment of every type and every struct field in a
// package, keyed by the object it declares.
//
// go/types does not carry comments, so this walks the syntax once and pairs
// each declaration with what go/types made of it. A field with no comment of
// its own borrows the one on the line beside it, which is where a short
// explanation of a flag usually goes.
func fieldDocs(p *gen.Package) map[types.Object]string {
	out := map[types.Object]string{}
	if p.TypesInfo == nil {
		return out
	}
	def := func(name *ast.Ident, text string) {
		if obj, ok := p.TypesInfo.Defs[name]; ok && obj != nil && text != "" {
			out[obj] = text
		}
	}

	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			d, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range d.Specs {
				s, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				// A comment above a parenthesized group is about the group, so
				// it only counts when there is one type in it. That is the rule
				// godoc uses.
				doc := s.Doc
				if doc == nil && len(d.Specs) == 1 {
					doc = d.Doc
				}
				def(s.Name, oneLine(doc.Text()))
			}
		}

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
				for _, name := range f.Names {
					def(name, text)
				}
			}
			return true
		})
	}
	return out
}

// unprefix drops the type's own name from the front of its doc comment, so
// "UsersPrune deletes users who never verified their email" reads as "Deletes
// users who never verified their email" in a list of commands.
//
// Go writes a doc comment starting with the name of what it is about, and a
// command list has the name in the column beside it already. A desc argument on
// the marker is there for when this is not the sentence you want.
func unprefix(doc, typeName string) string {
	rest, ok := strings.CutPrefix(doc, typeName+" ")
	if !ok {
		return doc
	}
	r, size := utf8.DecodeRuneInString(rest)
	if size == 0 {
		return doc
	}
	return string(unicode.ToUpper(r)) + rest[size:]
}

// oneLine is a doc comment as a single line, which is what a help listing has
// room for. Everything after the first sentence is left for the source.
func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "\n\n"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSuffix(strings.Join(strings.Fields(s), " "), ".")
}
