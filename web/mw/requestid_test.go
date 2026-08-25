package mw

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/clock"
	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/str"
	"github.com/go-mizu/mizu/web"
)

// seen serves one request through mw and reports the id the handler was given
// along with the response.
func seen(tb testing.TB, mw web.Middleware, r *http.Request) (string, *httptest.ResponseRecorder) {
	tb.Helper()

	var id string
	w := httptest.NewRecorder()
	mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		id, _ = ctxdata.Get(r.Context(), web.RequestIDKey)
	})).ServeHTTP(w, r)
	return id, w
}

func TestRequestIDPutsTheSameIdInBothPlaces(t *testing.T) {
	id, w := seen(t, RequestID(), httptest.NewRequest("GET", "/", nil))

	if id == "" {
		t.Fatal("the handler was given no request id")
	}
	if got := w.Header().Get(RequestIDHeader); got != id {
		t.Errorf("the response says %q and the handler was given %q", got, id)
	}
}

func TestTheIdIsAULID(t *testing.T) {
	id, _ := seen(t, RequestID(), httptest.NewRequest("GET", "/", nil))
	if !str.IsULID(id) {
		t.Errorf("the id %q is not a ULID", id)
	}
}

// TestRequestIDIgnoresWhatTheClientSent is the reason there are two
// constructors: the one with the short name does not read the wire.
func TestRequestIDIgnoresWhatTheClientSent(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(RequestIDHeader, "01ARZ3NDEKTSV4RRFFQ69G5FAV")

	if id, _ := seen(t, RequestID(), r); id == "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Error("RequestID used the id the client sent")
	}
}

func TestRequestIDFromUsesTheHeaderItWasPointedAt(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Correlation-Id", "abc-123")

	id, w := seen(t, RequestIDFrom("X-Correlation-Id"), r)
	if id != "abc-123" {
		t.Errorf("the handler was given %q, want abc-123", id)
	}
	// The response header is the one everything reads, whichever one was read.
	if got := w.Header().Get(RequestIDHeader); got != "abc-123" {
		t.Errorf("the response says %q, want abc-123", got)
	}
}

func TestRequestIDFromMakesOneWhenTheHeaderIsNotThere(t *testing.T) {
	id, _ := seen(t, RequestIDFrom("X-Correlation-Id"), httptest.NewRequest("GET", "/", nil))
	if !str.IsULID(id) {
		t.Errorf("the id %q is not a ULID", id)
	}
}

func TestAnIdThatIsNotShapedLikeOneIsReplaced(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"a ULID", "01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		{"a UUID", "b5f4a3c2-1111-4222-8333-444455556666", "b5f4a3c2-1111-4222-8333-444455556666"},
		{"a span id with dots", "trace.1.2", "trace.1.2"},
		{"an underscore", "req_9", "req_9"},
		{"one character", "x", "x"},
		{"nothing", "", ""},
		{"a space", "one two", ""},
		{"a newline, which is a log line somebody else wrote", "id\nlevel=ERROR msg=hi", ""},
		{"a quote, which is a JSON field somebody else wrote", `id","level":"ERROR`, ""},
		{"a slash", "a/b", ""},
		{"sixty four characters", strings.Repeat("a", 64), strings.Repeat("a", 64)},
		{"sixty five characters", strings.Repeat("a", 65), ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := plausible(c.in); got != c.want {
				t.Errorf("plausible(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestTheTimestampIsTheOneOnTheClock uses the example from the ULID
// specification, which is the only outside answer there is to check against.
func TestTheTimestampIsTheOneOnTheClock(t *testing.T) {
	const (
		ms   = 1469918176385
		want = "01ARYZ6S41"
	)

	if got := ulid(time.UnixMilli(ms))[:len(want)]; got != want {
		t.Errorf("the timestamp reads %q, want %q", got, want)
	}
}

func TestTheClockComesFromTheContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	fake := clock.Fake(time.UnixMilli(1469918176385))
	r = r.WithContext(clock.With(r.Context(), fake))

	id, _ := seen(t, RequestID(), r)
	if got, want := id[:10], "01ARYZ6S41"; got != want {
		t.Errorf("the id starts %q, want %q", got, want)
	}
}

func TestTwoIdsFromOneInstantStillDiffer(t *testing.T) {
	at := time.UnixMilli(1469918176385)
	if a, b := ulid(at), ulid(at); a == b {
		t.Errorf("two ids made in the same millisecond are both %q", a)
	}
}

func TestIdsSortIntoTheOrderTheyWereMade(t *testing.T) {
	at := time.UnixMilli(1469918176385)

	var ids []string
	for i := range 8 {
		ids = append(ids, ulid(at.Add(time.Duration(i)*time.Millisecond)))
	}

	if !slices.IsSorted(ids) {
		t.Errorf("ids made in order do not sort in it:\n%s", strings.Join(ids, "\n"))
	}
}

// TestTheEncodingReachesBothEndsOfTheAlphabet is the check on the bit shuffling
// that nothing else covers: an all zero value and an all ones value are the two
// places an off by one shows up.
func TestTheEncodingReachesBothEndsOfTheAlphabet(t *testing.T) {
	var zero [16]byte
	if got, want := encode(zero), strings.Repeat("0", 26); got != want {
		t.Errorf("the zero value encodes as %q, want %q", got, want)
	}

	var ones [16]byte
	for i := range ones {
		ones[i] = 0xff
	}
	if got, want := encode(ones), "7"+strings.Repeat("Z", 25); got != want {
		t.Errorf("the all ones value encodes as %q, want %q", got, want)
	}
}

// TestEveryIdIsOneStrWillAccept runs the two packages against each other, since
// the encoder here and the predicate there were written from the same
// specification and never from each other.
func TestEveryIdIsOneStrWillAccept(t *testing.T) {
	for range 512 {
		if id := ulid(time.Now()); !str.IsULID(id) {
			t.Fatalf("str.IsULID rejected %q", id)
		}
	}
}
