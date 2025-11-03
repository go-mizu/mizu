package mizu

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewRouterAndNotFound(t *testing.T) {
	r := NewRouter()
	if r == nil {
		t.Fatal("NewRouter returned nil")
	}
	if r.Logger() == nil {
		t.Fatal("Logger should not be nil")
	}
	if r.Compat == nil {
		t.Fatal("Compat should not be nil")
	}

	// No routes registered yet, expect 404 from default NotFoundCore
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Not Found") {
		t.Fatalf("expected Not Found body, got %q", rec.Body.String())
	}
}

func TestMethodHelpersAndHandle(t *testing.T) {
	r := NewRouter()

	// Register one route per helper
	r.Get("/m-get", func(c *Ctx) error {
		return c.Text(http.StatusOK, "GET")
	})
	r.Head("/m-head", func(c *Ctx) error {
		return c.Text(http.StatusOK, "HEAD")
	})
	r.Post("/m-post", func(c *Ctx) error {
		return c.Text(http.StatusOK, "POST")
	})
	r.Put("/m-put", func(c *Ctx) error {
		return c.Text(http.StatusOK, "PUT")
	})
	r.Patch("/m-patch", func(c *Ctx) error {
		return c.Text(http.StatusOK, "PATCH")
	})
	r.Delete("/m-delete", func(c *Ctx) error {
		return c.Text(http.StatusOK, "DELETE")
	})
	r.Connect("/m-connect", func(c *Ctx) error {
		return c.Text(http.StatusOK, "CONNECT")
	})
	r.Trace("/m-trace", func(c *Ctx) error {
		return c.Text(http.StatusOK, "TRACE")
	})

	// Handle should upper case the method name
	r.Handle("post", "/m-handle", func(c *Ctx) error {
		return c.Text(http.StatusOK, "HANDLE-POST")
	})

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"get", http.MethodGet, "/m-get", http.StatusOK, "GET"},
		{"head", http.MethodHead, "/m-head", http.StatusOK, "HEAD"},
		{"post", http.MethodPost, "/m-post", http.StatusOK, "POST"},
		{"put", http.MethodPut, "/m-put", http.StatusOK, "PUT"},
		{"patch", http.MethodPatch, "/m-patch", http.StatusOK, "PATCH"},
		{"delete", http.MethodDelete, "/m-delete", http.StatusOK, "DELETE"},
		{"connect", http.MethodConnect, "/m-connect", http.StatusOK, "CONNECT"},
		{"trace", http.MethodTrace, "/m-trace", http.StatusOK, "TRACE"},
		{"handle-post", http.MethodPost, "/m-handle", http.StatusOK, "HANDLE-POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			// For HEAD some frameworks strip the body, but our handler writes it,
			// so we assert the same body text for all methods.
			if strings.TrimSpace(rec.Body.String()) != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, rec.Body.String())
			}
		})
	}
}

func TestGetPostAndAllowHeader(t *testing.T) {
	r := NewRouter()

	r.Get("/ok", func(c *Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// GET /ok
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Host = "example.com"

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body ok, got %q", rec.Body.String())
	}

	// HEAD /ok is routed to the same handler.
	// This router does not strip the body for HEAD, so we only assert the status.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodHead, "/ok", nil)
	req2.RemoteAddr = "192.0.2.1:1234"
	req2.Host = "example.com"

	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 for HEAD /ok, got %d", rec2.Code)
	}

	// allowHeader on a path with GET should include GET, HEAD, OPTIONS
	hdr := r.allowHeader("/ok")
	if !strings.Contains(hdr, http.MethodGet) ||
		!strings.Contains(hdr, http.MethodHead) ||
		!strings.Contains(hdr, http.MethodOptions) {
		t.Fatalf("allowHeader(/ok) missing expected methods, got %q", hdr)
	}

	// allowHeader on unknown path defaults to OPTIONS only
	hdr2 := r.allowHeader("/no-such-path")
	if hdr2 != http.MethodOptions {
		t.Fatalf("expected OPTIONS for unknown path, got %q", hdr2)
	}
}

func TestMethodNotAllowed405(t *testing.T) {
	r := NewRouter()

	r.Post("/onlypost", func(c *Ctx) error {
		return c.Text(http.StatusCreated, "created")
	})

	// POST should work
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/onlypost", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	// GET should trigger 405 since POST is registered for that path
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/onlypost", nil)
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET /onlypost, got %d", rec2.Code)
	}
	allow := rec2.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodPost) || !strings.Contains(allow, http.MethodOptions) {
		t.Fatalf("Allow header for /onlypost missing methods, got %q", allow)
	}
}

func TestAllowHeaderWithOptionsPresent(t *testing.T) {
	r := NewRouter()
	// manually add OPTIONS to set, should not duplicate
	r.allowAdd("/path", http.MethodOptions)
	r.allowAdd("/path", http.MethodGet)

	hdr := r.allowHeader("/path")
	if strings.Count(hdr, http.MethodOptions) != 1 {
		t.Fatalf("expected OPTIONS only once, got %q", hdr)
	}
}

func TestUseAndUseFirstOrder(t *testing.T) {
	r := NewRouter()
	var calls []string

	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			calls = append(calls, "mid2-before")
			err := next(c)
			calls = append(calls, "mid2-after")
			return err
		}
	})

	r.UseFirst(func(next Handler) Handler {
		return func(c *Ctx) error {
			calls = append(calls, "mid1-before")
			err := next(c)
			calls = append(calls, "mid1-after")
			return err
		}
	})

	r.Get("/order", func(c *Ctx) error {
		calls = append(calls, "handler")
		return c.Text(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/order", nil)
	r.ServeHTTP(rec, req)

	want := []string{
		"mid1-before",
		"mid2-before",
		"handler",
		"mid2-after",
		"mid1-after",
	}
	if len(calls) != len(want) {
		t.Fatalf("expected %d calls, got %d (%v)", len(want), len(calls), calls)
	}
	for i, v := range want {
		if calls[i] != v {
			t.Fatalf("at %d expected %q, got %q", i, v, calls[i])
		}
	}
}

func TestErrorHandlerForErrorAndPanic(t *testing.T) {
	r := NewRouter()

	var seenError error
	var seenPanic *PanicError

	r.ErrorHandler(func(c *Ctx, err error) {
		switch e := err.(type) {
		case *PanicError:
			seenPanic = e
			_ = c.Text(590, "panic:"+fmtAny(e.Value))
		default:
			seenError = e
			_ = c.Text(580, "err:"+e.Error())
		}
	})

	r.Get("/err", func(c *Ctx) error {
		return errors.New("boom")
	})
	r.Get("/panic", func(c *Ctx) error {
		panic("ohno")
	})

	// error case
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/err", nil)
	r.ServeHTTP(rec1, req1)

	if rec1.Code != 580 {
		t.Fatalf("expected status 580, got %d", rec1.Code)
	}
	if seenError == nil || seenError.Error() != "boom" {
		t.Fatalf("expected seenError boom, got %#v", seenError)
	}
	if !strings.HasPrefix(rec1.Body.String(), "err:boom") {
		t.Fatalf("unexpected error body %q", rec1.Body.String())
	}

	// panic case
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(rec2, req2)

	if rec2.Code != 590 {
		t.Fatalf("expected status 590, got %d", rec2.Code)
	}
	if seenPanic == nil || fmtAny(seenPanic.Value) != "ohno" {
		t.Fatalf("expected seenPanic value ohno, got %#v", seenPanic)
	}
	if !strings.HasPrefix(rec2.Body.String(), "panic:") {
		t.Fatalf("unexpected panic body %q", rec2.Body.String())
	}
}

func fmtAny(v any) string {
	return strings.TrimSpace(strings.Trim(fmt.Sprintf("%v", v), "\n"))
}

func TestErrorBranchWithoutErrorHandler(t *testing.T) {
	r := NewRouter()
	// remove logger to exercise r.log == nil branch
	r.SetLogger(nil)

	r.Get("/err500", func(c *Ctx) error {
		return errors.New("fail")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/err500", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestPanicBranchWithoutErrorHandler(t *testing.T) {
	r := NewRouter()
	// remove logger to exercise r.log == nil path in panic branch
	r.SetLogger(nil)

	r.Get("/panic500", func(c *Ctx) error {
		panic("panic in handler")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic500", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestStaticServesFilesAndUsesMiddleware(t *testing.T) {
	r := NewRouter()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello static"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	// attach simple header middleware to show Static goes through chain
	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			c.Writer().Header().Set("X-Mizu-Static", "true")
			return next(c)
		}
	})

	r.Static("/assets/", http.Dir(dir))

	// should serve file
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/hello.txt", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "hello static" {
		t.Fatalf("unexpected body %q", body)
	}
	if rec.Header().Get("X-Mizu-Static") != "true" {
		t.Fatalf("expected header X-Mizu-Static=true")
	}
}

func TestPrefixAndGroupRouting(t *testing.T) {
	r := NewRouter()

	api := r.Prefix("/api/")
	api.Get("v1", func(c *Ctx) error {
		return c.Text(http.StatusOK, "api-v1")
	})

	r.Group("/g", func(g *Router) {
		g.Get("/h", func(c *Ctx) error {
			return c.Text(http.StatusOK, "group-h")
		})
	})

	// /api/v1
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK || rec1.Body.String() != "api-v1" {
		t.Fatalf("unexpected response for /api/v1: %d %q", rec1.Code, rec1.Body.String())
	}

	// /g/h
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/g/h", nil)
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || rec2.Body.String() != "group-h" {
		t.Fatalf("unexpected response for /g/h: %d %q", rec2.Code, rec2.Body.String())
	}
}

func TestCompatHandleRootCatchAll(t *testing.T) {
	r := NewRouter()

	// custom root handler via Compat
	r.Compat.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(299)
		_, _ = io.WriteString(w, "root-compat")
	}))

	// request to "/" should be served by Compat handler
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 299 {
		t.Fatalf("expected 299 for Compat root, got %d", rec.Code)
	}
	if rec.Body.String() != "root-compat" {
		t.Fatalf("unexpected root body %q", rec.Body.String())
	}

	// request to /missing is also matched by "/" pattern in ServeMux
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/missing", nil)
	r.ServeHTTP(rec2, req2)
	if rec2.Code != 299 {
		t.Fatalf("expected 299 for /missing, got %d", rec2.Code)
	}
	if rec2.Body.String() != "root-compat" {
		t.Fatalf("unexpected /missing body %q", rec2.Body.String())
	}
}

func TestCompatHandleMethod(t *testing.T) {
	r := NewRouter()

	r.Compat.HandleMethod(http.MethodGet, "/compat", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(207)
		_, _ = io.WriteString(w, "compat-method")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/compat", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 207 {
		t.Fatalf("expected 207, got %d", rec.Code)
	}
	if rec.Body.String() != "compat-method" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestAdaptStdMiddlewareViaCompatUse(t *testing.T) {
	r := NewRouter()

	// std middleware that sets a header
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Std", "yes")
			next.ServeHTTP(w, req)
		})
	}

	r.Compat.Use(mw)

	r.Get("/mw", func(c *Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mw", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Std") != "yes" {
		t.Fatalf("expected X-Std=yes, got %q", rec.Header().Get("X-Std"))
	}
}

func TestNotFoundOverride(t *testing.T) {
	r := NewRouter()

	r.NotFound(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(470)
		_, _ = io.WriteString(w, "custom-notfound")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing-custom", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 470 {
		t.Fatalf("expected 470, got %d", rec.Code)
	}
	if rec.Body.String() != "custom-notfound" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestFullPathAndJoinPathVariants(t *testing.T) {
	r := NewRouter()

	// simple route at root
	r.Get("/", func(c *Ctx) error {
		return c.Text(http.StatusOK, "root")
	})

	// nested prefixes exercise joinPath branches
	p1 := r.Prefix("/api")
	p2 := p1.Prefix("/v1/")
	p2.Get("items/", func(c *Ctx) error {
		return c.Text(http.StatusOK, "items")
	})

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 for root, got %d", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/items/", nil)
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/items/, got %d", rec2.Code)
	}
	if rec2.Body.String() != "items" {
		t.Fatalf("unexpected body %q", rec2.Body.String())
	}
}

func TestSetLoggerOverridesDefault(t *testing.T) {
	base := NewRouter()
	if base.Logger() == nil {
		t.Fatal("default logger should not be nil")
	}

	lg := slog.New(slog.NewTextHandler(io.Discard, nil))
	base.SetLogger(lg)
	if base.Logger() != lg {
		t.Fatalf("expected SetLogger to replace logger")
	}
}

func TestMountUsesCompat(t *testing.T) {
	r := NewRouter()

	r.Mount("/mount", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(234)
		_, _ = io.WriteString(w, "mounted")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mount", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 234 {
		t.Fatalf("expected 234, got %d", rec.Code)
	}
	if rec.Body.String() != "mounted" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

// Simple smoke test that calls ServeHTTP more than once to make sure
// ensureNotFoundBound is idempotent and does not panic.
func TestServeHTTPNotFoundIdempotent(t *testing.T) {
	r := NewRouter()

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/nothing-here", nil)
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("iteration %d expected 404, got %d", i, rec.Code)
		}
	}
}

// Small helper to give time for logging paths to be hit
func TestLoggingDoesNotCrash(t *testing.T) {
	r := NewRouter()
	r.Get("/log", func(c *Ctx) error {
		return c.Text(http.StatusOK, "log")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// give time for logging side effects if any
	time.Sleep(10 * time.Millisecond)
}

func TestAdaptStdMiddlewareWithErrorHandler(t *testing.T) {
	r := NewRouter()

	var seenErr error

	// Custom error handler that writes a special status and body
	r.ErrorHandler(func(c *Ctx, err error) {
		seenErr = err
		_ = c.Text(599, "stdmw:"+err.Error())
	})

	// Standard middleware that just calls the next handler
	sm := func(next http.Handler) http.Handler {
		return next
	}

	mw := r.adaptStdMiddleware(sm)

	// Next handler returns an error, which should be sent to ErrorHandler
	h := mw(func(c *Ctx) error {
		return errors.New("boom-stdmw")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stdmw", nil)

	// Wrap the Mizu Handler with adapt to run it as http.Handler
	r.adapt(h).ServeHTTP(rec, req)

	if rec.Code != 599 {
		t.Fatalf("expected status 599, got %d", rec.Code)
	}
	if seenErr == nil || seenErr.Error() != "boom-stdmw" {
		t.Fatalf("expected seenErr boom-stdmw, got %#v", seenErr)
	}
	if !strings.HasPrefix(rec.Body.String(), "stdmw:boom-stdmw") {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestAdaptStdMiddlewareErrorNoErrorHandler(t *testing.T) {
	r := NewRouter()

	// No ErrorHandler, and logger cleared to hit the r.log == nil branch
	r.ErrorHandler(nil)
	r.SetLogger(nil)

	sm := func(next http.Handler) http.Handler {
		return next
	}

	mw := r.adaptStdMiddleware(sm)

	h := mw(func(c *Ctx) error {
		return errors.New("oops-stdmw")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stdmw500", nil)

	r.adapt(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), http.StatusText(http.StatusInternalServerError)) {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestHTTPRouterPrefix(t *testing.T) {
	r := NewRouter()

	// Create a compat router prefixed at /api
	api := r.Compat.Prefix("/api")

	// Register a handler at /v1 on the prefixed compat router
	api.Handle("/v1", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(210)
		_, _ = io.WriteString(w, "api-prefix-v1")
	}))

	// /api/v1 should hit the compat-prefixed handler
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 210 {
		t.Fatalf("expected 210 for /api/v1, got %d", rec.Code)
	}
	if rec.Body.String() != "api-prefix-v1" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}

	// /v1 without prefix should not match
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/v1", nil)
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for /v1, got %d", rec2.Code)
	}
}

func TestHTTPRouterGroup(t *testing.T) {
	r := NewRouter()

	// Group under /grp using compat facade
	r.Compat.Group("/grp", func(g *httpRouter) {
		g.Handle("/hello", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(211)
			_, _ = io.WriteString(w, "group-hello")
		}))
	})

	// /grp/hello should be served by the grouped handler
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/grp/hello", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 211 {
		t.Fatalf("expected 211 for /grp/hello, got %d", rec.Code)
	}
	if rec.Body.String() != "group-hello" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestHTTPRouterMount(t *testing.T) {
	r := NewRouter()

	// Mount should behave exactly like Handle but via the Mount helper
	r.Compat.Mount("/mount-compat", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(212)
		_, _ = io.WriteString(w, "mounted-compat")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mount-compat", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 212 {
		t.Fatalf("expected 212 for /mount-compat, got %d", rec.Code)
	}
	if rec.Body.String() != "mounted-compat" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}
