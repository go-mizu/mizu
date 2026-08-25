package router

import "strings"

// The decision tree.
//
// The root branches on the host, its children branch on the method, and every
// level below that is one segment of the path. A leaf holds the route that was
// registered there.
//
// At a path level the children are tried in order of how much they say:
//
//	a literal            /posts/new
//	a constrained wildcard, in registration order   /posts/{id:uuid}
//	a bare wildcard      /posts/{id}
//	a trailing wildcard  /posts/{rest...}
//
// A failure further down comes back here and tries the next one, so
// /posts/{id}/edit still matches /posts/1/edit when /posts/new exists. That
// backtracking is what makes "the more specific pattern wins" true for whole
// patterns rather than for single segments.

// A node is one level of the tree. The same struct is a leaf and an interior
// node, which is what lets a pattern end anywhere.
type node struct {
	route *Route // set on a leaf

	lit   mapping // literal children, "/" for {$}, and "" for any host or any method
	wild  []*wildChild
	any   *node // bare single wildcard
	multi *node // trailing wildcard
}

// A wildChild is a single wildcard that carries a constraint, kept apart from
// the bare one so that a constrained pattern is tried first.
type wildChild struct {
	con   string
	check Constraint
	node  *node
}

// add puts a route in the tree at its host, its method, and its segments.
func (n *node) add(rt *Route) {
	n = n.child(rt.pat.host)
	n = n.child(rt.pat.method)
	n.addSegments(rt.pat.segs, rt)
}

func (n *node) addSegments(segs []segment, rt *Route) {
	if len(segs) == 0 {
		n.route = rt
		return
	}
	s := segs[0]
	switch {
	case s.multi:
		n.multi = &node{route: rt}
	case s.wild && s.con == "":
		if n.any == nil {
			n.any = &node{}
		}
		n.any.addSegments(segs[1:], rt)
	case s.wild:
		n.constrained(s).addSegments(segs[1:], rt)
	default:
		n.child(s.s).addSegments(segs[1:], rt)
	}
}

// constrained is the child for one constraint name, added at the end so that
// the order they are tried in is the order they were registered in.
func (n *node) constrained(s segment) *node {
	for _, w := range n.wild {
		if w.con == s.con {
			return w.node
		}
	}
	w := &wildChild{con: s.con, check: s.check, node: &node{}}
	n.wild = append(n.wild, w)
	return w.node
}

// child is the literal child under key, added if it is not there yet.
func (n *node) child(key string) *node {
	if c := n.lit.find(key); c != nil {
		return c
	}
	c := &node{}
	n.lit.add(key, c)
	return c
}

// match walks the tree for one request and returns the leaf it lands on,
// recording the wildcard values in v as it goes.
//
// A host that has patterns of its own is tried first and falls through to the
// host-less patterns when nothing under it matches, which is precedence rule 1.
func (n *node) match(host, method, path string, v *values) *node {
	if host != "" {
		if m := n.lit.find(host).matchMethod(method, path, v); m != nil {
			return m
		}
		v.reset()
	}
	return n.lit.find("").matchMethod(method, path, v)
}

func (n *node) matchMethod(method, path string, v *values) *node {
	if n == nil {
		return nil
	}
	if m := n.lit.find(method).matchPath(path, v); m != nil {
		return m
	}
	v.reset()

	// A GET route answers HEAD, which is the one place a method matches
	// something it does not spell.
	if method == "HEAD" {
		if m := n.lit.find("GET").matchPath(path, v); m != nil {
			return m
		}
		v.reset()
	}
	return n.lit.find("").matchPath(path, v)
}

func (n *node) matchPath(path string, v *values) *node {
	if n == nil {
		return nil
	}
	if path == "" {
		// An interior node with no route on it is a pattern that carries on
		// past here, so landing on it is not a match.
		if n.route == nil {
			return nil
		}
		return n
	}

	seg, rest := firstSegment(path)
	if m := n.lit.find(seg).matchPath(rest, v); m != nil {
		return m
	}

	// A single wildcard does not match the end of a path, so a trailing slash
	// goes straight to the trailing wildcard.
	if seg != "/" {
		mark := v.n
		for _, w := range n.wild {
			if !w.check(seg) {
				continue
			}
			v.push(seg)
			if m := w.node.matchPath(rest, v); m != nil {
				return m
			}
			v.take(mark)
		}
		if n.any != nil {
			v.push(seg)
			if m := n.any.matchPath(rest, v); m != nil {
				return m
			}
			v.take(mark)
		}
	}

	if c := n.multi; c != nil {
		// A trailing slash makes a wildcard with no name, and there is nothing
		// to record a value under.
		if c.route.pat.last().s != "" {
			v.push(unescape(path[1:]))
		}
		return c
	}
	return nil
}

// methods is every method that would match this host and path, which is what
// the Allow header of a 405 lists.
func (n *node) methods(host, path string, out []string) []string {
	if host != "" {
		out = n.lit.find(host).methodsUnder(path, out)
	}
	out = n.lit.find("").methodsUnder(path, out)
	if contains(out, "GET") && !contains(out, "HEAD") {
		out = append(out, "HEAD")
	}
	return out
}

func (n *node) methodsUnder(path string, out []string) []string {
	if n == nil {
		return out
	}
	var v values
	n.lit.each(func(method string, c *node) {
		// The child under the empty key holds the patterns with no method,
		// and it is not reached here: this runs only when matching by method
		// already failed, and a method-less pattern would have answered then.
		if method == "" || contains(out, method) {
			return
		}
		v.reset()
		if c.matchPath(path, &v) != nil {
			out = append(out, method)
		}
	})
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// firstSegment splits a path that starts with a slash into its first segment
// and what is left, with the segment's percent escapes read.
//
// A path that is only a slash comes back as the segment "/", which is the end
// of the path and is what {$} was stored as.
func firstSegment(path string) (seg, rest string) {
	if path == "/" {
		return "/", ""
	}
	path = path[1:]
	i := strings.IndexByte(path, '/')
	if i < 0 {
		i = len(path)
	}
	return unescape(path[:i]), path[i:]
}

// A mapping is the literal children of a node.
//
// It stays a slice while it is small, which is nearly always: a scan over a few
// strings beats hashing one, and the strings being compared differ in their
// first byte almost every time. The map arrives when a node has enough children
// for the scan to be the slower of the two, which happens at the root of a
// large route table and hardly anywhere else.
type mapping struct {
	list []pair
	m    map[string]*node
}

type pair struct {
	key   string
	value *node
}

// scanUpTo is how many children a node keeps as a slice. It is the number
// net/http settled on for the same structure.
const scanUpTo = 8

func (m *mapping) find(key string) *node {
	if m.m != nil {
		return m.m[key]
	}
	for _, p := range m.list {
		if p.key == key {
			return p.value
		}
	}
	return nil
}

func (m *mapping) add(key string, n *node) {
	if m.m == nil && len(m.list) < scanUpTo {
		m.list = append(m.list, pair{key, n})
		return
	}
	if m.m == nil {
		m.m = make(map[string]*node, len(m.list)+1)
		for _, p := range m.list {
			m.m[p.key] = p.value
		}
		m.list = nil
	}
	m.m[key] = n
}

// each calls fn for every child. The order is the order they were added while
// the mapping is a slice, and is not defined once it is a map, so nothing that
// depends on the order uses this.
func (m *mapping) each(fn func(key string, n *node)) {
	for _, p := range m.list {
		fn(p.key, p.value)
	}
	for k, v := range m.m {
		fn(k, v)
	}
}

// values collects the wildcard values of a match.
//
// The first few live in an array so that a match allocates nothing, which is
// what the budget for this package is about. Beyond that it spills to a slice,
// which is a route with nine wildcards in it and is somebody else's problem.
//
// take is what backtracking uses: a branch that failed leaves whatever it
// pushed behind, and the caller cuts the record back to where it was before
// trying the next child.
type values struct {
	n     int
	fixed [onStack]string
	spill []string
}

// onStack is how many wildcards a route can have before a match starts
// allocating. Three or fewer covers nearly every route anybody writes.
const onStack = 8

func (v *values) push(s string) {
	if v.n < onStack {
		v.fixed[v.n] = s
	} else {
		v.spill = append(v.spill, s)
	}
	v.n++
}

func (v *values) take(n int) {
	if n >= v.n {
		return
	}
	if v.n > onStack {
		v.spill = v.spill[:max(0, n-onStack)]
	}
	v.n = n
}

func (v *values) reset() { v.take(0) }

func (v *values) at(i int) string {
	if i < onStack {
		return v.fixed[i]
	}
	return v.spill[i-onStack]
}
