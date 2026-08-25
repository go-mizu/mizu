package mw_test

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/clock"
	"github.com/go-mizu/mizu/router"
	"github.com/go-mizu/mizu/web"
	"github.com/go-mizu/mizu/web/mw"
)

// printer writes records to standard output with the timestamp left off, since
// an example asserts on what it printed.
func printer() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

// stopped is a request whose clock does not move, so that the duration in a
// record is the same on every run.
func stopped(r *http.Request) *http.Request {
	return r.WithContext(clock.With(r.Context(), clock.Fake(time.Unix(0, 0).UTC())))
}

// The chain most services start with, outermost first.
func Example() {
	log := printer()

	routes := router.New()
	routes.Handle("GET /posts/{id:int}", web.H(func(c *web.Ctx) error {
		return c.Text("post " + c.Param("id"))
	}))

	srv := web.Chain(routes,
		mw.Recover(log),
		mw.RealIP(mw.Private()...),
		mw.RequestIDFrom(mw.RequestIDHeader),
		mw.Logger(log),
		mw.MaxBody(1<<20),
		mw.Timeout(10*time.Second),
	)

	r := stopped(httptest.NewRequest("GET", "/posts/7", nil))
	r.Header.Set(mw.RequestIDHeader, "01ARYZ6S41TSV4RRFFQ69G5FAV")

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	fmt.Println(w.Code, w.Body, w.Header().Get(mw.RequestIDHeader))
	// Output:
	// level=INFO msg=request request_id=01ARYZ6S41TSV4RRFFQ69G5FAV method=GET path=/posts/7 status=200 dur=0s bytes=6 ip=192.0.2.1 ua=""
	// 200 post 7 01ARYZ6S41TSV4RRFFQ69G5FAV
}

func ExampleRequestID() {
	var seen string
	h := mw.RequestID()(web.H(func(c *web.Ctx) error {
		seen = c.RequestID()
		return nil
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	// The id is a different ULID every run, so what is printed is what can be
	// said about it: twenty six characters, and the same on both sides.
	fmt.Println(len(seen), seen == w.Header().Get(mw.RequestIDHeader))
	// Output:
	// 26 true
}

func ExampleRequestIDFrom() {
	h := mw.RequestIDFrom("X-Correlation-Id")(web.H(func(c *web.Ctx) error {
		fmt.Println(c.RequestID())
		return nil
	}))

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Correlation-Id", "checkout-4821")

	h.ServeHTTP(httptest.NewRecorder(), r)
	// Output:
	// checkout-4821
}

func ExampleRealIP() {
	h := mw.RealIP(mw.Private()...)(web.H(func(c *web.Ctx) error {
		fmt.Println(c.IP())
		return nil
	}))

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234" // the proxy, which we run
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.5")

	h.ServeHTTP(httptest.NewRecorder(), r)
	// Output:
	// 203.0.113.7
}

func ExampleLogger() {
	h := mw.Logger(printer())(web.H(func(c *web.Ctx) error {
		return c.Status(http.StatusCreated).Text("made it")
	}))

	r := stopped(httptest.NewRequest("POST", "/orders", strings.NewReader("{}")))
	r.Header.Set("User-Agent", "curl/8.5.0")

	h.ServeHTTP(httptest.NewRecorder(), r)
	// Output:
	// level=INFO msg=request method=POST path=/orders status=201 dur=0s bytes=7 ip=192.0.2.1 ua=curl/8.5.0
}

func ExampleRecover() {
	h := mw.Recover(slog.New(slog.NewTextHandler(io.Discard, nil)))(web.H(func(c *web.Ctx) error {
		panic("the cache client was never wired up")
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	fmt.Println(w.Code, strings.TrimSpace(w.Body.String()))
	// Output:
	// 500 Internal Server Error
}

func ExampleTimeout() {
	h := mw.Timeout(time.Millisecond)(web.H(func(c *web.Ctx) error {
		<-c.Context().Done() // a query, a call out, or anything else that takes its context
		return nil
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	fmt.Println(w.Code, strings.TrimSpace(w.Body.String()))
	// Output:
	// 504 Gateway Timeout
}

func ExampleMaxBody() {
	h := mw.MaxBody(64)(web.H(func(c *web.Ctx) error {
		return c.NoContent()
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/uploads", strings.NewReader(strings.Repeat("a", 65))))

	fmt.Println(w.Code, strings.TrimSpace(w.Body.String()))
	// Output:
	// 413 Request Entity Too Large
}

func ExampleConcurrency() {
	// One slot, and the request in the outer handler is holding it while the
	// inner one arrives.
	var srv http.Handler
	srv = mw.Concurrency(1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/first" {
			return
		}
		second := httptest.NewRecorder()
		srv.ServeHTTP(second, httptest.NewRequest("GET", "/second", nil))
		fmt.Println(second.Code, second.Header().Get("Retry-After"))
	}))

	srv.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/first", nil))
	// Output:
	// 503 1
}
