package web

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/go-mizu/mizu/errs"
)

// Bind decodes the request into a T.
//
//	type search struct {
//		Q    string `query:"q"`
//		Page int    `query:"page"`
//	}
//
//	func list(c *web.Ctx) error {
//		in, err := web.Bind[search](c)
//		if err != nil {
//			return err
//		}
//		...
//	}
//
// The error is an [github.com/go-mizu/mizu/errs.Error] of kind Invalid, which
// is a 400, carrying one Field per value that would not decode. Nothing is
// returned alongside it: a request that did not decode has no half of it worth
// acting on.
//
// What each field is read from, and what it is read from it, is in the package
// comment under Binding.
func Bind[T any](c *Ctx) (T, error) {
	var dst T
	if err := c.Bind(&dst); err != nil {
		var zero T
		return zero, err
	}
	return dst, nil
}

// BindInto is [Bind] for a value the caller already has.
//
// It is for a struct that embeds another one and is filled in stages, and for a
// handler that wants the fields that did decode when one of them did not.
func BindInto[T any](c *Ctx, dst *T) error {
	return c.Bind(dst)
}

// Bind decodes the request into the struct dst points at.
//
// It is [Bind] without the type parameter, for when the type is not known where
// the call is written. Passing anything that is not a pointer to a struct is a
// mistake in the program rather than in the request, and it comes back as an
// error of kind Internal.
func (c *Ctx) Bind(dst any) error {
	c.live("Bind")

	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return errs.Newf(errs.Internal, "bind.target",
			"web: cannot bind into %T, which is not a pointer to a struct", dst)
	}
	v = v.Elem()

	p := planFor(v.Type())
	if p.err != nil {
		return p.err
	}
	return p.run(c, v)
}

// A source is where one field's value comes from.
//
// The body is not one of them. A JSON or an XML body is decoded over the struct
// by the decoder that speaks it, after these have run, so that a member the
// body carries wins over a query parameter of the same name.
type source uint8

const (
	fromValues source = iota // the query string, and a form in the body
	fromPath
	fromHeader
	fromCookie
)

// A plan is how one struct type is filled in, worked out once and kept.
//
// Building it is the reflection, and it happens on the first request that binds
// the type. Everything after that walks a flat slice and calls a closure per
// field, which is why a nested struct is flattened here rather than at run
// time.
type plan struct {
	fields []binding
	values bool  // whether anything reads the query or the form
	err    error // a field this cannot bind, reported to every caller
}

// A binding is one field of the struct and where it is read from.
type binding struct {
	index []int  // the path to the field, through embedded and nested structs
	name  string // the name the request uses, which is not the field's name
	src   source
	set   setter
	list  bool // a slice, which takes every value sent under the name
	text  bool // whether an empty value means anything to the target
}

// plans is the plan for each type that has been bound.
var plans sync.Map // reflect.Type -> *plan

// planFor is the plan for t, built the first time t is bound.
//
// Two requests binding the same type at once may both build one, and one of the
// two is thrown away. That is cheaper than a lock held across the reflection,
// and the plans are equal anyway.
func planFor(t reflect.Type) *plan {
	if p, ok := plans.Load(t); ok {
		return p.(*plan)
	}
	p := new(plan)
	p.walk(t, "", nil, map[reflect.Type]bool{t: true})
	actual, _ := plans.LoadOrStore(t, p)
	return actual.(*plan)
}

// walk adds every field of t to the plan, following embedded and nested structs
// down.
//
// prefix is what a nested struct's fields are named under, so a City inside an
// Address is address.city. An embedded struct has no prefix, which is what
// embedding means everywhere else in Go and in encoding/json.
//
// seen is the struct types on the way here, so a type that holds one of itself
// stops rather than going round. The field it stopped at is left unbound: a
// tree is not a shape a query string has, and a body decoder handles it without
// help from here.
func (p *plan) walk(t reflect.Type, prefix string, index []int, seen map[reflect.Type]bool) {
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		name, src, tagged, ok := nameOf(sf)
		if !ok {
			continue
		}

		// The index is copied rather than appended to in place, since every
		// field below this one would otherwise share the array and overwrite
		// each other's last element.
		at := append(index[:len(index):len(index)], i)

		ft := sf.Type
		list := false
		if ft.Kind() == reflect.Slice && !isBytes(ft) {
			list, ft = true, ft.Elem()
		}

		set, text := setterFor(ft)
		if set != nil {
			// Only the query and the form have a prefix on them. A path
			// parameter, a header and a cookie are flat namespaces the request
			// shares with everything else, and address.x-country is not the
			// name of a header wherever the field happens to sit.
			if src == fromValues {
				name = prefix + name
				p.values = true
			}
			p.fields = append(p.fields, binding{
				index: at, name: name, src: src, set: set, list: list, text: text,
			})
			continue
		}

		// Not a value a string turns into. A struct is worth going into, and
		// anything else is worth saying out loud only when somebody asked for
		// it by name.
		st := ft
		for st.Kind() == reflect.Pointer {
			st = st.Elem()
		}
		if list || st.Kind() != reflect.Struct {
			if tagged {
				p.fail(t, sf)
			}
			continue
		}

		// A type that holds one of itself stops here, tag or no tag. The tag is
		// not the mistake: a query string has no way to carry a tree, and the
		// body decoder that does handles it without a plan from this.
		if seen[st] {
			continue
		}

		under := prefix
		if !sf.Anonymous || tagged {
			under = prefix + name + "."
		}
		seen[st] = true
		p.walk(st, under, at, seen)
		delete(seen, st)
	}
}

// fail records a field somebody tagged and this cannot read.
//
// A tag is somebody saying where the value comes from, so a tag on a type that
// no query parameter can be turned into is a mistake worth hearing about. An
// untagged field of the same type is not: it is a field of the struct that
// binding has nothing to do with, and there is one in nearly every struct.
func (p *plan) fail(t reflect.Type, sf reflect.StructField) {
	if p.err != nil {
		return
	}
	p.err = errs.Newf(errs.Internal, "bind.field",
		"web: cannot bind %s.%s, because nothing turns a string into a %s",
		t.Name(), sf.Name, sf.Type)
}

// nameOf is the name the request uses for a field, and where it is read from.
//
// A path, header or cookie tag says so and is the whole answer. A form or query
// tag names the field in the query string and in a form body, which net/http
// keeps together and so does this. With none of those, the name comes from the
// json tag and then from the field's own name in snake case, so a struct
// written for a JSON body binds from a query string without a second set of
// tags on it.
//
// The false result is a field binding leaves alone, which is bind:"-" and
// json:"-".
func nameOf(sf reflect.StructField) (name string, src source, tagged, ok bool) {
	if v, has := sf.Tag.Lookup("bind"); has && tagName(v) == "-" {
		return "", 0, false, false
	}

	for _, t := range []struct {
		tag string
		src source
	}{
		{"path", fromPath},
		{"header", fromHeader},
		{"cookie", fromCookie},
		{"form", fromValues},
		{"query", fromValues},
	} {
		v, has := sf.Tag.Lookup(t.tag)
		if !has {
			continue
		}
		switch name := tagName(v); name {
		case "-":
			return "", 0, false, false
		case "":
			return snake(sf.Name), t.src, true, true
		default:
			return name, t.src, true, true
		}
	}

	if v, has := sf.Tag.Lookup("json"); has {
		name := tagName(v)
		if name == "-" {
			return "", 0, false, false
		}
		if name != "" {
			return name, fromValues, false, true
		}
	}
	return snake(sf.Name), fromValues, false, true
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
// It is here rather than borrowed from
// [github.com/go-mizu/mizu/str.Snake], which does the same thing correctly for
// any language, because the input is a Go identifier and the general version
// carries the Unicode casing tables with it. Anybody taking this package would
// be taking those tables to lower case the word Page.
//
// A byte that is not an ASCII letter goes through as it is, which leaves an
// identifier written in another script alone rather than cutting it up.
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

// run fills v in from the request.
//
// Every field that would not decode is reported, rather than the first one, so
// a form comes back with everything wrong with it marked at once. That is what
// a form redisplay needs and it is what validation does in the next step, so
// the two read the same way.
func (p *plan) run(c *Ctx, v reflect.Value) error {
	if p.values {
		c.values()
		if c.formErr != nil {
			return formError(c.formErr)
		}
	}

	var bad []errs.Field
	for i := range p.fields {
		b := &p.fields[i]
		got, ok := b.read(c)
		if !ok {
			continue
		}
		if err := b.assign(fieldByIndex(v, b.index), got); err != nil {
			bad = append(bad, errs.Field{Name: b.name, Code: err.code, Msg: err.msg})
		}
	}
	if bad == nil {
		return nil
	}
	return errs.New(errs.Invalid, "bind.invalid", "That request could not be read.").WithFields(bad...)
}

// formError is what a body that would not parse comes back as.
func formError(err error) error {
	var big *http.MaxBytesError
	if errors.As(err, &big) {
		return errs.Wrapf(err, errs.TooLarge, "bind.too_large",
			"That request body is over the %d byte limit.", big.Limit)
	}
	return errs.Wrap(err, errs.Invalid, "bind.unreadable", "That form could not be read.")
}

// read is every value the request sent under this field's name, and whether it
// sent any.
//
// A name the request did not use leaves the field alone, so a struct is filled
// in from what arrived rather than reset to the zero value of everything that
// did not.
func (b *binding) read(c *Ctx) ([]string, bool) {
	switch b.src {
	case fromPath:
		if v := c.params.Get(b.name); v != "" {
			return []string{v}, true
		}
	case fromHeader:
		if v := c.r.Header.Values(b.name); len(v) > 0 {
			return v, true
		}
	case fromCookie:
		if ck, err := c.r.Cookie(b.name); err == nil {
			return []string{ck.Value}, true
		}
	default:
		return c.input(b.name)
	}
	return nil, false
}

// assign puts what arrived into the field.
//
// An empty value is treated as nothing having been sent, unless the field is
// one an empty string means something to. A text input somebody did not fill in
// posts an empty value, and a number field is not wrong because a form has a
// blank on it, it is unset. Requiring one is what validation is for.
func (b *binding) assign(v reflect.Value, got []string) *badValue {
	if !b.list {
		if got[0] == "" && !b.text {
			return nil
		}
		return b.set(v, got[0])
	}

	out := reflect.MakeSlice(v.Type(), 0, len(got))
	elem := v.Type().Elem()
	for _, s := range got {
		if s == "" && !b.text {
			continue
		}
		e := reflect.New(elem).Elem()
		if err := b.set(e, s); err != nil {
			return err
		}
		out = reflect.Append(out, e)
	}
	v.Set(out)
	return nil
}

// fieldByIndex is the field at the end of an index path, with the pointers on
// the way there filled in.
//
// A nested struct behind a pointer is allocated here rather than when the plan
// is built, so a request that names nothing inside it leaves it nil.
func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

// input is every value sent under a name, body first.
//
// The body wins over the query string, which is the precedence a POST that also
// carries a page number in the URL needs. net/http keeps the two apart in
// PostForm and Form, and the only reason to read them in this order is that
// Form has them in one order for an urlencoded body and the other for a
// multipart one.
func (c *Ctx) input(key string) ([]string, bool) {
	c.values()
	if v, ok := c.r.PostForm[key]; ok && len(v) > 0 {
		return v, true
	}
	if v, ok := c.r.Form[key]; ok && len(v) > 0 {
		return v, true
	}
	return nil, false
}
