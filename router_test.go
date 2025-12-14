// File: router_test.go
package mizu

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestNewRouter_ColorAndPlainBranches(t *testing.T) {
	// Best-effort: these env vars only help if your forceColorOn()/supportsColorEnv()
	// consult them. Test ensures NewRouter doesn't panic in either configuration attempt.

	t.Setenv("NO_COLOR", "1")
	_ = NewRouter()

	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	_ = NewRouter()
}

func TestRouter_BasicRoutingAndErrors(t *testing.T) {
	r := NewRouter()

	r.Get("/ok", func(c *Ctx) error {
		return c.Text(200, "ok")
	})

	r.Get("/err", func(c *Ctx) error {
		return errors.New("boom")
	})

	r.Get("/panic", func(c *Ctx) error {
		panic("x")
	})

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != 200 || rr.Body.String() != "ok" {
			t.Fatalf("want 200 ok, got %d %q", rr.Code, rr.Body.String())
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/err", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != 500 {
			t.Fatalf("want 500, got %d", rr.Code)
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != 500 {
			t.Fatalf("want 500, got %d", rr.Code)
		}
	}
}

func TestRouter_HandleMethodEmptyDefaultsToGET(t *testing.T) {
	r := NewRouter()
	r.Handle("", "/m", func(c *Ctx) error { return c.Text(200, "m") })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/m", nil)
	r.ServeHTTP(rr, req)
	if rr.Code != 200 || rr.Body.String() != "m" {
		t.Fatalf("want 200 m, got %d %q", rr.Code, rr.Body.String())
	}
}

func TestRouter_GroupPrefixWithJoinPath(t *testing.T) {
	r := NewRouter()

	r.Group("/api", func(g *Router) {
		g.Get("ping", func(c *Ctx) error {
			return c.Text(200, "pong")
		})
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	r.ServeHTTP(rr, req)
	if rr.Code != 200 || rr.Body.String() != "pong" {
		t.Fatalf("want 200 pong, got %d %q", rr.Code, rr.Body.String())
	}

	// nil fn should be a no-op
	r.Group("/x", nil)
}

func TestRouter_WithAddsScopedMiddleware(t *testing.T) {
	r := NewRouter()

	seen := make([]string, 0, 2)
	mw1 := func(next Handler) Handler {
		return func(c *Ctx) error {
			seen = append(seen, "mw1")
			return next(c)
		}
	}
	mw2 := func(next Handler) Handler {
		return func(c *Ctx) error {
			seen = append(seen, "mw2")
			return next(c)
		}
	}

	r.Use(mw1)
	r.With(mw2).Get("/x", func(c *Ctx) error {
		return c.Text(200, "x")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if len(seen) != 2 || seen[0] != "mw1" || seen[1] != "mw2" {
		t.Fatalf("want [mw1 mw2], got %v", seen)
	}
}

func TestRouter_SetLoggerNilDoesNotPanic(t *testing.T) {
	r := NewRouter()
	r.SetLogger(nil)

	r.Get("/x", func(c *Ctx) error { return c.Text(200, "x") })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestRouter_handleErrDoesNotClobberIfAlreadyWroteHeader(t *testing.T) {
	r := NewRouter()

	r.Get("/partial", func(c *Ctx) error {
		_ = c.Text(200, "started")
		return errors.New("later")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/partial", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if rr.Body.String() != "started" {
		t.Fatalf("want body preserved, got %q", rr.Body.String())
	}
}

func TestRouter_Static_NetHTTPSemantics(t *testing.T) {
	r := NewRouter()

	fsys := http.FS(fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("hi")},
	})

	r.Static("/assets", fsys)

	// ServeMux redirects "/assets" -> "/assets/" for subtree patterns.
	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets", nil)
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusMovedPermanently && rr.Code != http.StatusPermanentRedirect {
			t.Fatalf("want redirect for /assets, got %d", rr.Code)
		}
		loc := rr.Header().Get("Location")
		if loc != "/assets/" {
			t.Fatalf("want Location /assets/, got %q", loc)
		}
	}

	// "/assets/hello.txt" should serve file.
	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/hello.txt", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != 200 || rr.Body.String() != "hi" {
			t.Fatalf("want 200 hi, got %d %q", rr.Code, rr.Body.String())
		}
	}

	// nil fsys should be a no-op and not panic.
	r.Static("/noop", nil)
}

func TestCompat_HandleAndMethodPatternCoexist(t *testing.T) {
	r := NewRouter()

	// Method-pattern route.
	r.Get("/same", func(c *Ctx) error { return c.Text(200, "mizu") })

	// Compat route on same path, should not conflict, and should not override GET.
	r.Compat.Handle("/same", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "compat")
	}))

	// GET should hit method-pattern.
	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/same", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != 200 || rr.Body.String() != "mizu" {
			t.Fatalf("want mizu for GET, got %d %q", rr.Code, rr.Body.String())
		}
	}

	// POST should hit compat handler (no method-pattern POST route exists).
	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/same", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != 200 || rr.Body.String() != "compat" {
			t.Fatalf("want compat for POST, got %d %q", rr.Code, rr.Body.String())
		}
	}
}

func TestCompat_HandleMethodWorks(t *testing.T) {
	r := NewRouter()

	r.Compat.HandleMethod("POST", "/p", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "posted")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/p", nil)
	r.ServeHTTP(rr, req)
	if rr.Code != 200 || rr.Body.String() != "posted" {
		t.Fatalf("want 200 posted, got %d %q", rr.Code, rr.Body.String())
	}
}

func TestCompat_UseStdMiddlewareMutatesRequestAndPropagatesToCtx(t *testing.T) {
	r := NewRouter()

	type key struct{}
	stdmw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(context.WithValue(req.Context(), key{}, "v"))
			next.ServeHTTP(w, req)
		})
	}

	r.Compat.Use(stdmw)

	r.Get("/ctx", func(c *Ctx) error {
		if got, _ := c.Request().Context().Value(key{}).(string); got != "v" {
			return c.Text(500, "missing")
		}
		return c.Text(200, "ok")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ctx", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != 200 || rr.Body.String() != "ok" {
		t.Fatalf("want 200 ok, got %d %q", rr.Code, rr.Body.String())
	}
}

func TestPathHelpers(t *testing.T) {
	r := NewRouter()

	if got := r.fullPath(""); got != "/" {
		t.Fatalf("want /, got %q", got)
	}
	if got := r.fullPath("a"); got != "/a" {
		t.Fatalf("want /a, got %q", got)
	}

	if got := joinPath("", "a"); got != "/a" {
		t.Fatalf("joinPath: want /a, got %q", got)
	}
	if got := joinPath("/", "a"); got != "/a" {
		t.Fatalf("joinPath: want /a, got %q", got)
	}
	if got := joinPath("/b", ""); got != "/b" {
		t.Fatalf("joinPath: want /b, got %q", got)
	}
	if got := joinPath("/b", "/"); got != "/b" {
		t.Fatalf("joinPath: want /b, got %q", got)
	}
	if got := joinPath("/b", "c"); got != "/b/c" {
		t.Fatalf("joinPath: want /b/c, got %q", got)
	}

	if got := cleanLeading(""); got != "/" {
		t.Fatalf("cleanLeading: want /, got %q", got)
	}
	if got := cleanLeading("x"); got != "/x" {
		t.Fatalf("cleanLeading: want /x, got %q", got)
	}
	if got := cleanLeading("/x"); got != "/x" {
		t.Fatalf("cleanLeading: want /x, got %q", got)
	}
}

func TestCompat_PrefixAndGroupNoPanic(t *testing.T) {
	r := NewRouter()

	cp := r.Compat.Prefix("/v1")
	cp.HandleMethod("GET", "/a", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "a")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/a", nil)
	r.ServeHTTP(rr, req)
	if rr.Code != 200 || rr.Body.String() != "a" {
		t.Fatalf("want 200 a, got %d %q", rr.Code, rr.Body.String())
	}

	r.Compat.Group("/g", nil)
}
