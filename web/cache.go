package web

import (
	"iter"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// cacheControl is the header all three of the caching methods write.
const cacheControl = "Cache-Control"

// CacheFor says the response may be kept for d.
//
//	return c.CacheFor(5 * time.Minute).Bytes("image/png", chart)
//
// It writes max-age in seconds, rounded down, and public alongside it unless
// [Ctx.Private] has already said otherwise. public is what lets a shared cache
// keep a response to a request that carried an Authorization header, which is
// the one case where leaving it out changes what a proxy is allowed to do.
//
// A negative duration is zero, which is a response that may be cached and has
// to be revalidated every time.
func (c *Ctx) CacheFor(d time.Duration) *Ctx {
	c.live("CacheFor")

	if d < 0 {
		d = 0
	}
	age := "max-age=" + strconv.FormatInt(int64(d/time.Second), 10)

	if has(c.res.Header().Get(cacheControl), "private") {
		return c.cache(age)
	}
	return c.cache("public", age)
}

// NoStore says the response is not to be kept anywhere.
//
//	return c.NoStore().Text(oneTimeCode)
//
// It is the whole statement rather than the four directives that usually get
// sent together. no-store means a cache must not write the response down, so
// no-cache, must-revalidate and max-age=0 next to it are three answers to a
// question the first one already closed, and the HTTP/1.0 proxies they were
// aimed at are not on the path any more.
//
// It replaces whatever was on the header, and a [Ctx.CacheFor] after it
// replaces this, since two contradictory statements about the same response
// come down to whichever was made last.
func (c *Ctx) NoStore() *Ctx {
	c.live("NoStore")
	return c.cache("no-store")
}

// Private says the response belongs to one user, so a shared cache must not
// keep it and the browser still may.
//
//	return c.Private().CacheFor(time.Minute).Text(balance)
//
// Order does not matter. This takes public off if [Ctx.CacheFor] put it there,
// and a CacheFor after this leaves the private on rather than replacing it.
func (c *Ctx) Private() *Ctx {
	c.live("Private")
	return c.cache("private")
}

// ETag sets the entity tag, which is the version of what is about to be sent.
//
//	c.ETag(post.Version)
//
// The tag is quoted here, so a caller passes the version and not the syntax.
// One that arrives already quoted is not quoted twice, a W/ prefix is kept, and
// a quote or a control character inside is dropped rather than closing the
// header early. An empty tag removes the header, since a version of nothing is
// not a version.
//
// What this is for is a handler that knows its version without rendering the
// response, from a row's updated_at or a content hash it already has. The
// middleware in [github.com/go-mizu/mizu/web/mw] hashes what the handler wrote,
// which needs no cooperation and needs the handler to have run.
func (c *Ctx) ETag(tag string) *Ctx {
	c.live("ETag")

	tag = quoteETag(tag)
	if tag == "" {
		c.res.Header().Del("ETag")
		return c
	}
	c.res.Header().Set("ETag", tag)
	return c
}

// LastModified sets when what is about to be sent last changed.
//
//	c.LastModified(post.UpdatedAt)
//
// The header carries whole seconds, so anything finer is lost on the way out
// and [Ctx.NotModified] compares what a client would have seen. The zero time
// removes the header rather than sending the year 1.
func (c *Ctx) LastModified(t time.Time) *Ctx {
	c.live("LastModified")

	if t.IsZero() {
		c.res.Header().Del("Last-Modified")
		return c
	}
	c.res.Header().Set("Last-Modified", t.UTC().Format(http.TimeFormat))
	return c
}

// NotModified answers a conditional request when the client already has what
// the handler was about to send.
//
//	func show(c *web.Ctx) error {
//		p, err := posts.Find(c.Context(), c.Param("id"))
//		if err != nil {
//			return err
//		}
//		c.ETag(p.Version).LastModified(p.UpdatedAt)
//		if c.NotModified() {
//			return nil
//		}
//		return web.JSON(c, p)
//	}
//
// It reads the tag and the time already on the response, so the two calls in
// front of it are what it compares against and there is no second place to say
// them. It writes the 304 itself and returns true, and a handler that gets true
// returns without sending a body.
//
// If-None-Match decides on its own when the request carries one, and
// If-Modified-Since is read only when it does not. That order is RFC 9110
// section 13.2.2 and it is the part hand-written implementations get wrong: a
// client that sends both is saying the tag is what it trusts, and a response
// that answers the date instead sends a body to somebody who has it.
//
// The comparison is weak, which is what If-None-Match asks for, so W/"x" and
// "x" are the same version. A request that sends * is asking whether anything
// is there at all, and a handler that got as far as calling this has something.
//
// Only GET and HEAD are answered. A conditional PUT or DELETE that fails its
// precondition is a 412 and it is the handler's to send, since only the handler
// knows whether it went ahead.
func (c *Ctx) NotModified() bool {
	c.live("NotModified")

	if c.res.Status() != 0 {
		return false
	}
	if c.r.Method != http.MethodGet && c.r.Method != http.MethodHead {
		return false
	}

	h := c.res.Header()
	if match := c.r.Header.Get("If-None-Match"); match != "" {
		if !matchETag(match, h.Get("ETag")) {
			return false
		}
	} else if !unchangedSince(c.r.Header.Get("If-Modified-Since"), h.Get("Last-Modified")) {
		return false
	}

	// The same fields net/http.ServeContent takes off a 304, so a handler that
	// answers one by hand and a file that answers one through ServeContent send
	// the same shape of response. A 304 has no body, so the three that describe
	// one would be describing the body it is not sending.
	h.Del("Content-Type")
	h.Del("Content-Length")
	h.Del("Content-Encoding")
	if h.Get("ETag") != "" {
		h.Del("Last-Modified")
	}

	c.res.WriteHeader(http.StatusNotModified)
	return true
}

// cache rewrites Cache-Control with add on it, dropping what add contradicts
// and keeping everything else that was there.
//
// Keeping the rest is what lets middleware and a handler both have an opinion,
// and it is why the three methods can be called in any order.
func (c *Ctx) cache(add ...string) *Ctx {
	h := c.res.Header()

	// Nothing to merge with is the ordinary case, and it is worth taking
	// straight rather than through a builder that starts empty.
	if h.Get(cacheControl) == "" {
		h.Set(cacheControl, strings.Join(add, ", "))
		return c
	}

	var b strings.Builder

	for d := range list(h.Get(cacheControl)) {
		if contradicted(d, add) {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		b.WriteString(d)
	}
	for _, d := range add {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		b.WriteString(d)
	}

	h.Set(cacheControl, b.String())
	return c
}

// contradicted reports whether a directive already on the header is answered by
// one of the new ones.
func contradicted(have string, add []string) bool {
	name := directive(have)
	for _, a := range add {
		switch directive(a) {
		case name:
			return true
		case "no-store":
			return true
		case "public", "private":
			if name == "public" || name == "private" || name == "no-store" {
				return true
			}
		case "max-age":
			if name == "no-store" {
				return true
			}
		}
	}
	return false
}

// directive is the name of a Cache-Control directive, without its value.
func directive(d string) string {
	name, _, _ := strings.Cut(d, "=")
	return strings.ToLower(strings.TrimSpace(name))
}

// has reports whether a Cache-Control header carries a directive.
func has(header, name string) bool {
	for d := range list(header) {
		if directive(d) == name {
			return true
		}
	}
	return false
}

// matchETag reports whether tag is one of the entity tags in an If-None-Match
// header.
//
// The comparison is weak, which is what If-None-Match asks for: the weak marker
// says the two representations mean the same thing rather than that they are
// the same bytes, and that is the question a cache is asking.
func matchETag(header, tag string) bool {
	want := strings.TrimPrefix(tag, "W/")

	for e := range list(header) {
		if e == "*" {
			return true
		}
		if want != "" && strings.TrimPrefix(e, "W/") == want {
			return true
		}
	}
	return false
}

// unchangedSince reports whether a Last-Modified header is no later than an
// If-Modified-Since one.
//
// Both are header values rather than times, so both are already the whole
// seconds the format carries and there is nothing to round. A missing or
// unreadable date is not an answer, so the response is sent.
func unchangedSince(since, modified string) bool {
	if since == "" || modified == "" {
		return false
	}

	asked, err := http.ParseTime(since)
	if err != nil {
		return false
	}
	changed, err := http.ParseTime(modified)
	if err != nil {
		return false
	}
	return !changed.After(asked)
}

// list walks a comma separated header value.
//
// A comma inside a quoted string is part of the value rather than a separator,
// which is true of an entity tag and of a Cache-Control directive with a field
// list on it. Splitting on every comma is the ordinary way to get both of those
// wrong.
func list(header string) iter.Seq[string] {
	return func(yield func(string) bool) {
		quoted, start := false, 0

		for i := 0; i < len(header); i++ {
			switch header[i] {
			case '"':
				quoted = !quoted
			case ',':
				if quoted {
					continue
				}
				if e := strings.TrimSpace(header[start:i]); e != "" && !yield(e) {
					return
				}
				start = i + 1
			}
		}

		if e := strings.TrimSpace(header[start:]); e != "" {
			yield(e)
		}
	}
}

// quoteETag puts a tag in the form the header wants, or answers with nothing
// when there is no tag left to write.
func quoteETag(tag string) string {
	weak := false
	if rest, ok := strings.CutPrefix(tag, "W/"); ok {
		weak, tag = true, rest
	}

	tag = strings.Map(etagc, strings.Trim(tag, `"`))
	if tag == "" {
		return ""
	}
	if weak {
		return `W/"` + tag + `"`
	}
	return `"` + tag + `"`
}

// etagc keeps the characters an entity tag is allowed to carry, which is
// everything printable except the quote that would end it early.
func etagc(r rune) rune {
	if r == 0x21 || (r >= 0x23 && r <= 0x7e) || r >= 0x80 {
		return r
	}
	return -1
}
