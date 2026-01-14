package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage/driver/local"
	"github.com/go-mizu/mizu"
)

const testJWTSecret = "test-secret-key-for-jwt-tokens-minimum-32-chars"

func newAuthTestServer(t *testing.T, authConfig AuthConfig) (*httptest.Server, func()) {
	t.Helper()

	ctx := context.Background()
	store, err := local.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}

	app := mizu.New()
	RegisterWithAuth(app, "/storage/v1", store, authConfig)

	srv := httptest.NewServer(app)

	cleanup := func() {
		srv.Close()
		_ = store.Close()
	}

	return srv, cleanup
}

func createTestToken(t *testing.T, claims *Claims, secret string) string {
	t.Helper()
	token, err := createToken(claims, secret)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return token
}

func TestAuthMiddleware(t *testing.T) {
	authConfig := AuthConfig{
		JWTSecret:            testJWTSecret,
		AllowAnonymousPublic: false,
	}
	srv, cleanup := newAuthTestServer(t, authConfig)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{
			name:       "no authorization header",
			token:      "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token format",
			token:      "invalid-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid bearer format",
			token:      "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "valid token",
			token: "Bearer " + createTestToken(t, &Claims{
				Sub:  "user-123",
				Role: "authenticated",
				Exp:  time.Now().Add(1 * time.Hour).Unix(),
			}, testJWTSecret),
			wantStatus: http.StatusOK,
		},
		{
			name: "expired token",
			token: "Bearer " + createTestToken(t, &Claims{
				Sub:  "user-123",
				Role: "authenticated",
				Exp:  time.Now().Add(-1 * time.Hour).Unix(),
			}, testJWTSecret),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "token with wrong secret",
			token: "Bearer " + createTestToken(t, &Claims{
				Sub:  "user-123",
				Role: "authenticated",
				Exp:  time.Now().Add(1 * time.Hour).Unix(),
			}, "wrong-secret-key-definitely-not-the-right-one"),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, base+"/bucket", nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("status = %d, want %d, body = %s", resp.StatusCode, tt.wantStatus, string(body))
			}

			if tt.wantStatus == http.StatusUnauthorized {
				body, _ := io.ReadAll(resp.Body)
				var errResp errorPayload
				if err := json.Unmarshal(body, &errResp); err != nil {
					t.Errorf("failed to decode error response: %v", err)
				}
				if errResp.Error != "Unauthorized" {
					t.Errorf("error = %q, want 'Unauthorized'", errResp.Error)
				}
			}
		})
	}
}

func TestAuthAnonymousPublicAccess(t *testing.T) {
	authConfig := AuthConfig{
		JWTSecret:            testJWTSecret,
		AllowAnonymousPublic: true,
	}
	srv, cleanup := newAuthTestServer(t, authConfig)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create a bucket with valid token first
	validToken := "Bearer " + createTestToken(t, &Claims{
		Sub:  "user-123",
		Role: "authenticated",
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}, testJWTSecret)

	req, _ := http.NewRequest(http.MethodPost, base+"/bucket", strings.NewReader(`{"name":"public-test","public":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", validToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create bucket status = %d", resp.StatusCode)
	}

	// Upload an object with valid token
	uploadReq, _ := http.NewRequest(http.MethodPost, base+"/object/public-test/file.txt", strings.NewReader("hello world"))
	uploadReq.Header.Set("Content-Type", "text/plain")
	uploadReq.Header.Set("Authorization", validToken)

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		t.Fatalf("upload object: %v", err)
	}
	_ = uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK {
		t.Fatalf("upload status = %d", uploadResp.StatusCode)
	}

	// Access public endpoint without auth should work
	t.Run("public endpoint without auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, base+"/object/public/public-test/file.txt", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	// Bucket management without auth should still fail (requires auth)
	t.Run("bucket list without auth fails", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, base+"/bucket", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
	})
}

func TestRequireRole(t *testing.T) {
	// Note: RequireRole middleware is tested via service_account role
	authConfig := AuthConfig{
		JWTSecret:            testJWTSecret,
		AllowAnonymousPublic: false,
	}

	ctx := context.Background()
	store, err := local.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	app := mizu.New()

	// Add a test endpoint with role requirement
	testHandler := func(c *mizu.Ctx) error {
		role, ok := GetRole(c)
		if !ok {
			return c.JSON(http.StatusOK, map[string]string{"role": "unknown"})
		}
		return c.JSON(http.StatusOK, map[string]string{"role": role})
	}

	app.Get("/admin", RequireRole(authConfig, "service_role", "admin")(testHandler))
	app.Get("/user", RequireAuth(authConfig)(testHandler))

	srv := httptest.NewServer(app)
	defer srv.Close()

	tests := []struct {
		name       string
		endpoint   string
		role       string
		wantStatus int
	}{
		{
			name:       "admin endpoint with service_role",
			endpoint:   "/admin",
			role:       "service_role",
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin endpoint with admin role",
			endpoint:   "/admin",
			role:       "admin",
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin endpoint with authenticated role fails",
			endpoint:   "/admin",
			role:       "authenticated",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "user endpoint with any role succeeds",
			endpoint:   "/user",
			role:       "authenticated",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := createTestToken(t, &Claims{
				Sub:  "user-123",
				Role: tt.role,
				Exp:  time.Now().Add(1 * time.Hour).Unix(),
			}, testJWTSecret)

			req, _ := http.NewRequest(http.MethodGet, srv.URL+tt.endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("status = %d, want %d, body = %s", resp.StatusCode, tt.wantStatus, string(body))
			}
		})
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	authConfig := AuthConfig{
		JWTSecret:            testJWTSecret,
		AllowAnonymousPublic: false,
	}

	ctx := context.Background()
	store, err := local.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	app := mizu.New()

	// Add a test endpoint that reads claims
	testHandler := func(c *mizu.Ctx) error {
		claims, ok := GetClaims(c)
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "no claims"})
		}

		userID, _ := GetUserID(c)
		role, _ := GetRole(c)

		return c.JSON(http.StatusOK, map[string]any{
			"sub":           claims.Sub,
			"role":          role,
			"user_id":       userID,
			"authenticated": IsAuthenticated(c),
		})
	}

	app.Get("/me", RequireAuth(authConfig)(testHandler))

	srv := httptest.NewServer(app)
	defer srv.Close()

	token := createTestToken(t, &Claims{
		Sub:  "user-456",
		Role: "authenticated",
		Aud:  "test-audience",
		Iss:  "test-issuer",
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}, testJWTSecret)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result["sub"] != "user-456" {
		t.Errorf("sub = %v, want user-456", result["sub"])
	}
	if result["role"] != "authenticated" {
		t.Errorf("role = %v, want authenticated", result["role"])
	}
	if result["user_id"] != "user-456" {
		t.Errorf("user_id = %v, want user-456", result["user_id"])
	}
	if result["authenticated"] != true {
		t.Errorf("authenticated = %v, want true", result["authenticated"])
	}
}

func TestAuthErrorResponses(t *testing.T) {
	authConfig := AuthConfig{
		JWTSecret:            testJWTSecret,
		AllowAnonymousPublic: false,
	}
	srv, cleanup := newAuthTestServer(t, authConfig)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	tests := []struct {
		name        string
		authHeader  string
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "missing authorization header",
			authHeader:  "",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "missing authorization header",
		},
		{
			name:        "malformed bearer token",
			authHeader:  "Bearer",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "invalid authorization header format",
		},
		{
			name:        "invalid header format",
			authHeader:  "NotBearer token",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "invalid authorization header format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, base+"/bucket", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			body, _ := io.ReadAll(resp.Body)
			var errResp errorPayload
			if err := json.Unmarshal(body, &errResp); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if errResp.StatusCode != tt.wantStatus {
				t.Errorf("errorPayload.statusCode = %d, want %d", errResp.StatusCode, tt.wantStatus)
			}

			if !strings.Contains(errResp.Message, tt.wantMessage) {
				t.Errorf("message = %q, want to contain %q", errResp.Message, tt.wantMessage)
			}
		})
	}
}
