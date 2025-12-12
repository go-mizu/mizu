package bearerauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(token string) bool {
		return token == "valid-token"
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "protected")
	})

	t.Run("allows valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("rejects missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("rejects wrong scheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestWithHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithHeader("X-API-Token", func(token string) bool {
		return token == "api-key-123"
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("X-API-Token", "Bearer api-key-123")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_ValidatorWithContext(t *testing.T) {
	type UserClaims struct {
		UserID string
		Role   string
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ValidatorWithContext: func(token string) (any, bool) {
			if token == "admin-token" {
				return UserClaims{UserID: "123", Role: "admin"}, true
			}
			return nil, false
		},
	}))

	var capturedClaims any
	app.Get("/protected", func(c *mizu.Ctx) error {
		capturedClaims = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	claims, ok := capturedClaims.(UserClaims)
	if !ok {
		t.Fatal("expected UserClaims in context")
	}
	if claims.UserID != "123" || claims.Role != "admin" {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator: func(token string) bool { return false },
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": err.Error(),
			})
		},
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWithOptions_CustomScheme(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator:  func(token string) bool { return token == "valid" },
		AuthScheme: "Token",
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("accepts custom scheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Token valid")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("rejects bearer scheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(token string) bool { return true }))

	var capturedToken string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedToken = Token(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer my-token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedToken != "my-token" {
		t.Errorf("expected token 'my-token', got %q", capturedToken)
	}
}

func TestWithOptions_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic without validator")
		}
	}()
	WithOptions(Options{})
}

func TestClaims(t *testing.T) {
	type MyClaims struct {
		Sub string
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ValidatorWithContext: func(token string) (any, bool) {
			return MyClaims{Sub: "user123"}, true
		},
	}))

	var sub string
	app.Get("/test", func(c *mizu.Ctx) error {
		claims, ok := Claims[MyClaims](c)
		if ok {
			sub = claims.Sub
		}
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if sub != "user123" {
		t.Errorf("expected sub 'user123', got %q", sub)
	}
}

func TestErrors(t *testing.T) {
	if ErrTokenMissing.Error() != "token missing" {
		t.Error("unexpected error message")
	}
	if ErrTokenInvalid.Error() != "token invalid" {
		t.Error("unexpected error message")
	}
	if ErrInvalidScheme.Error() != "invalid auth scheme" {
		t.Error("unexpected error message")
	}
}
