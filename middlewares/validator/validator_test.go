package validator

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(
		Field("name", "required", "min:3"),
		Field("email", "required", "email"),
	))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=John&email=john@example.com", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?email=john@example.com", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=John&email=invalid", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("min length violation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=Jo&email=john@example.com", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestOptionalField(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(
		Field("name", "required"),
		OptionalField("age", "numeric"),
	))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("without optional field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=John", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("with valid optional field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=John&age=25", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("with invalid optional field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=John&age=abc", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestRules(t *testing.T) {
	tests := []struct {
		name      string
		rule      string
		value     string
		shouldErr bool
	}{
		{"required pass", "required", "value", false},
		{"required fail", "required", "", true},
		{"min pass", "min:3", "abc", false},
		{"min fail", "min:3", "ab", true},
		{"max pass", "max:5", "abc", false},
		{"max fail", "max:5", "abcdef", true},
		{"email pass", "email", "test@example.com", false},
		{"email fail", "email", "invalid", true},
		{"numeric pass", "numeric", "123.45", false},
		{"numeric fail", "numeric", "abc", true},
		{"integer pass", "integer", "123", false},
		{"integer fail", "integer", "12.3", true},
		{"alpha pass", "alpha", "abc", false},
		{"alpha fail", "alpha", "abc123", true},
		{"alphanum pass", "alphanum", "abc123", false},
		{"alphanum fail", "alphanum", "abc-123", true},
		{"in pass", "in:a,b,c", "b", false},
		{"in fail", "in:a,b,c", "d", true},
		{"url pass", "url", "https://example.com", false},
		{"url fail", "url", "not-a-url", true},
		{"uuid pass", "uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"uuid fail", "uuid", "invalid-uuid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyRule("field", tt.value, tt.rule, "")
			if tt.shouldErr && err == nil {
				t.Error("expected validation error")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Rules: []Rule{
			Field("name", "required"),
		},
		ErrorHandler: func(c *mizu.Ctx, errors ValidationErrors) error {
			return c.Text(http.StatusUnprocessableEntity, errors.Error())
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected %d, got %d", http.StatusUnprocessableEntity, rec.Code)
	}
}

func TestJSON(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(JSON(map[string][]string{
		"name":  {"required", "min:3"},
		"email": {"required", "email"},
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid JSON", func(t *testing.T) {
		body := `{"name":"John","email":"john@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		body := `{"name":"Jo","email":"invalid"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		body := `{invalid}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestValidationErrors_Error(t *testing.T) {
	errors := ValidationErrors{
		{Field: "name", Message: "is required"},
		{Field: "email", Message: "must be valid"},
	}

	expected := "name: is required; email: must be valid"
	if errors.Error() != expected {
		t.Errorf("expected %q, got %q", expected, errors.Error())
	}
}

func TestValidationErrors_Empty(t *testing.T) {
	var errors ValidationErrors
	if errors.Error() != "validation failed" {
		t.Errorf("expected 'validation failed', got %q", errors.Error())
	}
}

func TestField(t *testing.T) {
	rule := Field("name", "required", "min:3")

	if rule.Field != "name" {
		t.Errorf("expected field 'name', got %q", rule.Field)
	}
	if len(rule.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rule.Rules))
	}
	if rule.Optional {
		t.Error("expected optional false")
	}
}
