package micro

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/web"
	"github.com/go-mizu/mizu/web/mw"
)

func init() {
	register("ctx/acquire", benchCtxAcquire)
	register("mw/chain", benchChain)
	register("mw/requestid", benchRequestID)
	register("mw/lifecycle", benchLifecycle)
	register("mw/compress", benchCompress)
	register("mw/etag", benchETag)
	register("bind/form", benchBindForm)
	register("bind/json", benchBindJSON)
}

// page is the response the two body middleware are measured against: eight
// kilobytes of markup, which is a middling server rendered page.
var page = []byte(strings.Repeat("<p>mizu is water and water is mizu.</p>\n", 200))

// html serves page with a content type on it.
func html() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
}

// benchChain is what a middleware chain costs before any of it does anything.
//
// Eight layers, each with a frame of its own and nothing in it, in front of a
// handler that also does nothing. What is left in the loop is the eight indirect
// calls in and the eight returns out, which is the floor under every chain an
// application builds and the number to compare a middleware's own cost against.
//
// The stack is built once, which is where the ordering work happens, so a
// request pays for the calls and nothing else.
func benchChain(b *testing.B) {
	var s web.Stack
	for i := range 8 {
		s.Add(string(rune('a'+i)), func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
	}
	s.Priority("h", "g", "f", "e", "d", "c", "b", "a")

	h := s.Then(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	r := httptest.NewRequest("GET", "/things/7", nil)
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(w, r)
	}
}

// benchRequestID is the one middleware that does work on every request whether
// anything reads the result or not.
//
// What is in the loop is a ULID, which is a clock read and ten bytes out of
// crypto/rand, then the context node and the request copy that carry the id
// inward, then the response header. It has a row of its own because it is the
// floor under any chain that logs, and because the id is the one thing here that
// cannot be made lazy: the response header has to go out before the handler
// starts writing.
func benchRequestID(b *testing.B) {
	h := mw.RequestID()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	r := httptest.NewRequest("GET", "/things/7", nil)
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(w, r)
	}
}

// benchLifecycle is the chain a service actually runs, in front of a handler
// that does nothing.
//
// Six middleware, a real access log record written to io.Discard through a JSON
// handler, and web.H at the bottom. Everything in it is work a request pays for
// before the application has been asked anything, so this is the number to
// subtract when a route looks slower than the handler in it.
//
// The log record is the largest part and it is deliberately not stubbed out. A
// service that does not write one is a service nobody can operate, so the
// realistic floor is the one with it in.
func benchLifecycle(b *testing.B) {
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))

	h := web.Chain(web.H(func(*web.Ctx) error { return nil }),
		mw.Recover(log),
		mw.RealIP(mw.Private()...),
		mw.RequestID(),
		mw.Logger(log),
		mw.MaxBody(1<<20),
		mw.Timeout(10*time.Second),
	)

	r := httptest.NewRequest("GET", "/things/7", nil)
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(w, r)
	}
}

// benchCompress is gzip over an eight kilobyte page.
//
// This is the one middleware whose cost is the work rather than the plumbing,
// and it is CPU that buys bandwidth. The number to compare it against is what
// the same page costs to send: at a megabyte a second of link, which is a phone
// on a bad connection, eight kilobytes is eight milliseconds and this is
// microseconds.
//
// The gzip writer comes from a pool, so what is in the loop is the compression
// itself, the reset and the header work. A run that shows the allocation count
// jumping by a 32KB window is a pool that stopped working.
func benchCompress(b *testing.B) {
	h := mw.Compress()(html())

	r := httptest.NewRequest("GET", "/things/7", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := &discardWriter{header: make(http.Header)}

	b.SetBytes(int64(len(page)))
	b.ReportAllocs()
	for b.Loop() {
		// Vary is added rather than set, so the header has to go back to empty
		// or the slice behind it grows for the length of the run.
		clear(w.header)
		h.ServeHTTP(w, r)
	}
}

// benchETag is the request the middleware exists for: a client that already has
// the page, and a 304 that costs no body.
//
// The work is holding eight kilobytes, one SHA-256 pass over them and the
// comparison, and it is spent to send nothing at all. The saving is the whole
// response, so this is worth it at any number that is not absurd, which is why
// the budget is generous and what it watches for is the shape changing rather
// than the number moving.
func benchETag(b *testing.B) {
	h := mw.ETag()(html())

	// One request to learn the tag, which is what a client would have kept.
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest("GET", "/things/7", nil))

	r := httptest.NewRequest("GET", "/things/7", nil)
	r.Header.Set("If-None-Match", first.Header().Get("ETag"))
	w := &discardWriter{header: make(http.Header)}

	b.SetBytes(int64(len(page)))
	b.ReportAllocs()
	for b.Loop() {
		clear(w.header)
		h.ServeHTTP(w, r)
	}
}

// benchCtxAcquire is what the web package adds to a request and nothing else.
//
// The handler does no work, the writer is a recorder that is reused, and the
// request is built once, so what is left in the loop is web.H: take a Ctx, fill
// it in, put the Ctx in the request context, run the handler, put the Ctx back.
// Every request an application serves pays this, which is why it has a row.
//
// The two allocations are the request context. Putting the Ctx where
// web.FromContext can find it means a new context node and the shallow copy of
// the request that carries it, and net/http has no way to change a request's
// context without one. The Ctx itself allocates nothing, which is what the pool
// is for.
func benchCtxAcquire(b *testing.B) {
	h := web.H(func(c *web.Ctx) error { return nil })
	r := httptest.NewRequest("GET", "/things/7", nil)
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(w, r)
	}
}

// listing is the struct the binding benchmarks fill: twelve fields, in the shape
// a search page sends, with a slice and a date in it because both cost more than
// a string does.
//
// The tags are the names the fields would get anyway, since a field with no tag
// is read under its own name in snake case. They are written out so the JSON
// benchmark and the form benchmark are filling the same struct from the same
// names, rather than one of them leaning on a decoder that matches names
// loosely.
type listing struct {
	Q        string    `json:"q"`
	Tags     []string  `json:"tags"`
	Page     int       `json:"page"`
	PerPage  int       `json:"per_page"`
	Sort     string    `json:"sort"`
	Order    string    `json:"order"`
	MinPrice float64   `json:"min_price"`
	MaxPrice float64   `json:"max_price"`
	InStock  bool      `json:"in_stock"`
	Since    time.Time `json:"since"`
	Kind     string    `json:"kind"`
	Cursor   string    `json:"cursor"`
}

// listingForm is one request's worth of listing, as a browser would post it.
const listingForm = "q=water&tags=go&tags=web&page=2&per_page=25&sort=name&" +
	"order=asc&min_price=1.5&max_price=99&in_stock=on&since=2026-01-02&" +
	"kind=all&cursor=eyJpZCI6MTAwfQ"

// listingJSON is the same request as an API client would send it.
const listingJSON = `{"q":"water","tags":["go","web"],"page":2,"per_page":25,` +
	`"sort":"name","order":"asc","min_price":1.5,"max_price":99,` +
	`"in_stock":true,"since":"2026-01-02T00:00:00Z","kind":"all",` +
	`"cursor":"eyJpZCI6MTAwfQ"}`

// benchBindForm is what reading a request into a struct costs.
//
// The parse is inside the measurement, because that is what a handler pays and
// because most of it is the parse. ParseForm on its own is about forty of the
// allocations: a url.Values is a map with a slice in every entry and a string in
// every slice. Binding then walks a plan it built the first time it saw the type
// and puts each string where it goes, which is the rest.
//
// That split is what the generator is for. A generated binder reads the values
// without building the map, and the budget has a row waiting for it.
//
// The form is posted rather than put in the query string so the body decoding
// path is the one being measured. Both forms are cleared each time around, since
// ParseForm keeps what it read on the request and a benchmark that let it would
// be measuring the second half only.
func benchBindForm(b *testing.B) {
	h := web.H(func(c *web.Ctx) error {
		_, err := web.Bind[listing](c)
		return err
	})

	body := &replay{s: listingForm}
	r := httptest.NewRequest("POST", "/things", nil)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Body = body
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		body.i = 0
		r.Form, r.PostForm = nil, nil
		h.ServeHTTP(w, r)
	}
}

// benchBindJSON is the same struct filled from a JSON body instead of a form.
//
// It is the cheaper of the two, which is worth knowing before anybody reaches for
// a form for an API. The decoder reads the bytes once and writes the fields as it
// goes, where a form is parsed into a map of slices of strings first and then
// read out of it.
//
// The query string is empty, so nothing here parses a form. Binding a struct with
// no query fields in it would skip that anyway, but a struct like this one has
// them and the point is to measure the body.
func benchBindJSON(b *testing.B) {
	h := web.H(func(c *web.Ctx) error {
		_, err := web.Bind[listing](c)
		return err
	})

	body := &replay{s: listingJSON}
	r := httptest.NewRequest("POST", "/things", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Body = body
	w := &discardWriter{header: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		body.i = 0
		r.Form, r.PostForm = nil, nil
		h.ServeHTTP(w, r)
	}
}

// replay is a request body that hands out the same bytes every time, so the
// loop can start a request over without allocating a reader for it.
type replay struct {
	s string
	i int
}

func (b *replay) Read(p []byte) (int, error) {
	if b.i >= len(b.s) {
		return 0, io.EOF
	}
	n := copy(p, b.s[b.i:])
	b.i += n
	return n, nil
}

func (b *replay) Close() error { return nil }

// discardWriter is a ResponseWriter that keeps nothing, so the benchmark
// measures the Ctx rather than a recorder's buffer.
type discardWriter struct{ header http.Header }

func (d *discardWriter) Header() http.Header         { return d.header }
func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }
func (d *discardWriter) WriteHeader(int)             {}
