package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-mizu/mizu/router"
)

// A Handler answers a request and says whether it worked.
//
// The error is the whole difference from net/http.HandlerFunc. A handler that
// can return means it can stop at the first thing that went wrong without
// having written half an answer first, and it means what a failure looks like
// is decided once, by [Errors], rather than in every handler that can fail.
//
// Returning nil means the response is written and finished with. Returning an
// error means it is not, and whatever the error handler does with it is the
// response.
type Handler func(c *Ctx) error

// Middleware is net/http's shape, unchanged.
//
// It is not a shape of this package's own, which is deliberate: middleware
// written for anything else works here, and middleware written here works
// anywhere. A middleware that wants the Ctx calls [FromContext].
type Middleware func(http.Handler) http.Handler

// ErrorHandler turns an error a handler returned into a response.
type ErrorHandler func(c *Ctx, err error)

// H adapts a [Handler] to a net/http.Handler.
//
// This is where the Ctx is taken from the pool, filled in, and put back, and it
// is the only place that happens. What comes out is an ordinary
// net/http.Handler, so it goes in a route table next to handlers that have
// never heard of this package.
//
// The Ctx is put back when h returns, whether it returned an error or panicked,
// so a panic that something upstream recovers does not leak one. What it does
// not do is wait for anything h started: a goroutine still holding the Ctx at
// that point is the bug the package comment is about, and conc.Go is the way to
// start one that finishes first.
func H(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := acquire()
		defer release(c)

		c.res.ResponseWriter = w
		c.route, c.params, _ = router.Matched(r)

		// The Ctx goes in the context so that anything the handler calls that
		// was handed nothing but a context can find it. It is a pointer to the
		// Ctx being served, so everything the package comment says about
		// holding one applies to what comes back out of FromContext too.
		//
		// This is the one allocation the package adds to a request, and it is
		// two: the context node, and the shallow copy of the request that
		// carries it, because net/http has no way to change a request's context
		// without one. It happens whether the handler asks or not, since a
		// request whose context has the Ctx only sometimes is worse than one
		// that costs a little more every time.
		c.r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, c))

		if err := h(c); err != nil {
			c.fail(err)
		}
	})
}

// HF adapts a net/http.HandlerFunc to a [Handler].
//
// It is for a handler that wants nothing to do with the Ctx and still has to go
// in a table of handlers that do. It never fails, because the thing it is
// wrapping has no way to say so.
func HF(h http.HandlerFunc) Handler {
	return func(c *Ctx) error {
		h(c.Writer(), c.Request())
		return nil
	}
}

// ctxKey is where H leaves the Ctx for FromContext to find.
//
// It is a plain context key rather than a ctxdata one because a ctxdata key is
// for something that gets logged or propagated to the next service, and neither
// is true of a pointer that stops being valid when the handler returns.
type ctxKey struct{}

// FromContext is the Ctx being served, for code that was handed a context and
// needs more than one.
//
// It is how middleware written in net/http's shape reaches the request id, the
// route and the logger without taking them apart again. The second result is
// false when the request did not come through [H], which is a supported way to
// arrive here.
//
// What comes back belongs to the handler that is still running. Everything in
// the package comment about not keeping a *Ctx applies to it.
func FromContext(ctx context.Context) (*Ctx, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Ctx)
	return c, ok
}

// errorKey is where Errors leaves the handler for fail to find.
type errorKey struct{}

// Errors returns middleware that decides what a failed handler looks like.
//
// One of these in front of the route table is how an application says that
// once, and everything under it uses it. Without one, [DefaultErrors] answers.
//
//	mux.Handle("/", web.Errors(problem.Render)(routes))
func Errors(fn ErrorHandler) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), errorKey{}, fn)))
		})
	}
}

// fail hands an error to whatever is meant to render it.
//
// The lookup happens here rather than on the way in, because most requests do
// not fail and a request that does not fail should not pay for the machinery
// that renders one.
func (c *Ctx) fail(err error) {
	if fn, ok := c.r.Context().Value(errorKey{}).(ErrorHandler); ok {
		fn(c, err)
		return
	}
	DefaultErrors(c, err)
}

// DefaultErrors is what a failed handler looks like when nothing has said
// otherwise.
//
// It logs the error and writes a 500 that says nothing else, because an error
// nobody has decided how to render is an error whose text was written for a
// developer. The renderer that turns one into an RFC 9457 document, with the
// status the error asked for, arrives with the response helpers.
//
// A handler that has already started writing gets the log and nothing else,
// since the status is gone by then and a second one would be a warning in the
// server's log rather than anything the client sees.
func DefaultErrors(c *Ctx, err error) {
	c.Log().LogAttrs(c.Context(), slog.LevelError, "handler failed", slog.Any("error", err))
	if c.res.wrote {
		return
	}
	http.Error(c.Writer(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
