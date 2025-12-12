package responsesize

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello world")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_Callback(t *testing.T) {
	var size int64

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnSize: func(c *mizu.Ctx, s int64) {
			size = s
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "12345") // 5 bytes
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if size != 5 {
		t.Errorf("expected 5 bytes, got %d", size)
	}
}

func TestBytesWritten(t *testing.T) {
	var bytesWritten int64

	app := mizu.NewRouter()
	app.Use(WithCallback(func(c *mizu.Ctx, size int64) {
		bytesWritten = size
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "test response body")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if bytesWritten != int64(len("test response body")) {
		t.Errorf("expected %d, got %d", len("test response body"), bytesWritten)
	}
}

func TestGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "data")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info == nil {
		t.Fatal("expected info to be set")
	}
}

func TestWithCallback(t *testing.T) {
	var called bool
	var reportedSize int64

	app := mizu.NewRouter()
	app.Use(WithCallback(func(c *mizu.Ctx, size int64) {
		called = true
		reportedSize = size
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "abc")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !called {
		t.Error("expected callback to be called")
	}
	if reportedSize != 3 {
		t.Errorf("expected 3, got %d", reportedSize)
	}
}

func TestEmptyResponse(t *testing.T) {
	var size int64

	app := mizu.NewRouter()
	app.Use(WithCallback(func(c *mizu.Ctx, s int64) {
		size = s
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		c.Writer().WriteHeader(http.StatusNoContent)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if size != 0 {
		t.Errorf("expected 0 bytes for empty response, got %d", size)
	}
}

func TestMultipleWrites(t *testing.T) {
	var size int64

	app := mizu.NewRouter()
	app.Use(WithCallback(func(c *mizu.Ctx, s int64) {
		size = s
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		c.Writer().Write([]byte("first"))
		c.Writer().Write([]byte("second"))
		c.Writer().Write([]byte("third"))
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expected := int64(len("firstsecondthird"))
	if size != expected {
		t.Errorf("expected %d, got %d", expected, size)
	}
}
