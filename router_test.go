package mizu

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"testing"
)

// helper to run a request
func do(r http.Handler, method, url string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRouter_New_DefaultNotFound_BoundOnce(t *testing.T) {
	rt := NewRouter()

	// Before any route registration, ensure NotFound works via ensureNotFoundBound
	w := do(rt, http.MethodGet, "http://x.test/unknown", "")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), http.StatusText(http.StatusNotFound)) {
		t.Fatalf("default NotFound failed: %d %q", w.Code, w.Body.String())
	}

	// Second call exercises notFoundBindOnce fast path
	w2 := do(rt, http.MethodGet, "http://x.test/unknown2", "")
	if w2.Code != http.StatusNotFound {
		t.Fatalf("NotFound second call: %d", w2.Code)
	}
}

func TestRouter_NotFound_Override(t *testing.T) {
	rt := NewRouter()
	rt.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "custom", http.StatusTeapot)
	}))

	w := do(rt, http.MethodGet, "http://x/c", "")
	if w.Code != http.StatusTeapot || !strings.Contains(w.Body.String(), "custom") {
		t.Fatalf("custom NotFound not used: %d %q", w.Code, w.Body.String())
	}
}

func TestRouter_GetHead405_AllowHeader(t *testing.T) {
	rt := NewRouter()
	rt.Get("/a", func(c *Ctx) error { return c.Text(200, "ok") })

	// GET works
	w1 := do(rt, http.MethodGet, "http://x/a", "")
	if w1.Code != 200 || w1.Body.String() != "ok" {
		t.Fatalf("GET failed: %d %q", w1.Code, w1.Body.String())
	}

	// HEAD auto allowed for GET, empty body
	w2 := do(rt, http.MethodHead, "http://x/a", "")
	if w2.Code != 200 {
		t.Fatalf("HEAD should be allowed via GET: %d", w2.Code)
	}

	// POST should 405 with Allow header including GET, HEAD, OPTIONS
	w3 := do(rt, http.MethodPost, "http://x/a", "")
	allow := w3.Header().Get("Allow")
	if w3.Code != http.StatusMethodNotAllowed || !strings.Contains(allow, "GET") || !strings.Contains(allow, "HEAD") || !strings.Contains(allow, "OPTIONS") {
		t.Fatalf("405 Allow wrong: code=%d Allow=%q", w3.Code, allow)
	}
}

func TestRouter_Handle_AllMethods_AndCompatHandleMethod(t *testing.T) {
	rt := NewRouter()

	calls := int32(0)
	h := func(c *Ctx) error { atomic.AddInt32(&calls, 1); return c.Text(201, "h") }

	// Register using Router methods
	rt.Post("/m", h)
	rt.Put("/m", h)
	rt.Patch("/m", h)
	rt.Delete("/m", h)
	rt.Connect("/m", h)
	rt.Trace("/m", h)
	rt.Head("/m", h)

	// A compat-only method registration
	rt.Compat.HandleMethod(http.MethodPost, "/c", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
		_, _ = w.Write([]byte("c"))
	}))
	// And also GET through compat to add HEAD auto-allow in compat path
	rt.Compat.HandleMethod(http.MethodGet, "/c", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(203)
	}))

	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodTrace, http.MethodHead} {
		w := do(rt, m, "http://x/m", "")
		if w.Code == 0 {
			t.Fatalf("%s /m no response", m)
		}
	}

	// compat POST
	wc := do(rt, http.MethodPost, "http://x/c", "")
	if wc.Code != 202 || wc.Body.String() != "c" {
		t.Fatalf("compat post failed: %d %q", wc.Code, wc.Body.String())
	}
	// compat HEAD allowed via GET
	wh := do(rt, http.MethodHead, "http://x/c", "")
	if wh.Code != 203 {
		t.Fatalf("compat HEAD via GET expected 203, got %d", wh.Code)
	}

	if calls == 0 {
		t.Fatal("router handlers did not run")
	}
}

func TestRouter_Use_UseFirst_ComposeOrder(t *testing.T) {
	rt := NewRouter()
	var order []string

	rt.Use(func(next Handler) Handler {
		return func(c *Ctx) error { order = append(order, "last"); return next(c) }
	})
	rt.UseFirst(func(next Handler) Handler {
		return func(c *Ctx) error { order = append(order, "first"); return next(c) }
	})
	rt.Get("/x", func(c *Ctx) error { order = append(order, "handler"); return c.NoContent() })

	_ = do(rt, http.MethodGet, "http://x/x", "")

	got := strings.Join(order, ",")
	if got != "first,last,handler" {
		t.Fatalf("middleware order wrong: %s", got)
	}
}

func TestRouter_Group_Prefix_fullPath_TrailingSlash_Subtree(t *testing.T) {
	rt := NewRouter()
	rt.Group("/api", func(g *Router) {
		g.Get("/v1/users", func(c *Ctx) error { return c.Text(200, "users") })
		g.Get("v1/projects/", func(c *Ctx) error { return c.Text(200, "projects") }) // subtree
	})
	// Plain match
	w1 := do(rt, http.MethodGet, "http://x/api/v1/users", "")
	if w1.Code != 200 || w1.Body.String() != "users" {
		t.Fatalf("group route failed")
	}
	// Subtree match due to trailing slash preservation
	w2 := do(rt, http.MethodGet, "http://x/api/v1/projects/123", "")
	if w2.Code != 200 || w2.Body.String() != "projects" {
		t.Fatalf("subtree match failed: %d %q", w2.Code, w2.Body.String())
	}
}

func TestRouter_Static_And_Mount(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(fp, []byte("filedata"), 0o600); err != nil {
		t.Fatal(err)
	}

	rt := NewRouter()
	// http.Dir implements http.FileSystem
	rt.Static("/assets", http.Dir(dir))

	// File served under prefix
	w1 := do(rt, http.MethodGet, "http://x/assets/a.txt", "")
	if w1.Code != 200 || !strings.Contains(w1.Body.String(), "filedata") {
		t.Fatalf("static file failed: %d %q", w1.Code, w1.Body.String())
	}

	// Mount plain handler
	rt.Mount("/hi", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("hi")) }))
	w2 := do(rt, http.MethodGet, "http://x/hi", "")
	if w2.Body.String() != "hi" {
		t.Fatalf("mount failed: %q", w2.Body.String())
	}
}

func TestRouter_adapt_ErrorAndPanic_Paths(t *testing.T) {
	rt := NewRouter()

	// Error path without custom error handler writes 500
	rt.Get("/err", func(c *Ctx) error { return errors.New("boom") })
	w1 := do(rt, http.MethodGet, "http://x/err", "")
	if w1.Code != 500 {
		t.Fatalf("expected 500 on error, got %d", w1.Code)
	}

	// Panic path without custom handler -> 500, logged
	rt.Get("/panic", func(c *Ctx) error { panic("kapow") })
	w2 := do(rt, http.MethodGet, "http://x/panic", "")
	if w2.Code != 500 {
		t.Fatalf("expected 500 on panic, got %d", w2.Code)
	}

	// With custom error handler receives PanicError and returned error
	var seen []string
	rt2 := NewRouter()
	rt2.ErrorHandler(func(c *Ctx, err error) {
		switch e := err.(type) {
		case *PanicError:
			if e.Value == nil || len(e.Stack) == 0 || !bytes.Contains(e.Stack, []byte("runtime")) {
				t.Fatalf("panic error missing data: val=%v stack=%d", e.Value, len(e.Stack))
			}
			_ = c.Text(599, "panic handled")
		default:
			_ = c.Text(598, "err handled")
		}
		seen = append(seen, err.Error())
	})
	rt2.Get("/e", func(c *Ctx) error { return errors.New("e1") })
	rt2.Get("/p", func(c *Ctx) error { panic("p1") })

	w3 := do(rt2, http.MethodGet, "http://x/e", "")
	w4 := do(rt2, http.MethodGet, "http://x/p", "")
	if w3.Code != 598 || w4.Code != 599 {
		t.Fatalf("custom error handler not applied: %d %d", w3.Code, w4.Code)
	}
	if len(seen) != 2 || !strings.Contains(seen[1], "panic: p1") {
		t.Fatalf("custom error handler did not see both errors: %v", seen)
	}
}

func TestRouter_adaptStdMiddleware(t *testing.T) {
	rt := NewRouter()
	// std middleware adds a header, then calls next
	std := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Std", "1")
			next.ServeHTTP(w, r)
		})
	}
	rt.Compat.Use(std)

	rt.Get("/z", func(c *Ctx) error {
		return c.Text(200, "z")
	})

	w := do(rt, http.MethodGet, "http://x/z", "")
	if w.Header().Get("X-Std") != "1" || w.Code != 200 {
		t.Fatalf("adaptStdMiddleware failed: code=%d X-Std=%q", w.Code, w.Header().Get("X-Std"))
	}
}

func TestRouter_allowHeader_Sorting_OptionsPresence(t *testing.T) {
	rt := NewRouter()
	full := "/t"
	// Simulate a mix of methods
	rt.allowAdd(full, "POST")
	rt.allowAdd(full, "DELETE")
	rt.allowAdd(full, "GET") // auto adds HEAD in handle, we add manually here to test set behavior
	rt.allowAdd(full, "HEAD")

	h := rt.allowHeader(full)
	// Must include OPTIONS and be sorted
	if !strings.Contains(h, "OPTIONS") {
		t.Fatalf("allow header missing OPTIONS: %q", h)
	}
	got := strings.Split(h, ", ")
	wantOrder := []string{"DELETE", "GET", "HEAD", "OPTIONS", "POST"}
	if len(got) != len(wantOrder) {
		t.Fatalf("allow header length mismatch: %q", h)
	}
	for i, v := range wantOrder {
		if got[i] != v {
			t.Fatalf("allow header not sorted, got %q want %q (%q)", got[i], v, h)
		}
	}

	// Path with no known methods returns OPTIONS
	if rt.allowHeader("/none") != "OPTIONS" {
		t.Fatal("unknown path should return OPTIONS")
	}
}

func Test_joinPath_cleanLeading_fullPath(t *testing.T) {
	// joinPath base variations
	if got := joinPath("", "x"); got != "/x" {
		t.Fatalf("joinPath 1: %q", got)
	}
	if got := joinPath("/", "/x"); got != "/x" {
		t.Fatalf("joinPath 2: %q", got)
	}
	if got := joinPath("a", "b"); got != "/a/b" {
		t.Fatalf("joinPath 3: %q", got)
	}
	if got := cleanLeading("x"); got != "/x" {
		t.Fatalf("cleanLeading: %q", got)
	}

	// fullPath keeps subtree trailing slash intent
	rt := NewRouter()
	rt.base = "/api"
	if got := rt.fullPath("v1"); got != "/api/v1" {
		t.Fatalf("fullPath no slash: %q", got)
	}
	if got := rt.fullPath("/v1/"); got != "/api/v1/" {
		t.Fatalf("fullPath subtree: %q", got)
	}
	// empty path resolves to the router base
	if got := rt.fullPath(""); got != "/api" {
		t.Fatalf("fullPath empty: %q", got)
	}
}

func TestRouter_Dev_InstallsLoggerAndMiddleware(t *testing.T) {
	rt := NewRouter()
	// Force non-color path to avoid ANSI assertions
	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Setenv("NO_COLOR", "")
	}()

	rt.Dev(true)
	if rt.Logger() == nil {
		t.Fatal("Dev should set logger")
	}

	// Install a trivial route and ensure request flows
	rt.Get("/d", func(c *Ctx) error { return c.NoContent() })
	w := do(rt, http.MethodGet, "http://x/d", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("Dev route flow failed: %d", w.Code)
	}
}

func Test_httpRouter_Prefix_Group_Handle(t *testing.T) {
	rt := NewRouter()
	h := rt.Compat.Prefix("/p")
	h.Handle("/k", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }))
	w := do(rt, http.MethodGet, "http://x/p/k", "")
	if w.Body.String() != "ok" {
		t.Fatalf("compat prefix handle failed: %q", w.Body.String())
	}

	// Group mounts inside
	rt2 := NewRouter()
	rt2.Compat.Group("/g", func(g *httpRouter) {
		g.Handle("/h", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("gg")) }))
	})
	w2 := do(rt2, http.MethodGet, "http://x/g/h", "")
	if w2.Body.String() != "gg" {
		t.Fatalf("compat group failed: %q", w2.Body.String())
	}
}

func TestPanicError_ErrorString(t *testing.T) {
	p := &PanicError{Value: "v", Stack: debug.Stack()}
	if !strings.Contains(p.Error(), "panic: v") {
		t.Fatalf("PanicError string unexpected: %q", p.Error())
	}
}

func TestRouter_Static_WithCustomFS(t *testing.T) {
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "x.txt")
	if err := os.WriteFile(fp, []byte("Y"), 0o600); err != nil {
		t.Fatal(err)
	}
	rt := NewRouter()
	// http.FS converts an fs.FS into an http.FileSystem
	rt.Static("/s", http.FS(os.DirFS(tmp)))
	w := do(rt, http.MethodGet, "http://x/s/x.txt", "")
	if w.Body.String() != "Y" {
		t.Fatalf("custom FS static failed: %q", w.Body.String())
	}
}

func TestRouter_SetLogger_GetLogger(t *testing.T) {
	rt := NewRouter()
	if rt.Logger() == nil {
		t.Fatal("default logger nil")
	}
	buf := new(bytes.Buffer)
	lg := slog.New(slog.NewTextHandler(buf, nil))
	rt.SetLogger(lg)
	if rt.Logger() != lg {
		t.Fatal("SetLogger did not take effect")
	}
}

// Covers the 405 guard inside httpRouter.HandleMethod and its Allow header.
func TestCompat_HandleMethod_405_AllowHeader(t *testing.T) {
	r := NewRouter()

	// Only allow POST on /compat
	r.Compat.HandleMethod("POST", "/compat", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))

	// Wrong method -> 405 and Allow includes POST and OPTIONS
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/compat", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	allow := rr.Header().Get("Allow")
	if !strings.Contains(allow, "POST") || !strings.Contains(allow, "OPTIONS") {
		t.Fatalf("Allow header missing methods, got %q", allow)
	}
}

// Covers httpRouter.Mount delegating to Handle and returning the compat router.
func TestHttpRouter_Mount_Chaining(t *testing.T) {
	r := NewRouter()

	var called bool
	ret := r.Compat.Mount("/mounted", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	if ret == nil {
		t.Fatalf("Mount should return *httpRouter for chaining")
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mounted", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "ok" || !called {
		t.Fatalf("mount failed: code=%d body=%q called=%v", rr.Code, rr.Body.String(), called)
	}
}

// Covers Dev(false) early return and that slog.Default is not replaced.
func TestRouter_Dev_Disabled_NoChange(t *testing.T) {
	old := slog.Default()

	r := NewRouter()
	ret := r.Dev(false) // should be a no-op
	if ret != r {
		t.Fatalf("Dev(false) should return receiver")
	}
	if slog.Default() != old {
		t.Fatalf("Dev(false) should not change slog.Default")
	}

	// Router still works
	r.Get("/ok", func(c *Ctx) error { c.Writer().WriteHeader(200); return nil })
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("route failed after Dev(false): %d", rr.Code)
	}
}

// Covers the color handler branch used by Dev when FORCE_COLOR is set.
func TestRouter_Dev_ColorHandlerBranch(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	defer t.Setenv("FORCE_COLOR", "")

	r := NewRouter()
	r.Dev(true) // should choose newColorTextHandler internally

	// Smoke a request to ensure logger middleware path is active
	r.Get("/devcolor", func(c *Ctx) error {
		return c.Text(200, "ok")
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/devcolor", nil)
	r.ServeHTTP(rr, req)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Fatalf("dev color path failed: %d %q", rr.Code, rr.Body.String())
	}
}

// Covers adaptStdMiddleware error branch where no custom ErrorHandler is set.
// Ensures it logs and writes a 500 fallback.
func TestAdaptStdMiddleware_ErrorFallback500(t *testing.T) {
	r := NewRouter()
	// Ensure router has a non nil logger to exercise the log path
	r.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Adapt one std middleware to go through adaptStdMiddleware
	r.Compat.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req)
		})
	})

	// Mizu handler returns an error to trigger the 500 fallback
	r.Get("/err", func(c *Ctx) error { return io.EOF })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 fallback, got %d", rr.Code)
	}
}

// Covers Router.Handle uppercasing of method and registration path.
func TestRouter_Handle_LowercaseMethod(t *testing.T) {
	r := NewRouter()
	r.Handle("post", "/h", func(c *Ctx) error { c.Writer().WriteHeader(201); return nil })

	// POST works
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/h", nil)
	r.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("POST want 201 got %d", rr.Code)
	}

	// GET yields 405 with Allow containing POST and OPTIONS
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/h", nil)
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET want 405 got %d", rr2.Code)
	}
	al := rr2.Header().Get("Allow")
	if !strings.Contains(al, "POST") || !strings.Contains(al, "OPTIONS") {
		t.Fatalf("Allow missing POST or OPTIONS: %q", al)
	}
}

// Ensures Dev installs logger middleware using color handler when environment allows.
// Writes to stderr to match Dev defaults but keeps output discarded for the test process.
func TestRouter_Dev_UsesStderrColorHandler(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	defer t.Setenv("FORCE_COLOR", "")

	// Point stderr to a pipe to avoid polluting test output
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pr.Close() }()
	defer func() { _ = pw.Close() }()

	oldStderr := os.Stderr
	os.Stderr = pw
	defer func() { os.Stderr = oldStderr }()

	r := NewRouter()
	r.Dev(true)

	// Simple request to exercise handler path and produce a log line
	r.Get("/x", func(c *Ctx) error { return c.Text(200, "ok") })
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
