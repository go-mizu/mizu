package web

import (
	"html/template"
	"io"
	"net/http"
)

// Status sets the status the next write sends, and returns the Ctx so it reads
// in front of the write it applies to.
//
//	return c.Status(http.StatusCreated).Text(user.ID)
//
// Nothing goes out here. The status is sent by whatever writes next, and a
// handler that sets one and then writes nothing has sent a 200, because that is
// what net/http sends for a handler that returns without writing.
func (c *Ctx) Status(code int) *Ctx {
	c.live("Status")
	c.status = code
	return c
}

// StatusCode is the status that went out, or zero when nothing has gone out
// yet.
//
// It is for middleware and for an access log, which run after the handler and
// have no other way to find out. A handler already knows.
func (c *Ctx) StatusCode() int {
	c.live("StatusCode")
	return c.res.status
}

// SetHeader sets a response header.
//
// It has to happen before the first write, since the header goes out with the
// status and this is a wrapper over the map net/http sends. Setting one after
// that changes nothing and says nothing about it, which is net/http's behaviour
// and not something worth departing from.
func (c *Ctx) SetHeader(key, value string) *Ctx {
	c.live("SetHeader")
	c.res.Header().Set(key, value)
	return c
}

// SetCookie adds a cookie to the response.
//
// The cookie is sent as given. Nothing here fills in Secure, HttpOnly or
// SameSite, because a cookie that quietly gains flags is a cookie whose
// behaviour depends on which function set it. The middleware that signs and
// encrypts cookies, which does have opinions about the flags, arrives with the
// session package.
func (c *Ctx) SetCookie(ck *http.Cookie) *Ctx {
	c.live("SetCookie")
	http.SetCookie(c.res, ck)
	return c
}

// Write sends bytes, which makes a Ctx an io.Writer.
//
// It sends the status [Ctx.Status] was given, or a 200, and it is the plain way
// to stream a response that this package has no helper for.
func (c *Ctx) Write(p []byte) (int, error) {
	c.live("Write")
	c.head("")
	return c.res.Write(p)
}

// head sends the status and a content type, if neither has gone yet.
func (c *Ctx) head(contentType string) {
	if c.res.status != 0 {
		return
	}
	if contentType != "" && c.res.Header().Get("Content-Type") == "" {
		c.res.Header().Set("Content-Type", contentType)
	}
	if c.status != 0 {
		c.res.WriteHeader(c.status)
	}
}

// Text sends a string as text/plain.
func (c *Ctx) Text(s string) error {
	c.live("Text")
	c.head("text/plain; charset=utf-8")
	_, err := io.WriteString(c.res, s)
	return err
}

// HTML sends a string as text/html.
//
//	return c.HTML(page)
//
// It takes a [html/template.HTML] rather than a string, which is the standard
// library's way of saying that whoever produced this decided it was safe to send
// as markup. A string that came from a person does not convert on its own, so
// c.HTML(comment) does not compile and the conversion that would make it compile
// is a line somebody has to write and a reviewer can see.
//
// What escapes the values inside a page is the template package, and rendering
// one is [github.com/go-mizu/mizu/view]'s job. This is the whole body at once,
// for a fragment that is already built.
func (c *Ctx) HTML(h template.HTML) error {
	c.live("HTML")
	c.head("text/html; charset=utf-8")
	_, err := io.WriteString(c.res, string(h))
	return err
}

// Bytes sends bytes under a content type of the caller's choosing.
//
// An empty contentType leaves it to net/http, which sniffs the first 512 bytes.
// That is a fine answer for a file whose type nobody recorded and a poor one
// for anything a user uploaded, so the type is a parameter rather than
// something this works out.
func (c *Ctx) Bytes(contentType string, b []byte) error {
	c.live("Bytes")
	c.head(contentType)
	_, err := c.res.Write(b)
	return err
}

// Stream copies a reader into the response under a content type.
//
// It is what serves a file that is not on disk, a proxied body, or anything
// else too big to hold. Nothing here closes r, since this did not open it.
func (c *Ctx) Stream(contentType string, r io.Reader) error {
	c.live("Stream")
	c.head(contentType)
	_, err := io.Copy(c.res, r)
	return err
}

// NoContent sends a status and no body.
//
// It is the answer to a DELETE that worked and a PUT that changed nothing worth
// sending back. The status is whatever [Ctx.Status] was given, or 204.
func (c *Ctx) NoContent() error {
	c.live("NoContent")
	if c.status == 0 {
		c.status = http.StatusNoContent
	}
	c.res.WriteHeader(c.status)
	return nil
}

// Redirect sends the client somewhere else.
//
// The status is whatever [Ctx.Status] was given, or 303, which is the one that
// turns a POST into a GET and is what a form that worked wants. A permanent
// move is 301 and a redirect that keeps the method is 307, and both of those
// are worth saying out loud:
//
//	return c.Status(http.StatusMovedPermanently).Redirect("/new")
func (c *Ctx) Redirect(url string) error {
	c.live("Redirect")
	if c.status == 0 {
		c.status = http.StatusSeeOther
	}
	http.Redirect(c.res, c.r, url, c.status)
	return nil
}
