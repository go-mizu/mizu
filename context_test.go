package mizu

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

type flusherRecorder struct {
	hdr         http.Header
	code        int
	buf         bytes.Buffer
	wroteHeader bool
	flushed     bool
}

func newFlusherRecorder() *flusherRecorder {
	return &flusherRecorder{hdr: make(http.Header)}
}

func (r *flusherRecorder) Header() http.Header { return r.hdr }
func (r *flusherRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.code = code
	r.wroteHeader = true
}
func (r *flusherRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.buf.Write(p)
}
func (r *flusherRecorder) Flush() { r.flushed = true }

type deadlineDuplexRW struct {
	*flusherRecorder
	setDeadlineCalled bool
	fullDuplexCalled  bool
}

func newDeadlineDuplexRW() *deadlineDuplexRW {
	return &deadlineDuplexRW{flusherRecorder: newFlusherRecorder()}
}

func (w *deadlineDuplexRW) SetWriteDeadline(t time.Time) error {
	w.setDeadlineCalled = true
	return nil
}
func (w *deadlineDuplexRW) EnableFullDuplex() error {
	w.fullDuplexCalled = true
	return nil
}

type hijackRW struct {
	*flusherRecorder
	hijacked bool
	conn     net.Conn
	brw      *bufio.ReadWriter
}

func newHijackRW() *hijackRW {
	pr, pw := net.Pipe()
	return &hijackRW{
		flusherRecorder: newFlusherRecorder(),
		conn:            pr,
		brw:             bufio.NewReadWriter(bufio.NewReader(pr), bufio.NewWriter(pw)),
	}
}

func (h *hijackRW) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return h.conn, h.brw, nil
}

func newReq(method, target string, body io.Reader) *http.Request {
	if body == nil {
		body = http.NoBody
	}
	r := httptest.NewRequest(method, target, body)
	return r
}

func newTestCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	return newCtx(w, r, nil)
}

func TestCtx_AccessorsAndNilURL(t *testing.T) {
	w := newFlusherRecorder()
	r := newReq(http.MethodGet, "http://example.com/x?x=1", nil)
	c := newTestCtx(w, r)

	if c.Request() != r {
		t.Fatalf("Request mismatch")
	}
	// Writer may be wrapped for status capture, so verify it works correctly
	if c.Writer() == nil {
		t.Fatalf("Writer is nil")
	}
	// Verify writes go to the underlying recorder
	c.Writer().Write([]byte("test"))
	if !strings.Contains(w.buf.String(), "test") {
		t.Fatalf("Writer does not delegate to underlying writer")
	}
	if c.Header() == nil {
		t.Fatalf("Header is nil")
	}
	if c.Context() == nil {
		t.Fatalf("Context is nil")
	}
	if c.Logger() == nil {
		t.Fatalf("Logger is nil")
	}

	// Nil URL path.
	r2 := newReq(http.MethodGet, "http://example.com/", nil)
	r2.URL = nil
	c2 := newTestCtx(newFlusherRecorder(), r2)

	if got := c2.Query("k"); got != "" {
		t.Fatalf("Query expected empty, got %q", got)
	}
	vals := c2.QueryValues()
	if vals == nil || len(vals) != 0 {
		t.Fatalf("QueryValues expected empty map, got %#v", vals)
	}
}

func TestCtx_Form_ParseError(t *testing.T) {
	// Invalid percent encoding should make ParseForm error.
	body := strings.NewReader("a=%ZZ")
	r := newReq(http.MethodPost, "http://example.com/form", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c := newTestCtx(newFlusherRecorder(), r)

	_, err := c.Form()
	if err == nil {
		t.Fatalf("expected ParseForm error")
	}
}

func TestCtx_MultipartForm_SuccessAndCleanup(t *testing.T) {
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, err := mw.CreateFormFile("file", "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("hello"))
	_ = mw.WriteField("x", "1")
	_ = mw.Close()

	r := newReq(http.MethodPost, "http://example.com/upload", &b)
	r.Header.Set("Content-Type", mw.FormDataContentType())

	c := newTestCtx(newFlusherRecorder(), r)

	form, cleanup, err := c.MultipartForm(1 << 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if form == nil {
		t.Fatalf("expected form")
	}
	defer cleanup()

	if got := form.Value["x"]; len(got) != 1 || got[0] != "1" {
		t.Fatalf("expected field x=1, got %#v", got)
	}
	if _, ok := form.File["file"]; !ok {
		t.Fatalf("expected file part")
	}
}

func TestCtx_MultipartForm_ParseError(t *testing.T) {
	r := newReq(http.MethodPost, "http://example.com/upload", strings.NewReader("not-multipart"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	c := newTestCtx(newFlusherRecorder(), r)

	_, cleanup, err := c.MultipartForm(1 << 20)
	if err == nil {
		t.Fatalf("expected error")
	}
	// cleanup must be safe even on error
	cleanup()
}

func TestCtx_BindJSON_AllCases(t *testing.T) {
	type payload struct {
		A string `json:"a"`
	}

	t.Run("ok", func(t *testing.T) {
		r := newReq(http.MethodPost, "http://example.com/", strings.NewReader(`{"a":"x"}`))
		c := newTestCtx(newFlusherRecorder(), r)
		var p payload
		if err := c.BindJSON(&p, 0); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if p.A != "x" {
			t.Fatalf("expected x")
		}
	})

	t.Run("unknown_field", func(t *testing.T) {
		r := newReq(http.MethodPost, "http://example.com/", strings.NewReader(`{"a":"x","b":1}`))
		c := newTestCtx(newFlusherRecorder(), r)
		var p payload
		if err := c.BindJSON(&p, 0); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("trailing_data", func(t *testing.T) {
		r := newReq(http.MethodPost, "http://example.com/", strings.NewReader(`{"a":"x"} {"a":"y"}`))
		c := newTestCtx(newFlusherRecorder(), r)
		var p payload
		if err := c.BindJSON(&p, 0); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("invalid_json", func(t *testing.T) {
		r := newReq(http.MethodPost, "http://example.com/", strings.NewReader(`{"a":`))
		c := newTestCtx(newFlusherRecorder(), r)
		var p payload
		if err := c.BindJSON(&p, 0); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("max_bytes", func(t *testing.T) {
		// 20 bytes JSON, set max smaller.
		r := newReq(http.MethodPost, "http://example.com/", strings.NewReader(`{"a":"0123456789"}`))
		c := newTestCtx(newFlusherRecorder(), r)
		var p payload
		if err := c.BindJSON(&p, 5); err == nil {
			t.Fatalf("expected body too large error")
		}
	})
}

func TestCtx_StatusHeadersAndWrites(t *testing.T) {
	w := newFlusherRecorder()
	r := newReq(http.MethodGet, "http://example.com/", nil)
	c := newTestCtx(w, r)

	c.Status(201)
	if c.StatusCode() != 201 {
		t.Fatalf("expected status 201")
	}

	// HeaderIfNone
	c.Header().Set("X", "1")
	c.HeaderIfNone("X", "2")
	if c.Header().Get("X") != "1" {
		t.Fatalf("HeaderIfNone overwrote existing")
	}
	c.HeaderIfNone("Y", "2")
	if c.Header().Get("Y") != "2" {
		t.Fatalf("HeaderIfNone did not set missing")
	}

	// Write triggers header once.
	if _, err := c.Write([]byte("a")); err != nil {
		t.Fatal(err)
	}
	if w.code != 201 {
		t.Fatalf("expected written status 201, got %d", w.code)
	}

	// Status after wroteHeader should not change already sent code.
	c.Status(418)
	_, _ = c.WriteString("b")
	if w.code != 201 {
		t.Fatalf("expected status unchanged, got %d", w.code)
	}
}

// TestCtx_DirectWriteHeaderCapture verifies that status codes are captured
// when WriteHeader is called directly on the writer (bypassing Ctx helpers).
// This is important for handlers that use http.ServeContent or similar functions
// that write status codes like 206 Partial Content directly.
func TestCtx_DirectWriteHeaderCapture(t *testing.T) {
	t.Run("captures 206 Partial Content", func(t *testing.T) {
		w := newFlusherRecorder()
		c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

		// Simulate what http.ServeContent does for Range requests
		c.Writer().WriteHeader(206) // Direct call, bypasses Ctx helpers
		c.Writer().Write([]byte("partial"))

		// Verify the status was captured in the context
		if c.StatusCode() != 206 {
			t.Fatalf("expected StatusCode() = 206, got %d", c.StatusCode())
		}

		// Verify the underlying writer also received 206
		if w.code != 206 {
			t.Fatalf("expected writer code = 206, got %d", w.code)
		}
	})

	t.Run("captures 201 Created", func(t *testing.T) {
		w := newFlusherRecorder()
		c := newTestCtx(w, newReq(http.MethodPost, "http://example.com/", nil))

		c.Writer().WriteHeader(201)
		c.Writer().Write([]byte("created"))

		if c.StatusCode() != 201 {
			t.Fatalf("expected StatusCode() = 201, got %d", c.StatusCode())
		}
	})

	t.Run("defaults to 200 when Write called without WriteHeader", func(t *testing.T) {
		w := newFlusherRecorder()
		c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

		c.Writer().Write([]byte("content"))

		// Per HTTP spec, Write without WriteHeader implies 200
		if c.StatusCode() != 200 {
			t.Fatalf("expected StatusCode() = 200, got %d", c.StatusCode())
		}
	})
}

func TestCtx_JSON_SetsContentType(t *testing.T) {
	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

	if err := c.JSON(0, map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected json content-type, got %q", w.Header().Get("Content-Type"))
	}
}

func TestCtx_HTML_SetsContentTypeAndStatus(t *testing.T) {
	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

	if err := c.HTML(202, "<p>x</p>"); err != nil {
		t.Fatal(err)
	}
	if w.code != 202 {
		t.Fatalf("expected 202, got %d", w.code)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected html content-type, got %q", w.Header().Get("Content-Type"))
	}
}

func TestCtx_Text_UTF8_SetsTextPlain(t *testing.T) {
	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

	if err := c.Text(0, "hello"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("expected text/plain, got %q", w.Header().Get("Content-Type"))
	}
}

func TestCtx_Text_InvalidUTF8_FallsBackToOctetStream(t *testing.T) {
	b := []byte{0xff, 0xfe, 0xfd}
	if utf8.Valid(b) {
		t.Fatalf("expected invalid utf8 bytes")
	}

	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

	if err := c.Text(0, string(b)); err != nil {
		t.Fatal(err)
	}
	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("expected octet-stream, got %q", w.Header().Get("Content-Type"))
	}
}

func TestCtx_Bytes_DefaultContentType(t *testing.T) {
	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

	if err := c.Bytes(0, []byte{1, 2, 3}, ""); err != nil {
		t.Fatal(err)
	}
	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("expected octet-stream, got %q", w.Header().Get("Content-Type"))
	}
}

func TestCtx_Bytes_ExplicitContentType(t *testing.T) {
	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

	if err := c.Bytes(0, []byte("x"), "text/x-test"); err != nil {
		t.Fatal(err)
	}
	if w.Header().Get("Content-Type") != "text/x-test" {
		t.Fatalf("expected text/x-test, got %q", w.Header().Get("Content-Type"))
	}
}

func TestCtx_File_OpenError(t *testing.T) {
	dir := t.TempDir()

	c := newTestCtx(newFlusherRecorder(), newReq(http.MethodGet, "http://example.com/", nil))
	err := c.File(0, filepath.Join(dir, "missing.txt"))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCtx_File_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "d")
	if err := os.MkdirAll(subdir, 0o700); err != nil {
		t.Fatal(err)
	}

	c := newTestCtx(newFlusherRecorder(), newReq(http.MethodGet, "http://example.com/", nil))
	err := c.File(0, subdir)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCtx_File_200_UsesServeContent(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(fp, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	w := newFlusherRecorder()
	r := newReq(http.MethodGet, "http://example.com/hello.txt", nil)
	c := newTestCtx(w, r)

	if err := c.File(0, fp); err != nil {
		t.Fatal(err)
	}
	if !c.wroteHeader {
		t.Fatalf("expected wroteHeader true")
	}
	if w.buf.String() != "hello" {
		t.Fatalf("expected body hello, got %q", w.buf.String())
	}
}

func TestCtx_File_Non200_DeterministicCopy(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(fp, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	w := newFlusherRecorder()
	r := newReq(http.MethodGet, "http://example.com/hello.txt", nil)
	c := newTestCtx(w, r)

	if err := c.File(206, fp); err != nil {
		t.Fatal(err)
	}
	if w.code != 206 {
		t.Fatalf("expected 206, got %d", w.code)
	}
	if w.buf.String() != "hello" {
		t.Fatalf("expected body hello, got %q", w.buf.String())
	}
}

func TestCtx_Download_SetsDispositionAndContentTypeFromNameExt(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(fp, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	w := newFlusherRecorder()
	r := newReq(http.MethodGet, "http://example.com/dl", nil)
	c := newTestCtx(w, r)

	if err := c.Download(0, fp, "x.json"); err != nil {
		t.Fatal(err)
	}

	if disp := w.Header().Get("Content-Disposition"); !strings.Contains(disp, "attachment") {
		t.Fatalf("expected attachment, got %q", disp)
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Fatalf("expected content-type set")
	}
}

func TestCtx_Download_RespectsExistingContentType(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(fp, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	w := newFlusherRecorder()
	r := newReq(http.MethodGet, "http://example.com/dl", nil)
	c := newTestCtx(w, r)

	c.Header().Set("Content-Type", "text/x-custom")
	if err := c.Download(0, fp, "x.bin"); err != nil {
		t.Fatal(err)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/x-custom" {
		t.Fatalf("expected preserved ct, got %q", ct)
	}
}

func TestCtx_NoContent_Redirect(t *testing.T) {
	t.Run("no_content", func(t *testing.T) {
		w := newFlusherRecorder()
		c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))
		if err := c.NoContent(); err != nil {
			t.Fatal(err)
		}
		if w.code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.code)
		}
	})

	t.Run("redirect_default_302", func(t *testing.T) {
		w := newFlusherRecorder()
		c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))
		if err := c.Redirect(0, "/x"); err != nil {
			t.Fatal(err)
		}
		if w.code != http.StatusFound {
			t.Fatalf("expected 302, got %d", w.code)
		}
		if w.Header().Get("Location") != "/x" {
			t.Fatalf("expected Location /x")
		}
	})
}

func TestCtx_Stream(t *testing.T) {
	t.Run("stream_ok_flushes", func(t *testing.T) {
		w := newFlusherRecorder()
		c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))
		if err := c.Stream(func(w io.Writer) error {
			_, _ = w.Write([]byte("x"))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if !w.flushed {
			t.Fatalf("expected flushed")
		}
	})

	t.Run("stream_error_propagates", func(t *testing.T) {
		w := newFlusherRecorder()
		c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))
		want := errors.New("boom")
		if err := c.Stream(func(io.Writer) error { return want }); !errors.Is(err, want) {
			t.Fatalf("expected %v, got %v", want, err)
		}
	})
}

type basicRW struct {
	hdr  http.Header
	code int
	buf  bytes.Buffer
}

func newBasicRW() *basicRW { return &basicRW{hdr: make(http.Header)} }

func (w *basicRW) Header() http.Header  { return w.hdr }
func (w *basicRW) WriteHeader(code int) { w.code = code }
func (w *basicRW) Write(p []byte) (int, error) {
	if w.code == 0 {
		w.code = http.StatusOK
	}
	return w.buf.Write(p)
}

func TestCtx_SSE_AllPaths(t *testing.T) {
	t.Run("requires_flusher", func(t *testing.T) {
		w := newBasicRW() // does NOT implement http.Flusher
		c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

		ch := make(chan any)
		close(ch)

		if err := c.SSE(ch); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("writes_data_and_end", func(t *testing.T) {
		w := newFlusherRecorder()
		r := newReq(http.MethodGet, "http://example.com/", nil)
		c := newTestCtx(w, r)

		ch := make(chan any, 1)
		ch <- map[string]any{"a": "b"}
		close(ch)

		if err := c.SSE(ch); err != nil {
			t.Fatalf("unexpected: %v", err)
		}

		out := w.buf.String()
		if !strings.Contains(out, "data:") {
			t.Fatalf("expected data event, got %q", out)
		}
		if !strings.Contains(out, "event: end") {
			t.Fatalf("expected end event, got %q", out)
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
			t.Fatalf("expected sse content-type, got %q", ct)
		}
	})

	t.Run("multiline_data_is_prefixed", func(t *testing.T) {
		w := newFlusherRecorder()
		c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

		ch := make(chan any, 1)
		ch <- "a\nb"
		close(ch)

		if err := c.SSE(ch); err != nil {
			t.Fatal(err)
		}
		out := w.buf.String()
		if !strings.Contains(out, "data:") || !strings.Contains(out, "\ndata: ") {
			t.Fatalf("expected multiline prefixing, got %q", out)
		}
	})
}

func TestCtx_Flush_WritesHeaderIfNeeded(t *testing.T) {
	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))
	c.Status(207)
	if err := c.Flush(); err != nil {
		t.Fatal(err)
	}
	if w.code != 207 {
		t.Fatalf("expected 207, got %d", w.code)
	}
	if !w.flushed {
		t.Fatalf("expected flushed")
	}
}

func TestCtx_SetWriter_RebuildsResponseController(t *testing.T) {
	w1 := newDeadlineDuplexRW()
	c := newTestCtx(w1, newReq(http.MethodGet, "http://example.com/", nil))

	if err := c.SetWriteDeadline(time.Now()); err != nil {
		t.Fatal(err)
	}
	if !w1.setDeadlineCalled {
		t.Fatalf("expected deadline called on w1")
	}

	w2 := newDeadlineDuplexRW()
	c.SetWriter(w2)

	if err := c.EnableFullDuplex(); err != nil {
		t.Fatal(err)
	}
	if !w2.fullDuplexCalled {
		t.Fatalf("expected full duplex called on w2 after SetWriter")
	}
}

func TestCtx_Hijack(t *testing.T) {
	t.Run("unsupported", func(t *testing.T) {
		c := newTestCtx(newFlusherRecorder(), newReq(http.MethodGet, "http://example.com/", nil))
		_, _, err := c.Hijack()
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("supported", func(t *testing.T) {
		hw := newHijackRW()
		c := newTestCtx(hw, newReq(http.MethodGet, "http://example.com/", nil))
		conn, brw, err := c.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		if conn == nil || brw == nil {
			t.Fatalf("expected conn and brw")
		}
		if !hw.hijacked {
			t.Fatalf("expected hijacked flag")
		}
	})
}

func TestCtx_WriteHeaderNow_NoOpWhenAlreadyWritten(t *testing.T) {
	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))

	_ = c.writeHeaderNow(201, "text/plain")
	if w.code != 201 {
		t.Fatalf("expected 201")
	}
	// Second call should not overwrite.
	_ = c.writeHeaderNow(202, "text/html")
	if w.code != 201 {
		t.Fatalf("expected still 201, got %d", w.code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("expected original ct, got %q", ct)
	}
}

func TestCtx_QueryValues_MutableButIndependent(t *testing.T) {
	r := newReq(http.MethodGet, "http://example.com/?a=1&a=2", nil)
	c := newTestCtx(newFlusherRecorder(), r)

	v := c.QueryValues()
	if got := v["a"]; len(got) != 2 {
		t.Fatalf("expected 2 values, got %#v", got)
	}

	// URL.Query returns a copy, so changes here do not affect c.Query.
	v.Add("b", "3")
	if c.Query("b") != "" {
		t.Fatalf("expected b to remain empty, got %q", c.Query("b"))
	}
}

func TestCtx_Param(t *testing.T) {
	// PathValue requires patterns registered on a ServeMux.
	// Here we only cover that Param calls PathValue; in plain requests it returns "".
	r := newReq(http.MethodGet, "http://example.com/users/1", nil)
	c := newTestCtx(newFlusherRecorder(), r)
	if got := c.Param("id"); got != "" {
		t.Fatalf("expected empty without mux pattern, got %q", got)
	}
}

func TestCtx_SetCookie(t *testing.T) {
	w := newFlusherRecorder()
	c := newTestCtx(w, newReq(http.MethodGet, "http://example.com/", nil))
	c.SetCookie(nil)
	c.SetCookie(&http.Cookie{Name: "a", Value: "1"})
	if sc := w.Header().Values("Set-Cookie"); len(sc) != 1 {
		t.Fatalf("expected 1 Set-Cookie, got %#v", sc)
	}
}

func TestSanitizeToken(t *testing.T) {
	got := sanitizeToken("a b\tc;d")
	if strings.ContainsAny(got, " \t;") {
		t.Fatalf("expected sanitized, got %q", got)
	}
}

func TestSSE_ContextCancelPath(t *testing.T) {
	w := newFlusherRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := httptest.NewRequest(http.MethodGet, "http://example.com/", nil).WithContext(ctx)
	c := newTestCtx(w, r)

	ch := make(chan any)
	// keep channel open so context path wins
	var once sync.Once
	go func() {
		once.Do(func() { time.Sleep(10 * time.Millisecond) })
		close(ch)
	}()

	err := c.SSE(ch)
	if err == nil {
		t.Fatalf("expected context error")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected canceled/deadline, got %v", err)
	}
}

func TestCtx_Query_EmptyWhenURLNil(t *testing.T) {
	r := newReq(http.MethodGet, "http://example.com/", nil)
	r.URL = (*url.URL)(nil)
	c := newTestCtx(newFlusherRecorder(), r)
	if c.Query("x") != "" {
		t.Fatalf("expected empty")
	}
}
