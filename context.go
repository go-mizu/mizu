package mizu

import (
	"bufio"
	"context"
	"encoding/json"
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

// Handler is the function signature for request handlers.
type Handler func(*Ctx) error

// Ctx wraps request and response with small helpers and is not safe for concurrent use.
type Ctx struct {
	req         *http.Request
	res         http.ResponseWriter
	rc          *http.ResponseController
	status      int
	wroteHeader bool
	log         *slog.Logger
}

// newCtx builds a Ctx from net/http types.
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

// Request returns the underlying *http.Request.
func (c *Ctx) Request() *http.Request { return c.req }

// Response returns the underlying http.ResponseWriter.
func (c *Ctx) Response() http.ResponseWriter { return c.res }

// Header returns the response headers.
func (c *Ctx) Header() http.Header { return c.res.Header() }

// Context returns the request context.
func (c *Ctx) Context() context.Context { return c.req.Context() }

// Logger returns the request scoped logger.
func (c *Ctx) Logger() *slog.Logger { return c.log }

// --- Request helpers ---

// Param gets a path parameter captured by the Go 1.22 router.
func (c *Ctx) Param(name string) string { return c.req.PathValue(name) }

// Query gets the first query value for a key.
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

// Form parses and returns form values from URL encoded or multipart forms.
func (c *Ctx) Form() (url.Values, error) {
	if err := c.req.ParseForm(); err != nil {
		return nil, err
	}
	return c.req.Form, nil
}

// MultipartForm parses multipart form data and returns a cleanup function.
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

// ClientIP returns the best effort client IP.
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

// BindJSON reads JSON into v with size limit and rejects unknown fields and trailing data.
func (c *Ctx) BindJSON(v any, max int64) error {
	r := c.req.Body
	if max > 0 {
		r = http.MaxBytesReader(c.res, r, max)
	}
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if dec.More() {
		return errors.New("invalid JSON: trailing data")
	}
	return nil
}

// --- Response helpers ---

// Status sets the response status code to apply on first write.
func (c *Ctx) Status(code int) {
	if code > 0 {
		c.status = code
	}
}

// StatusCode returns the current status code with default 200.
func (c *Ctx) StatusCode() int { return c.status }

// HeaderIfNone sets a header only if it is not already present.
func (c *Ctx) HeaderIfNone(key, value string) {
	if c.Header().Get(key) == "" {
		c.Header().Set(key, value)
	}
}

// NoContent sends 204 No Content.
func (c *Ctx) NoContent() error {
	return c.writeHeaderNow(http.StatusNoContent, "")
}

// Redirect sends a redirect with Location and optional status.
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

// JSON writes JSON with optional status and sets Content-Type.
func (c *Ctx) JSON(code int, v any) error {
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		c.HeaderIfNone("Content-Type", "application/json; charset=utf-8")
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	enc := json.NewEncoder(c.res)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// HTML writes UTF-8 HTML with optional status.
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

// Text writes UTF-8 text with optional status and falls back to octet stream for invalid UTF-8.
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

// Bytes writes raw bytes with optional status and content type.
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

// Write writes bytes and ensures headers are sent once.
func (c *Ctx) Write(b []byte) (int, error) {
	if !c.wroteHeader {
		c.res.WriteHeader(c.status)
		c.wroteHeader = true
	}
	return c.res.Write(b)
}

// WriteString writes a string and ensures headers are sent once.
func (c *Ctx) WriteString(s string) (int, error) {
	return c.Write([]byte(s))
}

// File serves a file from disk and respects any preset status.
func (c *Ctx) File(path string) error {
	return c.fileWithCode(0, path)
}

// Download serves a file as an attachment with RFC 5987 filename support.
func (c *Ctx) Download(path, name string) error {
	return c.downloadWithCode(0, path, name)
}

// FileCode serves a file with an explicit status code.
func (c *Ctx) FileCode(code int, path string) error {
	return c.fileWithCode(code, path)
}

// DownloadCode serves an attachment with an explicit status code.
func (c *Ctx) DownloadCode(code int, path, name string) error {
	return c.downloadWithCode(code, path, name)
}

func (c *Ctx) fileWithCode(code int, path string) error {
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

// Stream writes streamed output and flushes at the end.
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

// SSE sends Server Sent Events with keepalive pings and returns on disconnect.
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
			buf, err := json.Marshal(v)
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

// Flush flushes buffered data to the client if supported.
func (c *Ctx) Flush() error {
	if f, ok := c.res.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

// Hijack takes over the connection if supported.
func (c *Ctx) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := c.res.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijack not supported")
}

// SetWriteDeadline sets the write deadline via ResponseController.
func (c *Ctx) SetWriteDeadline(t time.Time) error { return c.rc.SetWriteDeadline(t) }

// EnableFullDuplex allows concurrent read and write via ResponseController.
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
