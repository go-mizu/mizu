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

func TestWithOptions_NotBeforeToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Token not valid yet
	token := createToken(map[string]any{
		"sub": "user123",
		"nbf": float64(time.Now().Add(time.Hour).Unix()),
		"exp": float64(time.Now().Add(2 * time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d for not-yet-valid token, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWithOptions_ValidNotBefore(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Token valid now
	token := createToken(map[string]any{
		"sub": "user123",
		"nbf": float64(time.Now().Add(-time.Hour).Unix()),
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_Audience(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:   testSecret,
		Audience: []string{"my-api", "other-api"},
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid audience string", func(t *testing.T) {
		token := createToken(map[string]any{
			"sub": "user123",
			"aud": "my-api",
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

	t.Run("valid audience array", func(t *testing.T) {
		token := createToken(map[string]any{
			"sub": "user123",
			"aud": []any{"my-api", "another-api"},
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

	t.Run("invalid audience", func(t *testing.T) {
		token := createToken(map[string]any{
			"sub": "user123",
			"aud": "wrong-api",
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

	t.Run("missing audience", func(t *testing.T) {
		token := createToken(map[string]any{
			"sub": "user123",
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d for missing aud, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestWithOptions_IssuerMissing(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret: testSecret,
		Issuer: "required-issuer",
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Token without issuer
	token := createToken(map[string]any{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d for missing issuer, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGetClaims_NoClaims(t *testing.T) {
	app := mizu.NewRouter()

	var claims map[string]any
	app.Get("/test", func(c *mizu.Ctx) error {
		claims = GetClaims(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if claims != nil {
		t.Error("expected nil claims without JWT middleware")
	}
}

func TestSubject_NoClaims(t *testing.T) {
	app := mizu.NewRouter()

	var sub string
	app.Get("/test", func(c *mizu.Ctx) error {
		sub = Subject(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if sub != "" {
		t.Errorf("expected empty subject, got %q", sub)
	}
}

func TestSubject_NoSubClaim(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	var sub string
	app.Get("/test", func(c *mizu.Ctx) error {
		sub = Subject(c)
		return c.Text(http.StatusOK, "ok")
	})

	// Token without sub claim
	token := createToken(map[string]any{
		"email": "user@example.com",
		"exp":   float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if sub != "" {
		t.Errorf("expected empty subject when no sub claim, got %q", sub)
	}
}

func TestWithOptions_CookieLookup(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:      testSecret,
		TokenLookup: "cookie:jwt",
		AuthScheme:  "",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createToken(map[string]any{
		"sub": "user",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_CookieMissing(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:      testSecret,
		TokenLookup: "cookie:jwt",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for missing cookie, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWithOptions_InvalidScheme(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createToken(map[string]any{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d for invalid scheme, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWithOptions_CustomErrorHandler(t *testing.T) {
	errorHandlerCalled := false
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret: testSecret,
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			errorHandlerCalled = true
			return c.Text(http.StatusTeapot, "custom error")
		},
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !errorHandlerCalled {
		t.Error("expected custom error handler to be called")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("expected %d, got %d", http.StatusTeapot, rec.Code)
	}
}

func TestWithOptions_CustomAuthScheme(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:     testSecret,
		AuthScheme: "Token",
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createToken(map[string]any{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestValidateToken_MalformedParts(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	tests := []struct {
		name  string
		token string
	}{
		{"no dots", "nodots"},
		{"one dot", "one.dot"},
		{"invalid base64 signature", "header.payload.!!!invalid!!!"},
		{"invalid base64 payload", "eyJhbGciOiJIUzI1NiJ9.!!!invalid!!!.signature"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("expected %d for %s, got %d", http.StatusForbidden, tc.name, rec.Code)
			}
		})
	}
}

func TestValidateToken_InvalidJSON(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(testSecret))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Create token with invalid JSON payload
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	message := header + "." + payload
	h := hmac.New(sha256.New, testSecret)
	h.Write([]byte(message))
	sig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	token := message + "." + sig

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d for invalid JSON, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWithOptions_QueryMissing(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:      testSecret,
		TokenLookup: "query:token",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for missing query token, got %d", http.StatusUnauthorized, rec.Code)
	}
}
