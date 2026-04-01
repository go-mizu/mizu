// File: router_test.go
package mizu

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ---- helpers

func mustReq(t *testing.T, method, target string, body io.Reader) *http.Request {
	t.Helper()
	return httptest.NewRequest(method, target, body)
}

func ok(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func has(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Fatalf("expected substring %q in %q", sub, s)
	}
}

// mwTap records execution order.
func mwTap(name string, buf *[]string) Middleware {
	return func(next Handler) Handler {
		return func(c *Ctx) error {
			*buf = append(*buf, name)
			return next(c)
		}
	}
}

func mwHeader(key, val string) Middleware {
	return func(next Handler) Handler {
		return func(c *Ctx) error {
			c.Writer().Header().Set(key, val)
			return next(c)
		}
	}
}

func handlerText(s string) Handler {
	return func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte(s))
		return nil
	}
}

// ---- tests

func TestJoinPathAndCleanLeading(t *testing.T) {
	ok(t, cleanLeading(""), "/")
	ok(t, cleanLeading("x"), "/x")
	ok(t, cleanLeading("/x"), "/x")

	ok(t, joinPath("", ""), "/")
	ok(t, joinPath("", "/"), "/")
	ok(t, joinPath("/", "api"), "/api")
	ok(t, joinPath("/api", "v1"), "/api/v1")
	ok(t, joinPath("/api/", "/v1/"), "/api/v1")
	ok(t, joinPath("/api", "/"), "/api")
	ok(t, joinPath("/api", ""), "/api")
}

func TestFullPath(t *testing.T) {
	r := &Router{mux: http.NewServeMux()}
	ok(t, r.fullPath(""), "/")
	ok(t, r.fullPath("/"), "/")
	ok(t, r.fullPath("x"), "/x")

	r.base = "/api"
	ok(t, r.fullPath("/ping"), "/api/ping")
	ok(t, r.fullPath("ping"), "/api/ping")
	ok(t, r.fullPath("/"), "/api")
}

func TestServeHTTP_RunsGlobalChainAndRoutes(t *testing.T) {
	r := NewRouter()

	var order []string
	r.Use(mwTap("g1", &order), mwTap("g2", &order))

	r.Get("/ok", func(c *Ctx) error {
		order = append(order, "handler")
		c.Writer().WriteHeader(http.StatusOK)
		_, _ = c.Writer().Write([]byte("hi"))
		return nil
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/ok", nil))

	ok(t, rr.Code, http.StatusOK)
	ok(t, rr.Body.String(), "hi")

	joined := strings.Join(order, ",")
	has(t, joined, "g1")
	has(t, joined, "g2")
	has(t, joined, "handler")

	i1 := strings.Index(joined, "g1")
	i2 := strings.Index(joined, "g2")
	ih := strings.Index(joined, "handler")

	if i1 < 0 || i2 < 0 || ih < 0 || i1 >= ih || i2 >= ih {
		t.Fatalf("expected g1/g2 before handler, got %v", order)
	}
}

func TestHandle_MethodPatterns(t *testing.T) {
	r := NewRouter()

	r.Get("/same", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("GET"))
		return nil
	})
	r.Post("/same", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("POST"))
		return nil
	})

	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/same", nil))
		ok(t, rr.Code, http.StatusOK)
		ok(t, rr.Body.String(), "GET")
	}
	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodPost, "http://example/same", bytes.NewBufferString("x")))
		ok(t, rr.Code, http.StatusOK)
		ok(t, rr.Body.String(), "POST")
	}
}

func TestPrefix_Group_With_ScopedMiddleware(t *testing.T) {
	r := NewRouter()

	var got []string
	r.Use(mwTap("global", &got))

	api := r.Prefix("/api")
	api.Use(mwTap("global2", &got)) // affects only serving api directly, not root r

	apiV1 := api.With(mwTap("scoped", &got))
	apiV1.Get("/ping", func(c *Ctx) error {
		got = append(got, "handler")
		_, _ = c.Writer().Write([]byte("pong"))
		return nil
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/api/ping", nil))
	ok(t, rr.Code, http.StatusOK)
	ok(t, rr.Body.String(), "pong")

	joined := strings.Join(got, ",")
	has(t, joined, "global")
	has(t, joined, "scoped")
	has(t, joined, "handler")

	ig := strings.Index(joined, "global")
	is := strings.Index(joined, "scoped")
	ih := strings.Index(joined, "handler")

	if ig < 0 || is < 0 || ih < 0 || ig >= ih || is >= ih {
		t.Fatalf("unexpected order: %v", got)
	}
}

func TestErrorHandling_Default500(t *testing.T) {
	r := NewRouter()
	r.Get("/err", func(c *Ctx) error {
		return errors.New("boom")
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/err", nil))

	ok(t, rr.Code, http.StatusInternalServerError)
	has(t, rr.Body.String(), http.StatusText(http.StatusInternalServerError))
}

func TestErrorHandling_CustomErrorHandler(t *testing.T) {
	r := NewRouter()

	var called atomic.Bool
	r.ErrorHandler(func(c *Ctx, err error) {
		called.Store(true)
		c.Writer().WriteHeader(499)
		_, _ = c.Writer().Write([]byte("custom"))
	})

	r.Get("/err", func(c *Ctx) error { return errors.New("x") })

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/err", nil))

	if !called.Load() {
		t.Fatalf("expected error handler called")
	}
	ok(t, rr.Code, 499)
	ok(t, rr.Body.String(), "custom")
}

func TestPanicRecovery_Default500(t *testing.T) {
	r := NewRouter()
	r.Get("/panic", func(c *Ctx) error {
		panic("kaboom")
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/panic", nil))

	ok(t, rr.Code, http.StatusInternalServerError)
}

func TestPanicRecovery_CustomErrorHandlerReceivesPanicError(t *testing.T) {
	r := NewRouter()

	var saw atomic.Bool
	r.ErrorHandler(func(c *Ctx, err error) {
		var pe *PanicError
		if errors.As(err, &pe) && pe != nil && len(pe.Stack) > 0 {
			saw.Store(true)
		}
		c.Writer().WriteHeader(599)
	})

	r.Get("/panic", func(c *Ctx) error {
		panic("x")
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/panic", nil))

	if !saw.Load() {
		t.Fatalf("expected PanicError with stack")
	}
	ok(t, rr.Code, 599)
}

func TestStatic_ServesFiles_AndRedirects(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	r := NewRouter()
	r.Static("/assets", http.FS(os.DirFS(dir)))

	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/assets", nil))
		ok(t, rr.Code, http.StatusMovedPermanently)
		ok(t, rr.Header().Get("Location"), "/assets/")
	}
	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/assets/hello.txt", nil))
		ok(t, rr.Code, http.StatusOK)
		ok(t, rr.Body.String(), "hello")
	}
	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodHead, "http://example/assets/hello.txt", nil))
		ok(t, rr.Code, http.StatusOK)
		ok(t, rr.Body.String(), "")
	}
}

func TestCompat_Handle_Mount_AndHandleMethod(t *testing.T) {
	r := NewRouter()

	r.Compat.Handle("/plain", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Plain", "1")
		w.WriteHeader(204)
	}))

	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/plain", nil))
		ok(t, rr.Code, 204)
		ok(t, rr.Header().Get("X-Plain"), "1")
	}
	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodPost, "http://example/plain", nil))
		ok(t, rr.Code, 204)
		ok(t, rr.Header().Get("X-Plain"), "1")
	}

	r.Compat.HandleMethod(http.MethodPost, "/m", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(201)
	}))

	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodPost, "http://example/m", nil))
		ok(t, rr.Code, 201)
	}
	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/m", nil))
		ok(t, rr.Code, http.StatusMethodNotAllowed)
	}

	r.Compat.Mount("/mount", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(202)
	}))
	{
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/mount", nil))
		ok(t, rr.Code, 202)
	}
}

func TestCompat_Use_StdMiddleware_Bridge(t *testing.T) {
	r := NewRouter()

	stdMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Std", "1")
			next.ServeHTTP(w, req)
		})
	}

	r.Compat.Use(stdMW)

	r.Get("/ok", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("ok"))
		return nil
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/ok", nil))
	ok(t, rr.Code, http.StatusOK)
	ok(t, rr.Header().Get("X-Std"), "1")
	ok(t, rr.Body.String(), "ok")
}

func TestUseAndSetLogger_NonNil(t *testing.T) {
	r := NewRouter()
	if r.Logger() == nil {
		t.Fatalf("expected logger")
	}

	old := r.Logger()
	r.SetLogger(nil)
	if r.Logger() != old {
		t.Fatalf("expected logger unchanged")
	}
}

func TestStatic_RootPrefix(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("root"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	r := NewRouter()
	r.Static("/", http.FS(os.DirFS(dir)))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/", nil))

	ok(t, rr.Code, http.StatusOK)
	ok(t, rr.Body.String(), "root")
}

func TestScopedMiddleware_OnStatic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	r := NewRouter()
	r2 := r.With(mwHeader("X-Scoped", "1"))
	r2.Static("/s", http.FS(os.DirFS(dir)))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/s/a.txt", nil))
	ok(t, rr.Code, http.StatusOK)
	ok(t, rr.Header().Get("X-Scoped"), "1")
	ok(t, rr.Body.String(), "a")
}

func TestGroup_HTTPRouter(t *testing.T) {
	r := NewRouter()

	r.Compat.Group("/g", func(g *httpRouter) {
		g.Handle("/x", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(204)
		}))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/g/x", nil))
	ok(t, rr.Code, 204)
}

func TestGlobalMiddleware_SeesOriginalPath(t *testing.T) {
	r := NewRouter()

	var seen string
	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			seen = c.Request().URL.Path
			return next(c)
		}
	})

	r.Get("/x/", handlerText("ok"))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/x/", nil))

	ok(t, seen, "/x/")
}

func TestNoGlobalMiddleware_StillWorks(t *testing.T) {
	r := &Router{
		mux: http.NewServeMux(),
	}
	r.Compat = &httpRouter{r: r}

	r.Get("/ok", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("ok"))
		return nil
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/ok", nil))

	ok(t, rr.Code, http.StatusOK)
	ok(t, rr.Body.String(), "ok")
}

func TestTimeoutStyleStdMiddleware_DoesNotBreak(t *testing.T) {
	r := NewRouter()

	stdMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			time.Sleep(1 * time.Millisecond)
			next.ServeHTTP(w, req)
		})
	}

	r.Compat.Use(stdMW)

	r.Get("/ok", func(c *Ctx) error {
		_, _ = c.Writer().Write([]byte("ok"))
		return nil
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, mustReq(t, http.MethodGet, "http://example/ok", nil))
	ok(t, rr.Code, http.StatusOK)
	ok(t, rr.Body.String(), "ok")
}
