package validate

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// Struct checks a value against the validate tags on its fields.
//
//	type CreatePost struct {
//		Title string   `json:"title" validate:"required,min=3,max=200"`
//		Body  string   `json:"body" validate:"required"`
//		Site  string   `json:"site" validate:"omitempty,url"`
//		Tags  []string `json:"tags" validate:"max=5,dive,slug"`
//	}
//
//	if err := validate.Struct(ctx, in); err != nil {
//		return err
//	}
//
// It is the mode that runs when the generator has not. The generator turns the
// same tags into a [Validator] method, which is the same rules in the same
// order without the reflection, so a program moves between the two by running
// mizu gen:validate and nothing it wrote changes.
//
// This does not look for a [Validator] method, and calling it on a type that
// has a generated one runs the tags a second time rather than the method. The
// caller chooses which mode it is in, and web.Bind is the caller that chooses
// per struct.
//
// The value may be a struct or a pointer to one. Anything else is a mistake in
// the program and comes back as an error of kind [errs.Internal], as does a tag
// this cannot make sense of. What a request got wrong comes back the way every
// other check reports it, from [Errors.OrNil].
//
// Sentences are written in English. The seam for a translation is
// [Errors.Msgs], which the tag interpreter has no way to reach yet: it arrives
// with mizu/i18n, which is where a request's locale is known.
func Struct(ctx context.Context, value any) error {
	rv := deref(reflect.ValueOf(value))
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return errs.Newf(errs.Internal, "validate.target",
			"validate: cannot check a %T, which is not a struct", value)
	}

	p := planFor(rv.Type())
	if p.err != nil {
		return p.err
	}

	v := New()
	if err := p.run(ctx, v, rv, ""); err != nil {
		return err
	}
	return v.Err()
}

// dive is the rule that says the rules after it are about the elements rather
// than about the field.
const dive = "dive"

// plainRules are the rules a tag writes on their own, with no parameters.
//
// The format checks are in here by name, taken from the table in format.go, so
// a format that lands is a format a tag can ask for without a second list to
// keep in step with the first.
var plainRules = func() map[string]func(*Check) *Check {
	m := map[string]func(*Check) *Check{
		"required":  (*Check).Required,
		"omitempty": (*Check).Optional,
	}
	for name, ok := range formats {
		m[name] = func(c *Check) *Check { return c.format(name, ok) }
	}
	return m
}()

// sizeRules are the rules that take one bound.
var sizeRules = map[string]func(*Check, any) *Check{
	"min":  (*Check).Min,
	"max":  (*Check).Max,
	"size": (*Check).Size,
}

// spanRules are the rules that take two bounds.
var spanRules = map[string]func(*Check, any, any) *Check{
	"between": (*Check).Between,
}

// A structPlan is how one struct type is checked, worked out once and kept.
//
// Building it is the reflection and the tag parsing, and it happens the first
// time the type is checked. Everything after that walks a flat slice and calls
// a closure per rule, which is why a nested struct is flattened here rather
// than at run time.
type structPlan struct {
	fields []fieldPlan
	err    error // a tag this cannot run, reported to every caller
}

// A fieldPlan is one field and the rules on it.
type fieldPlan struct {
	index []int  // the path to the field, through embedded and nested structs
	name  string // the name the request uses, which is not the field's name
	rules []step // what runs on the value itself
	dive  bool   // whether the rules go on into the field's elements
	each  []step // what runs on each element
	elem  *structPlan

	// zero is what a nil pointer field is checked as, which is the zero value
	// of the type it points at. A rule reads through a pointer anyway, so this
	// only decides what a rule sees when there is nothing to read through to,
	// and the answer is the same thing an absent value of that type would be.
	// It is also what the generated validator has in its hands, since a field
	// it cannot follow is a local holding that type's zero.
	zero reflect.Value

	// elemZero is the same thing for one element of a field that dived.
	elemZero reflect.Value
}

// A step is one rule, resolved to the thing that runs it.
//
// A rule this package ships with is a closure over the builder, so the tag
// interpreter and a chain somebody wrote out by hand are running the same code
// and cannot answer differently. A registered rule is held as itself, since it
// wants the context and the tag's parameters.
type step struct {
	run    func(*Check) *Check
	rule   Rule
	params []string
}

// plans is the plan for each type that has been checked.
var plans sync.Map // reflect.Type -> *structPlan

// A build is what one call to [planFor] carries around while it works.
//
// seen is the struct types on the path here, so a nested struct that holds one
// of itself stops rather than flattening for ever. elems is the plan for each
// struct type a dive has reached, which is what lets a list of a type hold
// itself: the plan for the element is the plan being built, and the recursion
// happens over the values at run time instead of over the types here.
type build struct {
	seen  map[reflect.Type]bool
	elems map[reflect.Type]*structPlan
	err   error
}

// planFor is the plan for t, built the first time t is checked.
//
// Two goroutines checking the same type at once may both build one, and one of
// the two is thrown away. That is cheaper than a lock held across the
// reflection, and the plans are equal anyway.
func planFor(t reflect.Type) *structPlan {
	if p, ok := plans.Load(t); ok {
		return p.(*structPlan)
	}

	p := &structPlan{}
	b := &build{
		seen:  map[reflect.Type]bool{t: true},
		elems: map[reflect.Type]*structPlan{t: p},
	}
	p.walk(b, t, "", nil)
	p.err = b.err

	actual, _ := plans.LoadOrStore(t, p)
	return actual.(*structPlan)
}

// walk adds every tagged field of t to the plan, following embedded and nested
// structs down.
//
// prefix is what a nested struct's fields are named under, so a City inside an
// Address is address.city. An embedded struct has no prefix, which is what
// embedding means everywhere else in Go and in encoding/json.
func (p *structPlan) walk(b *build, t reflect.Type, prefix string, index []int) {
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		tag := sf.Tag.Get("validate")
		if tag == "-" {
			continue
		}

		name, ok := nameOf(sf)
		if !ok {
			continue
		}

		// The index is copied rather than appended to in place, since every
		// field below this one would otherwise share the array and overwrite
		// each other's last element.
		at := append(index[:len(index):len(index)], i)

		if tag != "" {
			p.field(b, t, sf, at, prefix+name, tag)
		}

		// A struct is worth going into whether or not it had a tag of its own,
		// because the rules are on its fields. One with no tags anywhere under
		// it adds nothing, which is what makes a time.Time in the middle of a
		// request struct cost a walk at plan time and nothing after that.
		st := base(sf.Type)
		if st.Kind() != reflect.Struct || b.seen[st] {
			continue
		}

		under := prefix
		if !sf.Anonymous {
			under = prefix + name + "."
		}
		b.seen[st] = true
		p.walk(b, st, under, at)
		delete(b.seen, st)
	}
}

// field works out the rules on one tagged field and adds it to the plan.
func (p *structPlan) field(b *build, t reflect.Type, sf reflect.StructField, at []int, name, tag string) {
	rules, err := parseTag(tag)
	if err != nil {
		b.fail(t, sf, err.Error())
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

	fp := fieldPlan{index: at, name: name, dive: dived, zero: reflect.Zero(base(sf.Type))}
	fp.rules = p.steps(b, t, sf, before, sf.Type)
	if !dived {
		p.fields = append(p.fields, fp)
		return
	}

	et, ok := elemOf(sf.Type)
	if !ok {
		b.fail(t, sf, "dive is for a slice or an array and this field is a "+kindOf(sf.Type))
		return
	}
	fp.each = p.steps(b, t, sf, after, et)
	fp.elem = b.elemPlan(base(et))
	fp.elemZero = reflect.Zero(base(et))

	p.fields = append(p.fields, fp)
}

// elemPlan is the plan for a struct a dive reached, or nil when the elements
// are not structs.
//
// A type already being planned gets that plan back rather than a second one.
// That is what makes a comment holding a list of comments work: the element
// plan and the plan holding it are the same plan, so the recursion runs over
// the values that arrived instead of over the types, and a list that is empty
// is where it stops.
//
// The walk starts from an empty path rather than the one that got here, so the
// plan for a type is the same plan wherever it was reached from. Carrying the
// outer path in would mean a struct type checked one way inside an order and
// another way on its own, which is not something anybody could predict from
// reading the struct.
func (b *build) elemPlan(st reflect.Type) *structPlan {
	if st.Kind() != reflect.Struct {
		return nil
	}
	if sub, ok := b.elems[st]; ok {
		return sub
	}

	sub := &structPlan{}
	b.elems[st] = sub

	outer := b.seen
	b.seen = map[reflect.Type]bool{st: true}
	sub.walk(b, st, "", nil)
	b.seen = outer

	return sub
}

// steps resolves a list of rules as a tag spelled them, for a value of type ft.
func (p *structPlan) steps(b *build, t reflect.Type, sf reflect.StructField, rules []tagRule, ft reflect.Type) []step {
	if len(rules) == 0 {
		return nil
	}
	steps := make([]step, 0, len(rules))
	for _, r := range rules {
		s, err := resolve(r, ft)
		if err != nil {
			b.fail(t, sf, err.Error())
			return nil
		}
		steps = append(steps, s)
	}
	return steps
}

// resolve is what one rule from a tag runs, for a value of type ft.
//
// Everything that can be settled from the type is settled here rather than at
// run time. The bound a size rule takes is turned into a value, and a rule on a
// type it has nothing to say about is refused, so a tag that says min=x or
// email on an int is a mistake heard about on the first check and not on the
// first request that fills the field in.
func resolve(r tagRule, ft reflect.Type) (step, error) {
	if _, ok := formats[r.name]; ok && !textual(ft) {
		return step{}, fmt.Errorf("%s is a check on a string and this field is a %s", r.name, kindOf(ft))
	}
	if run, ok := plainRules[r.name]; ok {
		if len(r.params) != 0 {
			return step{}, fmt.Errorf("%s takes no parameters", r.name)
		}
		return step{run: run}, nil
	}

	if size, ok := sizeRules[r.name]; ok {
		if len(r.params) != 1 {
			return step{}, fmt.Errorf("%s takes one bound, as in %s=3", r.name, r.name)
		}
		n, err := sized(r.name, r.params[0], ft)
		if err != nil {
			return step{}, err
		}
		return step{run: func(c *Check) *Check { return size(c, n) }}, nil
	}

	if span, ok := spanRules[r.name]; ok {
		if len(r.params) != 2 {
			return step{}, fmt.Errorf("%s takes two bounds, as in %s=3 10", r.name, r.name)
		}
		lo, err := sized(r.name, r.params[0], ft)
		if err != nil {
			return step{}, err
		}
		hi, err := sized(r.name, r.params[1], ft)
		if err != nil {
			return step{}, err
		}
		return step{run: func(c *Check) *Check { return span(c, lo, hi) }}, nil
	}

	if rule, ok := ruleFor(r.name); ok {
		return step{rule: rule, params: r.params}, nil
	}

	if r.name == dive {
		return step{}, errors.New("a second dive, and one list of elements is all this reads")
	}
	return step{}, fmt.Errorf("there is no rule called %s", r.name)
}

// durationType is time.Duration, which is the one number whose bound is written
// as words rather than as digits.
var durationType = reflect.TypeFor[time.Duration]()

// sized is the bound for a size rule, once the type is known to have a size to
// compare it with.
func sized(rule, param string, ft reflect.Type) (any, error) {
	if !countable(ft) {
		return nil, fmt.Errorf("%s counts something and a %s has nothing to count", rule, kindOf(ft))
	}
	return bound(param, ft)
}

// textual is whether a value of this type can be read as a string.
//
// An interface is, as far as this can tell: what it holds is not known until a
// value arrives, and refusing the field would refuse a rule that works.
func textual(t reflect.Type) bool {
	switch t = base(t); t.Kind() {
	case reflect.String, reflect.Interface:
		return true
	}
	return false
}

// countable is whether a size rule has something to count on a value of this
// type. It is the list [sizeOf] reads, which is what a chain ends up calling.
func countable(t reflect.Type) bool {
	switch t = base(t); t.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Interface:
		return true
	}
	return false
}

// base is the type with the pointers taken off it, since a rule reads through
// them and a nil one is the same as a missing value.
func base(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// kindOf is what to call a type in a sentence about a tag, which is its own
// name when it has one and its kind when it does not.
func kindOf(t reflect.Type) string {
	if t = base(t); t.Name() != "" {
		return t.String()
	}
	return t.Kind().String()
}

// bound turns a tag's bound into the value a size rule compares against.
//
// The field's type decides what the text means, so min=1h on a time.Duration is
// an hour and min=3 on anything else is the number three. The value keeps that
// type, because the type is what the sentence prints: an hour reads 1h0m0s and
// not a count of nanoseconds.
//
// A whole number comes back as an int rather than as a wider one, which is what
// somebody writing Min(3) by hand would pass and what the generator writes into
// the code it produces, so a bound reads the same however the rule got there.
func bound(s string, ft reflect.Type) (any, error) {
	if base(ft) == durationType {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("%s is not a length of time, as in 1h30m", s)
		}
		return d, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return nil, fmt.Errorf("%s is not a number", s)
}

// elemOf is what a list holds, and whether the type is a list at all.
//
// A map is not, on purpose. The failures come out in the order they were found
// and a map has no order, so a map dived into would report the same problems in
// a different order on every run, which is not a document to test against or to
// send twice.
func elemOf(t reflect.Type) (reflect.Type, bool) {
	switch t = base(t); t.Kind() {
	case reflect.Slice, reflect.Array:
		return t.Elem(), true
	}
	return nil, false
}

// fail records a tag this cannot run, and why.
//
// The first one wins and the rest are dropped, because the sentence is for
// whoever wrote the struct and the first field with a bad tag is where they
// start reading.
func (b *build) fail(t reflect.Type, sf reflect.StructField, why string) {
	if b.err != nil {
		return
	}
	b.err = errs.Newf(errs.Internal, "validate.tag",
		"validate: cannot check %s.%s, because %s", t.Name(), sf.Name, why)
}

// run checks every field in the plan, adding what failed to v.
//
// prefix is empty except inside a dive, where it is the element's name and a
// dot, so the fields of the second address in a list are named addresses.1.city
// without the plan holding a copy of itself per element.
func (p *structPlan) run(ctx context.Context, v *V, rv reflect.Value, prefix string) error {
	for i := range p.fields {
		fp := &p.fields[i]

		fv, ok := fieldAt(rv, fp.index)
		if !ok {
			// A nil struct pointer on the way down. Whether it was allowed to
			// be nil is the pointer's own required to answer, and the fields
			// under it are not there to be wrong.
			continue
		}

		name := fp.name
		if prefix != "" {
			name = prefix + fp.name
		}

		// The pointers come off here rather than in each rule. A rule reads
		// through them anyway, and doing it once means a nil pointer arrives
		// as the zero value of what it points at instead of as a nil that a
		// size rule has nothing to count.
		val := deref(fv)
		if !val.IsValid() {
			val = fp.zero
		}

		c := v.Field(name, val.Interface())
		if err := runSteps(ctx, v, c, fp.rules); err != nil {
			return err
		}
		if !fp.dive || c.done {
			continue
		}
		if err := p.elements(ctx, v, fp, val, name); err != nil {
			return err
		}
	}
	return nil
}

// elements checks each element of a field that dived.
func (p *structPlan) elements(ctx context.Context, v *V, fp *fieldPlan, list reflect.Value, name string) error {
	for i := range list.Len() {
		item := list.Index(i)
		at := name + "." + strconv.Itoa(i)

		// A nil element is checked as the zero of what it points at, the same
		// way a nil field is, but there is nothing under it to go into. What
		// the element itself had to be is the element's own rules to say.
		sv := deref(item)
		val := sv
		if !val.IsValid() {
			val = fp.elemZero
		}

		c := v.Field(at, val.Interface())
		if err := runSteps(ctx, v, c, fp.each); err != nil {
			return err
		}
		if fp.elem == nil || c.done || !sv.IsValid() {
			continue
		}
		if err := fp.elem.run(ctx, v, sv, at+"."); err != nil {
			return err
		}
	}
	return nil
}

// runSteps puts one field through its rules.
//
// A rule this package ships with runs against the Check, which is where the
// chain already stops itself at the first failure. A registered rule is asked
// whether anything was added, since it says so by calling [Field.Fail] rather
// than by returning something.
func runSteps(ctx context.Context, v *V, c *Check, steps []step) error {
	for i := range steps {
		s := &steps[i]
		if s.run != nil {
			s.run(c)
			continue
		}
		if c.done {
			continue
		}

		before := v.bad.Len()
		err := s.rule.Validate(ctx, Field{
			Name:   c.name,
			Value:  indirect(c.value),
			Params: s.params,
			bad:    &v.bad,
			rule:   s.rule.Name(),
		})
		if err != nil {
			return err
		}
		if v.bad.Len() > before {
			c.done = true
		}
	}
	return nil
}

// fieldAt is the value at the end of an index path, and whether it is there at
// all.
func fieldAt(rv reflect.Value, index []int) (reflect.Value, bool) {
	for i, at := range index {
		if i > 0 {
			if rv = deref(rv); !rv.IsValid() {
				return reflect.Value{}, false
			}
		}
		rv = rv.Field(at)
	}
	return rv, true
}

// deref follows pointers until there are none left, and is the zero Value when
// one of them was nil.
func deref(rv reflect.Value) reflect.Value {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return reflect.Value{}
		}
		rv = rv.Elem()
	}
	return rv
}

// nameOf is the name the request uses for a field.
//
// It is the order web.Bind names a field in: a path, header, cookie, form or
// query tag says so, then the json tag, and then the field's own name in snake
// case. The two have to agree, because a request that failed to bind and a
// request that failed to validate come back in one document, and a field named
// two ways in it is a field a form cannot mark.
//
// A field the request never sends, json:"-", is one no rule here has an opinion
// about. A field bind:"-" is not the same thing: it is filled in by the handler
// rather than by the request, and a handler that filled it in wrongly is worth
// hearing about.
func nameOf(sf reflect.StructField) (string, bool) {
	for _, tag := range [...]string{"path", "header", "cookie", "form", "query"} {
		v, has := sf.Tag.Lookup(tag)
		if !has {
			continue
		}
		switch name := tagName(v); name {
		case "-":
			return "", false
		case "":
			return snake(sf.Name), true
		default:
			return name, true
		}
	}

	if v, has := sf.Tag.Lookup("json"); has {
		switch name := tagName(v); name {
		case "-":
			return "", false
		case "":
		default:
			return name, true
		}
	}
	return snake(sf.Name), true
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
// the same answer for one field to have one name.
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
