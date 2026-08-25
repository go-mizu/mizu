package mw

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-mizu/mizu/web"
)

// Timeout puts a deadline on the request context and answers 504 when the
// handler runs past it without having written anything.
//
// The deadline is the point. Cancelling the context is what stops the database
// query, the outbound HTTP call and the retry loop underneath the handler, and
// every one of those is where a slow request is actually spending its time. A
// handler that ignores its context is not stopped by this, and there is no way
// to stop one from outside in Go.
//
// It is not net/http's TimeoutHandler. That one buffers the whole response in
// memory so it can throw it away and write a 504 instead, which means it cannot
// be used in front of a download, a server sent event stream or anything else
// that writes as it goes. This writes through, so the 504 is only available
// while the response has not started, which is the trade and it is the right way
// round: a slow handler that has already sent a status has already told the
// client something.
//
// Put it inside [Recover] and [Logger], so that the 504 is logged, and outside
// anything that does work.
func Timeout(d time.Duration) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			rec := web.Record(w)
			next.ServeHTTP(rec, r.WithContext(ctx))

			if errors.Is(ctx.Err(), context.DeadlineExceeded) && rec.Status() == 0 {
				http.Error(rec, http.StatusText(http.StatusGatewayTimeout), http.StatusGatewayTimeout)
			}
		})
	}
}

// MaxBody caps how many bytes of request body a handler can read.
//
// Without a cap, a request body is whatever the client decides to send, and a
// handler that decodes JSON into a struct will read all of it first. One client
// and one long upload is enough to take a service down, and it does not look
// like an attack in any log.
//
// Two things happen. A request whose Content-Length is over the limit is refused
// with a 413 before the handler runs, which is the case for an honest client and
// costs nothing. Everything else goes through http.MaxBytesReader, so a body
// that runs over while it is being read fails the read with an
// *net/http.MaxBytesError and the connection is closed rather than drained.
//
// The second one surfaces as an error from whatever was reading, which today is
// [github.com/go-mizu/mizu/web.Ctx.Body] and the error handler. Turning that
// error into a 413 rather than a 500 is the binding layer's job and it arrives
// with it.
func MaxBody(n int64) web.Middleware {
	return func(next http.Handler) http.Handler {
		capped := http.MaxBytesHandler(next, n)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > n {
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
			capped.ServeHTTP(w, r)
		})
	}
}

// Concurrency caps how many requests are inside it at once, and answers 503 with
// Retry-After to the ones that arrive when it is full.
//
// It is a bulkhead rather than a rate limit. A rate limit is about fairness
// between callers and it belongs per route and per caller; this is about the one
// resource behind the handler, usually a connection pool or the memory a request
// needs while it runs, and it is about the process rather than about anybody in
// particular.
//
// It refuses rather than queues. Holding a request until a slot opens turns a
// busy service into an unresponsive one, and it does it invisibly: the queue
// grows, every request gets slower, and nothing in the metrics says the limit was
// reached. A 503 says it, the client's retry handles it, and a load balancer in
// front takes the instance out of rotation on its own.
//
// The count belongs to the value this returns, so applying the same middleware to
// two chains gives them one shared limit, which is what a limit on a process
// means. Two calls to Concurrency are two limits.
//
// It panics if n is less than one, since a limit of zero refuses every request
// and is a typo rather than a configuration.
func Concurrency(n int) web.Middleware {
	if n < 1 {
		panic("mw.Concurrency: the limit has to be at least one")
	}
	slots := make(chan struct{}, n)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			default:
				w.Header().Set("Retry-After", "1")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
