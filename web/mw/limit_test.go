package mw

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/web"
)

func TestAHandlerThatFinishesInTimeIsLeftAlone(t *testing.T) {
	w := httptest.NewRecorder()
	Timeout(time.Minute)(web.H(func(c *web.Ctx) error {
		return c.Text("done")
	})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != http.StatusOK || w.Body.String() != "done" {
		t.Errorf("the response is a %d saying %q, want 200 and done", w.Code, w.Body)
	}
}

func TestTheDeadlineIsOnTheContextTheHandlerGets(t *testing.T) {
	var deadline time.Time
	var ok bool

	Timeout(time.Minute)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		deadline, ok = r.Context().Deadline()
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	if !ok {
		t.Fatal("the handler's context has no deadline on it")
	}
	if until := time.Until(deadline); until <= 0 || until > time.Minute {
		t.Errorf("the deadline is %s away, want somewhere under a minute", until)
	}
}

// TestAHandlerThatRanOutOfTimeGetsA504 uses a handler that waits on its context
// the way a database driver does, which is what the deadline is for.
func TestAHandlerThatRanOutOfTimeGetsA504(t *testing.T) {
	w := httptest.NewRecorder()
	Timeout(time.Millisecond)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("the response is a %d, want 504", w.Code)
	}
}

// TestAResponseThatHadAlreadyStartedIsNotReplaced is the trade this makes
// against net/http's TimeoutHandler: nothing is buffered, so nothing can be
// taken back.
func TestAResponseThatHadAlreadyStartedIsNotReplaced(t *testing.T) {
	w := httptest.NewRecorder()
	Timeout(time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("streaming"))
		<-r.Context().Done()
	})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != http.StatusAccepted {
		t.Errorf("the response is a %d, want the 202 that had already gone out", w.Code)
	}
	if w.Body.String() != "streaming" {
		t.Errorf("the body is %q, want streaming", w.Body)
	}
}

// TestACancelledRequestIsNotATimeout separates the two things a done context
// means. A client that hung up leaves nobody to send a 504 to.
func TestACancelledRequestIsNotATimeout(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	ctx, cancel := context.WithCancel(r.Context())
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	Timeout(time.Minute)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	})).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("the response is a %d, and a cancelled request is not a timeout", w.Code)
	}
}

func TestABodyThatSaysItIsTooBigIsRefusedBeforeTheHandlerRuns(t *testing.T) {
	ran := false
	w := httptest.NewRecorder()

	r := httptest.NewRequest("POST", "/", strings.NewReader(strings.Repeat("a", 100)))
	MaxBody(64)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		ran = true
	})).ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("the response is a %d, want 413", w.Code)
	}
	if ran {
		t.Error("the handler ran for a body that was refused")
	}
}

func TestABodyUnderTheLimitIsReadInFull(t *testing.T) {
	var got []byte
	r := httptest.NewRequest("POST", "/", strings.NewReader("small"))

	MaxBody(64)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
	})).ServeHTTP(httptest.NewRecorder(), r)

	if string(got) != "small" {
		t.Errorf("the handler read %q, want small", got)
	}
}

// TestABodyThatRunsOverWhileItIsBeingReadFailsTheRead is the case the
// Content-Length check cannot catch, which is a chunked body or a lying client.
func TestABodyThatRunsOverWhileItIsBeingReadFailsTheRead(t *testing.T) {
	var err error
	r := httptest.NewRequest("POST", "/", io.NopCloser(strings.NewReader(strings.Repeat("a", 100))))
	if r.ContentLength != -1 {
		t.Fatalf("the request says its length is %d, and this case is about not knowing", r.ContentLength)
	}

	MaxBody(64)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err = io.ReadAll(r.Body)
	})).ServeHTTP(httptest.NewRecorder(), r)

	var over *http.MaxBytesError
	if !errors.As(err, &over) {
		t.Fatalf("reading the body failed with %v, want a *http.MaxBytesError", err)
	}
	if over.Limit != 64 {
		t.Errorf("the error names a limit of %d, want 64", over.Limit)
	}
}

func TestConcurrencyRefusesWhatItCannotHold(t *testing.T) {
	gate := make(chan struct{})
	entered := make(chan struct{}, 4)

	h := Concurrency(1)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		entered <- struct{}{}
		<-gate
	}))

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}()
	<-entered

	full := httptest.NewRecorder()
	h.ServeHTTP(full, httptest.NewRequest("GET", "/", nil))
	if full.Code != http.StatusServiceUnavailable {
		t.Errorf("the second request got a %d, want 503", full.Code)
	}
	if got := full.Header().Get("Retry-After"); got != "1" {
		t.Errorf("Retry-After is %q, want 1", got)
	}
	if len(entered) != 0 {
		t.Error("the handler ran for a request that was refused")
	}

	close(gate)
	<-done

	// The slot goes back when the handler returns, so the next one is served.
	after := httptest.NewRecorder()
	h.ServeHTTP(after, httptest.NewRequest("GET", "/", nil))
	if after.Code != http.StatusOK {
		t.Errorf("the request after the slot was freed got a %d, want 200", after.Code)
	}
}

// TestOneConcurrencyIsOneLimit is what makes this a limit on the process rather
// than on a chain: the count lives in the value the constructor returned.
func TestOneConcurrencyIsOneLimit(t *testing.T) {
	gate := make(chan struct{})
	entered := make(chan struct{}, 4)

	m := Concurrency(1)
	blocking := m(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		entered <- struct{}{}
		<-gate
	}))
	other := m(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	done := make(chan struct{})
	go func() {
		defer close(done)
		blocking.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}()
	<-entered

	w := httptest.NewRecorder()
	other.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("the other chain got a %d, want the 503 the shared limit owes it", w.Code)
	}

	close(gate)
	<-done
}

func TestALimitOfNoneIsATypo(t *testing.T) {
	for _, n := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Concurrency(%d) returned rather than panicking", n)
				}
			}()
			Concurrency(n)
		}()
	}
}
