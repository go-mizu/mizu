package router

import (
	"maps"
	"regexp"
	"slices"
)

// A Constraint reports whether a path segment is an acceptable value for a
// wildcard that names it.
//
// It is asked one segment at a time while a request is being matched, so it
// runs on the hot path and is written to allocate nothing. The built-in ones
// are hand-written scanners rather than regexps for that reason.
//
// A constraint answers yes or no and nothing else. It does not convert, because
// a match that converted would have to put the result somewhere, and the
// somewhere is an any, and an any is an allocation on a path that is budgeted
// for none. Conversion belongs to whoever reads the parameter, which knows the
// type it wants.
type Constraint func(segment string) bool

// constraints is the set of names a pattern may write after a colon.
type constraints map[string]Constraint

// names is the constraint names in order, which is what a message about an
// unknown one lists.
func (cs constraints) names() []string { return slices.Sorted(maps.Keys(cs)) }

// builtin is the constraint set every router starts with.
//
// They are the ones an identifier in a URL turns out to be nearly every time.
// Anything else is a Constraint somebody writes and registers with [Constrain],
// including a regexp through [Regexp].
var builtin = constraints{
	"int":   isInt,
	"uint":  isUint,
	"uuid":  isUUID,
	"ulid":  isULID,
	"slug":  isSlug,
	"alpha": isAlpha,
	"word":  isWord,
	"date":  isDate,
}

// Regexp is a constraint that accepts a segment the expression matches in
// full.
//
// The expression is anchored at both ends before it is compiled, so
// Regexp("[0-9]+") does not accept "12ab". It panics when the expression will
// not compile, since a route table is written once and a broken one is a
// mistake in the program rather than a condition to handle.
//
// A regexp is the slow way to say this. Every request that reaches the wildcard
// runs the engine, where a built-in constraint is a loop over bytes. Reach for
// it when the shape is genuinely irregular, and write a [Constraint] by hand
// when it is not.
func Regexp(expr string) Constraint {
	return regexp.MustCompile(`\A(?:` + expr + `)\z`).MatchString
}

// The bounds isInt and isUint check against, written out so that a long run of
// digits is compared rather than converted.
const (
	maxInt64  = "9223372036854775807"
	minInt64  = "9223372036854775808" // without the sign, so it compares as digits
	maxUint64 = "18446744073709551615"
)

// isInt accepts an optionally signed decimal that fits an int64.
//
// The length test carries nearly every segment: eighteen digits or fewer always
// fits, so the bound is only looked at for the handful that are longer.
func isInt(s string) bool {
	i, neg := 0, false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		i, neg = 1, s[0] == '-'
	}
	d := s[i:]
	if !digits(d) {
		return false
	}
	if len(d) <= 18 {
		return true
	}
	if neg {
		return fits(d, minInt64)
	}
	return fits(d, maxInt64)
}

// isUint accepts an unsigned decimal that fits a uint64.
func isUint(s string) bool {
	if !digits(s) {
		return false
	}
	return len(s) <= 19 || fits(s, maxUint64)
}

// fits reports whether a run of digits is at most limit.
//
// It compares them as they are written rather than converting, since
// strconv reports a number that does not fit by allocating an error and this
// runs on a request.
func fits(s, limit string) bool {
	for len(s) > 1 && s[0] == '0' {
		s = s[1:]
	}
	if len(s) != len(limit) {
		return len(s) < len(limit)
	}
	return s <= limit
}

func digits(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

// isUUID accepts the canonical 8-4-4-4-12 form in either case. The braced and
// unhyphenated forms are not accepted, because a URL that carries one of those
// is a URL somebody built by hand from the wrong end.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := range len(s) {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			if !isHex(s[i]) {
				return false
			}
		}
	}
	return true
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

// isULID accepts 26 characters of Crockford base32 in either case.
//
// The first character carries the top bits of a 128 bit value, so anything
// above 7 there is a string of the right length that is not a ULID, and
// accepting it would hand the handler something no ULID reader will parse.
func isULID(s string) bool {
	if len(s) != 26 || s[0] < '0' || s[0] > '7' {
		return false
	}
	for i := range len(s) {
		if !isCrockford(s[i]) {
			return false
		}
	}
	return true
}

// isCrockford is the base32 alphabet with I, L, O and U left out, which is what
// keeps a ULID from being misread out loud.
func isCrockford(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z':
		switch c | 0x20 {
		case 'i', 'l', 'o', 'u':
			return false
		}
		return true
	}
	return false
}

// isSlug accepts lower case letters and digits in groups separated by single
// hyphens, which is the shape a title turns into.
func isSlug(s string) bool {
	if s == "" || s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for i := range len(s) {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', '0' <= c && c <= '9':
		case c == '-':
			if s[i-1] == '-' {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// isAlpha accepts one or more letters, in either case, and nothing else.
func isAlpha(s string) bool {
	for i := range len(s) {
		c := s[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
			return false
		}
	}
	return s != ""
}

// isWord accepts letters, digits and underscores, which is a name somebody can
// type without escaping any of it.
func isWord(s string) bool {
	for i := range len(s) {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9', c == '_':
		default:
			return false
		}
	}
	return s != ""
}

// isDate accepts a calendar date written YYYY-MM-DD.
//
// The day is checked against the month and the year, so 2025-02-30 is refused
// here rather than in the handler. Leap years are the Gregorian rule, which is
// the rule every date in a URL is written under.
func isDate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	if !digits(s[:4]) || !digits(s[5:7]) || !digits(s[8:]) {
		return false
	}
	year := number(s[:4])
	month := number(s[5:7])
	day := number(s[8:])
	if month < 1 || month > 12 || day < 1 {
		return false
	}
	return day <= daysIn(year, month)
}

// number is a run of digits already known to be digits, read without strconv so
// that a segment that is not a date costs nothing to turn down.
func number(s string) int {
	n := 0
	for i := range len(s) {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func daysIn(year, month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	return 31
}
