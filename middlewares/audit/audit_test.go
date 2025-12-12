package audit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	var capturedEntry *Entry

	app := mizu.NewRouter()
	app.Use(New(func(entry *Entry) {
		capturedEntry = entry
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
	req.Header.Set("User-Agent", "TestAgent")
	req.Header.Set("X-Request-ID", "req-123")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedEntry == nil {
		t.Fatal("expected entry to be captured")
	}

	if capturedEntry.Method != "GET" {
		t.Errorf("expected GET, got %s", capturedEntry.Method)
	}
	if capturedEntry.Path != "/test" {
		t.Errorf("expected /test, got %s", capturedEntry.Path)
	}
	if capturedEntry.Query != "foo=bar" {
		t.Errorf("expected foo=bar, got %s", capturedEntry.Query)
	}
	if capturedEntry.UserAgent != "TestAgent" {
		t.Errorf("expected TestAgent, got %s", capturedEntry.UserAgent)
	}
	if capturedEntry.RequestID != "req-123" {
		t.Errorf("expected req-123, got %s", capturedEntry.RequestID)
	}
	if capturedEntry.Status != http.StatusOK {
		t.Errorf("expected 200, got %d", capturedEntry.Status)
	}
	if capturedEntry.Latency == 0 {
		t.Error("expected non-zero latency")
	}
}

func TestWithOptions_RequestBody(t *testing.T) {
	var capturedEntry *Entry

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Handler: func(entry *Entry) {
			capturedEntry = entry
		},
		IncludeRequestBody: true,
		MaxBodySize:        100,
	}))

	app.Post("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := strings.NewReader(`{"name":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedEntry.RequestBody != `{"name":"test"}` {
		t.Errorf("expected request body, got %q", capturedEntry.RequestBody)
	}
}

func TestWithOptions_Skip(t *testing.T) {
	var callCount int

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Handler: func(entry *Entry) {
			callCount++
		},
		Skip: func(c *mizu.Ctx) bool {
			return c.Request().URL.Path == "/health"
		},
	}))

	app.Get("/health", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Health should be skipped
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if callCount != 0 {
		t.Error("expected /health to be skipped")
	}

	// API should be logged
	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestWithOptions_Metadata(t *testing.T) {
	var capturedEntry *Entry

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Handler: func(entry *Entry) {
			capturedEntry = entry
		},
		Metadata: func(c *mizu.Ctx) map[string]string {
			return map[string]string{
				"version": "1.0",
				"env":     "test",
			}
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedEntry.Metadata["version"] != "1.0" {
		t.Error("expected version metadata")
	}
	if capturedEntry.Metadata["env"] != "test" {
		t.Error("expected env metadata")
	}
}

func TestWithOptions_Error(t *testing.T) {
	var capturedEntry *Entry

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Handler: func(entry *Entry) {
			capturedEntry = entry
		},
	}))

	app.Get("/error", func(c *mizu.Ctx) error {
		return c.Text(http.StatusInternalServerError, "error")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedEntry.Status != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", capturedEntry.Status)
	}
}

func TestChannelHandler(t *testing.T) {
	ch := make(chan *Entry, 10)

	app := mizu.NewRouter()
	app.Use(New(ChannelHandler(ch)))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	select {
	case entry := <-ch:
		if entry.Method != "GET" {
			t.Errorf("expected GET, got %s", entry.Method)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for entry")
	}
}

func TestBufferedHandler(t *testing.T) {
	var flushedEntries []*Entry
	var flushCount int

	handler := NewBufferedHandler(3, time.Hour, func(entries []*Entry) {
		flushedEntries = entries
		flushCount++
	})
	defer handler.Close()

	// Add entries
	handler.Handle(&Entry{Method: "GET", Path: "/1"})
	handler.Handle(&Entry{Method: "GET", Path: "/2"})

	// Should not flush yet
	if flushCount != 0 {
		t.Error("should not flush before maxSize")
	}

	// Third entry should trigger flush
	handler.Handle(&Entry{Method: "GET", Path: "/3"})

	// Wait for flush
	time.Sleep(10 * time.Millisecond)

	if flushCount != 1 {
		t.Errorf("expected 1 flush, got %d", flushCount)
	}
	if len(flushedEntries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(flushedEntries))
	}
}

func TestBufferedHandler_Flush(t *testing.T) {
	var flushedEntries []*Entry

	handler := NewBufferedHandler(100, time.Hour, func(entries []*Entry) {
		flushedEntries = entries
	})
	defer handler.Close()

	handler.Handle(&Entry{Method: "GET", Path: "/test"})
	handler.Flush()

	if len(flushedEntries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(flushedEntries))
	}
}

func TestBufferedHandler_Handler(t *testing.T) {
	handler := NewBufferedHandler(10, time.Hour, func(entries []*Entry) {})
	defer handler.Close()

	handlerFunc := handler.Handler()
	if handlerFunc == nil {
		t.Error("expected handler function")
	}
}
