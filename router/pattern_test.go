package router

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in     string
		method string
		host   string
		path   string
		segs   string // one segment per space, in the notation shown by show
		names  string
	}{
		{in: "/", path: "/", segs: "*"},
		{in: "/posts", path: "/posts", segs: "posts"},
		{in: "/posts/", path: "/posts/", segs: "posts *"},
		{in: "GET /posts", method: "GET", path: "/posts", segs: "posts"},
		{in: "GET\t/posts", method: "GET", path: "/posts", segs: "posts"},
		{in: "GET   /posts", method: "GET", path: "/posts", segs: "posts"},
		{in: "get /posts", method: "get", path: "/posts", segs: "posts"},
		{in: "M-SEARCH /x", method: "M-SEARCH", path: "/x", segs: "x"},
		{in: "example.com/posts", host: "example.com", path: "/posts", segs: "posts"},
		{in: "GET example.com/", method: "GET", host: "example.com", path: "/", segs: "*"},
		{in: "GET /posts/{id}", method: "GET", path: "/posts/{id}", segs: "posts {id}", names: "id"},
		{in: "/{a}/{b}", path: "/{a}/{b}", segs: "{a} {b}", names: "a b"},
		{in: "/files/{path...}", path: "/files/{path...}", segs: "files {path...}", names: "path"},
		{in: "/posts/{$}", path: "/posts/{$}", segs: "posts $"},
		{in: "/{$}", path: "/{$}", segs: "$"},
		{in: "/posts/{id:int}", path: "/posts/{id:int}", segs: "posts {id:int}", names: "id"},
		{in: "GET /a/{b:uuid}/c", method: "GET", path: "/a/{b:uuid}/c", segs: "a {b:uuid} c", names: "b"},

		// A literal segment holds what it matches, so the escapes are read once
		// here rather than on every request.
		{in: "/caf%C3%A9", path: "/caf%C3%A9", segs: "café"},
		{in: "/a%2Fb", path: "/a%2Fb", segs: "a/b"},
		// A segment that will not unescape is kept as it stands, and simply
		// matches nothing.
		{in: "/a%zz", path: "/a%zz", segs: "a%zz"},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			p, err := parse(c.in, builtin)
			if err != nil {
				t.Fatalf("parse(%q): %v", c.in, err)
			}
			if p.String() != c.in {
				t.Errorf("String() = %q, want %q", p, c.in)
			}
			if p.method != c.method {
				t.Errorf("method = %q, want %q", p.method, c.method)
			}
			if p.host != c.host {
				t.Errorf("host = %q, want %q", p.host, c.host)
			}
			if got := p.path(); got != c.path {
				t.Errorf("path() = %q, want %q", got, c.path)
			}
			if got := show(p); got != c.segs {
				t.Errorf("segments = %q, want %q", got, c.segs)
			}
			if got := strings.Join(p.names(), " "); got != c.names {
				t.Errorf("names() = %q, want %q", got, c.names)
			}
		})
	}
}

// show writes the segments of a pattern the way the table above spells them, so
// that a failure names the shape rather than a struct dump.
func show(p *pattern) string {
	out := make([]string, 0, len(p.segs))
	for _, s := range p.segs {
		switch {
		case s.multi && s.s == "":
			out = append(out, "*")
		case s.multi:
			out = append(out, "{"+s.s+"...}")
		case s.wild && s.con != "":
			out = append(out, "{"+s.s+":"+s.con+"}")
		case s.wild:
			out = append(out, "{"+s.s+"}")
		case s.s == "/":
			out = append(out, "$")
		default:
			out = append(out, s.s)
		}
	}
	return strings.Join(out, " ")
}

func TestParseRefuses(t *testing.T) {
	// The wording of each of these is checked by the golden corpus under
	// testdata/diag. This table is about which patterns are turned down.
	bad := []string{
		"",
		" ",
		"GET",
		"GET ",
		"GET\tPOST /x",
		"G ET /x",
		"GE\x00T /x",
		"posts",
		"example.com",
		"{host}/x",
		"/a//b",
		"/a/./b",
		"/a/../b",
		"/a/..",
		"/a/.",
		"GET /a//b",
		"/x{a}",
		"/{a}x",
		"/}a{",
		"/{$}/x",
		"/{a...}/x",
		"/{a...:int}",
		"/{}",
		"/{...}",
		"/{:int}",
		"/{a-b}",
		"/{1a}",
		"/{a b}",
		"/{a}/{a}",
		"/{a}/b/{a...}",
		"/{a:nope}",
		"/{a:}x",
	}
	for _, s := range bad {
		t.Run(s, func(t *testing.T) {
			p, err := parse(s, builtin)
			if err == nil {
				t.Fatalf("parse(%q) gave %v, want an error", s, p)
			}
			if !strings.Contains(err.Error(), "bad pattern") {
				t.Errorf("parse(%q) = %v, want it to name the pattern", s, err)
			}
		})
	}
}

func TestCleanPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "/"},
		{"/", "/"},
		{"a", "/a"},
		{"/a", "/a"},
		{"/a/", "/a/"},
		{"/a//b", "/a/b"},
		{"//", "/"},
		{"/a/./b", "/a/b"},
		{"/a/../b", "/b"},
		{"/a/..", "/"},
		{"/a/.", "/a"},
		{"/a/b/../", "/a/"},
		{"/../a", "/a"},
		{"/a/b/c", "/a/b/c"},
	}
	for _, c := range cases {
		if got := cleanPath(c.in); got != c.want {
			t.Errorf("cleanPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsName(t *testing.T) {
	for _, s := range []string{"a", "_", "Id", "a1", "_1", "héllo"} {
		if !isName(s) {
			t.Errorf("isName(%q) = false", s)
		}
	}
	for _, s := range []string{"", "1", "1a", "a-b", "a.b", "a b", "a{"} {
		if isName(s) {
			t.Errorf("isName(%q) = true", s)
		}
	}
}

func TestIsMethod(t *testing.T) {
	for _, s := range []string{"GET", "get", "M-SEARCH", "X_1", "!#$%&'*+-.^_`|~"} {
		if !isMethod(s) {
			t.Errorf("isMethod(%q) = false", s)
		}
	}
	for _, s := range []string{"", "GET POST", "GET\t", "GE(T", "GET/", "GE\x00T", "GÉT"} {
		if isMethod(s) {
			t.Errorf("isMethod(%q) = true", s)
		}
	}
}
