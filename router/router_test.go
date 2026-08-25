package router

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

var http200 = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

// A table with one of everything in it, used by the matching tests below.
var table = []string{
	"GET /{$}",
	"GET /posts",
	"POST /posts",
	"GET /posts/new",
	"GET /posts/{id:int}",
	"GET /posts/{slug:slug}",
	"GET /posts/{id:int}/edit",
	"GET /posts/{id:int}/comments/{comment:int}",
	"GET /files/{path...}",
	"GET /docs/{$}",
	"GET /docs/{page}",
	"/any",
	"GET admin.example.com/",
	"GET admin.example.com/posts",
}

func TestLookup(t *testing.T) {
	r := New()
	for _, p := range table {
		r.Handle(p, http200)
	}

	cases := []struct {
		method, host, path string
		want               string // the pattern that should win, or empty for no match
		params             string
	}{
		{"GET", "", "/", "GET /{$}", ""},
		{"GET", "", "/posts", "GET /posts", ""},
		{"POST", "", "/posts", "POST /posts", ""},
		{"HEAD", "", "/posts", "GET /posts", ""},
		{"GET", "", "/posts/new", "GET /posts/new", ""},
		{"GET", "", "/posts/7", "GET /posts/{id:int}", "id=7"},
		{"GET", "", "/posts/hello-world", "GET /posts/{slug:slug}", "slug=hello-world"},
		{"GET", "", "/posts/Hello", "", ""},
		{"GET", "", "/posts/7/edit", "GET /posts/{id:int}/edit", "id=7"},
		{"GET", "", "/posts/7/comments/9", "GET /posts/{id:int}/comments/{comment:int}", "id=7 comment=9"},
		{"GET", "", "/posts/7/comments", "", ""},
		{"GET", "", "/files/a/b/c", "GET /files/{path...}", "path=a/b/c"},
		{"GET", "", "/files/", "GET /files/{path...}", "path="},
		{"GET", "", "/files", "", ""},
		{"GET", "", "/docs/", "GET /docs/{$}", ""},
		{"GET", "", "/docs/intro", "GET /docs/{page}", "page=intro"},
		// {page} takes "a", the rest fails, and the walk comes back and unwinds
		// the value it had pushed.
		{"GET", "", "/docs/a/b", "", ""},
		{"DELETE", "", "/any", "/any", ""},
		{"GET", "", "/any", "/any", ""},
		// HEAD tries HEAD, then GET, then the patterns with no method at all.
		{"HEAD", "", "/any", "/any", ""},
		{"GET", "admin.example.com", "/posts", "GET admin.example.com/posts", ""},
		{"POST", "admin.example.com", "/posts", "POST /posts", ""},
		{"GET", "other.example.com", "/posts", "GET /posts", ""},
		{"GET", "", "/nothing/here", "", ""},

		// The escapes come off one segment at a time, so a slash inside a
		// segment stays inside it.
		{"GET", "", "/docs/a%2Fb", "GET /docs/{page}", "page=a/b"},
		{"GET", "", "/posts/caf%C3%A9", "", ""},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.host+c.path, func(t *testing.T) {
			rt, ps, ok := r.Lookup(c.method, c.host, c.path)
			if !ok {
				if c.want != "" {
					t.Fatalf("no match, want %q", c.want)
				}
				return
			}
			if c.want == "" {
				t.Fatalf("matched %q, want no match", rt.Info().Pattern)
			}
			if got := rt.Info().Pattern; got != c.want {
				t.Fatalf("matched %q, want %q", got, c.want)
			}
			if got := params(ps); got != c.params {
				t.Errorf("params = %q, want %q", got, c.params)
			}
		})
	}
}

func params(ps Params) string {
	var out []string
	for name, value := range ps.All() {
		out = append(out, name+"="+value)
	}
	return strings.Join(out, " ")
}

// The whole point of the values array is that a match costs nothing, so this is
// the test that keeps it that way.
func TestMatchingDoesNotAllocate(t *testing.T) {
	r := New()
	for _, p := range table {
		r.Handle(p, http200)
	}
	paths := []string{"/", "/posts/7/comments/9", "/posts/new", "/files/a/b/c", "/nothing"}

	got := testing.AllocsPerRun(100, func() {
		for _, p := range paths {
			r.Lookup("GET", "", p)
		}
	})
	if got != 0 {
		t.Errorf("Lookup allocated %v times per run", got)
	}
}

func TestServeHTTP(t *testing.T) {
	r := New()
	for _, p := range table {
		r.Handle(p, named(p))
	}

	cases := []struct {
		method, target string
		code           int
		body           string
		allow          string
	}{
		{"GET", "/posts", 200, "GET /posts", ""},
		{"HEAD", "/posts", 200, "GET /posts", ""},
		{"PUT", "/posts", 405, "", "GET, HEAD, OPTIONS, POST"},
		{"OPTIONS", "/posts", 204, "", "GET, HEAD, OPTIONS, POST"},
		{"GET", "/nothing", 404, "", ""},
		{"OPTIONS", "/nothing", 404, "", ""},
		{"PUT", "/posts/7", 405, "", "GET, HEAD, OPTIONS"},
		{"PUT", "/any", 200, "/any", ""},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.target, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(c.method, c.target, nil))

			if w.Code != c.code {
				t.Errorf("code = %d, want %d", w.Code, c.code)
			}
			if c.body != "" && w.Body.String() != c.body {
				t.Errorf("body = %q, want %q", w.Body.String(), c.body)
			}
			if got := w.Header().Get("Allow"); got != c.allow {
				t.Errorf("Allow = %q, want %q", got, c.allow)
			}
		})
	}
}

// named is a handler that writes the pattern it was registered under, so a test
// can see which route ran.
func named(pattern string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(pattern))
	})
}

func TestMatchedInsideAHandler(t *testing.T) {
	r := New()
	want := r.Handle("GET /posts/{id:int}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rt, ps, ok := Matched(req)
		if !ok {
			t.Error("Matched said the request was not routed")
			return
		}
		if rt.Info().Name != "posts.show" {
			t.Errorf("route name = %q, want %q", rt.Info().Name, "posts.show")
		}
		if got := ps.Get("id"); got != "7" {
			t.Errorf("id = %q, want %q", got, "7")
		}
		if got := PathValue(req, "id"); got != "7" {
			t.Errorf("PathValue(id) = %q, want %q", got, "7")
		}
		// The standard field stays empty, which is the documented trade.
		if got := req.PathValue("id"); got != "" {
			t.Errorf("req.PathValue(id) = %q, want it empty", got)
		}
	})).Name("posts.show")

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/posts/7", nil))

	if got, ok := r.Named("posts.show"); !ok || got != want {
		t.Errorf("Named(posts.show) = %v, %v, want the route it was set on", got, ok)
	}
	if _, ok := r.Named("nothing"); ok {
		t.Error("Named found a name nobody set")
	}
}

func TestMatchedOnAnUnroutedRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/posts/7", nil)
	if _, _, ok := Matched(req); ok {
		t.Error("Matched found a route on a request nothing routed")
	}
	if got := PathValue(req, "id"); got != "" {
		t.Errorf("PathValue = %q, want it empty", got)
	}
}

func TestHandleFunc(t *testing.T) {
	r := New()
	r.HandleFunc("GET /x", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ran")) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	if w.Body.String() != "ran" {
		t.Errorf("body = %q, want %q", w.Body.String(), "ran")
	}
}

func TestRoutes(t *testing.T) {
	r := New()
	r.Handle("GET admin.example.com/posts/{id:int}/edit", http200).Name("posts.edit").Meta("auth", "admin")
	r.Handle("/files/{path...}", http200)

	got := r.Routes()
	if len(got) != 2 {
		t.Fatalf("Routes() has %d entries, want 2", len(got))
	}

	first := got[0]
	if first.Method != "GET" || first.Host != "admin.example.com" || first.Path != "/posts/{id:int}/edit" {
		t.Errorf("first route taken apart wrong: %+v", first)
	}
	if first.Pattern != "GET admin.example.com/posts/{id:int}/edit" || first.Name != "posts.edit" {
		t.Errorf("first route: %+v", first)
	}
	if !slices.Equal(first.Params, []string{"id"}) {
		t.Errorf("Params = %v, want [id]", first.Params)
	}
	if !strings.Contains(first.Source, "router_test.go:") {
		t.Errorf("Source = %q, want the line it was registered at", first.Source)
	}

	second := got[1]
	if second.Method != "" || second.Host != "" || second.Path != "/files/{path...}" {
		t.Errorf("second route taken apart wrong: %+v", second)
	}

	// A trailing slash makes a wildcard with no name, and it is not a parameter.
	r.Handle("GET /tree/", http200)
	if p := r.Routes()[2].Params; len(p) != 0 {
		t.Errorf("Params = %v, want none", p)
	}
}

func TestMeta(t *testing.T) {
	r := New()
	rt := r.Handle("GET /x", http200).Meta("auth", "admin").Meta("rate", 10)

	if got := rt.Value("auth"); got != "admin" {
		t.Errorf("Value(auth) = %v, want admin", got)
	}
	if got := rt.Value("rate"); got != 10 {
		t.Errorf("Value(rate) = %v, want 10", got)
	}
	if got := rt.Value("nothing"); got != nil {
		t.Errorf("Value(nothing) = %v, want nil", got)
	}
	if rt.Handler() == nil {
		t.Error("Handler() came back nil")
	}
}

func TestNameCanBeMoved(t *testing.T) {
	r := New()
	a := r.Handle("GET /a", http200).Name("x")
	// Setting the same name on the same route again is not a clash.
	a.Name("x")
	if got, _ := r.Named("x"); got != a {
		t.Fatal("Named did not find the route the name was set on")
	}
	// Renaming it frees the old name.
	a.Name("y")
	if _, ok := r.Named("x"); ok {
		t.Error("the old name is still taken")
	}
	if got, _ := r.Named("y"); got != a {
		t.Error("Named did not find the route under its new name")
	}
	r.Handle("GET /b", http200).Name("x")
}

func TestRedirectTrailingSlash(t *testing.T) {
	r := New(RedirectTrailingSlash())
	r.Handle("GET /posts", http200)
	r.Handle("GET /docs/", http200)

	cases := []struct {
		target string
		code   int
		to     string
	}{
		{"/posts/", 308, "/posts"},
		{"/docs", 308, "/docs/"},
		{"/posts", 200, ""},
		{"/docs/", 200, ""},
		{"/nothing/", 404, ""},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", c.target, nil))
		if w.Code != c.code {
			t.Errorf("GET %s: code = %d, want %d", c.target, w.Code, c.code)
		}
		if got := w.Header().Get("Location"); got != c.to {
			t.Errorf("GET %s: Location = %q, want %q", c.target, got, c.to)
		}
	}

	// Off by default, which is the point.
	plain := New()
	plain.Handle("GET /posts", http200)
	w := httptest.NewRecorder()
	plain.ServeHTTP(w, httptest.NewRequest("GET", "/posts/", nil))
	if w.Code != 404 {
		t.Errorf("with no option, GET /posts/ = %d, want 404", w.Code)
	}
}

func TestRedirectCleanPath(t *testing.T) {
	r := New(RedirectCleanPath())
	r.Handle("GET /posts/new", http200)

	cases := []struct {
		target string
		code   int
		to     string
	}{
		{"/posts//new", 308, "/posts/new"},
		{"/posts/./new", 308, "/posts/new"},
		{"/posts/7/../new", 308, "/posts/new"},
		{"/posts/new", 200, ""},
		{"/posts//nothing", 404, ""},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://example.com"+c.target, nil)
		req.URL.Path, req.URL.RawPath = c.target, c.target
		w.Code = 200
		r.ServeHTTP(w, req)
		if w.Code != c.code {
			t.Errorf("GET %s: code = %d, want %d", c.target, w.Code, c.code)
		}
		if got := w.Header().Get("Location"); got != c.to {
			t.Errorf("GET %s: Location = %q, want %q", c.target, got, c.to)
		}
	}
}

func TestNotFoundAndMethodNotAllowedHandlers(t *testing.T) {
	r := New(
		NotFound(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "nothing there", http.StatusNotFound)
		})),
		MethodNotAllowed(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// The Allow header is already set, which is the contract.
			http.Error(w, "try "+w.Header().Get("Allow"), http.StatusMethodNotAllowed)
		})),
	)
	r.Handle("GET /x", http200)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/nothing", nil))
	if w.Code != 404 || !strings.Contains(w.Body.String(), "nothing there") {
		t.Errorf("404 handler: %d %q", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/x", nil))
	if w.Code != 405 || !strings.Contains(w.Body.String(), "try GET, HEAD, OPTIONS") {
		t.Errorf("405 handler: %d %q", w.Code, w.Body.String())
	}
}

func TestConstrain(t *testing.T) {
	even := func(s string) bool { return isInt(s) && (s[len(s)-1]-'0')%2 == 0 }
	r := New(Constrain("even", even))
	r.Handle("GET /n/{v:even}", named("even"))
	r.Handle("GET /n/{v}", named("any"))

	for _, c := range []struct{ path, want string }{
		{"/n/4", "even"},
		{"/n/5", "any"},
		{"/n/x", "any"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", c.path, nil))
		if w.Body.String() != c.want {
			t.Errorf("GET %s went to %q, want %q", c.path, w.Body.String(), c.want)
		}
	}
}

func TestConstrainRefuses(t *testing.T) {
	cases := []struct {
		name string
		opt  Option
	}{
		{"a built-in name", Constrain("int", isInt)},
		{"nothing to check with", Constrain("even", nil)},
	}
	for _, c := range cases {
		if _, err := open(c.opt); err == nil {
			t.Errorf("%s: open took it", c.name)
		}
	}
	if _, err := open(Constrain("even", isInt), Constrain("even", isInt)); err == nil {
		t.Error("the same name twice: open took it")
	}
}

func TestNewPanicsOnABadOption(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New took an option that does not work")
		}
	}()
	New(Constrain("int", isInt))
}

func TestRegisterRefuses(t *testing.T) {
	// The wording of each of these is in testdata/diag. This is about what is
	// turned down.
	cases := []struct {
		name  string
		table []string
	}{
		{"a pattern that will not parse", []string{"GET /{"}},
		{"the same pattern twice", []string{"GET /a", "GET /a"}},
		{"a pattern that overlaps another", []string{"/posts/{id}/edit", "/{kind}/latest/edit"}},
		{"more methods against fewer paths", []string{"GET /a/{x}", "/a/b"}},
		{"the same, the other way round", []string{"/a/b", "GET /a/{x}"}},
		{"a constraint nobody registered", []string{"GET /a/{x:nope}"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := New()
			var err error
			for _, p := range c.table {
				if _, err = r.register(p, http200, "app/routes.go:1"); err != nil {
					break
				}
			}
			if err == nil {
				t.Errorf("registered %q with no complaint", c.table)
			}
		})
	}
}

func TestHandlePanics(t *testing.T) {
	for _, fn := range []func(){
		func() { New().Handle("GET /{", http200) },
		func() { New().HandleFunc("GET /{", func(http.ResponseWriter, *http.Request) {}) },
		func() { New().Handle("GET /a", nil) },
		func() {
			r := New()
			r.Handle("GET /a", http200).Name("x")
			r.Handle("GET /b", http200).Name("x")
		},
	} {
		func() {
			defer func() {
				v := recover()
				if v == nil {
					t.Error("no panic")
					return
				}
				if s, ok := v.(string); !ok || !strings.HasPrefix(s, "router: ") {
					t.Errorf("panicked with %v, want a message this package names itself in", v)
				}
			}()
			fn()
		}()
	}
}

func TestConflictNamesWhereTheOtherOneIs(t *testing.T) {
	r := New()
	if _, err := r.register("GET /a/{x}", http200, "app/routes.go:12"); err != nil {
		t.Fatal(err)
	}
	_, err := r.register("/a/b", http200, "app/routes.go:13")
	if err == nil {
		t.Fatal("registered a conflicting pattern")
	}
	if !strings.Contains(err.Error(), "app/routes.go:12") {
		t.Errorf("%v does not say where the other pattern is", err)
	}
}

func TestRegisteringAfterAMatch(t *testing.T) {
	r := New()
	r.Handle("GET /a", named("a"))
	if _, _, ok := r.Lookup("GET", "", "/a"); !ok {
		t.Fatal("/a did not match")
	}

	// The tree is thrown away and made again, so the second route is there.
	r.Handle("GET /b", named("b"))
	if _, _, ok := r.Lookup("GET", "", "/b"); !ok {
		t.Fatal("/b did not match after the tree had already been built")
	}
	if _, _, ok := r.Lookup("GET", "", "/a"); !ok {
		t.Fatal("/a stopped matching")
	}
}

func TestWhereFallsBack(t *testing.T) {
	if got := where(1000); got != "an unknown place" {
		t.Errorf("where(1000) = %q, want it to say it does not know", got)
	}
}

// The tree is thrown away by a registration and made again by the next request,
// so registering while requests are being served has to be safe. Run this under
// the race detector for it to be worth anything.
func TestRegisteringWhileMatching(t *testing.T) {
	r := New()
	r.Handle("GET /a/{x}", http200)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 200 {
			r.Handle("GET /b"+itoa(i), http200)
		}
	}()
	for range 2000 {
		if _, _, ok := r.Lookup("GET", "", "/a/1"); !ok {
			t.Error("/a/1 stopped matching while routes were being added")
			break
		}
	}
	<-done
}

func TestHostOf(t *testing.T) {
	cases := []struct{ in, want string }{
		{"example.com", "example.com"},
		{"example.com:8080", "example.com"},
		{"[::1]:8080", "::1"},
		{"[::1]", "[::1]"},
		{"example.com:", "example.com"},
		{"a:b:c", "a:b:c"},
		{"", ""},
	}
	for _, c := range cases {
		req := &http.Request{Host: c.in}
		if got := hostOf(req); got != c.want {
			t.Errorf("hostOf(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParams(t *testing.T) {
	r := New()
	r.Handle("GET /a/{x}/{y}", http200)
	_, ps, ok := r.Lookup("GET", "", "/a/1/2")
	if !ok {
		t.Fatal("no match")
	}

	if ps.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", ps.Len())
	}
	if got := ps.At(0); got != (Param{"x", "1"}) {
		t.Errorf("At(0) = %v, want {x 1}", got)
	}
	if got := ps.At(1); got != (Param{"y", "2"}) {
		t.Errorf("At(1) = %v, want {y 2}", got)
	}
	if v, ok := ps.Lookup("x"); !ok || v != "1" {
		t.Errorf("Lookup(x) = %q, %v, want 1, true", v, ok)
	}
	if v, ok := ps.Lookup("z"); ok || v != "" {
		t.Errorf("Lookup(z) = %q, %v, want an empty string and false", v, ok)
	}
	if got := ps.Get("z"); got != "" {
		t.Errorf("Get(z) = %q, want it empty", got)
	}

	// Stopping early stops the walk.
	var seen int
	for range ps.All() {
		seen++
		break
	}
	if seen != 1 {
		t.Errorf("the walk yielded %d times after a break, want 1", seen)
	}

	var zero Params
	if zero.Len() != 0 || zero.Get("x") != "" {
		t.Error("the zero Params has something in it")
	}
}

func TestParamsAtOutOfRange(t *testing.T) {
	for _, i := range []int{-1, 0, 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("At(%d) on an empty Params did not panic", i)
				}
			}()
			var ps Params
			ps.At(i)
		}()
	}
}

func TestItoa(t *testing.T) {
	for _, c := range []struct {
		in   int
		want string
	}{{0, "0"}, {7, "7"}, {-7, "-7"}, {1234567890, "1234567890"}} {
		if got := itoa(c.in); got != c.want {
			t.Errorf("itoa(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
