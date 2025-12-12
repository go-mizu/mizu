package sentry

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCaptureError(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/error", func(c *mizu.Ctx) error {
		return errors.New("test error")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if captured == nil {
		t.Fatal("expected error to be captured")
	}

	if captured.Message != "test error" {
		t.Errorf("expected 'test error', got %q", captured.Message)
	}

	if captured.Level != "error" {
		t.Errorf("expected level 'error', got %q", captured.Level)
	}
}

func TestCaptureMessage(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		CaptureMessage(c, "test message")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if captured == nil {
		t.Fatal("expected message to be captured")
	}

	if captured.Message != "test message" {
		t.Errorf("expected 'test message', got %q", captured.Message)
	}

	if captured.Level != "info" {
		t.Errorf("expected level 'info', got %q", captured.Level)
	}
}

func TestCapturePanic(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/panic", func(c *mizu.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	defer func() {
		_ = recover() // Catch the re-panic

		if captured == nil {
			t.Fatal("expected panic to be captured")
		}

		if captured.Message != "test panic" {
			t.Errorf("expected 'test panic', got %q", captured.Message)
		}

		if captured.Level != "fatal" {
			t.Errorf("expected level 'fatal', got %q", captured.Level)
		}
	}()

	app.ServeHTTP(rec, req)
}

func TestEnvironmentAndRelease(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Environment: "test",
		Release:     "1.0.0",
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if captured.Environment != "test" {
		t.Errorf("expected environment 'test', got %q", captured.Environment)
	}

	if captured.Release != "1.0.0" {
		t.Errorf("expected release '1.0.0', got %q", captured.Release)
	}
}

func TestTags(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Tags: map[string]string{
			"service": "api",
		},
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if captured.Tags["service"] != "api" {
		t.Errorf("expected tag 'service'='api', got %q", captured.Tags["service"])
	}
}

func TestBeforeSend(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BeforeSend: func(event *Event) *Event {
			event.Tags["modified"] = "true"
			return event
		},
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if captured.Tags["modified"] != "true" {
		t.Error("expected BeforeSend to modify event")
	}
}

func TestBeforeSendDrop(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BeforeSend: func(event *Event) *Event {
			return nil // Drop event
		},
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if captured != nil {
		t.Error("expected event to be dropped")
	}
}

func TestRequestInfo(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return errors.New("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
	req.Header.Set("User-Agent", "TestAgent")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if captured.Request == nil {
		t.Fatal("expected request info")
	}

	if captured.Request.Method != "GET" {
		t.Errorf("expected method GET, got %q", captured.Request.Method)
	}

	if captured.Request.QueryString != "foo=bar" {
		t.Errorf("expected query string 'foo=bar', got %q", captured.Request.QueryString)
	}

	if captured.Request.Headers["User-Agent"] != "TestAgent" {
		t.Errorf("expected User-Agent header, got %q", captured.Request.Headers["User-Agent"])
	}
}

func TestSensitiveHeadersFiltered(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("X-Api-Key", "secret-key")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if _, ok := captured.Request.Headers["Authorization"]; ok {
		t.Error("Authorization header should be filtered")
	}

	if _, ok := captured.Request.Headers["Cookie"]; ok {
		t.Error("Cookie header should be filtered")
	}

	if _, ok := captured.Request.Headers["X-Api-Key"]; ok {
		t.Error("X-Api-Key header should be filtered")
	}
}

func TestStacktrace(t *testing.T) {
	var captured *Event

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnError: func(event *Event) {
			captured = event
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(captured.Exception) == 0 {
		t.Fatal("expected exception")
	}

	if captured.Exception[0].Stacktrace == nil {
		t.Fatal("expected stacktrace")
	}

	if len(captured.Exception[0].Stacktrace.Frames) == 0 {
		t.Error("expected stacktrace frames")
	}
}

func TestMockTransport(t *testing.T) {
	transport := &MockTransport{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Transport: transport,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		CaptureMessage(c, "test")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Give transport time to receive event
	// In real code, would use proper synchronization
}

func TestHubEvents(t *testing.T) {
	hub := &Hub{
		opts:   Options{SampleRate: 1.0},
		events: make([]*Event, 0),
	}

	event := &Event{EventID: "test-id", Message: "test"}
	hub.captureEvent(event)

	events := hub.Events()
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	hub.Clear()
	events = hub.Events()
	if len(events) != 0 {
		t.Errorf("expected 0 events after clear, got %d", len(events))
	}
}

func TestGetHub(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var hubFound bool

	app.Get("/", func(c *mizu.Ctx) error {
		hub := GetHub(c)
		hubFound = hub != nil
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !hubFound {
		t.Error("expected hub to be in context")
	}
}

func TestSetUser(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		SetUser(c, &User{
			ID:    "123",
			Email: "test@example.com",
		})
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSetTagAndExtra(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		SetTag(c, "key", "value")
		SetExtra(c, "data", "extra-data")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
