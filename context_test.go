// ctx_test.go
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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"log/slog"
)

type flusherRecorder struct {
	*httptest.ResponseRecorder
}

func (f flusherRecorder) Flush() {}

type hijackableRecorder struct {
	*httptest.ResponseRecorder
	hijacked bool
	pr       net.Conn
	pw       net.Conn
	rw       *bufio.ReadWriter
}

func newHijackableRecorder() *hijackableRecorder {
	pr, pw := net.Pipe()
	return &hijackableRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		pr:               pr,
		pw:               pw,
		rw:               bufio.NewReadWriter(bufio.NewReader(pr), bufio.NewWriter(pw)),
	}
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return h.pw, h.rw, nil
}

func newCtxRW(method, url string) (*Ctx, *httptest.ResponseRecorder, *http.Request) {
	r := httptest.NewRequest(method, url, nil)
	w := httptest.NewRecorder()
	return newCtx(w, r, nil), w, r
}

func TestAccessorsAndLogger(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test/a?b=1", nil)
	w := httptest.NewRecorder()
	lg := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := newCtx(w, r, lg)

	if c.Request() != r {
		t.Fatal("Request accessor mismatch")
	}
	if c.Writer() != w {
		t.Fatal("Writer accessor mismatch")
	}
	if c.Header() == nil {
		t.Fatal("Header should not be nil")
	}
	if c.Context() != r.Context() {
		t.Fatal("Context accessor mismatch")
	}
	if c.Logger() != lg {
		t.Fatal("custom logger not returned")
	}

	// nil logger should fall back to default
	c2 := newCtx(w, r, nil)
	if c2.Logger() == nil {
		t.Fatal("default logger should be non-nil")
	}
}

func TestParamQueryFormCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test/p?x=1&y=2&y=3", nil)
	r.SetPathValue("id", "42")
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)

	if got := c.Param("id"); got != "42" {
		t.Fatalf("Param got %q want %q", got, "42")
	}
	if got := c.Query("x"); got != "1" {
		t.Fatalf("Query got %q want %q", got, "1")
	}
	qs := c.QueryValues()
	if qs.Get("y") != "2" || len(qs["y"]) != 2 {
		t.Fatal("QueryValues should return all values")
	}

	// Form
	r2 := httptest.NewRequest(http.MethodPost, "http://x.test/form", strings.NewReader("a=1&b=2"))
	r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c2 := newCtx(httptest.NewRecorder(), r2, nil)
	form, err := c2.Form()
	if err != nil {
		t.Fatalf("Form error: %v", err)
	}
	if form.Get("a") != "1" || form.Get("b") != "2" {
		t.Fatal("Form values wrong")
	}

	// Cookie
	ck := &http.Cookie{Name: "sid", Value: "abc"}
	r.AddCookie(ck)
	got, err := c.Cookie("sid")
	if err != nil || got.Value != "abc" {
		t.Fatalf("Cookie got %v err %v", got, err)
	}
}

func buildMultipartBody(t *testing.T) (contentType string, bodyBytes []byte) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormField("name")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(fw, "mizu")
	pw, err := mw.CreateFormFile("file", "x.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(pw, "hello")
	_ = mw.Close()
	return mw.FormDataContentType(), body.Bytes()
}

func TestMultipartAndCleanup(t *testing.T) {
	// First request
	ct1, b1 := buildMultipartBody(t)
	r := httptest.NewRequest(http.MethodPost, "http://x.test/upload", bytes.NewReader(b1))
	r.Header.Set("Content-Type", ct1)
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)

	form, cleanup1, err := c.MultipartForm(32 << 10)
	if err != nil {
		t.Fatalf("MultipartForm error: %v", err)
	}
	defer cleanup1()
	if form.Value["name"][0] != "mizu" {
		t.Fatal("Multipart value mismatch")
	}

	// Second request with a fresh body and content type
	ct2, b2 := buildMultipartBody(t)
	r2 := httptest.NewRequest(http.MethodPost, "http://x.test/upload", bytes.NewReader(b2))
	r2.Header.Set("Content-Type", ct2)
	c2 := newCtx(httptest.NewRecorder(), r2, nil)

	form2, cleanup2, err := c2.MultipartForm(32 << 10)
	if err != nil {
		t.Fatalf("MultipartForm error: %v", err)
	}
	if form2.File["file"] == nil || form2.Value["name"][0] != "mizu" {
		t.Fatal("Multipart form contents unexpected")
	}
	cleanup2() // should not panic
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)

	// RemoteAddr host:port
	r.RemoteAddr = "203.0.113.10:5555"
	if ip := c.ClientIP(); ip != "203.0.113.10" {
		t.Fatalf("ClientIP got %q", ip)
	}

	// X-Real-IP
	r.Header.Set("X-Real-IP", "203.0.113.11")
	if ip := c.ClientIP(); ip != "203.0.113.11" {
		t.Fatalf("ClientIP X-Real-IP got %q", ip)
	}

	// X-Forwarded-For first hop
	r.Header.Set("X-Forwarded-For", "203.0.113.12, 10.0.0.1")
	if ip := c.ClientIP(); ip != "203.0.113.12" {
		t.Fatalf("ClientIP XFF got %q", ip)
	}

	// Invalid headers fallback to RemoteAddr string
	r.Header.Set("X-Forwarded-For", "not-an-ip")
	r.Header.Set("X-Real-IP", "still-bad")
	r.RemoteAddr = "unparseable"
	if ip := c.ClientIP(); ip != "unparseable" {
		t.Fatalf("ClientIP fallback got %q", ip)
	}
}

func TestBindJSON(t *testing.T) {
	type In struct {
		Name string `json:"name"`
	}
	// OK
	body := `{"name":"mizu"}`
	r := httptest.NewRequest(http.MethodPost, "http://x.test", strings.NewReader(body))
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)
	var in In
	if err := c.BindJSON(&in, 1024); err != nil || in.Name != "mizu" {
		t.Fatalf("BindJSON ok err=%v name=%q", err, in.Name)
	}
	// Unknown field
	r2 := httptest.NewRequest(http.MethodPost, "http://x.test", strings.NewReader(`{"name":"x","extra":1}`))
	c2 := newCtx(httptest.NewRecorder(), r2, nil)
	if err := c2.BindJSON(&in, 1024); err == nil {
		t.Fatal("BindJSON should fail on unknown field")
	}
	// Trailing data
	r3 := httptest.NewRequest(http.MethodPost, "http://x.test", strings.NewReader(`{"name":"x"}{"oops":1}`))
	c3 := newCtx(httptest.NewRecorder(), r3, nil)
	if err := c3.BindJSON(&in, 1024); err == nil {
		t.Fatal("BindJSON should fail on trailing data")
	}
	// Too large
	r4 := httptest.NewRequest(http.MethodPost, "http://x.test", strings.NewReader(strings.Repeat("a", 2048)))
	c4 := newCtx(httptest.NewRecorder(), r4, nil)
	if err := c4.BindJSON(&in, 10); err == nil {
		t.Fatal("BindJSON should fail on size limit")
	}
}

// ===== Split former TestStatusHeadersAndWrites into smaller tests =====

func TestStatusAndHeaderIfNone(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Status(201)
	if c.StatusCode() != 201 {
		t.Fatal("StatusCode not set")
	}
	c.HeaderIfNone("X-Test", "a")
	c.HeaderIfNone("X-Test", "b")
	if w.Header().Get("X-Test") != "a" {
		t.Fatal("HeaderIfNone should not overwrite")
	}
}

func TestJSONWriteOnce(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Status(201)
	if err := c.JSON(0, map[string]string{"ok": "1"}); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatal("JSON status or content-type wrong")
	}
	_ = c.JSON(0, map[string]string{"ok": "2"})
	if w.Code != 201 {
		t.Fatal("second JSON changed status")
	}
}

func TestHTMLWrite(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	_ = c.HTML(200, "<b>hi</b>")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "hi") {
		t.Fatal("HTML write failed")
	}
}

func TestTextValidUTF8(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	_ = c.Text(0, "hello")
	if w.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatal("Text content-type wrong")
	}
}

func TestTextInvalidUTF8FallsBackToBytes(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	invalid := string([]byte{0xff, 0xfe, 0xfd})
	_ = c.Text(0, invalid)
	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatal("invalid UTF-8 should use octet-stream")
	}
}

func TestBytesDefaultContentType(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	_ = c.Bytes(0, []byte{1, 2, 3}, "")
	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatal("Bytes default content type expected")
	}
}

func TestWriteAndWriteString(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Status(202)
	_, _ = c.Write([]byte("a"))
	_, _ = c.WriteString("b")
	if w.Code != 202 || w.Body.String() != "ab" {
		t.Fatal("Write/WriteString behavior wrong")
	}
}

func TestNoContent(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	_ = c.NoContent()
	if w.Code != http.StatusNoContent {
		t.Fatal("NoContent status wrong")
	}
}

func TestRedirectBasic(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	_ = c.Redirect(0, "/to")
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/to" {
		t.Fatal("Redirect behavior wrong")
	}
}

// ===== Split former TestFileAndDownload into smaller tests =====

func TestFile_Default200(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(fp, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	if err := c.File(fp); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || !strings.Contains(w.Body.String(), "content") {
		t.Fatal("File default failed")
	}
}

func TestFile_RespectsPresetStatus(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(fp, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Status(206)
	if err := c.File(fp); err != nil {
		t.Fatal(err)
	}
	if w.Code != 206 {
		t.Fatalf("File should respect preset status, got %d", w.Code)
	}
}

func TestFileCode_Explicit(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(fp, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	if err := c.FileCode(203, fp); err != nil {
		t.Fatal(err)
	}
	if w.Code != 203 {
		t.Fatalf("FileCode status got %d", w.Code)
	}
}

func TestDownload_BasicHeaders(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(fp, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	if err := c.Download(fp, "name.txt"); err != nil {
		t.Fatal(err)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, `attachment;`) || !strings.Contains(cd, `filename="name.txt"`) {
		t.Fatalf("Content-Disposition missing or invalid: %q", cd)
	}
	// Depending on platform, text/plain may or may not add charset
	ct := w.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" && ct != "text/plain" {
		t.Fatalf("Content-Type unexpected: %q", ct)
	}
}

func TestDownloadCode_Explicit(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(fp, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	if err := c.DownloadCode(207, fp, "name.txt"); err != nil {
		t.Fatal(err)
	}
	if w.Code != 207 {
		t.Fatalf("DownloadCode status got %d", w.Code)
	}
}

func TestStream(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)

	err := c.Stream(func(w io.Writer) error {
		_, _ = io.WriteString(w, "x")
		return nil
	})
	if err != nil || w.Body.String() != "x" {
		t.Fatalf("Stream failed: err=%v body=%q", err, w.Body.String())
	}
}

// A ResponseWriter that does NOT implement http.Flusher.
type nonFlusherRW struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func newNonFlusherRW() *nonFlusherRW {
	return &nonFlusherRW{header: make(http.Header)}
}

func (n *nonFlusherRW) Header() http.Header { return n.header }
func (n *nonFlusherRW) Write(b []byte) (int, error) {
	if n.code == 0 {
		n.code = http.StatusOK
	}
	return n.body.Write(b)
}
func (n *nonFlusherRW) WriteHeader(statusCode int) { n.code = statusCode }

func TestSSE_NoFlusher(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	w := newNonFlusherRW() // does not implement http.Flusher
	c := newCtx(w, r, nil)

	ch := make(chan any)
	go func() {
		ch <- map[string]string{"a": "b"}
		close(ch)
	}()

	if err := c.SSE(ch); err == nil {
		t.Fatal("SSE should fail without http.Flusher")
	}
}

func TestSSE_WithFlusher(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	w := flusherRecorder{httptest.NewRecorder()}
	c := newCtx(w, r, nil)

	ch := make(chan any)
	done := make(chan struct{})
	go func() {
		_ = c.SSE(ch)
		close(done)
	}()

	// Send two events then close
	ch <- map[string]string{"k": "v"}
	ch <- "plain"
	close(ch)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE did not finish after channel close")
	}
	body := w.Body.String()
	if !strings.Contains(body, "data:") || !strings.Contains(body, "event: end") {
		t.Fatalf("SSE output unexpected:\n%s", body)
	}
}

func TestSSE_EndFrameIsEmptyObject(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	w := flusherRecorder{httptest.NewRecorder()} // implements http.Flusher
	c := newCtx(w, r, nil)

	ch := make(chan any)
	done := make(chan struct{})

	go func() {
		_ = c.SSE(ch) // should write the end frame with data: {}
		close(done)
	}()

	// Close without sending any events to trigger the end branch
	close(ch)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE did not finish after channel close")
	}

	// Assert required SSE headers were set
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream; charset=utf-8" {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("unexpected Cache-Control: %q", cc)
	}
	if conn := w.Header().Get("Connection"); conn != "keep-alive" {
		t.Fatalf("unexpected Connection: %q", conn)
	}
	if nab := w.Header().Get("X-Accel-Buffering"); nab != "no" {
		t.Fatalf("unexpected X-Accel-Buffering: %q", nab)
	}

	// Assert the end event frame contains the empty JSON object
	body := w.Body.String()
	want := "event: end\ndata: {}\n\n"
	if !strings.Contains(body, want) {
		t.Fatalf("SSE end frame missing or malformed.\nBody:\n%s", body)
	}
}

func TestFlushHijackAndRC(t *testing.T) {
	// Flush with and without flusher
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)
	if err := c.Flush(); err != nil {
		t.Fatal(err)
	}

	w2 := flusherRecorder{httptest.NewRecorder()}
	c2 := newCtx(w2, r, nil)
	if err := c2.Flush(); err != nil {
		t.Fatal(err)
	}

	// Hijack unsupported
	if _, _, err := c.Hijack(); err == nil {
		t.Fatal("Hijack should fail on recorder")
	}

	// Hijack supported
	hw := newHijackableRecorder()
	c3 := newCtx(hw, r, nil)
	conn, rw, err := c3.Hijack()
	if err != nil || conn == nil || rw == nil || !hw.hijacked {
		t.Fatalf("Hijack expected success, got err=%v hijacked=%v", err, hw.hijacked)
	}
	_ = conn.Close()

	// ResponseController passthroughs should not panic
	_ = c.SetWriteDeadline(time.Now().Add(time.Second))
	_ = c.EnableFullDuplex()
}

func TestInternalHelpers(t *testing.T) {
	if got := firstNonZero(0, 2); got != 2 {
		t.Fatal("firstNonZero failed A")
	}
	if got := firstNonZero(3, 2); got != 3 {
		t.Fatal("firstNonZero failed B")
	}

	// sanitizeToken removes bad chars and controls
	in := "na\"me\r\n<>/[];:\t{}"
	out := sanitizeToken(in)
	if strings.ContainsAny(out, "\"\r\n<>/[];:\t{}") {
		t.Fatalf("sanitizeToken did not clean: %q", out)
	}

	// writeHeaderNow sets content type when provided
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	if err := c.writeHeaderNow(201, "text/plain"); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 || w.Header().Get("Content-Type") != "text/plain" {
		t.Fatal("writeHeaderNow did not apply code or content type")
	}
	// second call should be no-op
	if err := c.writeHeaderNow(202, "x/y"); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 {
		t.Fatal("writeHeaderNow should not change after first write")
	}
}

func TestRedirectAndURLNilSafety(t *testing.T) {
	// Ensure Query and QueryValues are safe if URL is nil
	r := &http.Request{} // intentionally minimal
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)
	if c.Query("x") != "" || len(c.QueryValues()) != 0 {
		t.Fatal("Query helpers should be safe on nil URL")
	}
	// Redirect sanity already tested, assert Location formatting again
	r2 := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	w2 := httptest.NewRecorder()
	c2 := newCtx(w2, r2, nil)
	if err := c2.Redirect(307, "/again"); err != nil {
		t.Fatal(err)
	}
	if w2.Code != 307 || w2.Header().Get("Location") != "/again" {
		t.Fatal("Redirect status or location wrong")
	}
}

func TestJSONEncoderEscaping(t *testing.T) {
	// Verify SetEscapeHTML(false) preserves characters
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	type S struct {
		X string `json:"x"`
	}
	_ = c.JSON(200, S{X: "<b>&"})
	if !strings.Contains(w.Body.String(), `"<b>&"`) {
		t.Fatalf("JSON should not escape html: %q", w.Body.String())
	}
}

func TestContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)
	if c.Context() != ctx {
		t.Fatal("Context should propagate")
	}
}

func TestCtx_SetCookie(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	c := newCtx(w, req, nil)

	ck := &http.Cookie{
		Name:  "session_id",
		Value: "abc123",
		Path:  "/",
	}

	c.SetCookie(ck)

	hdr := w.Header().Get("Set-Cookie")
	if hdr == "" {
		t.Fatalf("expected Set-Cookie header, got empty")
	}
	if want := "session_id=abc123"; hdr[:len(want)] != want {
		t.Fatalf("expected cookie %q, got %q", want, hdr)
	}
}

func TestCtx_SetWriter(t *testing.T) {
	w1 := httptest.NewRecorder()
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	c := newCtx(w1, req, nil)

	// Initial writer should be w1
	if c.Writer() != w1 {
		t.Fatal("initial writer mismatch")
	}

	// Set a new writer
	c.SetWriter(w2)

	// Writer should now be w2
	if c.Writer() != w2 {
		t.Fatal("SetWriter did not change the writer")
	}

	// Write should go to w2
	_, _ = c.Write([]byte("test"))
	if w2.Body.String() != "test" {
		t.Fatalf("expected w2 to have 'test', got %q", w2.Body.String())
	}
	if w1.Body.String() != "" {
		t.Fatalf("w1 should be empty, got %q", w1.Body.String())
	}
}

func TestCtx_SetCookie_Nil(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(w, req, nil)

	// Should not panic with nil cookie
	c.SetCookie(nil)

	if hdr := w.Header().Get("Set-Cookie"); hdr != "" {
		t.Fatalf("expected no Set-Cookie header for nil cookie, got %q", hdr)
	}
}

func TestForm_ParseError(t *testing.T) {
	// Create a request that will fail form parsing
	// POST request with content-type but invalid body
	r := httptest.NewRequest(http.MethodPost, "http://x.test/form", strings.NewReader("%invalid"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c := newCtx(httptest.NewRecorder(), r, nil)

	_, err := c.Form()
	if err == nil {
		t.Fatal("Form should return error for invalid form data")
	}
}

func TestMultipartForm_ParseError(t *testing.T) {
	// Create a request with invalid multipart content
	r := httptest.NewRequest(http.MethodPost, "http://x.test/upload", strings.NewReader("not valid multipart"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	c := newCtx(httptest.NewRecorder(), r, nil)

	_, cleanup, err := c.MultipartForm(32 << 10)
	if err == nil {
		t.Fatal("MultipartForm should return error for invalid multipart data")
	}
	// cleanup should be a no-op function even on error
	cleanup()
}

func TestSSE_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil).WithContext(ctx)
	w := flusherRecorder{httptest.NewRecorder()}
	c := newCtx(w, r, nil)

	ch := make(chan any)
	done := make(chan error, 1)

	go func() {
		done <- c.SSE(ch)
	}()

	// Cancel context while SSE is running
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("SSE should return context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SSE did not exit on context cancellation")
	}
}

func TestSSE_WriteError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	ew := &errorWriter{ResponseRecorder: httptest.NewRecorder()}
	c := newCtx(ew, r, nil)

	ch := make(chan any, 1)
	ch <- "test"

	err := c.SSE(ch)
	if err == nil {
		t.Fatal("SSE should return error on write failure")
	}
}

type errorWriter struct {
	*httptest.ResponseRecorder
}

func (e *errorWriter) Write(b []byte) (int, error) {
	return 0, io.EOF
}

func (e *errorWriter) Flush() {}

func TestStream_ContentTypePreset(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Header().Set("Content-Type", "text/plain")

	err := c.Stream(func(w io.Writer) error {
		_, _ = io.WriteString(w, "streamed")
		return nil
	})
	if err != nil || w.Header().Get("Content-Type") != "text/plain" {
		t.Fatal("Stream should respect preset Content-Type")
	}
}

func TestStream_Error(t *testing.T) {
	c, _, _ := newCtxRW(http.MethodGet, "http://x.test")

	err := c.Stream(func(w io.Writer) error {
		return io.EOF
	})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Stream should return handler error, got %v", err)
	}
}

func TestBytes_ContentTypePreset(t *testing.T) {
	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Header().Set("Content-Type", "application/xml")

	_ = c.Bytes(200, []byte("<xml/>"), "")
	if w.Header().Get("Content-Type") != "application/xml" {
		t.Fatal("Bytes should respect preset Content-Type when empty string passed")
	}
}

func TestFileWithCode_NoExtension(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "noext")
	if err := os.WriteFile(fp, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Status(201)
	if err := c.File(fp); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 {
		t.Fatalf("File should respect preset status, got %d", w.Code)
	}
}

func TestDownloadWithCode_EmptyName(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "myfile.txt")
	if err := os.WriteFile(fp, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Status(201)
	if err := c.Download(fp, ""); err != nil {
		t.Fatal(err)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "myfile.txt") {
		t.Fatalf("Download with empty name should use base name, got %q", cd)
	}
}

func TestDownloadWithCode_NoExtension(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "noext")
	if err := os.WriteFile(fp, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	c, w, _ := newCtxRW(http.MethodGet, "http://x.test")
	c.Status(201)
	if err := c.Download(fp, "noext"); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 {
		t.Fatalf("Download should respect preset status, got %d", w.Code)
	}
}

func TestStatus_ZeroOrNegative(t *testing.T) {
	c, _, _ := newCtxRW(http.MethodGet, "http://x.test")

	c.Status(0)
	if c.StatusCode() != http.StatusOK {
		t.Fatal("Status(0) should not change default")
	}

	c.Status(-1)
	if c.StatusCode() != http.StatusOK {
		t.Fatal("Status(-1) should not change status")
	}

	c.Status(201)
	if c.StatusCode() != 201 {
		t.Fatal("Status(201) should set status")
	}
}

func TestBindJSON_DecodeError(t *testing.T) {
	type In struct {
		Name string `json:"name"`
	}
	// Malformed JSON
	r := httptest.NewRequest(http.MethodPost, "http://x.test", strings.NewReader(`{malformed`))
	c := newCtx(httptest.NewRecorder(), r, nil)
	var in In
	if err := c.BindJSON(&in, 1024); err == nil {
		t.Fatal("BindJSON should fail on malformed JSON")
	}
}

func TestBindJSON_TokenError(t *testing.T) {
	type In struct {
		Name string `json:"name"`
	}
	// Valid JSON but reader fails after first object
	// Use a reader that will succeed for decode but then fail for Token()
	// by sending a stream that can't be fully tokenized
	r := httptest.NewRequest(http.MethodPost, "http://x.test", strings.NewReader(`{"name":"x"}extra-not-json`))
	c := newCtx(httptest.NewRecorder(), r, nil)
	var in In
	if err := c.BindJSON(&in, 1024); err == nil {
		t.Fatal("BindJSON should fail on trailing non-JSON data")
	}
}

func TestSSE_MarshalError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x.test", nil)
	w := flusherRecorder{httptest.NewRecorder()}
	c := newCtx(w, r, nil)

	ch := make(chan any, 1)
	done := make(chan struct{})

	go func() {
		_ = c.SSE(ch)
		close(done)
	}()

	// Send something that can't be marshaled to JSON (channel type)
	ch <- make(chan int)
	close(ch)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE did not finish")
	}

	// The body should contain a quoted fallback representation
	body := w.Body.String()
	if !strings.Contains(body, "data:") {
		t.Fatalf("SSE output should contain fallback data, got %q", body)
	}
}
