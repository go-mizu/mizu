package router

import "strings"

// Which of two patterns wins, and when neither does.
//
// A router that stores two patterns matching the same request has to pick one,
// and the pick has to be the same one every time, obvious from reading the two
// patterns, and the same as what ServeMux would have picked. The rules are
// ServeMux's rules, taken from net/http:
//
//  1. A pattern with a host beats one without.
//  2. Otherwise the more specific of the two wins, where one pattern is more
//     specific than another when the other matches every request it does and
//     more.
//
// When neither is more specific and both match some request, there is no answer
// and registration refuses the second pattern. That is a compile-time-shaped
// error reported at startup, which is where a route table is built.
//
// Constraints are the one thing mizu adds, and they are handled below rather
// than in the comparison.

// A relation is how the set of requests one pattern matches sits against the
// set another matches.
type relation int

const (
	same        relation = iota // both match the same requests
	wider                       // the first matches everything the second does, and more
	narrower                    // the second matches everything the first does, and more
	apart                       // no request matches both
	overlapping                 // some request matches both, and neither is wider
)

// conflicts reports whether two patterns can match the same request with no
// rule saying which of them should win.
//
// A constraint that tells the two apart is the one thing that stops a pair from
// conflicting. Two wildcards in the same position with different constraints
// are separate children of the tree, tried in the order they were registered,
// so the answer for a request both accept is written down in the route table
// rather than being a coin toss. Whether two constraints can both accept the
// same segment is not a question this package can answer, since a constraint is
// a Go function, so it does not ask it.
func conflicts(a, b *pattern) bool {
	if a.host != b.host {
		// Either one has a host and the other does not, and rule 1 settles it,
		// or both have hosts and they are different, and no request carries
		// both.
		return false
	}
	if told(a, b) {
		return false
	}
	rel := combine(methods(a, b), paths(a, b))
	return rel == same || rel == overlapping
}

// told reports whether a constraint tells the two patterns apart, which means
// finding one position where they cannot both be satisfied.
//
// There are two ways for that to happen. Against another wildcard, a different
// constraint name is enough, since the tree keeps them as separate children and
// tries them in the order they were registered. Against a literal, there is a
// real answer to be had: the constraint is a function and the literal is a
// value, so run it, and a literal it turns down is a segment the two patterns
// can never agree on.
func told(a, b *pattern) bool {
	for i := range min(len(a.segs), len(b.segs)) {
		s, t := a.segs[i], b.segs[i]
		if s.multi || t.multi {
			continue
		}
		switch {
		case s.wild && t.wild:
			if s.con != t.con {
				return true
			}
		case s.wild && s.check != nil && !s.check(t.s):
			return true
		case t.wild && t.check != nil && !t.check(s.s):
			return true
		}
	}
	return false
}

// methods is the relation between the method halves.
//
// A pattern with no method matches every method, so it is the wider one. GET
// also answers HEAD, so GET is wider than HEAD. Anything else matches only
// itself.
func methods(a, b *pattern) relation {
	switch {
	case a.method == b.method:
		return same
	case a.method == "":
		return wider
	case b.method == "":
		return narrower
	case a.method == "GET" && b.method == "HEAD":
		return wider
	case a.method == "HEAD" && b.method == "GET":
		return narrower
	}
	return apart
}

// paths is the relation between the path halves, which is the relation of each
// pair of segments folded together.
func paths(a, b *pattern) relation {
	// A pattern that does not end in a trailing wildcard matches paths of one
	// length only, so two of them with different lengths have nothing in
	// common and there is nothing to compare.
	if len(a.segs) != len(b.segs) && !a.last().multi && !b.last().multi {
		return apart
	}

	rel := same
	x, y := a.segs, b.segs
	for ; len(x) > 0 && len(y) > 0; x, y = x[1:], y[1:] {
		rel = combine(rel, segments(x[0], y[0]))
		if rel == apart {
			return apart
		}
	}
	if len(x) == 0 && len(y) == 0 {
		return rel
	}

	// One pattern ran out first. The only way the two still have a request in
	// common is if the shorter one ends in a trailing wildcard, which takes
	// everything the longer one had left.
	if len(x) < len(y) && a.last().multi {
		return combine(rel, wider)
	}
	if len(y) < len(x) && b.last().multi {
		return combine(rel, narrower)
	}
	return apart
}

// segments is the relation between two segments in the same position.
//
// Constraints are left out on purpose. What they do to precedence is decided by
// the tree, which tries a constrained wildcard before a bare one, and what they
// do to conflicts is decided by told.
func segments(s, t segment) relation {
	switch {
	case s.multi && t.multi:
		return same
	case s.multi:
		return wider
	case t.multi:
		return narrower
	case s.wild && t.wild:
		return same
	case s.wild:
		// A single wildcard does not match the end of a path, which is what
		// {$} is.
		if t.s == "/" {
			return apart
		}
		return wider
	case t.wild:
		if s.s == "/" {
			return apart
		}
		return narrower
	case s.s == t.s:
		return same
	}
	return apart
}

// combine folds the relation of one part of a pattern together with the
// relation of the rest.
//
// Wider in one part and narrower in another is the case worth naming: each
// pattern matches something the other does not, so neither wins and the two
// overlap.
func combine(a, b relation) relation {
	switch a {
	case same:
		return b
	case apart:
		return apart
	case overlapping:
		if b == apart {
			return apart
		}
		return overlapping
	default:
		switch b {
		case same:
			return a
		case inverse(a):
			return overlapping
		}
		return b
	}
}

func inverse(r relation) relation {
	switch r {
	case wider:
		return narrower
	case narrower:
		return wider
	}
	return r
}

// shared is a path both patterns match, which is what a message about a
// conflict shows so that the reader has a request in front of them rather than
// two abstractions.
//
// It is only called for patterns that do have one.
func shared(a, b *pattern) string {
	var out strings.Builder
	x, y := a.segs, b.segs
	for ; len(x) > 0 && len(y) > 0; x, y = x[1:], y[1:] {
		// Take whichever of the two says more, so that a literal is written as
		// itself rather than as the other pattern's placeholder.
		if segments(x[0], y[0]) == wider {
			writeSegment(&out, y[0])
		} else {
			writeSegment(&out, x[0])
		}
	}
	for _, s := range x {
		writeSegment(&out, s)
	}
	for _, s := range y {
		writeSegment(&out, s)
	}
	return out.String()
}

// writeSegment writes a segment as a piece of path that matches it. A wildcard
// is written as its own name, which reads as a placeholder to anybody looking
// at the message.
func writeSegment(out *strings.Builder, s segment) {
	out.WriteByte('/')
	if !s.multi && s.s != "/" {
		out.WriteString(s.s)
	}
}
