// File: context_test.go
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
)

func TestCtx_AccessorsAndBasics(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/p?q=1", nil)

	c := newCtx(rr, req, nil)
	if c.Request() != req {
		t.Fatalf("Request() mismatch")
	}
	if c.Writer() != rr {
		t.Fatalf("Writer() mismatch")
	}
	if c.Header() == nil {
		t.Fatalf("Header() is nil")
	}
	if c.Context() == nil {
		t.Fatalf("Context() is nil")
	}
	if c.Logger() == nil {
		t.Fatalf("Logger() is nil")
	}

	// Status defaults.
	if got := c.StatusCode(); got != http.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}

	c.Status(201)
	if got := c.StatusCode(); got != 201 {
		t.Fatalf("want 201, got %d", got)
	}
}

func TestCtx_Param(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req.SetPathValue("id", "123")

	c := newCtx(rr, req, nil)
	if got := c.Param("id"); got != "123" {
		t.Fatalf("want 123, got %q", got)
	}
}

func TestCtx_QueryAndQueryValues(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/p?a=1&a=2&b=x", nil)

	c := newCtx(rr, req, nil)
	if got := c.Query("a"); got != "1" {
		t.Fatalf("want 1, got %q", got)
	}
	vals := c.QueryValues()
	if vals.Get("b") != "x" {
		t.Fatalf("want b=x, got %q", vals.Get("b"))
	}
	if got := vals["a"]; len(got) != 2 {
		t.Fatalf("want 2 values for a, got %v", got)
	}

	// nil URL case
	req2 := &http.Request{Method: http.MethodGet}
	c2 := newCtx(rr, req2, nil)
	if got := c2.Query("a"); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
	if got := c2.QueryValues(); got == nil || len(got) != 0 {
		t.Fatalf("want empty values, got %v", got)
	}
}

func TestCtx_Form(t *testing.T) {
	rr := httptest.NewRecorder()
	body := strings.NewReader("a=1&b=hello")
	req := httptest.NewRequest(http.MethodPost, "/form", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c := newCtx(rr, req, nil)
	v, err := c.Form()
	if err != nil {
		t.Fatalf("Form() err: %v", err)
	}
	if v.Get("a") != "1" || v.Get("b") != "hello" {
		t.Fatalf("unexpected form values: %v", v)
	}
}

func TestCtx_MultipartFormAndCleanup(t *testing.T) {
	rr := httptest.NewRecorder()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	_ = mw.WriteField("k", "v")
	fw, err := mw.CreateFormFile("file", "a.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, _ = fw.Write([]byte("hi"))
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/mp", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	c := newCtx(rr, req, nil)
	form, cleanup, err := c.MultipartForm(32 << 20)
	if err != nil {
		t.Fatalf("MultipartForm: %v", err)
	}
	defer cleanup()

	if form.Value["k"][0] != "v" {
		t.Fatalf("want k=v, got %v", form.Value)
	}
	if len(form.File["file"]) != 1 {
		t.Fatalf("want 1 file, got %d", len(form.File["file"]))
	}
}

func TestCtx_Cookie(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "a", Value: "1"})

	c := newCtx(rr, req, nil)
	ck, err := c.Cookie("a")
	if err != nil {
		t.Fatalf("Cookie err: %v", err)
	}
	if ck.Value != "1" {
		t.Fatalf("want 1, got %q", ck.Value)
	}
}

func TestCtx_SetCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(rr, req, nil)

	c.SetCookie(&http.Cookie{Name: "sid", Value: "x"})
	h := rr.Header().Values("Set-Cookie")
	if len(h) == 0 {
		t.Fatalf("expected Set-Cookie header")
	}
}

func TestCtx_Bind_JSON_OK_Unknown_Trailing(t *testing.T) {
	type payload struct {
		A string `json:"a"`
	}

	t.Run("ok", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"x"}`))
		c := newCtx(rr, req, nil)

		var p payload
		if err := c.Bind(&p, 0); err != nil {
			t.Fatalf("Bin err: %v", err)
		}
		if p.A != "x" {
			t.Fatalf("want x, got %q", p.A)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"x","b":1}`))
		c := newCtx(rr, req, nil)

		var p payload
		if err := c.Bind(&p, 0); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("trailing data", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"x"} {"a":"y"}`))
		c := newCtx(rr, req, nil)

		var p payload
		if err := c.Bind(&p, 0); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("max bytes", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"toolong"}`))
		c := newCtx(rr, req, nil)

		var p payload
		if err := c.Bind(&p, 5); err == nil {
			t.Fatalf("expected error due to size limit")
		}
	})
}

func TestCtx_NoContentAndRedirect(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(rr, req, nil)

	if err := c.NoContent(); err != nil {
		t.Fatalf("NoContent err: %v", err)
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	c2 := newCtx(rr2, req2, nil)

	if err := c2.Redirect(0, "/x"); err != nil {
		t.Fatalf("Redirect err: %v", err)
	}
	if rr2.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rr2.Code)
	}
	if rr2.Header().Get("Location") != "/x" {
		t.Fatalf("want Location /x, got %q", rr2.Header().Get("Location"))
	}
}

func TestCtx_JSON_HTML_Text_Bytes_Write_WriteString(t *testing.T) {
	t.Run("JSON sets content-type if absent", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := newCtx(rr, req, nil)

		if err := c.JSON(200, map[string]any{"a": 1}); err != nil {
			t.Fatalf("JSON err: %v", err)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("unexpected content-type: %q", ct)
		}
	})

	t.Run("HTML sets content-type if absent", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := newCtx(rr, req, nil)

		if err := c.HTML(200, "<h1>x</h1>"); err != nil {
			t.Fatalf("HTML err: %v", err)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Fatalf("unexpected content-type: %q", ct)
		}
	})

	t.Run("Text utf8", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := newCtx(rr, req, nil)

		if err := c.Text(200, "hello"); err != nil {
			t.Fatalf("Text err: %v", err)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
			t.Fatalf("unexpected content-type: %q", ct)
		}
	})

	t.Run("Text invalid utf8 becomes octet-stream", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := newCtx(rr, req, nil)

		s := string([]byte{0xff})
		if err := c.Text(200, s); err != nil {
			t.Fatalf("Text err: %v", err)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/octet-stream" {
			t.Fatalf("unexpected content-type: %q", ct)
		}
	})

	t.Run("Bytes default content-type", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := newCtx(rr, req, nil)

		if err := c.Bytes(200, []byte("x"), ""); err != nil {
			t.Fatalf("Bytes err: %v", err)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/octet-stream" {
			t.Fatalf("unexpected content-type: %q", ct)
		}
	})

	t.Run("Write and WriteString honor Status()", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := newCtx(rr, req, nil)

		c.Status(201)
		_, _ = c.Write([]byte("a"))
		if rr.Code != 201 {
			t.Fatalf("want 201, got %d", rr.Code)
		}

		rr2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		c2 := newCtx(rr2, req2, nil)
		c2.Status(202)
		_, _ = c2.WriteString("b")
		if rr2.Code != 202 {
			t.Fatalf("want 202, got %d", rr2.Code)
		}
	})
}

func TestCtx_FileAndDownload(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(fp, []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Run("File uses ctx status when code=0", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/a.txt", nil)
		c := newCtx(rr, req, nil)

		c.Status(201)
		if err := c.File(0, fp); err != nil {
			t.Fatalf("File err: %v", err)
		}
		if rr.Code != 201 {
			t.Fatalf("want 201, got %d", rr.Code)
		}
		if rr.Body.String() != "hi" {
			t.Fatalf("want body hi, got %q", rr.Body.String())
		}
	})

	t.Run("File overrides status when code!=0", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/a.txt", nil)
		c := newCtx(rr, req, nil)

		c.Status(200)
		if err := c.File(206, fp); err != nil {
			t.Fatalf("File err: %v", err)
		}
		if rr.Code != 206 {
			t.Fatalf("want 206, got %d", rr.Code)
		}
	})

	t.Run("Download sets disposition and status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/d", nil)
		c := newCtx(rr, req, nil)

		if err := c.Download(202, fp, "hello.txt"); err != nil {
			t.Fatalf("Download err: %v", err)
		}
		if rr.Code != 202 {
			t.Fatalf("want 202, got %d", rr.Code)
		}
		if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
			t.Fatalf("missing attachment disposition: %q", cd)
		}
	})
}

func TestCtx_Stream(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(rr, req, nil)

	err := c.Stream(func(w io.Writer) error {
		_, _ = io.WriteString(w, "x")
		return nil
	})
	if err != nil {
		t.Fatalf("Stream err: %v", err)
	}
	if rr.Body.String() != "x" {
		t.Fatalf("want x, got %q", rr.Body.String())
	}
	if rr.Header().Get("Content-Type") == "" {
		t.Fatalf("expected content-type to be set")
	}
}

type noFlushWriter struct {
	h http.Header
	b bytes.Buffer
}

func (w *noFlushWriter) Header() http.Header { return w.h }
func (w *noFlushWriter) Write(p []byte) (int, error) {
	if w.h == nil {
		w.h = make(http.Header)
	}
	return w.b.Write(p)
}
func (w *noFlushWriter) WriteHeader(int) {}

func TestCtx_SSE_NoFlusherErrors(t *testing.T) {
	w := &noFlushWriter{h: make(http.Header)}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(w, req, nil)

	ch := make(chan any)
	defer close(ch)

	if err := c.SSE(ch); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCtx_SSE_WritesAndEnds(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(rr, req, nil)

	ch := make(chan any, 1)
	ch <- map[string]any{"a": 1}
	close(ch)

	err := c.SSE(ch)
	if err != nil {
		t.Fatalf("SSE err: %v", err)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "data:") {
		t.Fatalf("expected data in body, got %q", body)
	}
	if !strings.Contains(body, "event: end") {
		t.Fatalf("expected end event, got %q", body)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("unexpected content-type: %q", ct)
	}
}

func TestCtx_Flush_NoPanic(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(rr, req, nil)

	c.Flush() // recorder implements Flusher
}

func TestCtx_SetWriter_RebuildsResponseController(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(rr, req, nil)

	rr2 := httptest.NewRecorder()
	c.SetWriter(rr2)

	if c.Writer() != rr2 {
		t.Fatalf("writer not updated")
	}
	if c.rc == nil {
		t.Fatalf("response controller is nil")
	}

	// ResponseController methods may return ErrNotSupported; assert no panic.
	_ = c.SetWriteDeadline(time.Now().Add(1 * time.Second))
	_ = c.EnableFullDuplex()
}

type hijackRW struct {
	rr *httptest.ResponseRecorder
}

func (h *hijackRW) Header() http.Header         { return h.rr.Header() }
func (h *hijackRW) Write(p []byte) (int, error) { return h.rr.Write(p) }
func (h *hijackRW) WriteHeader(code int)        { h.rr.WriteHeader(code) }

// Implement Flusher to keep behavior similar.
func (h *hijackRW) Flush() {}

// Implement Hijacker.
func (h *hijackRW) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c1, c2 := net.Pipe()
	// Close peer side when test ends.
	go func() { _ = c2.Close() }()
	rw := bufio.NewReadWriter(bufio.NewReader(c1), bufio.NewWriter(c1))
	return c1, rw, nil
}

func TestCtx_Hijack_SupportedAndUnsupported(t *testing.T) {
	t.Run("unsupported", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := newCtx(rr, req, nil)

		_, _, err := c.Hijack()
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("supported", func(t *testing.T) {
		rr := httptest.NewRecorder()
		hw := &hijackRW{rr: rr}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := newCtx(hw, req, nil)

		conn, rw, err := c.Hijack()
		if err != nil {
			t.Fatalf("Hijack err: %v", err)
		}
		if conn == nil || rw == nil {
			t.Fatalf("expected conn and rw")
		}
		_ = conn.Close()
	})
}

func TestCtx_WriteHeaderOnceBehavior(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(rr, req, nil)

	// First write locks status.
	c.Status(201)
	_, _ = c.WriteString("a")

	// Changing status later should not affect recorder.
	c.Status(202)
	_, _ = c.WriteString("b")

	if rr.Code != 201 {
		t.Fatalf("want 201, got %d", rr.Code)
	}
	if rr.Body.String() != "ab" {
		t.Fatalf("want ab, got %q", rr.Body.String())
	}
}

func TestCtx_Stream_ErrorPropagates(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(rr, req, nil)

	want := errors.New("x")
	if err := c.Stream(func(io.Writer) error { return want }); !errors.Is(err, want) {
		t.Fatalf("want %v, got %v", want, err)
	}
}

func TestCtx_SSE_ContextCancel(t *testing.T) {
	rr := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	c := newCtx(rr, req, nil)

	ch := make(chan any)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		_ = c.SSE(ch)
	}()

	cancel()
	close(ch)
	wg.Wait()
}

func TestCtx_QueryValuesNilURLAndFormNilURLValues(t *testing.T) {
	// Extra sanity: QueryValues returns empty Values when URL is nil.
	rr := httptest.NewRecorder()
	req := &http.Request{Method: http.MethodGet}
	c := newCtx(rr, req, nil)

	v := c.QueryValues()
	if v == nil {
		t.Fatalf("expected non-nil values")
	}
	if len(v) != 0 {
		t.Fatalf("expected empty values")
	}

	// Form parsing needs a URL in some cases, but ParseForm handles missing URL.
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url.Values{"a": {"1"}}.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c2 := newCtx(rr, req2, nil)
	f, err := c2.Form()
	if err != nil {
		t.Fatalf("Form err: %v", err)
	}
	if f.Get("a") != "1" {
		t.Fatalf("want a=1, got %v", f)
	}
}
