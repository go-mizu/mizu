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
// Middleware that wants to change the response body rather than watch it calls
// [Recorder.Through], which puts a writer underneath the Recorder rather than
// around it. That is how a compressor sees bytes the handler wrote through
// [Ctx.Writer], which is the Recorder itself and so has nothing above it.
//
// The middleware nearly every service needs is in
// [github.com/go-mizu/mizu/web/mw], which is where RequestID, RealIP, Logger,
// Recover, Timeout, MaxBody, Concurrency, CORS, Secure, MethodOverride, Compress
// and ETag live.
//
// # Binding
//
// [Bind] fills a struct from the request, so that a handler reads its input
// once, in one place, with the types it wanted rather than the strings that
// arrived.
//
//	type search struct {
//		Q       string   `query:"q"`
//		Tags    []string `query:"tag"`
//		Page    int
//		PerPage int
//	}
//
//	in, err := web.Bind[search](c)
//
// A path, header or cookie tag says where a field comes from and is the whole
// answer. A form or query tag names the field in the query string and in a form
// body, which net/http keeps together and so does this. A field with none of
// those is read from the query and the form under the name in its json tag, or
// under its own name in snake case, so a struct written for a JSON body binds
// from a query string without a second set of tags on it. bind:"-" and json:"-"
// are the two ways to say a field is not input, and they are what stops a
// request from setting the fields the handler fills in itself.
//
// The body outranks the query string, so a search that posts a filter and
// carries the page number in the URL works. An embedded struct is flattened and
// a nested one is named under its field, which is address.city. A field the
// request did not name is left alone, so binding into a struct that already has
// something in it fills in the rest.
//
// A value that will not decode is reported rather than dropped. What comes back
// is an [github.com/go-mizu/mizu/errs.Error] of kind Invalid, carrying one
// Field per value, named as the request named it, with a code such as
// invalid_number and a message to put next to the field. Every field is
// reported, not the first, since a form should say everything wrong with it at
// once.
//
// An empty value is treated as a field nobody filled in rather than as a
// mistake, unless the field is a string, where an empty string is a value. A
// blank number input posts an empty string and that is not a client sending
// nonsense. Saying it had to be filled in is validation's job, which is the
// next section.
//
// # Validation
//
// [Bind] checks what it bound before it returns, so a handler that got a struct
// back got one whose rules passed.
//
//	type signup struct {
//		Email string `form:"email" validate:"required,email"`
//		Name  string `form:"name" validate:"required,min=2"`
//	}
//
// A struct with a Validate method is asked it, which is the method mizu
// gen:validate writes and the one somebody writes by hand for a rule that reads
// two fields at once. Everything else has its validate tags read by
// [github.com/go-mizu/mizu/validate.Struct]. The method wins because a
// generated one holds those same tags, and running both would report every
// failure twice. A hand-written method that wants the tags as well calls
// validate.Struct itself, which does not look for the method and so cannot
// recurse.
//
// What comes back is an error of kind Unprocessable, which is a 422, carrying
// one Field per rule that failed under the name the request used. That is the
// same shape a bind failure has, so a form redisplay does not care which it
// got.
//
// A request that would not bind is not checked. A field the decoder could not
// read is at its zero value, and answering that it was required would blame a
// box somebody filled in, so the 400 that names the field that would not decode
// is the whole answer.
//
// There is no way to bind without checking. A handler that wants the values as
// they arrived has [Ctx.Query], [Ctx.Form], [Ctx.Param] and [Ctx.JSON], which
// read the request without a plan and without a rule.
//
// # Bodies
//
// What the body is read as is what the request said it was. A JSON content
// type, or one ending in +json, is decoded as JSON, and an XML one as XML.
// Nothing else with a body in it is read, and a request that sends one under a
// type this does not decode comes back as an error of kind Unsupported, which
// is a 415. A body sent with no content type at all is left where it is and
// only the query string is read.
//
// A member the struct has no field for is a mistake, on the grounds that a
// client sending titel instead of title should hear about it rather than watch
// the field stay empty. A struct that embeds [AllowUnknown] says the opposite,
// which is what a webhook payload wants.
//
//	type hook struct {
//		web.AllowUnknown
//
//		Event string `json:"event"`
//	}
//
// [Ctx.JSON] reads the body as JSON into anything, without the query string,
// the path or the headers coming into it. It is for a payload that is not a
// struct, or one whose signature was already checked, since a body
// [Ctx.BodyBytes] has read is the same body afterwards.
//
// # Uploads
//
// A field of type *[Upload] binds a file out of a multipart form, and a
// []*[Upload] binds every file sent under the name.
//
//	type avatar struct {
//		Name  string      `form:"name"`
//		Image *web.Upload `form:"image"`
//	}
//
// An Upload is a handle rather than the bytes. A file under the in-memory limit
// is held in memory and a larger one is in a temporary file that net/http
// removes when the request is over, so anything that has to outlive the request
// is copied somewhere first. [Upload.Open] reads it, [Upload.Bytes] is the
// whole of a small one, and [Upload.Image] is the size of a picture without
// decoding the picture.
//
// Filename is what the client called the file with any directory in front of it
// removed, and it is a label rather than a name to write to disk with. MIME is
// what the file's first bytes say it is, not what the part header claimed, so a
// program that refuses anything but an image is checking something the client
// does not control.
//
// # What is not here yet
//
// Reading a request is here and so is enough writing to answer one. Storing an
// upload, content negotiation, the pagination types and the RFC 9457 renderer
// arrive with their own milestones.
//
// Scope, Locale, User and Session are in doc 08 and are not here, because each
// of them would return a type from a package that does not exist yet. They
// arrive with it.
package web
