package csrf2

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	// Should set CSRF cookie
	cookies := rec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}

	if csrfCookie == nil {
		t.Error("expected CSRF cookie to be set")
	}
}

func TestGetToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	var token string

	app.Get("/", func(c *mizu.Ctx) error {
		token = GetToken(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if token == "" {
		t.Error("expected token in context")
	}
}

func TestValidateTokenHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	var token string

	app.Get("/", func(c *mizu.Ctx) error {
		token = GetToken(c)
		return c.Text(http.StatusOK, "ok")
	})

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	// First get the token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get cookie
	cookies := rec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}

	// POST with valid token
	req = httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.AddCookie(csrfCookie)
	req.Header.Set("X-Csrf-Token", token)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with valid token, got %d", http.StatusOK, rec.Code)
	}
}

func TestValidateTokenForm(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	var token string

	app.Get("/", func(c *mizu.Ctx) error {
		token = GetToken(c)
		return c.Text(http.StatusOK, "ok")
	})

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	// First get the token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}

	// POST with token in form
	form := "_csrf=" + token
	req = httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with form token, got %d", http.StatusOK, rec.Code)
	}
}

func TestMissingToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	// POST without token
	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d without token, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestInvalidToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	// Get cookie
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}

	// POST with invalid token
	req = httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.AddCookie(csrfCookie)
	req.Header.Set("X-Csrf-Token", "invalid-token")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d with invalid token, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestSkipPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:    "test-secret",
		SkipPaths: []string{"/api/webhook"},
	}))

	app.Post("/api/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "webhook")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/webhook", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for skipped path, got %d", http.StatusOK, rec.Code)
	}
}

func TestSkipMethods(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Handle("OPTIONS", "/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// GET should be skipped
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for GET, got %d", http.StatusOK, rec.Code)
	}

	// OPTIONS should be skipped
	req = httptest.NewRequest(http.MethodOptions, "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for OPTIONS, got %d", http.StatusOK, rec.Code)
	}
}

func TestCustomErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret: "test-secret",
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "custom csrf error",
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
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}

	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Error("expected JSON response from custom handler")
	}
}

func TestCustomCookieName(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:     "test-secret",
		CookieName: "my-csrf",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "my-csrf" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected custom cookie name")
	}
}

func TestCustomHeaderName(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Secret:     "test-secret",
		HeaderName: "X-My-Csrf",
	}))

	var token string

	app.Get("/", func(c *mizu.Ctx) error {
		token = GetToken(c)
		return c.Text(http.StatusOK, "ok")
	})

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	// Get token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_csrf" {
			csrfCookie = c
			break
		}
	}

	// POST with custom header
	req = httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.AddCookie(csrfCookie)
	req.Header.Set("X-My-Csrf", token)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with custom header, got %d", http.StatusOK, rec.Code)
	}
}

func TestTokenHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Get("/csrf-token", Token())

	req := httptest.NewRequest(http.MethodGet, "/csrf-token", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "token") {
		t.Error("expected token in response")
	}
}

func TestFormInput(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Get("/", func(c *mizu.Ctx) error {
		html := FormInput(c, "")
		return c.HTML(http.StatusOK, html)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `type="hidden"`) {
		t.Error("expected hidden input")
	}
	if !strings.Contains(body, `name="_csrf"`) {
		t.Error("expected _csrf name")
	}
}

func TestMetaTag(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("test-secret"))

	app.Get("/", func(c *mizu.Ctx) error {
		meta := MetaTag(c)
		return c.HTML(http.StatusOK, meta)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `name="csrf-token"`) {
		t.Error("expected csrf-token meta tag")
	}
}

func TestMaskUnmask(t *testing.T) {
	original := "test-token-value"
	masked := Mask(original)

	// Masked should be different
	if masked == original {
		t.Error("masked token should be different")
	}

	// Unmasked should match original
	unmasked := Unmask(masked)
	if unmasked != original {
		t.Errorf("expected %q after unmask, got %q", original, unmasked)
	}
}

func TestFingerprint(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		fp := Fingerprint(c)
		return c.Text(http.StatusOK, fp)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "TestAgent")
	req.Header.Set("Accept-Language", "en-US")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	fp1 := rec.Body.String()

	// Same headers should produce same fingerprint
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "TestAgent")
	req.Header.Set("Accept-Language", "en-US")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	fp2 := rec.Body.String()

	if fp1 != fp2 {
		t.Error("expected same fingerprint for same headers")
	}

	// Different headers should produce different fingerprint
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "DifferentAgent")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	fp3 := rec.Body.String()

	if fp1 == fp3 {
		t.Error("expected different fingerprint for different headers")
	}
}
