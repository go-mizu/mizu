package mizutest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

// Request is a request being built. Every method returns the same Request, so a
// call reads as one statement:
//
//	res := app.Post("/posts").
//		JSON(map[string]any{"title": "Hello"}).
//		WithHeader("Accept", "application/json").
//		Do()
//
// Nothing happens until [Request.Do]. A mistake made along the way, a body that
// will not marshal or a path that will not parse, is kept and reported there,
// so that building a request never fails in the middle of a chain.
type Request struct {
	app     *App
	method  string
	target  string
	query   url.Values
	header  http.Header
	cookies []*http.Cookie
	body    []byte
	remote  string
	err     error
}

// Get starts a GET request. The target is a path, with a query string on it if
// you want one there rather than in [Request.WithQuery].
func (a *App) Get(target string) *Request { return a.Request(http.MethodGet, target) }

// Post starts a POST request.
func (a *App) Post(target string) *Request { return a.Request(http.MethodPost, target) }

// Put starts a PUT request.
func (a *App) Put(target string) *Request { return a.Request(http.MethodPut, target) }

// Patch starts a PATCH request.
func (a *App) Patch(target string) *Request { return a.Request(http.MethodPatch, target) }

// Delete starts a DELETE request.
func (a *App) Delete(target string) *Request { return a.Request(http.MethodDelete, target) }

// Head starts a HEAD request.
func (a *App) Head(target string) *Request { return a.Request(http.MethodHead, target) }

// Options starts an OPTIONS request.
func (a *App) Options(target string) *Request { return a.Request(http.MethodOptions, target) }

// Request starts a request with any method, for the ones without a helper.
func (a *App) Request(method, target string) *Request {
	return &Request{
		app:    a,
		method: method,
		target: target,
		query:  url.Values{},
		header: http.Header{},
	}
}

// JSON sets the body to v encoded as JSON, and the content type to
// application/json.
//
// v is anything json.Marshal takes. A []byte, a json.RawMessage or a string is
// treated as JSON already written and passed through, so a test can hold the
// exact bytes it wants to send.
func (r *Request) JSON(v any) *Request {
	switch b := v.(type) {
	case nil:
		r.body = []byte("null")
	case json.RawMessage:
		r.body = b
	case []byte:
		r.body = b
	case string:
		r.body = []byte(b)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return r.fail("mizutest: encoding the JSON body: %w", err)
		}
		r.body = encoded
	}
	r.header.Set("Content-Type", "application/json")
	return r
}

// Form sets the body to values encoded as a form, and the content type to
// application/x-www-form-urlencoded.
func (r *Request) Form(values url.Values) *Request {
	r.body = []byte(values.Encode())
	r.header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// Multipart sets the body to a multipart form built by fn, and the content type
// to the boundary that goes with it.
//
//	app.Post("/avatar").Multipart(func(w *multipart.Writer) error {
//		f, err := w.CreateFormFile("avatar", "me.png")
//		if err != nil {
//			return err
//		}
//		_, err = f.Write(png)
//		return err
//	}).Do()
//
// The writer is closed afterwards, so fn does not have to.
func (r *Request) Multipart(fn func(*multipart.Writer) error) *Request {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := fn(w); err != nil {
		return r.fail("mizutest: building the multipart body: %w", err)
	}
	if err := w.Close(); err != nil {
		return r.fail("mizutest: closing the multipart body: %w", err)
	}
	r.body = buf.Bytes()
	r.header.Set("Content-Type", w.FormDataContentType())
	return r
}

// Body reads the body from a reader. It sets no content type, since the reader
// does not say what it holds.
func (r *Request) Body(body io.Reader) *Request {
	b, err := io.ReadAll(body)
	if err != nil {
		return r.fail("mizutest: reading the body: %w", err)
	}
	r.body = b
	return r
}

// Raw sets the body to bytes exactly as given, with no content type. It is for
// the tests that are about what the server does with something malformed.
func (r *Request) Raw(b []byte) *Request {
	r.body = b
	return r
}

// WithHeader sets a header, replacing any value already there.
func (r *Request) WithHeader(key, value string) *Request {
	r.header.Set(key, value)
	return r
}

// AddHeader adds a header, keeping any value already there. It is for the few
// headers that are a list, such as Accept and Via.
func (r *Request) AddHeader(key, value string) *Request {
	r.header.Add(key, value)
	return r
}

// WithHeaders sets several headers at once.
func (r *Request) WithHeaders(h http.Header) *Request {
	for k, vs := range h {
		r.header[http.CanonicalHeaderKey(k)] = append([]string(nil), vs...)
	}
	return r
}

// WithQuery adds a query parameter. Calling it twice with the same key sends
// the key twice, which is what a repeated parameter means on the wire.
func (r *Request) WithQuery(key, value string) *Request {
	r.query.Add(key, value)
	return r
}

// WithCookie adds a cookie by name and value. Use [Request.WithCookieValue] for
// one that needs its other fields set.
func (r *Request) WithCookie(name, value string) *Request {
	return r.WithCookieValue(&http.Cookie{Name: name, Value: value})
}

// WithCookieValue adds a cookie as given.
func (r *Request) WithCookieValue(c *http.Cookie) *Request {
	r.cookies = append(r.cookies, c)
	return r
}

// WithIP sets the address the request appears to come from, which is what a
// handler reads from RemoteAddr and what a rate limiter or an audit log keys
// on. A bare address gets a port, since RemoteAddr always has one.
func (r *Request) WithIP(addr string) *Request {
	if !strings.Contains(addr, ":") {
		addr += ":12345"
	}
	r.remote = addr
	return r
}

// Do sends the request through the handler and returns the response.
//
// There is no server and no socket. The request goes to the handler directly
// through an httptest recorder, on this goroutine, and Do returns when the
// handler does.
func (r *Request) Do() *Response {
	tb := r.app.tb
	tb.Helper()

	if r.err != nil {
		tb.Fatal(r.err)
		return nil
	}

	req, err := r.build()
	if err != nil {
		tb.Fatal(err)
		return nil
	}

	before := len(r.app.log.Entries())
	rec := httptest.NewRecorder()
	r.app.handler.ServeHTTP(rec, req)

	res := rec.Result()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		tb.Fatalf("mizutest: reading the response body: %v", err)
		return nil
	}
	res.Body.Close()

	// A real server drops the body of a HEAD response and the recorder does
	// not, so a test would see something here that no client ever will. Drop it
	// too, or the fixture is easier to pass than production.
	if r.method == http.MethodHead {
		body = nil
	}

	return &Response{
		tb:   tb,
		app:  r.app,
		req:  req,
		sent: r.body,
		res:  res,
		body: body,
		logs: r.app.log.Entries()[before:],
	}
}

// build turns the pieces into a request, with the fixture's context on it so a
// handler that reads the clock reads the one the test set.
func (r *Request) build() (*http.Request, error) {
	target := r.target
	if len(r.query) > 0 {
		sep := "?"
		if strings.Contains(target, "?") {
			sep = "&"
		}
		target += sep + r.query.Encode()
	}

	u, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, fmt.Errorf("mizutest: %s %s: %w", r.method, target, err)
	}

	var body io.Reader
	if r.body != nil {
		body = bytes.NewReader(r.body)
	}
	req := httptest.NewRequestWithContext(r.app.ctx, r.method, u.String(), body)
	for k, vs := range r.header {
		req.Header[k] = vs
	}
	for _, c := range r.cookies {
		req.AddCookie(c)
	}
	if r.remote != "" {
		req.RemoteAddr = r.remote
	}
	return req, nil
}

// fail records the first thing that went wrong and keeps the chain going, so
// that Do is the one place a test can fail from.
func (r *Request) fail(format string, args ...any) *Request {
	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
	return r
}
