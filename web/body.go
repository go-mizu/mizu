package web

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/go-mizu/mizu/errs"
)

// AllowUnknown is embedded in a struct to say that a member of the body the
// struct has no field for is not a mistake.
//
//	type Webhook struct {
//		web.AllowUnknown
//
//		Event string `json:"event"`
//		ID    string `json:"id"`
//	}
//
// Without it a body carrying titl where the struct says title is a 400 naming
// titl, because a request that was quietly half read is a bug somebody finds in
// production rather than in a test.
//
// With it the extra members are dropped. That is what a webhook payload wants,
// since the sender adds fields on its own schedule and a receiver that reads
// two of them should not break the day a third arrives.
//
// It says nothing about the query string or a form. Those carry whatever a
// browser felt like sending, an analytics parameter on the end of a link is not
// a client error, and nothing rejects an unknown one either way.
type AllowUnknown struct{}

var allowUnknownType = reflect.TypeFor[AllowUnknown]()

// laxTypes remembers which types embed [AllowUnknown], since the answer is a
// property of the type and the question comes up once per request.
var laxTypes sync.Map // reflect.Type -> bool

// laxFor reports whether t embeds [AllowUnknown].
//
// Only the fields of t itself are looked at. A struct that embeds one which
// embeds AllowUnknown is not covered, because a type that takes whatever it is
// sent should say so where somebody reading it can see.
func laxFor(t reflect.Type) bool {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return false
	}
	if v, ok := laxTypes.Load(t); ok {
		return v.(bool)
	}

	lax := false
	for i := range t.NumField() {
		if t.Field(i).Type == allowUnknownType {
			lax = true
			break
		}
	}
	laxTypes.Store(t, lax)
	return lax
}

// JSON decodes the request body as JSON into dst.
//
//	var payload map[string]any
//	if err := c.JSON(&payload); err != nil {
//		return err
//	}
//
// It is the body and nothing else: no query string, no path parameters, no
// headers. [Ctx.Bind] is what fills a struct from the whole request, and this
// is here for the times the body is not a struct, a webhook payload read into a
// map or a patch document kept as it arrived.
//
// It reads the body as JSON whatever the request said it was, since the caller
// asking for JSON is the caller saying so. A member the target has no field for
// is an error unless the target embeds [AllowUnknown], the same as binding.
//
// [Ctx.BodyBytes] before this reads the same body rather than an empty one, so
// verifying a webhook signature and then decoding what it signed is two calls
// and one body.
func (c *Ctx) JSON(dst any) error {
	c.live("JSON")

	fields, err := c.decodeJSON(dst, laxFor(reflect.TypeOf(dst)))
	if err != nil {
		return err
	}
	if fields == nil {
		return nil
	}
	return invalid(fields)
}

// decodeJSON reads the body as JSON and sorts what came back into the one
// member it was about and everything else.
func (c *Ctx) decodeJSON(dst any, lax bool) ([]errs.Field, error) {
	err := jsonDecode(c.bodyReader(), dst, lax)
	if err == nil {
		return nil, nil
	}
	if f, ok := jsonField(err); ok {
		return []errs.Field{f}, nil
	}
	return nil, bodyError(err)
}

// decodeBody reads the body into dst according to what the request says it is.
//
// The fields it returns are the ones the body was wrong about, when the decoder
// said enough to name them. Everything else is the error, and an error here
// stops binding: a body that will not parse has nothing left in it to report on
// field by field.
func (c *Ctx) decodeBody(dst any, lax bool) ([]errs.Field, error) {
	switch bodyOf(c.r) {
	case bodyNone, bodyForm:
		// A form has already been read by then, into the same values the query
		// string went into, which is the whole reason the two share a source.
		return nil, nil

	case bodyJSON:
		return c.decodeJSON(dst, lax)

	case bodyXML:
		// encoding/xml has no way to refuse an element the target has no field
		// for and no way to say which element it was unhappy about, so an XML
		// body is all or nothing and AllowUnknown means nothing to it.
		if err := xml.NewDecoder(c.bodyReader()).Decode(dst); err != nil {
			return nil, bodyError(err)
		}
		return nil, nil
	}

	return nil, errs.New(errs.Unsupported, "bind.content_type",
		"That content type is not one this endpoint reads.")
}

// bodyReader is the body, from the start when [Ctx.BodyBytes] has already read
// it and from wherever it is otherwise.
//
// Only the caller who asked for the bytes pays for holding them. A body nobody
// read is decoded as it arrives, which is what keeps a large upload or a long
// export off the heap.
func (c *Ctx) bodyReader() io.Reader {
	if c.read {
		return bytes.NewReader(c.body)
	}
	return c.r.Body
}

// bodyError is what a body that would not decode comes back as.
//
// The decoder's own message is wrapped rather than shown. It was written for
// whoever called the decoder and it quotes the input, so a message built from
// it is a message that renders whatever somebody sent.
func bodyError(err error) error {
	if over := tooLarge(err); over != nil {
		return over
	}
	return errs.Wrap(err, errs.Invalid, "bind.body", "That request body could not be read.")
}

// formError is what a form that would not parse comes back as.
func formError(err error) error {
	if over := tooLarge(err); over != nil {
		return over
	}
	return errs.Wrap(err, errs.Invalid, "bind.unreadable", "That form could not be read.")
}

// tooLarge is what a body over the limit comes back as, and nil when the size
// was not what was wrong with it.
//
// The limit is in the message because a client that sent too much has to be
// told how much is too much, and it is the one number in any of these that
// says nothing about what arrived.
func tooLarge(err error) error {
	var big *http.MaxBytesError
	if !errors.As(err, &big) {
		return nil
	}
	return errs.Wrapf(err, errs.TooLarge, "bind.too_large",
		"That request body is over the %d byte limit.", big.Limit)
}

// invalid is the one error a request with things wrong in it comes back as.
func invalid(fields []errs.Field) error {
	return errs.New(errs.Invalid, "bind.invalid", "That request could not be read.").WithFields(fields...)
}

// A bodyKind is what the request says its body is.
type bodyKind uint8

const (
	bodyNone  bodyKind = iota // there is not one, or nothing said what it is
	bodyForm                  // urlencoded or multipart, read as values
	bodyJSON                  // application/json, or a type ending in +json
	bodyXML                   // application/xml, text/xml, or a type ending in +xml
	bodyOther                 // something said, and not something this reads
)

// bodyOf reads the request enough to know what to do with its body.
func bodyOf(r *http.Request) bodyKind {
	// GET and HEAD do not carry one. A body on either is not something to
	// decode, it is something a proxy on the way here is about to disagree with
	// the next proxy about.
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		return bodyNone
	}

	// Zero is a body that is not there. A body of unknown length, which is what
	// chunked transfer encoding leaves behind, is minus one and is a body.
	if r.ContentLength == 0 {
		return bodyNone
	}

	kind, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch {
	case kind == "":
		// Nothing said what it is. RFC 9110 says to guess at bytes, and a
		// handler that wanted the bytes would have asked for them, so this
		// reads the query string and leaves the body where it is.
		return bodyNone
	case isJSON(kind):
		return bodyJSON
	case kind == "application/xml", kind == "text/xml", strings.HasSuffix(kind, "+xml"):
		return bodyXML
	case kind == "application/x-www-form-urlencoded", kind == "multipart/form-data":
		return bodyForm
	}
	return bodyOther
}
