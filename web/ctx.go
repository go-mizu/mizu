package web

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/router"
)

// A Ctx is one request being served.
//
// It lives for exactly as long as the handler does. The rules for that, and
// what enforces them, are in the package comment, and they are the only unusual
// thing about the type: everything else on it is a reader or a writer over the
// request and the response.
//
// The pair it wraps is reachable through [Ctx.Request] and [Ctx.Writer] for
// anything this package does not do. Neither is an exported field, so that
// every way to reach the request goes past the check that says the request is
// still running.
type Ctx struct {
	// res is the request's Recorder, and rec is where it lives when the Ctx is
	// the one that made it. A chain with middleware in it has already made one,
	// in which case res points at that and rec is unused, which is how a
	// handler and the middleware around it agree about what went out.
	res *Recorder
	rec Recorder

	r *http.Request

	route  *router.Route
	params router.Params

	log    *slog.Logger // built on first use, see Log
	status int          // what Status was told, 0 until it is told

	body []byte // what BodyBytes read, kept so it reads once
	read bool
	form bool // whether the form has been parsed, see values

	// gen is the generation counter the package comment describes. It counts
	// up on every acquire and goes to zero on release, so zero means the
	// request this Ctx belonged to has finished. In a build without the check
	// it is never read, and was is never written: it is what a stale use
	// panics with, and only the guarded build has anything to say.
	gen uint64
	was string
}

// Request is the request being served.
//
// It is the escape hatch: everything this package does not do is done with
// this and with [Ctx.Writer]. Nothing here hides them or wraps them in a type
// of its own, because a toolkit that cannot be stepped around is a framework.
func (c *Ctx) Request() *http.Request {
	c.live("Request")
	return c.r
}

// Writer is where the response goes.
//
// It is not the writer the server handed over. It is the request's [Recorder],
// so a handler that writes through this rather than through the helpers still
// gets a sensible answer from [Ctx.StatusCode], still gets an error page rather
// than a second status, and is still counted by whatever middleware is logging
// the request. Flushing, hijacking and the deadline calls go through
// http.ResponseController and reach the server's writer unchanged.
func (c *Ctx) Writer() http.ResponseWriter {
	c.live("Writer")
	return c.res
}

// record points the Ctx at the chain's Recorder, or at its own when the chain
// has not made one.
//
// The second case is a route with no middleware in front of it, and it is the
// one worth keeping cheap: the Recorder is a field of the pooled Ctx, so
// wrapping the server's writer costs nothing.
func (c *Ctx) record(w http.ResponseWriter) {
	if rec, ok := w.(*Recorder); ok {
		c.res = rec
		return
	}
	warnBuried(w)
	c.rec.ResponseWriter = w
	c.res = &c.rec
}

// Context is the request's context, and it is safe to keep.
//
// This is the answer to almost every question that starts with wanting to hold
// on to the Ctx. It carries the deadline, the cancellation and whatever the
// middleware put in it, it is an ordinary context.Context, and it has nothing
// to do with the pool.
//
// It is cancelled when the response finishes, which is what makes it the wrong
// context for work that outlives the response. [Ctx.Detach] is that one.
func (c *Ctx) Context() context.Context {
	c.live("Context")
	return c.r.Context()
}

// Detach is a context for work that outlives the response.
//
// It carries the values of the request's context and none of its cancellation,
// so a task started here keeps running after the client has its answer. What it
// does stop for is the server going down, when the runner has said which
// context means that with [WithShutdown].
//
// Starting work nothing waits for is a decision rather than a convenience. The
// task has to finish or give up on its own, and a task that does neither is
// what makes a graceful shutdown hang.
func (c *Ctx) Detach() context.Context {
	c.live("Detach")

	ctx := context.WithoutCancel(c.r.Context())
	stop, ok := ctx.Value(shutdownKey{}).(context.Context)
	if !ok {
		return ctx
	}
	ctx, cancel := context.WithCancel(ctx)
	context.AfterFunc(stop, cancel)
	return ctx
}

// shutdownKey holds the context that is cancelled when the server is going
// down, put there by the runner that owns the server.
type shutdownKey struct{}

// WithShutdown says which context means the server is going down.
//
// The runner calls it once, on the base context of the server, and every
// request served from that base inherits it. It is what gives [Ctx.Detach]
// something to stop for.
func WithShutdown(ctx, stop context.Context) context.Context {
	return context.WithValue(ctx, shutdownKey{}, stop)
}

// Deadline is when the request has to be answered by, if anything said.
func (c *Ctx) Deadline() (time.Time, bool) {
	c.live("Deadline")
	return c.r.Context().Deadline()
}

// Route is the route that matched, or nil.
//
// It is nil when this handler was reached by something other than a mizu
// router, which is a supported way to use the package and not a mistake, so
// this is worth checking rather than assuming.
func (c *Ctx) Route() *router.Route {
	c.live("Route")
	return c.route
}

// Params is every path parameter the route matched, in the order the pattern
// wrote them.
//
// [Ctx.Param] is the one to reach for when the name is known, which it is
// almost always. This is for printing a route, for a generator, and for
// middleware that does not know which route it is in front of.
func (c *Ctx) Params() router.Params {
	c.live("Params")
	return c.params
}

// Param is a path parameter, or the empty string when the route has no such
// wildcard.
//
// The empty string is also what a wildcard that matched nothing gives, and the
// two are not told apart here. A pattern is written once and read often, so a
// name that is not in it is a typo rather than a condition, and the reason it
// is not a panic is that a typo should not take the process down in front of a
// user.
func (c *Ctx) Param(name string) string {
	c.live("Param")
	return c.params.Get(name)
}

// ParamInt is a path parameter read as an integer.
//
// The second result is false when the route has no such wildcard, when it
// matched nothing, or when what it matched is not a number. A route written
// {id:int} has already been checked by the router, so this cannot fail there,
// and it is the one to write.
func (c *Ctx) ParamInt(name string) (int, bool) {
	c.live("ParamInt")
	n, err := strconv.Atoi(c.params.Get(name))
	if err != nil {
		return 0, false
	}
	return n, true
}

// RequestID is what identifies this request in the logs, or the empty string
// when nothing has set one.
//
// The middleware that reads it off a header or makes one up arrives with the
// middleware chain. Anything can set it in the meantime, with
// ctxdata.With(ctx, web.RequestIDKey, id).
func (c *Ctx) RequestID() string {
	c.live("RequestID")
	id, _ := ctxdata.Get(c.r.Context(), RequestIDKey)
	return id
}

// RequestIDKey is where the request id lives.
//
// It is a ctxdata key rather than a field on the Ctx because the id is set
// before there is a Ctx, by middleware that also puts it on the response, and
// because everything downstream of the handler reads it from the context.
// Logged means it comes out in the log line without anybody naming it.
var RequestIDKey = ctxdata.NewKey[string]("request_id", ctxdata.Logged(), ctxdata.Propagated())

// Log is a logger that already knows which request it is in.
//
// It carries the request id, the route, and whatever the middleware put in the
// context under a key marked ctxdata.Logged, so a handler logs the thing that
// happened and nothing about where it happened.
//
// It is built once, the first time it is asked for, since most requests never
// log anything.
func (c *Ctx) Log() *slog.Logger {
	c.live("Log")
	if c.log != nil {
		return c.log
	}

	attrs := ctxdata.Attrs(c.r.Context())
	if c.route != nil {
		if info := c.route.Info(); info.Name != "" {
			attrs = append(attrs, slog.String("route", info.Name))
		} else {
			attrs = append(attrs, slog.String("route", info.Pattern))
		}
	}

	c.log = slog.Default()
	if len(attrs) > 0 {
		c.log = c.log.With(anys(attrs)...)
	}
	return c.log
}

// anys is slog.Attr values in the shape With takes them.
func anys(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i, a := range attrs {
		out[i] = a
	}
	return out
}

// IP is the address the request came from.
//
// It reads what the connection says and believes nothing in a header, so behind
// a proxy this is the proxy until something in front of the handler has
// rewritten RemoteAddr from a forwarding header it had reason to trust. That is
// the middleware's job, and doing it here would mean trusting a header anybody
// can send.
//
// The zero Addr comes back when RemoteAddr is not an address, which happens in
// tests and over a Unix socket.
func (c *Ctx) IP() netip.Addr {
	c.live("IP")
	ap, err := netip.ParseAddrPort(c.r.RemoteAddr)
	if err != nil {
		addr, err := netip.ParseAddr(c.r.RemoteAddr)
		if err != nil {
			return netip.Addr{}
		}
		return addr
	}
	return ap.Addr().Unmap()
}

// reset puts a Ctx back to the state a fresh one is in.
//
// Everything is named here rather than assigned from a zero value, so that a
// field added without a line in this function is a compile error nowhere and a
// leak between two requests. The test that catches it is in ctx_test.go, and it
// reads this struct with reflection.
func (c *Ctx) reset() {
	c.res, c.rec, c.r = nil, Recorder{}, nil
	c.route, c.params = nil, router.Params{}
	c.log, c.status = nil, 0
	c.body, c.read, c.form = nil, false, false
	c.was = ""
}
