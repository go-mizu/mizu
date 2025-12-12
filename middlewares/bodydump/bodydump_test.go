package bodydump

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	var dumpedReq, dumpedResp []byte

	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx, reqBody, respBody []byte) {
		dumpedReq = reqBody
		dumpedResp = respBody
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "response body")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request body"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if string(dumpedReq) != "request body" {
		t.Errorf("expected request body, got %q", dumpedReq)
	}
	if string(dumpedResp) != "response body" {
		t.Errorf("expected response body, got %q", dumpedResp)
	}
}

func TestWithOptions_RequestOnly(t *testing.T) {
	var dumpedReq []byte

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Request:  true,
		Response: false,
		Handler: func(c *mizu.Ctx, reqBody, respBody []byte) {
			dumpedReq = reqBody
			if len(respBody) > 0 {
				t.Error("expected empty response body")
			}
		},
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "response")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if string(dumpedReq) != "request" {
		t.Errorf("expected request, got %q", dumpedReq)
	}
}

func TestWithOptions_ResponseOnly(t *testing.T) {
	var dumpedResp []byte

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Request:  false,
		Response: true,
		Handler: func(c *mizu.Ctx, reqBody, respBody []byte) {
			if len(reqBody) > 0 {
				t.Error("expected empty request body")
			}
			dumpedResp = respBody
		},
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "response")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if string(dumpedResp) != "response" {
		t.Errorf("expected response, got %q", dumpedResp)
	}
}

func TestWithOptions_MaxSize(t *testing.T) {
	var dumpedReq []byte

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Request: true,
		MaxSize: 5,
		Handler: func(c *mizu.Ctx, reqBody, _ []byte) {
			dumpedReq = reqBody
		},
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("this is a long body"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(dumpedReq) > 5 {
		t.Errorf("expected max 5 bytes, got %d", len(dumpedReq))
	}
}

func TestWithOptions_SkipPaths(t *testing.T) {
	var called bool

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SkipPaths: []string{"/skip"},
		Handler: func(c *mizu.Ctx, _, _ []byte) {
			called = true
		},
	}))

	app.Post("/skip", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/skip", strings.NewReader("body"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if called {
		t.Error("expected handler to not be called for skipped path")
	}
}

func TestRequestOnly(t *testing.T) {
	var dumpedBody []byte

	app := mizu.NewRouter()
	app.Use(RequestOnly(func(c *mizu.Ctx, body []byte) {
		dumpedBody = body
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "response")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request body"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if string(dumpedBody) != "request body" {
		t.Errorf("expected request body, got %q", dumpedBody)
	}
}

func TestResponseOnly(t *testing.T) {
	var dumpedBody []byte

	app := mizu.NewRouter()
	app.Use(ResponseOnly(func(c *mizu.Ctx, body []byte) {
		dumpedBody = body
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "response body")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if string(dumpedBody) != "response body" {
		t.Errorf("expected response body, got %q", dumpedBody)
	}
}

func TestBodyPreserved(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx, _, _ []byte) {}))

	var bodyRead string
	app.Post("/", func(c *mizu.Ctx) error {
		body := make([]byte, 100)
		n, _ := c.Request().Body.Read(body)
		bodyRead = string(body[:n])
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("preserved"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if bodyRead != "preserved" {
		t.Errorf("expected body to be preserved, got %q", bodyRead)
	}
}
