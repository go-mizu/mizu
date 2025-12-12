package errorpage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/notfound", func(c *mizu.Ctx) error {
		c.Writer().WriteHeader(http.StatusNotFound)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "404") {
		t.Error("expected 404 in body")
	}
	if !strings.Contains(rec.Body.String(), "Not Found") {
		t.Error("expected 'Not Found' in body")
	}
}

func TestWithOptions_CustomPages(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Pages: map[int]*Page{
			404: {Code: 404, Title: "Oops!", Message: "The page was not found."},
		},
	}))

	app.Get("/notfound", func(c *mizu.Ctx) error {
		c.Writer().WriteHeader(http.StatusNotFound)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Oops!") {
		t.Error("expected custom title 'Oops!'")
	}
	if !strings.Contains(body, "The page was not found.") {
		t.Error("expected custom message")
	}
}

func TestWithOptions_CustomTemplate(t *testing.T) {
	customTemplate := `<html><body><h1>Error {{.Code}}: {{.Title}}</h1></body></html>`

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		DefaultTemplate: customTemplate,
	}))

	app.Get("/error", func(c *mizu.Ctx) error {
		c.Writer().WriteHeader(http.StatusInternalServerError)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "Error 500:") {
		t.Error("expected custom template format")
	}
}

func TestWithOptions_NotFoundHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		NotFoundHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		},
	}))

	app.Get("/notfound", func(c *mizu.Ctx) error {
		c.Writer().WriteHeader(http.StatusNotFound)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), `"error":"not found"`) {
		t.Error("expected JSON error response")
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ErrorHandler: func(c *mizu.Ctx, code int) error {
			return c.Text(code, "Custom error handler")
		},
	}))

	app.Get("/error", func(c *mizu.Ctx) error {
		c.Writer().WriteHeader(http.StatusBadRequest)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "Custom error handler" {
		t.Errorf("expected custom handler response, got %q", rec.Body.String())
	}
}

func TestWithOptions_SuccessDoesNotShowErrorPage(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/success", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/success", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "success" {
		t.Errorf("expected 'success', got %q", rec.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(NotFound())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestCustom(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Custom(map[int]*Page{
		503: {Code: 503, Title: "Maintenance", Message: "We'll be back soon!"},
	}))

	app.Get("/maintenance", func(c *mizu.Ctx) error {
		c.Writer().WriteHeader(http.StatusServiceUnavailable)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/maintenance", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "Maintenance") {
		t.Error("expected 'Maintenance' in response")
	}
}

func TestPage404(t *testing.T) {
	page := Page404("Custom 404", "Page not found")

	if page.Code != 404 {
		t.Errorf("expected code 404, got %d", page.Code)
	}
	if page.Title != "Custom 404" {
		t.Errorf("expected title 'Custom 404', got %q", page.Title)
	}
}

func TestPage500(t *testing.T) {
	page := Page500("Server Error", "Something broke")

	if page.Code != 500 {
		t.Errorf("expected code 500, got %d", page.Code)
	}
	if page.Title != "Server Error" {
		t.Errorf("expected title 'Server Error', got %q", page.Title)
	}
}

func TestDefaultPages(t *testing.T) {
	codes := []int{400, 401, 403, 404, 405, 500, 502, 503, 504}

	for _, code := range codes {
		page := defaultPages[code]
		if page == nil {
			t.Errorf("expected default page for %d", code)
		}
		if page.Code != code {
			t.Errorf("expected code %d, got %d", code, page.Code)
		}
	}
}
