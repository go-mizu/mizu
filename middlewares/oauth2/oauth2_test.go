package oauth2

import (
	"net/http"
	"net/http/httptest"
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
