// Package web is the request handle a handler is written against.
//
// A handler takes one argument and returns an error:
//
//	func show(c *web.Ctx) error {
//		return c.Text("post " + c.Param("id"))
//	}
//
//	r.Handle("GET /posts/{id:int}", web.H(show))
//
// [H] adapts it to an [net/http.Handler], which is what the router stores, so
// a mizu handler and a plain one sit next to each other in the same table.
// Nothing below this package knows which kind it is holding.
//
// # Why a Ctx at all
//
// http.ResponseWriter and *http.Request are enough to serve a request and they
// are not enough to write an application against. Everything an application
// does with a request goes looking for something that is not in either of them:
// the route that matched, the logger already tagged with the request id, the
// user, the session, the locale. Fetching each of those out of the request
// context by hand is the same six lines at the top of every handler, and the
// version of those six lines that is wrong in one handler is the one nobody
// notices.
//
// So there is one type that has them, and it is passed rather than reached for.
//
// The other half is the error. A handler that returns an error is a handler
// that can stop in the middle without having written half a response, and it
// puts the decision about what a failure looks like in one place rather than in
// every handler. What that place does is [Errors], and until the RFC 9457
// renderer lands it is a 500 and a line in the log.
//
// # The pooling contract
//
// A *Ctx comes from a pool and goes back when the handler returns. That makes
// it the one place in the toolkit where a use after free is possible, so the
// rules are short and they are enforced rather than written down.
//
// A *Ctx must not outlive the handler. Do not store it in a struct, do not
// return it, do not close over it in a goroutine, do not put it in a channel.
//
// To do work after the handler returns, take what you need out of the Ctx
// first. [Ctx.Context] is safe to keep, because it is the request's context and
// not the Ctx. [Ctx.Detach] is the one to keep when the work outlives the
// response.
//
//	func store(c *web.Ctx) error {
//		ctx, body := c.Detach(), c.Param("id")
//		conc.Go(c.Context(), func(context.Context) error {
//			return archive(ctx, body) // ctx and body, never c
//		})
//		return c.NoContent()
//	}
//
// Two things enforce that. Building with -race or with -tags mizudebug turns on
// a generation counter: the pool is bypassed, every Ctx is used once, and every
// method checks that the request it belongs to is still running. A stale one
// panics with what was called and which route it belonged to, which is a test
// failure rather than a wrong answer served to somebody. In an ordinary build
// the check compiles to nothing.
//
// The other is mizu lint ctx, which reads the source rather than running it and
// reports a *web.Ctx stored in a field, returned from a function, or captured
// by a go statement. It runs as a stage of mizu verify, so the first person to
// write one hears about it before the tests do.
//
// # Middleware
//
// A [Middleware] is func(http.Handler) http.Handler, which is net/http's shape
// and not one of this package's own, so middleware written for anything else
// works here and middleware written here works anywhere.
//
// [Chain] puts a handler inside some, outermost first:
//
//	srv := web.Chain(routes, mw.RequestID(), mw.Logger(l), mw.Recover())
//
// [Stack] is the same with names on the layers, so that the order they run in
// can be declared apart from the order they were added in. That matters for the
// handful of pairs where the wrong order is wrong rather than different: a
// session started after the middleware that reads it looks like a user who is
// not logged in, and nothing about it looks like a bug.
//
// Middleware that wants to know how the request was answered calls [Record] on
// the way in and reads the [Recorder] on the way out. There is one Recorder per
// request and everything inside shares it, including the Ctx, so a chain of ten
// wraps the response writer once and http.ResponseController still reaches the
// server's writer through it.
//
// The middleware nearly every service needs is in
// [github.com/go-mizu/mizu/web/mw], which is where RequestID, RealIP, Logger,
// Recover, Timeout, MaxBody and Concurrency live.
//
// # What is not here yet
//
// Reading a request is here and so is enough writing to answer one. Binding,
// validation, content negotiation, the pagination types and the RFC 9457
// renderer arrive with their own milestones.
//
// Scope, Locale, User and Session are in doc 08 and are not here, because each
// of them would return a type from a package that does not exist yet. They
// arrive with it.
package web
