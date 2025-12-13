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
			if errors.Is(err, customErr) {
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

func TestPanicWithNonError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		CatchPanic: true,
		Handler: func(c *mizu.Ctx, err error) error {
			return c.Text(http.StatusInternalServerError, "caught: "+err.Error())
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		panic("string panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "caught: panic occurred" {
		t.Errorf("expected 'caught: panic occurred', got %q", rec.Body.String())
	}
}

func TestPanicWithDefaultHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		CatchPanic:     true,
		DefaultMessage: "Server error",
		// No Handler set - uses default
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		panic("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "Server error" {
		t.Errorf("expected 'Server error', got %q", rec.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(NotFound(func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "custom not found")
	}))

	app.Get("/exists", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Test 404 case
	req := httptest.NewRequest(http.MethodGet, "/notexists", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Note: The actual behavior depends on how router handles 404
}

func TestForStatus(t *testing.T) {
	middleware := ForStatus(http.StatusBadRequest, func(c *mizu.Ctx) error {
		return c.Text(http.StatusBadRequest, "bad request handled")
	})

	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestChain_NoHandlerMatches(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Chain(
		func(c *mizu.Ctx, err error) (bool, error) {
			// Don't handle anything
			return false, nil
		},
		func(c *mizu.Ctx, err error) (bool, error) {
			// Also don't handle
			return false, nil
		},
	))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("unhandled error")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if rec.Body.String() != "An error occurred" {
		t.Errorf("expected default error message, got %q", rec.Body.String())
	}
}

func TestResponseCapture_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	capture := &responseCapture{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// First WriteHeader should set status
	capture.WriteHeader(http.StatusNotFound)
	if capture.statusCode != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, capture.statusCode)
	}
	if !capture.written {
		t.Error("expected written to be true")
	}

	// Second WriteHeader should not change status
	capture.WriteHeader(http.StatusOK)
	if capture.statusCode != http.StatusNotFound {
		t.Errorf("status should not change, got %d", capture.statusCode)
	}
}

func TestResponseCapture_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	capture := &responseCapture{
		ResponseWriter: rec,
		statusCode:     0,
	}

	// Write should set status to 200 if not already set
	n, err := capture.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes, got %d", n)
	}
	if capture.statusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, capture.statusCode)
	}
	if !capture.written {
		t.Error("expected written to be true")
	}
}

func TestWithOptions_DefaultsUsed(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("test")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Default message should be "An error occurred"
	if rec.Body.String() != "An error occurred" {
		t.Errorf("expected default message, got %q", rec.Body.String())
	}
}

func TestPanicError_ErrorMethod(t *testing.T) {
	// Test with error value
	pe := &panicError{value: errors.New("inner error")}
	if pe.Error() != "inner error" {
		t.Errorf("expected 'inner error', got %q", pe.Error())
	}

	// Test with non-error value
	pe2 := &panicError{value: "string value"}
	if pe2.Error() != "panic occurred" {
		t.Errorf("expected 'panic occurred', got %q", pe2.Error())
	}

	// Test with int value
	pe3 := &panicError{value: 123}
	if pe3.Error() != "panic occurred" {
		t.Errorf("expected 'panic occurred', got %q", pe3.Error())
	}
}
