package requestid

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var capturedID string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedID = FromContext(c)
		return c.Text(http.StatusOK, capturedID)
	})

	t.Run("generates request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		respID := rec.Header().Get("X-Request-ID")
		if respID == "" {
			t.Error("expected X-Request-ID header to be set")
		}
		if capturedID == "" {
			t.Error("expected request ID in context")
		}
		if respID != capturedID {
			t.Errorf("response header %q != context ID %q", respID, capturedID)
		}
	})

	t.Run("uses existing request ID", func(t *testing.T) {
		existingID := "existing-request-id-123"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", existingID)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		respID := rec.Header().Get("X-Request-ID")
		if respID != existingID {
			t.Errorf("expected %q, got %q", existingID, respID)
		}
		if capturedID != existingID {
			t.Errorf("expected context ID %q, got %q", existingID, capturedID)
		}
	})
}

func TestWithOptions_CustomHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Header: "X-Correlation-ID",
	}))

	var capturedID string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedID = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Correlation-ID", "custom-header-id")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	respID := rec.Header().Get("X-Correlation-ID")
	if respID != "custom-header-id" {
		t.Errorf("expected 'custom-header-id', got %q", respID)
	}
	if capturedID != "custom-header-id" {
		t.Errorf("expected context ID 'custom-header-id', got %q", capturedID)
	}
}

func TestWithOptions_CustomGenerator(t *testing.T) {
	counter := 0
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Generator: func() string {
			counter++
			return "custom-id-" + string(rune('0'+counter))
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, FromContext(c))
	})

	// First request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "custom-id-1" {
		t.Errorf("expected 'custom-id-1', got %q", rec.Body.String())
	}

	// Second request
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "custom-id-2" {
		t.Errorf("expected 'custom-id-2', got %q", rec.Body.String())
	}
}

func TestGenerateID(t *testing.T) {
	id := generateID()

	// Check format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Errorf("expected 5 parts, got %d", len(parts))
	}

	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			t.Errorf("part %d: expected length %d, got %d", i, expectedLengths[i], len(part))
		}
	}

	// Check version byte (should be 4)
	if parts[2][0] != '4' {
		t.Errorf("expected version 4, got %c", parts[2][0])
	}

	// Check uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateID()
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var id1, id2 string
	app.Get("/test", func(c *mizu.Ctx) error {
		id1 = FromContext(c)
		id2 = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if id1 != id2 {
		t.Errorf("FromContext and Get should return same value: %q != %q", id1, id2)
	}
}
