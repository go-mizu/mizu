package envelope

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"name": "John"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Error("expected success: true")
	}
	if resp["data"] == nil {
		t.Error("expected data field")
	}
}

func TestWithOptions_IncludeMeta(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IncludeMeta: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"name": "John"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "req-123")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if meta, ok := resp["meta"].(map[string]any); ok {
		if meta["request_id"] != "req-123" {
			t.Error("expected request_id in meta")
		}
		if meta["status_code"] != float64(200) {
			t.Error("expected status_code in meta")
		}
	} else {
		t.Error("expected meta field")
	}
}

func TestWithOptions_CustomFields(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SuccessField: "ok",
		DataField:    "result",
		ErrorField:   "message",
		IncludeMeta:  false,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"name": "John"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp["ok"] != true {
		t.Error("expected ok: true")
	}
	if resp["result"] == nil {
		t.Error("expected result field")
	}
}

func TestSuccess(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		return Success(c, map[string]string{"name": "John"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if !resp.Success {
		t.Error("expected success: true")
	}
}

func TestError(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		return Error(c, http.StatusBadRequest, "invalid input")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Success {
		t.Error("expected success: false")
	}
	if resp.Error != "invalid input" {
		t.Errorf("expected error 'invalid input', got %q", resp.Error)
	}
}

func TestCreated(t *testing.T) {
	app := mizu.NewRouter()

	app.Post("/", func(c *mizu.Ctx) error {
		return Created(c, map[string]int{"id": 123})
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestBadRequest(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		return BadRequest(c, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		return Unauthorized(c, "unauthorized")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestForbidden(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		return Forbidden(c, "forbidden")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestNotFound(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		return NotFound(c, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestInternalError(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		return InternalError(c, "internal error")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestNoContent(t *testing.T) {
	app := mizu.NewRouter()

	app.Delete("/", func(c *mizu.Ctx) error {
		return NoContent(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
}
