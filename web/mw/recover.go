package mw

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/web"
)

// Recover turns a panic in a handler into a 500 and a line in the log.
//
// Without it, net/http catches the panic, prints the stack to the server's error
// log and closes the connection without a response. The client sees a connection
// reset, which is indistinguishable from the process having died, and the stack
// goes somewhere that is not the structured log everything else is in.
//
// This is the outermost middleware. A panic in the middleware inside it is a
// panic like any other, and anything it does not wrap is a dropped connection.
//
// The record is at ERROR, carries the stack under the key stack, and carries the
// panic value under panic. It also carries the request id and everything else
// marked Logged, so the 500 the client saw and the stack that caused it are
// found by the same query.
//
// The response is a bare 500. A handler that had already started writing gets the
// record and nothing else, since the status went out before the panic and a
// second one would be a warning in the server's log rather than anything the
// client sees. The RFC 9457 document arrives with the response helpers, and it
// arrives here too.
//
// http.ErrAbortHandler is passed along rather than recovered. It is net/http's
// way for a handler to say stop and close the connection without logging, which
// httputil.ReverseProxy uses when the client hangs up, and treating it as a
// failure would mean an ERROR every time somebody closes a tab.
func Recover(l *slog.Logger) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := web.Record(w)

			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if v == http.ErrAbortHandler {
					panic(v)
				}

				// debug.Stack here is the stack at the point of the panic, since
				// a deferred function runs before its frame is gone.
				l.LogAttrs(r.Context(), slog.LevelError, "handler panicked",
					append(ctxdata.Attrs(r.Context()),
						slog.Any("panic", v),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.String("stack", string(debug.Stack())),
					)...)

				if rec.Status() == 0 {
					http.Error(rec, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(rec, r)
		})
	}
}
