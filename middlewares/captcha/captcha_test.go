package captcha

import (
	"errors"
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

func TestNew(t *testing.T) {
	middleware := New("test-secret")
	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestErrors(t *testing.T) {
	if ErrMissingToken.Error() != "captcha token missing" {
		t.Errorf("unexpected error message: %s", ErrMissingToken.Error())
	}
	if ErrInvalidToken.Error() != "captcha verification failed" {
		t.Errorf("unexpected error message: %s", ErrInvalidToken.Error())
	}
	if ErrVerifyFailed.Error() != "captcha verification request failed" {
		t.Errorf("unexpected error message: %s", ErrVerifyFailed.Error())
	}
}

func TestExtractToken_InvalidLookup(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderCustom,
		TokenLookup: "invalid", // No colon separator
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return true, nil
		},
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for invalid lookup, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestVerifyToken_WithMockServer(t *testing.T) {
	// Create mock verification server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := r.FormValue("response")
		if response == "valid-token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success": true, "score": 0.9}`))
		} else {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success": false}`))
		}
	}))
	defer server.Close()

	// Override the verification URL for testing
	originalURL := verifyURLs[ProviderRecaptchaV2]
	verifyURLs[ProviderRecaptchaV2] = server.URL
	defer func() { verifyURLs[ProviderRecaptchaV2] = originalURL }()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderRecaptchaV2,
		Secret:      "test-secret",
		TokenLookup: "form:g-recaptcha-response",
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("valid token", func(t *testing.T) {
		form := url.Values{}
		form.Set("g-recaptcha-response", "valid-token")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		form := url.Values{}
		form.Set("g-recaptcha-response", "invalid-token")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestVerifyToken_RecaptchaV3Score(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := r.FormValue("response")
		w.Header().Set("Content-Type", "application/json")
		if response == "high-score" {
			_, _ = w.Write([]byte(`{"success": true, "score": 0.9}`))
		} else {
			_, _ = w.Write([]byte(`{"success": true, "score": 0.3}`))
		}
	}))
	defer server.Close()

	originalURL := verifyURLs[ProviderRecaptchaV3]
	verifyURLs[ProviderRecaptchaV3] = server.URL
	defer func() { verifyURLs[ProviderRecaptchaV3] = originalURL }()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderRecaptchaV3,
		Secret:      "test-secret",
		TokenLookup: "form:g-recaptcha-response",
		MinScore:    0.5,
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("score above threshold", func(t *testing.T) {
		form := url.Values{}
		form.Set("g-recaptcha-response", "high-score")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("score below threshold", func(t *testing.T) {
		form := url.Values{}
		form.Set("g-recaptcha-response", "low-score")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d for low score, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestVerifyToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := verifyURLs[ProviderRecaptchaV2]
	verifyURLs[ProviderRecaptchaV2] = server.URL
	defer func() { verifyURLs[ProviderRecaptchaV2] = originalURL }()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderRecaptchaV2,
		Secret:      "test-secret",
		TokenLookup: "form:g-recaptcha-response",
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	form := url.Values{}
	form.Set("g-recaptcha-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for server error, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestVerifyToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	originalURL := verifyURLs[ProviderRecaptchaV2]
	verifyURLs[ProviderRecaptchaV2] = server.URL
	defer func() { verifyURLs[ProviderRecaptchaV2] = originalURL }()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderRecaptchaV2,
		Secret:      "test-secret",
		TokenLookup: "form:g-recaptcha-response",
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	form := url.Values{}
	form.Set("g-recaptcha-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for invalid JSON, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestVerifyToken_UnknownProvider(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    Provider("unknown"),
		Secret:      "test-secret",
		TokenLookup: "form:token",
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	form := url.Values{}
	form.Set("token", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for unknown provider, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that remoteip was sent
		remoteIP := r.FormValue("remoteip")
		w.Header().Set("Content-Type", "application/json")
		if remoteIP == "1.2.3.4" {
			_, _ = w.Write([]byte(`{"success": true}`))
		} else {
			_, _ = w.Write([]byte(`{"success": false}`))
		}
	}))
	defer server.Close()

	originalURL := verifyURLs[ProviderRecaptchaV2]
	verifyURLs[ProviderRecaptchaV2] = server.URL
	defer func() { verifyURLs[ProviderRecaptchaV2] = originalURL }()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderRecaptchaV2,
		Secret:      "test-secret",
		TokenLookup: "form:g-recaptcha-response",
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	form := url.Values{}
	form.Set("g-recaptcha-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d when IP matches, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteIP := r.FormValue("remoteip")
		w.Header().Set("Content-Type", "application/json")
		if remoteIP == "9.9.9.9" {
			_, _ = w.Write([]byte(`{"success": true}`))
		} else {
			_, _ = w.Write([]byte(`{"success": false}`))
		}
	}))
	defer server.Close()

	originalURL := verifyURLs[ProviderRecaptchaV2]
	verifyURLs[ProviderRecaptchaV2] = server.URL
	defer func() { verifyURLs[ProviderRecaptchaV2] = originalURL }()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderRecaptchaV2,
		Secret:      "test-secret",
		TokenLookup: "form:g-recaptcha-response",
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	form := url.Values{}
	form.Set("g-recaptcha-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Real-IP", "9.9.9.9")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d when IP matches, got %d", http.StatusOK, rec.Code)
	}
}

func TestReCaptchaV2(t *testing.T) {
	middleware := ReCaptchaV2("test-secret")
	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestReCaptchaV3(t *testing.T) {
	middleware := ReCaptchaV3("test-secret", 0.7)
	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestHCaptcha(t *testing.T) {
	middleware := HCaptcha("test-secret")
	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestTurnstile(t *testing.T) {
	middleware := Turnstile("test-secret")
	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestWithOptions_VerifierError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider:    ProviderCustom,
		TokenLookup: "form:captcha",
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return false, errors.New("verifier error")
		},
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	form := url.Values{}
	form.Set("captcha", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for verifier error, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWithOptions_HeadMethod(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider: ProviderCustom,
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return false, nil
		},
	}))

	app.Head("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected HEAD to skip captcha, got %d", rec.Code)
	}
}

func TestWithOptions_OptionsMethod(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Provider: ProviderCustom,
		Verifier: func(token string, c *mizu.Ctx) (bool, error) {
			return false, nil
		},
	}))

	// Use Handle with OPTIONS method
	app.Handle(http.MethodOptions, "/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected OPTIONS to skip captcha, got %d", rec.Code)
	}
}
