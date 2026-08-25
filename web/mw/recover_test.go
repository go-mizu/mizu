package mw

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/web"
)

// panicked serves one request through Recover in front of h and reports the
// response and the one record that came out.
func panicked(tb testing.TB, h http.Handler) (*httptest.ResponseRecorder, slog.Record) {
	tb.Helper()

	c := new(collector)
	w := httptest.NewRecorder()
	Recover(slog.New(c))(h).ServeHTTP(w, httptest.NewRequest("POST", "/orders", nil))
	return w, c.only(tb)
}

func TestAPanicBecomesA500(t *testing.T) {
	w, rec := panicked(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(errors.New("the database is on fire"))
	}))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("the response is a %d, want 500", w.Code)
	}
	if rec.Level != slog.LevelError {
		t.Errorf("the record is at %s, want ERROR", rec.Level)
	}
	if rec.Message != "handler panicked" {
		t.Errorf("the record says %q", rec.Message)
	}

	got := attrs(rec)
	if !strings.Contains(got["panic"], "the database is on fire") {
		t.Errorf("panic is %q, and it does not name what was panicked with", got["panic"])
	}
	if got["method"] != "POST" || got["path"] != "/orders" {
		t.Errorf("the record says %s %s, want POST /orders", got["method"], got["path"])
	}
}

// TestTheStackNamesTheLineThatPanicked is the whole reason the stack is
// captured in the deferred function rather than after it.
func TestTheStackNamesTheLineThatPanicked(t *testing.T) {
	_, rec := panicked(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("nope")
	}))

	stack := attrs(rec)["stack"]
	if !strings.Contains(stack, "recover_test.go") {
		t.Errorf("the stack does not reach the handler that panicked:\n%s", stack)
	}
}

// TestAPanicAfterTheResponseStartedIsLoggedAndNothingElse covers the case where
// there is no status left to send.
func TestAPanicAfterTheResponseStartedIsLoggedAndNothingElse(t *testing.T) {
	w, rec := panicked(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("half an answer"))
		panic("halfway through")
	}))

	if w.Code != http.StatusAccepted {
		t.Errorf("the response is a %d, want the 202 that had already gone out", w.Code)
	}
	if w.Body.String() != "half an answer" {
		t.Errorf("the body is %q, want the half that had already gone out", w.Body)
	}
	if rec.Level != slog.LevelError {
		t.Errorf("the record is at %s, want ERROR", rec.Level)
	}
}

// TestAbortIsPassedAlong keeps net/http's own signal working. httputil's
// reverse proxy panics with this when the client hangs up, and turning that into
// an ERROR and a 500 would mean one of each per closed tab.
func TestAbortIsPassedAlong(t *testing.T) {
	c := new(collector)
	h := Recover(slog.New(c))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Errorf("the panic that came out is %v, want http.ErrAbortHandler", v)
		}
		if len(c.got) != 0 {
			t.Errorf("aborting logged %d records, want none", len(c.got))
		}
	}()

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	t.Error("the handler returned, and it was meant to panic")
}

func TestAHandlerThatDoesNotPanicIsLeftAlone(t *testing.T) {
	c := new(collector)
	w := httptest.NewRecorder()

	Recover(slog.New(c))(web.H(func(c *web.Ctx) error {
		return c.Text("fine")
	})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Body.String() != "fine" {
		t.Errorf("the body is %q, want fine", w.Body)
	}
	if len(c.got) != 0 {
		t.Errorf("a request that worked logged %d records, want none", len(c.got))
	}
}

// TestThePanicIsLoggedWithTheRequestId is the ordering that makes the 500 and
// the stack findable by the same query.
func TestThePanicIsLoggedWithTheRequestId(t *testing.T) {
	c := new(collector)
	h := web.Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("nope")
	}), RequestID(), Recover(slog.New(c)))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	id := attrs(c.only(t))["request_id"]
	if id == "" {
		t.Fatal("the record has no request_id on it")
	}
	if got := w.Header().Get(RequestIDHeader); got != id {
		t.Errorf("the response says %q and the record says %q", got, id)
	}
}
