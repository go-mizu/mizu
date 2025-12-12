package mirror

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	var mirrorReceived int32

	// Create mirror server
	mirrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&mirrorReceived, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer mirrorServer.Close()

	// Test using the New() helper
	app := mizu.NewRouter()
	app.Use(New(mirrorServer.URL))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	if mirrorReceived != 1 {
		t.Errorf("expected mirror to receive 1 request, got %d", mirrorReceived)
	}
}

func TestWithOptions_Async(t *testing.T) {
	var mirrorReceived int32

	mirrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&mirrorReceived, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer mirrorServer.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Targets: []Target{{URL: mirrorServer.URL, Percentage: 100}},
		Async:   true,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	start := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	// Should return quickly (async)
	if elapsed > 30*time.Millisecond {
		t.Errorf("expected fast response with async, took %v", elapsed)
	}

	// Wait for mirror to complete
	time.Sleep(100 * time.Millisecond)

	if mirrorReceived != 1 {
		t.Errorf("expected mirror to receive 1 request, got %d", mirrorReceived)
	}
}

func TestWithOptions_Percentage(t *testing.T) {
	var mirrorReceived int32

	mirrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&mirrorReceived, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer mirrorServer.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Targets: []Target{{URL: mirrorServer.URL, Percentage: 50}},
		Async:   false,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Send 100 requests
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// Should be roughly 50%
	received := atomic.LoadInt32(&mirrorReceived)
	if received < 40 || received > 60 {
		t.Errorf("expected ~50 mirrored requests, got %d", received)
	}
}

func TestWithOptions_MultipleTargets(t *testing.T) {
	var target1Received, target2Received int32

	target1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&target1Received, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target1.Close()

	target2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&target2Received, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target2.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Targets: []Target{
			{URL: target1.URL, Percentage: 100},
			{URL: target2.URL, Percentage: 100},
		},
		Async: false,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if target1Received != 1 {
		t.Errorf("expected target1 to receive 1 request, got %d", target1Received)
	}
	if target2Received != 1 {
		t.Errorf("expected target2 to receive 1 request, got %d", target2Received)
	}
}

func TestWithOptions_OnError(t *testing.T) {
	var errorCalled bool

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Targets: []Target{{URL: "http://invalid.invalid:99999", Percentage: 100}},
		Async:   false,
		Timeout: 100 * time.Millisecond,
		OnError: func(target string, err error) {
			errorCalled = true
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !errorCalled {
		t.Error("expected OnError to be called")
	}
}

func TestWithOptions_OnSuccess(t *testing.T) {
	var successCalled bool
	var responseStatus int

	mirrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer mirrorServer.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Targets: []Target{{URL: mirrorServer.URL, Percentage: 100}},
		Async:   false,
		OnSuccess: func(target string, resp *http.Response) {
			successCalled = true
			responseStatus = resp.StatusCode
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !successCalled {
		t.Error("expected OnSuccess to be called")
	}
	if responseStatus != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, responseStatus)
	}
}

func TestWithOptions_MirroredHeader(t *testing.T) {
	var receivedHeader string

	mirrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Mirrored-From")
		w.WriteHeader(http.StatusOK)
	}))
	defer mirrorServer.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Targets: []Target{{URL: mirrorServer.URL, Percentage: 100}},
		Async:   false,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if receivedHeader != "example.com" {
		t.Errorf("expected X-Mirrored-From header, got %q", receivedHeader)
	}
}

func TestPercentage(t *testing.T) {
	target := Percentage("http://example.com", 25)

	if target.URL != "http://example.com" {
		t.Errorf("expected URL 'http://example.com', got %q", target.URL)
	}
	if target.Percentage != 25 {
		t.Errorf("expected Percentage 25, got %d", target.Percentage)
	}
}

func TestWithOptions_CopyBody(t *testing.T) {
	var receivedBody string

	mirrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer mirrorServer.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Targets:  []Target{{URL: mirrorServer.URL, Percentage: 100}},
		Async:    false,
		CopyBody: true,
	}))

	app.Post("/test", func(c *mizu.Ctx) error {
		// Read body in handler
		body, _ := io.ReadAll(c.Request().Body)
		return c.Text(http.StatusOK, string(body))
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test body"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if receivedBody != "test body" {
		t.Errorf("expected mirror to receive body, got %q", receivedBody)
	}
}

func TestWithOptions_NoCopyBody(t *testing.T) {
	var receivedBody string

	mirrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer mirrorServer.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Targets:  []Target{{URL: mirrorServer.URL, Percentage: 100}},
		Async:    false,
		CopyBody: false,
	}))

	app.Post("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test body"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// With CopyBody=false, mirror shouldn't receive body
	if receivedBody != "" {
		t.Errorf("expected mirror to receive empty body, got %q", receivedBody)
	}
}

func TestNew_MultipleTargets(t *testing.T) {
	var target1Received, target2Received int32

	target1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&target1Received, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target1.Close()

	target2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&target2Received, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target2.Close()

	app := mizu.NewRouter()
	app.Use(New(target1.URL, target2.URL))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Wait for async mirrors to complete
	time.Sleep(100 * time.Millisecond)

	if target1Received != 1 {
		t.Errorf("expected target1 to receive 1 request, got %d", target1Received)
	}
	if target2Received != 1 {
		t.Errorf("expected target2 to receive 1 request, got %d", target2Received)
	}
}
