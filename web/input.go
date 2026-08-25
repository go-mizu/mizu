package web

import (
	"errors"
	"io"
	"mime"
	"net/url"
	"strings"
)

// Query is a query string parameter, or the empty string.
func (c *Ctx) Query(key string) string {
	c.live("Query")
	return c.r.URL.Query().Get(key)
}

// QueryDefault is a query string parameter, or def when it was not sent or was
// sent empty.
func (c *Ctx) QueryDefault(key, def string) string {
	c.live("QueryDefault")
	if v := c.r.URL.Query().Get(key); v != "" {
		return v
	}
	return def
}

// QueryAll is every value sent under one key, for the ?tag=a&tag=b shape.
func (c *Ctx) QueryAll(key string) []string {
	c.live("QueryAll")
	return c.r.URL.Query()[key]
}

// Has reports whether the key was sent at all, in the query or in the form.
//
// A checkbox that is off is not sent, so this is the difference between a field
// somebody cleared and a field that was never on the form. [Ctx.Filled] is the
// stricter one.
func (c *Ctx) Has(key string) bool {
	c.live("Has")
	_, ok := c.values()[key]
	return ok
}

// Filled reports whether the key was sent with something in it.
func (c *Ctx) Filled(key string) bool {
	c.live("Filled")
	for _, v := range c.values()[key] {
		if v != "" {
			return true
		}
	}
	return false
}

// Form is a form field, or the empty string.
//
// It reads the body for the shapes a form comes in, application/x-www-form-
// urlencoded and multipart/form-data, and it reads the query string too, which
// is what net/http does and what a handler that does not care where a value
// came from wants.
//
// Reading the body happens once. An error reading it, a body over whatever
// limit is in front of this handler, or a body that is not a form leaves every
// field empty, which is the same answer as a form with nothing in it. A handler
// that needs to tell those apart is binding rather than reading fields, and
// binding reports what went wrong.
func (c *Ctx) Form(key string) string {
	c.live("Form")
	return c.values().Get(key)
}

// values is the query and the form together, parsed at most once.
//
// The record of having parsed is kept here rather than read off the request,
// because a body that would not parse leaves r.Form set to exactly what a
// request with no form leaves it set to, and asking twice about a body that is
// already gone is asking twice about nothing.
func (c *Ctx) values() url.Values {
	if !c.form {
		c.form = true

		// ParseMultipartForm handles both shapes. It calls ParseForm itself,
		// which reads the query string and an urlencoded body, and it stops
		// there when the body is not multipart. The limit is how much of a
		// multipart body stays in memory: the parts over it go to temporary
		// files that net/http removes when the request is done.
		_ = c.r.ParseMultipartForm(formMemory)
	}
	return c.r.Form
}

// formMemory is how much of a multipart body is held in memory before the rest
// goes to a temporary file.
//
// It is not a limit on the body. What limits the body is http.MaxBytesReader,
// which the middleware in front of the handler installs, because a limit
// belongs where somebody can set it per route rather than in a constant here.
const formMemory = 32 << 20

// Header is a request header, or the empty string.
func (c *Ctx) Header(key string) string {
	c.live("Header")
	return c.r.Header.Get(key)
}

// Cookie is a cookie's value, and whether it was there.
func (c *Ctx) Cookie(name string) (string, bool) {
	c.live("Cookie")
	ck, err := c.r.Cookie(name)
	if err != nil {
		return "", false
	}
	return ck.Value, true
}

// Bearer is the token out of an Authorization: Bearer header, and whether there
// was one.
//
// The scheme is matched without regard to case, which RFC 7235 asks for and
// which clients rely on more than they should.
func (c *Ctx) Bearer() (string, bool) {
	c.live("Bearer")
	const prefix = "bearer "
	h := c.r.Header.Get("Authorization")
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(h[len(prefix):])
	return token, token != ""
}

// IsAJAX reports whether the request says it came from a script.
//
// It is the X-Requested-With header that jQuery started and that every library
// after it copied. Nothing makes a browser send it, so this is a hint and not a
// fact, and it is here because the redirect-or-status decision on a form post
// has wanted it for fifteen years.
func (c *Ctx) IsAJAX() bool {
	c.live("IsAJAX")
	return strings.EqualFold(c.r.Header.Get("X-Requested-With"), "XMLHttpRequest")
}

// WantsJSON reports whether the client would rather have JSON than HTML.
//
// It reads Accept and compares what the client said about JSON with what it
// said about HTML, so */* is not a vote for either and text/html;q=0.9 loses to
// application/json. A request that sent no Accept at all, which is most
// programs and no browsers, is taken at its Content-Type: a JSON body asking
// for anything is a client that wants JSON back.
//
// It answers one question, which is the one nearly every handler asks. The full
// negotiation, with more than two offers and a 406 when none of them fit,
// arrives with the response helpers.
func (c *Ctx) WantsJSON() bool {
	c.live("WantsJSON")

	accept := c.r.Header.Get("Accept")
	if accept == "" {
		return isJSON(c.r.Header.Get("Content-Type"))
	}

	json, html := 0.0, 0.0
	for spec := range strings.SplitSeq(accept, ",") {
		kind, q := quality(spec)
		switch {
		case isJSON(kind):
			json = max(json, q)
		case kind == "text/html", kind == "application/xhtml+xml":
			html = max(html, q)
		}
	}
	return json > 0 && json >= html
}

// isJSON reports whether a media type is one of the JSON ones.
func isJSON(kind string) bool {
	if i := strings.IndexByte(kind, ';'); i >= 0 {
		kind = strings.TrimSpace(kind[:i])
	}
	return kind == "application/json" || strings.HasSuffix(kind, "+json")
}

// quality takes one entry of an Accept header apart into the media type and
// what the client thinks of it.
//
// A entry with no q is worth 1, which is what RFC 9110 says. One that will not
// parse is worth nothing, since a header this broken is not evidence of
// anything.
func quality(spec string) (kind string, q float64) {
	kind, params, err := mime.ParseMediaType(strings.TrimSpace(spec))
	if err != nil {
		return "", 0
	}
	q = 1
	if v, ok := params["q"]; ok {
		q = 0
		if f, err := parseQ(v); err == nil {
			q = f
		}
	}
	return kind, q
}

// parseQ reads a quality value, which is a number between 0 and 1 with at most
// three digits after the point.
func parseQ(s string) (float64, error) {
	var whole, frac, scale float64 = 0, 0, 1
	i := 0
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		whole = whole*10 + float64(s[i]-'0')
	}
	if i == 0 {
		return 0, errors.New("no digits")
	}
	if i < len(s) && s[i] == '.' {
		for i++; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
			scale *= 10
			frac = frac*10 + float64(s[i]-'0')
		}
	}
	if i != len(s) {
		return 0, errors.New("trailing text")
	}
	if v := whole + frac/scale; v <= 1 {
		return v, nil
	}
	return 0, errors.New("over one")
}

// Body is the request body.
//
// It is whatever net/http handed over, so a limit installed in front of this
// handler is still on it and closing it is still the server's job. Reading it
// here and then calling [Ctx.BodyBytes] gets what is left rather than what was
// sent.
func (c *Ctx) Body() io.ReadCloser {
	c.live("Body")
	return c.r.Body
}

// BodyBytes is the whole body, read once and kept.
//
// A webhook signature is computed over the bytes that arrived, which means
// reading them before deciding whether to trust them, and it is why this is
// here rather than left to whatever decodes the body. It is explicit because
// holding every request body in memory is a way to run out of it, and the thing
// that stops that is the limit the middleware puts in front, not this.
func (c *Ctx) BodyBytes() ([]byte, error) {
	c.live("BodyBytes")
	if c.read {
		return c.body, nil
	}
	b, err := io.ReadAll(c.r.Body)
	if err != nil {
		return nil, err
	}
	c.body, c.read = b, true
	return b, nil
}
