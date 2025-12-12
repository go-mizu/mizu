package oidc

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func createTestToken(claims map[string]any) string {
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// For testing, we use a dummy signature
	signature := base64.RawURLEncoding.EncodeToString([]byte("test-signature"))

	return headerB64 + "." + claimsB64 + "." + signature
}

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("https://issuer.example.com", "client-id"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Without token - should fail
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d without token, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWithValidToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			return c.Text(http.StatusInternalServerError, "no claims")
		}
		return c.Text(http.StatusOK, claims.Subject)
	})

	token := createTestToken(map[string]any{
		"iss": "https://issuer.example.com",
		"sub": "user-123",
		"aud": "client-id",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with valid token, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "user-123" {
		t.Errorf("expected subject 'user-123', got %q", rec.Body.String())
	}
}

func TestExpiredToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createTestToken(map[string]any{
		"iss": "https://issuer.example.com",
		"sub": "user-123",
		"aud": "client-id",
		"exp": time.Now().Add(-time.Hour).Unix(), // Expired
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for expired token, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestInvalidIssuer(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createTestToken(map[string]any{
		"iss": "https://wrong-issuer.com",
		"sub": "user-123",
		"aud": "client-id",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for invalid issuer, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestInvalidAudience(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createTestToken(map[string]any{
		"iss": "https://issuer.example.com",
		"sub": "user-123",
		"aud": "wrong-client",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for invalid audience, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestSkipPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
		SkipPaths: []string{"/health", "/public"},
	}))

	app.Get("/health", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "healthy")
	})
	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "protected")
	})

	// Health check should work without token
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for skipped path, got %d", http.StatusOK, rec.Code)
	}

	// Protected should require token
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for protected path, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCustomTokenExtractor(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
		TokenExtractor: func(r *http.Request) string {
			return r.URL.Query().Get("token")
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	token := createTestToken(map[string]any{
		"iss": "https://issuer.example.com",
		"sub": "user-123",
		"aud": "client-id",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/?token="+token, nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with token in query, got %d", http.StatusOK, rec.Code)
	}
}

func TestCustomOnError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
		OnError: func(c *mizu.Ctx, err error) error {
			return c.JSON(http.StatusForbidden, map[string]string{
				"custom_error": err.Error(),
			})
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected custom error code %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestClaimsHasAudience(t *testing.T) {
	t.Run("string audience", func(t *testing.T) {
		claims := &Claims{Audience: "client-id"}
		if !claims.HasAudience("client-id") {
			t.Error("expected to find audience")
		}
		if claims.HasAudience("other") {
			t.Error("expected not to find audience")
		}
	})

	t.Run("array audience", func(t *testing.T) {
		claims := &Claims{Audience: []any{"client-1", "client-2"}}
		if !claims.HasAudience("client-1") {
			t.Error("expected to find audience")
		}
		if !claims.HasAudience("client-2") {
			t.Error("expected to find audience")
		}
		if claims.HasAudience("client-3") {
			t.Error("expected not to find audience")
		}
	})
}

func TestClaimsHasGroup(t *testing.T) {
	claims := &Claims{Groups: []string{"admin", "users"}}

	if !claims.HasGroup("admin") {
		t.Error("expected to find group admin")
	}
	if claims.HasGroup("superadmin") {
		t.Error("expected not to find group")
	}
}

func TestClaimsHasRole(t *testing.T) {
	claims := &Claims{Roles: []string{"editor", "viewer"}}

	if !claims.HasRole("editor") {
		t.Error("expected to find role editor")
	}
	if claims.HasRole("admin") {
		t.Error("expected not to find role")
	}
}

func TestRequireGroup(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
	}))

	// Chain the RequireGroup middleware with the handler
	adminHandler := RequireGroup("admin")(func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "admin access")
	})
	app.Get("/admin", adminHandler)

	// Token with admin group
	adminToken := createTestToken(map[string]any{
		"iss":    "https://issuer.example.com",
		"sub":    "user-123",
		"aud":    "client-id",
		"exp":    time.Now().Add(time.Hour).Unix(),
		"groups": []string{"admin"},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for admin group, got %d", http.StatusOK, rec.Code)
	}

	// Token without admin group
	userToken := createTestToken(map[string]any{
		"iss":    "https://issuer.example.com",
		"sub":    "user-123",
		"aud":    "client-id",
		"exp":    time.Now().Add(time.Hour).Unix(),
		"groups": []string{"users"},
	})

	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d without admin group, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireScope(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IssuerURL: "https://issuer.example.com",
		ClientID:  "client-id",
	}))

	// Chain the RequireScope middleware with the handler
	apiHandler := RequireScope("read:api")(func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api access")
	})
	app.Get("/api", apiHandler)

	// Token with required scope
	token := createTestToken(map[string]any{
		"iss":   "https://issuer.example.com",
		"sub":   "user-123",
		"aud":   "client-id",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"scope": "read:api write:api",
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with scope, got %d", http.StatusOK, rec.Code)
	}

	// Token without required scope
	tokenNoScope := createTestToken(map[string]any{
		"iss":   "https://issuer.example.com",
		"sub":   "user-123",
		"aud":   "client-id",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"scope": "other:scope",
	})

	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Authorization", "Bearer "+tokenNoScope)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d without scope, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestInvalidToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("https://issuer.example.com", "client-id"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for invalid token, got %d", http.StatusUnauthorized, rec.Code)
	}
}
