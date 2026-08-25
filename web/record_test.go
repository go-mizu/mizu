package web

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecordWrapsAWriterThatIsNotOneYet(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	if rec.Unwrap() != http.ResponseWriter(w) {
		t.Error("the Recorder is not wrapping the writer it was given")
	}
	if rec.Status() != 0 || rec.Written() != 0 {
		t.Errorf("a fresh Recorder reports %d and %d bytes, want nothing", rec.Status(), rec.Written())
	}
}

func TestRecordHandsBackTheOneAlreadyInTheChain(t *testing.T) {
	first := Record(httptest.NewRecorder())
	if second := Record(first); second != first {
		t.Error("Record wrapped a Recorder in a second Recorder")
	}
}

// TestTheHandlerRecordsIntoTheChainsRecorder is the whole point of the type.
// Middleware makes one on the way in, the handler writes through the Ctx, and
// the middleware reads the answer on the way out without having to wrap
// anything again.
func TestTheHandlerRecordsIntoTheChainsRecorder(t *testing.T) {
	var status int
	var written int64

	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := Record(w)
			next.ServeHTTP(rec, r)
			status, written = rec.Status(), rec.Written()
		})
	}

	h := Chain(H(func(c *Ctx) error {
		return c.Status(http.StatusTeapot).Text("short and stout")
	}), outer)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	if status != http.StatusTeapot {
		t.Errorf("the middleware saw the status %d, want 418", status)
	}
	if want := int64(len("short and stout")); written != want {
		t.Errorf("the middleware saw %d bytes, want %d", written, want)
	}
}

func TestTheHandlerUsesItsOwnRecorderWhenTheChainHasNone(t *testing.T) {
	var inner, outer *Recorder
	w := httptest.NewRecorder()

	H(func(c *Ctx) error {
		inner = c.res
		outer = Record(c.Writer())
		return c.Text("body")
	}).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if inner != outer {
		t.Error("Record on the Ctx's writer made a second Recorder")
	}
	if w.Body.String() != "body" {
		t.Errorf("the response is %q, want body", w.Body)
	}
}

func TestTheStatusIsTheFirstOneSent(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	rec.WriteHeader(http.StatusCreated)
	rec.WriteHeader(http.StatusTeapot) // net/http ignores this and so does the Recorder

	if rec.Status() != http.StatusCreated {
		t.Errorf("the Recorder reports %d, want 201", rec.Status())
	}
}

func TestAWriteWithNoStatusInFrontOfItIsA200(t *testing.T) {
	rec := Record(httptest.NewRecorder())
	if _, err := rec.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	if rec.Status() != http.StatusOK {
		t.Errorf("the Recorder reports %d, want 200", rec.Status())
	}
	if rec.Written() != 5 {
		t.Errorf("the Recorder counted %d bytes, want 5", rec.Written())
	}
}

func TestWrittenAddsUpAcrossWrites(t *testing.T) {
	rec := Record(httptest.NewRecorder())
	for range 3 {
		if _, err := io.WriteString(rec, "abcd"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := rec.ReadFrom(strings.NewReader("ef")); err != nil {
		t.Fatal(err)
	}

	if rec.Written() != 14 {
		t.Errorf("the Recorder counted %d bytes, want 14", rec.Written())
	}
}

func TestWrittenCountsWhatLandedRatherThanWhatWasOffered(t *testing.T) {
	rec := Record(shortWriter{httptest.NewRecorder()})
	n, err := rec.Write([]byte("ten bytes!"))
	if err == nil {
		t.Fatal("a writer that failed part of the way through came back with no error")
	}
	if n != 4 || rec.Written() != 4 {
		t.Errorf("the write landed %d bytes and the Recorder counted %d, want 4 and 4", n, rec.Written())
	}
}

// shortWriter is a ResponseWriter whose connection went away in the middle of
// a write, which is what a client that hung up looks like from here.
type shortWriter struct{ http.ResponseWriter }

func (shortWriter) Write(p []byte) (int, error) { return 4, io.ErrClosedPipe }

func TestReadFromCountsAndStartsTheResponse(t *testing.T) {
	rec := Record(httptest.NewRecorder())
	n, err := rec.ReadFrom(strings.NewReader("five!"))
	if err != nil {
		t.Fatal(err)
	}

	if n != 5 || rec.Written() != 5 {
		t.Errorf("ReadFrom copied %d bytes and the Recorder counted %d, want 5 and 5", n, rec.Written())
	}
	if rec.Status() != http.StatusOK {
		t.Errorf("ReadFrom recorded the status %d, want 200", rec.Status())
	}
}

// TestAForeignWrapperIsMentionedOnce covers the case the Recorder cannot do
// anything about: something that is not mizu has wrapped the chain's Recorder,
// so the next one along has to make a second.
func TestAForeignWrapperIsMentionedOnce(t *testing.T) {
	// The set of types already mentioned is package level and outlives a test,
	// which is what once means, so this one starts by forgetting.
	foreign.Clear()

	var out bytes.Buffer
	quiet := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&out, nil)))
	t.Cleanup(func() { slog.SetDefault(quiet) })

	inner := Record(httptest.NewRecorder())
	for range 3 {
		if rec := Record(&stranger{inner}); rec == inner {
			t.Fatal("Record reached through a writer it does not know about")
		}
	}

	if got := strings.Count(out.String(), "recorded twice"); got != 1 {
		t.Errorf("the foreign writer was mentioned %d times, want once:\n%s", got, out.String())
	}
	if !strings.Contains(out.String(), "web.stranger") {
		t.Errorf("the warning does not name the writer:\n%s", out.String())
	}
}

// stranger is middleware from anywhere else, which wraps the writer in a type
// of its own and forwards Unwrap because http.ResponseController needs it.
type stranger struct{ http.ResponseWriter }

func (s *stranger) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func TestAPlainWriterIsNotWarnedAbout(t *testing.T) {
	foreign.Clear()

	var out bytes.Buffer
	quiet := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&out, nil)))
	t.Cleanup(func() { slog.SetDefault(quiet) })

	Record(&stranger{httptest.NewRecorder()})
	Record(httptest.NewRecorder())

	if out.Len() != 0 {
		t.Errorf("a chain with no Recorder in it produced a warning:\n%s", out.String())
	}
}

// TestTheWalkForAForeignWrapperGivesUp keeps buried from hanging on a chain
// that unwraps to itself, which is a mistake somebody will make once.
func TestTheWalkForAForeignWrapperGivesUp(t *testing.T) {
	var loop cycle
	loop.ResponseWriter = &loop
	if buried(&loop) {
		t.Error("a writer that unwraps to itself reported a Recorder under it")
	}

	var nothing nilUnwrapper
	if buried(&nothing) {
		t.Error("a writer that unwraps to nil reported a Recorder under it")
	}
}

// cycle is the mistake: a wrapper whose Unwrap returns the wrapper.
type cycle struct{ http.ResponseWriter }

func (c *cycle) Unwrap() http.ResponseWriter { return c.ResponseWriter }

// nilUnwrapper is the other one: a wrapper that was never filled in.
type nilUnwrapper struct{ http.ResponseWriter }

func (n *nilUnwrapper) Unwrap() http.ResponseWriter { return nil }
