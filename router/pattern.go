package router

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"
	"unicode"
)

// A pattern is a registered route taken apart: the method, the host, and the
// path as segments.
//
// The path is kept as segments rather than as the string somebody wrote,
// because every question the router asks of a pattern is about one segment at a
// time. Which child of the tree it goes under, whether it can match the same
// request as another pattern, and whether a value is allowed there are all
// segment questions.
//
// Two shapes are written one way and stored another, both copied from
// net/http.ServeMux so that the two agree about what they mean:
//
//	/files/    a literal segment "files" and a nameless trailing wildcard
//	/posts/{$} a literal segment "posts" and a literal segment "/"
type pattern struct {
	str     string // what was registered, for messages and for Routes
	method  string // empty means any method
	host    string // empty means any host
	rawPath string // the path as written, from the first slash on
	segs    []segment
}

func (p *pattern) String() string { return p.str }

// last is the segment a path pattern ends with. A pattern always has at least
// one segment, since the path it was parsed from starts with a slash.
func (p *pattern) last() segment { return p.segs[len(p.segs)-1] }

// path is the pattern without the method and the host, which is what a route
// table prints in its own column.
func (p *pattern) path() string { return p.rawPath }

// names is the wildcard names in the order they appear, which is the order the
// values come back from a match in.
//
// The nameless wildcard a trailing slash turns into is left out, since a match
// records no value for it.
func (p *pattern) names() []string {
	var out []string
	for _, s := range p.segs {
		if s.wild && s.s != "" {
			out = append(out, s.s)
		}
	}
	return out
}

// A segment is one piece of a path pattern.
//
// A literal has wild false and holds the text it matches, or "/" when it came
// from {$}. A single wildcard has wild true and holds its name, and may carry a
// constraint. A trailing wildcard has both wild and multi true, and its name is
// empty when it came from a trailing slash rather than from {rest...}.
type segment struct {
	s     string
	wild  bool
	multi bool

	// con is the name written after the colon, and check is what that name
	// resolved to. Both are empty on everything except a single wildcard that
	// was given a constraint.
	con   string
	check Constraint
}

// parse takes a pattern apart, resolving the constraint names in it against cs.
//
// The syntax is [METHOD ][HOST]/[PATH], which is ServeMux's syntax, and the
// parse follows ServeMux's parse closely enough that the two accept the same
// patterns. Everything mizu adds is inside a wildcard, after a colon.
func parse(s string, cs constraints) (*pattern, error) {
	p, err := parseUnwrapped(s, cs)
	if err != nil {
		return nil, fmt.Errorf("bad pattern %q: %w", s, err)
	}
	return p, nil
}

func parseUnwrapped(s string, cs constraints) (*pattern, error) {
	if s == "" {
		return nil, errors.New("there is nothing in it to match")
	}

	method, rest, found := s, "", false
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		method, rest, found = s[:i], strings.TrimLeft(s[i+1:], " \t"), true
	}
	if !found {
		rest, method = method, ""
	}
	if method != "" && !isMethod(method) {
		return nil, fmt.Errorf("%q is not a method, and a method is what comes before the space", method)
	}

	p := &pattern{str: s, method: method}
	i := strings.IndexByte(rest, '/')
	if i < 0 {
		return nil, errors.New("there is no / in it, and the path a pattern matches starts with one")
	}
	p.host, rest = rest[:i], rest[i:]
	p.rawPath = rest
	if strings.IndexByte(p.host, '{') >= 0 {
		return nil, fmt.Errorf("%q reads as the host, so the path is missing the / it starts with", p.host)
	}
	if strings.ContainsAny(p.host, " \t") {
		return nil, fmt.Errorf("%q reads as the host and has a space in it, so the method and the path have run together", p.host)
	}
	if c := cleanPath(rest); c != rest {
		return nil, fmt.Errorf("%q is not a path a request can carry, since a request path is cleaned before it is matched: write %q", rest, c)
	}

	var seen []string
	for len(rest) > 0 {
		// Invariant: rest begins with a slash.
		rest = rest[1:]
		if rest == "" {
			// A trailing slash matches the subtree, which is the same thing a
			// {rest...} does without a name to record it under.
			p.segs = append(p.segs, segment{wild: true, multi: true})
			break
		}

		i := strings.IndexByte(rest, '/')
		if i < 0 {
			i = len(rest)
		}
		var seg string
		seg, rest = rest[:i], rest[i:]

		if strings.IndexByte(seg, '{') < 0 {
			p.segs = append(p.segs, segment{s: unescape(seg)})
			continue
		}
		if seg[0] != '{' || seg[len(seg)-1] != '}' {
			return nil, fmt.Errorf("%q is part literal and part wildcard, and a segment is one or the other", seg)
		}

		body := seg[1 : len(seg)-1]
		if body == "$" {
			if rest != "" {
				return nil, errors.New("{$} says the path ends there, so nothing can come after it")
			}
			p.segs = append(p.segs, segment{s: "/"})
			break
		}

		name, con, _ := strings.Cut(body, ":")
		name, multi := strings.CutSuffix(name, "...")
		switch {
		case multi && rest != "":
			return nil, fmt.Errorf("{%s...} takes the rest of the path, so nothing can come after it", name)
		case multi && con != "":
			return nil, fmt.Errorf("{%s...} takes the rest of the path, and a constraint reads one segment", name)
		case name == "":
			return nil, fmt.Errorf("%q has no name, and a wildcard is read back out by its name", seg)
		case !isName(name):
			return nil, fmt.Errorf("%q is not a name a wildcard can have, which is what a Go identifier looks like", name)
		case slices.Contains(seen, name):
			return nil, fmt.Errorf("%q names two wildcards, and each one is a different piece of the path", name)
		}
		seen = append(seen, name)

		out := segment{s: name, wild: true, multi: multi, con: con}
		if con != "" {
			check, ok := cs[con]
			if !ok {
				return nil, fmt.Errorf("%q is not a constraint, and these are: %s", con, strings.Join(cs.names(), ", "))
			}
			out.check = check
		}
		p.segs = append(p.segs, out)
	}
	return p, nil
}

// isName reports whether a wildcard may be called this, which is the rule
// ServeMux uses: a name is a Go identifier, so that a generator can turn it into
// a parameter without rewriting it.
func isName(s string) bool {
	for i, c := range s {
		if !unicode.IsLetter(c) && c != '_' && (i == 0 || !unicode.IsDigit(c)) {
			return false
		}
	}
	return s != ""
}

// isMethod reports whether s is shaped like a method name. Any token is
// allowed, since a method is an extension point and refusing an unknown one
// would mean this package deciding which protocols exist.
func isMethod(s string) bool {
	for i := range len(s) {
		if !isTokenByte(s[i]) {
			return false
		}
	}
	return s != ""
}

// isTokenByte is the token rule from RFC 9110 section 5.6.2.
func isTokenByte(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	}
	return strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0
}

// unescape is a path segment with its percent escapes read, or the segment as
// it stands when the escapes are not valid.
//
// A segment that will not unescape is left alone rather than refused, because
// the pattern it fails to match is the answer either way and a 404 is a better
// answer than a panic.
func unescape(s string) string {
	out, err := url.PathUnescape(s)
	if err != nil {
		return s
	}
	return out
}

// cleanPath is the path with its dot segments and double slashes taken out,
// keeping a trailing slash, which is what net/http does to a request path
// before it matches anything.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	out := path.Clean(p)
	if p[len(p)-1] == '/' && out != "/" {
		// path.Clean drops the trailing slash, and "/a/." has to come back as
		// "/a/" rather than as "/a".
		if len(p) == len(out)+1 && strings.HasPrefix(p, out) {
			out = p
		} else {
			out += "/"
		}
	}
	return out
}
