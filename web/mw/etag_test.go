package mw

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/web"
)

// page is the response most of these tests are about.
const page = "<h1>the front page</h1>"

// conditional serves one request through ETag with the given If-None-Match and
// reports the response.
func conditional(tb testing.TB, method, inm string, h http.HandlerFunc) *httptest.ResponseRecorder {
	tb.Helper()

	r := httptest.NewRequest(method, "/", nil)
	if inm != "" {
		r.Header.Set("If-None-Match", inm)
	}

	w := httptest.NewRecorder()
	ETag()(h).ServeHTTP(w, r)
	return w
}

// html is a handler that writes a page.
func html(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, body)
	}
}

func TestAResponseComesBackWithATag(t *testing.T) {
	w := conditional(t, "GET", "", html(page))

	tag := w.Header().Get("ETag")
	if !strings.HasPrefix(tag, `W/"`) || !strings.HasSuffix(tag, `"`) {
		t.Errorf("ETag is %q, want a weak tag in quotes", tag)
	}
	if w.Code != http.StatusOK || w.Body.String() != page {
		t.Errorf("the response is %d %q, want 200 and the page", w.Code, w.Body)
	}
}

// TestTheSameRequestAgainCostsNoBody is the whole of it. The client sends back
// what it was given and the body does not go out a second time.
func TestTheSameRequestAgainCostsNoBody(t *testing.T) {
	tag := conditional(t, "GET", "", html(page)).Header().Get("ETag")

	w := conditional(t, "GET", tag, html(page))

	if w.Code != http.StatusNotModified {
		t.Errorf("the response is %d, want 304", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("a 304 came back with %d bytes of body", w.Body.Len())
	}
	if got := w.Header().Get("ETag"); got != tag {
		t.Errorf("the 304 carries the tag %q, want %q", got, tag)
	}
}

// TestA304DropsTheHeadersThatDescribeABody keeps a cache from being told the
// length and the type of something that is not there.
func TestA304DropsTheHeadersThatDescribeABody(t *testing.T) {
	tag := conditional(t, "GET", "", html(page)).Header().Get("ETag")

	w := conditional(t, "GET", tag, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=0")
		io.WriteString(w, page)
	})

	for _, name := range []string{"Content-Type", "Content-Length"} {
		if got := w.Header().Get(name); got != "" {
			t.Errorf("the 304 carries %s: %q", name, got)
		}
	}
	if got := w.Header().Get("Cache-Control"); got != "private, max-age=0" {
		t.Errorf("Cache-Control is %q, and a cache updates what it holds from a 304", got)
	}
}

func TestAStaleTagGetsTheWholeThing(t *testing.T) {
	w := conditional(t, "GET", `W/"0123456789abcdef0123456789abcdef"`, html(page))

	if w.Code != http.StatusOK {
		t.Errorf("the response is %d, want 200", w.Code)
	}
	if w.Body.String() != page {
		t.Errorf("the body is %q, want the page", w.Body)
	}
}

func TestADifferentBodyGetsADifferentTag(t *testing.T) {
	first := conditional(t, "GET", "", html(page)).Header().Get("ETag")
	second := conditional(t, "GET", "", html(page+" again")).Header().Get("ETag")

	if first == second {
		t.Errorf("two different pages have the same tag %q", first)
	}
}

func TestTheSameBodyGetsTheSameTag(t *testing.T) {
	first := conditional(t, "GET", "", html(page)).Header().Get("ETag")
	second := conditional(t, "GET", "", html(page)).Header().Get("ETag")

	if first != second {
		t.Errorf("the same page got %q and then %q", first, second)
	}
}

func TestIfNoneMatchIsRead(t *testing.T) {
	tag := conditional(t, "GET", "", html(page)).Header().Get("ETag")
	bare := strings.TrimPrefix(tag, "W/")

	cases := map[string]struct {
		inm  string
		want int
	}{
		"the tag":                      {tag, http.StatusNotModified},
		"the tag without its marker":   {bare, http.StatusNotModified},
		"a star":                       {"*", http.StatusNotModified},
		"a list with the tag in it":    {`W/"aaa", ` + tag + `, W/"bbb"`, http.StatusNotModified},
		"a list without it":            {`W/"aaa", W/"bbb"`, http.StatusOK},
		"a list with spaces around it": {"  " + tag + "  ", http.StatusNotModified},
		"nothing":                      {"", http.StatusOK},
		"something that is not a tag":  {"garbage", http.StatusOK},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := conditional(t, "GET", c.inm, html(page)).Code; got != c.want {
				t.Errorf("If-None-Match %q came back %d, want %d", c.inm, got, c.want)
			}
		})
	}
}

// TestATagTheHandlerSetIsKept is how a handler that knows a cheaper validator,
// a version column or a modification time, gets to use it and still have the
// 304 written here.
func TestATagTheHandlerSetIsKept(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("ETag", `"v41"`)
		io.WriteString(w, page)
	}

	fresh := conditional(t, "GET", "", handler)
	if got := fresh.Header().Get("ETag"); got != `"v41"` {
		t.Errorf("ETag is %q, want the one the handler set", got)
	}

	again := conditional(t, "GET", `"v41"`, handler)
	if again.Code != http.StatusNotModified {
		t.Errorf("the handler's own tag came back %d, want 304", again.Code)
	}
}

func TestOnlyGETAndHEADAreTagged(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		t.Run(method, func(t *testing.T) {
			w := conditional(t, method, "*", html(page))

			if got := w.Header().Get("ETag"); got != "" {
				t.Errorf("a %s came back with the tag %q", method, got)
			}
			if w.Code != http.StatusOK || w.Body.String() != page {
				t.Errorf("a %s came back %d with %q", method, w.Code, w.Body)
			}
		})
	}
}

func TestAHEADIsTaggedFromTheBodyTheHandlerWrote(t *testing.T) {
	get := conditional(t, "GET", "", html(page)).Header().Get("ETag")
	head := conditional(t, "HEAD", "", html(page)).Header().Get("ETag")

	if get != head {
		t.Errorf("the HEAD tag is %q and the GET tag is %q, and they are the same representation", head, get)
	}
}

// TestOnlyA200IsTagged keeps a conditional request from being answered about a
// response that is not a representation of anything.
func TestOnlyA200IsTagged(t *testing.T) {
	for _, code := range []int{http.StatusCreated, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			w := conditional(t, "GET", "*", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(code)
				io.WriteString(w, "the reason")
			})

			if w.Code != code {
				t.Errorf("the response is %d, want %d", w.Code, code)
			}
			if got := w.Header().Get("ETag"); got != "" {
				t.Errorf("a %d came back with the tag %q", code, got)
			}
			if w.Body.String() != "the reason" {
				t.Errorf("the body is %q, want the reason", w.Body)
			}
		})
	}
}

// TestABodyTooBigToHoldGoesOutUntagged is the ceiling. Holding a response per
// request is how a server runs out of memory from the outside.
func TestABodyTooBigToHoldGoesOutUntagged(t *testing.T) {
	big := strings.Repeat("m", maxTag+1)

	w := conditional(t, "GET", "*", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, big)
	})

	if got := w.Header().Get("ETag"); got != "" {
		t.Errorf("a body over the ceiling came back with the tag %q", got)
	}
	if w.Body.Len() != len(big) {
		t.Errorf("the body is %d bytes, want %d", w.Body.Len(), len(big))
	}
	if w.Code != http.StatusOK {
		t.Errorf("the response is %d, want 200", w.Code)
	}
}

// TestAFlushedResponseGoesOutUntagged covers a stream. A flush means the bytes
// are wanted now rather than once something has finished thinking about them.
func TestAFlushedResponseGoesOutUntagged(t *testing.T) {
	r := httptest.NewRequest("GET", "/events", nil)
	r.Header.Set("If-None-Match", "*")
	w := httptest.NewRecorder()

	var afterFlush int
	ETag()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: one\n\n")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("the flush failed: %v", err)
		}
		afterFlush = w.(*web.Recorder).Unwrap().(*httptest.ResponseRecorder).Body.Len()
		io.WriteString(w, "data: two\n\n")
	})).ServeHTTP(w, r)

	if afterFlush == 0 {
		t.Error("the first event was still being held after it was flushed")
	}
	if got := w.Header().Get("ETag"); got != "" {
		t.Errorf("a stream came back with the tag %q", got)
	}
	if w.Body.String() != "data: one\n\ndata: two\n\n" {
		t.Errorf("the stream came back as %q", w.Body)
	}
}

// TestFlushingAfterTheBodySpilledIsHarmless. The body already outgrew the buffer
// and went out, so the flush finds nothing being held and carries on.
func TestFlushingAfterTheBodySpilledIsHarmless(t *testing.T) {
	big := strings.Repeat("m", maxTag+1)
	w := httptest.NewRecorder()

	ETag()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, big)
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("the flush failed: %v", err)
		}
	})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if !w.Flushed {
		t.Error("the flush never reached the server")
	}
	if w.Body.Len() != len(big) {
		t.Errorf("the body is %d bytes, want %d", w.Body.Len(), len(big))
	}
}

// TestABodyThatCannotBeSentStopsWhereItIs is a client that hung up while the
// body was still being held. The write that spills reports the failure to the
// handler rather than swallowing it, which is what a handler writing a large
// response needs in order to stop.
func TestABodyThatCannotBeSentStopsWhereItIs(t *testing.T) {
	var err error

	ETag()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, strings.Repeat("m", maxTag))
		_, err = io.WriteString(w, "one byte too many")
	})).ServeHTTP(gone{h: http.Header{}}, httptest.NewRequest("GET", "/", nil))

	if !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("the write came back with %v, want the connection error", err)
	}
}

// gone is the writer for a connection that is not there any more.
type gone struct{ h http.Header }

func (g gone) Header() http.Header     { return g.h }
func (gone) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func (gone) WriteHeader(int)           {}

func TestAResponseWithNoBodyStillGetsATag(t *testing.T) {
	w := conditional(t, "GET", "", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	if got := w.Header().Get("ETag"); got == "" {
		t.Error("an empty 200 came back with no tag, and empty is a representation like any other")
	}
	if w.Code != http.StatusOK {
		t.Errorf("the response is %d, want 200", w.Code)
	}
}

// TestTheTagIsForTheBodyRatherThanForTheCompressionOfIt is why ETag goes inside
// Compress. Two clients asking for the same page, one that takes gzip and one
// that does not, hold the same validator.
func TestTheTagIsForTheBodyRatherThanForTheCompressionOfIt(t *testing.T) {
	srv := Compress()(ETag()(html(strings.Repeat(page, 100))))

	tag := func(accept string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/", nil)
		if accept != "" {
			r.Header.Set("Accept-Encoding", accept)
		}
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		return w
	}

	zip, raw := tag("gzip"), tag("")

	if zip.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("the compressed request came back as %q", zip.Header().Get("Content-Encoding"))
	}
	if got, want := zip.Header().Get("ETag"), raw.Header().Get("ETag"); got != want {
		t.Errorf("the compressed response is tagged %q and the plain one %q", got, want)
	}
	if got := plain(t, zip); got != raw.Body.String() {
		t.Error("the two responses are not the same page")
	}
}

// TestA304UnderCompressIsNotEncoded is the pair of the test above. There is no
// body, so there is nothing to have encoded, and a Content-Encoding on a 304
// would be a lie a cache remembers.
func TestA304UnderCompressIsNotEncoded(t *testing.T) {
	body := strings.Repeat(page, 100)
	srv := Compress()(ETag()(html(body)))

	first := httptest.NewRecorder()
	srv.ServeHTTP(first, httptest.NewRequest("GET", "/", nil))

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("If-None-Match", first.Header().Get("ETag"))
	second := httptest.NewRecorder()
	srv.ServeHTTP(second, r)

	if second.Code != http.StatusNotModified {
		t.Fatalf("the second request came back %d, want 304", second.Code)
	}
	if got := second.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("the 304 carries Content-Encoding: %q", got)
	}
	if second.Body.Len() != 0 {
		t.Errorf("the 304 carries %d bytes of body", second.Body.Len())
	}
}
