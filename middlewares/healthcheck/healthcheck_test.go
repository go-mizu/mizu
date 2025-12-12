package healthcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestLiveness(t *testing.T) {
	app := mizu.NewRouter()
	app.Get("/healthz", Liveness())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected 'ok', got %q", rec.Body.String())
	}
}

func TestReadiness(t *testing.T) {
	t.Run("no checks", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Get("/readyz", Readiness())

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("all healthy", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Get("/readyz", Readiness(
			Check{Name: "db", Check: func(ctx context.Context) error { return nil }},
			Check{Name: "cache", Check: func(ctx context.Context) error { return nil }},
		))

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
			t.Error("expected ok status")
		}
	})

	t.Run("one unhealthy", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Get("/readyz", Readiness(
			Check{Name: "db", Check: func(ctx context.Context) error { return nil }},
			Check{Name: "cache", Check: func(ctx context.Context) error { return errors.New("connection failed") }},
		))

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"error"`) {
			t.Error("expected error status")
		}
	})
}

func TestReadiness_Timeout(t *testing.T) {
	app := mizu.NewRouter()
	app.Get("/readyz", Readiness(
		Check{
			Name: "slow",
			Check: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second):
					return nil
				}
			},
			Timeout: 50 * time.Millisecond,
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d due to timeout, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestNew(t *testing.T) {
	handler := New(Options{
		LivenessPath:  "/live",
		ReadinessPath: "/ready",
		Checks: []Check{
			{Name: "test", Check: func(ctx context.Context) error { return nil }},
		},
	})

	app := mizu.NewRouter()
	app.Get("/live", handler)
	app.Get("/ready", handler)

	t.Run("liveness", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/live", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("readiness", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestRegister(t *testing.T) {
	app := mizu.NewRouter()
	Register(app, Options{
		Checks: []Check{
			{Name: "test", Check: func(ctx context.Context) error { return nil }},
		},
	})

	t.Run("default liveness", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("default readiness", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestDBCheck(t *testing.T) {
	check := DBCheck("postgres", func(ctx context.Context) error {
		return nil
	})

	if check.Name != "postgres" {
		t.Errorf("expected name 'postgres', got %q", check.Name)
	}
	if check.Timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", check.Timeout)
	}
	if err := check.Check(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
