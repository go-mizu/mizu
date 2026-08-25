package web

import (
	"encoding"
	"iter"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// A Binder is a type that fills itself from a request.
//
// [Bind] uses it when the type has one, so a struct with a generated binder on
// it is bound by that and everything else is bound by reflection. Nothing else
// changes: the call is the same call, the error is the same error, and a
// handler cannot tell which one ran.
//
//	//mizu:bind
//	type Search struct {
//		Q    string `query:"q"`
//		Page int    `query:"page"`
//	}
//
// mizu gen bind writes the method. Writing one by hand is allowed and is
// nobody's idea of a good afternoon: the reflective binder is a hundred lines
// of decisions about tags, empty values and precedence, and a hand-written
// binder that gets one of them wrong is a bug that only shows up for the type
// it was written for.
type Binder interface {
	// BindRequest fills the value from c, reporting every field that would not
	// decode rather than the first.
	BindRequest(c *Ctx) error
}

// A Binding is one bind in progress: what the request carries, and what has
// gone wrong with it so far.
//
// It is the vocabulary a generated binder is written against. It is exported
// because generated code has to call it, and it is here rather than in the
// generator so that both binders make the same decisions about the same
// request.
type Binding struct {
	c *Ctx

	bad []errs.Field
	err error
}

// Binding starts a bind.
//
// It is what the first line of a generated BindRequest calls, and [Binding.Err]
// is what the last line returns.
func (c *Ctx) Binding() *Binding {
	c.live("Binding")
	return &Binding{c: c}
}

// Err is what the bind came to: nil, or one error carrying a Field per value
// that would not decode.
//
// A request whose form would not parse at all comes back as that instead, since
// there are no fields to report on when nothing could be read.
func (b *Binding) Err() error {
	switch {
	case b.err != nil:
		return b.err
	case b.bad != nil:
		return invalid(b.bad)
	}
	return nil
}

// Invalid records that the value sent under name is not one the field takes.
//
// The code is the machine name a client switches on and the message is what
// goes next to the field on a form. Neither should quote what arrived: a
// message built from the input is a message that renders whatever somebody
// sent.
func (b *Binding) Invalid(name, code, msg string) {
	b.bad = append(b.bad, errs.Field{Name: name, Code: code, Msg: msg})
}

func (b *Binding) invalid(name string, v *badValue) {
	b.Invalid(name, v.code, v.msg)
}

// fail records something that stopped the bind rather than something wrong with
// one field. The first one is the one reported, since the rest are what
// happened after it.
func (b *Binding) fail(err error) {
	if b.err == nil {
		b.err = err
	}
}

// Values is every name and value the request carries in its query string and in
// a form body, one pair at a time and each exactly as it was sent.
//
//	for name, value := range b.Values() {
//		switch name {
//		case "q":
//			v.Q = value
//		case "page":
//			web.Int(b, &v.Page, name, value)
//		}
//	}
//
// This is where a generated binder earns its keep. The reflective binder asks
// net/http for the values, which builds a map with a slice in every entry and a
// string in every slice before anybody reads the first field. A generated
// binder knows the names it wants, so it reads the pairs as they come and
// writes each one straight into the field it belongs to.
//
// The body comes first and the query string after it, minus the names the body
// already used, which is the rule that makes a body outrank a query parameter
// of the same name. A body that is not a form is left for [Binding.Body], so a
// JSON request sees its query string here and its members there.
//
// A form that will not parse stops the loop and is reported by [Binding.Err].
// Fields set before it stopped keep what they were given, which [Bind] throws
// away with the rest of the struct.
func (b *Binding) Values() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		if b.err != nil {
			return
		}
		c := b.c
		query := c.r.URL.RawQuery

		if bodyOf(c.r) != bodyForm {
			b.scan(query, nil, yield)
			return
		}
		if strings.HasPrefix(c.r.Header.Get("Content-Type"), "multipart/") {
			b.parts(query, yield)
			return
		}

		body, ok := b.urlencoded()
		if !ok {
			return
		}
		if body == "" {
			b.scan(query, nil, yield)
			return
		}

		// The names the body used are collected only when there is a query
		// string to compare them against, which is the request that carries a
		// page number in the URL and everything else in the body.
		var used []string
		if query != "" {
			used = sentNames(body)
		}
		if b.scan(body, nil, yield) {
			b.scan(query, used, yield)
		}
	}
}

// urlencoded is a form body as it arrived, and whether the caller should carry
// on.
func (b *Binding) urlencoded() (string, bool) {
	// Reading through BodyBytes rather than around it means the bytes are the
	// ones Ctx.BodyBytes hands out afterwards, and that a handler which reads
	// the body first and binds second scans what it already has.
	data, err := b.c.BodyBytes()
	if err != nil {
		b.fail(formError(err))
		return "", false
	}
	if len(data) > maxFormSize {
		b.fail(errs.Newf(errs.TooLarge, "bind.too_large",
			"That form is over the %d byte limit.", maxFormSize))
		return "", false
	}
	return string(data), true
}

// parts yields a multipart body's fields, and then the query string minus the
// names the body used.
//
// A multipart body is parsed by net/http rather than scanned here, because
// net/http has to build the map anyway to find the file parts in it and reading
// the parts a second time would save nothing. The fields come out in whatever
// order the map is in, which is no order at all, and the values sent under one
// name keep theirs.
func (b *Binding) parts(query string, yield func(string, string) bool) {
	c := b.c
	c.values()
	if c.formErr != nil {
		b.fail(formError(c.formErr))
		return
	}

	var used []string
	for name, list := range c.r.PostForm {
		if query != "" {
			used = append(used, name)
		}
		for _, value := range list {
			if !yield(name, value) {
				return
			}
		}
	}
	b.scan(query, used, yield)
}

// maxFormSize is how large an urlencoded body may be, which is the number
// net/http uses for the same job.
//
// It is a floor rather than the limit. What limits a body properly is
// http.MaxBytesReader, installed per route by the middleware in front of the
// handler, and this is what stops a request with no such middleware in front of
// it from being a way to fill the heap.
const maxFormSize = 10 << 20

// scan walks an urlencoded string and yields each pair, skipping the names in
// used.
//
// It reports whether the caller should carry on, which is false when the
// consumer stopped or when a pair would not unescape.
func (b *Binding) scan(s string, used []string, yield func(string, string) bool) bool {
	for len(s) > 0 {
		var pair string
		pair, s, _ = strings.Cut(s, "&")
		if pair == "" {
			continue
		}

		key, value, _ := strings.Cut(pair, "=")
		name, err := url.QueryUnescape(key)
		if err != nil {
			b.fail(formError(err))
			return false
		}
		if contains(used, name) {
			continue
		}
		got, err := url.QueryUnescape(value)
		if err != nil {
			b.fail(formError(err))
			return false
		}
		if !yield(name, got) {
			return false
		}
	}
	return true
}

// sentNames is every name an urlencoded string uses, with the duplicates left in.
//
// A slice rather than a set: a request carries a handful of parameters, and
// walking ten strings costs less than the map that would save the walk.
func sentNames(s string) []string {
	var out []string
	for len(s) > 0 {
		var pair string
		pair, s, _ = strings.Cut(s, "&")
		key, _, _ := strings.Cut(pair, "=")
		if name, err := url.QueryUnescape(key); err == nil {
			out = append(out, name)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Path is a route parameter, and whether the route had one.
func (b *Binding) Path(name string) (string, bool) {
	v := b.c.params.Get(name)
	return v, v != ""
}

// Header is the first value sent under a header name, and whether it was sent.
func (b *Binding) Header(name string) (string, bool) {
	v := b.c.r.Header.Values(name)
	if len(v) == 0 {
		return "", false
	}
	return v[0], true
}

// Headers is every value sent under a header name, for a field that takes a
// list of them.
func (b *Binding) Headers(name string) []string {
	return b.c.r.Header.Values(name)
}

// Cookie is a cookie's value, and whether it was sent.
func (b *Binding) Cookie(name string) (string, bool) {
	ck, err := b.c.r.Cookie(name)
	if err != nil {
		return "", false
	}
	return ck.Value, true
}

// Files is every file sent under a name, which is empty when none were.
//
// It reads the form itself, so a struct made of nothing but files does not have
// to range [Binding.Values] first to have one to read.
func (b *Binding) Files(name string) []*Upload {
	c := b.c
	c.values()
	if c.formErr != nil {
		b.fail(formError(c.formErr))
		return nil
	}
	if c.r.MultipartForm == nil {
		return nil
	}
	got := c.r.MultipartForm.File[name]
	if len(got) == 0 {
		return nil
	}
	out := make([]*Upload, len(got))
	for i, fh := range got {
		out[i] = newUpload(fh)
	}
	return out
}

// Body decodes the request body over dst, which is the struct being bound.
//
// It goes last, after the values, so a member the body carries wins over a
// query parameter of the same name. Both decoders merge into a struct that
// already has something in it rather than resetting it, which is what makes
// that work with nothing keeping track of where each field came from.
//
// A member the struct has no field for is a mistake. [Binding.BodyAllowUnknown]
// is the other answer, for a struct that embeds [AllowUnknown].
func (b *Binding) Body(dst any) { b.body(dst, false) }

// BodyAllowUnknown is [Binding.Body] for a struct that takes whatever it is
// sent.
func (b *Binding) BodyAllowUnknown(dst any) { b.body(dst, true) }

func (b *Binding) body(dst any, lax bool) {
	if b.err != nil {
		return
	}
	fields, err := b.c.decodeBody(dst, lax)
	if err != nil {
		b.fail(err)
		return
	}
	b.bad = append(b.bad, fields...)
}

// Int puts a whole number into a field of any integer type.
//
// An empty value is a field nobody filled in rather than a mistake, so it is
// left alone and the result is false. A blank number input posts one, and
// saying it had to be filled in is validation's job.
//
// The result is whether the field was set, which is what a slice field appends
// on.
func Int[T ~int | ~int8 | ~int16 | ~int32 | ~int64](b *Binding, dst *T, name, value string) bool {
	if value == "" {
		return false
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		b.invalid(name, numberError(err, "Must be a whole number."))
		return false
	}
	// The parse is always at 64 bits and the width is checked by converting,
	// which costs nothing and saves the type having to say how wide it is.
	out := T(n)
	if int64(out) != n {
		b.invalid(name, &badValue{"out_of_range", "Is too large for this field."})
		return false
	}
	*dst = out
	return true
}

// Uint is [Int] for a field that holds no negative numbers.
func Uint[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](b *Binding, dst *T, name, value string) bool {
	if value == "" {
		return false
	}
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		b.invalid(name, numberError(err, "Must be a whole number and not negative."))
		return false
	}
	out := T(n)
	if uint64(out) != n {
		b.invalid(name, &badValue{"out_of_range", "Is too large for this field."})
		return false
	}
	*dst = out
	return true
}

// Float is [Int] for a field that holds a number with a point in it.
func Float[T ~float32 | ~float64](b *Binding, dst *T, name, value string) bool {
	if value == "" {
		return false
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		b.invalid(name, numberError(err, "Must be a number."))
		return false
	}
	// A number too large for a float32 becomes an infinity on the way in, and
	// one that arrived as an infinity did not.
	out := T(f)
	if math.IsInf(float64(out), 0) && !math.IsInf(f, 0) {
		b.invalid(name, &badValue{"out_of_range", "Is too large for this field."})
		return false
	}
	*dst = out
	return true
}

// Bool puts a true or a false into a field, taking the on and off a checkbox
// sends as well as the words strconv knows.
func Bool[T ~bool](b *Binding, dst *T, name, value string) bool {
	if value == "" {
		return false
	}
	switch value {
	case "on", "yes":
		*dst = true
		return true
	case "off", "no":
		*dst = false
		return true
	}

	v, err := strconv.ParseBool(value)
	if err != nil {
		b.invalid(name, &badValue{"invalid_boolean", "Must be true or false."})
		return false
	}
	*dst = T(v)
	return true
}

// Time puts a date, or a date and a time, into a field.
//
// It takes RFC 3339, which is what a program sends, and the three shapes the
// date and datetime-local inputs send, which carry no zone and are read as UTC.
func Time(b *Binding, dst *time.Time, name, value string) bool {
	if value == "" {
		return false
	}
	for _, layout := range timeLayouts {
		t, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		*dst = t
		return true
	}
	b.invalid(name, &badValue{"invalid_time", "Must be a date, such as 2006-01-02, or a date and a time."})
	return false
}

// Duration puts a length of time into a field, in the shape time.ParseDuration
// reads.
func Duration[T ~int64](b *Binding, dst *T, name, value string) bool {
	if value == "" {
		return false
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		b.invalid(name, &badValue{"invalid_duration", "Must be a length of time, such as 30s or 5m."})
		return false
	}
	*dst = T(d)
	return true
}

// Text hands the value to a field that reads itself out of text, which covers
// netip.Addr, a uuid, and every id type an application writes for itself.
//
// What the type says when it refuses is not repeated. Its error was written for
// whoever called it, and putting it in a response is how the name of an
// internal type ends up on a page.
func Text(b *Binding, dst encoding.TextUnmarshaler, name, value string) bool {
	if value == "" {
		return false
	}
	if err := dst.UnmarshalText([]byte(value)); err != nil {
		b.invalid(name, &badValue{"invalid_value", "Is not in the right format."})
		return false
	}
	return true
}
