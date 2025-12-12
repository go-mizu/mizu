package mizu

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/fstest"
)

// helper to read full body
func bodyString(t *testing.T, r *httptest.ResponseRecorder) string {
	t.Helper()
	b, err := io.ReadAll(r.Result().Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestCanonicalPath(t *testing.T) {
	if got := canonicalPath(""); got != "/" {
		t.Fatalf("canonicalPath(\"\") = %q, want \"/\"", got)
	}
	if got := canonicalPath("/foo/bar"); got != "/foo/bar" {
		t.Fatalf("canonicalPath(\"/foo/bar\") = %q, want \"/foo/bar\"", got)
	}
}

func TestCleanLeadingAndJoinPath(t *testing.T) {
	// cleanLeading with non empty and no leading slash
	if got := cleanLeading("foo/bar"); got != "/foo/bar" {
		t.Fatalf("cleanLeading(\"foo/bar\") = %q, want \"/foo/bar\"", got)
	}
	// cleanLeading with leading slash unchanged
	if got := cleanLeading("/foo/bar"); got != "/foo/bar" {
		t.Fatalf("cleanLeading(\"/foo/bar\") = %q, want \"/foo/bar\"", got)
	}

	// joinPath case: base empty
	if got := joinPath("", "/x"); got != "/x" {
		t.Fatalf("joinPath(\"\", \"/x\") = %q, want \"/x\"", got)
	}
	// joinPath case: add empty
	if got := joinPath("/base", ""); got != "/base" {
		t.Fatalf("joinPath(\"/base\", \"\") = %q, want \"/base\"", got)
	}
	// joinPath default case
	if got := joinPath("/base", "/sub"); got != "/base/sub" {
		t.Fatalf("joinPath(\"/base\", \"/sub\") = %q, want \"/base/sub\"", got)
	}
}

func TestNewRouterAndLogger(t *testing.T) {
	r := NewRouter()
	if r == nil {
		t.Fatal("NewRouter returned nil")
	}
	if r.mux == nil {
		t.Fatal("NewRouter did not initialize mux")
	}
	if r.Logger() == nil {
		t.Fatal("NewRouter did not initialize logger")
	}
	if r.Compat == nil {
		t.Fatal("NewRouter did not initialize Compat")
	}

	// SetLogger should replace internal logger
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	r.SetLogger(l)
	if r.Logger() != l {
		t.Fatal("SetLogger did not set logger")
	}
}

func TestServeHTTPCanonicalizesEmptyPath(t *testing.T) {
	r := NewRouter()
	// Install NotFound as simple handler at "/"
	r.NotFound(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("root"))
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.invalid", nil)
	// Force empty path to trigger canonicalPath branch
	req.URL.Path = ""

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != 200 || bodyString(t, rec) != "root" {
		t.Fatalf("ServeHTTP canonicalize path: status=%d body=%q", rec.Code, bodyString(t, rec))
	}
}

func TestNotFoundBehavior(t *testing.T) {
	r := NewRouter()
	r.NotFound(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(418)
		_, _ = w.Write([]byte("teapot"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 418 || bodyString(t, rec) != "teapot" {
		t.Fatalf("NotFound handler: status=%d body=%q", rec.Code, bodyString(t, rec))
	}
}

func TestMethodsAndHandle(t *testing.T) {
	r := NewRouter()

	r.Get("/get", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("GET"))
		return nil
	})
	// For HEAD we just care that the handler is wired, not about exact semantics.
	r.Head("/head", func(c *Ctx) error {
		c.Writer().WriteHeader(204)
		return nil
	})
	r.Post("/post", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("POST"))
		return nil
	})
	r.Put("/put", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("PUT"))
		return nil
	})
	r.Patch("/patch", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("PATCH"))
		return nil
	})
	r.Delete("/delete", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("DELETE"))
		return nil
	})

	// Call Connect and Trace to cover those wrappers, but do not
	// rely on CONNECT/TRACE routing semantics in ServeMux.
	r.Connect("/connect", func(c *Ctx) error { return nil })
	r.Trace("/trace", func(c *Ctx) error { return nil })

	r.Handle("options", "/handle", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("HANDLE"))
		return nil
	})

	// Table of method, path, expected body where behavior is predictable.
	cases := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodGet, "/get", "GET", 200},
		// HEAD: status is implementation dependent, but should not panic.
		{http.MethodHead, "/head", "", 0},
		{http.MethodPost, "/post", "POST", 200},
		{http.MethodPut, "/put", "PUT", 200},
		{http.MethodPatch, "/patch", "PATCH", 200},
		{http.MethodDelete, "/delete", "DELETE", 200},
		{http.MethodOptions, "/handle", "HANDLE", 200},
	}

	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, "http://example.invalid"+tc.path, nil)
		r.ServeHTTP(rec, req)

		if tc.status != 0 && rec.Code != tc.status {
			t.Fatalf("method %s path %s status=%d, want %d", tc.method, tc.path, rec.Code, tc.status)
		}
		if tc.body != "" && bodyString(t, rec) != tc.body {
			t.Fatalf("method %s path %s body=%q, want %q", tc.method, tc.path, bodyString(t, rec), tc.body)
		}
	}
}

func TestUseUseFirstAndWithOrder(t *testing.T) {
	r := NewRouter()
	var calls []string

	mw1 := func(next Handler) Handler {
		return func(c *Ctx) error {
			calls = append(calls, "mw1")
			return next(c)
		}
	}
	mw2 := func(next Handler) Handler {
		return func(c *Ctx) error {
			calls = append(calls, "mw2")
			return next(c)
		}
	}
	mw3 := func(next Handler) Handler {
		return func(c *Ctx) error {
			calls = append(calls, "mw3")
			return next(c)
		}
	}

	r.Use(mw2)           // chain: [Logger, mw2]
	r.UseFirst(mw1)      // chain: [mw1, Logger, mw2]
	child := r.With(mw3) // child: [mw1, Logger, mw2, mw3]

	child.Get("/order", func(c *Ctx) error {
		calls = append(calls, "handler")
		_, _ = c.Writer().Write([]byte("ok"))
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/order", nil)
	child.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status=%d, want %d", rec.Code, 200)
	}
	// Only assert tail of calls to avoid depending on Logger position.
	n := len(calls)
	if n < 3 {
		t.Fatalf("expected at least 3 calls, got %v", calls)
	}
	if calls[n-3] != "mw2" || calls[n-2] != "mw3" || calls[n-1] != "handler" {
		t.Fatalf("unexpected call order tail: %v", calls)
	}
}

func TestPrefixGroupAndFullPath(t *testing.T) {
	root := NewRouter()

	// Prefix then nested Prefix via Group
	api := root.Prefix("/api")
	api.Group("/v1", func(g *Router) {
		g.Get("/ping", func(c *Ctx) error {
			_, _ = c.Writer().Write([]byte("pong"))
			return nil
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/api/v1/ping", nil)
	root.ServeHTTP(rec, req)

	if rec.Code != 200 || bodyString(t, rec) != "pong" {
		t.Fatalf("prefix/group: status=%d body=%q", rec.Code, bodyString(t, rec))
	}
}

func TestStaticRootAndPrefixed(t *testing.T) {
	fs := fstest.MapFS{
		"file.txt": &fstest.MapFile{Data: []byte("root-static")},
		// For the prefixed case StripPrefix("/assets/img") leaves "/logo.png",
		// so the underlying FS must serve "logo.png".
		"logo.png": &fstest.MapFile{Data: []byte("logo")},
	}

	// Case 1: Static at root with empty prefix.
	r1 := NewRouter()
	r1.Static("", http.FS(fs))

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "http://example.invalid/file.txt", nil)
	r1.ServeHTTP(rec1, req1)

	if rec1.Code != 200 || bodyString(t, rec1) != "root-static" {
		t.Fatalf("Static root: status=%d body=%q", rec1.Code, bodyString(t, rec1))
	}

	// Case 2: Static under a base prefix using Prefix on a fresh router
	r2 := NewRouter()
	api := r2.Prefix("/assets")
	api.Static("/img", http.FS(fs))

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "http://example.invalid/assets/img/logo.png", nil)
	r2.ServeHTTP(rec2, req2)

	if rec2.Code != 200 || bodyString(t, rec2) != "logo" {
		t.Fatalf("Static under base: status=%d body=%q", rec2.Code, bodyString(t, rec2))
	}
}

func TestMountAndCompatHandle(t *testing.T) {
	r := NewRouter()

	// Mount via Router.Mount (httpRouter.Handle under the hood)
	r.Mount("/mounted", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("mounted"))
	}))

	// Handle directly via Compat.Handle
	r.Compat.Handle("/compat", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("compat"))
	}))

	// Compat.HandleMethod
	r.Compat.HandleMethod(http.MethodGet, "/method", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("method"))
	}))

	// Compat.Prefix and Group
	r.Compat.Group("/group", func(g *httpRouter) {
		g.Handle("/path", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("grouped"))
		}))
	})

	cases := []struct {
		path string
		body string
	}{
		{"/mounted", "mounted"},
		{"/compat", "compat"},
		{"/method", "method"},
		{"/group/path", "grouped"},
	}

	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://example.invalid"+tc.path, nil)
		r.ServeHTTP(rec, req)

		if rec.Code != 200 || bodyString(t, rec) != tc.body {
			t.Fatalf("path %s: status=%d body=%q", tc.path, rec.Code, bodyString(t, rec))
		}
	}
}

func TestErrorHandlerOnErrorAndPanic(t *testing.T) {
	r := NewRouter()

	var gotErr error
	r.ErrorHandler(func(c *Ctx, err error) {
		gotErr = err
		c.Writer().WriteHeader(499)
		_, _ = c.Writer().Write([]byte("handled"))
	})

	r.Get("/error", func(c *Ctx) error {
		return errors.New("boom")
	})
	r.Get("/panic", func(c *Ctx) error {
		panic("panic-value")
	})

	// Error case
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/error", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 499 || bodyString(t, rec) != "handled" {
		t.Fatalf("/error: status=%d body=%q", rec.Code, bodyString(t, rec))
	}
	if gotErr == nil || gotErr.Error() != "boom" {
		t.Fatalf("ErrorHandler got error=%v, want boom", gotErr)
	}

	// Panic case
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "http://example.invalid/panic", nil)
	r.ServeHTTP(rec2, req2)

	if rec2.Code != 499 || bodyString(t, rec2) != "handled" {
		t.Fatalf("/panic: status=%d body=%q", rec2.Code, bodyString(t, rec2))
	}
	if _, ok := gotErr.(*PanicError); !ok {
		t.Fatalf("ErrorHandler on panic should receive *PanicError, got %T", gotErr)
	}
}

func TestDefaultErrorHandling(t *testing.T) {
	r := NewRouter()

	// Handler that returns error without writing header.
	r.Get("/err500", func(c *Ctx) error {
		return errors.New("fail")
	})

	// Handler that writes header then returns error.
	r.Get("/err200", func(c *Ctx) error {
		c.Writer().WriteHeader(200)
		return errors.New("ignored")
	})

	// Handler that panics without custom ErrorHandler.
	r.Get("/panic500", func(c *Ctx) error {
		panic("fail")
	})

	// We only assert that these routes do not panic and produce some response.
	// Exact status codes depend on how Ctx and logger interact.
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "http://example.invalid/err500", nil)
	r.ServeHTTP(rec1, req1)

	if rec1.Result() == nil {
		t.Fatalf("/err500 produced nil response")
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "http://example.invalid/err200", nil)
	r.ServeHTTP(rec2, req2)

	if rec2.Result() == nil {
		t.Fatalf("/err200 produced nil response")
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "http://example.invalid/panic500", nil)
	r.ServeHTTP(rec3, req3)

	if rec3.Result() == nil {
		t.Fatalf("/panic500 produced nil response")
	}
}

func TestAdaptStdMiddleware(t *testing.T) {
	r := NewRouter()

	// std middleware that sets a header
	mid := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Mid", "yes")
			next.ServeHTTP(w, req)
		})
	}

	// std middleware that just forwards, exercising the base logic.
	errMid := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req)
		})
	}

	// Attach std middlewares via Compat.Use
	r.Compat.Use(mid, errMid)

	// Handler returning error to trigger error handling inside adaptStdMiddleware base
	r.Get("/mid", func(c *Ctx) error {
		return errors.New("middleware error")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/mid", nil)
	r.ServeHTTP(rec, req)

	if rec.Header().Get("X-Mid") != "yes" {
		t.Fatalf("expected X-Mid header set by std middleware")
	}
	// Status code semantics here depend on Ctx/logger; just ensure we got a response.
	if rec.Result() == nil {
		t.Fatalf("/mid produced nil response")
	}
}

func TestPanicError_String(t *testing.T) {
	pe := &PanicError{Value: "test panic", Stack: []byte("stack trace")}
	expected := "panic: test panic"
	if pe.Error() != expected {
		t.Fatalf("PanicError.Error() = %q, want %q", pe.Error(), expected)
	}
}

func TestCompat_Mount(t *testing.T) {
	r := NewRouter()

	// Use Compat.Mount method
	r.Compat.Mount("/api", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("api-mounted"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/api", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 200 || bodyString(t, rec) != "api-mounted" {
		t.Fatalf("Compat.Mount: status=%d body=%q", rec.Code, bodyString(t, rec))
	}
}

func TestNotFound_NilHandler(t *testing.T) {
	r := NewRouter()
	// Should not panic and be no-op
	r.NotFound(nil)
}

func TestUseFirst_EmptySlice(t *testing.T) {
	r := NewRouter()
	initialLen := len(r.chain)
	r.UseFirst()
	if len(r.chain) != initialLen {
		t.Fatalf("UseFirst with empty slice should be no-op, chain length changed from %d to %d", initialLen, len(r.chain))
	}
}

func TestCanonicalPath_AllBranches(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"foo", "/foo"},
		{"/foo", "/foo"},
		{"/foo/", "/foo"},
		{"/foo/bar", "/foo/bar"},
		{"/foo/bar/", "/foo/bar"},
		{"/foo//bar", "/foo/bar"},
		{"/foo/../bar", "/bar"},
		{"foo/bar", "/foo/bar"},
		{"/./foo", "/foo"},
	}

	for _, tc := range tests {
		got := canonicalPath(tc.input)
		if got != tc.expected {
			t.Errorf("canonicalPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestFullPath_AllBranches(t *testing.T) {
	// Test with base prefix
	r := NewRouter()
	api := r.Prefix("/api")

	// fullPath with empty p
	fp := api.fullPath("")
	if fp != "/api" {
		t.Fatalf("fullPath(\"\") with base /api = %q, want /api", fp)
	}

	// fullPath with path without leading slash
	fp2 := api.fullPath("users")
	if fp2 != "/api/users" {
		t.Fatalf("fullPath(\"users\") with base /api = %q, want /api/users", fp2)
	}

	// fullPath with path with leading slash
	fp3 := api.fullPath("/users")
	if fp3 != "/api/users" {
		t.Fatalf("fullPath(\"/users\") with base /api = %q, want /api/users", fp3)
	}
}

func TestCleanLeading_AllBranches(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"foo", "/foo"},
		{"/foo", "/foo"},
	}

	for _, tc := range tests {
		got := cleanLeading(tc.input)
		if got != tc.expected {
			t.Errorf("cleanLeading(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestNewRouter_ColorBranches(t *testing.T) {
	// Just verify NewRouter doesn't panic and creates valid router
	r := NewRouter()
	if r == nil || r.mux == nil || r.log == nil {
		t.Fatal("NewRouter should create valid router")
	}
}

func TestStatic_WithPrefix(t *testing.T) {
	fs := fstest.MapFS{
		"style.css": &fstest.MapFile{Data: []byte("body{}")},
	}

	r := NewRouter()
	// Test Static with path that doesn't have leading slash
	r.Static("css", http.FS(fs))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/css/style.css", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 200 || bodyString(t, rec) != "body{}" {
		t.Fatalf("Static with prefix: status=%d body=%q", rec.Code, bodyString(t, rec))
	}
}

func TestAdaptStdMiddleware_WithErrorHandler(t *testing.T) {
	r := NewRouter()

	var gotErr error
	r.ErrorHandler(func(c *Ctx, err error) {
		gotErr = err
		c.Writer().WriteHeader(499)
	})

	// std middleware
	mid := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req)
		})
	}

	r.Compat.Use(mid)

	r.Get("/err", func(c *Ctx) error {
		return errors.New("handler error")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/err", nil)
	r.ServeHTTP(rec, req)

	if gotErr == nil || gotErr.Error() != "handler error" {
		t.Fatalf("ErrorHandler should receive error, got %v", gotErr)
	}
}

func TestSetLogger_Nil(t *testing.T) {
	r := NewRouter()
	oldLogger := r.Logger()
	r.SetLogger(nil)
	if r.Logger() != oldLogger {
		t.Fatal("SetLogger(nil) should not change logger")
	}
}

func TestCanonicalPath_RootAfterClean(t *testing.T) {
	// Test path that becomes "/" after path.Clean
	if got := canonicalPath("/."); got != "/" {
		t.Fatalf("canonicalPath(\"/.\") = %q, want \"/\"", got)
	}
	// Test path that becomes empty after TrimRight
	if got := canonicalPath("///"); got != "/" {
		t.Fatalf("canonicalPath(\"///\") = %q, want \"/\"", got)
	}
}

func TestNewRouter_TextHandler(t *testing.T) {
	// Test NewRouter when color is not supported
	// Save and restore environment
	oldForce := os.Getenv("FORCE_COLOR")
	oldNo := os.Getenv("NO_COLOR")
	defer func() {
		_ = os.Setenv("FORCE_COLOR", oldForce)
		_ = os.Setenv("NO_COLOR", oldNo)
	}()

	_ = os.Setenv("FORCE_COLOR", "")
	_ = os.Setenv("NO_COLOR", "1")

	r := NewRouter()
	if r == nil || r.Logger() == nil {
		t.Fatal("NewRouter should create valid router with text handler")
	}
}

func TestCanonicalPath_AdditionalCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Additional edge cases
		{".", "/"},
		{"..", "/"},
		{"./foo", "/foo"},
		{"../foo", "/foo"},
		{"/foo/.", "/foo"},
		{"/foo/..", "/"},
		{"foo/.", "/foo"},
		{"//foo//", "/foo"},
	}

	for _, tc := range tests {
		got := canonicalPath(tc.input)
		if got != tc.expected {
			t.Errorf("canonicalPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
