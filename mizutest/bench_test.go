package mizutest

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/go-mizu/mizu"
)

// None of this is on anyone's hot path. A test suite makes thousands of
// requests, not millions, and the handler under test is nearly always the
// expensive part. The numbers are here to say what the fixture itself adds, so
// that a suite which has become slow can be blamed on the right thing.

// BenchmarkNewApp measures building a fixture. It runs over a recorder rather
// than over b, because NewApp registers a cleanup and holds a context until the
// test ends, and a million of those outlive the loop that made them. The
// recorder costs an allocation of its own, which is in the number below.
func BenchmarkNewApp(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		r := &recorder{name: "bench"}
		_ = NewApp(r)
		for i := len(r.cleanups) - 1; i >= 0; i-- {
			r.cleanups[i]()
		}
	}
}

// BenchmarkRequest is the fixture's own cost: build a request, serve it, record
// what came back. The handler is a plain one so that nothing in the framework
// is being measured here.
func BenchmarkRequest(b *testing.B) {
	app := NewApp(b, Serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":1,"title":"Hello"}`))
	})))

	b.ReportAllocs()
	for b.Loop() {
		app.Get("/posts/1").Do()
	}
}

// BenchmarkRequestThroughTheRouter is the same thing with a mizu application
// behind it, which is what a real test has. The difference between this and
// [BenchmarkRequest] is the router, the middleware and the encoding.
func BenchmarkRequestThroughTheRouter(b *testing.B) {
	app := NewApp(b)
	app.Routes().Get("/posts/{id}", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]any{"id": c.Param("id"), "title": "Hello"})
	})

	// Every request logs a line, and the log is kept rather than printed, so
	// without this the fixture holds every entry the benchmark ever made. The
	// reset is far enough apart that it is not what is being measured.
	var n int
	b.ReportAllocs()
	for b.Loop() {
		app.Get("/posts/1").Do()
		if n++; n%1024 == 0 {
			app.Log().Reset()
		}
	}
}

func BenchmarkRequestWithAJSONBody(b *testing.B) {
	app := NewApp(b, Serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})))
	post := map[string]any{"title": "Hello", "body": "the body of the post", "tags": []string{"go", "web"}}

	b.ReportAllocs()
	for b.Loop() {
		app.Post("/posts").JSON(post).Do()
	}
}

// BenchmarkAssertions run against one response, which is what a test does: make
// a request and then ask several things about the answer.
func BenchmarkAssertStatus(b *testing.B) {
	res := benchResponse(b, `{"id":1}`)

	b.ReportAllocs()
	for b.Loop() {
		res.AssertOK()
	}
}

func BenchmarkAssertSee(b *testing.B) {
	res := benchResponse(b, benchDocument)

	b.ReportAllocs()
	for b.Loop() {
		res.AssertSee("Third")
	}
}

// BenchmarkAssertJSON pays for the expected value each time and for the body
// once, since a response decodes itself on the first JSON assertion and keeps
// the result for the rest.
func BenchmarkAssertJSON(b *testing.B) {
	res := benchResponse(b, benchDocument)
	want := json.RawMessage(benchDocument)

	b.ReportAllocs()
	for b.Loop() {
		res.AssertJSON(want)
	}
}

func BenchmarkAssertJSONSubset(b *testing.B) {
	res := benchResponse(b, benchDocument)
	want := map[string]any{"meta": map[string]any{"total": 3}}

	b.ReportAllocs()
	for b.Loop() {
		res.AssertJSONSubset(want)
	}
}

func BenchmarkAssertJSONPath(b *testing.B) {
	res := benchResponse(b, benchDocument)

	b.ReportAllocs()
	for b.Loop() {
		res.AssertJSONPath("$.data[2].title", "Third")
	}
}

// BenchmarkDecodeJSON is the part of a JSON assertion that a response pays once
// however many things are asked of it.
func BenchmarkDecodeJSON(b *testing.B) {
	body := []byte(benchDocument)

	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		if _, err := decodeJSON(body); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePath(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := parsePath("$.data[2].tags[-1]"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLogHandler is what every line a handler logs costs. It is the one
// thing here that runs as often as the code under test does.
func BenchmarkLogHandler(b *testing.B) {
	var log Log
	lg := slog.New(log.Handler()).With("service", "api")

	var n int
	b.ReportAllocs()
	for b.Loop() {
		lg.Info("handled", "route", "posts.show", "ms", 12)
		if n++; n%1024 == 0 {
			log.Reset()
		}
	}
}

// BenchmarkExchange is the failure output, which only runs when a test is
// already failing and so is here to be watched rather than tuned.
func BenchmarkExchange(b *testing.B) {
	res := benchResponse(b, benchDocument)

	b.ReportAllocs()
	for b.Loop() {
		res.exchange()
	}
}

// benchDocument is a list response of the size a real endpoint returns, which
// is what the numbers should be read against.
const benchDocument = `{
	"data": [
		{"id": 1, "title": "First", "tags": ["go", "web"], "published": true},
		{"id": 2, "title": "Second", "tags": ["go"], "published": false},
		{"id": 3, "title": "Third", "tags": ["web", "http"], "published": true}
	],
	"meta": {"total": 3, "page": 1, "per_page": 25}
}`

func benchResponse(b *testing.B, body string) *Response {
	b.Helper()

	app := NewApp(b, Serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	})))
	return app.Get("/posts").Do()
}
