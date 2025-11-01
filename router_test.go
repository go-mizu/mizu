package mizu

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func doReq(t *testing.T, h http.Handler, method, target string, body io.Reader, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestNewRouterAndSimpleServe(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", func(c *Ctx) error {
		c.Response().WriteHeader(http.StatusOK)
		_, _ = c.Response().Write([]byte("ok"))
		return nil
	})

	rr := doReq(t, r, http.MethodGet, "/hi", nil, nil)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Fatalf("expected 200 ok, got %d %q", rr.Code, rr.Body.String())
	}
}

func TestNotFoundDefaultAndLazyBind(t *testing.T) {
	r := NewRouter()

	// Add one middleware before first request. It should wrap NotFound.
	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			c.Response().Header().Add("X-Pre", "1")
			return next(c)
		}
	})

	// First 404 binds NotFound with current chain.
	rr := doReq(t, r, http.MethodGet, "/nope", nil, nil)
	if rr.Code != 404 {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if rr.Header().Get("X-Pre") != "1" {
		t.Fatalf("expected X-Pre=1 on first 404")
	}

	// Add another middleware after first request. It should NOT affect NotFound anymore.
	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			c.Response().Header().Add("X-Late", "1")
			return next(c)
		}
	})
	rr2 := doReq(t, r, http.MethodGet, "/still-nope", nil, nil)
	if rr2.Header().Get("X-Late") != "" {
		t.Fatalf("did not expect X-Late on 404 after first bind")
	}
}

func TestUseFirstOrder(t *testing.T) {
	r := NewRouter()

	// Outer first
	r.UseFirst(func(next Handler) Handler {
		return func(c *Ctx) error {
			seq := c.Response().Header().Get("X-Seq")
			if seq != "" {
				seq += ","
			}
			c.Response().Header().Set("X-Seq", seq+"A")
			return next(c)
		}
	})

	// Inner later
	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			seq := c.Response().Header().Get("X-Seq")
			if seq != "" {
				seq += ","
			}
			c.Response().Header().Set("X-Seq", seq+"B")
			return next(c)
		}
	})

	r.Get("/order", func(c *Ctx) error {
		seq := c.Response().Header().Get("X-Seq")
		if seq != "" {
			seq += ","
		}
		c.Response().Header().Set("X-Seq", seq+"H")
		c.Response().WriteHeader(200)
		return nil
	})

	rr := doReq(t, r, http.MethodGet, "/order", nil, nil)
	if got := rr.Header().Get("X-Seq"); got != "A,B,H" {
		t.Fatalf("sequence mismatch, want A,B,H got %q", got)
	}
}

func TestPrefixAndGroup(t *testing.T) {
	r := NewRouter()
	r.Group("/api", func(g *Router) {
		g.Get("/v1/ping", func(c *Ctx) error {
			c.Response().WriteHeader(201)
			_, _ = c.Response().Write([]byte("pong"))
			return nil
		})
	})

	rr := doReq(t, r, http.MethodGet, "/api/v1/ping", nil, nil)
	if rr.Code != 201 || strings.TrimSpace(rr.Body.String()) != "pong" {
		t.Fatalf("unexpected response %d %q", rr.Code, rr.Body.String())
	}
}

func Test405GuardAndAllowHeader(t *testing.T) {
	r := NewRouter()
	r.Post("/things", func(c *Ctx) error { c.Response().WriteHeader(204); return nil })

	// Wrong method -> 405 and Allow includes OPTIONS and POST
	rr := doReq(t, r, http.MethodGet, "/things", nil, nil)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	allow := strings.Split(rr.Header().Get("Allow"), ", ")
	wantHas := map[string]bool{"POST": false, "OPTIONS": false}
	for _, m := range allow {
		if _, ok := wantHas[m]; ok {
			wantHas[m] = true
		}
	}
	for k, ok := range wantHas {
		if !ok {
			t.Fatalf("Allow header missing %s in %v", k, allow)
		}
	}

	// Register GET, then wrong method should show GET and HEAD and OPTIONS
	r.Get("/a", func(c *Ctx) error { c.Response().WriteHeader(200); return nil })
	rr2 := doReq(t, r, http.MethodPost, "/a", nil, nil)
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr2.Code)
	}
	allow2 := strings.Split(rr2.Header().Get("Allow"), ", ")
	want2 := map[string]bool{"GET": false, "HEAD": false, "OPTIONS": false}
	for _, m := range allow2 {
		if _, ok := want2[m]; ok {
			want2[m] = true
		}
	}
	for k, ok := range want2 {
		if !ok {
			t.Fatalf("Allow header missing %s in %v", k, allow2)
		}
	}
}

func TestAllowHeaderWhenPathUnknown(t *testing.T) {
	r := NewRouter()
	// No routes at /x yet
	if got := r.allowHeader("/x"); got != "OPTIONS" {
		t.Fatalf("expected OPTIONS for unknown path, got %q", got)
	}
}

func TestStaticServesFiles(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(fp, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewRouter()
	r.Static("/assets/", http.Dir(dir))

	rr := doReq(t, r, http.MethodGet, "/assets/hello.txt", nil, nil)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "hello" {
		t.Fatalf("expected file content, got %d %q", rr.Code, rr.Body.String())
	}
}

/*
func TestStaticWithEmbedFS(t *testing.T) {
	fs := fstest.MapFS{
		"public/hi.txt": {Data: []byte("ok-embed")},
	}
	r := NewRouter()
	r.Static("/s/", http.FS(fs))

	rr := doReq(t, r, http.MethodGet, "/s/public/hi.txt", nil, nil)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "ok-embed" {
		t.Fatalf("expected ok-embed, got %d %q", rr.Code, rr.Body.String())
	}
}
*/

func TestMount(t *testing.T) {
	r := NewRouter()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(202)
		_, _ = w.Write([]byte("mounted"))
	})
	r.Mount("/h", h)

	rr := doReq(t, r, http.MethodGet, "/h", nil, nil)
	if rr.Code != 202 || strings.TrimSpace(rr.Body.String()) != "mounted" {
		t.Fatalf("unexpected mount response %d %q", rr.Code, rr.Body.String())
	}
}

func TestErrorHandlerOnReturnedError(t *testing.T) {
	r := NewRouter()
	r.ErrorHandler(func(c *Ctx, err error) {
		c.Response().WriteHeader(418)
		_, _ = c.Response().Write([]byte("teapot"))
	})
	r.Get("/e", func(c *Ctx) error { return errors.New("boom") })

	rr := doReq(t, r, http.MethodGet, "/e", nil, nil)
	if rr.Code != 418 || strings.TrimSpace(rr.Body.String()) != "teapot" {
		t.Fatalf("unexpected error handler response %d %q", rr.Code, rr.Body.String())
	}
}

func TestPanicRecoveryDefault500(t *testing.T) {
	r := NewRouter()
	r.Get("/p", func(c *Ctx) error {
		panic("kaboom")
	})
	rr := doReq(t, r, http.MethodGet, "/p", nil, nil)
	if rr.Code != 500 {
		t.Fatalf("expected 500 on panic, got %d", rr.Code)
	}
}

func TestPanicHandledByErrorHandler(t *testing.T) {
	r := NewRouter()
	r.ErrorHandler(func(c *Ctx, err error) {
		c.Response().WriteHeader(599)
		_, _ = fmt.Fprintf(c.Response(), "handled:%T", err)
	})
	r.Get("/p2", func(c *Ctx) error {
		panic("crash")
	})
	rr := doReq(t, r, http.MethodGet, "/p2", nil, nil)
	if rr.Code != 599 || !strings.HasPrefix(rr.Body.String(), "handled:") {
		t.Fatalf("unexpected handler panic handling %d %q", rr.Code, rr.Body.String())
	}
}

/*
func TestCompatUseAndHandleMethod(t *testing.T) {
	r := NewRouter()

	// std middleware that stamps a header
	stdMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Std", "1")
			next.ServeHTTP(w, req)
		})
	}
	r.Compat.Use(stdMW)

	// Only allow POST on /compat
	r.Compat.HandleMethod("POST", "/compat", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))

	// Wrong method -> 405 and header from std middleware should still appear on 405 guard
	rr := doReq(t, r, http.MethodGet, "/compat", nil, nil)
	if rr.Code != 405 {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	// Allow header sanity
	if rr.Header().Get("Allow") == "" {
		t.Fatalf("expected Allow header on 405")
	}

	// Correct method -> 200 and std header present
	rr2 := doReq(t, r, http.MethodPost, "/compat", nil, nil)
	if rr2.Code != 200 || rr2.Header().Get("X-Std") != "1" || strings.TrimSpace(rr2.Body.String()) != "ok" {
		t.Fatalf("unexpected compat response %d hdr=%q body=%q", rr2.Code, rr2.Header().Get("X-Std"), rr2.Body.String())
	}
}
*/

func TestPrefixChainingJoins(t *testing.T) {
	r := NewRouter()
	api := r.Prefix("/api/")
	v1 := api.Prefix("v1")
	v1.Get("ping", func(c *Ctx) error {
		c.Response().WriteHeader(200)
		_, _ = c.Response().Write([]byte("pong"))
		return nil
	})

	rr := doReq(t, r, http.MethodGet, "/api/v1/ping", nil, nil)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "pong" {
		t.Fatalf("unexpected response %d %q", rr.Code, rr.Body.String())
	}
}

func TestConnectAndTrace(t *testing.T) {
	r := NewRouter()
	var got []string
	var mu sync.Mutex

	r.Connect("/tunnel", func(c *Ctx) error {
		mu.Lock()
		got = append(got, "CONNECT")
		mu.Unlock()
		c.Response().WriteHeader(200)
		_, _ = c.Response().Write([]byte("tunnel"))
		return nil
	})
	r.Trace("/trace", func(c *Ctx) error {
		mu.Lock()
		got = append(got, "TRACE")
		mu.Unlock()
		c.Response().WriteHeader(200)
		_, _ = c.Response().Write([]byte("trace"))
		return nil
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	req1, _ := http.NewRequest(http.MethodConnect, srv.URL+"/tunnel", nil)
	res1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatal(err)
	}
	b1, _ := io.ReadAll(res1.Body)
	_ = res1.Body.Close()
	if res1.StatusCode != 200 || strings.TrimSpace(string(b1)) != "tunnel" {
		t.Fatalf("CONNECT failed %d %q", res1.StatusCode, string(b1))
	}

	req2, _ := http.NewRequest(http.MethodTrace, srv.URL+"/trace", nil)
	res2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := io.ReadAll(res2.Body)
	_ = res2.Body.Close()
	if res2.StatusCode != 200 || strings.TrimSpace(string(b2)) != "trace" {
		t.Fatalf("TRACE failed %d %q", res2.StatusCode, string(b2))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "CONNECT" || got[1] != "TRACE" {
		t.Fatalf("unexpected method order %v", got)
	}
}

func TestServeHTTPBindsNotFoundOnce(t *testing.T) {
	r := NewRouter()

	// Force first bind
	_ = doReq(t, r, http.MethodGet, "/missing", nil, nil)

	// After first bind, add middleware that would alter NotFound if it could
	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			c.Response().Header().Set("X-After", "1")
			return next(c)
		}
	})

	rr := doReq(t, r, http.MethodGet, "/still-missing", nil, nil)
	if rr.Header().Get("X-After") != "" {
		t.Fatalf("NotFound should be bound once and not see later middleware")
	}
}

func TestHandleRegisterHeadAlongsideGet(t *testing.T) {
	r := NewRouter()
	r.Get("/x", func(c *Ctx) error { c.Response().WriteHeader(200); return nil })
	// Ask for HEAD explicitly to verify it is allowed
	rr := doReq(t, r, http.MethodHead, "/x", nil, nil)
	if rr.Code != 200 {
		t.Fatalf("HEAD on GET route should be 200, got %d", rr.Code)
	}
}

func TestErrorHandlerWritesOnlyOnceIfHandlerAlreadyWrote(t *testing.T) {
	r := NewRouter()
	r.ErrorHandler(func(c *Ctx, err error) {
		// Try to write again. Router should avoid double write by relying on Ctx.WroteHeader().
		c.Response().WriteHeader(500)
		_, _ = c.Response().Write([]byte("second"))
	})

	r.Get("/already", func(c *Ctx) error {
		c.Response().WriteHeader(204)
		_, _ = c.Response().Write([]byte("first"))
		return errors.New("fail after write")
	})

	rr := doReq(t, r, http.MethodGet, "/already", nil, nil)
	// We cannot assert exact body contents here since first write wins.
	if rr.Code != 204 {
		t.Fatalf("expected first write 204 to win, got %d", rr.Code)
	}
}

func TestCompatHandle(t *testing.T) {
	r := NewRouter()
	buf := new(bytes.Buffer)

	r.Compat.Handle("/h", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(buf, "hit")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))

	rr := doReq(t, r, http.MethodGet, "/h", nil, nil)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "ok" || strings.TrimSpace(buf.String()) != "hit" {
		t.Fatalf("compat handle failed %d %q buf=%q", rr.Code, rr.Body.String(), buf.String())
	}
}

func TestCompatPrefixAndGroup(t *testing.T) {
	r := NewRouter()
	r.Compat.Group("/c", func(g *httpRouter) {
		g.Handle("/pong", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(200)
			_, _ = w.Write([]byte("pong"))
		}))
	})
	rr := doReq(t, r, http.MethodGet, "/c/pong", nil, nil)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "pong" {
		t.Fatalf("compat group failed %d %q", rr.Code, rr.Body.String())
	}
}

// Dev is part of the public surface. Exercise the branch safely.
// This assumes the package provides supportsColor and Logger middleware.
func TestDevNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	r := NewRouter()
	r.Dev(true) // should not panic and should install logger middleware
	// Quick smoke request to ensure router still handles traffic
	r.Get("/dev", func(c *Ctx) error { c.Response().WriteHeader(200); _, _ = c.Response().Write([]byte("ok")); return nil })
	rr := doReq(t, r, http.MethodGet, "/dev", nil, nil)
	if rr.Code != 200 {
		t.Fatalf("dev route failed: %d", rr.Code)
	}
}

// Ensure that calling ServeHTTP without any routes still works through NotFound.
func TestServeHTTPNotFoundPathOnly(t *testing.T) {
	r := NewRouter()
	rr := doReq(t, r, http.MethodGet, "/", nil, nil)
	if rr.Code != 404 {
		t.Fatalf("expected 404 at root when no handlers, got %d", rr.Code)
	}
}

// Verify that standard net/http middleware can be adapted through Compat.Use.
func TestCompatUseMultiple(t *testing.T) {
	r := NewRouter()

	hdrOrder := make([]string, 0, 2)
	var mu sync.Mutex
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			mu.Lock()
			hdrOrder = append(hdrOrder, "mw1")
			mu.Unlock()
			next.ServeHTTP(w, req)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			mu.Lock()
			hdrOrder = append(hdrOrder, "mw2")
			mu.Unlock()
			next.ServeHTTP(w, req)
		})
	}

	r.Compat.Use(mw1, mw2)
	r.Get("/chain", func(c *Ctx) error { c.Response().WriteHeader(200); return nil })

	rr := doReq(t, r, http.MethodGet, "/chain", nil, nil)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(hdrOrder) != 2 || hdrOrder[0] != "mw1" || hdrOrder[1] != "mw2" {
		t.Fatalf("unexpected middleware order %v", hdrOrder)
	}
}

func TestHandleExplicitMethod(t *testing.T) {
	r := NewRouter()
	r.Handle("POST", "/upload", func(c *Ctx) error { c.Response().WriteHeader(201); return nil })

	// POST works
	rr := doReq(t, r, http.MethodPost, "/upload", nil, nil)
	if rr.Code != 201 {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	// GET should 405 with Allow including POST and OPTIONS
	rr2 := doReq(t, r, http.MethodGet, "/upload", nil, nil)
	if rr2.Code != 405 {
		t.Fatalf("expected 405, got %d", rr2.Code)
	}
	allow := rr2.Header().Get("Allow")
	if !strings.Contains(allow, "POST") || !strings.Contains(allow, "OPTIONS") {
		t.Fatalf("Allow header missing required methods: %q", allow)
	}
}

func TestSingleCtxByBehavior(t *testing.T) {
	r := NewRouter()
	var seen int
	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			seen++
			return next(c)
		}
	})
	r.Get("/one", func(c *Ctx) error {
		c.Response().WriteHeader(200)
		return nil
	})
	rr := doReq(t, r, http.MethodGet, "/one", nil, nil)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if seen != 1 {
		t.Fatalf("middleware should run once, got %d", seen)
	}
}

// Ensure ErrorHandler can inspect context values if needed.
func TestErrorHandlerReceivesCtx(t *testing.T) {
	r := NewRouter()
	r.ErrorHandler(func(c *Ctx, err error) {
		if c.Request() == nil || c.Response() == nil {
			t.Fatalf("ctx not initialized in error handler")
		}
		c.Response().WriteHeader(500)
	})
	r.Get("/err", func(c *Ctx) error { return errors.New("x") })
	rr := doReq(t, r, http.MethodGet, "/err", nil, nil)
	if rr.Code != 500 {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// Verify timeouts or context cancellation can propagate through middlewares.
func TestMiddlewareContextPropagation(t *testing.T) {
	r := NewRouter()
	r.Use(func(next Handler) Handler {
		return func(c *Ctx) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), 50*time.Millisecond)
			defer cancel()
			*c.req = *c.Request().WithContext(ctx)
			return next(c)
		}
	})
	r.Get("/ctx", func(c *Ctx) error {
		select {
		case <-c.Request().Context().Done():
			c.Response().WriteHeader(200)
		case <-time.After(100 * time.Millisecond):
			c.Response().WriteHeader(504)
		}
		return nil
	})

	rr := doReq(t, r, http.MethodGet, "/ctx", nil, nil)
	if rr.Code != 200 {
		t.Fatalf("expected 200 due to early context Done, got %d", rr.Code)
	}
}

func TestNotFound_CustomAndNil(t *testing.T) {
	r := NewRouter()

	// Custom NotFound
	r.NotFound(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-NF", "custom")
		http.Error(w, "nope", 470)
	}))

	rr := doReq(t, r, http.MethodGet, "/missing", nil, nil)
	if rr.Code != 470 {
		t.Fatalf("expected custom NotFound 470, got %d", rr.Code)
	}
	if rr.Header().Get("X-NF") != "custom" || !strings.Contains(rr.Body.String(), "nope") {
		t.Fatalf("custom NotFound handler not used")
	}

	// Calling NotFound(nil) should be a no-op and keep the prior handler
	r.NotFound(nil)
	rr2 := doReq(t, r, http.MethodGet, "/still-missing", nil, nil)
	if rr2.Code != 470 || rr2.Header().Get("X-NF") != "custom" {
		t.Fatalf("NotFound(nil) should not change handler")
	}
}

func TestMethodHelpers_All(t *testing.T) {
	r := NewRouter()

	// Unique codes per method to make assertions unambiguous
	r.Get("/m", func(c *Ctx) error {
		c.Response().WriteHeader(200)
		_, _ = c.Response().Write([]byte("GET"))
		return nil
	})
	r.Head("/m", func(c *Ctx) error { c.Response().WriteHeader(201); return nil })
	r.Post("/m", func(c *Ctx) error {
		c.Response().WriteHeader(202)
		_, _ = c.Response().Write([]byte("POST"))
		return nil
	})
	r.Put("/m", func(c *Ctx) error {
		c.Response().WriteHeader(203)
		_, _ = c.Response().Write([]byte("PUT"))
		return nil
	})
	r.Patch("/m", func(c *Ctx) error {
		c.Response().WriteHeader(204)
		_, _ = c.Response().Write([]byte("PATCH"))
		return nil
	})
	r.Delete("/m", func(c *Ctx) error {
		c.Response().WriteHeader(205)
		_, _ = c.Response().Write([]byte("DELETE"))
		return nil
	})

	// CONNECT and TRACE require a real server round trip
	r.Connect("/m", func(c *Ctx) error {
		c.Response().WriteHeader(206)
		_, _ = c.Response().Write([]byte("CONNECT"))
		return nil
	})
	r.Trace("/m", func(c *Ctx) error {
		c.Response().WriteHeader(207)
		_, _ = c.Response().Write([]byte("TRACE"))
		return nil
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	// Table for simple methods we can exercise with recorder
	type row struct {
		method string
		want   int
		body   string
	}
	tests := []row{
		{http.MethodGet, 200, "GET"},
		{http.MethodHead, 201, ""}, // HEAD has no body here
		{http.MethodPost, 202, "POST"},
		{http.MethodPut, 203, "PUT"},
		{http.MethodPatch, 204, "PATCH"},
		{http.MethodDelete, 205, "DELETE"},
	}
	for _, tc := range tests {
		rr := doReq(t, r, tc.method, "/m", nil, nil)
		if rr.Code != tc.want {
			t.Fatalf("%s: want %d got %d", tc.method, tc.want, rr.Code)
		}
		if strings.TrimSpace(rr.Body.String()) != tc.body {
			t.Fatalf("%s: want body %q got %q", tc.method, tc.body, rr.Body.String())
		}
	}

	// CONNECT via client
	reqC, _ := http.NewRequest(http.MethodConnect, srv.URL+"/m", nil)
	resC, err := http.DefaultClient.Do(reqC)
	if err != nil {
		t.Fatal(err)
	}
	bC, _ := io.ReadAll(resC.Body)
	_ = resC.Body.Close()
	if resC.StatusCode != 206 || strings.TrimSpace(string(bC)) != "CONNECT" {
		t.Fatalf("CONNECT: want 206 CONNECT, got %d %q", resC.StatusCode, string(bC))
	}

	// TRACE via client
	reqT, _ := http.NewRequest(http.MethodTrace, srv.URL+"/m", nil)
	resT, err := http.DefaultClient.Do(reqT)
	if err != nil {
		t.Fatal(err)
	}
	bT, _ := io.ReadAll(resT.Body)
	_ = resT.Body.Close()
	if resT.StatusCode != 207 || strings.TrimSpace(string(bT)) != "TRACE" {
		t.Fatalf("TRACE: want 207 TRACE, got %d %q", resT.StatusCode, string(bT))
	}
}

func TestCompatUseAndHandleMethod(t *testing.T) {
	r := NewRouter()

	// std middleware stamps a header — it will not affect Compat.HandleMethod routes
	stdMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Std", "1")
			next.ServeHTTP(w, req)
		})
	}
	r.Compat.Use(stdMW)

	// Only allow POST on /compat
	r.Compat.HandleMethod("POST", "/compat", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))

	// Wrong method -> 405
	rr := doReq(t, r, http.MethodGet, "/compat", nil, nil)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	if rr.Header().Get("Allow") == "" {
		t.Fatalf("expected Allow header on 405")
	}

	// Correct method -> 200; no X-Std because compat handlers bypass Mizu chain
	rr2 := doReq(t, r, http.MethodPost, "/compat", nil, nil)
	if rr2.Code != 200 || strings.TrimSpace(rr2.Body.String()) != "ok" {
		t.Fatalf("unexpected compat response %d body=%q", rr2.Code, rr2.Body.String())
	}
	if got := rr2.Header().Get("X-Std"); got != "" {
		t.Fatalf("did not expect X-Std to be set for compat handler, got %q", got)
	}
}

func TestAdaptStdMiddleware_ViaCompatUse(t *testing.T) {
	t.Run("applies to Mizu handlers and stamps header", func(t *testing.T) {
		r := NewRouter()

		stdMW := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("X-Std", "yes")
				next.ServeHTTP(w, req)
			})
		}
		// This calls r.adaptStdMiddleware under the hood
		r.Compat.Use(stdMW)

		r.Get("/x", func(c *Ctx) error {
			c.Response().WriteHeader(200)
			_, _ = c.Response().Write([]byte("ok"))
			return nil
		})

		rr := doReq(t, r, http.MethodGet, "/x", nil, nil)
		if rr.Code != 200 {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if got := rr.Header().Get("X-Std"); got != "yes" {
			t.Fatalf("expected X-Std=yes, got %q", got)
		}
		if strings.TrimSpace(rr.Body.String()) != "ok" {
			t.Fatalf("unexpected body %q", rr.Body.String())
		}
	})

	t.Run("error path uses ErrorHandler when set", func(t *testing.T) {
		r := NewRouter()

		// No-op std middleware still exercises the adapted path
		r.Compat.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req)
			})
		})

		r.ErrorHandler(func(c *Ctx, err error) {
			c.Response().WriteHeader(599)
			_, _ = c.Response().Write([]byte("handled"))
		})

		r.Get("/err", func(c *Ctx) error { return errors.New("boom") })

		rr := doReq(t, r, http.MethodGet, "/err", nil, nil)
		if rr.Code != 599 || strings.TrimSpace(rr.Body.String()) != "handled" {
			t.Fatalf("expected 599 handled, got %d %q", rr.Code, rr.Body.String())
		}
	})

	t.Run("error path falls back to 500 when no ErrorHandler", func(t *testing.T) {
		r := NewRouter()

		r.Compat.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req)
			})
		})

		r.Get("/err500", func(c *Ctx) error { return errors.New("fail") })

		rr := doReq(t, r, http.MethodGet, "/err500", nil, nil)
		if rr.Code != 500 {
			t.Fatalf("expected 500 fallback, got %d", rr.Code)
		}
	})
}

func TestDevMode_ColorBranch(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	r := NewRouter()

	// Should use newColorTextHandler branch and set slog.Default to r.Logger()
	r.Dev(true)

	if slog.Default() != r.Logger() {
		t.Fatalf("expected slog.Default to be replaced by router logger")
	}

	// Add a basic route to confirm request handling still works
	r.Get("/devcolor", func(c *Ctx) error {
		c.Response().WriteHeader(200)
		_, _ = c.Response().Write([]byte("ok"))
		return nil
	})

	rr := doReq(t, r, http.MethodGet, "/devcolor", nil, nil)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Fatalf("dev color mode failed, got %d %q", rr.Code, rr.Body.String())
	}
}

func TestCompatMount_DelegatesToHandle(t *testing.T) {
	r := NewRouter()

	called := false
	ret := r.Compat.Mount("/mounted", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))

	// Ensure chaining returns the same compat router type
	if ret == nil {
		t.Fatalf("Mount should return *httpRouter for chaining")
	}

	rr := doReq(t, r, http.MethodGet, "/mounted", nil, nil)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Fatalf("expected 200 ok, got %d %q", rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatalf("mounted handler was not called")
	}
}

func TestPanicError_ErrorString(t *testing.T) {
	e1 := (&PanicError{Value: "boom"}).Error()
	if e1 != "panic: boom" {
		t.Fatalf("unexpected Error() string: %q", e1)
	}

	e2 := (&PanicError{Value: 123}).Error()
	if e2 != "panic: 123" {
		t.Fatalf("unexpected Error() string for int: %q", e2)
	}
}
