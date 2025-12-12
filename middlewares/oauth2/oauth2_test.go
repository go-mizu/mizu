package oauth2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	validator := func(token string) (*Token, error) {
		if token == "valid-token" {
			return &Token{
				Value:   token,
				Subject: "user123",
				Scope:   []string{"read", "write"},
			}, nil
		}
		return nil, ErrInvalidToken
	}

	app := mizu.NewRouter()
	app.Use(New(validator))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestWithOptions_RequiredScopes(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{
			Value: token,
			Scope: []string{"read"},
		}, nil
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator:      validator,
		RequiredScopes: []string{"write"},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d for insufficient scope, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWithOptions_ExpiredToken(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{
			Value:     token,
			ExpiresAt: time.Now().Add(-time.Hour),
		}, nil
	}

	app := mizu.NewRouter()
	app.Use(New(validator))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for expired token, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestGet(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{
			Value:   token,
			Subject: "user456",
		}, nil
	}

	app := mizu.NewRouter()
	app.Use(New(validator))

	var tokenInfo *Token
	app.Get("/", func(c *mizu.Ctx) error {
		tokenInfo = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer mytoken")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if tokenInfo == nil || tokenInfo.Subject != "user456" {
		t.Errorf("expected token with subject user456, got %v", tokenInfo)
	}
}

func TestSubject(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Subject: "test-subject"}, nil
	}

	app := mizu.NewRouter()
	app.Use(New(validator))

	var subject string
	app.Get("/", func(c *mizu.Ctx) error {
		subject = Subject(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if subject != "test-subject" {
		t.Errorf("expected 'test-subject', got %q", subject)
	}
}

func TestHasScope(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Scope: []string{"read", "write"}}, nil
	}

	app := mizu.NewRouter()
	app.Use(New(validator))

	var hasRead, hasDelete bool
	app.Get("/", func(c *mizu.Ctx) error {
		hasRead = HasScope(c, "read")
		hasDelete = HasScope(c, "delete")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !hasRead {
		t.Error("expected HasScope(read) to be true")
	}
	if hasDelete {
		t.Error("expected HasScope(delete) to be false")
	}
}

func TestRequireScopes(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Scope: []string{"read"}}, nil
	}

	app := mizu.NewRouter()
	app.Use(New(validator))
	app.Use(RequireScopes("read", "write"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d for missing scope, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWWWAuthenticateHeader(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return nil, ErrInvalidToken
	}

	app := mizu.NewRouter()
	app.Use(New(validator))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestExtractToken_Query(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Value: token, Subject: "test"}, nil
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator:   validator,
		TokenLookup: "query:access_token",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?access_token=mytoken", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestExtractToken_Form(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Value: token, Subject: "test"}, nil
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator:   validator,
		TokenLookup: "form:access_token",
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("access_token=mytoken"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestExtractToken_InvalidLookup(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Value: token}, nil
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator:   validator,
		TokenLookup: "invalid",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for invalid lookup, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestExtractToken_UnknownSource(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Value: token}, nil
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator:   validator,
		TokenLookup: "unknown:field",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for unknown source, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestExtractToken_HeaderWithoutBearer(t *testing.T) {
	validator := func(token string) (*Token, error) {
		if token == "plain-token" {
			return &Token{Value: token}, nil
		}
		return nil, ErrInvalidToken
	}

	app := mizu.NewRouter()
	app.Use(New(validator))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "plain-token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_NoValidator(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for no validator, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWithOptions_CustomErrorHandler(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return nil, ErrInvalidToken
	}

	errorHandlerCalled := false
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator: validator,
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			errorHandlerCalled = true
			return c.Text(http.StatusTeapot, "custom error")
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !errorHandlerCalled {
		t.Error("expected custom error handler to be called")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("expected %d from custom handler, got %d", http.StatusTeapot, rec.Code)
	}
}

func TestIntrospectToken(t *testing.T) {
	// Mock introspection server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		token := r.FormValue("token")
		w.Header().Set("Content-Type", "application/json")

		if token == "active-token" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"active": true,
				"sub":    "user123",
				"iss":    "test-issuer",
				"scope":  "read write",
				"exp":    time.Now().Add(time.Hour).Unix(),
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"active": false,
			})
		}
	}))
	defer server.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IntrospectionURL: server.URL,
		ClientID:         "client",
		ClientSecret:     "secret",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("active token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer active-token")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("inactive token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer inactive-token")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestIntrospectToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IntrospectionURL: server.URL,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d on server error, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestIntrospectToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IntrospectionURL: server.URL,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d on invalid JSON, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestIntrospectToken_ConnectionError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IntrospectionURL: "http://localhost:99999/introspect",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d on connection error, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestGet_NoToken(t *testing.T) {
	app := mizu.NewRouter()

	var token *Token
	app.Get("/", func(c *mizu.Ctx) error {
		token = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if token != nil {
		t.Error("expected nil token when no middleware")
	}
}

func TestSubject_NoToken(t *testing.T) {
	app := mizu.NewRouter()

	var subject string
	app.Get("/", func(c *mizu.Ctx) error {
		subject = Subject(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if subject != "" {
		t.Errorf("expected empty subject, got %q", subject)
	}
}

func TestScopes(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Scope: []string{"read", "write"}}, nil
	}

	app := mizu.NewRouter()
	app.Use(New(validator))

	var scopes []string
	app.Get("/", func(c *mizu.Ctx) error {
		scopes = Scopes(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(scopes) != 2 || scopes[0] != "read" || scopes[1] != "write" {
		t.Errorf("expected [read, write], got %v", scopes)
	}
}

func TestScopes_NoToken(t *testing.T) {
	app := mizu.NewRouter()

	var scopes []string
	app.Get("/", func(c *mizu.Ctx) error {
		scopes = Scopes(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if scopes != nil {
		t.Errorf("expected nil scopes, got %v", scopes)
	}
}

func TestRequireScopes_NoToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(RequireScopes("read"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d for no token, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireScopes_Sufficient(t *testing.T) {
	validator := func(token string) (*Token, error) {
		return &Token{Scope: []string{"read", "write"}}, nil
	}

	app := mizu.NewRouter()
	app.Use(New(validator))
	app.Use(RequireScopes("read"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with sufficient scopes, got %d", http.StatusOK, rec.Code)
	}
}

func TestHasScope_NoToken(t *testing.T) {
	app := mizu.NewRouter()

	var hasRead bool
	app.Get("/", func(c *mizu.Ctx) error {
		hasRead = HasScope(c, "read")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if hasRead {
		t.Error("expected HasScope to return false without token")
	}
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		err      oauthError
		expected string
	}{
		{ErrMissingToken, "missing access token"},
		{ErrInvalidToken, "invalid access token"},
		{ErrExpiredToken, "access token expired"},
		{ErrInsufficientScope, "insufficient scope"},
		{ErrNoValidator, "no token validator configured"},
	}

	for _, tc := range tests {
		if tc.err.Error() != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, tc.err.Error())
		}
	}
}

func TestWithOptions_IntrospectionWithoutClientCreds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No basic auth expected
		if _, _, ok := r.BasicAuth(); ok {
			t.Error("expected no basic auth")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active": true,
			"sub":    "user",
			"exp":    time.Now().Add(time.Hour).Unix(),
		})
	}))
	defer server.Close()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		IntrospectionURL: server.URL,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
