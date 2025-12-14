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
	request     *http.Request
	writer      http.ResponseWriter
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
		request: r,
		writer:  w,
		rc:      http.NewResponseController(w),
		status:  http.StatusOK,
		log:     lg,
	}
}

// --- Accessors ---

func (c *Ctx) Request() *http.Request      { return c.request }
func (c *Ctx) Writer() http.ResponseWriter { return c.writer }
func (c *Ctx) Header() http.Header         { return c.writer.Header() }
func (c *Ctx) Context() context.Context    { return c.request.Context() }
func (c *Ctx) Logger() *slog.Logger        { return c.log }

// --- Request helpers ---

// Param returns a path parameter captured by Go 1.22 router.
func (c *Ctx) Param(name string) string { return c.request.PathValue(name) }

// Query returns the first query value for key.
func (c *Ctx) Query(key string) string {
	if c.request.URL == nil {
		return ""
	}
	return c.request.URL.Query().Get(key)
}

// QueryValues returns all query parameters.
func (c *Ctx) QueryValues() url.Values {
	if c.request.URL == nil {
		return url.Values{}
	}
	return c.request.URL.Query()
}

// Form parses and returns form values.
func (c *Ctx) Form() (url.Values, error) {
	if err := c.request.ParseForm(); err != nil {
		return nil, err
	}
	return c.request.Form, nil
}

// MultipartForm parses multipart form data and returns a cleanup func to remove temp files.
func (c *Ctx) MultipartForm(maxMemory int64) (*multipart.Form, func(), error) {
	if err := c.request.ParseMultipartForm(maxMemory); err != nil {
		return nil, func() {}, err
	}
	return c.request.MultipartForm, func() {
		if c.request.MultipartForm != nil {
			_ = c.request.MultipartForm.RemoveAll()
		}
	}, nil
}

// Cookie returns a named cookie or http.ErrNoCookie.
func (c *Ctx) Cookie(name string) (*http.Cookie, error) { return c.request.Cookie(name) }

// --- Request body binding ---

// Bind reads JSON into v with a max size limit.
// It disallows unknown fields and rejects trailing data.
//
// Call Bind before writing the response, since MaxBytesReader may need to emit an
// error status when the limit is exceeded.
func (c *Ctx) Bind(v any, max int64) error {
	r := c.request.Body
	if max > 0 {
		r = http.MaxBytesReader(c.writer, r, max)
	}
	dec := newJSONDecoder(r)
	decDisallowUnknownFields(dec)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return errors.New("invalid JSON: trailing data")
		}
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// --- Writer helpers ---

// Status sets the response status (applied on first write).
func (c *Ctx) Status(code int) {
	if code > 0 {
		c.status = code
	}
}

// StatusCode returns the currently set status code (default 200).
func (c *Ctx) StatusCode() int { return c.status }

// NoContent sends a 204 No Content response.
func (c *Ctx) NoContent() error {
	c.writeHeaderNow(http.StatusNoContent, "")
	return nil
}

// Redirect sends a redirect with Location header.
func (c *Ctx) Redirect(code int, location string) error {
	if code == 0 {
		code = http.StatusFound
	}
	c.Header().Set("Location", location)
	c.writeHeaderNow(code, "")
	return nil
}

// SetCookie adds a Set-Cookie header.
func (c *Ctx) SetCookie(ck *http.Cookie) {
	if ck != nil {
		http.SetCookie(c.writer, ck)
	}
}

// JSON writes a JSON response.
func (c *Ctx) JSON(code int, v any) error {
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	enc := newJSONEncoder(c.writer)
	encSetEscapeHTML(enc, false)
	return enc.Encode(v)
}

// HTML writes a UTF-8 HTML response.
func (c *Ctx) HTML(code int, html string) error {
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	_, err := io.WriteString(c.writer, html)
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
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	_, err := io.WriteString(c.writer, s)
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
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	_, err := c.writer.Write(b)
	return err
}

// Write writes bytes, ensuring headers are sent once.
func (c *Ctx) Write(b []byte) (int, error) {
	if !c.wroteHeader {
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	return c.writer.Write(b)
}

// WriteString writes a string, ensuring headers are sent once.
func (c *Ctx) WriteString(s string) (int, error) {
	if !c.wroteHeader {
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	return io.WriteString(c.writer, s)
}

// File serves a file from disk.
//
// If code is 0, it uses the currently set status (default 200).
// If code is non-zero, it overrides the currently set status.
//
// If the effective status is non-200, headers are written before delegating to ServeFile
// so net/http does not force a 200 on the response.
func (c *Ctx) File(code int, filePath string) error {
	c.fileWithCode(code, filePath)
	return nil
}

// Download serves a file as an attachment with RFC 5987 filename support.
//
// If code is 0, it uses the currently set status (default 200).
// If code is non-zero, it overrides the currently set status.
func (c *Ctx) Download(code int, filePath, name string) error {
	c.downloadWithCode(code, filePath, name)
	return nil
}

func (c *Ctx) fileWithCode(code int, filePath string) {
	needHeader := (code > 0) || (c.status != http.StatusOK)
	if needHeader && !c.wroteHeader {
		ct := ""
		if ext := filepath.Ext(filePath); ext != "" {
			if guess := mime.TypeByExtension(ext); guess != "" {
				ct = guess
			}
		}
		c.writeHeaderNow(firstNonZero(code, c.status), ct)
	}
	http.ServeFile(c.writer, c.request, filePath)
}

func (c *Ctx) downloadWithCode(code int, filePath, name string) {
	if name == "" {
		name = filepath.Base(filePath)
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
		c.writeHeaderNow(firstNonZero(code, c.status), c.Header().Get("Content-Type"))
	}
	http.ServeFile(c.writer, c.request, filePath)
}

// Stream streams output using fn.
func (c *Ctx) Stream(fn func(io.Writer) error) error {
	if !c.wroteHeader {
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", "application/octet-stream")
		}
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	if err := fn(c.writer); err != nil {
		return err
	}
	c.Flush()
	return nil
}

// SSE writes Server-Sent Events from ch.
func (c *Ctx) SSE(ch <-chan any) error {
	if _, ok := c.writer.(http.Flusher); !ok {
		return errors.New("SSE requires http.Flusher")
	}
	if !c.wroteHeader {
		h := c.Header()
		h.Set("Content-Type", "text/event-stream; charset=utf-8")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
		c.Flush()
	}
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-c.Context().Done():
			return c.Context().Err()
		case v, ok := <-ch:
			if !ok {
				_, _ = io.WriteString(c.writer, "event: end\ndata: {}\n\n")
				c.Flush()
				return nil
			}
			buf, err := jsonMarshal(v)
			if err != nil {
				buf = []byte(strconv.Quote(fmt.Sprint(v)))
			}
			if _, err := c.Write(append([]byte("data: "), append(buf, '\n', '\n')...)); err != nil {
				return err
			}
			c.Flush()
		case <-tick.C:
			_, _ = io.WriteString(c.writer, ": ping\n\n")
			c.Flush()
		}
	}
}

// --- Low-level passthroughs ---

// Flush flushes buffered data to the client.
func (c *Ctx) Flush() {
	if f, ok := c.writer.(http.Flusher); ok {
		f.Flush()
	}
}

// SetWriter swaps the underlying ResponseWriter.
// ResponseController is bound to the writer, so we rebuild it.
func (c *Ctx) SetWriter(w http.ResponseWriter) {
	c.writer = w
	c.rc = http.NewResponseController(w)
}

// Hijack hijacks the underlying connection.
func (c *Ctx) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := c.writer.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijack not supported")
}

// SetWriteDeadline sets the write deadline on the connection.
func (c *Ctx) SetWriteDeadline(t time.Time) error { return c.rc.SetWriteDeadline(t) }

// EnableFullDuplex enables concurrent read and write on the connection.
func (c *Ctx) EnableFullDuplex() error { return c.rc.EnableFullDuplex() }

// --- Internal helpers ---

func (c *Ctx) writeHeaderNow(code int, contentType string) {
	if code > 0 {
		c.status = code
	}
	if c.wroteHeader {
		return
	}
	if contentType != "" {
		c.Header().Set("Content-Type", contentType)
	}
	c.writer.WriteHeader(c.status)
	c.wroteHeader = true
}

func sanitizeToken(s string) string {
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
