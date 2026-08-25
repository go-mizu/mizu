package web

import (
	"bytes"
	"errors"
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

func TestAFilterSeesTheBodyAndTheServerSeesWhatItProduced(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	restore := rec.Through(upper{rec.Sink()})
	io.WriteString(rec, "quiet")
	restore()
	io.WriteString(rec, " again")

	if w.Body.String() != "QUIET again" {
		t.Errorf("the response is %q, want QUIET again", w.Body)
	}
	if rec.Written() != 11 {
		t.Errorf("the Recorder counted %d bytes, want 11", rec.Written())
	}
}

// upper is the smallest filter there is: it writes what it was given in capitals
// to whatever is under it.
type upper struct{ w io.Writer }

func (u upper) Write(p []byte) (int, error) {
	if _, err := u.w.Write([]byte(strings.ToUpper(string(p)))); err != nil {
		return 0, err
	}
	// The count that goes back is the handler's, since io.Writer promises the
	// caller that a write with no error took everything.
	return len(p), nil
}

// TestFiltersNest is the arrangement Compress and ETag are in: the outer
// middleware installs first and ends up furthest from the handler, so the inner
// one writes through it.
func TestFiltersNest(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	defer rec.Through(surround{rec.Sink(), "<", ">"})()
	defer rec.Through(surround{rec.Sink(), "[", "]"})()

	io.WriteString(rec, "body")

	if w.Body.String() != "<[body]>" {
		t.Errorf("the response is %q, want <[body]>", w.Body)
	}
}

// surround wraps each write in a pair, which makes the order of a stack of
// filters visible in the output.
type surround struct {
	w           io.Writer
	open, close string
}

func (s surround) Write(p []byte) (int, error) {
	if _, err := io.WriteString(s.w, s.open+string(p)+s.close); err != nil {
		return 0, err
	}
	return len(p), nil
}

func TestRestoringPutsBackWhatWasThereBefore(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	defer rec.Through(surround{rec.Sink(), "<", ">"})()

	restore := rec.Through(surround{rec.Sink(), "[", "]"})
	io.WriteString(rec, "one")
	restore()
	io.WriteString(rec, "two")

	if w.Body.String() != "<[one]><two>" {
		t.Errorf("the response is %q, want <[one]><two>", w.Body)
	}
}

func TestReadFromGoesThroughTheFilter(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	defer rec.Through(upper{rec.Sink()})()
	if _, err := rec.ReadFrom(strings.NewReader("copied")); err != nil {
		t.Fatal(err)
	}

	if w.Body.String() != "COPIED" {
		t.Errorf("the response is %q, want COPIED", w.Body)
	}
}

// TestHoldKeepsTheStatusBackUntilSomebodyHasSeenTheBody is what an ETag needs:
// the handler answers, the middleware reads what it wrote, and there is still a
// different status to send.
func TestHoldKeepsTheStatusBackUntilSomebodyHasSeenTheBody(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	var held bytes.Buffer
	send := rec.Hold()
	restore := rec.Through(&held)

	rec.WriteHeader(http.StatusOK)
	io.WriteString(rec, "the whole page")
	restore()

	if w.Code != 200 || w.Body.Len() != 0 {
		t.Fatalf("the server saw %d and %d bytes before the hold was sent", w.Code, w.Body.Len())
	}

	rec.WriteHeader(http.StatusNotModified)
	send()

	if w.Code != http.StatusNotModified {
		t.Errorf("the response went out as %d, want 304", w.Code)
	}
	if rec.Status() != http.StatusNotModified {
		t.Errorf("the Recorder reports %d, want 304", rec.Status())
	}
	if held.String() != "the whole page" {
		t.Errorf("the filter held %q, want the whole page", held.String())
	}
}

// TestABodyByteEndsTheHold covers the middleware that holds the status and then
// decides to let the response go as it was. A status has to be in front of a
// body on the wire, so the first byte through sends it.
func TestABodyByteEndsTheHold(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	send := rec.Hold()
	rec.WriteHeader(http.StatusCreated)
	io.WriteString(rec, "made it")
	send()

	if w.Code != http.StatusCreated {
		t.Errorf("the response went out as %d, want 201", w.Code)
	}
	if w.Body.String() != "made it" {
		t.Errorf("the response is %q, want made it", w.Body)
	}
}

func TestAHoldWithNoStatusInItIsA200(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	rec.Hold()()

	if w.Code != http.StatusOK || rec.Status() != http.StatusOK {
		t.Errorf("the response went out as %d and the Recorder reports %d, want 200 and 200", w.Code, rec.Status())
	}
}

func TestSendingATwiceIsHarmless(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	send := rec.Hold()
	rec.WriteHeader(http.StatusTeapot)
	send()
	send()

	if w.Code != http.StatusTeapot {
		t.Errorf("the response went out as %d, want 418", w.Code)
	}
}

func TestHoldingATwiceIsAChainThatNeedsFixing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("two middleware both held the status and neither was told")
		}
	}()

	rec := Record(httptest.NewRecorder())
	rec.Hold()
	rec.Hold()
}

// TestHoldingAResponseThatHasStartedDoesNothing is the case where the middleware
// is registered somewhere that cannot work. The status has gone and holding it
// is not going to bring it back, so the response is served as it stands.
func TestHoldingAResponseThatHasStartedDoesNothing(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	rec.WriteHeader(http.StatusCreated)
	send := rec.Hold()
	rec.WriteHeader(http.StatusNotModified)
	send()

	if w.Code != http.StatusCreated {
		t.Errorf("the response went out as %d, want the 201 that had already gone", w.Code)
	}
}

// TestAFlushReachesTheFiltersAndTheHold is the one that stops a compressor's
// buffer sitting in memory while the flush reports success.
func TestAFlushReachesTheFiltersAndTheHold(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)

	f := &flushable{w: rec.Sink()}
	defer rec.Through(f)()
	send := rec.Hold()
	defer send()

	io.WriteString(rec, "buffered")
	if w.Body.Len() != 0 {
		t.Fatalf("the filter passed %q straight through", w.Body)
	}

	if err := http.NewResponseController(rec).Flush(); err != nil {
		t.Fatal(err)
	}

	if w.Body.String() != "buffered" {
		t.Errorf("the flush left %q on the server, want buffered", w.Body)
	}
	if w.Code != http.StatusOK {
		t.Errorf("the flush went out as %d, want the held header at 200", w.Code)
	}
}

// flushable holds what it is given until it is flushed, which is what every
// compressor does.
type flushable struct {
	w   io.Writer
	buf bytes.Buffer
}

func (f *flushable) Write(p []byte) (int, error) { return f.buf.Write(p) }

func (f *flushable) Flush() error {
	_, err := f.buf.WriteTo(f.w)
	return err
}

// TestAFilterWithNoFlushIsSteppedOver. A filter that does not hold anything back
// has nothing to flush, and the flush carries on past it to the server.
func TestAFilterWithNoFlushIsSteppedOver(t *testing.T) {
	w := httptest.NewRecorder()
	rec := Record(w)
	defer rec.Through(upper{rec.Sink()})()

	io.WriteString(rec, "mizu")

	if err := http.NewResponseController(rec).Flush(); err != nil {
		t.Fatal(err)
	}
	if w.Body.String() != "MIZU" {
		t.Errorf("the server has %q, want MIZU", w.Body)
	}
	if !w.Flushed {
		t.Error("the flush stopped at the filter and never reached the server")
	}
}

// TestARecorderIsNotAFlusherOnItsOwn keeps a handler from being told it can
// stream through a writer that cannot. The controller finds FlushError and gets
// the real answer; a type assertion for http.Flusher finds nothing.
func TestARecorderIsNotAFlusherOnItsOwn(t *testing.T) {
	if _, ok := any(Record(httptest.NewRecorder())).(http.Flusher); ok {
		t.Error("a Recorder claims to be an http.Flusher, so a handler cannot tell whether the server is one")
	}

	rec := Record(unflushable{httptest.NewRecorder()})
	if err := rec.FlushError(); !errors.Is(err, http.ErrNotSupported) {
		t.Errorf("flushing a writer that cannot flush came back with %v, want ErrNotSupported", err)
	}
}

// unflushable is a writer with no Flush of any kind, which is what a hijacked
// connection or a test double usually is.
type unflushable struct{ http.ResponseWriter }

func (u unflushable) Unwrap() http.ResponseWriter { return nil }

func TestAFilterThatFailsToFlushStopsTheFlush(t *testing.T) {
	rec := Record(httptest.NewRecorder())
	defer rec.Through(brokenFlusher{})()

	if err := rec.FlushError(); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("the flush came back with %v, want the filter's error", err)
	}
}

// brokenFlusher is a filter whose flush fails, which is what a compressor
// writing to a connection that went away looks like.
type brokenFlusher struct{}

func (brokenFlusher) Write(p []byte) (int, error) { return len(p), nil }
func (brokenFlusher) Flush() error                { return io.ErrClosedPipe }
