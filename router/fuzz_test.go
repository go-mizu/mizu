package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

// The router writes its own tree, so the tree has to stay the same as the one
// in the standard library.
//
// This is the test that says so. It takes a set of patterns, registers them in
// a [http.ServeMux] and in a [Router] in the same order, and asks both the same
// question. Registration has to succeed and fail in the same places, the same
// pattern has to win, and a request that matches nothing has to be a 404 in
// both and a 405 with the same Allow header in both.
//
// Three things are left out on purpose, and each is a place mizu was meant to
// differ:
//
// A pattern with a colon in it is mizu syntax and ServeMux has no opinion about
// it, so those are skipped. That is what "the shared syntax" means.
//
// A pattern whose path is not clean is skipped, and so is one whose host has a
// space in it. ServeMux takes both, on the grounds that no request can reach
// them: it cleans the path before matching, and no Host header has a space. mizu
// turns them down instead, since a pattern nothing can reach is a typo and the
// place to say so is where it was written.
//
// A redirect is skipped. ServeMux answers /tree with a redirect to /tree/ when
// the second is registered and mizu answers 404, which is what
// [RedirectTrailingSlash] is for and is checked where that is checked.
func FuzzMatchesServeMux(f *testing.F) {
	seeds := []struct{ table, method, host, path string }{
		{"GET /posts\nGET /posts/{id}\nGET /posts/new", "GET", "", "/posts/new"},
		{"GET /posts/{id}/edit\nGET /posts/{id}\n/posts/", "GET", "", "/posts/1/edit"},
		{"GET /files/{path...}\nGET /files/readme", "GET", "", "/files/a/b/c"},
		{"GET /a/{$}\nGET /a/{x}\nGET /a/", "GET", "", "/a/"},
		{"POST /posts\nGET /posts", "PUT", "", "/posts"},
		{"GET /\nGET example.com/", "GET", "example.com", "/"},
		{"HEAD /x\nGET /x", "HEAD", "", "/x"},
		{"/any\nGET /other", "DELETE", "", "/any"},
		{"GET /a/b/c\nGET /a/{x}/c\nGET /a/b/{y}", "GET", "", "/a/b/c"},
		{"GET /e/{name}", "GET", "", "/e/caf%C3%A9"},
	}
	for _, s := range seeds {
		f.Add(s.table, s.method, s.host, s.path)
	}

	f.Fuzz(func(t *testing.T, table, method, host, path string) {
		pats := understood(table)
		if len(pats) == 0 || !isMethod(method) || method == "CONNECT" {
			return
		}
		req, ok := request(method, host, path)
		if !ok {
			return
		}

		mux := http.NewServeMux()
		r := New()
		for _, p := range pats {
			muxErr := recovered(func() { mux.Handle(p, echo(p)) })
			ourErr := recovered(func() { r.Handle(p, echo(p)) })
			switch {
			case muxErr == nil && ourErr != nil:
				t.Fatalf("ServeMux took %q after %q and this router did not: %v", p, pats, ourErr)
			case muxErr != nil && ourErr == nil:
				t.Fatalf("this router took %q after %q and ServeMux did not: %v", p, pats, muxErr)
			}
		}

		want := answer(mux, req)
		if want.code == http.StatusTemporaryRedirect {
			return
		}
		if want.code != http.StatusOK {
			// ServeMux builds the Allow list of a 405 out of this path and out
			// of this path with a slash on the end, since it would have
			// redirected to the second one if the method had matched. mizu
			// answers about the path that arrived and nothing else, so where
			// the two differ this is why, and there is nothing to compare.
			if probe, ok := request(method, host, path+"/"); ok && answer(mux, probe).code != http.StatusNotFound {
				return
			}
		}
		if got := answer(r, req); got != want {
			t.Errorf("%s %s%s against %q\n got %v\nwant %v", method, host, path, pats, got, want)
		}
	})
}

// understood is the patterns in a table that both routers read the same way.
func understood(table string) []string {
	var out []string
	for line := range strings.SplitSeq(table, "\n") {
		if line == "" || strings.ContainsRune(line, ':') {
			continue
		}
		// Split it the way the parser does, so that what is skipped here is
		// exactly what mizu is stricter about.
		method, rest := "", line
		if i := strings.IndexAny(line, " \t"); i >= 0 {
			method, rest = line[:i], strings.TrimLeft(line[i+1:], " \t")
		}
		i := strings.IndexByte(rest, '/')
		if i < 0 || method == "CONNECT" {
			continue
		}
		host, p := rest[:i], rest[i:]
		if strings.ContainsAny(host, " \t") || cleanPath(p) != p {
			continue
		}
		out = append(out, line)
	}
	return out
}

// request builds one without going through httptest.NewRequest, which parses a
// request line and panics on anything the fuzzer is likely to produce.
func request(method, host, path string) (*http.Request, bool) {
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, " \t\r\n?#") {
		return nil, false
	}
	u, err := url.Parse(path)
	if err != nil {
		return nil, false
	}
	if strings.ContainsAny(host, " \t\r\n/") {
		return nil, false
	}
	return &http.Request{Method: method, URL: u, Host: host, Proto: "HTTP/1.1"}, true
}

// A reply is what a router said, cut down to the parts the two are meant to
// agree about.
type reply struct {
	code  int
	body  string // the pattern that won, for a match
	allow string // the methods a 405 named, without OPTIONS
}

func (r reply) String() string { return fmt.Sprintf("%d %q allow=%q", r.code, r.body, r.allow) }

func answer(h http.Handler, req *http.Request) reply {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req.Clone(req.Context()))

	out := reply{code: w.Code}
	if w.Code == http.StatusOK {
		out.body = w.Body.String()
	}
	if w.Code == http.StatusMethodNotAllowed {
		// mizu says OPTIONS is available and ServeMux does not, since ServeMux
		// has nothing that would answer one.
		var kept []string
		for _, m := range strings.Split(w.Header().Get("Allow"), ", ") {
			if m != http.MethodOptions && m != "" {
				kept = append(kept, m)
			}
		}
		slices.Sort(kept)
		out.allow = strings.Join(kept, ", ")
	}
	return out
}

// echo is a handler that says which pattern it was registered under, so that a
// match names the pattern that won.
func echo(pattern string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(pattern))
	})
}

// recovered runs fn and returns what it panicked with, since both routers
// report a bad or conflicting pattern by panicking.
func recovered(fn func()) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("%v", v)
		}
	}()
	fn()
	return nil
}
