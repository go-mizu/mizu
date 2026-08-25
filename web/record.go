package web

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

// A Recorder is the http.ResponseWriter a request is served through.
//
// It wraps the server's writer and remembers two things about the response: the
// status that went out and how many bytes followed it. That is what an access
// log reports, what a metric counts, and what the error handler checks before
// deciding there is still a response to write.
//
// There is one of these per request, made by whatever is outermost in the chain
// and passed down to everything inside it. [Record] is how middleware gets hold
// of it and [Ctx.StatusCode] is how a handler does, so neither has to wrap the
// writer again.
//
// Unwrap is what http.ResponseController uses to reach the server's writer, so
// flushing, hijacking and the deadline calls all still work through this.
// ReadFrom is here for the same reason: without it, an io.Copy into the response
// loses the fast path the server has for a file.
type Recorder struct {
	http.ResponseWriter

	status  int
	written int64
}

// Record is the [Recorder] for this request, wrapping w if it is not one
// already.
//
// Middleware that wants to know how the request was answered calls this on the
// way in and reads the result on the way out:
//
//	func Logger(l *slog.Logger) web.Middleware {
//		return func(next http.Handler) http.Handler {
//			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				rec := web.Record(w)
//				next.ServeHTTP(rec, r)
//				l.Info("request", "status", rec.Status(), "bytes", rec.Written())
//			})
//		}
//	}
//
// Passing rec on rather than w is the whole of the contract. Everything further
// in, including [H], finds it and records into it, so a chain of ten middleware
// wraps the writer once.
func Record(w http.ResponseWriter) *Recorder {
	if rec, ok := w.(*Recorder); ok {
		return rec
	}
	warnBuried(w)
	return &Recorder{ResponseWriter: w}
}

// Status is the status that went out, or zero when nothing has gone out yet.
func (rec *Recorder) Status() int { return rec.status }

// Written is how many bytes of body have gone out.
//
// It counts what the server accepted rather than what the handler offered, so a
// write that failed part of the way through is counted for the part that
// landed. Zero with a status set is a response with no body, which is what a 204
// and a 304 are.
func (rec *Recorder) Written() int64 { return rec.written }

// Unwrap is the writer the server handed over.
func (rec *Recorder) Unwrap() http.ResponseWriter { return rec.ResponseWriter }

func (rec *Recorder) WriteHeader(code int) {
	if rec.status == 0 {
		rec.status = code
	}
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *Recorder) Write(p []byte) (int, error) {
	rec.began()
	n, err := rec.ResponseWriter.Write(p)
	rec.written += int64(n)
	return n, err
}

func (rec *Recorder) ReadFrom(r io.Reader) (int64, error) {
	rec.began()
	n, err := rec.readFrom(r)
	rec.written += n
	return n, err
}

func (rec *Recorder) readFrom(r io.Reader) (int64, error) {
	if rf, ok := rec.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}
	return io.Copy(rec.ResponseWriter, r)
}

// began records the implied 200 that a write with no WriteHeader in front of it
// sends.
func (rec *Recorder) began() {
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
}

// warnBuried says something when a Recorder is already in the chain and
// something else has wrapped it.
//
// Middleware from anywhere else wraps the writer in a type of its own, and once
// it has, the Recorder underneath is unreachable and the one made here is the
// second on the request. Both of them record, the inner one sees whatever the
// outer one passed through, and an io.Copy into the response only reaches
// sendfile if the writer in between forwards ReadFrom, which most do not.
//
// None of that is broken, and all of it is slower than it looks, which is why
// it is a line in the log rather than an error. It is said once per writer type
// because it is a fact about the chain and not about the request.
func warnBuried(w http.ResponseWriter) {
	if !buried(w) {
		return
	}
	name := fmt.Sprintf("%T", w)
	if _, said := foreign.LoadOrStore(name, struct{}{}); said {
		return
	}
	slog.Warn("a response writer in the chain is wrapped by something that is not mizu, so the response is recorded twice and a file copy no longer reaches sendfile",
		slog.String("writer", name))
}

// foreign is the writer types warnBuried has already mentioned.
var foreign sync.Map

// buried reports whether there is a Recorder somewhere under w.
//
// The walk is the one http.ResponseController does, and it stops after sixteen
// wrappers: past that the chain is either a cycle or a mistake, and either way
// the answer is not worth hanging a request over.
func buried(w http.ResponseWriter) bool {
	for range 16 {
		u, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		if w = u.Unwrap(); w == nil {
			return false
		}
		if _, ok := w.(*Recorder); ok {
			return true
		}
	}
	return false
}
