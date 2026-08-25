package mw

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// posted serves one form post through MethodOverride and reports the method the
// handler saw.
func posted(tb testing.TB, form url.Values, edit func(*http.Request)) (string, *http.Request) {
	tb.Helper()

	r := httptest.NewRequest("POST", "https://example.com/posts/7", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://example.com")
	if edit != nil {
		edit(r)
	}

	var seen *http.Request
	MethodOverride()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = r
	})).ServeHTTP(httptest.NewRecorder(), r)
	return seen.Method, seen
}

func TestAFormCanSayItMeantSomethingElse(t *testing.T) {
	for _, want := range []string{"PUT", "PATCH", "DELETE"} {
		t.Run(want, func(t *testing.T) {
			got, _ := posted(t, url.Values{methodField: {strings.ToLower(want)}}, nil)
			if got != want {
				t.Errorf("the handler saw %s, want %s", got, want)
			}
		})
	}
}

func TestTheOtherFieldsSurviveTheOverride(t *testing.T) {
	_, r := posted(t, url.Values{methodField: {"put"}, "title": {"a new title"}}, nil)

	if got := r.PostForm.Get("title"); got != "a new title" {
		t.Errorf("title is %q, and the handler reads the form the middleware already parsed", got)
	}
}

func TestAMethodAFormCouldAlreadySendIsNotAnOverride(t *testing.T) {
	cases := []string{"GET", "HEAD", "POST", "OPTIONS", "TRACE", "CONNECT", "BREW", ""}

	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if got, _ := posted(t, url.Values{methodField: {in}}, nil); got != "POST" {
				t.Errorf("_method=%q made it a %s", in, got)
			}
		})
	}
}

// TestOnlyAPostCanBeOverridden keeps a link out of it. A GET carrying
// _method=delete is a URL somebody can be sent, and a crawler will follow it.
func TestOnlyAPostCanBeOverridden(t *testing.T) {
	r := httptest.NewRequest("GET", "https://example.com/posts/7?_method=delete", nil)
	r.Header.Set("Origin", "https://example.com")

	var seen string
	MethodOverride()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = r.Method
	})).ServeHTTP(httptest.NewRecorder(), r)

	if seen != "GET" {
		t.Errorf("a GET with _method in the query became a %s", seen)
	}
}

func TestABodyThatIsNotAFormIsLeftAlone(t *testing.T) {
	cases := map[string]string{
		"json":      "application/json",
		"multipart": "multipart/form-data; boundary=x",
		"nothing":   "",
	}

	for name, ct := range cases {
		t.Run(name, func(t *testing.T) {
			got, _ := posted(t, url.Values{methodField: {"delete"}}, func(r *http.Request) {
				if ct == "" {
					r.Header.Del("Content-Type")
					return
				}
				r.Header.Set("Content-Type", ct)
			})
			if got != "POST" {
				t.Errorf("a %s body became a %s", name, got)
			}
		})
	}
}

func TestTheContentTypeIsReadWithoutItsParameters(t *testing.T) {
	got, _ := posted(t, url.Values{methodField: {"delete"}}, func(r *http.Request) {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	})
	if got != "DELETE" {
		t.Errorf("the handler saw %s, and a charset parameter is not a different type", got)
	}
}

func TestAFormOnSomebodyElsesPageCannotOverride(t *testing.T) {
	cases := []struct {
		name string
		edit func(*http.Request)
		want string
	}{
		{
			"a matching Origin",
			func(r *http.Request) { r.Header.Set("Origin", "https://example.com") },
			"DELETE",
		},
		{
			"somebody else's Origin",
			func(r *http.Request) { r.Header.Set("Origin", "https://evil.example.net") },
			"POST",
		},
		{
			"a matching Referer and no Origin",
			func(r *http.Request) {
				r.Header.Del("Origin")
				r.Header.Set("Referer", "https://example.com/posts/7/edit")
			},
			"DELETE",
		},
		{
			"somebody else's Referer",
			func(r *http.Request) {
				r.Header.Del("Origin")
				r.Header.Set("Referer", "https://evil.example.net/trap")
			},
			"POST",
		},
		{
			"an Origin that beats a matching Referer",
			func(r *http.Request) {
				r.Header.Set("Origin", "https://evil.example.net")
				r.Header.Set("Referer", "https://example.com/posts/7/edit")
			},
			"POST",
		},
		{
			"neither, which is not a browser",
			func(r *http.Request) { r.Header.Del("Origin") },
			"DELETE",
		},
		{
			"a Referer that is not a URL",
			func(r *http.Request) {
				r.Header.Del("Origin")
				r.Header.Set("Referer", "://")
			},
			"POST",
		},
		{
			"a relative Referer",
			func(r *http.Request) {
				r.Header.Del("Origin")
				r.Header.Set("Referer", "/posts/7/edit")
			},
			"POST",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, _ := posted(t, url.Values{methodField: {"delete"}}, c.edit); got != c.want {
				t.Errorf("the handler saw %s, want %s", got, c.want)
			}
		})
	}
}

// TestTheServersRequestIsNotTouched is net/http's rule about the request a
// handler is given, and it matters here because the method is the thing being
// changed.
func TestTheServersRequestIsNotTouched(t *testing.T) {
	r := httptest.NewRequest("POST", "https://example.com/posts/7", strings.NewReader(methodField+"=delete"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://example.com")

	MethodOverride()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(httptest.NewRecorder(), r)

	if r.Method != "POST" {
		t.Errorf("the server's request is now a %s", r.Method)
	}
	if r.PostForm != nil {
		t.Error("the server's request has a parsed form on it")
	}
}

// TestABodyThatCannotBeParsedStillReachesTheHandler covers the case where the
// body was read here and the request that goes on has to be the one that was
// read, or the handler is left with nothing to read and no way to know why.
func TestABodyThatCannotBeParsedStillReachesTheHandler(t *testing.T) {
	r := httptest.NewRequest("POST", "https://example.com/posts/7", io.NopCloser(brokenReader{}))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://example.com")

	ran := false
	MethodOverride()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ran = true
		if r.Method != "POST" {
			t.Errorf("a body that could not be read became a %s", r.Method)
		}
	})).ServeHTTP(httptest.NewRecorder(), r)

	if !ran {
		t.Error("the handler did not run")
	}
}

// brokenReader is a body that fails on the first read, which is what a client
// that hung up mid request looks like from here.
type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
