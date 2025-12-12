package language

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("en", "es", "fr"))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, detectedLang)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "es" {
		t.Errorf("expected 'es', got %q", detectedLang)
	}
}

func TestWithOptions_QueryParam(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Supported:  []string{"en", "de"},
		QueryParam: "language",
	}))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?language=de", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "de" {
		t.Errorf("expected 'de', got %q", detectedLang)
	}
}

func TestWithOptions_Cookie(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Supported:  []string{"en", "ja"},
		CookieName: "user_lang",
	}))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "user_lang", Value: "ja"})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "ja" {
		t.Errorf("expected 'ja', got %q", detectedLang)
	}
}

func TestWithOptions_PathPrefix(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Supported:  []string{"en", "fr"},
		PathPrefix: true,
	}))

	var detectedLang, path string
	app.Get("/page", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		path = c.Request().URL.Path
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/fr/page", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "fr" {
		t.Errorf("expected 'fr', got %q", detectedLang)
	}
	if path != "/page" {
		t.Errorf("expected '/page', got %q", path)
	}
}

func TestWithOptions_AcceptLanguageQuality(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("en", "es", "fr"))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	// fr has higher quality
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en;q=0.5, fr;q=0.9, es;q=0.3")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "fr" {
		t.Errorf("expected 'fr', got %q", detectedLang)
	}
}

func TestWithOptions_AcceptLanguageRegion(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("en", "en-US", "en-GB"))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en-GB")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "en-GB" {
		t.Errorf("expected 'en-GB', got %q", detectedLang)
	}
}

func TestWithOptions_Fallback(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Supported: []string{"en", "es"},
		Default:   "en",
	}))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	// Request unsupported language
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "en" {
		t.Errorf("expected 'en' (default), got %q", detectedLang)
	}
}

func TestWithOptions_Priority(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Supported:  []string{"en", "es", "fr"},
		QueryParam: "lang",
	}))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	// Query param should take priority over Accept-Language
	req := httptest.NewRequest(http.MethodGet, "/?lang=es", nil)
	req.Header.Set("Accept-Language", "fr")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "es" {
		t.Errorf("expected 'es' (query param), got %q", detectedLang)
	}
}

func TestFromContext(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("en"))

	var lang1, lang2 string
	app.Get("/", func(c *mizu.Ctx) error {
		lang1 = Get(c)
		lang2 = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if lang1 != lang2 {
		t.Error("Get and FromContext should return same value")
	}
}

func TestWithOptions_NoHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Supported: []string{"en"},
		Default:   "en",
	}))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "en" {
		t.Errorf("expected 'en', got %q", detectedLang)
	}
}

func TestWithOptions_CaseInsensitive(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("en", "ES"))

	var detectedLang string
	app.Get("/", func(c *mizu.Ctx) error {
		detectedLang = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?lang=es", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if detectedLang != "ES" {
		t.Errorf("expected 'ES', got %q", detectedLang)
	}
}
