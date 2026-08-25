// Package mw is the middleware nearly every service ends up writing.
//
// Each constructor returns a [github.com/go-mizu/mizu/web.Middleware], which is
// net/http's func(http.Handler) http.Handler and nothing else, so these go in a
// mizu chain, in a chi chain, or straight around an http.ServeMux.
//
//	srv := web.Chain(routes,
//		mw.Recover(log),
//		mw.RealIP(mw.Private()...),
//		mw.RequestID(),
//		mw.Logger(log),
//		mw.MaxBody(1<<20),
//		mw.Timeout(10*time.Second),
//	)
//
// # The order they go in
//
// Outermost first, which is the order [github.com/go-mizu/mizu/web.Chain] takes
// them in and the order they are listed above.
//
// Three of the pairs are wrong rather than different in the other order, and
// none of them look wrong from the outside:
//
// [Recover] goes outside everything, because a panic anywhere under it is what
// it is for and anything it does not wrap is a dropped connection.
//
// [RealIP] goes outside [Logger], because the log line reports the address on
// the request as it stands when the record is written.
//
// [RequestID] goes outside [Logger] for the same reason. The id lives in the
// context, the context belongs to the request the middleware passed inward, and
// a Logger wrapped around RequestID rather than inside it is holding the request
// from before the id was set. The line comes out with no request_id on it and
// nothing else about it looks wrong.
//
// [github.com/go-mizu/mizu/web.Stack] is the way to say that once rather than
// depending on the order somebody happened to add them in.
//
// # What each one costs
//
// [Recover], [Timeout] and [Logger] take a [github.com/go-mizu/mizu/web.Recorder]
// on the way in, which is free when one of the others already made one and one
// small allocation when none did. [RequestID] costs a context node, a request
// copy and the twenty six bytes of the id. [RealIP] copies the request only when
// it has something to rewrite. [MaxBody] and [Timeout] copy the request always,
// since both change something on it. [Concurrency] costs a channel send and a
// receive.
//
// # What is not here yet
//
// The table in doc 07 section 5.3 has twenty six of these and this package has
// seven. The response side, Compress, ETag, CORS, Secure and MethodOverride,
// arrives next. The other twelve each need a package that does not exist yet:
// sessions and CSRF, the auth guards, rate limiting, tracing and metrics,
// locales, maintenance mode and Telescope.
package mw
