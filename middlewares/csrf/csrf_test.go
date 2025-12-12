package csrf

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

var testSecret = []byte("test-secret-key-for-csrf-testing")

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{Secret: testSecret}))

	app.Get("/form", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, Token(c))
	})

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	t.Run("sets cookie on GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/form", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		cookies := rec.Result().Cookies()
		var csrfCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "_csrf" {
				csrfCookie = c
				break
			}
		}
		if csrfCookie == nil {
			t.Error("expected _csrf cookie to be set")
		}
	})

	t.Run("rejects POST without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/submit", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("accepts POST with valid token", func(t *testing.T) {
		// First, get the token
		getReq := httptest.NewRequest(http.MethodGet, "/form", nil)
		getRec := httptest.NewRecorder()
		app.ServeHTTP(getRec, getReq)

		var csrfCookie *http.Cookie
		for _, c := range getRec.Result().Cookies() {
			if c.Name == "_csrf" {
				csrfCookie = c
				break
			}
		}
		if csrfCookie == nil {
			t.Fatal("no csrf cookie found")
		}

		token := getRec.Body.String()

		// Now POST with the token
		req := httptest.NewRequest(http.MethodPost, "/submit", nil)
		req.AddCookie(csrfCookie)
		req.Header.Set("X-Csrf-Token", token)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("rejects POST with invalid token", func(t *testing.T) {
		// Get a valid cookie
		getReq := httptest.NewRequest(http.MethodGet, "/form", nil)
		getRec := httptest.NewRecorder()
		app.ServeHTTP(getRec, getReq)

		var csrfCookie *http.Cookie
		for _, c := range getRec.Result().Cookies() {
			if c.Name == "_csrf" {
				csrfCookie = c
				break
			}
		}

		req := httptest.NewRequest(http.MethodPost, "/submit", nil)
		req.AddCookie(csrfCookie)
		req.Header.Set("X-Csrf-Token", "invalid-token")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestNew_FormToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		Secret:      testSecret,
		TokenLookup: "form:_csrf",
	}))

	app.Get("/form", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, Token(c))
	})

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	// Get token
	getReq := httptest.NewRequest(http.MethodGet, "/form", nil)
	getRec := httptest.NewRecorder()
	app.ServeHTTP(getRec, getReq)

	var csrfCookie *http.Cookie
	for _, c := range getRec.Result().Cookies() {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}
	token := getRec.Body.String()

	// POST with form token
	form := url.Values{}
	form.Set("_csrf", token)
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNew_QueryToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		Secret:      testSecret,
		TokenLookup: "query:_csrf",
	}))

	app.Get("/form", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, Token(c))
	})

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	// Get token
	getReq := httptest.NewRequest(http.MethodGet, "/form", nil)
	getRec := httptest.NewRecorder()
	app.ServeHTTP(getRec, getReq)

	var csrfCookie *http.Cookie
	for _, c := range getRec.Result().Cookies() {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}
	token := getRec.Body.String()

	// POST with query token
	req := httptest.NewRequest(http.MethodPost, "/submit?_csrf="+url.QueryEscape(token), nil)
	req.AddCookie(csrfCookie)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNew_SkipPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		Secret:    testSecret,
		SkipPaths: []string{"/api/webhook"},
	}))

	app.Post("/api/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "webhook")
	})

	app.Post("/api/other", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "other")
	})

	t.Run("skips listed path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/webhook", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("protects non-listed path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/other", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestNew_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		Secret: testSecret,
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": err.Error(),
			})
		},
	}))

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "token missing") {
		t.Error("expected error message in response")
	}
}

func TestTemplateField(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{Secret: testSecret}))

	var field string
	app.Get("/form", func(c *mizu.Ctx) error {
		field = TemplateField(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/form", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.HasPrefix(field, `<input type="hidden" name="_csrf"`) {
		t.Errorf("unexpected template field: %s", field)
	}
}

func TestProtect(t *testing.T) {
	mw := Protect(testSecret)
	if mw == nil {
		t.Error("expected middleware")
	}
}

func TestProtectDev(t *testing.T) {
	mw := ProtectDev(testSecret)
	if mw == nil {
		t.Error("expected middleware")
	}
}

func TestGenerateSecret(t *testing.T) {
	secret1 := GenerateSecret()
	secret2 := GenerateSecret()

	if len(secret1) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(secret1))
	}

	if string(secret1) == string(secret2) {
		t.Error("secrets should be unique")
	}
}

func TestNew_Panics(t *testing.T) {
	t.Run("panics without secret", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		New(Options{})
	})

	t.Run("panics with invalid TokenLookup", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		New(Options{
			Secret:      testSecret,
			TokenLookup: "invalid",
		})
	})
}

func TestGenerateToken(t *testing.T) {
	token := generateToken(32, testSecret)

	if token == "" {
		t.Error("expected token")
	}

	if !strings.Contains(token, ".") {
		t.Error("expected token to contain signature separator")
	}

	// Test uniqueness
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		tok := generateToken(32, testSecret)
		if tokens[tok] {
			t.Error("duplicate token generated")
		}
		tokens[tok] = true
	}
}

func TestValidateToken(t *testing.T) {
	token := generateToken(32, testSecret)

	t.Run("valid token", func(t *testing.T) {
		if !validateToken(token, token, testSecret) {
			t.Error("expected token to be valid")
		}
	})

	t.Run("mismatched tokens", func(t *testing.T) {
		other := generateToken(32, testSecret)
		if validateToken(token, other, testSecret) {
			t.Error("expected validation to fail for different tokens")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		wrongSecret := []byte("wrong-secret")
		if validateToken(token, token, wrongSecret) {
			t.Error("expected validation to fail with wrong secret")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		if validateToken("malformed", "malformed", testSecret) {
			t.Error("expected validation to fail for malformed token")
		}
	})
}
