package mizutest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// Response is what came back, and the assertions about it.
//
// Every assertion returns the same Response, so they chain, and every one of
// them reports a failure rather than stopping the test. The first failure
// prints the whole exchange and the ones after it print the assertion alone,
// because a response that is wrong is usually wrong for one reason and printing
// two kilobytes of body three times does not make it clearer.
type Response struct {
	tb   testing.TB
	app  *App
	req  *http.Request
	sent []byte
	res  *http.Response
	body []byte
	logs []Entry

	shown  bool // the exchange has been printed once already
	parsed any  // the decoded body, kept because most tests ask more than once
	failed bool // the body would not parse, so JSON assertions stop trying
}

// Status is the status code.
func (r *Response) Status() int { return r.res.StatusCode }

// Header is the response headers.
func (r *Response) Header() http.Header { return r.res.Header }

// Body is the response body.
func (r *Response) Body() []byte { return r.body }

// Text is the response body as a string.
func (r *Response) Text() string { return string(r.body) }

// Cookies is the cookies the response set.
func (r *Response) Cookies() []*http.Cookie { return r.res.Cookies() }

// Cookie is the cookie of a given name, or nil.
func (r *Response) Cookie(name string) *http.Cookie {
	for _, c := range r.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// Decode unmarshals the body into v, and fails the test if it will not.
//
// It is for the assertions a test wants to make in Go rather than through a
// path: a decoded value goes into a comparison, a loop, or another request.
func (r *Response) Decode(v any) *Response {
	r.tb.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		r.fail("the body is not JSON that fits %T: %v", v, err)
	}
	return r
}

// AssertOK asserts 200.
func (r *Response) AssertOK() *Response { r.tb.Helper(); return r.AssertStatus(http.StatusOK) }

// AssertCreated asserts 201.
func (r *Response) AssertCreated() *Response {
	r.tb.Helper()
	return r.AssertStatus(http.StatusCreated)
}

// AssertNoContent asserts 204 and an empty body.
func (r *Response) AssertNoContent() *Response {
	r.tb.Helper()
	r.AssertStatus(http.StatusNoContent)
	if len(r.body) > 0 {
		r.fail("the status is 204 and the body has %d bytes in it", len(r.body))
	}
	return r
}

// AssertUnauthorized asserts 401.
func (r *Response) AssertUnauthorized() *Response {
	r.tb.Helper()
	return r.AssertStatus(http.StatusUnauthorized)
}

// AssertForbidden asserts 403.
func (r *Response) AssertForbidden() *Response {
	r.tb.Helper()
	return r.AssertStatus(http.StatusForbidden)
}

// AssertNotFound asserts 404.
func (r *Response) AssertNotFound() *Response {
	r.tb.Helper()
	return r.AssertStatus(http.StatusNotFound)
}

// AssertUnprocessable asserts 422, which is what a request that parsed and did
// not validate comes back as.
func (r *Response) AssertUnprocessable() *Response {
	r.tb.Helper()
	return r.AssertStatus(http.StatusUnprocessableEntity)
}

// AssertStatus asserts an exact status code.
func (r *Response) AssertStatus(want int) *Response {
	r.tb.Helper()
	if got := r.Status(); got != want {
		r.fail("the status is %d %s, want %d %s",
			got, http.StatusText(got), want, http.StatusText(want))
	}
	return r
}

// AssertRedirect asserts a 3xx status and a Location header.
func (r *Response) AssertRedirect(to string) *Response {
	r.tb.Helper()
	if got := r.Status(); got < 300 || got > 399 {
		r.fail("the status is %d %s, want a redirect", got, http.StatusText(got))
		return r
	}
	if got := r.Header().Get("Location"); got != to {
		r.fail("the redirect goes to %q, want %q", got, to)
	}
	return r
}

// AssertHeader asserts a header has a value.
func (r *Response) AssertHeader(key, want string) *Response {
	r.tb.Helper()
	got, ok := r.Header()[http.CanonicalHeaderKey(key)]
	if !ok {
		r.fail("there is no %s header, and there is %s", key, headerNames(r.Header()))
		return r
	}
	if len(got) == 1 && got[0] == want {
		return r
	}
	if len(got) == 1 {
		r.fail("%s is %q, want %q", key, got[0], want)
		return r
	}
	r.fail("%s is sent %d times as %q, want the one value %q", key, len(got), got, want)
	return r
}

// AssertHeaderMissing asserts a header is not there at all. A header set to the
// empty string counts as there, since that is what the handler decided to send.
func (r *Response) AssertHeaderMissing(key string) *Response {
	r.tb.Helper()
	if got, ok := r.Header()[http.CanonicalHeaderKey(key)]; ok {
		r.fail("%s is set to %q, want it not sent at all", key, got)
	}
	return r
}

// AssertCookie asserts a cookie was set with a value.
func (r *Response) AssertCookie(name, want string) *Response {
	r.tb.Helper()
	c := r.Cookie(name)
	if c == nil {
		r.fail("no cookie named %q was set, and %s were", name, cookieNames(r.Cookies()))
		return r
	}
	if c.Value != want {
		r.fail("the cookie %q is %q, want %q", name, c.Value, want)
	}
	return r
}

// AssertCookieExpired asserts a cookie was sent back with an expiry in the
// past, which is how a cookie is deleted.
func (r *Response) AssertCookieExpired(name string) *Response {
	r.tb.Helper()
	c := r.Cookie(name)
	if c == nil {
		r.fail("no cookie named %q was set, and %s were", name, cookieNames(r.Cookies()))
		return r
	}
	if c.MaxAge < 0 {
		return r
	}
	if !c.Expires.IsZero() && c.Expires.Before(r.now()) {
		return r
	}
	r.fail("the cookie %q is not expired: MaxAge is %d and Expires is %v", name, c.MaxAge, c.Expires)
	return r
}

// AssertSee asserts the body contains a string.
func (r *Response) AssertSee(want string) *Response {
	r.tb.Helper()
	if !bytes.Contains(r.body, []byte(want)) {
		r.fail("the body does not contain %q", want)
	}
	return r
}

// AssertDontSee asserts the body does not contain a string.
func (r *Response) AssertDontSee(unwanted string) *Response {
	r.tb.Helper()
	if i := bytes.Index(r.body, []byte(unwanted)); i >= 0 {
		r.fail("the body contains %q at byte %d, want it absent", unwanted, i)
	}
	return r
}

// AssertSeeInOrder asserts the body contains every string, each one after the
// last. It is the assertion for output that is a list, where the order is the
// point and the contents around it are not.
func (r *Response) AssertSeeInOrder(want ...string) *Response {
	r.tb.Helper()
	at := 0
	for i, w := range want {
		j := bytes.Index(r.body[at:], []byte(w))
		if j < 0 {
			if bytes.Contains(r.body, []byte(w)) {
				r.fail("the body has %q before %q, want it after", w, want[i-1])
			} else {
				r.fail("the body does not contain %q", w)
			}
			return r
		}
		at += j + len(w)
	}
	return r
}

// AssertJSON asserts the body is exactly this document, member for member.
//
// want is anything json.Marshal takes, and goes through the same encoding as
// the response, so a struct and a map that say the same thing both match and
// the comparison is between documents rather than between Go values. To write
// the document out as text, say so with a json.RawMessage:
//
//	res.AssertJSON(json.RawMessage(`{"id":1,"title":"Hello"}`))
func (r *Response) AssertJSON(want any) *Response {
	r.tb.Helper()
	got, ok := r.document()
	if !ok {
		return r
	}
	w, ok := r.decodeWant(want)
	if !ok {
		return r
	}
	if !same(got, w) {
		r.fail("the body is not the document expected\ngot:  %s\nwant: %s", pretty(got), pretty(w))
	}
	return r
}

// AssertJSONSubset asserts the body contains this document.
//
// An object matches when every member of want is present and matches, so extra
// members in the response are fine. An array has to be the same length and
// match element by element, since an assertion that a list contains something
// somewhere is one that survives the list being wrong.
func (r *Response) AssertJSONSubset(want any) *Response {
	r.tb.Helper()
	got, ok := r.document()
	if !ok {
		return r
	}
	w, ok := r.decodeWant(want)
	if !ok {
		return r
	}
	if !contains(got, w) {
		r.fail("the body does not contain the document expected\ngot:  %s\nwant: %s", pretty(got), pretty(w))
	}
	return r
}

// AssertJSONPath asserts the value at a path.
//
//	res.AssertJSONPath("$.data.title", "Hello")
//	res.AssertJSONPath("$.data.tags[0]", "go")
//	res.AssertJSONPath("$.meta.total", 42)
//
// A number compares by value rather than by type, so 42 matches whether the
// document holds an integer or a float, and an id too large for a float64
// still compares exactly.
func (r *Response) AssertJSONPath(path string, want any) *Response {
	r.tb.Helper()
	got, ok := r.at(path)
	if !ok {
		return r
	}
	w, ok := r.decodeWant(want)
	if !ok {
		return r
	}
	if !same(got, w) {
		r.fail("%s is %s, want %s", path, pretty(got), pretty(w))
	}
	return r
}

// AssertJSONMissing asserts there is nothing at a path, which is the assertion
// for a field that is meant to be filtered out of a response.
func (r *Response) AssertJSONMissing(path string) *Response {
	r.tb.Helper()
	doc, ok := r.document()
	if !ok {
		return r
	}
	if got, err := evaluate(doc, path); err == nil {
		r.fail("%s is %s, want it absent", path, pretty(got))
	}
	return r
}

// AssertJSONCount asserts how many elements or members are at a path.
func (r *Response) AssertJSONCount(path string, want int) *Response {
	r.tb.Helper()
	got, ok := r.at(path)
	if !ok {
		return r
	}
	switch v := got.(type) {
	case []any:
		if len(v) != want {
			r.fail("%s has %d elements, want %d", path, len(v), want)
		}
	case map[string]any:
		if len(v) != want {
			r.fail("%s has %d members, want %d", path, len(v), want)
		}
	default:
		r.fail("%s is %s, which has nothing to count", path, describe(got))
	}
	return r
}

// document decodes the body once and hands it back on every call after.
func (r *Response) document() (any, bool) {
	r.tb.Helper()
	if r.parsed != nil || r.failed {
		return r.parsed, !r.failed
	}
	v, err := decodeJSON(r.body)
	if err != nil {
		r.failed = true
		r.fail("the body is not JSON: %v", err)
		return nil, false
	}
	r.parsed = v
	return v, true
}

// at is document plus a path, since every path assertion starts the same way.
func (r *Response) at(path string) (any, bool) {
	r.tb.Helper()
	doc, ok := r.document()
	if !ok {
		return nil, false
	}
	got, err := evaluate(doc, path)
	if err != nil {
		r.fail("%v\nthe body is %s", err, pretty(doc))
		return nil, false
	}
	return got, true
}

// decodeWant puts the expected value through the same encoding as the response,
// so that the comparison is between two documents and an int, an int64 and a
// float64 are the same number.
func (r *Response) decodeWant(want any) (any, bool) {
	r.tb.Helper()
	v, err := normalizeValue(want)
	if err != nil {
		r.fail("the expected value will not encode as JSON: %v", err)
		return nil, false
	}
	return v, true
}

// now is the fixture's time, for the assertions that need to know whether
// something is in the past. It is the fake clock rather than the real one, so a
// handler that set an expiry from the same clock is judged against it.
func (r *Response) now() time.Time { return r.app.Clock().Now() }

// fail reports one assertion. The first one on a response prints the exchange
// with it, since that is what the reader needs and hunting for it in the
// scrollback is wasted time. Later ones do not, since it is already there.
func (r *Response) fail(format string, args ...any) {
	r.tb.Helper()
	msg := fmt.Sprintf("%s %s: %s", r.req.Method, r.req.URL.RequestURI(), fmt.Sprintf(format, args...))
	if r.shown {
		r.tb.Error(msg)
		return
	}
	r.shown = true
	r.tb.Errorf("%s\n%s", msg, indent(r.exchange()))
}

func headerNames(h http.Header) string {
	if len(h) == 0 {
		return "no headers were sent"
	}
	return strings.Join(slices.Sorted(maps.Keys(h)), ", ") + " were"
}

func cookieNames(cs []*http.Cookie) string {
	if len(cs) == 0 {
		return "none"
	}
	names := make([]string, 0, len(cs))
	for _, c := range cs {
		names = append(names, c.Name)
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}
