package configgen

import (
	"fmt"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-mizu/mizu/gen"
)

// A Plan is everything the generator worked out about one configuration
// struct, in the order the fields are declared.
type Plan struct {
	Pkg    string    // the package the struct is in
	Type   string    // the name of the struct type
	Source string    // the file it was declared in, relative to the module
	Fields []Field   // one per leaf, in declaration order
	Errors []error   // everything wrong with the struct, in the order found
	imps   *imports  // the packages the output has to import
	seen   []*seenAt // paths already used, for the duplicate check
}

// A Field is one leaf of the struct: something that holds a value rather than
// more fields.
type Field struct {
	Name    string // the Go name, as in Database.MaxOpenConns
	Path    string // the TOML path, as in database.max_open_conns
	Env     string // the environment variable, empty for none
	Default string // the default as written on the tag
	Secret  bool
	Doc     string // the field's doc comment, as one line

	Type   string // the Go type, written for the output's imports
	Parse  string // the parser expression, as config.Duration
	Show   string // how to write the value out, as config.Show
	Zero   string // the value a secret is replaced with in Redact
	Clone  string // how Redact copies it, empty when a plain copy is enough
	IsCopy bool   // whether Clone is set, so templates need no string test
}

type seenAt struct {
	path string
	name string
}

// A walker holds the state of one walk down a struct.
type walker struct {
	plan  *Plan
	docs  map[types.Object]string
	fset  *token.FileSet
	depth int
}

// maxDepth is how far the walk goes down. Configuration nests a few levels,
// app.name and http.tls.cert and not much deeper, and a struct past this is
// either a mistake or something that wants breaking up. Stopping also means
// the walk is bounded whatever go/types hands over, which matters because a
// package that failed to type check is still walked.
const maxDepth = 12

// Analyze works out what a configuration struct needs, without writing
// anything. It returns a plan even when there are errors in it, so a caller
// can report all of them at once.
func Analyze(t gen.Target, docs map[types.Object]string) *Plan {
	p := &Plan{
		Pkg:  t.Pkg.Name,
		Type: t.Name(),
		imps: newImports(t.Pkg.Types.Path()),
	}
	p.imps.add(configPkg)

	st, ok := structOf(t.Object)
	if !ok {
		p.errf(t.Pos(), "%s is marked as configuration and is not a struct", t.Name())
		return p
	}

	w := &walker{plan: p, docs: docs, fset: t.Pkg.Fset}
	w.walk(st, "", "", t.Pos())
	return p
}

// Imports are the packages the generated file has to import, in the order the
// output writes them.
func (p *Plan) Imports() []importLine { return p.imps.lines() }

func (p *Plan) errf(pos token.Position, format string, args ...any) {
	p.Errors = append(p.Errors, fmt.Errorf("%s: %s", pos, fmt.Sprintf(format, args...)))
}

// walk visits every field of a struct, going into the ones that hold more
// fields and recording the ones that hold values.
func (w *walker) walk(st *types.Struct, name, path string, pos token.Position) {
	if w.depth > maxDepth {
		w.plan.errf(pos, "%s nests more than %d deep, which is further than configuration goes", name, maxDepth)
		return
	}
	w.depth++
	defer func() { w.depth-- }()

	for i := range st.NumFields() {
		f := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))
		if !f.Exported() {
			// Generated code cannot reach it, and a configuration struct with
			// a private field usually means the field is not configuration.
			continue
		}
		w.field(f, tag, name, path)
	}
}

func (w *walker) field(f *types.Var, tag reflect.StructTag, name, path string) {
	pos := w.fset.Position(f.Pos())

	childName := join(name, f.Name(), ".")
	childPath := path
	if f.Embedded() {
		// An embedded struct adds no segment, so the fields of an embedded
		// mizu.Base sit where the application expects to find them.
		childName = name
	} else {
		childPath = join(path, segment(f.Name(), tag), ".")
	}

	// A struct that has no parser of its own holds more fields rather than a
	// value, and the walk goes into it.
	if st, ok := f.Type().Underlying().(*types.Struct); ok && parserFor(f.Type(), w.plan.imps) == nil {
		w.walk(st, childName, childPath, pos)
		return
	}

	parse := parserFor(f.Type(), w.plan.imps)
	if parse == nil {
		w.plan.errf(pos, "%s is a %s, which no parser reads", childName, types.TypeString(f.Type(), types.RelativeTo(f.Pkg())))
		return
	}

	field := Field{
		Name:    childName,
		Path:    childPath,
		Default: tag.Get("default"),
		Secret:  tag.Get("secret") == "true",
		Doc:     w.docs[f],
		Type:    w.plan.imps.docString(f.Type()),
		Parse:   parse.expr,
		Show:    parse.show,
	}
	// Zero is written out only for a secret, and asking for it costs an
	// import, so a field that is not one does not ask.
	if field.Secret {
		field.Zero = zeroOf(f.Type(), w.plan.imps)
	}
	field.Clone, field.IsCopy = cloneOf(f.Type(), w.plan.imps)
	field.Env = envName(childPath, tag)

	if err := checkDefault(field); err != nil {
		w.plan.errf(pos, "%s: %v", childName, err)
	}
	if other := w.plan.claim(field); other != "" {
		w.plan.errf(pos, "%s and %s are both %s", other, childName, field.Path)
	}
	w.plan.Fields = append(w.plan.Fields, field)
}

// claim records a path and reports the field that already had it, since two
// fields answering to one name means one of them can never be set.
func (p *Plan) claim(f Field) string {
	for _, s := range p.seen {
		if s.path == f.Path {
			return s.name
		}
	}
	p.seen = append(p.seen, &seenAt{path: f.Path, name: f.Name})
	return ""
}

// checkDefault refuses the two forms from doc 05 that are not implemented,
// rather than reading them as the text they are written as.
func checkDefault(f Field) error {
	switch {
	case strings.Contains(f.Default, "|"):
		return fmt.Errorf("the default %q means one value in local and another elsewhere, which arrives with mizu.Base", f.Default)
	case strings.Contains(f.Default, "{") && strings.Contains(f.Default, "}"):
		return fmt.Errorf("the default %q refers to another field, which arrives with mizu.Base", f.Default)
	}
	return nil
}

// envName is the variable a field answers to. The env tag names it outright,
// env:"-" says there is none, and otherwise it is the path shouted.
func envName(path string, tag reflect.StructTag) string {
	if name, ok := tag.Lookup("env"); ok {
		if name == "-" {
			return ""
		}
		return name
	}
	return strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
}

// segment is one part of a path. The toml tag names it, and otherwise the Go
// name is lowered with underscores, so MaxOpenConns becomes max_open_conns and
// DSN becomes dsn.
func segment(name string, tag reflect.StructTag) string {
	if v, ok := tag.Lookup("toml"); ok {
		if v, _, _ := strings.Cut(v, ","); v != "" && v != "-" {
			return v
		}
	}
	runes := []rune(name)
	var b strings.Builder
	for i, r := range runes {
		if !unicode.IsUpper(r) {
			b.WriteRune(r)
			continue
		}
		prevIsLower := i > 0 && !unicode.IsUpper(runes[i-1])
		nextIsLower := i+1 < len(runes) && !unicode.IsUpper(runes[i+1])
		if i > 0 && (prevIsLower || nextIsLower) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func join(prefix, name, sep string) string {
	if prefix == "" {
		return name
	}
	return prefix + sep + name
}

func structOf(obj types.Object) (*types.Struct, bool) {
	st, ok := obj.Type().Underlying().(*types.Struct)
	return st, ok
}
