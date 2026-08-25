package mizutest

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

// serve builds a fixture whose every request goes to one handler, which is what
// a test about an assertion wants: one response, and nothing else in the way.
func serve(t *testing.T, h func(*mizu.Ctx) error) *App {
	t.Helper()

	app := NewApp(t)
	app.Routes().Handle("GET", "/", h)
	return app
}

// serveFake is serve over a recorder, for the tests about what a failure says.
func serveFake(t *testing.T, h func(*mizu.Ctx) error) (*App, *recorder) {
	t.Helper()

	app, r := fake(t)
	app.Routes().Handle("GET", "/", h)
	return app, r
}

func status(code int) func(*mizu.Ctx) error {
	return func(c *mizu.Ctx) error { return c.Text(code, http.StatusText(code)) }
}

func body(s string) func(*mizu.Ctx) error {
	return func(c *mizu.Ctx) error { return c.Text(http.StatusOK, s) }
}

func document(s string) func(*mizu.Ctx) error {
	return func(c *mizu.Ctx) error {
		return c.Bytes(http.StatusOK, []byte(s), "application/json")
	}
}

func TestTheStatusAssertions(t *testing.T) {
	tests := []struct {
		code   int
		assert func(*Response) *Response
	}{
		{http.StatusOK, (*Response).AssertOK},
		{http.StatusCreated, (*Response).AssertCreated},
		{http.StatusUnauthorized, (*Response).AssertUnauthorized},
		{http.StatusForbidden, (*Response).AssertForbidden},
		{http.StatusNotFound, (*Response).AssertNotFound},
		{http.StatusUnprocessableEntity, (*Response).AssertUnprocessable},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			app := serve(t, status(tt.code))
			tt.assert(app.Get("/").Do())

			app2, r := serveFake(t, status(http.StatusTeapot))
			tt.assert(app2.Get("/").Do())
			r.says(t, "the status is 418")
		})
	}
}

// TestAssertStatusShowsTheExchange is the failure the package exists to
// improve. The status alone says the test failed and the body says why.
func TestAssertStatusShowsTheExchange(t *testing.T) {
	app, r := serveFake(t, func(*mizu.Ctx) error { return errBoom })

	app.Get("/").Do().AssertOK()

	r.says(t,
		"the status is 500 Internal Server Error, want 200 OK",
		"> GET /",
		"< 500 Internal Server Error",
		"log ERROR",
		"error=boom",
	)
}

// TestOnlyTheFirstFailureShowsTheExchange keeps three failed assertions on one
// response from printing three copies of the same two kilobytes.
func TestOnlyTheFirstFailureShowsTheExchange(t *testing.T) {
	app, r := serveFake(t, status(http.StatusTeapot))

	res := app.Get("/").Do()
	res.AssertOK()
	res.AssertCreated()
	res.AssertNotFound()

	if n := len(r.failures); n != 3 {
		t.Fatalf("%d failures were reported, want 3", n)
	}
	if !strings.Contains(r.failures[0], "> GET /") {
		t.Errorf("the first failure does not show the exchange:\n%s", r.failures[0])
	}
	for _, later := range r.failures[1:] {
		if strings.Contains(later, "> GET /") {
			t.Errorf("a later failure repeats the exchange:\n%s", later)
		}
	}
}

func TestAssertNoContent(t *testing.T) {
	app := serve(t, func(c *mizu.Ctx) error { return c.NoContent() })
	app.Get("/").Do().AssertNoContent()

	app2, r := serveFake(t, func(c *mizu.Ctx) error {
		return c.Bytes(http.StatusNoContent, []byte("something"), "text/plain")
	})
	app2.Get("/").Do().AssertNoContent()
	r.says(t, "the status is 204 and the body has 9 bytes in it")
}

func TestAssertRedirect(t *testing.T) {
	app := serve(t, func(c *mizu.Ctx) error {
		return c.Redirect(http.StatusFound, "/elsewhere")
	})
	app.Get("/").Do().AssertRedirect("/elsewhere")

	app2, r2 := serveFake(t, func(c *mizu.Ctx) error {
		return c.Redirect(http.StatusFound, "/elsewhere")
	})
	app2.Get("/").Do().AssertRedirect("/somewhere")
	r2.says(t, `the redirect goes to "/elsewhere", want "/somewhere"`)

	app3, r3 := serveFake(t, status(http.StatusOK))
	app3.Get("/").Do().AssertRedirect("/elsewhere")
	r3.says(t, "the status is 200 OK, want a redirect")
}

func TestTheHeaderAssertions(t *testing.T) {
	app := serve(t, func(c *mizu.Ctx) error {
		c.Header().Set("X-One", "first")
		c.Header().Add("X-Many", "a")
		c.Header().Add("X-Many", "b")
		return c.Text(http.StatusOK, "ok")
	})

	res := app.Get("/").Do()
	res.AssertHeader("X-One", "first")
	res.AssertHeader("x-one", "first") // the name is not case sensitive
	res.AssertHeaderMissing("X-Nothing")
}

func TestTheHeaderAssertionsSayWhatWentWrong(t *testing.T) {
	tests := map[string]struct {
		assert func(*Response)
		want   []string
	}{
		"a header that is not there": {
			assert: func(r *Response) { r.AssertHeader("X-Nothing", "x") },
			want:   []string{"there is no X-Nothing header", "X-One"},
		},
		"a header with the wrong value": {
			assert: func(r *Response) { r.AssertHeader("X-One", "second") },
			want:   []string{`X-One is "first", want "second"`},
		},
		"a header sent more than once": {
			assert: func(r *Response) { r.AssertHeader("X-Many", "a") },
			want:   []string{"X-Many is sent 2 times"},
		},
		"a header that should not be there": {
			assert: func(r *Response) { r.AssertHeaderMissing("X-One") },
			want:   []string{`X-One is set to ["first"]`},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			app, r := serveFake(t, func(c *mizu.Ctx) error {
				c.Header().Set("X-One", "first")
				c.Header().Add("X-Many", "a")
				c.Header().Add("X-Many", "b")
				return c.Text(http.StatusOK, "ok")
			})
			tt.assert(app.Get("/").Do())
			r.says(t, tt.want...)
		})
	}
}

func TestTheCookieAssertions(t *testing.T) {
	app := serve(t, func(c *mizu.Ctx) error {
		c.SetCookie(&http.Cookie{Name: "session", Value: "abc"})
		c.SetCookie(&http.Cookie{Name: "old", Value: "", MaxAge: -1})
		c.SetCookie(&http.Cookie{Name: "stale", Expires: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
		return c.Text(http.StatusOK, "ok")
	})

	res := app.Get("/").Do()
	res.AssertCookie("session", "abc")
	res.AssertCookieExpired("old")
	res.AssertCookieExpired("stale")

	if got := res.Cookie("nothing"); got != nil {
		t.Errorf("Cookie of a name that was not set gave %v, want nil", got)
	}
}

func TestTheCookieAssertionsSayWhatWentWrong(t *testing.T) {
	tests := map[string]struct {
		assert func(*Response)
		want   []string
	}{
		"a cookie that was not set": {
			assert: func(r *Response) { r.AssertCookie("nothing", "x") },
			want:   []string{`no cookie named "nothing" was set`, "session"},
		},
		"a cookie with the wrong value": {
			assert: func(r *Response) { r.AssertCookie("session", "xyz") },
			want:   []string{`the cookie "session" is "abc", want "xyz"`},
		},
		"a cookie that is not expired": {
			assert: func(r *Response) { r.AssertCookieExpired("session") },
			want:   []string{`the cookie "session" is not expired`},
		},
		"expiring a cookie that was not set": {
			assert: func(r *Response) { r.AssertCookieExpired("nothing") },
			want:   []string{`no cookie named "nothing" was set`},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			app, r := serveFake(t, func(c *mizu.Ctx) error {
				c.SetCookie(&http.Cookie{Name: "session", Value: "abc"})
				return c.Text(http.StatusOK, "ok")
			})
			tt.assert(app.Get("/").Do())
			r.says(t, tt.want...)
		})
	}
}

// TestAssertCookieExpiredUsesTheFixtureClock is why the assertion asks the
// fixture rather than time.Now. A handler that set an expiry from the same
// clock has to be judged against it, or a test that advances time gets an
// answer about the wrong day.
func TestAssertCookieExpiredUsesTheFixtureClock(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	app := NewApp(t, At(at))
	app.Routes().Get("/", func(c *mizu.Ctx) error {
		c.SetCookie(&http.Cookie{Name: "session", Value: "abc", Expires: at.Add(time.Hour)})
		return c.Text(http.StatusOK, "ok")
	})

	res := app.Get("/").Do()
	app.Clock().Advance(2 * time.Hour)

	res.AssertCookieExpired("session")
}

func TestTheBodyAssertions(t *testing.T) {
	app := serve(t, body("Alpha Beta Gamma"))

	res := app.Get("/").Do()
	res.AssertSee("Beta")
	res.AssertDontSee("Delta")
	res.AssertSeeInOrder("Alpha", "Beta", "Gamma")

	if got := res.Text(); got != "Alpha Beta Gamma" {
		t.Errorf("Text gave %q", got)
	}
	if got := len(res.Body()); got != 16 {
		t.Errorf("Body gave %d bytes, want 16", got)
	}
}

func TestTheBodyAssertionsSayWhatWentWrong(t *testing.T) {
	tests := map[string]struct {
		assert func(*Response)
		want   []string
	}{
		"something missing": {
			assert: func(r *Response) { r.AssertSee("Delta") },
			want:   []string{`the body does not contain "Delta"`},
		},
		"something present": {
			assert: func(r *Response) { r.AssertDontSee("Beta") },
			want:   []string{`the body contains "Beta" at byte 6`},
		},
		"out of order": {
			assert: func(r *Response) { r.AssertSeeInOrder("Gamma", "Beta") },
			want:   []string{`the body has "Beta" before "Gamma", want it after`},
		},
		"in order but missing": {
			assert: func(r *Response) { r.AssertSeeInOrder("Alpha", "Delta") },
			want:   []string{`the body does not contain "Delta"`},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			app, r := serveFake(t, body("Alpha Beta Gamma"))
			tt.assert(app.Get("/").Do())
			r.says(t, tt.want...)
		})
	}
}

func TestAssertJSON(t *testing.T) {
	app := serve(t, document(`{"id":1,"title":"Hello","tags":["go","web"]}`))

	res := app.Get("/").Do()
	res.AssertJSON(map[string]any{
		"id": 1, "title": "Hello", "tags": []string{"go", "web"},
	})
	res.AssertJSON(json.RawMessage(`{"tags":["go","web"],"title":"Hello","id":1}`))
}

func TestAssertJSONSubset(t *testing.T) {
	app := serve(t, document(`{"id":1,"title":"Hello","tags":["go","web"]}`))

	res := app.Get("/").Do()
	res.AssertJSONSubset(map[string]any{"title": "Hello"})
	res.AssertJSONSubset(map[string]any{"tags": []string{"go", "web"}})
}

func TestAssertJSONSubsetStillSeesADifference(t *testing.T) {
	tests := map[string]any{
		"a member that is not there":  map[string]any{"missing": 1},
		"a member with another value": map[string]any{"title": "Goodbye"},
		"an array of the wrong size":  map[string]any{"tags": []string{"go"}},
	}
	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			app, r := serveFake(t, document(`{"id":1,"title":"Hello","tags":["go","web"]}`))
			app.Get("/").Do().AssertJSONSubset(want)
			r.says(t, "does not contain the document expected")
		})
	}
}

func TestAssertJSONPath(t *testing.T) {
	app := serve(t, document(`{
		"data": [
			{"id": 1234567890123456789, "title": "Hello", "tags": ["go"]},
			{"id": 2, "title": "World", "tags": []}
		],
		"meta": {"total": 2, "next": null, "ratio": 1.5}
	}`))

	res := app.Get("/").Do()
	res.AssertJSONPath("$.data[0].title", "Hello")
	res.AssertJSONPath("$.data[1].title", "World")
	res.AssertJSONPath("$.data[-1].title", "World")
	res.AssertJSONPath("$.data[0].tags[0]", "go")
	res.AssertJSONPath("$.meta.total", 2)
	res.AssertJSONPath("$.meta.ratio", 1.5)
	res.AssertJSONPath("$.meta.next", nil)
	res.AssertJSONMissing("$.meta.missing")

	// An id past what a float64 holds exactly, which is the reason the decoder
	// keeps numbers as text.
	res.AssertJSONPath("$.data[0].id", int64(1234567890123456789))

	res.AssertJSONCount("$.data", 2)
	res.AssertJSONCount("$.data[0]", 3)
	res.AssertJSONCount("$.data[1].tags", 0)
}

func TestAssertJSONPathSaysWhatWentWrong(t *testing.T) {
	const doc = `{"data":[{"id":1,"title":"Hello"}],"meta":{"total":1}}`

	tests := map[string]struct {
		assert func(*Response)
		want   []string
	}{
		"the wrong value": {
			assert: func(r *Response) { r.AssertJSONPath("$.data[0].title", "Goodbye") },
			want:   []string{`$.data[0].title is "Hello", want "Goodbye"`},
		},
		"a member that is not there": {
			assert: func(r *Response) { r.AssertJSONPath("$.data[0].author", "ana") },
			want:   []string{`$.data[0] has no member "author"`, "members id, title"},
		},
		"an index past the end": {
			assert: func(r *Response) { r.AssertJSONPath("$.data[3].id", 1) },
			want:   []string{"$.data has 1 elements"},
		},
		"an index into an object": {
			assert: func(r *Response) { r.AssertJSONPath("$.meta[0]", 1) },
			want:   []string{"$.meta is an object", "not an array"},
		},
		"a member of a number": {
			assert: func(r *Response) { r.AssertJSONPath("$.meta.total.x", 1) },
			want:   []string{"$.meta.total is the number 1", "not an object"},
		},
		"something that should not be there": {
			assert: func(r *Response) { r.AssertJSONMissing("$.meta.total") },
			want:   []string{"$.meta.total is 1, want it absent"},
		},
		"counting something with no count": {
			assert: func(r *Response) { r.AssertJSONCount("$.meta.total", 1) },
			want:   []string{"which has nothing to count"},
		},
		"the wrong count": {
			assert: func(r *Response) { r.AssertJSONCount("$.data", 3) },
			want:   []string{"$.data has 1 elements, want 3"},
		},
		"the wrong member count": {
			assert: func(r *Response) { r.AssertJSONCount("$.meta", 3) },
			want:   []string{"$.meta has 1 members, want 3"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			app, r := serveFake(t, document(doc))
			tt.assert(app.Get("/").Do())
			r.says(t, tt.want...)
		})
	}
}

// TestAssertJSONPathShowsTheDocument is the difference between a failure a
// reader can act on and one that sends them looking for the response.
func TestAssertJSONPathShowsTheDocument(t *testing.T) {
	app, r := serveFake(t, document(`{"data":{"title":"Hello"}}`))

	app.Get("/").Do().AssertJSONPath("$.data.author", "ana")

	r.says(t, "the body is", `{"data":{"title":"Hello"}}`)
}

// TestALongDocumentIsIndented, because a document that does not fit on a line
// is unreadable as one.
func TestALongDocumentIsIndented(t *testing.T) {
	app, r := serveFake(t, document(
		`{"data":{"title":"Hello there, this is a title long enough to need more than one line"}}`))

	app.Get("/").Do().AssertJSONPath("$.data.author", "ana")

	r.says(t, "\n  \"data\": {\n")
}

func TestJSONAssertionsSayWhenTheBodyIsNotJSON(t *testing.T) {
	app, r := serveFake(t, body("<html>not json</html>"))

	res := app.Get("/").Do()
	res.AssertJSONPath("$.title", "Hello")
	res.AssertJSON(map[string]any{})

	r.says(t, "the body is not JSON")

	// Once, rather than once per assertion. The body will not be JSON on the
	// second try either, and a test with four path assertions in it should not
	// report the same thing four times.
	if n := len(r.failures); n != 1 {
		t.Errorf("%d failures were reported, want 1", n)
	}
}

func TestDecodeReadsTheBodyIntoAValue(t *testing.T) {
	app := serve(t, document(`{"id":7,"title":"Hello"}`))

	var got struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	app.Get("/").Do().Decode(&got)

	if got.ID != 7 || got.Title != "Hello" {
		t.Errorf("Decode gave %+v", got)
	}
}

func TestDecodeSaysWhenItDoesNotFit(t *testing.T) {
	app, r := serveFake(t, document(`{"id":"seven"}`))

	var got struct {
		ID int `json:"id"`
	}
	app.Get("/").Do().Decode(&got)

	r.says(t, "the body is not JSON that fits")
}

func TestAssertionsChain(t *testing.T) {
	app := serve(t, document(`{"id":1}`))

	// Every assertion hands the response back, which is the only thing this
	// checks and the reason a test reads as one statement.
	app.Get("/").Do().
		AssertOK().
		AssertHeader("Content-Type", "application/json").
		AssertJSONPath("$.id", 1).
		AssertJSONCount("$", 1).
		AssertSee("id")
}

func TestTheJSONAssertionsReportAnExpectedValueTheyCannotEncode(t *testing.T) {
	tests := map[string]func(*Response, any) *Response{
		"AssertJSON":       (*Response).AssertJSON,
		"AssertJSONSubset": (*Response).AssertJSONSubset,
		"AssertJSONPath": func(r *Response, want any) *Response {
			return r.AssertJSONPath("$.id", want)
		},
	}
	for name, assert := range tests {
		t.Run(name, func(t *testing.T) {
			app, r := serveFake(t, document(`{"id":1}`))

			assert(app.Get("/").Do(), make(chan int))

			r.says(t, "the expected value will not encode as JSON", "chan int")
		})
	}
}

func TestAssertJSONMissingSaysWhatIsThere(t *testing.T) {
	app, r := serveFake(t, document(`{"password":"hunter2"}`))

	app.Get("/").Do().AssertJSONMissing("$.password")

	r.says(t, `$.password is "hunter2"`, "want it absent")
}

// TestAssertJSONCountNeedsSomethingToCount is the case where the path is right
// and the value is not the kind of thing a count means anything about.
func TestAssertJSONCountNeedsSomethingToCount(t *testing.T) {
	app, r := serveFake(t, document(`{"data":{"total":3},"title":"Hello"}`))
	res := app.Get("/").Do()

	res.AssertJSONCount("$.title", 5)
	r.says(t, "$.title", "the string \"Hello\"", "nothing to count")

	// An object counts its members, so this one is a real count and passes.
	res.AssertJSONCount("$.data", 1)
}

// TestAJSONAssertionStopsAtABadPath keeps one mistake from turning into two
// failures, the second about a value that was never read.
func TestAJSONAssertionStopsAtABadPath(t *testing.T) {
	app, r := serveFake(t, document(`{"id":1}`))
	res := app.Get("/").Do()

	res.AssertJSONPath("$.nope", 1)
	res.AssertJSONCount("$.nope", 1)

	if got := len(r.failures); got != 2 {
		t.Errorf("two bad paths gave %d failures, want one each:\n%s", got, strings.Join(r.failures, "\n"))
	}
}

func TestHandlerIsTheThingRequestsGoTo(t *testing.T) {
	app := NewApp(t)
	if app.Handler() == nil {
		t.Fatal("a fixture over an application has no handler")
	}

	// Wrapping it is the point: a test can put middleware in front of the
	// application for one test without touching the application.
	var seen bool
	wrapped := NewApp(t, NoParallel(), Serve(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		seen = true
		app.Handler().ServeHTTP(w, req)
	})))
	app.Routes().Get("/", body("ok"))

	wrapped.Get("/").Do().AssertOK().AssertSee("ok")
	if !seen {
		t.Error("the wrapper was not called")
	}
}

func TestTheHeaderAssertionSaysWhenNothingWasSent(t *testing.T) {
	app, r := serveFake(t, func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	res := app.Get("/").Do()
	res.res.Header = http.Header{} // a response that set nothing at all

	res.AssertHeader("X-Request-Id", "req-1")
	r.says(t, "no headers were sent")
}

func TestTheCookieAssertionSaysWhenNoneWereSet(t *testing.T) {
	app, r := serveFake(t, body("ok"))

	app.Get("/").Do().AssertCookie("session", "abc")

	r.says(t, "none")
}
