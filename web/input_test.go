package web

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQueryReadsTheQueryString(t *testing.T) {
	r := httptest.NewRequest("GET", "/?q=hello&tag=a&tag=b&empty=", nil)
	serve(t, r, func(c *Ctx) error {
		if c.Query("q") != "hello" {
			t.Errorf("q is %q, want hello", c.Query("q"))
		}
		if c.Query("nothing") != "" {
			t.Errorf("a key that was not sent is %q", c.Query("nothing"))
		}
		if c.QueryDefault("q", "fallback") != "hello" {
			t.Error("QueryDefault used the fallback for a key that was sent")
		}
		if c.QueryDefault("nothing", "fallback") != "fallback" {
			t.Error("QueryDefault did not use the fallback")
		}
		if c.QueryDefault("empty", "fallback") != "fallback" {
			t.Error("QueryDefault did not treat an empty value as missing")
		}
		if got := c.QueryAll("tag"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Errorf("QueryAll(tag) is %v, want [a b]", got)
		}
		if got := c.QueryAll("nothing"); got != nil {
			t.Errorf("QueryAll for a key that was not sent is %v, want nil", got)
		}
		return nil
	})
}

func TestHasAndFilledTellApartClearedAndAbsent(t *testing.T) {
	r := httptest.NewRequest("GET", "/?set=yes&empty=", nil)
	serve(t, r, func(c *Ctx) error {
		for _, tc := range []struct {
			key    string
			has    bool
			filled bool
		}{
			{"set", true, true},
			{"empty", true, false},
			{"absent", false, false},
		} {
			if got := c.Has(tc.key); got != tc.has {
				t.Errorf("Has(%s) is %v, want %v", tc.key, got, tc.has)
			}
			if got := c.Filled(tc.key); got != tc.filled {
				t.Errorf("Filled(%s) is %v, want %v", tc.key, got, tc.filled)
			}
		}
		return nil
	})
}

func TestFormReadsAnUrlencodedBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/?from=query", strings.NewReader("name=ada&note="))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	serve(t, r, func(c *Ctx) error {
		if c.Form("name") != "ada" {
			t.Errorf("name is %q, want ada", c.Form("name"))
		}
		if c.Form("from") != "query" {
			t.Errorf("the query string is not in the form, from is %q", c.Form("from"))
		}
		if !c.Has("note") || c.Filled("note") {
			t.Error("a field that was sent empty should be present and not filled")
		}
		return nil
	})
}

func TestFormReadsAMultipartBody(t *testing.T) {
	var body strings.Builder
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("name", "ada"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/", strings.NewReader(body.String()))
	r.Header.Set("Content-Type", mw.FormDataContentType())
	serve(t, r, func(c *Ctx) error {
		if c.Form("name") != "ada" {
			t.Errorf("name is %q, want ada", c.Form("name"))
		}
		return nil
	})
}

func TestFormOnABodyThatWillNotParseIsEmptyRatherThanASecondAttempt(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("this is not a form"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=nope")
	serve(t, r, func(c *Ctx) error {
		if c.Form("name") != "" {
			t.Errorf("a body that will not parse gave the field %q", c.Form("name"))
		}
		if c.Has("name") {
			t.Error("a body that will not parse reported a field")
		}
		return nil
	})
}

func TestHeaderAndCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Thing", "value")
	r.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
	serve(t, r, func(c *Ctx) error {
		if c.Header("x-thing") != "value" {
			t.Errorf("X-Thing is %q, want value", c.Header("x-thing"))
		}
		if c.Header("X-Missing") != "" {
			t.Error("a header that was not sent has a value")
		}
		if v, ok := c.Cookie("session"); !ok || v != "abc" {
			t.Errorf("the session cookie is %q, %v, want abc, true", v, ok)
		}
		if _, ok := c.Cookie("nothing"); ok {
			t.Error("a cookie that was not sent was found")
		}
		return nil
	})
}

func TestBearer(t *testing.T) {
	for _, tc := range []struct {
		header string
		token  string
		ok     bool
	}{
		{"Bearer abc", "abc", true},
		{"bearer abc", "abc", true},
		{"BEARER   abc  ", "abc", true},
		{"Bearer ", "", false},
		{"Basic abc", "", false},
		{"abc", "", false},
		{"", "", false},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		if tc.header != "" {
			r.Header.Set("Authorization", tc.header)
		}
		serve(t, r, func(c *Ctx) error {
			token, ok := c.Bearer()
			if token != tc.token || ok != tc.ok {
				t.Errorf("Authorization %q gave %q, %v, want %q, %v", tc.header, token, ok, tc.token, tc.ok)
			}
			return nil
		})
	}
}

func TestIsAJAX(t *testing.T) {
	for header, want := range map[string]bool{
		"XMLHttpRequest": true,
		"xmlhttprequest": true,
		"fetch":          false,
		"":               false,
	} {
		r := httptest.NewRequest("GET", "/", nil)
		if header != "" {
			r.Header.Set("X-Requested-With", header)
		}
		serve(t, r, func(c *Ctx) error {
			if got := c.IsAJAX(); got != want {
				t.Errorf("X-Requested-With %q gave %v, want %v", header, got, want)
			}
			return nil
		})
	}
}

func TestWantsJSON(t *testing.T) {
	for _, tc := range []struct {
		accept string
		body   string
		want   bool
	}{
		{accept: "application/json", want: true},
		{accept: "application/vnd.api+json", want: true},
		{accept: "application/json, text/plain, */*", want: true},
		{accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", want: false},
		{accept: "application/json;q=0.9, text/html", want: false},
		{accept: "text/html;q=0.8, application/json;q=0.9", want: true},
		{accept: "application/json, text/html", want: true},
		{accept: "*/*", want: false},
		{accept: "text/plain", want: false},
		{accept: "((broken))", want: false},
		{accept: "application/json;q=nonsense", want: false},
		{accept: "", body: "application/json", want: true},
		{accept: "", body: "application/x-www-form-urlencoded", want: false},
		{accept: "", body: "", want: false},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		if tc.accept != "" {
			r.Header.Set("Accept", tc.accept)
		}
		if tc.body != "" {
			r.Header.Set("Content-Type", tc.body)
		}
		serve(t, r, func(c *Ctx) error {
			if got := c.WantsJSON(); got != tc.want {
				t.Errorf("Accept %q with Content-Type %q gave %v, want %v", tc.accept, tc.body, got, tc.want)
			}
			return nil
		})
	}
}

func TestBodyBytesReadsOnce(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("the whole body"))
	serve(t, r, func(c *Ctx) error {
		first, err := c.BodyBytes()
		if err != nil {
			t.Fatal(err)
		}
		if string(first) != "the whole body" {
			t.Errorf("the body is %q", first)
		}
		second, err := c.BodyBytes()
		if err != nil {
			t.Fatal(err)
		}
		if string(second) != string(first) {
			t.Errorf("the second read gave %q, want %q", second, first)
		}
		return nil
	})
}

func TestBodyBytesReportsAReaderThatFailed(t *testing.T) {
	r := httptest.NewRequest("POST", "/", errReader{})
	serve(t, r, func(c *Ctx) error {
		if _, err := c.BodyBytes(); err == nil {
			t.Error("a body that will not read came back with no error")
		}
		return nil
	})
}

func TestBodyIsTheReaderTheServerGave(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("bytes"))
	serve(t, r, func(c *Ctx) error {
		if c.Body() != c.r.Body {
			t.Error("Body is not the request's body")
		}
		return nil
	})
}

// errReader is a body that fails on the first read.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errBroken }

var errBroken = &brokenError{}

type brokenError struct{}

func (*brokenError) Error() string { return "the connection went away" }
