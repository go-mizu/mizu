package mizu

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Handler is Mizu's handler shape.
// It composes well with middleware and central error handling.
type Handler func(*Ctx) error

// Ctx wraps an HTTP request and response with small, explicit helpers.
// Not safe for concurrent use without external synchronization.
type Ctx struct {
	req         *http.Request
	res         http.ResponseWriter
	rc          *http.ResponseController
	status      int
	wroteHeader bool
	log         *slog.Logger
}

// newCtx creates a Ctx from net/http types (router adapter).
func newCtx(w http.ResponseWriter, r *http.Request, lg *slog.Logger) *Ctx {
	if lg == nil {
		lg = slog.Default()
	}
	return &Ctx{
		req:    r,
		res:    w,
		rc:     http.NewResponseController(w),
		status: http.StatusOK,
		log:    lg,
	}
}

// --- Accessors ---

// Request returns the underlying http.Request.
func (c *Ctx) Request() *http.Request { return c.req }

// Response returns the underlying http.ResponseWriter.
func (c *Ctx) Response() http.ResponseWriter { return c.res }

// Header returns the response headers.
func (c *Ctx) Header() http.Header { return c.res.Header() }

// Context returns the request context.
func (c *Ctx) Context() context.Context { return c.req.Context() }

// Logger returns the request-scoped logger (from the router).
func (c *Ctx) Logger() *slog.Logger {
	return c.log
}

// --- Request helpers ---

// Param returns a path parameter captured by Go 1.22 router.
func (c *Ctx) Param(name string) string { return c.req.PathValue(name) }

// Query returns the first query value for key.
func (c *Ctx) Query(key string) string {
	if c.req.URL == nil {
		return ""
	}
	return c.req.URL.Query().Get(key)
}

// QueryValues returns all query parameters.
func (c *Ctx) QueryValues() url.Values {
	if c.req.URL == nil {
		return url.Values{}
	}
	return c.req.URL.Query()
}

// Form parses and returns form values.
func (c *Ctx) Form() (url.Values, error) {
	if err := c.req.ParseForm(); err != nil {
		return nil, err
	}
	return c.req.Form, nil
}

// MultipartForm parses multipart form data and returns a cleanup func
// to remove any temporary files created on disk.
// Always call the returned cleanup function when done:
//
//	form, cleanup, err := c.MultipartForm(32 << 20)
//	if err != nil { return err }
//	defer cleanup()
//	// use form.File and form.Value safely
func (c *Ctx) MultipartForm(maxMemory int64) (*multipart.Form, func(), error) {
	if err := c.req.ParseMultipartForm(maxMemory); err != nil {
		return nil, func() {}, err
	}
	return c.req.MultipartForm, func() {
		if c.req.MultipartForm != nil {
			_ = c.req.MultipartForm.RemoveAll()
		}
	}, nil
}

// Cookie returns a named cookie or http.ErrNoCookie.
func (c *Ctx) Cookie(name string) (*http.Cookie, error) { return c.req.Cookie(name) }

// ClientIP returns the client IP (best-effort).
// It trusts X-Forwarded-For and X-Real-IP only if they contain a parseable IP.
// For production behind proxies, consider injecting a stricter resolver.
func (c *Ctx) ClientIP() string {
	if xff := c.req.Header.Get("X-Forwarded-For"); xff != "" {
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	if xr := c.req.Header.Get("X-Real-IP"); xr != "" && net.ParseIP(xr) != nil {
		return xr
	}
	host, _, err := net.SplitHostPort(c.req.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	return c.req.RemoteAddr
}

// --- Request body binding ---

// BindJSON reads JSON into v with a max size limit.
// It disallows unknown fields and rejects trailing data.
func (c *Ctx) BindJSON(v any, max int64) error {
	r := c.req.Body
	if max > 0 {
		r = http.MaxBytesReader(c.res, r, max)
	}
	dec := newJSONDecoder(r)
	decDisallowUnknownFields(dec)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	// Ensure single JSON value by verifying no more tokens remain
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return errors.New("invalid JSON: trailing data")
		}
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// --- Response helpers ---

// Status sets the response status (applied on first write).
func (c *Ctx) Status(code int) {
	if code > 0 {
		c.status = code
	}
}

// StatusCode returns the currently set status code (default 200).
func (c *Ctx) StatusCode() int { return c.status }

// HeaderIfNone sets header key to value only if key is not already present.
func (c *Ctx) HeaderIfNone(key, value string) {
	if c.Header().Get(key) == "" {
		c.Header().Set(key, value)
	}
}

// NoContent sends a 204 No Content response.
func (c *Ctx) NoContent() error {
	return c.writeHeaderNow(http.StatusNoContent, "")
}

// Redirect sends a redirect with Location header.
func (c *Ctx) Redirect(code int, location string) error {
	if code == 0 {
		code = http.StatusFound
	}
	c.Header().Set("Location", location)
	return c.writeHeaderNow(code, "")
}

// SetCookie adds a Set-Cookie header.
func (c *Ctx) SetCookie(ck *http.Cookie) {
	if ck != nil {
		http.SetCookie(c.res, ck)
	}
}

// JSON writes a JSON response.
func (c *Ctx) JSON(code int, v any) error {
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		c.HeaderIfNone("Content-Type", "application/json; charset=utf-8")
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	enc := newJSONEncoder(c.res)
	encSetEscapeHTML(enc, false)
	return enc.Encode(v)
}

// HTML writes a UTF-8 HTML response.
func (c *Ctx) HTML(code int, html string) error {
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		c.HeaderIfNone("Content-Type", "text/html; charset=utf-8")
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	_, err := io.WriteString(c.res, html)
	return err
}

// Text writes a UTF-8 text response. If s is not valid UTF-8, it is sent as octet-stream.
func (c *Ctx) Text(code int, s string) error {
	if !utf8.ValidString(s) {
		return c.Bytes(code, []byte(s), "application/octet-stream")
	}
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		c.HeaderIfNone("Content-Type", "text/plain; charset=utf-8")
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	_, err := io.WriteString(c.res, s)
	return err
}

// Bytes writes raw bytes with an optional content type.
// If contentType is empty and no Content-Type is set, it defaults to application/octet-stream.
func (c *Ctx) Bytes(code int, b []byte, contentType string) error {
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		if contentType != "" {
			c.Header().Set("Content-Type", contentType)
		} else if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", "application/octet-stream")
		}
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	_, err := c.res.Write(b)
	return err
}

// Write writes bytes, ensuring headers are sent once.
func (c *Ctx) Write(b []byte) (int, error) {
	if !c.wroteHeader {
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	return c.res.Write(b)
}

// WriteString writes a string, ensuring headers are sent once.
func (c *Ctx) WriteString(s string) (int, error) {
	return c.Write([]byte(s))
}

// File serves a file from disk.
// If a non-200 status has been set via Status, it will be respected and
// a best-effort Content-Type will be applied before delegating to ServeFile.
func (c *Ctx) File(path string) error {
	return c.fileWithCode(0, path)
}

// Download serves a file as an attachment with RFC 5987 filename support.
// If a non-200 status has been set via Status, it will be respected.
func (c *Ctx) Download(path, name string) error {
	return c.downloadWithCode(0, path, name)
}

// FileCode serves a file with an explicit status code.
// If code is 0, it behaves like File.
func (c *Ctx) FileCode(code int, path string) error {
	return c.fileWithCode(code, path)
}

// DownloadCode serves an attachment with an explicit status code.
// If code is 0, it behaves like Download.
func (c *Ctx) DownloadCode(code int, path, name string) error {
	return c.downloadWithCode(code, path, name)
}

func (c *Ctx) fileWithCode(code int, path string) error {
	// If caller wants a non-default status, or c.status != 200,
	// write headers now with a best-effort Content-Type so ServeFile will not override status.
	needHeader := (code > 0) || (c.status != http.StatusOK)
	if needHeader && !c.wroteHeader {
		ct := ""
		if ext := filepath.Ext(path); ext != "" {
			if guess := mime.TypeByExtension(ext); guess != "" {
				ct = guess
			}
		}
		if err := c.writeHeaderNow(firstNonZero(code, c.status), ct); err != nil {
			return err
		}
	}
	http.ServeFile(c.res, c.req, path)
	return nil
}

func (c *Ctx) downloadWithCode(code int, path, name string) error {
	if name == "" {
		name = filepath.Base(path)
	}
	ascii := sanitizeToken(name)
	disp := fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(name))
	c.Header().Set("Content-Disposition", disp)
	if ext := filepath.Ext(name); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			c.Header().Set("Content-Type", ct)
		}
	}
	needHeader := (code > 0) || (c.status != http.StatusOK)
	if needHeader && !c.wroteHeader {
		if err := c.writeHeaderNow(firstNonZero(code, c.status), c.Header().Get("Content-Type")); err != nil {
			return err
		}
	}
	http.ServeFile(c.res, c.req, path)
	return nil
}

// Stream streams output using fn.
func (c *Ctx) Stream(fn func(io.Writer) error) error {
	if !c.wroteHeader {
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", "application/octet-stream")
		}
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	if err := fn(c.res); err != nil {
		return err
	}
	_ = c.Flush()
	return nil
}

// SSE writes Server-Sent Events from ch.
func (c *Ctx) SSE(ch <-chan any) error {
	if _, ok := c.res.(http.Flusher); !ok {
		return errors.New("SSE requires http.Flusher")
	}
	if !c.wroteHeader {
		h := c.Header()
		h.Set("Content-Type", "text/event-stream; charset=utf-8")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-c.Context().Done():
			return c.Context().Err()
		case v, ok := <-ch:
			if !ok {
				_, _ = io.WriteString(c.res, "event: end\ndata: {}\n\n")
				_ = c.Flush()
				return nil
			}
			buf, err := jsonMarshal(v)
			if err != nil {
				buf = []byte(strconv.Quote(fmt.Sprint(v)))
			}
			if _, err := c.Write(append([]byte("data: "), append(buf, '\n', '\n')...)); err != nil {
				return err
			}
			_ = c.Flush()
		case <-tick.C:
			_, _ = io.WriteString(c.res, ": ping\n\n")
			_ = c.Flush()
		}
	}
}

// --- Low-level passthroughs ---

// Flush flushes buffered data to the client.
func (c *Ctx) Flush() error {
	if f, ok := c.res.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

// Hijack hijacks the underlying connection.
func (c *Ctx) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := c.res.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijack not supported")
}

// SetWriteDeadline sets the write deadline on the connection.
func (c *Ctx) SetWriteDeadline(t time.Time) error { return c.rc.SetWriteDeadline(t) }

// EnableFullDuplex enables concurrent read and write on the connection.
func (c *Ctx) EnableFullDuplex() error { return c.rc.EnableFullDuplex() }

// --- Internal helpers ---

func (c *Ctx) writeHeaderNow(code int, contentType string) error {
	if code > 0 {
		c.status = code
	}
	if c.wroteHeader {
		return nil
	}
	if contentType != "" {
		c.Header().Set("Content-Type", contentType)
	}
	c.res.WriteHeader(c.status)
	c.wroteHeader = true
	return nil
}

func sanitizeToken(s string) string {
	// Replace control chars and tspecials with underscore
	const bad = "()<>@,;:\\\"/[]?={} \t\r\n"
	repl := func(r rune) rune {
		if r < 0x20 || r >= 0x7f || strings.ContainsRune(bad, r) {
			return '_'
		}
		return r
	}
	return strings.Map(repl, s)
}

func firstNonZero(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}
