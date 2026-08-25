/*
Package mizu is a small foundation for an HTTP server.

A router over [net/http.ServeMux], a request context with the helpers a handler
reaches for, and an [App] that owns the server's lifetime. It is the
composition root of the toolkit, which is the one package the rest never
import. Everything else here works on its own, so a project can take
[github.com/go-mizu/mizu/errs] or [github.com/go-mizu/mizu/log] and nothing
underneath them, and a project that wants the whole thing starts here.

What is published today is v0.1.0, which predates the design the milestones
describe. The shape below is the starting point rather than the destination,
and M1 rewrites most of it. Nothing is under a compatibility promise until 1.0.

# Routing

A handler takes a [Ctx] and returns an error. Patterns are the ones
[net/http.ServeMux] understands, so a wildcard is written {name} and read with
[Ctx.Param], and a pattern ending in a slash matches the subtree under it.

	app := mizu.New()
	app.Get("/posts/{id}", show)
	app.Post("/posts", create)

[Router.Prefix] returns a router that writes a prefix in front of everything
registered on it, and [Router.Group] does the same for a block of routes.

# Middleware

Middleware is [Middleware], which is func([Handler]) [Handler]. [Router.Use]
adds it to everything, and [Router.With] returns a router that adds it to what
is registered on that one.

Standard net/http middleware goes through the Compat facade, which takes
func(http.Handler) http.Handler.

# The context

[Ctx] wraps the request and the response writer. It reads path parameters, the
query, forms and JSON, and writes text, HTML, JSON, a file, a stream or
server-sent events. One Ctx is made per request and reused for the whole of it,
so it is not safe to hold on to after the handler returns.

# Errors

A handler returns an error rather than writing a status and hoping the caller
remembers. [Router.ErrorHandler] is the one place that turns an error into a
response, and without one the router writes 500. A panic is recovered at the
[Router.ServeHTTP] boundary and goes to the same place as a [PanicError].

# Logging

[Logger] is a middleware writing one structured record per request through
[log/slog]. [LoggerOptions] chooses the format, the fields, and where the
request id and the trace context come from.

# Lifecycle

[App] embeds [Router] and adds the server. [App.Listen], [App.ListenTLS] and
[App.Serve] start it and block until it stops. A signal starts a graceful
shutdown: readiness flips first, so a load balancer takes the instance out
before the connections go, then in-flight requests get [App.ShutdownTimeout] to
finish.

[App.LivezHandler] and [App.ReadyzHandler] are the two health endpoints, and
they are handlers rather than routes so they can be served somewhere other than
the port the traffic is on.

# Testing

[App] implements [net/http.Handler], so a test serves it with
httptest.NewServer or calls ServeHTTP with a recorder. There is no test mode
and nothing to start.
*/
package mizu
