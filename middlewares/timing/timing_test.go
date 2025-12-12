package timing

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		time.Sleep(10 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	timing := rec.Header().Get("Server-Timing")
	if timing == "" {
		t.Error("expected Server-Timing header")
	}
	if !strings.Contains(timing, "total;dur=") {
		t.Errorf("expected total timing, got %q", timing)
	}
}

func TestAdd(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		Add(c, "db", 50*time.Millisecond, "Database query")
		Add(c, "cache", 5*time.Millisecond, "")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	timing := rec.Header().Get("Server-Timing")
	if !strings.Contains(timing, "db;dur=50") {
		t.Errorf("expected db timing, got %q", timing)
	}
	if !strings.Contains(timing, `desc="Database query"`) {
		t.Errorf("expected description, got %q", timing)
	}
	if !strings.Contains(timing, "cache;dur=5") {
		t.Errorf("expected cache timing, got %q", timing)
	}
}

func TestStart(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		stop := Start(c, "operation")
		time.Sleep(20 * time.Millisecond)
		stop("Long operation")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	timing := rec.Header().Get("Server-Timing")
	if !strings.Contains(timing, "operation;dur=") {
		t.Errorf("expected operation timing, got %q", timing)
	}
	if !strings.Contains(timing, `desc="Long operation"`) {
		t.Errorf("expected description, got %q", timing)
	}
}

func TestTrack(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		Track(c, "compute", func() {
			time.Sleep(15 * time.Millisecond)
		})
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	timing := rec.Header().Get("Server-Timing")
	if !strings.Contains(timing, "compute;dur=") {
		t.Errorf("expected compute timing, got %q", timing)
	}
}

func TestAdd_NoMiddleware(t *testing.T) {
	app := mizu.NewRouter()
	// No timing middleware

	app.Get("/test", func(c *mizu.Ctx) error {
		Add(c, "test", time.Second, "desc") // Should not panic
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNew_MultipleMetrics(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		Add(c, "auth", 2*time.Millisecond, "Authentication")
		Add(c, "db", 10*time.Millisecond, "Database")
		Add(c, "template", 5*time.Millisecond, "Rendering")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	timing := rec.Header().Get("Server-Timing")
	parts := strings.Split(timing, ", ")
	if len(parts) != 4 { // total + 3 custom
		t.Errorf("expected 4 timing entries, got %d: %q", len(parts), timing)
	}
}
