package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// postView is the shape the doc comment describes: a value a handler returns,
// with the writing on it rather than in the handler.
type postView struct{ title string }

func (v postView) Respond(c *Ctx) error {
	return c.Status(http.StatusCreated).Text("post " + v.title)
}

// missingView is a pointer receiver whose nil is a value it knows what to do
// with, which is the case the interface nil check must not swallow.
type missingView struct{ title string }

func (v *missingView) Respond(c *Ctx) error {
	if v == nil {
		return c.Status(http.StatusNotFound).Text("no post")
	}
	return c.Text("post " + v.title)
}

// brokenView is one that fails while writing.
type brokenView struct{}

func (brokenView) Respond(c *Ctx) error { return errors.New("the view gave up") }

// caught runs a handler and returns what it wrote and what the error handler
// was given, since a handler's error is not the ServeHTTP call's to return.
func caught(t *testing.T, h Handler) (*httptest.ResponseRecorder, error) {
	t.Helper()

	var got error
	w := httptest.NewRecorder()
	Errors(func(c *Ctx, err error) { got = err })(H(h)).
		ServeHTTP(w, httptest.NewRequest("GET", "/posts", nil))
	return w, got
}

func TestRSendsWhatTheHandlerReturned(t *testing.T) {
	show := func(c *Ctx) (postView, error) { return postView{"water"}, nil }

	w := serve(t, httptest.NewRequest("GET", "/posts", nil), R(show))

	if w.Code != http.StatusCreated {
		t.Errorf("the status is %d, want 201", w.Code)
	}
	if got, want := w.Body.String(), "post water"; got != want {
		t.Errorf("the body is %q, want %q", got, want)
	}
}

func TestRPassesTheHandlersErrorOn(t *testing.T) {
	fail := errors.New("no such post")
	show := func(c *Ctx) (postView, error) { return postView{}, fail }

	w, got := caught(t, R(show))

	if !errors.Is(got, fail) {
		t.Errorf("the error handler was given %v, want %v", got, fail)
	}
	if w.Body.Len() != 0 {
		t.Errorf("a handler that failed wrote %q", w.Body)
	}
}

// TestRPassesRespondsErrorOn is the half that a handler returning an error
// cannot cover. A view that fails while writing is as much a failed request as
// one that never built a view.
func TestRPassesRespondsErrorOn(t *testing.T) {
	show := func(c *Ctx) (brokenView, error) { return brokenView{}, nil }

	_, got := caught(t, R(show))

	if got == nil || got.Error() != "the view gave up" {
		t.Errorf("the error handler was given %v, want the view's error", got)
	}
}

// TestANilResponderIsAResponseAlreadySent is the one case where returning
// nothing is not a mistake.
func TestANilResponderIsAResponseAlreadySent(t *testing.T) {
	show := func(c *Ctx) (Responder, error) {
		if err := c.Text("written by hand"); err != nil {
			return nil, err
		}
		return nil, nil
	}

	w, got := caught(t, R(show))

	if got != nil {
		t.Errorf("a handler that answered for itself was reported as failing: %v", got)
	}
	if body, want := w.Body.String(), "written by hand"; body != want {
		t.Errorf("the body is %q, want %q", body, want)
	}
}

// TestANilPointerIsStillTheHandlersValue holds the other half of that apart. A
// typed nil is a value the handler chose and its method knows what it means, so
// the check for a nil interface must not take it as one.
func TestANilPointerIsStillTheHandlersValue(t *testing.T) {
	show := func(c *Ctx) (*missingView, error) { return nil, nil }

	w, got := caught(t, R(show))

	if got != nil {
		t.Errorf("a nil pointer was reported as a failure: %v", got)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("the status is %d, want 404", w.Code)
	}
	if body, want := w.Body.String(), "no post"; body != want {
		t.Errorf("the body is %q, want %q", body, want)
	}
}

// TestTheInterfaceIsStillAReturnType is the shape doc 08 sketched, where the
// handler names Responder because it has more than one answer.
func TestTheInterfaceIsStillAReturnType(t *testing.T) {
	show := func(c *Ctx) (Responder, error) {
		if c.Query("missing") != "" {
			return (*missingView)(nil), nil
		}
		return postView{"water"}, nil
	}

	cases := []struct {
		name string
		url  string
		code int
		body string
	}{
		{"the one it found", "/posts", http.StatusCreated, "post water"},
		{"the one it did not", "/posts?missing=1", http.StatusNotFound, "no post"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(t, httptest.NewRequest("GET", tc.url, nil), R(show))

			if w.Code != tc.code {
				t.Errorf("the status is %d, want %d", w.Code, tc.code)
			}
			if got := w.Body.String(); got != tc.body {
				t.Errorf("the body is %q, want %q", got, tc.body)
			}
		})
	}
}
