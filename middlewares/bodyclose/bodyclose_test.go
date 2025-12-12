package bodyclose

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

type trackingBody struct {
	io.Reader
	closed  bool
	drained bool
}

func (t *trackingBody) Read(p []byte) (int, error) {
	n, err := t.Reader.Read(p)
	if err == io.EOF {
		t.drained = true
	}
	return n, err
}

func (t *trackingBody) Close() error {
	t.closed = true
	return nil
}

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := bytes.NewReader([]byte("test body"))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_DrainBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{DrainBody: true}))

	app.Post("/", func(c *mizu.Ctx) error {
		// Don't read the body at all
		return c.Text(http.StatusOK, "ok")
	})

	body := &trackingBody{Reader: bytes.NewReader([]byte("test body content"))}
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !body.closed {
		t.Error("expected body to be closed")
	}
}

func TestWithOptions_NoDrain(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{DrainBody: false}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := &trackingBody{Reader: bytes.NewReader([]byte("test body content"))}
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !body.closed {
		t.Error("expected body to be closed")
	}
}

func TestNilBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d with nil body, got %d", http.StatusOK, rec.Code)
	}
}

func TestDrain(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Drain())

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := &trackingBody{Reader: bytes.NewReader([]byte("test"))}
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNoDrain(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(NoDrain())

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := &trackingBody{Reader: bytes.NewReader([]byte("test"))}
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_MaxDrain(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		DrainBody: true,
		MaxDrain:  10, // Only drain 10 bytes
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Large body
	largeBody := make([]byte, 1000)
	body := &trackingBody{Reader: bytes.NewReader(largeBody)}
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
