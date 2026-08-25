package bindgen

import (
	"fmt"
	"go/token"
	"go/types"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/gen"
)

// webPkg is the package the output calls into. It is written here rather than
// read off a type, because a package with nothing marked in it imports nothing
// and a package with one marked struct imports exactly this.
const webPkg = "github.com/go-mizu/mizu/web"

// maxDepth is how far the walk goes down. A request struct nests a couple of
// levels, an address inside an order and not much further, and anything past
// this is either a mistake or something that wants breaking up. Stopping also
// means the walk is bounded whatever go/types hands over, which matters because
// a package that failed to type check is still walked.
const maxDepth = 12

// A Plan is every struct in one package that asked for a binder, in the order
// they are declared.
type Plan struct {
	Pkg     string   // the package name, for the package clause
	Dir     string   // the package directory, relative to the module
	Source  string   // the file the first marked struct was declared in
	Web     string   // what the output calls the web package
	Structs []Struct // in declaration order
	Errors  []error  // everything wrong, in the order found

	imps *imports
	fset *token.FileSet
}

// A Struct is one type that carries the marker.
type Struct struct {
	Type   string  // the Go type, as in Listing
	Lax    bool    // the struct embeds web.AllowUnknown
	Values bool    // something is read from the query string or a form
	Fields []Field // every leaf, flattened, in declaration order
	Vars   []Var   // the accumulators the values loop appends to
}

// A Var is one accumulator, declared before the values loop and written into
// the struct after it.
type Var struct {
	Name string
	Type string
}

// A Source is where one field is read from.
type Source int

const (
	FromValues Source = iota // the query string, and a form in the body
	FromPath
	FromHeader
	FromCookie
	FromFile // a file part of a multipart form
)

// A Kind is which of the web helpers turns a string into this field.
type Kind int

const (
	Assign Kind = iota // a string or a []byte, written straight in
	Int                // and every defined type over one
	Uint
	Float
	Bool
	Time
	Duration
	Text // an encoding.TextUnmarshaler
)

// A Field is one leaf of the struct and where its value comes from.
type Field struct {
	Name  string // the name the request uses, which is not the field's name
	Src   Source
	Go    string   // the field, as v.Address.City
	Prep  []string // the pointers to fill in before Go can be written to
	Kind  Kind
	Ptr   bool   // the value sits behind a pointer
	List  bool   // a slice, which takes every value sent under the name
	Type  string // the value's type, as the output spells it
	Slice string // the field's own type when List, as the output spells it
	Var   string // the accumulator, for a list read from the values
}

// Analyze works out what every marked struct in a package needs, without
// writing anything. It returns a plan even when there are errors in it, so a
// caller can report all of them at once.
func Analyze(pkg *gen.Package, targets []gen.Target) *Plan {
	p := &Plan{
		Pkg:  pkg.Name,
		Dir:  dirOf(pkg),
		imps: newImports(pkg.PkgPath),
		fset: pkg.Fset,
	}
	p.Web = p.imps.name(webPkg)

	for _, t := range targets {
		if !marked(t) {
			continue
		}
		if p.Source == "" {
			p.Source = path.Join(p.Dir, filepath.Base(t.Pos().Filename))
		}
		p.structure(t)
	}
	return p
}

// Imports are the packages the generated file has to import, in the order the
// output writes them.
func (p *Plan) Imports() []importLine { return p.imps.lines() }

func (p *Plan) errf(pos token.Position, format string, args ...any) {
	p.Errors = append(p.Errors, fmt.Errorf("%s: %s", pos, fmt.Sprintf(format, args...)))
}

// structure reads one marked declaration and adds what it found to the plan.
func (p *Plan) structure(t gen.Target) {
	pos := t.Pos()
	if t.Object == nil {
		p.errf(pos, "the bind marker is on something with no name")
		return
	}
	tn, ok := t.Object.(*types.TypeName)
	if !ok {
		p.errf(pos, "%s is marked to bind and is not a type", t.Name())
		return
	}
	st, ok := tn.Type().Underlying().(*types.Struct)
	if !ok {
		p.errf(pos, "%s is marked to bind and is not a struct", t.Name())
		return
	}

	before := len(p.Errors)
	s := &Struct{Type: t.Name()}
	w := &walker{plan: p, st: s, seen: []types.Type{tn.Type()}}
	w.walk(st, "", "v", nil, pos, true)

	if len(p.Errors) > before {
		return
	}
	// Nothing to read is reported here rather than when the file is written,
	// because here is where the struct is and a message about a declaration says
	// which one.
	if len(s.Fields) == 0 {
		p.errf(pos, "%s is marked to bind and has no fields the request can fill in", t.Name())
		return
	}
	p.Structs = append(p.Structs, *s)
}

// A walker holds the state of one walk down a struct.
type walker struct {
	plan  *Plan
	st    *Struct
	seen  []types.Type // the struct types on the way here, so a tree stops
	names []claim      // the names already read from the query or the form
	depth int
}

// A claim is one name and the field that took it, for the duplicate check.
type claim struct {
	name  string
	field string
}

// walk adds every field of st to the struct being planned, following embedded
// and nested structs down.
//
// prefix is what a nested struct's fields are named under, so a City inside an
// Address is address.city. goPath is the expression the output writes to reach
// this struct, and prep is what has to be allocated before writing through it.
func (w *walker) walk(st *types.Struct, prefix, goPath string, prep []string, pos token.Position, top bool) {
	if w.depth > maxDepth {
		w.plan.errf(pos, "%s nests more than %d deep, which is further down than a request goes",
			w.st.Type, maxDepth)
		return
	}
	w.depth++
	defer func() { w.depth-- }()

	for i := range st.NumFields() {
		f := st.Field(i)
		if !f.Exported() && !f.Embedded() {
			// Generated code cannot reach it, and a request struct with a
			// private field usually means the field is not part of the request.
			continue
		}

		// The marker is a marker. It has no fields and nothing sends one, and
		// walking into it would only be a slower way of finding that out. Only
		// the struct itself is looked at, which is the rule web.AllowUnknown is
		// documented with.
		if isNamed(f.Type(), webPkg, "AllowUnknown") {
			if top {
				w.st.Lax = true
			}
			continue
		}
		w.field(f, reflect.StructTag(st.Tag(i)), prefix, goPath, prep)
	}
}

func (w *walker) field(f *types.Var, tag reflect.StructTag, prefix, goPath string, prep []string) {
	p := w.plan
	pos := p.fset.Position(f.Pos())

	name, src, tagged, ok := nameOf(tag, f.Name())
	if !ok {
		return
	}
	expr := goPath + "." + f.Name()

	ft := f.Type()
	list := false
	if sl, ok := ft.Underlying().(*types.Slice); ok && !isBytes(sl) {
		list, ft = true, sl.Elem()
	}

	// An upload is a handle to a file that is somewhere else, so it binds
	// through a pointer and a field holding one by value is a mistake rather
	// than a struct worth walking into. Saying so is better than filling in its
	// Filename from the query string, which is what walking into it would do.
	if isNamed(ft, webPkg, "Upload") {
		p.errf(pos, "%s is a web.Upload, and an upload binds to a *web.Upload", w.path(expr))
		return
	}

	// An upload is a file part rather than a value, and it is the one field
	// whose source comes from its type rather than from its tag. A tag that says
	// otherwise is somebody expecting a header to carry a file.
	if isUpload(ft) {
		if src != FromValues {
			p.errf(pos, "%s is an upload and is tagged to arrive somewhere other than a form, and a file arrives in a form and nowhere else", w.path(expr))
			return
		}
		fd := Field{Name: prefix + name, Src: FromFile, Go: expr, Prep: prep, List: list}
		if list {
			fd.Slice = p.imps.typeString(f.Type())
		}
		w.add(fd, pos)
		return
	}

	// Everything below writes through at most one pointer, allocating it when
	// something arrives for the field. A chain of them is a shape a request has
	// no way to mean, and writing to the end of one takes a parenthesis the
	// reader has to unpick.
	base, depth := ft, 0
	for {
		ptr, ok := base.Underlying().(*types.Pointer)
		if !ok {
			break
		}
		base, depth = ptr.Elem(), depth+1
	}
	if depth > 1 {
		p.errf(pos, "%s is a %s, and binding writes through one pointer and not two",
			w.path(expr), p.imps.docString(f.Type()))
		return
	}

	kind, ok := kindOf(base)
	if !ok {
		// Not a value a string turns into. A struct is worth going into, and
		// anything else is worth saying out loud only when somebody asked for it
		// by name.
		nested, isStruct := base.Underlying().(*types.Struct)
		if list || !isStruct {
			if tagged {
				p.errf(pos, "%s is a %s, and nothing turns a string into one",
					w.path(expr), p.imps.docString(f.Type()))
			}
			return
		}

		// A type that holds one of itself stops here, tag or no tag. The tag is
		// not the mistake: a query string has no way to carry a tree, and the
		// body decoder that does handles it without a plan from this.
		if w.seenType(base) {
			return
		}

		under := prefix
		if !f.Embedded() || tagged {
			under = prefix + name + "."
		}
		childPrep := prep
		if depth == 1 {
			childPrep = append(prep[:len(prep):len(prep)],
				"if "+expr+" == nil {",
				"\t"+expr+" = new("+p.imps.typeString(base)+")",
				"}")
		}
		w.seen = append(w.seen, base)
		w.walk(nested, under, expr, childPrep, pos, false)
		w.seen = w.seen[:len(w.seen)-1]
		return
	}

	fd := Field{
		Name: name,
		Src:  src,
		Go:   expr,
		Prep: prep,
		Kind: kind,
		Ptr:  depth == 1,
		List: list,
	}
	// Only the query and the form have a prefix on them. A path parameter, a
	// header and a cookie are flat namespaces the request shares with everything
	// else, and address.x-country is not the name of a header wherever the field
	// happens to sit.
	if src == FromValues {
		fd.Name = prefix + name
	}
	// A type is spelled only where the output writes one, since asking for the
	// name of a type is what adds its package to the import block.
	if list {
		fd.Slice = p.imps.typeString(f.Type())
		fd.Type = p.imps.typeString(base)
	} else if fd.Ptr || kind == Assign {
		fd.Type = p.imps.typeString(base)
	}
	w.add(fd, pos)
}

// add records a field, having checked that nothing else answers to its name.
//
// Two fields reading one name from the query or the form are two cases of one
// switch on the same string, which is the one thing the output cannot write. It
// is a mistake either way: whichever field is second can never hold anything
// the first one does not.
func (w *walker) add(f Field, pos token.Position) {
	if f.Src != FromValues {
		w.st.Fields = append(w.st.Fields, f)
		return
	}
	for _, c := range w.names {
		if c.name == f.Name {
			w.plan.errf(pos, "%s and %s are both %s", c.field, w.path(f.Go), f.Name)
			return
		}
	}
	w.names = append(w.names, claim{name: f.Name, field: w.path(f.Go)})

	w.st.Values = true
	if f.List {
		f.Var = w.varName(f.Go)
		w.st.Vars = append(w.st.Vars, Var{Name: f.Var, Type: f.Slice})
	}
	w.st.Fields = append(w.st.Fields, f)
}

// seenType reports whether a struct type is already on the way here.
func (w *walker) seenType(t types.Type) bool {
	for _, s := range w.seen {
		if types.Identical(s, t) {
			return true
		}
	}
	return false
}

// path is a field as somebody reading the source would name it, so the field
// expression v.Address.City is Order.Address.City in a message.
func (w *walker) path(expr string) string {
	return w.st.Type + strings.TrimPrefix(expr, "v")
}

// taken are the names the generated function already uses, which an accumulator
// must not be one of.
var taken = map[string]bool{
	"b": true, "c": true, "v": true, "e": true, "s": true, "ok": true,
	"name": true, "value": true, "got": true, "list": true, "files": true,
}

// varName is the accumulator a list field appends to, named after where the
// field sits, so the Tags inside an Address is addressTags.
func (w *walker) varName(expr string) string {
	want := lower(strings.ReplaceAll(strings.TrimPrefix(expr, "v."), ".", ""))
	if want == "" || taken[want] {
		want += "List"
	}
	name := want
	for n := 2; w.hasVar(name); n++ {
		name = want + strconv.Itoa(n)
	}
	return name
}

func (w *walker) hasVar(name string) bool {
	for _, v := range w.st.Vars {
		if v.Name == name {
			return true
		}
	}
	return false
}

// nameOf is the name the request uses for a field, and where it is read from.
//
// It is the rule in web.Bind written a second time, and the two have to agree.
// A path, header or cookie tag says so and is the whole answer. A form or query
// tag names the field in the query string and in a form body, which net/http
// keeps together and so does this. With none of those, the name comes from the
// json tag and then from the field's own name in snake case, so a struct
// written for a JSON body binds from a query string without a second set of
// tags on it.
//
// The false result is a field binding leaves alone, which is bind:"-" and
// json:"-".
func nameOf(tag reflect.StructTag, field string) (name string, src Source, tagged, ok bool) {
	if v, has := tag.Lookup("bind"); has && tagName(v) == "-" {
		return "", 0, false, false
	}

	for _, t := range []struct {
		tag string
		src Source
	}{
		{"path", FromPath},
		{"header", FromHeader},
		{"cookie", FromCookie},
		{"form", FromValues},
		{"query", FromValues},
	} {
		v, has := tag.Lookup(t.tag)
		if !has {
			continue
		}
		switch name := tagName(v); name {
		case "-":
			return "", 0, false, false
		case "":
			return snake(field), t.src, true, true
		default:
			return name, t.src, true, true
		}
	}

	if v, has := tag.Lookup("json"); has {
		name := tagName(v)
		if name == "-" {
			return "", 0, false, false
		}
		if name != "" {
			return name, FromValues, false, true
		}
	}
	return snake(field), FromValues, false, true
}

// tagName is the name out of a struct tag, which is everything in front of the
// first comma.
func tagName(v string) string {
	name, _, _ := strings.Cut(v, ",")
	return name
}

// snake is a Go field name as a request would spell it, so PerPage is per_page
// and UserID is user_id.
//
// It is the same function web.Bind uses to name an untagged field, copied
// rather than shared because that one is unexported and the two have to give
// the same answer for the generated binder to bind the same request.
func snake(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)

	for i := range len(s) {
		c := s[i]
		if c < 'A' || c > 'Z' {
			b.WriteByte(c)
			continue
		}
		// A capital starts a new word unless it is in the middle of a run of
		// them, which is what makes HTTPServer two words rather than nine.
		if i > 0 && (!isUpper(s[i-1]) || (i+1 < len(s) && isLower(s[i+1]))) {
			b.WriteByte('_')
		}
		b.WriteByte(c + 'a' - 'A')
	}
	return b.String()
}

func isUpper(c byte) bool { return c >= 'A' && c <= 'Z' }
func isLower(c byte) bool { return c >= 'a' && c <= 'z' }

// textUnmarshaler is encoding.TextUnmarshaler, built here rather than looked up
// so that the generator does not need the encoding package loaded to recognise
// a type that implements it.
var textUnmarshaler = types.NewInterfaceType([]*types.Func{
	types.NewFunc(token.NoPos, nil, "UnmarshalText", types.NewSignatureType(
		nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, nil, "", types.NewSlice(types.Typ[types.Byte]))),
		types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Universe.Lookup("error").Type())),
		false,
	)),
}, nil).Complete()

// kindOf is which helper turns a string into a value of type t, and whether any
// of them does.
//
// The order is the order web.Bind's setterFor uses, and it matters: a time.Time
// reads itself out of text, and reading one that way would take the RFC 3339 it
// was written for and refuse the three shapes a date input sends.
func kindOf(t types.Type) (Kind, bool) {
	switch {
	case isNamed(t, "time", "Time"):
		return Time, true
	case isNamed(t, "time", "Duration"):
		return Duration, true
	case types.Implements(types.NewPointer(t), textUnmarshaler):
		return Text, true
	}

	if sl, ok := t.Underlying().(*types.Slice); ok && isBytes(sl) {
		return Assign, true
	}
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return 0, false
	}
	switch basic.Kind() {
	case types.String:
		return Assign, true
	case types.Bool:
		return Bool, true
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
		return Int, true
	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
		return Uint, true
	case types.Float32, types.Float64:
		return Float, true
	}
	return 0, false
}

// isBytes reports whether a slice is a []byte rather than a slice of something
// that is one byte wide and means something else.
func isBytes(sl *types.Slice) bool {
	basic, ok := sl.Elem().(*types.Basic)
	return ok && basic.Kind() == types.Byte
}

// isUpload reports whether a type is exactly *web.Upload, which is the one
// field whose source comes from its type.
func isUpload(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	return ok && isNamed(ptr.Elem(), webPkg, "Upload")
}

// isNamed reports whether t is the named type pkg.name, and not something
// declared over it.
func isNamed(t types.Type, pkg, name string) bool {
	n, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := n.Obj()
	return obj.Name() == name && obj.Pkg() != nil && obj.Pkg().Path() == pkg
}

// dirOf is the package directory relative to its module, which is what a path
// in a generated file is relative to.
func dirOf(p *gen.Package) string {
	if p.Module == "" || p.PkgPath == p.Module {
		return ""
	}
	return strings.TrimPrefix(p.PkgPath, p.Module+"/")
}

// lower makes the first letter of a name lower case, which is what turns a
// field's place in the struct into the name of a local variable.
func lower(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
