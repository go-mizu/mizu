package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// caching runs a handler that only touches the headers, for the calls that
// write nothing.
func caching(t *testing.T, fn func(*Ctx)) http.Header {
	t.Helper()

	w := httptest.NewRecorder()
	fn(direct(t, w))
	return w.Header()
}

func TestCacheForSaysHowLong(t *testing.T) {
	cases := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"five minutes", 5 * time.Minute, "public, max-age=300"},
		{"a day", 24 * time.Hour, "public, max-age=86400"},
		{"part of a second rounds down", 1500 * time.Millisecond, "public, max-age=1"},
		{"nothing", 0, "public, max-age=0"},
		{"less than nothing", -time.Hour, "public, max-age=0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := caching(t, func(c *Ctx) { c.CacheFor(tc.d) })

			if got := h.Get("Cache-Control"); got != tc.want {
				t.Errorf("Cache-Control is %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPrivateAndCacheForComposeEitherWay is the part that would be easy to get
// wrong by having each method write the whole header.
func TestPrivateAndCacheForComposeEitherWay(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*Ctx)
		want string
	}{
		{"private first", func(c *Ctx) { c.Private().CacheFor(time.Minute) }, "private, max-age=60"},
		{"private last", func(c *Ctx) { c.CacheFor(time.Minute).Private() }, "max-age=60, private"},
		{"private twice", func(c *Ctx) { c.Private().CacheFor(time.Minute).Private() }, "max-age=60, private"},
		{"nothing but private", func(c *Ctx) { c.Private() }, "private"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := caching(t, tc.fn)

			got := h.Get("Cache-Control")
			if !strings.Contains(got, "private") || strings.Contains(got, "public") {
				t.Errorf("Cache-Control is %q, which is not private", got)
			}
			if got != tc.want {
				t.Errorf("Cache-Control is %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTheLastWordBetweenStoringAndNotWins(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*Ctx)
		want string
	}{
		{"no-store on its own", func(c *Ctx) { c.NoStore() }, "no-store"},
		{"no-store after a duration", func(c *Ctx) { c.CacheFor(time.Hour).NoStore() }, "no-store"},
		{"a duration after no-store", func(c *Ctx) { c.NoStore().CacheFor(time.Hour) }, "public, max-age=3600"},
		{"no-store after private", func(c *Ctx) { c.Private().NoStore() }, "no-store"},
		{"private after no-store", func(c *Ctx) { c.NoStore().Private() }, "private"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := caching(t, tc.fn)

			if got := h.Get("Cache-Control"); got != tc.want {
				t.Errorf("Cache-Control is %q, want %q", got, tc.want)
			}
		})
	}
}

// TestWhatIsAlreadyOnTheHeaderIsKept is why these methods merge rather than
// write. Middleware and a handler can both have an opinion, and only the parts
// that disagree are taken off.
func TestWhatIsAlreadyOnTheHeaderIsKept(t *testing.T) {
	h := caching(t, func(c *Ctx) {
		c.SetHeader("Cache-Control", `immutable, MAX-AGE=5, no-cache="Set-Cookie"`)
		c.CacheFor(time.Hour)
	})

	want := `immutable, no-cache="Set-Cookie", public, max-age=3600`
	if got := h.Get("Cache-Control"); got != want {
		t.Errorf("Cache-Control is %q, want %q", got, want)
	}
}

// TestADurationDropsANoStoreSomebodyElseWrote is the same last-word rule as
// above, for the header a handler did not write itself.
func TestADurationDropsANoStoreSomebodyElseWrote(t *testing.T) {
	h := caching(t, func(c *Ctx) {
		c.SetHeader("Cache-Control", "private, no-store")
		c.CacheFor(time.Hour)
	})

	want := "private, max-age=3600"
	if got := h.Get("Cache-Control"); got != want {
		t.Errorf("Cache-Control is %q, want %q", got, want)
	}
}

func TestETagIsQuotedOnce(t *testing.T) {
	cases := []struct {
		name, tag, want string
	}{
		{"a bare version", "v7", `"v7"`},
		{"one already quoted", `"v7"`, `"v7"`},
		{"a weak one", `W/"v7"`, `W/"v7"`},
		{"a weak one unquoted", "W/v7", `W/"v7"`},
		{"a hash", "d41d8cd98f00b204e9800998ecf8427e", `"d41d8cd98f00b204e9800998ecf8427e"`},
		{"one with a comma in it", "a,b", `"a,b"`},
		{"one that would end early", `v"7`, `"v7"`},
		{"one that would split the header", "v\r\nX-Injected: yes", `"vX-Injected:yes"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := caching(t, func(c *Ctx) { c.ETag(tc.tag) })

			if got := h.Get("ETag"); got != tc.want {
				t.Errorf("ETag is %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAnEmptyTagIsNoTag(t *testing.T) {
	for _, tag := range []string{"", `""`, `W/""`, "\r\n"} {
		h := caching(t, func(c *Ctx) { c.SetHeader("ETag", `"old"`); c.ETag(tag) })

		if got, ok := h["Etag"]; ok {
			t.Errorf("ETag(%q) left %q on the response", tag, got)
		}
	}
}

func TestLastModifiedIsAnHTTPDate(t *testing.T) {
	zone := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	at := time.Date(2026, time.August, 26, 14, 30, 5, 999, zone)

	h := caching(t, func(c *Ctx) { c.LastModified(at) })

	want := "Wed, 26 Aug 2026 07:30:05 GMT"
	if got := h.Get("Last-Modified"); got != want {
		t.Errorf("Last-Modified is %q, want %q", got, want)
	}
}

func TestTheZeroTimeIsNoTime(t *testing.T) {
	h := caching(t, func(c *Ctx) {
		c.LastModified(time.Now())
		c.LastModified(time.Time{})
	})

	if got, ok := h["Last-Modified"]; ok {
		t.Errorf("the zero time left %q on the response", got)
	}
}

// conditional runs a handler that answers a conditional request the way the
// doc comment shows, and reports what went out.
func conditional(t *testing.T, req *http.Request, tag string, at time.Time) *httptest.ResponseRecorder {
	t.Helper()

	return serve(t, req, func(c *Ctx) error {
		c.ETag(tag).LastModified(at).CacheFor(time.Minute)
		c.SetHeader("Content-Type", "application/json")
		if c.NotModified() {
			return nil
		}
		return c.Bytes("application/json", []byte(`{"id":7}`))
	})
}

func asked(headers ...string) *http.Request {
	r := httptest.NewRequest("GET", "/posts/7", nil)
	for i := 0; i < len(headers); i += 2 {
		r.Header.Set(headers[i], headers[i+1])
	}
	return r
}

func TestAMatchingTagIsAnswered(t *testing.T) {
	at := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	w := conditional(t, asked("If-None-Match", `"v7"`), "v7", at)

	if w.Code != http.StatusNotModified {
		t.Fatalf("the status is %d, want 304", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("a 304 carried a body: %q", w.Body)
	}
	for _, gone := range []string{"Content-Type", "Content-Length", "Content-Encoding", "Last-Modified"} {
		if got := w.Header().Get(gone); got != "" {
			t.Errorf("the 304 still carries %s: %q", gone, got)
		}
	}
	if got, want := w.Header().Get("ETag"), `"v7"`; got != want {
		t.Errorf("the 304 has ETag %q, want %q", got, want)
	}
	if got, want := w.Header().Get("Cache-Control"), "public, max-age=60"; got != want {
		t.Errorf("the 304 has Cache-Control %q, want %q", got, want)
	}
}

func TestATagThatDoesNotMatchGetsTheBody(t *testing.T) {
	at := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	w := conditional(t, asked("If-None-Match", `"v6"`), "v7", at)

	if w.Code != http.StatusOK {
		t.Fatalf("the status is %d, want 200", w.Code)
	}
	if got, want := w.Body.String(), `{"id":7}`; got != want {
		t.Errorf("the body is %q, want %q", got, want)
	}
	if got := w.Header().Get("Last-Modified"); got == "" {
		t.Error("a 200 lost its Last-Modified")
	}
}

// TestATagOutranksTheDate is the precedence RFC 9110 section 13.2.2 asks for.
// The date on its own would have answered 304, and it is not read at all
// because the request sent a tag.
func TestATagOutranksTheDate(t *testing.T) {
	at := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	w := conditional(t, asked(
		"If-None-Match", `"v6"`,
		"If-Modified-Since", "Fri, 21 Aug 2026 09:00:00 GMT",
	), "v7", at)

	if w.Code != http.StatusOK {
		t.Fatalf("the status is %d, want 200, since the tag the client sent is not the one it would get", w.Code)
	}
}

func TestTheDateIsReadWhenNoTagWasSent(t *testing.T) {
	at := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		since string
		want  int
	}{
		{"the same second", "Thu, 20 Aug 2026 09:00:00 GMT", http.StatusNotModified},
		{"later", "Fri, 21 Aug 2026 09:00:00 GMT", http.StatusNotModified},
		{"earlier", "Wed, 19 Aug 2026 09:00:00 GMT", http.StatusOK},
		{"not a date", "yesterday", http.StatusOK},
		{"nothing", "", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := conditional(t, asked("If-Modified-Since", tc.since), "", at)

			if w.Code != tc.want {
				t.Errorf("the status is %d, want %d", w.Code, tc.want)
			}
		})
	}
}

func TestADateWithNothingToCompareItToGetsTheBody(t *testing.T) {
	w := conditional(t, asked("If-Modified-Since", "Fri, 21 Aug 2026 09:00:00 GMT"), "", time.Time{})

	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want 200, since the response never said when it changed", w.Code)
	}
}

// TestADateNobodyCanReadGetsTheBody covers a Last-Modified that came from
// somewhere other than [Ctx.LastModified], which is the only way one that is
// not an HTTP date gets onto a response.
func TestADateNobodyCanReadGetsTheBody(t *testing.T) {
	r := asked("If-Modified-Since", "Fri, 21 Aug 2026 09:00:00 GMT")

	w := serve(t, r, func(c *Ctx) error {
		c.SetHeader("Last-Modified", "whenever")
		if c.NotModified() {
			return nil
		}
		return c.Text("sent")
	})

	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want 200", w.Code)
	}
}

func TestTheTagsAreReadTheWayTheHeaderIsWritten(t *testing.T) {
	at := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		match string
		tag   string
		want  int
	}{
		{"one of several", `"v5", "v6", "v7"`, "v7", http.StatusNotModified},
		{"a weak client tag against a strong one", `W/"v7"`, "v7", http.StatusNotModified},
		{"a strong client tag against a weak one", `"v7"`, `W/"v7"`, http.StatusNotModified},
		{"anything at all", "*", "v7", http.StatusNotModified},
		{"anything at all, with no tag to show", "*", "", http.StatusNotModified},
		{"a comma inside a tag", `"a,b"`, "a,b", http.StatusNotModified},
		{"a comma read as a separator", `"a", "b"`, "a,b", http.StatusOK},
		{"an empty element", `, "v7",,`, "v7", http.StatusNotModified},
		{"none of them", `"v5", "v6"`, "v7", http.StatusOK},
		{"a tag against a response that has none", `"v7"`, "", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := conditional(t, asked("If-None-Match", tc.match), tc.tag, at)

			if w.Code != tc.want {
				t.Errorf("the status is %d, want %d", w.Code, tc.want)
			}
		})
	}
}

func TestOnlyAReadIsAnswered(t *testing.T) {
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			r := httptest.NewRequest(method, "/posts/7", nil)
			r.Header.Set("If-None-Match", `"v7"`)

			w := serve(t, r, func(c *Ctx) error {
				c.ETag("v7")
				if c.NotModified() {
					return nil
				}
				return c.Text("sent")
			})

			read := method == "GET" || method == "HEAD"
			if got := w.Code == http.StatusNotModified; got != read {
				t.Errorf("a %s was answered %d", method, w.Code)
			}
		})
	}
}

// TestNothingIsAnsweredAfterTheResponseStarted keeps the check from writing a
// second status over a response that has already gone out.
func TestNothingIsAnsweredAfterTheResponseStarted(t *testing.T) {
	r := asked("If-None-Match", "*")

	w := serve(t, r, func(c *Ctx) error {
		if err := c.Text("already sent"); err != nil {
			return err
		}
		if c.NotModified() {
			t.Error("a response that had already gone out was answered as a 304")
		}
		return nil
	})

	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want 200", w.Code)
	}
	if got, want := w.Body.String(), "already sent"; got != want {
		t.Errorf("the body is %q, want %q", got, want)
	}
}
