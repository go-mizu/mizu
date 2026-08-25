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
// hijacking and the deadline calls all still work through this. ReadFrom is here
// for the same reason: without it, an io.Copy into the response loses the fast
// path the server has for a file.
//
// A middleware that has to see the body rather than count it, which is what a
// compressor and an ETag are, puts a filter under the recorder with [Through]
// rather than wrapping it. [Hold] keeps the status back for the one that cannot
// know what the response is until the body is done.
type Recorder struct {
	http.ResponseWriter

	status  int
	written int64

	// filters is the stack Through builds, innermost last. The body goes to
	// the last one, and each writes to the one before it.
	filters []io.Writer

	// held is set while Hold is keeping the status and the headers back.
	held bool
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
	if rec.held {
		// While the status is held the last word wins, which is the whole point
		// of holding it: the middleware that has read the body is the one that
		// knows what the response turned out to be.
		rec.status = code
		return
	}
	if rec.status == 0 {
		rec.status = code
	}
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *Recorder) Write(p []byte) (int, error) {
	rec.began()
	return rec.Sink().Write(p)
}

func (rec *Recorder) ReadFrom(r io.Reader) (int64, error) {
	rec.began()
	if len(rec.filters) > 0 {
		// The server's fast path copies to the socket, which is the one place
		// these bytes are not going.
		return io.Copy(rec.filters[len(rec.filters)-1], r)
	}
	rec.release()
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

// FlushError sends what has been written, and is the method
// http.ResponseController.Flush looks for before it looks for an http.Flusher.
//
// It is here so that a flush reaches the filters first. Without it the
// controller would walk past the recorder to the server's writer, a compressor's
// buffer would still be sitting in memory, and the flush would report success.
//
// The name is FlushError rather than Flush on purpose. A Flush method would make
// every Recorder an http.Flusher whether or not the writer underneath can flush,
// and a handler that type asserts for one to decide whether it can stream would
// get the wrong answer.
func (rec *Recorder) FlushError() error {
	for i := len(rec.filters) - 1; i >= 0; i-- {
		f, ok := rec.filters[i].(interface{ Flush() error })
		if !ok {
			continue
		}
		if err := f.Flush(); err != nil {
			return err
		}
	}
	rec.began()
	rec.release()
	return http.NewResponseController(rec.ResponseWriter).Flush()
}

// Sink is where the body is going right now.
//
// That is the innermost filter when [Through] has installed one, and otherwise
// the server's writer with the byte count attached. A filter writes what it
// produces here, so [Recorder.Written] stays the number of bytes that reached
// the client rather than the number the handler offered.
//
// Read it before installing a filter, not after, or the filter is writing to
// itself:
//
//	gz := gzip.NewWriter(rec.Sink())
//	defer rec.Through(gz)()
func (rec *Recorder) Sink() io.Writer {
	if n := len(rec.filters); n > 0 {
		return rec.filters[n-1]
	}
	return sink{rec}
}

// Through sends the response body through w until the returned function puts it
// back.
//
// This is how a middleware that has to see the bytes, rather than count them,
// gets at them. The filter goes under the recorder rather than around it,
// because [Ctx.Writer] hands the handler the recorder itself, so a wrapper
// placed above it would never see a write.
//
//	rec := web.Record(w)
//	gz := gzip.NewWriter(rec.Sink())
//	defer rec.Through(gz)()
//	defer gz.Close()
//
//	next.ServeHTTP(rec, r)
//
// Filters nest, innermost last, and each one writes to the [Recorder.Sink] it
// read before installing itself. So an ETag inside a compressor hashes the body
// the handler wrote and the compressor compresses whatever the ETag let past.
// The returned function puts back whatever was there before, so a deferred call
// unwinds the stack in the order it was built.
//
// A filter that buffers should have a Flush method, since that is what
// [Recorder.FlushError] calls on the way down.
func (rec *Recorder) Through(w io.Writer) (restore func()) {
	depth := len(rec.filters)
	rec.filters = append(rec.filters, w)
	return func() { rec.filters = rec.filters[:depth] }
}

// Hold keeps the status and the headers back until the returned function sends
// them.
//
// A middleware that cannot know what the response is until it has seen the whole
// body needs this. An ETag is the example: the handler writes a 200, the
// middleware hashes what came out, the hash turns out to be one the client
// already has, and there is still a 304 to be sent because nothing has gone out
// yet.
//
//	send := rec.Hold()
//	defer send()
//
// The header map stays writable the whole time, and [Recorder.WriteHeader] takes
// the last status it is given rather than the first, so the middleware sets what
// it learned and then sends. The hold ends on its own the moment a byte of body
// reaches the server or the response is flushed, because a status has to go out
// in front of a body either way.
//
// Holding a response that has already started does nothing. The status is gone
// and there is nothing left to decide.
func (rec *Recorder) Hold() (send func()) {
	if rec.held {
		panic("web.Recorder: the response is already held, and two middleware holding it is two middleware deciding the status")
	}
	rec.held = rec.status == 0
	return rec.release
}

// release sends the status and the headers if they are still being held.
func (rec *Recorder) release() {
	if !rec.held {
		return
	}
	rec.held = false
	rec.began()
	rec.ResponseWriter.WriteHeader(rec.status)
}

// began records the implied 200 that a write with no WriteHeader in front of it
// sends.
func (rec *Recorder) began() {
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
}

// sink is the bottom of the filter stack: the server's writer, with the held
// header sent in front of the first byte and the byte count kept.
type sink struct{ rec *Recorder }

func (s sink) Write(p []byte) (int, error) {
	s.rec.began()
	s.rec.release()
	n, err := s.rec.ResponseWriter.Write(p)
	s.rec.written += int64(n)
	return n, err
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
