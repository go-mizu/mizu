package keyauth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(key string) (bool, error) {
		return key == "valid-key", nil
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("X-Api-Key", "valid-key")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("X-Api-Key", "invalid-key")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestWithOptions_QueryLookup(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator: func(key string) (bool, error) { return key == "api123", nil },
		KeyLookup: "query:api_key",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api?api_key=api123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_CookieLookup(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator: func(key string) (bool, error) { return key == "cookie-key", nil },
		KeyLookup: "cookie:auth_token",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "cookie-key"})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_AuthScheme(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator:  func(key string) (bool, error) { return key == "mykey", nil },
		KeyLookup:  "header:Authorization",
		AuthScheme: "ApiKey",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Authorization", "ApiKey mykey")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator: func(key string) (bool, error) { return false, nil },
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": err.Error(),
			})
		},
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("X-Api-Key", "bad")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWithOptions_ValidatorError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator: func(key string) (bool, error) {
			return false, errors.New("database error")
		},
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("X-Api-Key", "key")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestFromContext(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(key string) (bool, error) { return true, nil }))

	var capturedKey string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedKey = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Api-Key", "my-api-key")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedKey != "my-api-key" {
		t.Errorf("expected 'my-api-key', got %q", capturedKey)
	}
}

func TestGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(key string) (bool, error) { return true, nil }))

	var k1, k2 string
	app.Get("/test", func(c *mizu.Ctx) error {
		k1 = FromContext(c)
		k2 = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Api-Key", "key")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if k1 != k2 {
		t.Error("FromContext and Get should return same value")
	}
}

func TestValidateKeys(t *testing.T) {
	validator := ValidateKeys("key1", "key2", "key3")

	tests := []struct {
		key      string
		expected bool
	}{
		{"key1", true},
		{"key2", true},
		{"key3", true},
		{"key4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			valid, err := validator(tt.key)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if valid != tt.expected {
				t.Errorf("ValidateKeys(%q) = %v, want %v", tt.key, valid, tt.expected)
			}
		})
	}
}

func TestWithOptions_Panics(t *testing.T) {
	t.Run("no validator", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		WithOptions(Options{})
	})

	t.Run("invalid lookup", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		WithOptions(Options{
			Validator: func(key string) (bool, error) { return true, nil },
			KeyLookup: "invalid",
		})
	})
}

func TestErrors(t *testing.T) {
	if ErrKeyMissing.Error() != "API key missing" {
		t.Error("unexpected error message")
	}
	if ErrKeyInvalid.Error() != "API key invalid" {
		t.Error("unexpected error message")
	}
}
