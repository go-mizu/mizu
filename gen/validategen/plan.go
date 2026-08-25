package validategen

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"math"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/gen"
)

// validatePkg is the package the output calls into. It is written here rather
// than read off a type, because a package with nothing marked in it imports
// nothing and a package with one marked struct imports exactly this.
const validatePkg = "github.com/go-mizu/mizu/validate"

// dive is the rule that says the rules after it are about the elements rather
// than about the field.
const dive = "dive"

// The subjects a size rule can be about, which are what decides the sentence a
// failure is written with. They are the strings [validate.RuleError.Of] takes,
// spelled here because that package keeps them to itself.
const (
	subjString   = "string"
	subjNumeric  = "numeric"
	subjArray    = "array"
	subjDuration = "duration"
)

// maxDepth is how far a nested struct is followed. A request struct nests a
// couple of levels and anything past this is either a mistake or something that
// wants breaking up. Stopping also means the walk is bounded whatever go/types
// hands over, which matters because a package that failed to type check is
// still walked.
const maxDepth = 12

// A Plan is every struct in one package that asked for a validator, and the
// functions that check them.
type Plan struct {
	Pkg     string    // the package name, for the package clause
	Dir     string    // the package directory, relative to the module
	Source  string    // the file the first marked struct was declared in
	Val     string    // what the output calls the validate package
	Ctx     string    // what the output calls the context package
	Structs []Struct  // in declaration order
	Helpers []*Helper // in the order they were worked out, so a nested one is first
	Errors  []error   // everything wrong, in the order found

	imps  *imports
	fset  *token.FileSet
	taken map[string]bool // names already spoken for at package scope
	local map[string]bool // names already spoken for in the function being written
	owner string          // the struct being walked, for a message about a tag
	root  string          // the struct the walk started at, for a message about nesting
	seen  []types.Type    // the struct types on the way here, so a tree stops
	elems []elem          // the function that checks each struct type
	depth int
}

// A Struct is one type that carries the marker.
type Struct struct {
	Type string // the Go type, as in CreatePost
	Call string // the function its method hands the work to
}

// A Helper is the function that checks one struct type.
//
// There is one per struct rather than a method body per marked type, because a
// type holding a list of itself has to have something to call, and because a
// type reached from two marked structs is then written down once.
type Helper struct {
	Name string // validateOrderLine
	Type string // OrderLine, as the output spells it
	Doc  string // the same type, as a sentence names it
	Body string // the statements, unindented, for gofmt to lay out

	busy bool // the body is still being worked out, which a recursion sees
}

// An elem is the function that checks one struct type, found by the type.
type elem struct {
	typ types.Type
	h   *Helper
}

// Analyze works out what every marked struct in a package needs, without
// writing anything. It returns a plan even when there are errors in it, so a
// caller can report all of them at once.
func Analyze(pkg *gen.Package, targets []gen.Target) *Plan {
	p := &Plan{
		Pkg:   pkg.Name,
		Dir:   dirOf(pkg),
		imps:  newImports(pkg.PkgPath),
		fset:  pkg.Fset,
		taken: map[string]bool{},
	}
	p.Val = p.imps.name(validatePkg)
	p.Ctx = p.imps.name("context")

	// A function this writes must not be called what something in the package
	// is already called, and the package is the only scope generated code
	// shares with code somebody wrote.
	//
	// What is in the file this is about to replace does not count. That name
	// got there the last time this ran, and treating it as taken would mean
	// every run renamed everything the run before it wrote.
	if pkg.Types != nil {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if !p.ours(scope.Lookup(name).Pos()) {
				p.taken[name] = true
			}
		}
	}

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

// ours is whether something was declared in the file this generator writes,
// which is to say by the last run of it rather than by somebody.
func (p *Plan) ours(pos token.Pos) bool {
	return filepath.Base(p.fset.Position(pos).Filename) == outputFile
}

func (p *Plan) errf(pos token.Position, format string, args ...any) {
	p.Errors = append(p.Errors, fmt.Errorf("%s: %s", pos, fmt.Sprintf(format, args...)))
}

// bad records a tag this cannot write, and why. The sentence is the one the tag
// interpreter would have printed at run time, since it is the same mistake and
// the same reader.
func (p *Plan) bad(pos token.Position, f *types.Var, why string) {
	p.errf(pos, "cannot check %s.%s, because %s", p.owner, f.Name(), why)
}

// structure reads one marked declaration and adds what it found to the plan.
func (p *Plan) structure(t gen.Target) {
	pos := t.Pos()
	if t.Object == nil {
		p.errf(pos, "the validate marker is on something with no name")
		return
	}
	tn, ok := t.Object.(*types.TypeName)
	if !ok {
		p.errf(pos, "%s is marked to validate and is not a type", t.Name())
		return
	}
	if _, ok := tn.Type().Underlying().(*types.Struct); !ok {
		p.errf(pos, "%s is marked to validate and is not a struct", t.Name())
		return
	}
	// A type that checks itself already is a type with nothing to ask for. The
	// marker is the mistake, and saying so here is better than the compiler
	// saying the method is declared twice about a file nobody wrote.
	if n, ok := tn.Type().(*types.Named); ok {
		for i := range n.NumMethods() {
			m := n.Method(i)
			if m.Name() == "Validate" && !p.ours(m.Pos()) {
				p.errf(pos, "%s is marked to validate and already has a Validate method of its own", t.Name())
				return
			}
		}
	}

	before := len(p.Errors)
	h := p.helperFor(tn.Type())
	if len(p.Errors) > before {
		return
	}
	// Nothing to check is reported here rather than when the file is written,
	// because here is where the struct is and a message about a declaration
	// says which one.
	if h == nil || h.Body == "" {
		p.errf(pos, "%s is marked to validate and has no validate tags under it", t.Name())
		return
	}
	p.Structs = append(p.Structs, Struct{Type: t.Name(), Call: h.Name})
}

// helperFor is the function that checks one struct type, written the first time
// something needs it and handed back after that.
//
// The walk starts from an empty path rather than the one that got here, which
// is what the tag interpreter does, so the function for a type is the same
// function wherever it was reached from.
func (p *Plan) helperFor(t types.Type) *Helper {
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	for _, e := range p.elems {
		if types.Identical(e.typ, t) {
			return e.h
		}
	}

	h := &Helper{
		Name: p.helperName(t),
		Type: p.imps.typeString(t),
		Doc:  p.imps.docString(t),
		busy: true,
	}
	p.elems = append(p.elems, elem{typ: t, h: h})

	seen, local, depth, owner, root := p.seen, p.local, p.depth, p.owner, p.root
	p.seen, p.local, p.depth, p.owner, p.root = []types.Type{t}, newLocals(), 0, h.Doc, h.Doc

	var body block
	p.walk(&body, st, "", "v")
	h.Body = strings.TrimLeft(body.String(), "\n")
	h.busy = false

	p.seen, p.local, p.depth, p.owner, p.root = seen, local, depth, owner, root

	if h.Body != "" {
		p.Helpers = append(p.Helpers, h)
	}
	return h
}

// live is whether a call to this function is worth writing. A function still
// being worked out is, because the recursion that reached it is in its body and
// that is enough to make the body worth having.
func (h *Helper) live() bool { return h != nil && (h.busy || h.Body != "") }

// walk writes the checks for every tagged field of st, following embedded and
// nested structs down.
//
// prefix is what a nested struct's fields are named under, so a City inside an
// Address is address.city. expr is what the output writes to reach this struct.
func (p *Plan) walk(out *block, st *types.Struct, prefix, expr string) {
	p.depth++
	defer func() { p.depth-- }()

	for i := range st.NumFields() {
		f := st.Field(i)
		if !f.Exported() && !f.Embedded() {
			// Generated code cannot reach it, and the tag interpreter does not
			// look at one either.
			continue
		}

		tag := reflect.StructTag(st.Tag(i))
		rules := tag.Get("validate")
		if rules == "-" {
			continue
		}
		name, ok := nameOf(tag, f.Name())
		if !ok {
			continue
		}

		fexpr := expr + "." + f.Name()
		if rules != "" {
			p.field(out, f, fexpr, prefix+name, rules)
		}

		// A struct is worth going into whether or not it had a tag of its own,
		// because the rules are on its fields. One with no tags anywhere under
		// it writes nothing, which is what makes a time.Time in the middle of a
		// request struct cost a walk here and nothing after that.
		bt := base(f.Type())
		sub, ok := bt.Underlying().(*types.Struct)
		if !ok || p.sawType(bt) {
			continue
		}
		// A struct inside a struct inside a struct, twelve times over, is
		// either a mistake or something that wants breaking up, and stopping
		// here means the walk is bounded whatever go/types hands over.
		if p.depth > maxDepth {
			p.errf(p.fset.Position(f.Pos()),
				"%s nests more than %d deep, which is further down than a request goes",
				p.root, maxDepth)
			continue
		}

		under := prefix
		if !f.Embedded() {
			under = prefix + name + "."
		}

		// A pointer on the way down is a struct that may not be there. What it
		// had to be is the pointer's own required to say, and the fields under
		// it are not there to be wrong.
		depth := ptrDepth(f.Type())
		locals := make([]string, depth)
		for k := range depth {
			locals[k] = p.newLocal(f.Name())
		}
		into := fexpr
		if depth > 0 {
			into = locals[depth-1]
		}

		var inner block
		owner := p.owner
		p.owner, p.seen = p.imps.docString(bt), append(p.seen, bt)
		p.walk(&inner, sub, under, into)
		p.owner, p.seen = owner, p.seen[:len(p.seen)-1]

		if inner.empty() {
			continue
		}
		out.line("")
		for k := range depth {
			from := fexpr
			if k > 0 {
				from = "*" + locals[k-1]
			}
			out.line("if %s := %s; %s != nil {", locals[k], from, locals[k])
		}
		out.raw(strings.TrimLeft(inner.String(), "\n"))
		for range depth {
			out.line("}")
		}
	}
}

// field writes the checks for one tagged field.
func (p *Plan) field(out *block, f *types.Var, expr, name, tag string) {
	pos := p.fset.Position(f.Pos())

	rules, err := parseTag(tag)
	if err != nil {
		p.bad(pos, f, err.Error())
		return
	}

	// dive splits the list. What is in front of it is about the field, and what
	// is behind it is about each of the field's elements.
	before, after := rules, []tagRule(nil)
	dived := false
	for i, r := range rules {
		if r.name == dive {
			before, after, dived = rules[:i], rules[i+1:], true
			break
		}
	}

	var pre block
	val := p.value(&pre, expr, f.Type(), f.Name())

	steps, ok := p.steps(pos, f, before, f.Type(), val)
	if !ok {
		return
	}

	var tail block
	if dived {
		p.elements(&tail, pos, f, after, val, name)
	}

	var body block
	writeChain(&body, nameExpr(name), steps, tail.String())
	if body.empty() {
		return
	}
	out.line("")
	out.raw(pre.String())
	out.raw(body.String())
}

// elements writes the loop over a field that dived.
//
// The name of an element is the field's name and the position, tags.1, and a
// struct element's own fields are named under that, lines.1.sku, which is what
// the function it hands off to does with the prefix it is given.
func (p *Plan) elements(out *block, pos token.Position, f *types.Var, after []tagRule, val, name string) {
	et, ok := elemOf(f.Type())
	if !ok {
		kind := p.kindOf(f.Type())
		p.bad(pos, f, "dive is for a slice or an array and this field is "+article(kind)+" "+kind)
		return
	}
	h := p.helperFor(base(et))

	depth := ptrDepth(et)
	item := "e"

	var pre block
	if depth > 0 {
		pre.line("var el %s", p.imps.typeString(base(et)))
		pre.line("if %s {", nonNil("e", depth))
		pre.line("el = %s", strings.Repeat("*", depth)+"e")
		pre.line("}")
		item = "el"
	}

	steps, ok := p.steps(pos, f, after, et, item)
	if !ok {
		return
	}

	var tail block
	if h.live() {
		if depth > 0 {
			tail.line("if %s {", nonNil("e", depth))
			tail.line("%s(bad, name+\".\", %s)", h.Name, strings.Repeat("*", depth)+"e")
			tail.line("}")
		} else {
			tail.line("%s(bad, name+\".\", e)", h.Name)
		}
	}

	var body block
	writeChain(&body, "name", steps, tail.String())
	if body.empty() {
		return
	}

	out.line("for i, e := range %s {", val)
	out.line("name := %s + %s.Itoa(i)", nameExpr(name+"."), p.imps.name("strconv"))
	if len(steps) > 0 {
		out.raw(pre.String())
	}
	out.raw(body.String())
	out.line("}")
}

// value is the expression the rules read.
//
// It is the field itself, unless the field is behind a pointer, in which case
// it is a local holding what the pointer points at and holding that type's zero
// when there is nothing there. That is what the tag interpreter checks a nil
// pointer as, so a rule sees a string whether the field was one or pointed at
// one.
func (p *Plan) value(pre *block, expr string, ft types.Type, field string) string {
	depth := ptrDepth(ft)
	if depth == 0 {
		return expr
	}
	l := p.newLocal(field)
	pre.line("var %s %s", l, p.imps.typeString(base(ft)))
	pre.line("if %s {", nonNil(expr, depth))
	pre.line("%s = %s", l, strings.Repeat("*", depth)+expr)
	pre.line("}")
	return l
}

// A step is one rule, resolved to the condition under which it fails.
type step struct {
	skip bool   // omitempty, which stops the chain rather than recording anything
	init string // a count to share with the rules after it, as an if statement's init
	cond string // when this is true the rule failed
	fail string // the validate.Failed expression that says so
	stop string // for omitempty, that the value is there and the chain goes on
}

// steps resolves a list of rules as a tag spelled them, for a value of type ft.
func (p *Plan) steps(pos token.Position, f *types.Var, rules []tagRule, ft types.Type, val string) ([]step, bool) {
	if len(rules) == 0 {
		return nil, true
	}
	// An interface holds something the tag interpreter can look at when a value
	// arrives and this cannot, since this runs before there is one.
	if _, ok := base(ft).Underlying().(*types.Interface); ok {
		p.bad(pos, f, "generated code cannot see what an interface holds, and validate.Struct is the mode that can")
		return nil, false
	}

	out := make([]step, 0, len(rules))
	for _, r := range rules {
		s, err := p.resolve(r, ft, val)
		if err != nil {
			p.bad(pos, f, err.Error())
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

// formats are the format checks, and the function in the validate package that
// each one is.
//
// TestEveryFormatCheckIsHere reads the validate package and fails on one that
// is not in here, so a check that lands is a check a generated validator can
// run rather than one that quietly stops generating.
var formats = map[string]string{
	"cidr":     "IsCIDR",
	"e164":     "IsE164",
	"email":    "IsEmail",
	"hostname": "IsHostname",
	"ip":       "IsIP",
	"ipv4":     "IsIPv4",
	"ipv6":     "IsIPv6",
	"mac":      "IsMAC",
	"port":     "IsPort",
	"ulid":     "IsULID",
	"uri":      "IsURI",
	"url":      "IsURL",
	"uuid":     "IsUUID",
}

// sizeRules are the rules that take one bound, and the comparison that means
// the rule failed.
var sizeRules = map[string]string{"min": "<", "max": ">", "size": "!="}

// resolve is what one rule from a tag becomes, for a value of type ft read by
// the expression val.
//
// Everything the tag interpreter settles when it builds a plan is settled here,
// in the same order and with the same sentences, so a struct either generates
// or is refused for the reason it would have been refused at run time.
func (p *Plan) resolve(r tagRule, ft types.Type, val string) (step, error) {
	if fn, ok := formats[r.name]; ok {
		if !textual(ft) {
			kind := p.kindOf(ft)
			return step{}, fmt.Errorf("%s is a check on a string and this field is %s %s", r.name, article(kind), kind)
		}
		if len(r.params) != 0 {
			return step{}, fmt.Errorf("%s takes no parameters", r.name)
		}
		cond := fmt.Sprintf("!%s.%s(%s)", p.Val, fn, p.str(val, ft))
		return step{cond: cond, fail: p.failed(r.name)}, nil
	}

	switch r.name {
	case "required", "omitempty":
		if len(r.params) != 0 {
			return step{}, fmt.Errorf("%s takes no parameters", r.name)
		}
		empty, there, err := p.emptiness(val, ft)
		if err != nil {
			return step{}, err
		}
		if r.name == "omitempty" {
			return step{skip: true, stop: there}, nil
		}
		return step{cond: empty, fail: p.failed("required")}, nil

	case "between":
		if len(r.params) != 2 {
			return step{}, errors.New("between takes two bounds, as in between=3 10")
		}
		lo, hi, subject, left, init, err := p.span(r.name, r.params[0], r.params[1], ft, val)
		if err != nil {
			return step{}, err
		}
		cond := fmt.Sprintf("%s < %s || %s > %s", left, lo, left, hi)
		return step{init: init, cond: cond, fail: p.failed("between", lo, hi) + p.of(subject)}, nil
	}

	if op, ok := sizeRules[r.name]; ok {
		if len(r.params) != 1 {
			return step{}, fmt.Errorf("%s takes one bound, as in %s=3", r.name, r.name)
		}
		n, _, subject, left, init, err := p.span(r.name, r.params[0], r.params[0], ft, val)
		if err != nil {
			return step{}, err
		}
		cond := fmt.Sprintf("%s %s %s", left, op, n)
		return step{init: init, cond: cond, fail: p.failed(r.name, n) + p.of(subject)}, nil
	}

	if r.name == dive {
		return step{}, errors.New("a second dive, and one list of elements is all this reads")
	}
	return step{}, fmt.Errorf("there is no rule called %s, and one added with validate.Register is not a rule this can write, since it is looked up while the program runs", r.name)
}

// span works out both sides of a size comparison: the bounds as the output
// spells them, what the value is counted as, the expression to compare, and the
// count to hoist when the expression is one worth writing once.
func (p *Plan) span(rule, from, to string, ft types.Type, val string) (lo, hi, subject, left, init string, err error) {
	subject, count, hoist, err := p.count(rule, val, ft)
	if err != nil {
		return "", "", "", "", "", err
	}
	a, err := p.bound(from, ft)
	if err != nil {
		return "", "", "", "", "", err
	}
	b := a
	if to != from {
		if b, err = p.bound(to, ft); err != nil {
			return "", "", "", "", "", err
		}
	}

	left = count
	if hoist {
		left, init = "n", "n := "+count
	}
	// The comparison runs in float64 whenever the natural one would not give
	// the same answer, which is what the tag interpreter compares in and what
	// keeps a bound that does not fit the field's own type from failing to
	// compile.
	if !exact(ft, a) || !exact(ft, b) {
		left = "float64(" + left + ")"
	}
	return a.lit, b.lit, subject, left, init, nil
}

// count says what a size rule counts for a value of this type, how to count it,
// and whether the counting is worth doing once for a run of rules.
func (p *Plan) count(rule, val string, ft types.Type) (subject, expr string, hoist bool, err error) {
	bt := base(ft)
	if isNamed(bt, "time", "Duration") {
		return subjDuration, val, false, nil
	}
	switch u := bt.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Info()&types.IsString != 0:
			// Runes rather than bytes, so an emoji is one character the way
			// somebody typing it would say.
			return subjString, p.imps.name("unicode/utf8") + ".RuneCountInString(" + p.str(val, ft) + ")", true, nil
		case u.Info()&(types.IsInteger|types.IsFloat) != 0:
			return subjNumeric, val, false, nil
		}
	case *types.Slice, *types.Map, *types.Array:
		return subjArray, "len(" + val + ")", true, nil
	}
	kind := p.kindOf(ft)
	return "", "", false, fmt.Errorf("%s counts something and %s %s has nothing to count", rule, article(kind), kind)
}

// emptiness is how the output asks whether a value was filled in, and how it
// asks the opposite.
//
// The rule is the zero value for the type, with an empty list counting as
// missing even though a non-nil empty slice is not the zero value. That is what
// isEmpty in the validate package does, written out per type.
func (p *Plan) emptiness(val string, ft types.Type) (empty, there string, err error) {
	bt := base(ft)
	switch u := bt.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Info()&types.IsString != 0:
			return val + ` == ""`, val + ` != ""`, nil
		case u.Kind() == types.Bool:
			return "!" + val, val, nil
		case u.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			return val + " == 0", val + " != 0", nil
		}
	case *types.Slice, *types.Map, *types.Array:
		return "len(" + val + ") == 0", "len(" + val + ") != 0", nil
	case *types.Chan, *types.Signature:
		return val + " == nil", val + " != nil", nil
	case *types.Struct:
		if types.Comparable(bt) {
			zero := "(" + p.imps.typeString(bt) + "{})"
			return val + " == " + zero, val + " != " + zero, nil
		}
		// A struct with a slice in it cannot be compared, and asking reflect
		// whether it is the zero value is what the tag interpreter does anyway.
		zero := p.imps.name("reflect") + ".ValueOf(" + val + ").IsZero()"
		return zero, "!" + zero, nil
	}
	kind := p.kindOf(ft)
	return "", "", fmt.Errorf("required is about a value being filled in and %s %s has no way to say", article(kind), kind)
}

// A bnd is one bound from a tag, in the form the comparison needs and in the
// form the sentence prints.
type bnd struct {
	kind int // bInt, bFloat or bDur
	i    int
	lit  string
}

const (
	bInt = iota
	bFloat
	bDur
)

// bound turns a tag's bound into what the output compares against.
//
// The field's type decides what the text means, so min=1h on a time.Duration is
// an hour and min=3 on anything else is the number three. This is the same
// reading the tag interpreter does, and the literal it writes has the same type
// the interpreter's value has, so the sentence comes out the same.
func (p *Plan) bound(s string, ft types.Type) (bnd, error) {
	if isNamed(base(ft), "time", "Duration") {
		d, err := time.ParseDuration(s)
		if err != nil {
			return bnd{}, fmt.Errorf("%s is not a length of time, as in 1h30m", s)
		}
		return bnd{kind: bDur, lit: p.durLit(d)}, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return bnd{kind: bInt, i: n, lit: strconv.Itoa(n)}, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return bnd{kind: bFloat, lit: p.floatLit(f)}, nil
	}
	return bnd{}, fmt.Errorf("%s is not a number", s)
}

// exact is whether comparing a value of this type against this bound as it is
// written gives the same answer as comparing the two as float64, which is what
// the tag interpreter does.
//
// A bound that does not fit the field's type is the other thing this catches: a
// negative one on an unsigned field, or 300 on an int8, will not compile as
// written and has to go through the conversion.
func exact(ft types.Type, b bnd) bool {
	if b.kind == bDur {
		return true
	}
	k := basicKind(base(ft))
	if b.kind == bFloat {
		return k == types.Float64
	}
	// The comparisons below are in int64 rather than in int, because the widest
	// of them is wider than an int on a 32-bit machine and the generator has to
	// build there whatever it is generating for.
	n := int64(b.i)
	switch k {
	case types.Int8:
		return n >= math.MinInt8 && n <= math.MaxInt8
	case types.Int16:
		return n >= math.MinInt16 && n <= math.MaxInt16
	case types.Int, types.Int32:
		return n >= math.MinInt32 && n <= math.MaxInt32
	case types.Int64:
		return n >= -(1<<53) && n <= 1<<53
	case types.Uint8:
		return n >= 0 && n <= math.MaxUint8
	case types.Uint16:
		return n >= 0 && n <= math.MaxUint16
	case types.Uint, types.Uint32:
		return n >= 0 && n <= math.MaxUint32
	case types.Uint64:
		return n >= 0 && n <= 1<<53
	case types.Float64:
		return true
	case types.String:
		// A count of runes, which is an int and holds any bound that fits one.
		return true
	}
	// A float32 counts as itself and compares as a float64 in the interpreter,
	// and a list counts its elements, which is a string's case again.
	if k == types.Float32 {
		return false
	}
	return true
}

// durLit writes a duration the way somebody would, which is the largest unit
// that divides it evenly.
func (p *Plan) durLit(d time.Duration) string {
	t := p.imps.name("time")
	for _, u := range []struct {
		size time.Duration
		name string
	}{
		{time.Hour, "Hour"},
		{time.Minute, "Minute"},
		{time.Second, "Second"},
		{time.Millisecond, "Millisecond"},
		{time.Microsecond, "Microsecond"},
	} {
		if d == 0 || d%u.size != 0 {
			continue
		}
		switch n := int64(d / u.size); n {
		case 1:
			return t + "." + u.name
		case -1:
			return "-" + t + "." + u.name
		default:
			return strconv.FormatInt(n, 10) + " * " + t + "." + u.name
		}
	}
	if d == 0 {
		return t + ".Duration(0)"
	}
	return strconv.FormatInt(int64(d), 10) + " * " + t + ".Nanosecond"
}

// floatLit writes a bound that is not a whole number, and makes sure it is a
// float when it reads like an integer, since the sentence prints the type.
func (p *Plan) floatLit(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return p.imps.name("math") + ".Inf(1)"
	case math.IsInf(f, -1):
		return p.imps.name("math") + ".Inf(-1)"
	case math.IsNaN(f):
		return p.imps.name("math") + ".NaN()"
	}
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		return "float64(" + s + ")"
	}
	return s
}

// failed is the RuleError a failure records.
func (p *Plan) failed(rule string, params ...string) string {
	args := strconv.Quote(rule)
	for _, x := range params {
		args += ", " + x
	}
	return p.Val + ".Failed(" + args + ")"
}

// of is what a size rule counted, which is half of the key the sentence is
// looked up under.
func (p *Plan) of(subject string) string { return ".Of(" + strconv.Quote(subject) + ")" }

// str is a value on its way into a check that reads a string, converted when
// the field is a type of its own and written as it is when it is not.
func (p *Plan) str(val string, ft types.Type) string {
	if types.Identical(base(ft), types.Typ[types.String]) {
		return val
	}
	return "string(" + val + ")"
}

// nameExpr is what the output writes for the name a failure comes back under,
// which is the prefix the function was called with and the rest of it.
func nameExpr(name string) string { return "at + " + strconv.Quote(name) }

// nonNil is the condition that a pointer this deep can be read through.
func nonNil(expr string, depth int) string {
	parts := make([]string, depth)
	for k := range depth {
		parts[k] = strings.Repeat("*", k) + expr + " != nil"
	}
	return strings.Join(parts, " && ")
}

// textual is whether a value of this type can be read as a string.
func textual(t types.Type) bool {
	b, ok := base(t).Underlying().(*types.Basic)
	return ok && b.Info()&types.IsString != 0
}

// base is the type with the pointers taken off it, since a rule reads through
// them and a nil one is the same as a missing value.
func base(t types.Type) types.Type {
	for {
		ptr, ok := t.Underlying().(*types.Pointer)
		if !ok {
			return t
		}
		t = ptr.Elem()
	}
}

// ptrDepth is how many pointers there are to read through.
func ptrDepth(t types.Type) int {
	n := 0
	for {
		ptr, ok := t.Underlying().(*types.Pointer)
		if !ok {
			return n
		}
		t, n = ptr.Elem(), n+1
	}
}

// elemOf is what a list holds, and whether the type is a list at all.
//
// A map is not, on purpose. The failures come out in the order they were found
// and a map has no order, so a map dived into would report the same problems in
// a different order on every run.
func elemOf(t types.Type) (types.Type, bool) {
	switch u := base(t).Underlying().(type) {
	case *types.Slice:
		return u.Elem(), true
	case *types.Array:
		return u.Elem(), true
	}
	return nil, false
}

func basicKind(t types.Type) types.BasicKind {
	if b, ok := t.Underlying().(*types.Basic); ok {
		return b.Kind()
	}
	return types.Invalid
}

// kindOf is what to call a type in a sentence about a tag, which is its own
// name when it has one and what shape it is when it does not.
func (p *Plan) kindOf(t types.Type) string {
	t = base(t)
	if _, ok := t.(*types.Named); ok {
		return p.imps.docString(t)
	}
	if b, ok := t.(*types.Basic); ok {
		if b.Kind() == types.UnsafePointer {
			return "unsafe.Pointer"
		}
		return b.Name()
	}
	switch t.Underlying().(type) {
	case *types.Slice:
		return "slice"
	case *types.Array:
		return "array"
	case *types.Map:
		return "map"
	case *types.Struct:
		return "struct"
	case *types.Interface:
		return "interface"
	case *types.Chan:
		return "chan"
	case *types.Signature:
		return "func"
	}
	return p.imps.docString(t)
}

func (p *Plan) sawType(t types.Type) bool {
	for _, s := range p.seen {
		if types.Identical(s, t) {
			return true
		}
	}
	return false
}

// helperName is what the function that checks a type is called, which is the
// type's name with validate in front of it, and a number after it when the
// package already has something of that name.
func (p *Plan) helperName(t types.Type) string {
	want := "validateStruct"
	if _, ok := t.(*types.Named); ok {
		want = "validate" + ident(p.imps.docString(t))
	}
	name := want
	for k := 2; p.taken[name]; k++ {
		name = want + strconv.Itoa(k)
	}
	p.taken[name] = true
	return name
}

// newLocal is a local variable named after a field, with a number on it when
// something in the function is already called that.
func (p *Plan) newLocal(field string) string {
	want := lower(field)
	name := want
	for k := 2; p.local[name]; k++ {
		name = want + strconv.Itoa(k)
	}
	p.local[name] = true
	return name
}

// newLocals is the names a function starts with spoken for: the ones the output
// writes itself, the keywords, and the predeclared identifiers the output uses,
// since shadowing len is a way to stop a later line compiling.
func newLocals() map[string]bool {
	m := map[string]bool{}
	for _, n := range []string{
		"at", "bad", "ctx", "v", "e", "el", "i", "n", "name",
		"break", "case", "chan", "const", "continue", "default", "defer", "else",
		"fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
		"map", "package", "range", "return", "select", "struct", "switch", "type", "var",
		"append", "bool", "byte", "cap", "copy", "error", "false", "float32", "float64",
		"int", "len", "make", "new", "nil", "rune", "string", "true",
	} {
		m[n] = true
	}
	return m
}

// nameOf is the name the request uses for a field.
//
// It is the order web.Bind names a field in: a path, header, cookie, form or
// query tag says so, then the json tag, and then the field's own name in snake
// case. The two have to agree, because a request that failed to bind and a
// request that failed to validate come back in one document, and a field named
// two ways in it is a field a form cannot mark.
func nameOf(tag reflect.StructTag, field string) (string, bool) {
	for _, t := range [...]string{"path", "header", "cookie", "form", "query"} {
		v, has := tag.Lookup(t)
		if !has {
			continue
		}
		switch name := tagName(v); name {
		case "-":
			return "", false
		case "":
			return snake(field), true
		default:
			return name, true
		}
	}

	if v, has := tag.Lookup("json"); has {
		switch name := tagName(v); name {
		case "-":
			return "", false
		case "":
		default:
			return name, true
		}
	}
	return snake(field), true
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
// It is the same function web.Bind and validate.Struct use to name an untagged
// field, copied rather than shared because those are unexported and the three
// have to give the same answer for one field to have one name.
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

// lower turns a field's name into the name of a local, which is the leading run
// of capitals in lower case, so ID is id and IPAddr is ipAddr.
func lower(s string) string {
	if s == "" {
		return "x"
	}
	n := 0
	for n < len(s) && isUpper(s[n]) {
		n++
	}
	switch {
	case n == 0:
		return s
	case n == len(s):
		return strings.ToLower(s)
	case n == 1:
		return strings.ToLower(s[:1]) + s[1:]
	default:
		// The last capital of the run starts the next word, so IPAddr is one
		// word and two rather than three.
		return strings.ToLower(s[:n-1]) + s[n-1:]
	}
}

// ident turns a type as it is written into something that can be part of a
// name, so other.Line is OtherLine.
func ident(s string) string {
	var b strings.Builder
	up := true
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			if up {
				b.WriteString(strings.ToUpper(string(r)))
				up = false
			} else {
				b.WriteRune(r)
			}
		case r >= '0' && r <= '9' && b.Len() > 0:
			b.WriteRune(r)
		default:
			up = true
		}
	}
	if b.Len() == 0 {
		return "Struct"
	}
	return b.String()
}
