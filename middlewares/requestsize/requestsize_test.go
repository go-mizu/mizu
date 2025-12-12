package requestsize

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

	var info *Info
	app.Post("/", func(c *mizu.Ctx) error {
		info = Get(c)
		// Read the body
		body := make([]byte, 100)
		c.Request().Body.Read(body)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test body"))
	req.ContentLength = 9
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.ContentLength != 9 {
		t.Errorf("expected ContentLength 9, got %d", info.ContentLength)
	}
}

func TestWithOptions_Callback(t *testing.T) {
	var callbackInfo *Info

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnSize: func(c *mizu.Ctx, info *Info) {
			callbackInfo = info
		},
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		body := make([]byte, 100)
		c.Request().Body.Read(body)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if callbackInfo == nil {
		t.Fatal("expected callback to be called")
	}
	if callbackInfo.BytesRead != 5 {
		t.Errorf("expected 5 bytes read, got %d", callbackInfo.BytesRead)
	}
}

func TestContentLength(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var length int64
	app.Post("/", func(c *mizu.Ctx) error {
		length = ContentLength(c)
		return c.Text(http.StatusOK, "ok")
	})

	body := "test content"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if length != int64(len(body)) {
		t.Errorf("expected %d, got %d", len(body), length)
	}
}

func TestBytesRead(t *testing.T) {
	var bytesReadAfter int64

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnSize: func(c *mizu.Ctx, info *Info) {
			bytesReadAfter = info.BytesRead
		},
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		// Read only part of the body
		buf := make([]byte, 3)
		c.Request().Body.Read(buf)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello world"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if bytesReadAfter != 3 {
		t.Errorf("expected 3 bytes read, got %d", bytesReadAfter)
	}
}

func TestWithCallback(t *testing.T) {
	var called bool

	app := mizu.NewRouter()
	app.Use(WithCallback(func(c *mizu.Ctx, info *Info) {
		called = true
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("data"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !called {
		t.Error("expected callback to be called")
	}
}

func TestNoBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.ContentLength != 0 {
		t.Errorf("expected 0 content length, got %d", info.ContentLength)
	}
}
