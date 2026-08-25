package web

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHRunsTheHandler(t *testing.T) {
	ran := false
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		ran = true
		return c.Text("hello")
	})
	if !ran {
		t.Fatal("the handler did not run")
	}
	if w.Body.String() != "hello" {
		t.Errorf("the body is %q, want hello", w.Body.String())
	}
}

func TestHFWrapsANetHTTPHandler(t *testing.T) {
	h := HF(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/plain" {
			t.Errorf("the path is %q, want /plain", r.URL.Path)
		}
		w.WriteHeader(http.StatusTeapot)
	})

	w := serve(t, httptest.NewRequest("GET", "/plain", nil), h)
	if w.Code != http.StatusTeapot {
		t.Errorf("the status is %d, want 418", w.Code)
	}
}

func TestHFReportsWhatTheWrappedHandlerWrote(t *testing.T) {
	serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		if err := HF(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		})(c); err != nil {
			return err
		}
		if c.StatusCode() != http.StatusTeapot {
			t.Errorf("the status is %d, want 418, so the wrapper is not seeing the write", c.StatusCode())
		}
		return nil
	})
}

func TestFromContextFindsTheCtx(t *testing.T) {
	serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		got, ok := FromContext(c.Context())
		if !ok {
			t.Fatal("FromContext found nothing inside a handler")
		}
		if got != c {
			t.Error("FromContext found a different Ctx")
		}
		return nil
	})
}

func TestFromContextFindsNothingOutsideAHandler(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if _, ok := FromContext(r.Context()); ok {
		t.Error("FromContext found a Ctx on a request that never went through H")
	}
}

func TestAFailedHandlerGetsTheDefault(t *testing.T) {
	var buf strings.Builder
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return errors.New("the database is on fire")
	})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("the status is %d, want 500", w.Code)
	}
	if strings.Contains(w.Body.String(), "on fire") {
		t.Errorf("the error text went to the client:\n%s", w.Body.String())
	}
	if !strings.Contains(buf.String(), "on fire") {
		t.Errorf("the error text did not go to the log:\n%s", buf.String())
	}
}

func TestAFailedHandlerThatAlreadyWroteGetsOnlyTheLog(t *testing.T) {
	var buf strings.Builder
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		if err := c.Status(http.StatusOK).Text("half an answer"); err != nil {
			return err
		}
		return errors.New("and then it went wrong")
	})

	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want the 200 that had already gone out", w.Code)
	}
	if w.Body.String() != "half an answer" {
		t.Errorf("the body is %q, so a second answer was written", w.Body.String())
	}
	if !strings.Contains(buf.String(), "and then it went wrong") {
		t.Errorf("the error did not go to the log:\n%s", buf.String())
	}
}

func TestErrorsInstallsTheRenderer(t *testing.T) {
	var got error
	h := Errors(func(c *Ctx, err error) {
		got = err
		c.Status(http.StatusBadGateway).Text("rendered")
	})(H(func(c *Ctx) error {
		return errors.New("upstream said no")
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if got == nil || got.Error() != "upstream said no" {
		t.Errorf("the renderer was handed %v", got)
	}
	if w.Code != http.StatusBadGateway {
		t.Errorf("the status is %d, want 502", w.Code)
	}
	if w.Body.String() != "rendered" {
		t.Errorf("the body is %q, want rendered", w.Body.String())
	}
}

func TestErrorsIsNotConsultedWhenTheHandlerWorked(t *testing.T) {
	called := false
	h := Errors(func(c *Ctx, err error) { called = true })(H(func(c *Ctx) error {
		return c.Text("fine")
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if called {
		t.Error("the error renderer ran for a handler that worked")
	}
	if w.Body.String() != "fine" {
		t.Errorf("the body is %q, want fine", w.Body.String())
	}
}

func TestTheCtxGoesBackWhenTheHandlerPanics(t *testing.T) {
	var c *Ctx
	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic did not come back out of H")
			}
		}()
		H(func(inner *Ctx) error {
			c = inner
			panic("boom")
		}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}()

	if c == nil {
		t.Fatal("the handler did not run")
	}
	if c.r != nil {
		t.Error("the Ctx was not released after the panic")
	}
}
