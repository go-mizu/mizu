package mw

import (
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/go-mizu/mizu/clock"
	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/router"
	"github.com/go-mizu/mizu/web"
)

// Logger writes one record per request, after the request is answered.
//
// One record, with a fixed set of keys, is the whole design. A service that logs
// a line on the way in and another on the way out has doubled the size of its
// log to say nothing more, and the two halves have to be stitched back together
// by whoever is reading. A service whose key set drifts from handler to handler
// cannot be queried at all.
//
// The keys are request_id, method, path, route, status, dur, bytes, ip and ua,
// and doc 06 section 5.2 is where that list is fixed. Anything a middleware put
// in the context under a [github.com/go-mizu/mizu/ctxdata] key marked Logged
// comes out alongside them, which is how the tenant, the user and the locale get
// into the line without this package knowing they exist.
//
// The level is INFO, except for a 5xx, which is ERROR. A 4xx stays at INFO
// because a client asking for something that is not there is the service working.
//
// # Where the route comes from
//
// The route key is there when the router has already matched, and absent when it
// has not. Wrapped around a whole router, this runs before the match, so there
// is no route to name:
//
//	web.Chain(routes, mw.RequestID(), mw.Logger(log))   // no route key
//
// Inside a route, there is:
//
//	r.Handle("GET /posts/{id}", web.Chain(web.H(show), mw.Logger(log)))
//
// The first is what most services want and the missing key is the price. Route
// level middleware without the repetition arrives with router groups.
func Logger(l *slog.Logger) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := web.Record(w)
			start := clock.Now(r.Context())

			next.ServeHTTP(rec, r)

			// The context is read after the fact so that the request id, which
			// is set by middleware outside this one, is on it.
			ctx := r.Context()
			status := rec.Status()
			if status == 0 {
				// A handler that returned without writing anything still sent a
				// 200 and an empty body, because net/http writes the header.
				status = http.StatusOK
			}

			attrs := append(ctxdata.Attrs(ctx),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)
			if name := matched(r); name != "" {
				attrs = append(attrs, slog.String("route", name))
			}
			attrs = append(attrs,
				slog.Int("status", status),
				slog.Duration("dur", clock.Since(ctx, start)),
				slog.Int64("bytes", rec.Written()),
				slog.String("ip", from(r)),
				slog.String("ua", r.UserAgent()),
			)

			level := slog.LevelInfo
			if status >= http.StatusInternalServerError {
				level = slog.LevelError
			}
			l.LogAttrs(ctx, level, "request", attrs...)
		})
	}
}

// matched is what to call the route that ran, or the empty string when the
// router has not matched yet.
//
// The name wins over the pattern, the same way
// [github.com/go-mizu/mizu/web.Ctx.Log] picks, so that renaming a URL does not
// split a dashboard in two.
func matched(r *http.Request) string {
	route, _, ok := router.Matched(r)
	if !ok {
		return ""
	}
	info := route.Info()
	if info.Name != "" {
		return info.Name
	}
	return info.Pattern
}

// from is the address on the request, without the port.
//
// Behind a proxy this is the proxy unless [RealIP] has run, which is the same
// rule [github.com/go-mizu/mizu/web.Ctx.IP] follows and for the same reason.
func from(r *http.Request) string {
	if ap, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		return ap.Addr().Unmap().String()
	}
	if a, err := netip.ParseAddr(r.RemoteAddr); err == nil {
		return a.Unmap().String()
	}
	return r.RemoteAddr
}
