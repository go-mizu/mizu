package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	// Create upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "upstream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream response"))
	}))
	defer upstream.Close()

	app := mizu.NewRouter()
	app.Use(New(upstream.URL))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "should not reach")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "upstream response" {
		t.Errorf("expected 'upstream response', got %q", rec.Body.String())
	}
	if rec.Header().Get("X-Custom") != "upstream" {
		t.Error("expected X-Custom header from upstream")
	}
}

func TestWithOptions_Rewrite(t *testing.T) {
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Target: u,
		Rewrite: func(path string) string {
			return "/api/v2" + path
		},
	}))

	app.Get("/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "should not reach")
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if receivedPath != "/api/v2/users" {
		t.Errorf("expected '/api/v2/users', got %q", receivedPath)
	}
}

func TestWithOptions_ModifyRequest(t *testing.T) {
	var receivedHeader string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Modified")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Target: u,
		ModifyRequest: func(req *http.Request) {
			req.Header.Set("X-Modified", "yes")
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if receivedHeader != "yes" {
		t.Errorf("expected 'yes', got %q", receivedHeader)
	}
}

func TestWithOptions_ModifyResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Original", "value")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Target: u,
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Set("X-Modified", "true")
			return nil
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Modified") != "true" {
		t.Error("expected X-Modified header")
	}
}

func TestWithOptions_PreserveHost(t *testing.T) {
	var receivedHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Target:       u,
		PreserveHost: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "http://original.example.com/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if receivedHost != "original.example.com" {
		t.Errorf("expected 'original.example.com', got %q", receivedHost)
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	u, _ := url.Parse("http://localhost:99999") // Invalid port
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Target: u,
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			return c.Text(http.StatusServiceUnavailable, "service unavailable")
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestWithOptions_QueryString(t *testing.T) {
	var receivedQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Target: u}))

	app.Get("/search", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if receivedQuery != "q=test&page=1" {
		t.Errorf("expected 'q=test&page=1', got %q", receivedQuery)
	}
}

func TestWithOptions_POST(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Target: u}))

	app.Post("/data", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/data", nil)
	req.Body = io.NopCloser(io.Reader(httptest.NewRecorder().Body))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNew_InvalidURL(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	New("://invalid")
}

func TestWithOptions_NoTarget(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	WithOptions(Options{})
}

func TestBalancer(t *testing.T) {
	var calls []string
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "1")
		_, _ = w.Write([]byte("1"))
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "2")
		_, _ = w.Write([]byte("2"))
	}))
	defer upstream2.Close()

	app := mizu.NewRouter()
	app.Use(Balancer([]string{upstream1.URL, upstream2.URL}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Make multiple requests
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// Should have round-robin distribution
	if len(calls) != 4 {
		t.Errorf("expected 4 calls, got %d", len(calls))
	}
}

func TestBalancer_Empty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	Balancer([]string{})
}
