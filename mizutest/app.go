package mizutest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/clock"
)

// defaultAt is where the clock starts when a test does not say. It is a
// definite time and an obviously made up one, so a golden file that catches it
// by accident is recognisable rather than plausible.
var defaultAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// App is one application under test, with the clock, the log and the handler
// that the requests in a test go through.
//
// It is not safe for concurrent use. One test, one App, and the parallelism is
// between tests rather than inside one.
type App struct {
	tb      testing.TB
	handler http.Handler
	routes  *mizu.Router
	clock   *clock.FakeClock
	ctx     context.Context
	log     *Log
	ids     atomic.Int64
}

// NewApp returns a fixture bound to tb, with cleanup already registered.
//
// The application under test is a fresh [mizu.App] with the request logger
// pointed at a recorder rather than at stderr, so a passing test says nothing
// and a failing one says everything. Register routes on [App.Routes], or hand
// in a handler of your own with [Serve].
//
// It calls Parallel on tb unless [NoParallel] is given. See the package
// documentation for the two panics that can follow from that.
func NewApp(tb testing.TB, opts ...Option) *App {
	tb.Helper()

	s := settings{at: defaultAt, parallel: true}
	for _, opt := range opts {
		opt(&s)
	}
	if s.parallel {
		if p, ok := tb.(interface{ Parallel() }); ok {
			p.Parallel()
		}
	}

	a := &App{
		tb:    tb,
		clock: clock.Fake(s.at),
		log:   &Log{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	tb.Cleanup(cancel)
	a.ctx = clock.With(ctx, a.clock)

	switch h := s.handler.(type) {
	case nil:
		app := mizu.New()
		a.routes, a.handler = app.Router, app
	case *mizu.App:
		a.routes, a.handler = h.Router, h
	case *mizu.Router:
		a.routes, a.handler = h, h
	default:
		a.handler = h
	}
	if a.routes != nil {
		a.quieten()
	}
	return a
}

// quieten points the router at the recorder. The request logger is middleware
// holding the logger it was built with, so replacing it takes clearing the
// stack rather than SetLogger alone. Anything the test added itself is added
// after NewApp returns and so survives this.
func (a *App) quieten() {
	lg := slog.New(a.log.Handler())
	a.routes.SetLogger(lg)
	a.routes.ClearMiddleware()
	a.routes.Use(mizu.Logger(mizu.LoggerOptions{
		Logger:       lg,
		RequestIDGen: a.nextID,
	}))
}

// nextID hands out request ids in order, so that a response header or a golden
// file does not change from one run to the next.
func (a *App) nextID() string {
	return fmt.Sprintf("req-%d", a.ids.Add(1))
}

// Routes is the router the requests in this test go through, for registering
// the handlers under test.
//
//	app.Routes().Get("/posts/{id}", show)
//
// A fixture built with [Serve] over something that is not a mizu router has no
// routes to give back, and this reports that rather than returning nil.
func (a *App) Routes() *mizu.Router {
	a.tb.Helper()
	if a.routes == nil {
		a.tb.Fatal("mizutest: this fixture serves a handler of its own, so there is no router to register routes on")
		return nil
	}
	return a.routes
}

// Handler is what a request in this test is sent to. It is useful for wrapping
// the application in middleware for one test, or for handing it to something
// that wants an http.Handler.
func (a *App) Handler() http.Handler { return a.handler }

// Ctx is a context carrying this fixture's clock, cancelled when the test ends.
// It is the context to pass to anything a test calls directly rather than
// through a request.
func (a *App) Ctx() context.Context { return a.ctx }

// Clock is the clock the handlers see, which starts at [At] or at
// 2026-01-01T00:00:00Z and only moves when the test moves it.
func (a *App) Clock() *clock.FakeClock { return a.clock }

// Log is everything the application logged during the test.
func (a *App) Log() *Log { return a.log }

// settings is what the options collect, before there is an App to put them on.
type settings struct {
	at       time.Time
	parallel bool
	handler  http.Handler
}

// Option configures [NewApp].
type Option func(*settings)

// NoParallel keeps the test serial.
//
// Pass it when the test calls t.Parallel or t.Setenv itself, since both of
// those panic when the fixture has already made the test parallel.
func NoParallel() Option {
	return func(s *settings) { s.parallel = false }
}

// At starts the clock at a given time instead of the default.
//
//	app := mizutest.NewApp(t, mizutest.At(
//		time.Date(2026, 2, 28, 23, 59, 0, 0, time.UTC)))
func At(t time.Time) Option {
	return func(s *settings) { s.at = t }
}

// Serve tests a handler you built yourself rather than a fresh [mizu.App].
//
// It takes any http.Handler, which is what makes the fixture usable from a
// service that has one mizu route in it and everything else on a mux of its
// own. A [mizu.App] or [mizu.Router] passed here keeps [App.Routes] working and
// gets its logger pointed at the recorder like any other.
func Serve(h http.Handler) Option {
	return func(s *settings) { s.handler = h }
}
