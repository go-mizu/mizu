package mizutest

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/clock"
)

func TestNewAppServesTheRoutesRegisteredOnIt(t *testing.T) {
	app := NewApp(t)
	app.Routes().Get("/ping", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "pong")
	})

	app.Get("/ping").Do().AssertOK().AssertSee("pong")
}

// TestTheClockIsTheOneTheTestSet is the whole point of the fixture holding a
// clock: a handler reading the time gets the time the test chose, without the
// test reaching into the handler to put it there.
func TestTheClockIsTheOneTheTestSet(t *testing.T) {
	at := time.Date(2026, 2, 28, 23, 59, 0, 0, time.UTC)
	app := NewApp(t, At(at))
	app.Routes().Get("/now", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, clock.Now(c.Context()).Format(time.RFC3339))
	})

	app.Get("/now").Do().AssertSee("2026-02-28T23:59:00Z")

	app.Clock().Advance(time.Minute)
	app.Get("/now").Do().AssertSee("2026-03-01T00:00:00Z")
}

func TestTheClockStartsSomewhereDefinite(t *testing.T) {
	app := NewApp(t)

	if got := app.Clock().Now(); !got.Equal(defaultAt) {
		t.Errorf("the clock starts at %v, want %v", got, defaultAt)
	}
	if got := clock.Now(app.Ctx()); !got.Equal(defaultAt) {
		t.Errorf("the context clock is at %v, want %v", got, defaultAt)
	}
}

// TestCtxIsCancelledWhenTheTestEnds keeps a goroutine started by a test from
// outliving it, which is the kind of leak that shows up much later as a test
// that fails only when run with others.
func TestCtxIsCancelledWhenTheTestEnds(t *testing.T) {
	app, r := fake(t)
	ctx := app.Ctx()

	if err := ctx.Err(); err != nil {
		t.Fatalf("the context is already done: %v", err)
	}
	for i := len(r.cleanups) - 1; i >= 0; i-- {
		r.cleanups[i]()
	}
	if ctx.Err() == nil {
		t.Error("the context is still live after cleanup, want it cancelled")
	}
}

// TestNothingReachesStderr is why the fixture replaces the request logger
// rather than leaving it alone. A router built by mizu.New logs a line per
// request to stderr, and a package of two hundred tests would print two hundred
// lines nobody reads.
func TestNothingReachesStderr(t *testing.T) {
	app := NewApp(t)
	app.Routes().Get("/ping", func(c *mizu.Ctx) error { return c.Text(200, "pong") })

	app.Get("/ping").Do()

	entries := app.Log().Entries()
	if len(entries) == 0 {
		t.Fatal("the request was not logged anywhere, want it in the recorder")
	}
	found := false
	for _, e := range entries {
		if e.Attrs["path"] == "/ping" {
			found = true
		}
	}
	if !found {
		t.Errorf("the request log does not mention the path. it holds:\n%s", app.Log())
	}
}

// TestHandlerErrorsAreRecorded is what makes the failure output useful. The
// reason for an unexpected 500 is in the log, and the log is kept.
func TestHandlerErrorsAreRecorded(t *testing.T) {
	app := NewApp(t)
	app.Routes().Get("/boom", func(c *mizu.Ctx) error {
		return errBoom
	})

	res := app.Get("/boom").Do()

	if res.Status() != http.StatusInternalServerError {
		t.Fatalf("the status is %d, want 500", res.Status())
	}
	// Two lines mention it, one from the router and one from the request
	// logger, and both carry the error itself as an attribute rather than as
	// text, which is what makes it readable in a failure.
	var carried int
	for _, e := range app.Log().Errors() {
		if e.Attrs["error"] == error(errBoom) {
			carried++
		}
	}
	if carried == 0 {
		t.Errorf("no error entry carries the error the handler returned:\n%s", app.Log())
	}
}

// TestRequestIDsAreInOrder keeps a response header from changing between runs,
// which is what a golden file over one needs.
func TestRequestIDsAreInOrder(t *testing.T) {
	app := NewApp(t)
	app.Routes().Get("/ping", func(c *mizu.Ctx) error { return c.Text(200, "pong") })

	for _, want := range []string{"req-1", "req-2", "req-3"} {
		app.Get("/ping").Do().AssertHeader("X-Request-Id", want)
	}
}

func TestServeTakesAHandlerOfYourOwn(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("pong"))
	})

	app := NewApp(t, Serve(mux))

	app.Get("/ping").Do().AssertOK().AssertSee("pong")
}

// TestServeSaysWhyThereAreNoRoutes is the one place the fixture cannot do what
// was asked, so it says so rather than handing back a nil router to panic on.
func TestServeSaysWhyThereAreNoRoutes(t *testing.T) {
	app, r := fake(t, Serve(http.NewServeMux()))

	app.Routes()

	r.says(t, "handler of its own")
}

func TestServeKeepsARouterUsable(t *testing.T) {
	own := mizu.New()
	own.Get("/ping", func(c *mizu.Ctx) error { return c.Text(200, "pong") })

	app := NewApp(t, Serve(own))
	app.Routes().Get("/pong", func(c *mizu.Ctx) error { return c.Text(200, "ping") })

	app.Get("/ping").Do().AssertSee("pong")
	app.Get("/pong").Do().AssertSee("ping")
	if len(app.Log().Entries()) == 0 {
		t.Error("a router passed to Serve is not logging into the recorder")
	}
}

// TestNoParallelIsRespected checks the option reaches the decision, using a
// fake that counts the calls it would have made.
func TestNoParallelIsRespected(t *testing.T) {
	p := &parallelRecorder{recorder: recorder{name: t.Name()}}
	NewApp(p)
	if p.parallel != 1 {
		t.Errorf("Parallel was called %d times, want 1", p.parallel)
	}

	q := &parallelRecorder{recorder: recorder{name: t.Name()}}
	NewApp(q, NoParallel())
	if q.parallel != 0 {
		t.Errorf("Parallel was called %d times with NoParallel, want 0", q.parallel)
	}
}

type parallelRecorder struct {
	recorder
	parallel int
}

func (p *parallelRecorder) Parallel() { p.parallel++ }

var errBoom = &boom{}

type boom struct{}

func (*boom) Error() string { return "boom" }
