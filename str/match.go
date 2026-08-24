package str

import "strings"

// Is reports whether s matches pattern, where a star stands for any run of
// characters and everything else is a literal.
//
//	str.Is("admin/*", "admin/users/1")   // true
//	str.Is("*.jpg", "photo.jpg")         // true
//	str.Is("v?", "v1")                   // false, a question mark is a literal
//
// The pattern comes first, the way it does in [path.Match] and
// [regexp.MatchString], rather than second the way the subject does everywhere
// else in this package. Matchers in the standard library are unanimous about
// this and being the one that reads backwards would cost more than the
// inconsistency does.
//
// A star crosses anything, including a slash. That is the difference from
// [path.Match], which stops a star at a separator and would not match the first
// example above, and it is why route and permission patterns want this one.
// There is nothing else to the syntax: no question mark, no character class, no
// escaping, so a pattern that has to express more than a prefix, a suffix or a
// gap wants [regexp].
func Is(pattern, s string) bool {
	// The pattern is walked a piece at a time rather than split, because a
	// split allocates a slice on every call and matching a request path
	// against a list of patterns is a thing that happens once per request.
	star := strings.IndexByte(pattern, '*')
	if star < 0 {
		return pattern == s
	}

	// Whatever comes before the first star is anchored to the front, since
	// there is nothing in front of it that could have moved it along.
	if !strings.HasPrefix(s, pattern[:star]) {
		return false
	}
	s, pattern = s[star:], pattern[star+1:]

	for {
		star = strings.IndexByte(pattern, '*')
		if star < 0 {
			// And whatever comes after the last star is anchored to the back.
			return strings.HasSuffix(s, pattern)
		}

		// A piece with a star on either side can be anywhere, and the leftmost
		// place it fits is as good as any: passing over a later occurrence
		// never puts a piece still to come out of reach.
		piece := pattern[:star]
		at := strings.Index(s, piece)
		if at < 0 {
			return false
		}
		s, pattern = s[at+len(piece):], pattern[star+1:]
	}
}
