package captcha

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestWithOptions_MissingToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider: ProviderCustom,
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return true, nil
		},
	}))

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for missing token, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWithOptions_CustomVerifier(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderCustom,
		TokenLookup: "form:captcha",
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return token == "valid-token", nil
		},
	}))

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid token", func(t *testing.T) {
		form := url.Values{}
		form.Set("captcha", "valid-token")

		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		form := url.Values{}
		form.Set("captcha", "invalid-token")

		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestWithOptions_SkipSafeMethods(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider: ProviderCustom,
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return false, nil // Would fail if called
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected GET to skip captcha, got %d", rec.Code)
	}
}

func TestWithOptions_SkipPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:  ProviderCustom,
		SkipPaths: []string{"/webhook"},
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return false, nil
		},
	}))

	app.Post("/webhook", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected skip path to work, got %d", rec.Code)
	}
}

func TestWithOptions_HeaderToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderCustom,
		TokenLookup: "header:X-Captcha-Token",
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return token == "header-token", nil
		},
	}))

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.Header.Set("X-Captcha-Token", "header-token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_QueryToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderCustom,
		TokenLookup: "query:token",
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return token == "query-token", nil
		},
	}))

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/submit?token=query-token", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_CustomErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderCustom,
		TokenLookup: "form:captcha",
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return false, nil
		},
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "captcha failed",
			})
		},
	}))

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	form := url.Values{}
	form.Set("captcha", "bad-token")

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected custom error code %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestCustom(t *testing.T) {
	verifier := func(token string, c *mizu.Ctx) (bool, error) {
		return token == "custom-valid", nil
	}

	app := mizu.NewRouter()
	app.Use(Custom(verifier, "form:my-captcha"))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	form := url.Values{}
	form.Set("my-captcha", "custom-valid")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
