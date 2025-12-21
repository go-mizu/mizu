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
	"os"
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

// NewCtx returns new Ctx.
func NewCtx(w http.ResponseWriter, r *http.Request, lg *slog.Logger) *Ctx {
	// We make it public, to make writing tests easier.
	return newCtx(w, r, lg)
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

// CtxFromRequest creates a Ctx from an http.Request.
// Used by packages that need to create a Ctx outside the normal router flow.
// If w is nil, a no-op writer is used.
func CtxFromRequest(w http.ResponseWriter, r *http.Request) *Ctx {
	if w == nil {
		w = &noopWriter{}
	}
	return newCtx(w, r, nil)
}

// noopWriter is a no-op http.ResponseWriter for cases where we don't need response.
type noopWriter struct{}

func (noopWriter) Header() http.Header         { return http.Header{} }
func (noopWriter) Write(b []byte) (int, error) { return len(b), nil }
func (noopWriter) WriteHeader(int)             {}

// --- Accessors ---

func (c *Ctx) Request() *http.Request                   { return c.request }
func (c *Ctx) Writer() http.ResponseWriter              { return c.writer }
func (c *Ctx) Header() http.Header                      { return c.writer.Header() }
func (c *Ctx) Context() context.Context                 { return c.request.Context() }
func (c *Ctx) Logger() *slog.Logger                     { return c.log }
func (c *Ctx) StatusCode() int                          { return c.status }
func (c *Ctx) Param(name string) string                 { return c.request.PathValue(name) }
func (c *Ctx) Cookie(name string) (*http.Cookie, error) { return c.request.Cookie(name) }

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

// MultipartForm parses multipart form data and returns a cleanup func.
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

// --- Request body binding ---

// BindJSON reads JSON into v with a max size limit.
// It disallows unknown fields and rejects trailing data.
func (c *Ctx) BindJSON(v any, max int64) error {
	r := c.request.Body
	if max > 0 {
		r = http.MaxBytesReader(c.writer, r, max)
	}

	dec := newJSONDecoder(r)
	decDisallowUnknownFields(dec)

	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Must be exactly one JSON value.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
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

func (c *Ctx) SetCookie(ck *http.Cookie) {
	if ck != nil {
		http.SetCookie(c.writer, ck)
	}
}

func (c *Ctx) JSON(code int, v any) error {
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		c.HeaderIfNone("Content-Type", "application/json; charset=utf-8")
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	enc := newJSONEncoder(c.writer)
	encSetEscapeHTML(enc, false)
	return enc.Encode(v)
}

func (c *Ctx) HTML(code int, html string) error {
	if code > 0 {
		c.status = code
	}
	if !c.wroteHeader {
		c.HeaderIfNone("Content-Type", "text/html; charset=utf-8")
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
		c.HeaderIfNone("Content-Type", "text/plain; charset=utf-8")
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	_, err := io.WriteString(c.writer, s)
	return err
}

// Bytes writes raw bytes with an optional content type.
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

func (c *Ctx) WriteString(s string) (int, error) {
	return c.Write([]byte(s))
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
	_ = c.Flush()
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
				_ = c.Flush()
				return nil
			}

			buf, err := jsonMarshal(v)
			if err != nil {
				buf = []byte(strconv.Quote(fmt.Sprint(v)))
			}

			// SSE requires each line to be prefixed with "data: ".
			payload := string(buf)
			payload = strings.ReplaceAll(payload, "\n", "\ndata: ")

			if _, err := io.WriteString(c.writer, "data: "+payload+"\n\n"); err != nil {
				return err
			}
			_ = c.Flush()

		case <-tick.C:
			_, _ = io.WriteString(c.writer, ": ping\n\n")
			_ = c.Flush()
		}
	}
}

// --- File helpers (simplified API) ---

func (c *Ctx) File(code int, path string) error {
	return c.serveFile(code, path, "")
}

func (c *Ctx) Download(code int, path, name string) error {
	if name == "" {
		name = filepath.Base(path)
	}
	return c.serveFile(code, path, name)
}

func (c *Ctx) serveFile(code int, path string, downloadName string) error {
	path, err := cleanFilePath(path)
	if err != nil {
		return err
	}

	f, fi, err := openFileForServe(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := c.applyDownloadHeaders(downloadName); err != nil {
		return err
	}
	c.applyContentType(path, downloadName)

	final := c.finalStatus(code)

	if final == http.StatusOK && !c.wroteHeader {
		http.ServeContent(c.writer, c.request, filepath.Base(path), fi.ModTime(), f)
		c.wroteHeader = true
		return nil
	}

	if err := c.writeHeaderNow(final, c.Header().Get("Content-Type")); err != nil {
		return err
	}
	_, err = io.Copy(c.writer, f)
	return err
}

func cleanFilePath(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if strings.ContainsRune(p, 0) {
		return "", errors.New("invalid path")
	}
	return filepath.Clean(p), nil
}

func openFileForServe(path string) (*os.File, os.FileInfo, error) {
	// This function is a file-serving primitive. The caller is expected to control
	// which paths are reachable (routing, auth, config). Static analyzers flag
	// variable paths by default, but it is intentional here.
	//
	//nolint:gosec // G304: serving user-chosen paths is the purpose of this helper
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	if fi.IsDir() {
		_ = f.Close()
		return nil, nil, errors.New("path is a directory")
	}
	return f, fi, nil
}

func (c *Ctx) applyDownloadHeaders(name string) error {
	if name == "" {
		return nil
	}
	ascii := sanitizeToken(name)
	disp := fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(name))
	c.Header().Set("Content-Disposition", disp)
	return nil
}

func (c *Ctx) applyContentType(path, name string) {
	if c.Header().Get("Content-Type") != "" {
		return
	}

	ext := filepath.Ext(path)
	if name != "" {
		if e := filepath.Ext(name); e != "" {
			ext = e
		}
	}

	if ext != "" {
		if guess := mime.TypeByExtension(ext); guess != "" {
			c.Header().Set("Content-Type", guess)
			return
		}
	}
	c.Header().Set("Content-Type", "application/octet-stream")
}

func (c *Ctx) finalStatus(code int) int {
	if code > 0 {
		return code
	}
	if c.status > 0 {
		return c.status
	}
	return http.StatusOK
}

// --- Low-level passthroughs ---

func (c *Ctx) Flush() error {
	if !c.wroteHeader {
		c.writer.WriteHeader(c.status)
		c.wroteHeader = true
	}
	if f, ok := c.writer.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

func (c *Ctx) SetWriter(w http.ResponseWriter) {
	c.writer = w
	c.rc = http.NewResponseController(w)
}

func (c *Ctx) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := c.writer.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijack not supported")
}

func (c *Ctx) SetWriteDeadline(t time.Time) error { return c.rc.SetWriteDeadline(t) }
func (c *Ctx) EnableFullDuplex() error            { return c.rc.EnableFullDuplex() }

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
	c.writer.WriteHeader(c.status)
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
