package micro

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu/web"
	"github.com/go-mizu/mizu/web/mw"
)

func init() {
	register("ctx/acquire", benchCtxAcquire)
	register("mw/chain", benchChain)
	register("mw/requestid", benchRequestID)
	register("mw/lifecycle", benchLifecycle)
}

// benchChain is what a middleware chain costs before any of it does anything.
//
// Eight layers, each with a frame of its own and nothing in it, in front of a
// handler that also does nothing. What is left in the loop is the eight indirect
// calls in and the eight returns out, which is the floor under every chain an
// application builds and the number to compare a middleware's own cost against.
//
// The stack is built once, which is where the ordering work happens, so a
// request pays for the calls and nothing else.
func benchChain(b *testing.B) {
	var s web.Stack
	for i := range 8 {
		s.Add(string(rune('a'+i)), func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
	}
	s.Priority("h", "g", "f", "e", "d", "c", "b", "a")

	h := s.Then(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	r := httptest.NewRequest("GET", "/things/7", nil)
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(w, r)
	}
}

// benchRequestID is the one middleware that does work on every request whether
// anything reads the result or not.
//
// What is in the loop is a ULID, which is a clock read and ten bytes out of
// crypto/rand, then the context node and the request copy that carry the id
// inward, then the response header. It has a row of its own because it is the
// floor under any chain that logs, and because the id is the one thing here that
// cannot be made lazy: the response header has to go out before the handler
// starts writing.
func benchRequestID(b *testing.B) {
	h := mw.RequestID()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	r := httptest.NewRequest("GET", "/things/7", nil)
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(w, r)
	}
}

// benchLifecycle is the chain a service actually runs, in front of a handler
// that does nothing.
//
// Six middleware, a real access log record written to io.Discard through a JSON
// handler, and web.H at the bottom. Everything in it is work a request pays for
// before the application has been asked anything, so this is the number to
// subtract when a route looks slower than the handler in it.
//
// The log record is the largest part and it is deliberately not stubbed out. A
// service that does not write one is a service nobody can operate, so the
// realistic floor is the one with it in.
func benchLifecycle(b *testing.B) {
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))

	h := web.Chain(web.H(func(*web.Ctx) error { return nil }),
		mw.Recover(log),
		mw.RealIP(mw.Private()...),
		mw.RequestID(),
		mw.Logger(log),
		mw.MaxBody(1<<20),
		mw.Timeout(10*time.Second),
	)

	r := httptest.NewRequest("GET", "/things/7", nil)
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(w, r)
	}
}

// benchCtxAcquire is what the web package adds to a request and nothing else.
//
// The handler does no work, the writer is a recorder that is reused, and the
// request is built once, so what is left in the loop is web.H: take a Ctx, fill
// it in, put the Ctx in the request context, run the handler, put the Ctx back.
// Every request an application serves pays this, which is why it has a row.
//
// The two allocations are the request context. Putting the Ctx where
// web.FromContext can find it means a new context node and the shallow copy of
// the request that carries it, and net/http has no way to change a request's
// context without one. The Ctx itself allocates nothing, which is what the pool
// is for.
func benchCtxAcquire(b *testing.B) {
	h := web.H(func(c *web.Ctx) error { return nil })
	r := httptest.NewRequest("GET", "/things/7", nil)
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(w, r)
	}
}

// discardWriter is a ResponseWriter that keeps nothing, so the benchmark
// measures the Ctx rather than a recorder's buffer.
type discardWriter struct{ header http.Header }

func (d *discardWriter) Header() http.Header         { return d.header }
func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }
func (d *discardWriter) WriteHeader(int)             {}
