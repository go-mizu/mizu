package web

import (
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

// A post is the shape a handler sends back, with the two tags doc 08 tells
// people to write on one.
type reply struct {
	ID    int      `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitzero"`
}

func TestJSONSendsAValue(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c, reply{ID: 1, Title: "water"})
	})

	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want 200", w.Code)
	}
	if want := `{"id":1,"title":"water"}`; w.Body.String() != want {
		t.Errorf("the body is %s, want %s", w.Body.String(), want)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("the content type is %q, want application/json with no charset on it", got)
	}
}

// RFC 8259 registers no charset parameter for application/json, because JSON is
// UTF-8 and saying so twice is one of the two places it can disagree with
// itself.
func TestJSONWritesNoTrailingNewline(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c, reply{ID: 2, Title: "fire"})
	})
	if strings.HasSuffix(w.Body.String(), "\n") {
		t.Errorf("the body is %q and ends with a newline", w.Body.String())
	}
}

func TestJSONCountsWhatItSent(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c, reply{ID: 3, Title: "earth", Tags: []string{"a", "b"}})
	})

	n, err := strconv.Atoi(w.Header().Get("Content-Length"))
	if err != nil {
		t.Fatalf("the content length is %q: %v", w.Header().Get("Content-Length"), err)
	}
	if n != w.Body.Len() {
		t.Errorf("the content length is %d and the body is %d bytes", n, w.Body.Len())
	}
}

func TestJSONTakesTheStatusItWasGiven(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c.Status(http.StatusTeapot), reply{ID: 4})
	})
	if w.Code != http.StatusTeapot {
		t.Errorf("the status is %d, want 418", w.Code)
	}
}

func TestJSONStatusNamesTheStatus(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSONStatus(c, http.StatusConflict, reply{ID: 5, Title: "taken"})
	})
	if w.Code != http.StatusConflict {
		t.Errorf("the status is %d, want 409", w.Code)
	}
	if want := `{"id":5,"title":"taken"}`; w.Body.String() != want {
		t.Errorf("the body is %s, want %s", w.Body.String(), want)
	}
}

// A value that will not marshal is an error the handler returns, with nothing
// written and the status still to play for. That is what buffering the body
// before sending any of it is for.
func TestAValueThatWillNotMarshalWritesNothing(t *testing.T) {
	var err error
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		err = JSON(c, map[string]float64{"n": math.NaN()})
		return nil
	})

	if err == nil {
		t.Fatal("a NaN went out as JSON, which is not a JSON number")
	}
	if !errors.Is(err, errs.Internal) {
		t.Errorf("the kind is %v, want Internal", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "respond.json" {
		t.Errorf("the code is %q, want respond.json", got)
	}
	if w.Body.Len() != 0 {
		t.Errorf("the body is %q and nothing should have gone out", w.Body.String())
	}
}

func TestCreatedSaysWhereItPutIt(t *testing.T) {
	w := serve(t, httptest.NewRequest("POST", "/posts", nil), func(c *Ctx) error {
		return Created(c, "/posts/6", reply{ID: 6, Title: "new"})
	})

	if w.Code != http.StatusCreated {
		t.Errorf("the status is %d, want 201", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/posts/6" {
		t.Errorf("the location is %q, want /posts/6", got)
	}
	if want := `{"id":6,"title":"new"}`; w.Body.String() != want {
		t.Errorf("the body is %s, want %s", w.Body.String(), want)
	}
}

func TestCreatedWithNowhereToPointAtSendsNoLocation(t *testing.T) {
	w := serve(t, httptest.NewRequest("POST", "/posts", nil), func(c *Ctx) error {
		return Created(c, "", reply{ID: 7})
	})
	if _, ok := w.Header()["Location"]; ok {
		t.Errorf("the location is %q and there was none to give", w.Header().Get("Location"))
	}
}

func TestAcceptedSendsNoBody(t *testing.T) {
	w := serve(t, httptest.NewRequest("POST", "/imports", nil), func(c *Ctx) error {
		return Accepted(c, "/jobs/8")
	})

	if w.Code != http.StatusAccepted {
		t.Errorf("the status is %d, want 202", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/jobs/8" {
		t.Errorf("the location is %q, want /jobs/8", got)
	}
	if w.Body.Len() != 0 {
		t.Errorf("the body is %q, want nothing", w.Body.String())
	}
}

func TestAcceptedWithNowhereToPollSendsNoLocation(t *testing.T) {
	w := serve(t, httptest.NewRequest("POST", "/imports", nil), func(c *Ctx) error {
		return Accepted(c, "")
	})
	if w.Code != http.StatusAccepted {
		t.Errorf("the status is %d, want 202", w.Code)
	}
	if _, ok := w.Header()["Location"]; ok {
		t.Errorf("the location is %q and there was none to give", w.Header().Get("Location"))
	}
}

// A response has to be the same bytes every time or an ETag over it means
// nothing, and a Go map is the one shape where that is not free.
func TestAMapComesOutInKeyOrder(t *testing.T) {
	in := map[string]int{"z": 1, "a": 2, "m": 3, "b": 4}
	for range 20 {
		w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
			return JSON(c, in)
		})
		if want := `{"a":2,"b":4,"m":3,"z":1}`; w.Body.String() != want {
			t.Fatalf("the body is %s, want %s", w.Body.String(), want)
		}
	}
}

// The older encoder turns these three into escapes and the newer one does not,
// so the older one is told not to. A body served as application/json is not in
// a script tag, and one that is belongs to the template package.
func TestTheAngleBracketsGoOutAsThemselves(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c, reply{ID: 9, Title: "<b>&</b>"})
	})
	if want := `{"id":9,"title":"<b>&</b>"}`; w.Body.String() != want {
		t.Errorf("the body is %s, want %s", w.Body.String(), want)
	}
}

// The buffer the body is built in comes from a pool, so the second response of
// a shape allocates less than the first and neither of them is holding bytes
// from the one before.
func TestTheBodyBufferIsReusedAndNotCarriedOver(t *testing.T) {
	long := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c, reply{ID: 10, Title: strings.Repeat("x", 500)})
	})
	short := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c, reply{ID: 11, Title: "y"})
	})

	if want := `{"id":11,"title":"y"}`; short.Body.String() != want {
		t.Errorf("the body is %s, want %s", short.Body.String(), want)
	}
	if long.Body.Len() <= short.Body.Len() {
		t.Fatal("the two responses are the same size, so this checks nothing")
	}
}

// A buffer that grew past what is worth keeping is dropped rather than parked
// in the pool for the rest of the process.
func TestAHugeBodyDoesNotStayInThePool(t *testing.T) {
	big := &jsonBuf{b: make([]byte, 0, keepBuf+1)}
	putBuf(big)

	for range 8 {
		if got := bufs.Get(); got == big {
			t.Fatal("the buffer went back in the pool and it is bigger than keepBuf")
		}
	}
}

// omitempty is the one tag whose meaning changed between the two packages, and
// it changed on purpose. It is asserted rather than papered over, and omitzero,
// which doc 08 tells people to write, means the same thing in both.
func TestOmitemptyIsWhateverThisBuildSaysItIs(t *testing.T) {
	type counts struct {
		Zero int `json:"zero,omitempty"`
		Also int `json:"also,omitzero"`
	}

	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c, counts{})
	})

	want := `{"zero":0}`
	if jsonOmitemptyMeansZero {
		want = `{}`
	}
	if w.Body.String() != want {
		t.Errorf("the body is %s, want %s on this build", w.Body.String(), want)
	}
}

// A stamp writes itself, and the method is on the pointer, which is where most
// people put a method that fills something in.
type stamp struct{ At string }

func (s *stamp) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote("at " + s.At)), nil
}

// The value goes to the encoder as a pointer to it, so a marshaler declared on
// the pointer is found even when the handler passed the value. json.Marshal of
// the same value would skip the method without saying so, which is the older
// footgun this does not reproduce.
func TestAMarshalerOnThePointerIsFound(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return JSON(c, stamp{At: "noon"})
	})
	if want := `"at noon"`; w.Body.String() != want {
		t.Errorf("the body is %s, want %s", w.Body.String(), want)
	}
}

// Taking the address of the parameter means the encoder sees one more pointer
// than the caller passed, so the shapes that are already a pointer or already an
// interface have to come out the same as the plain value does.
func TestTheExtraPointerIsInvisible(t *testing.T) {
	var boxed any = reply{ID: 13, Title: "boxed"}
	var missing *reply

	cases := []struct {
		name string
		send func(*Ctx) error
		want string
	}{
		{"a value", func(c *Ctx) error { return JSON(c, reply{ID: 13, Title: "boxed"}) }, `{"id":13,"title":"boxed"}`},
		{"a pointer to one", func(c *Ctx) error { return JSON(c, &reply{ID: 13, Title: "boxed"}) }, `{"id":13,"title":"boxed"}`},
		{"one in an interface", func(c *Ctx) error { return JSON(c, boxed) }, `{"id":13,"title":"boxed"}`},
		{"a nil pointer", func(c *Ctx) error { return JSON(c, missing) }, `null`},
		// A bare nil has no type to infer T from, so this one has to name it.
		{"nothing at all", func(c *Ctx) error { return JSON[any](c, nil) }, `null`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(t, httptest.NewRequest("GET", "/", nil), tc.send)
			if w.Body.String() != tc.want {
				t.Errorf("the body is %s, want %s", w.Body.String(), tc.want)
			}
		})
	}
}

// A handler writes one response, and the helpers do not allocate a buffer for
// each one.
func TestJSONDoesNotAllocateItsBuffer(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	in := reply{ID: 12, Title: "measured", Tags: []string{"a"}}

	n := testing.AllocsPerRun(200, func() {
		w := httptest.NewRecorder()
		H(func(c *Ctx) error { return JSON(c, in) }).ServeHTTP(w, r)
	})

	// The recorder, the request copy H makes and the Content-Length string are
	// on this number too, so it is a ceiling rather than what JSON costs. The
	// point of pinning it is that a buffer allocated per response would put it
	// somewhere else entirely.
	if n > 20 {
		t.Errorf("a JSON response allocated %v times, which is more than the whole request should", n)
	}
}
