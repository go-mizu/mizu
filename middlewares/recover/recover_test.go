package recover

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/panic", func(c *mizu.Ctx) error {
		panic("test panic")
	})

	app.Get("/ok", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("recovers from panic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})

	t.Run("passes through normal requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "ok" {
			t.Errorf("expected body 'ok', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	var capturedErr any
	var capturedStack []byte

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ErrorHandler: func(c *mizu.Ctx, err any, stack []byte) error {
			capturedErr = err
			capturedStack = stack
			return c.Text(http.StatusServiceUnavailable, "custom error")
		},
	}))

	app.Get("/panic", func(c *mizu.Ctx) error {
		panic("custom panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	if rec.Body.String() != "custom error" {
		t.Errorf("expected body 'custom error', got %q", rec.Body.String())
	}
	if capturedErr != "custom panic" {
		t.Errorf("expected captured error 'custom panic', got %v", capturedErr)
	}
	if len(capturedStack) == 0 {
		t.Error("expected stack trace to be captured")
	}
}

func TestWithOptions_DisablePrintStack(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		DisablePrintStack: true,
		Logger:            logger,
	}))

	app.Get("/panic", func(c *mizu.Ctx) error {
		panic("silent panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if strings.Contains(buf.String(), "stack") {
		t.Error("expected no stack in log output")
	}
}

func TestWithOptions_CustomLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger: logger,
	}))

	app.Get("/panic", func(c *mizu.Ctx) error {
		panic("logged panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "panic recovered") {
		t.Error("expected panic to be logged")
	}
	if !strings.Contains(buf.String(), "logged panic") {
		t.Error("expected panic message in log")
	}
}

func TestWithOptions_StackSize(t *testing.T) {
	var capturedStack []byte

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		StackSize: 100,
		ErrorHandler: func(c *mizu.Ctx, err any, stack []byte) error {
			capturedStack = stack
			return c.Text(http.StatusInternalServerError, "error")
		},
	}))

	app.Get("/panic", func(c *mizu.Ctx) error {
		panic("stack test")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(capturedStack) > 100 {
		t.Errorf("expected stack size <= 100, got %d", len(capturedStack))
	}
}
