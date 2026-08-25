package web

import (
	"errors"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/router"
)

// bound runs one request through a handler and hands the Binding to fn, so
// that a test here is looking at the same Binding a generated binder is given.
func bound(t *testing.T, r *http.Request, fn func(*Binding)) {
	t.Helper()
	serve(t, r, func(c *Ctx) error {
		fn(c.Binding())
		return nil
	})
}

// pairs is every name and value Values yields, in order.
func pairs(t *testing.T, r *http.Request) ([]string, error) {
	t.Helper()

	var out []string
	var err error
	bound(t, r, func(b *Binding) {
		for name, value := range b.Values() {
			out = append(out, name+"="+value)
		}
		err = b.Err()
	})
	return out, err
}

func TestValuesReadsAQueryString(t *testing.T) {
	got, err := pairs(t, httptest.NewRequest("GET", "/?q=water&page=3&tags=a&tags=b", nil))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"q=water", "page=3", "tags=a", "tags=b"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q in the order the request wrote them", got, want)
	}
}

// A query string arrives whatever the body is, so a JSON request binds its path
// and its query the same way and leaves the members to Body.
func TestValuesReadsAQueryStringUnderABodyThatIsNotAForm(t *testing.T) {
	r := httptest.NewRequest("POST", "/?q=water", strings.NewReader(`{"page":3}`))
	r.Header.Set("Content-Type", "application/json")

	got, err := pairs(t, r)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"q=water"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValuesUnescapes(t *testing.T) {
	got, err := pairs(t, httptest.NewRequest("GET", "/?a+b=one+two&c%2Fd=three%2Ffour", nil))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a b=one two", "c/d=three/four"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A name with no equals sign after it is a name sent with nothing under it,
// which is what an empty text input posts and what net/http reads it as.
func TestValuesReadsANameWithNothingUnderIt(t *testing.T) {
	got, err := pairs(t, httptest.NewRequest("GET", "/?q&&page=", nil))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"q=", "page="}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q, so an empty pair is skipped and an empty value is not", got, want)
	}
}

func TestValuesReadsAFormBody(t *testing.T) {
	got, err := pairs(t, form("POST", url.Values{"q": {"water"}, "page": {"3"}}))
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if want := []string{"page=3", "q=water"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The body comes first and the query string after it, minus the names the body
// used, which is what makes a body outrank a query parameter of the same name.
func TestValuesPutsTheBodyInFrontOfTheQueryString(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func(*testing.T) *http.Request
		want []string
	}{
		{"urlencoded", func(*testing.T) *http.Request {
			r := form("POST", url.Values{"q": {"body"}})
			r.URL.RawQuery = "q=query&page=3"
			return r
		}, []string{"q=body", "page=3"}},
		{"multipart", func(t *testing.T) *http.Request {
			r := multipartOf(t, url.Values{"q": {"body"}})
			r.URL.RawQuery = "q=query&page=3"
			return r
		}, []string{"q=body", "page=3"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pairs(t, tc.make(t))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// An empty body is a POST with a form content type and nothing in it, which is
// a form with every field left blank.
func TestValuesReadsTheQueryStringUnderAnEmptyFormBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/?q=water", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ContentLength = -1

	got, err := pairs(t, r)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"q=water"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValuesReadsAMultipartBodyWithNoQueryString(t *testing.T) {
	got, err := pairs(t, multipartOf(t, url.Values{"q": {"water"}}))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"q=water"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Stopping the loop stops the scan, whichever half of the request the consumer
// was in when it had had enough.
func TestValuesStopsWhenTheLoopDoes(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func(*testing.T) *http.Request
		stop int
	}{
		{"a form body", func(*testing.T) *http.Request {
			r := form("POST", url.Values{"a": {"1"}, "b": {"2"}})
			r.URL.RawQuery = "c=3"
			return r
		}, 1},
		{"a query string", func(*testing.T) *http.Request {
			r := form("POST", url.Values{"a": {"1"}})
			r.URL.RawQuery = "b=2&c=3"
			return r
		}, 2},
		{"a multipart body", func(t *testing.T) *http.Request {
			return multipartOf(t, url.Values{"a": {"1"}, "b": {"2"}})
		}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seen := 0
			bound(t, tc.make(t), func(b *Binding) {
				for range b.Values() {
					if seen++; seen == tc.stop {
						break
					}
				}
			})
			if seen != tc.stop {
				t.Errorf("the loop ran %d times, want it to stop at %d", seen, tc.stop)
			}
		})
	}
}

// A handler that reads the body before it binds binds what it read, since the
// bytes are put back where net/http goes looking for them.
func TestAFormBindsAfterTheBodyWasRead(t *testing.T) {
	var got []string
	serve(t, multipartOf(t, url.Values{"q": {"water"}}), func(c *Ctx) error {
		if _, err := c.BodyBytes(); err != nil {
			return err
		}
		b := c.Binding()
		for name, value := range b.Values() {
			got = append(got, name+"="+value)
		}
		return b.Err()
	})
	if want := []string{"q=water"}; !slices.Equal(got, want) {
		t.Errorf("read %q, want %q", got, want)
	}
}

func TestValuesReportsAPairThatWillNotUnescape(t *testing.T) {
	cases := []struct{ name, target string }{
		{"in the name", "/?%zz=1"},
		{"in the value", "/?q=%zz"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := pairs(t, httptest.NewRequest("GET", c.target, nil))
			if err == nil {
				t.Fatalf("read %q, want an error", got)
			}
			if code := errs.CodeOf(err); code != "bind.unreadable" {
				t.Errorf("the code is %q, want bind.unreadable", code)
			}
		})
	}
}

// A body that will not unescape stops before the query string, since a request
// nobody could read is not one to read half of.
func TestValuesStopsAtABodyThatWillNotUnescape(t *testing.T) {
	r := httptest.NewRequest("POST", "/?page=3", strings.NewReader("q=%zz"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got, err := pairs(t, r)
	if err == nil {
		t.Fatalf("read %q, want an error", got)
	}
	if len(got) != 0 {
		t.Errorf("read %q, want nothing, since the body came first", got)
	}
}

// A name in the body that will not unescape is left out of the exclusion list
// rather than reported, because the body is scanned again straight after and
// that is where it is reported.
func TestValuesSkipsANameThatWillNotUnescapeWhenItCollectsThem(t *testing.T) {
	r := httptest.NewRequest("POST", "/?page=3", strings.NewReader("q=water&%zz=1"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got, err := pairs(t, r)
	if err == nil {
		t.Fatalf("read %q, want an error", got)
	}
	if want := []string{"q=water"}; !slices.Equal(got, want) {
		t.Errorf("read %q, want %q before it stopped", got, want)
	}
}

func TestValuesRefusesAFormOverTheLimit(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("q="+strings.Repeat("x", maxFormSize)))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got, err := pairs(t, r)
	if err == nil {
		t.Fatalf("read %q, want an error", got)
	}
	if kind := errs.KindOf(err); kind != errs.TooLarge {
		t.Errorf("the kind is %v, want %v", kind, errs.TooLarge)
	}
}

func TestValuesReportsABodyThatWillNotRead(t *testing.T) {
	want := errors.New("the connection went away")
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Body = io.NopCloser(iotest.ErrReader(want))
	r.ContentLength = -1

	_, err := pairs(t, r)
	if !errors.Is(err, want) {
		t.Errorf("the error is %v, want it to carry %v", err, want)
	}
}

func TestValuesReportsAMultipartBodyThatWillNotParse(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("not a multipart body"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=abc")

	got, err := pairs(t, r)
	if err == nil {
		t.Fatalf("read %q, want an error", got)
	}
	if code := errs.CodeOf(err); code != "bind.unreadable" {
		t.Errorf("the code is %q, want bind.unreadable", code)
	}
}

// A binding that has already given up reads nothing more, so a generated binder
// that ranges the values after a failed one does not run its whole switch over
// a request nobody could read.
func TestValuesYieldsNothingAfterAFailure(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("not a multipart body"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=abc")

	var second int
	bound(t, r, func(b *Binding) {
		for range b.Values() {
		}
		for range b.Values() {
			second++
		}
	})
	if second != 0 {
		t.Errorf("the second loop ran %d times, want none", second)
	}
}

func TestPath(t *testing.T) {
	var got, missing string
	var had, hadMissing bool

	rt := router.New()
	rt.Handle("GET /posts/{id}", H(func(c *Ctx) error {
		b := c.Binding()
		got, had = b.Path("id")
		missing, hadMissing = b.Path("slug")
		return nil
	}))
	rt.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/posts/12", nil))

	if got != "12" || !had {
		t.Errorf("id = %q, %v, want 12 and true", got, had)
	}
	if missing != "" || hadMissing {
		t.Errorf("slug = %q, %v, want the empty string and false", missing, hadMissing)
	}
}

func TestHeaderAndHeaders(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Add("X-Trace", "one")
	r.Header.Add("X-Trace", "two")

	bound(t, r, func(b *Binding) {
		if got, ok := b.Header("X-Trace"); got != "one" || !ok {
			t.Errorf("Header = %q, %v, want one and true", got, ok)
		}
		if got, ok := b.Header("X-Missing"); got != "" || ok {
			t.Errorf("Header of a header nobody sent = %q, %v, want the empty string and false", got, ok)
		}
		if got := b.Headers("X-Trace"); !slices.Equal(got, []string{"one", "two"}) {
			t.Errorf("Headers = %q, want one and two", got)
		}
		if got := b.Headers("X-Missing"); len(got) != 0 {
			t.Errorf("Headers of a header nobody sent = %q, want none", got)
		}
	})
}

func TestCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "seen", Value: "yes"})

	bound(t, r, func(b *Binding) {
		if got, ok := b.Cookie("seen"); got != "yes" || !ok {
			t.Errorf("Cookie = %q, %v, want yes and true", got, ok)
		}
		if got, ok := b.Cookie("missing"); got != "" || ok {
			t.Errorf("Cookie of one nobody sent = %q, %v, want the empty string and false", got, ok)
		}
	})
}

func TestFiles(t *testing.T) {
	var body strings.Builder
	w := multipart.NewWriter(&body)
	for _, f := range []struct{ field, name string }{{"photos", "one.txt"}, {"photos", "two.txt"}} {
		part, err := w.CreateFormFile(f.field, f.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/", strings.NewReader(body.String()))
	r.Header.Set("Content-Type", w.FormDataContentType())

	bound(t, r, func(b *Binding) {
		got := b.Files("photos")
		if len(got) != 2 {
			t.Fatalf("got %d files, want 2", len(got))
		}
		if got[0].Filename != "one.txt" || got[1].Filename != "two.txt" {
			t.Errorf("the files are %q and %q, want one.txt and two.txt", got[0].Filename, got[1].Filename)
		}
		if n := len(b.Files("avatar")); n != 0 {
			t.Errorf("got %d files under a name nobody sent, want none", n)
		}
		if err := b.Err(); err != nil {
			t.Errorf("Err = %v, want nil", err)
		}
	})
}

// A request with no multipart body in it has no files in it either, which is
// every GET and every JSON request.
func TestFilesOfARequestThatCarriesNone(t *testing.T) {
	bound(t, form("POST", url.Values{"q": {"water"}}), func(b *Binding) {
		if n := len(b.Files("avatar")); n != 0 {
			t.Errorf("got %d files from an urlencoded form, want none", n)
		}
		if err := b.Err(); err != nil {
			t.Errorf("Err = %v, want nil", err)
		}
	})
}

func TestFilesReportsAFormThatWillNotParse(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("not a multipart body"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=abc")

	bound(t, r, func(b *Binding) {
		if got := b.Files("avatar"); got != nil {
			t.Errorf("got %v, want nothing", got)
		}
		if err := b.Err(); err == nil {
			t.Error("Err = nil, want the form error")
		}
	})
}

func TestBodyDecodesOverTheStruct(t *testing.T) {
	type in struct {
		Q    string `json:"q"`
		Page int    `json:"page"`
	}

	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"page":3}`))
	r.Header.Set("Content-Type", "application/json")

	var got in
	bound(t, r, func(b *Binding) {
		got.Q = "from the query"
		b.Body(&got)
		if err := b.Err(); err != nil {
			t.Fatal(err)
		}
	})
	if got.Q != "from the query" || got.Page != 3 {
		t.Errorf("read %+v, want the body merged over what was there", got)
	}
}

func TestBodyRefusesAMemberNothingDeclared(t *testing.T) {
	type in struct {
		Q string `json:"q"`
	}
	type lax struct {
		AllowUnknown
		Q string `json:"q"`
	}

	make := func() *http.Request {
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"q":"water","nope":1}`))
		r.Header.Set("Content-Type", "application/json")
		return r
	}

	var strict error
	bound(t, make(), func(b *Binding) {
		var v in
		b.Body(&v)
		strict = b.Err()
	})
	if strict == nil {
		t.Error("Body took a member nothing declared, want it refused")
	}

	var loose error
	var v lax
	bound(t, make(), func(b *Binding) {
		b.BodyAllowUnknown(&v)
		loose = b.Err()
	})
	if loose != nil {
		t.Errorf("BodyAllowUnknown = %v, want nil", loose)
	}
	if v.Q != "water" {
		t.Errorf("q is %q, want water", v.Q)
	}
}

// A body that will not parse at all is the whole error, since there are no
// fields to report on when nothing could be read.
func TestBodyReportsABodyThatWillNotParse(t *testing.T) {
	type in struct {
		Q string `json:"q"`
	}

	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"q":`))
	r.Header.Set("Content-Type", "application/json")

	bound(t, r, func(b *Binding) {
		var v in
		b.Body(&v)
		if err := b.Err(); err == nil {
			t.Error("Err = nil, want the parse error")
		}
	})
}

// A binding that has already given up does not go on to read the body, because
// the body is where a request that could not be read spends the rest of its
// time.
func TestBodyDoesNothingAfterAFailure(t *testing.T) {
	type in struct {
		Q string `json:"q"`
	}

	r := httptest.NewRequest("POST", "/?q=%zz", strings.NewReader(`{"q":"water"}`))
	r.Header.Set("Content-Type", "application/json")

	var v in
	bound(t, r, func(b *Binding) {
		for range b.Values() {
		}
		b.Body(&v)
	})
	if v.Q != "" {
		t.Errorf("q is %q, want nothing, since the bind had already failed", v.Q)
	}
}

func TestErrAndInvalid(t *testing.T) {
	bound(t, httptest.NewRequest("GET", "/", nil), func(b *Binding) {
		if err := b.Err(); err != nil {
			t.Errorf("Err of a bind with nothing wrong = %v, want nil", err)
		}

		b.Invalid("page", "invalid_number", "Must be a whole number.")
		b.Invalid("limit", "out_of_range", "Is too large for this field.")

		err := b.Err()
		if kind := errs.KindOf(err); kind != errs.Invalid {
			t.Errorf("the kind is %v, want %v", kind, errs.Invalid)
		}
		fields := errs.Fields(err)
		if len(fields) != 2 {
			t.Fatalf("got %d fields, want 2", len(fields))
		}
		if fields[0].Name != "page" || fields[0].Code != "invalid_number" {
			t.Errorf("the first field is %+v, want page and invalid_number", fields[0])
		}
	})
}

// What stopped a bind outranks what was wrong with one field, since a form
// nobody could read has nothing worth saying about its fields.
func TestErrPutsWhatStoppedTheBindFirst(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("not a multipart body"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=abc")

	bound(t, r, func(b *Binding) {
		b.Invalid("page", "invalid_number", "Must be a whole number.")
		for range b.Values() {
		}
		if code := errs.CodeOf(b.Err()); code != "bind.unreadable" {
			t.Errorf("the code is %q, want bind.unreadable", code)
		}
	})
}

// setters exercises one helper over a set of values, which is the same shape
// for every one of them.
func setters[T comparable](t *testing.T, name string, set func(*Binding, *T, string, string) bool, zero T, cases []setterCase[T]) {
	t.Helper()

	for _, c := range cases {
		t.Run(name+"/"+c.value, func(t *testing.T) {
			bound(t, httptest.NewRequest("GET", "/", nil), func(b *Binding) {
				got := zero
				ok := set(b, &got, "field", c.value)
				if ok != c.ok {
					t.Errorf("%s(%q) = %v, want %v", name, c.value, ok, c.ok)
				}
				if ok && got != c.want {
					t.Errorf("%s(%q) put %v in the field, want %v", name, c.value, got, c.want)
				}
				if !ok && got != zero {
					t.Errorf("%s(%q) put %v in the field, want it left alone", name, c.value, got)
				}

				fields := errs.Fields(b.Err())
				if c.code == "" {
					if len(fields) != 0 {
						t.Errorf("%s(%q) reported %+v, want nothing", name, c.value, fields)
					}
					return
				}
				if len(fields) != 1 {
					t.Fatalf("%s(%q) reported %+v, want one field", name, c.value, fields)
				}
				if fields[0].Name != "field" || fields[0].Code != c.code {
					t.Errorf("%s(%q) reported %+v, want field and %s", name, c.value, fields[0], c.code)
				}
			})
		})
	}
}

type setterCase[T any] struct {
	value string
	want  T
	ok    bool
	code  string // what it reported, or the empty string for nothing
}

func TestInt(t *testing.T) {
	setters(t, "Int", Int[int], 0, []setterCase[int]{
		{value: "3", want: 3, ok: true},
		{value: "-3", want: -3, ok: true},
		{value: ""},
		{value: "three", code: "invalid_number"},
		{value: "99999999999999999999", code: "out_of_range"},
	})
	setters(t, "Int8", Int[int8], 0, []setterCase[int8]{
		{value: "127", want: 127, ok: true},
		{value: "300", code: "out_of_range"},
	})
}

func TestUint(t *testing.T) {
	setters(t, "Uint", Uint[uint], 0, []setterCase[uint]{
		{value: "3", want: 3, ok: true},
		{value: ""},
		{value: "-1", code: "invalid_number"},
		{value: "99999999999999999999999", code: "out_of_range"},
	})
	setters(t, "Uint8", Uint[uint8], 0, []setterCase[uint8]{
		{value: "255", want: 255, ok: true},
		{value: "300", code: "out_of_range"},
	})
}

func TestFloat(t *testing.T) {
	setters(t, "Float", Float[float64], 0, []setterCase[float64]{
		{value: "1.5", want: 1.5, ok: true},
		{value: ""},
		{value: "half", code: "invalid_number"},
	})
	setters(t, "Float32", Float[float32], 0, []setterCase[float32]{
		{value: "1.5", want: 1.5, ok: true},
		{value: "1e39", code: "out_of_range"},
		{value: "Inf", want: float32(math.Inf(1)), ok: true},
	})
}

func TestBool(t *testing.T) {
	setters(t, "Bool", Bool[bool], false, []setterCase[bool]{
		{value: "true", want: true, ok: true},
		{value: "1", want: true, ok: true},
		{value: "on", want: true, ok: true},
		{value: "yes", want: true, ok: true},
		{value: "off", ok: true},
		{value: "no", ok: true},
		{value: "false", ok: true},
		{value: ""},
		{value: "maybe", code: "invalid_boolean"},
	})
}

func TestTimeSetter(t *testing.T) {
	utc := func(s string) time.Time {
		got, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	setters(t, "Time", Time, time.Time{}, []setterCase[time.Time]{
		{value: "2026-08-25T09:30:00Z", want: utc("2026-08-25T09:30:00Z"), ok: true},
		{value: "2026-08-25", want: utc("2026-08-25T00:00:00Z"), ok: true},
		{value: "2026-08-25T09:30", want: utc("2026-08-25T09:30:00Z"), ok: true},
		{value: ""},
		{value: "yesterday", code: "invalid_time"},
	})
}

func TestDurationSetter(t *testing.T) {
	setters(t, "Duration", Duration[time.Duration], 0, []setterCase[time.Duration]{
		{value: "30s", want: 30 * time.Second, ok: true},
		{value: ""},
		{value: "a while", code: "invalid_duration"},
	})
}

func TestText(t *testing.T) {
	for _, c := range []struct {
		value string
		ok    bool
		code  string
	}{
		{value: "192.0.2.7", ok: true},
		{value: ""},
		{value: "not an address", code: "invalid_value"},
	} {
		t.Run("Text/"+c.value, func(t *testing.T) {
			bound(t, httptest.NewRequest("GET", "/", nil), func(b *Binding) {
				var got netip.Addr
				if ok := Text(b, &got, "field", c.value); ok != c.ok {
					t.Errorf("Text(%q) = %v, want %v", c.value, ok, c.ok)
				}
				fields := errs.Fields(b.Err())
				if c.code == "" {
					if len(fields) != 0 {
						t.Errorf("Text(%q) reported %+v, want nothing", c.value, fields)
					}
					return
				}
				if len(fields) != 1 || fields[0].Code != c.code {
					t.Errorf("Text(%q) reported %+v, want %s", c.value, fields, c.code)
				}
			})
		})
	}
}

// A generated binder is what Bind reaches for first, so a type that has one is
// bound by it and never by reflection.
func TestBindUsesABinderWhenTheTypeHasOne(t *testing.T) {
	in, err := bind[counted](t, httptest.NewRequest("GET", "/?q=water", nil))
	if err != nil {
		t.Fatal(err)
	}
	if in.Q != "water" {
		t.Errorf("q is %q, want water", in.Q)
	}
	if !in.Ran {
		t.Error("the binder did not run, so reflection bound the struct instead")
	}
}

// counted is a hand-written binder, which is what the generator writes and what
// this package has to work against without the generator in the way.
type counted struct {
	Q   string `query:"q"`
	Ran bool
}

func (v *counted) BindRequest(c *Ctx) error {
	b := c.Binding()
	v.Ran = true

	for name, value := range b.Values() {
		switch name {
		case "q":
			v.Q = value
		}
	}

	b.Body(v)
	return b.Err()
}

// A binder that reports something is a bind that failed, and Bind throws the
// half-filled struct away exactly as it does for the reflective one.
func TestBindReportsWhatABinderReports(t *testing.T) {
	in, err := bind[counted](t, httptest.NewRequest("GET", "/?q=%zz", nil))
	if err == nil {
		t.Fatalf("read %+v, want an error", in)
	}
	if in.Q != "" || in.Ran {
		t.Errorf("read %+v, want the zero value", in)
	}
}
