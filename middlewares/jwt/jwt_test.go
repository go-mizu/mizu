package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

var testSecret = []byte("test-secret-key-32-bytes-long!!!")

func createToken(claims map[string]any, secret []byte) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)

	message := header + "." + payloadEnc
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	sig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return message + "." + sig
}

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid token", func(t *testing.T) {
		token := createToken(map[string]any{
			"sub": "user123",
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		token := createToken(map[string]any{
			"sub": "user123",
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}, []byte("wrong-secret"))

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestWithOptions_ExpiredToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createToken(map[string]any{
		"sub": "user123",
		"exp": float64(time.Now().Add(-time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d for expired token, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWithOptions_Issuer(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret: testSecret,
		Issuer: "my-app",
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid issuer", func(t *testing.T) {
		token := createToken(map[string]any{
			"sub": "user123",
			"iss": "my-app",
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("invalid issuer", func(t *testing.T) {
		token := createToken(map[string]any{
			"sub": "user123",
			"iss": "other-app",
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestGetClaims(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	var capturedClaims map[string]any
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedClaims = GetClaims(c)
		return c.Text(http.StatusOK, "ok")
	})

	token := createToken(map[string]any{
		"sub":   "user123",
		"email": "user@example.com",
		"exp":   float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedClaims["sub"] != "user123" {
		t.Errorf("expected sub 'user123', got %v", capturedClaims["sub"])
	}
	if capturedClaims["email"] != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got %v", capturedClaims["email"])
	}
}

func TestSubject(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	var sub string
	app.Get("/test", func(c *mizu.Ctx) error {
		sub = Subject(c)
		return c.Text(http.StatusOK, "ok")
	})

	token := createToken(map[string]any{
		"sub": "user456",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if sub != "user456" {
		t.Errorf("expected 'user456', got %q", sub)
	}
}

func TestWithOptions_QueryLookup(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:      testSecret,
		TokenLookup: "query:token",
		AuthScheme:  "",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createToken(map[string]any{
		"sub": "user",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api?token="+token, nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_Panics(t *testing.T) {
	t.Run("no secret", func(t *testing.T) {
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
			Secret:      testSecret,
			TokenLookup: "invalid",
		})
	})
}
