package mw_test

import (
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func ExampleCORS() {
	h := mw.CORS(mw.CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST", "DELETE"},
		AllowedHeaders:   []string{"Authorization"},
		ExposedHeaders:   []string{mw.RequestIDHeader},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	})(web.H(func(c *web.Ctx) error {
		return c.Text("the posts")
	}))

	// The preflight a browser sends before a cross origin DELETE that carries an
	// Authorization header. It is answered here and the handler never sees it.
	pre := httptest.NewRequest("OPTIONS", "/posts/7", nil)
	pre.Header.Set("Origin", "https://app.example.com")
	pre.Header.Set("Access-Control-Request-Method", "DELETE")
	pre.Header.Set("Access-Control-Request-Headers", "Authorization")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, pre)
	fmt.Println(w.Code, w.Header().Get("Access-Control-Allow-Methods"), w.Header().Get("Access-Control-Max-Age"))

	// The request itself, which the handler serves.
	get := httptest.NewRequest("GET", "/posts", nil)
	get.Header.Set("Origin", "https://app.example.com")

	w = httptest.NewRecorder()
	h.ServeHTTP(w, get)
	fmt.Println(w.Code, w.Body, w.Header().Get("Access-Control-Allow-Origin"))
	// Output:
	// 204 GET, POST, DELETE 86400
	// 200 the posts https://app.example.com
}

func ExampleSecure() {
	h := mw.Secure(mw.SecureConfig{
		HSTS:               365 * 24 * time.Hour,
		HSTSSubdomains:     true,
		FrameOptions:       "DENY",
		ContentTypeOptions: true,
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		CSP:                "default-src 'self'",
	})(web.H(func(c *web.Ctx) error {
		return c.Text("a page")
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	for _, name := range []string{
		"Strict-Transport-Security",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Content-Security-Policy",
	} {
		fmt.Println(name+":", w.Header().Get(name))
	}
	// Output:
	// Strict-Transport-Security: max-age=31536000; includeSubDomains
	// X-Frame-Options: DENY
	// X-Content-Type-Options: nosniff
	// Referrer-Policy: strict-origin-when-cross-origin
	// Content-Security-Policy: default-src 'self'
}

func ExampleMethodOverride() {
	routes := router.New()
	routes.Handle("DELETE /posts/{id:int}", web.H(func(c *web.Ctx) error {
		return c.Text("deleted post " + c.Param("id"))
	}))

	// What the browser sends for a form whose method is post and whose hidden
	// _method field says delete.
	body := url.Values{"_method": {"delete"}}.Encode()
	r := httptest.NewRequest("POST", "https://example.com/posts/7", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://example.com")

	w := httptest.NewRecorder()
	mw.MethodOverride()(routes).ServeHTTP(w, r)

	fmt.Println(w.Code, w.Body)
	// Output:
	// 200 deleted post 7
}

func ExampleCompress() {
	page := strings.Repeat("mizu is water and water is mizu. ", 100)

	h := mw.Compress()(web.H(func(c *web.Ctx) error {
		return c.Text(page)
	}))

	r := httptest.NewRequest("GET", "/posts", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	sent := w.Body.Len()
	gz, err := gzip.NewReader(w.Body)
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		panic(err)
	}

	fmt.Println("Content-Encoding:", w.Header().Get("Content-Encoding"))
	fmt.Println("Vary:", w.Header().Get("Vary"))
	fmt.Println("smaller:", sent < len(page))
	fmt.Println("same page:", string(body) == page)
	// Output:
	// Content-Encoding: gzip
	// Vary: Accept-Encoding
	// smaller: true
	// same page: true
}

func ExampleETag() {
	// Compress goes outside ETag, so the tag is for the page rather than for
	// the compression of it, and a client that takes gzip and one that does not
	// hold the same validator.
	h := web.Chain(web.H(func(c *web.Ctx) error {
		return c.Text("the front page")
	}), mw.Compress(), mw.ETag())

	// The first request builds the page and comes back with a validator.
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest("GET", "/", nil))
	fmt.Println(first.Code, first.Body.Len(), "bytes")

	// The client sends the validator back and the body stays home.
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("If-None-Match", first.Header().Get("ETag"))
	second := httptest.NewRecorder()
	h.ServeHTTP(second, r)
	fmt.Println(second.Code, second.Body.Len(), "bytes")

	fmt.Println("same tag:", second.Header().Get("ETag") == first.Header().Get("ETag"))
	// Output:
	// 200 14 bytes
	// 304 0 bytes
	// same tag: true
}
