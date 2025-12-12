package fallback

import (
	"strings"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx, err error) error {
		return c.Text(http.StatusInternalServerError, "fallback: "+err.Error())
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("test error")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if rec.Body.String() != "fallback: test error" {
		t.Errorf("expected fallback message, got %q", rec.Body.String())
	}
}

func TestWithOptions_CatchPanic(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		CatchPanic: true,
		Handler: func(c *mizu.Ctx, err error) error {
			return c.Text(http.StatusInternalServerError, "caught panic")
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if rec.Body.String() != "caught panic" {
		t.Errorf("expected panic message, got %q", rec.Body.String())
	}
}

func TestWithOptions_DefaultMessage(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		DefaultMessage: "Custom error message",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("some error")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "Custom error message" {
		t.Errorf("expected custom message, got %q", rec.Body.String())
	}
}

func TestWithOptions_NoError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Handler: func(c *mizu.Ctx, err error) error {
			return c.Text(http.StatusInternalServerError, "should not see this")
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "success" {
		t.Errorf("expected success, got %q", rec.Body.String())
	}
}

func TestDefault(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Default("Something went wrong"))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("error")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "Something went wrong" {
		t.Errorf("expected default message, got %q", rec.Body.String())
	}
}

func TestJSON(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(JSON())

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("json error")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Error("expected JSON content type")
	}
}

func TestRedirect(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Redirect("/error", http.StatusFound))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("error")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected %d, got %d", http.StatusFound, rec.Code)
	}
	if rec.Header().Get("Location") != "/error" {
		t.Errorf("expected redirect to /error, got %q", rec.Header().Get("Location"))
	}
}

func TestChain(t *testing.T) {
	customErr := errors.New("custom error")

	app := mizu.NewRouter()
	app.Use(Chain(
		// First handler only handles custom errors
		func(c *mizu.Ctx, err error) (bool, error) {
			if err == customErr {
				return true, c.Text(http.StatusBadRequest, "custom handled")
			}
			return false, nil
		},
		// Second handler catches everything else
		func(c *mizu.Ctx, err error) (bool, error) {
			return true, c.Text(http.StatusInternalServerError, "generic handled")
		},
	))

	t.Run("custom error", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Use(Chain(
			func(c *mizu.Ctx, err error) (bool, error) {
				if err.Error() == "custom error" {
					return true, c.Text(http.StatusBadRequest, "custom handled")
				}
				return false, nil
			},
			func(c *mizu.Ctx, err error) (bool, error) {
				return true, c.Text(http.StatusInternalServerError, "generic")
			},
		))

		app.Get("/", func(c *mizu.Ctx) error {
			return errors.New("custom error")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("generic error", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Use(Chain(
			func(c *mizu.Ctx, err error) (bool, error) {
				if err.Error() == "custom error" {
					return true, c.Text(http.StatusBadRequest, "custom handled")
				}
				return false, nil
			},
			func(c *mizu.Ctx, err error) (bool, error) {
				return true, c.Text(http.StatusInternalServerError, "generic")
			},
		))

		app.Get("/", func(c *mizu.Ctx) error {
			return errors.New("other error")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestPanicWithError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		CatchPanic: true,
		Handler: func(c *mizu.Ctx, err error) error {
			return c.Text(http.StatusInternalServerError, "error: "+err.Error())
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		panic(errors.New("panic error"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "error: panic error" {
		t.Errorf("expected panic error message, got %q", rec.Body.String())
	}
}
