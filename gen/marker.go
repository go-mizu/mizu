package gen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A Marker is one mizu directive from a doc comment.
//
//	//mizu:rpc method=POST path=/v1/orders ability=order.create
//	//mizu:command name="users:prune" standalone
//	//mizu:response 409 ConflictBody
//
// The name is what follows the colon. After it come arguments separated by
// spaces, each either a key and a value joined by an equals sign, or a bare
// word. A bare word is both how a flag is written and how a positional
// argument is written, which is the same thing the go command does with
// //go:build and //go:generate.
//
// A value with a space in it is written in Go quotes, so escapes work the way
// they do everywhere else.
type Marker struct {
	Name string         // "rpc", the part after the colon
	Args []Arg          // in the order they were written
	Pos  token.Position // the start of the comment
	Text string         // the comment as written
}

// An Arg is one argument to a marker. Key is empty for a bare word, and in
// that case Value holds the word.
type Arg struct {
	Key   string
	Value string
}

func (m Marker) String() string { return m.Text }

// Get returns the value of a keyed argument.
func (m Marker) Get(key string) (string, bool) {
	for _, a := range m.Args {
		if a.Key == key {
			return a.Value, true
		}
	}
	return "", false
}

// Flag reports whether a boolean argument is on. It is on when the name
// appears as a bare word, and when it appears as name=true.
//
// Anything else, including name=false and name=yes, is off. A marker that
// wants to tell a typo from an intentional false can read the value with Get
// and decide for itself.
func (m Marker) Flag(name string) bool {
	for _, a := range m.Args {
		switch {
		case a.Key == "" && a.Value == name:
			return true
		case a.Key == name && a.Value == "true":
			return true
		}
	}
	return false
}

// List splits a comma-separated value, as in transport=grpc,connect,http.
// Spaces around each item are trimmed and empty items are dropped, so
// "grpc, ,connect" is two items. A missing key gives nil.
func (m Marker) List(key string) []string {
	v, ok := m.Get(key)
	if !ok {
		return nil
	}
	var out []string
	for item := range strings.SplitSeq(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// Words returns the bare words, in the order they were written. For
// //mizu:response 409 ConflictBody that is "409" and "ConflictBody".
func (m Marker) Words() []string {
	var out []string
	for _, a := range m.Args {
		if a.Key == "" {
			out = append(out, a.Value)
		}
	}
	return out
}

// A Target is a declaration carrying at least one marker.
//
// Node is what the comment was attached to: an [ast.File] for a marker in the
// package comment, an [ast.FuncDecl] for a function or method, an
// [ast.TypeSpec] or [ast.ValueSpec] for a declaration, or an [ast.Field] for a
// struct field or an interface method.
//
// Object is the thing that declaration declares, which is what a generator
// needs to ask questions about the type. It is nil when there is nothing to
// name, which happens for a package comment, for an embedded struct field, and
// for anything in a package that did not type-check.
type Target struct {
	Pkg     *Package
	File    *ast.File
	Node    ast.Node
	Object  types.Object
	Markers []Marker
}

// Name is the declared name, or the package name for a package comment.
func (t Target) Name() string {
	if t.Object != nil {
		return t.Object.Name()
	}
	if f, ok := t.Node.(*ast.File); ok && f.Name != nil {
		return f.Name.Name
	}
	return ""
}

// Pos is where the declaration starts, which is a better thing to point an
// error at than the marker, since the marker is usually right above it.
func (t Target) Pos() token.Position { return t.Pkg.Fset.Position(t.Node.Pos()) }

// Scan finds every marker in the given packages.
//
// Targets come back in source order: packages in the order given, files in the
// order the go command listed them, and declarations in the order they appear.
// Nothing here ranges over a map, because generated output has to be
// byte-identical from one run to the next.
//
// A comment that looks like a marker and is not one comes back in the second
// return value rather than being ignored. Silently skipping a marker with a
// typo in it is the failure mode where a generator produces nothing and says
// nothing, which is a bad afternoon.
//
// Generated files are scanned like any other. A generator that does not want
// to read its own output should skip files by name, since it is the one that
// knows what it writes.
func Scan(pkgs ...*Package) ([]Target, []Error) {
	var w walker
	for _, p := range pkgs {
		w.pkg = p
		for _, f := range p.Syntax {
			w.walk(f)
		}
	}
	return w.targets, w.errs
}

type walker struct {
	pkg     *Package
	file    *ast.File
	targets []Target
	errs    []Error
}

func (w *walker) walk(f *ast.File) {
	w.file = f
	// The package comment, which is where a file-level marker such as
	// //mizu:manual goes.
	w.add(f, nil, f.Doc)

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			w.add(d, w.def(d.Name), d.Doc)
		case *ast.GenDecl:
			w.genDecl(d)
		}
	}
}

func (w *walker) genDecl(d *ast.GenDecl) {
	for _, spec := range d.Specs {
		// A spec's own comment wins. Falling back to the declaration's comment
		// only when there is one spec is the same rule godoc uses, and it
		// keeps a comment above a parenthesized group from being copied onto
		// every name inside it.
		switch s := spec.(type) {
		case *ast.TypeSpec:
			w.add(s, w.def(s.Name), docFor(s.Doc, d))
			w.fields(s.Type)
		case *ast.ValueSpec:
			var obj types.Object
			if len(s.Names) > 0 {
				obj = w.def(s.Names[0])
			}
			w.add(s, obj, docFor(s.Doc, d))
		}
	}
}

// fields walks a type expression for struct fields and interface methods with
// comments on them. It goes all the way down, so a marker on a field of an
// anonymous struct nested inside another one is found.
func (w *walker) fields(expr ast.Expr) {
	ast.Inspect(expr, func(n ast.Node) bool {
		f, ok := n.(*ast.Field)
		if !ok || f.Doc == nil {
			return true
		}
		var obj types.Object
		if len(f.Names) > 0 {
			obj = w.def(f.Names[0])
		}
		w.add(f, obj, f.Doc)
		return true
	})
}

func docFor(own *ast.CommentGroup, d *ast.GenDecl) *ast.CommentGroup {
	if own != nil {
		return own
	}
	if len(d.Specs) == 1 {
		return d.Doc
	}
	return nil
}

// def looks a declared name up in the type information. It returns nil rather
// than failing when there is none, because a package with type errors still
// has markers in it and that is exactly when a generator is most useful.
func (w *walker) def(id *ast.Ident) types.Object {
	if id == nil || w.pkg.TypesInfo == nil {
		return nil
	}
	return w.pkg.TypesInfo.Defs[id]
}

func (w *walker) add(node ast.Node, obj types.Object, doc *ast.CommentGroup) {
	if doc == nil {
		return
	}
	var markers []Marker
	for _, c := range doc.List {
		pos := w.pkg.Fset.Position(c.Slash)
		m, err := parseMarker(c.Text, pos)
		switch {
		case err != nil:
			w.errs = append(w.errs, Error{Pos: pos.String(), Msg: err.Error(), Kind: MarkerError})
		case m.Name != "":
			markers = append(markers, m)
		}
	}
	if len(markers) > 0 {
		w.targets = append(w.targets, Target{Pkg: w.pkg, File: w.file, Node: node, Object: obj, Markers: markers})
	}
}

const markerPrefix = "//mizu:"

// parseMarker reads one comment. A comment that is not a marker comes back as
// the zero Marker and no error.
func parseMarker(text string, pos token.Position) (Marker, error) {
	if !strings.HasPrefix(text, markerPrefix) {
		return Marker{}, nearMiss(text)
	}
	rest := text[len(markerPrefix):]

	i := 0
	for i < len(rest) && isNameByte(rest[i]) {
		i++
	}
	name := rest[:i]
	if name == "" {
		return Marker{}, errors.New("marker has no name after the colon")
	}
	if i < len(rest) && rest[i] != ' ' && rest[i] != '\t' {
		return Marker{}, fmt.Errorf("marker name %q is followed by %q, which is not a name character or a space", name, rest[i:i+1])
	}

	args, err := parseArgs(rest[i:])
	if err != nil {
		return Marker{}, fmt.Errorf("mizu:%s: %w", name, err)
	}
	return Marker{Name: name, Args: args, Pos: pos, Text: text}, nil
}

// nearMiss catches the comment that was meant to be a marker and is not.
//
// The space in `// mizu:model` is the whole difference between a directive and
// a sentence, and nothing about looking at it says so. Go's own rule is that a
// directive is two slashes followed immediately by the name, which is why
// CommentGroup.Text strips one and keeps the other. Without this check the
// file compiles, the generator runs, and no code comes out.
//
// A sentence that starts with the word mizu and a colon would be flagged too,
// so the remainder has to read like arguments before this says anything:
// lowercase words, no sentence punctuation. That still leaves a sentence it
// can be fooled by. A wrong error message naming the exact line is a better
// afternoon than a marker that quietly does nothing, and rewording the
// sentence is the fix.
func nearMiss(text string) error {
	body, ok := strings.CutPrefix(text, "//")
	if !ok {
		return nil
	}
	trimmed := strings.TrimLeft(body, " \t")
	if len(trimmed) == len(body) || !strings.HasPrefix(trimmed, "mizu:") {
		return nil
	}
	fixed := "//" + trimmed
	m, err := parseMarker(fixed, token.Position{})
	if err != nil || !readsLikeArgs(m.Args) {
		return nil
	}
	return fmt.Errorf("%q has a space after the slashes, which makes it a comment rather than a marker; write %q", text, fixed)
}

// readsLikeArgs reports whether a marker's arguments look like arguments
// rather than like the rest of an English sentence.
func readsLikeArgs(args []Arg) bool {
	for _, a := range args {
		if a.Key != "" {
			// Nothing in a sentence looks like key=value.
			continue
		}
		if a.Value == "" || strings.ContainsAny(a.Value, ".,;:!?") {
			return false
		}
		if r, _ := utf8.DecodeRuneInString(a.Value); unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func isNameByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-'
}

func parseArgs(rest string) ([]Arg, error) {
	var args []Arg
	keys := map[string]bool{}
	for {
		rest = strings.TrimLeft(rest, " \t")
		if rest == "" {
			return args, nil
		}

		// A quoted word is always positional. There is nowhere for a key to go
		// in front of it.
		if rest[0] == '"' {
			word, more, err := parseValue(rest)
			if err != nil {
				return nil, err
			}
			args = append(args, Arg{Value: word})
			rest = more
			continue
		}

		i := strings.IndexAny(rest, " \t=")
		if i < 0 {
			return append(args, Arg{Value: rest}), nil
		}
		if rest[i] != '=' {
			args = append(args, Arg{Value: rest[:i]})
			rest = rest[i:]
			continue
		}

		key := rest[:i]
		if key == "" {
			return nil, errors.New("an argument starts with = and has no key")
		}
		if keys[key] {
			return nil, fmt.Errorf("%s is given twice", key)
		}
		keys[key] = true

		value, more, err := parseValue(rest[i+1:])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		args = append(args, Arg{Key: key, Value: value})
		rest = more
	}
}

// parseValue reads one value and returns what is left after it. A value is
// either Go-quoted, which is how it holds a space, or a bare run up to the
// next space.
func parseValue(s string) (value, rest string, err error) {
	if len(s) > 0 && s[0] == '"' {
		for i := 1; i < len(s); i++ {
			switch s[i] {
			case '\\':
				i++
			case '"':
				v, err := strconv.Unquote(s[:i+1])
				if err != nil {
					return "", "", fmt.Errorf("%s is not a valid quoted string: %w", s[:i+1], err)
				}
				return v, s[i+1:], nil
			}
		}
		return "", "", errors.New("a quoted value is missing its closing quote")
	}
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i], s[i:], nil
	}
	return s, "", nil
}
