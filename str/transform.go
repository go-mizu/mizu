package str

import (
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// latinFolding covers the characters that do not come apart into a letter and a
// mark, so no amount of normalising will turn them into ASCII. The list is the
// Latin alphabets of Europe plus the punctuation that word processors produce,
// which is the punctuation that turns up in text people paste in.
var latinFolding = strings.NewReplacer(
	"æ", "ae", "Æ", "AE",
	"œ", "oe", "Œ", "OE",
	"ø", "o", "Ø", "O",
	"ß", "ss", "ẞ", "SS",
	"þ", "th", "Þ", "Th",
	"ð", "d", "Ð", "D",
	"đ", "d", "Đ", "D",
	"ł", "l", "Ł", "L",
	"ħ", "h", "Ħ", "H",
	"ŧ", "t", "Ŧ", "T",
	"ı", "i", "İ", "I",
	"ŋ", "n", "Ŋ", "N",
	"ĸ", "k",
	"“", `"`, "”", `"`, "„", `"`,
	"‘", "'", "’", "'", "‚", "'",
	"«", `"`, "»", `"`,
	"–", "-", "—", "-", "−", "-",
	"…", "...",
	"\u00a0", " ", // a non-breaking space is still a space in ASCII
)

// Ascii returns s with the letters outside ASCII replaced by the nearest ASCII
// that means the same thing, and everything it cannot place removed.
//
//	str.Ascii("crème brûlée")   // creme brulee
//	str.Ascii("Køge Æblegrød")  // Koge AEblegrod
//
// Accents come off, the Latin letters that are not a letter plus an accent are
// spelled out, and typographic punctuation becomes the ASCII it stands in for.
//
// Text in a script that is not Latin has no nearest ASCII and is dropped, so
// Ascii of a Japanese sentence is empty. That is a real limit rather than a
// rounding error: this converts spelling, not writing systems, and there is no
// answer for a script whose letters are not letters at all. Code that has to
// keep something for every input wants to check the result before using it.
func Ascii(s string) string {
	// Text that is already ASCII cannot be changed by any of what follows,
	// because every key in the folding table is outside ASCII and so is every
	// character that normalising moves. Checking costs one pass and saves three.
	if isASCII(s) {
		return s
	}

	// Normalising to NFD pulls every accented letter apart into a plain letter
	// and a combining mark. Keeping only the ASCII bytes then does two jobs at
	// once, because the marks it drops are exactly the non-ASCII bytes that
	// come out of the split, and so is everything else with no ASCII spelling.
	folded := norm.NFD.String(latinFolding.Replace(s))

	var b strings.Builder
	b.Grow(len(folded))
	for i := 0; i < len(folded); i++ {
		if folded[i] < 0x80 {
			b.WriteByte(folded[i])
		}
	}
	return b.String()
}

// Slug returns s as a lowercase ASCII string with single hyphens between the
// words, which is the shape a URL path wants.
//
//	str.Slug("Hello, World!")       // hello-world
//	str.Slug("crème brûlée 2026")   // creme-brulee-2026
//
// Everything that is not a letter or a digit becomes a separator, runs of
// separators collapse into one, and the ends are trimmed. The result is safe in
// a path segment without escaping.
//
// It goes through [Ascii] first, so a title in a script that is not Latin slugs
// to an empty string. Code that builds a URL out of user supplied titles has to
// have an answer for that, usually an id in front of the slug.
func Slug(s string) string {
	s = strings.ToLower(Ascii(s))

	var b strings.Builder
	b.Grow(len(s))
	dash := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteByte(c)
			continue
		}
		dash = true
	}
	return b.String()
}

// Ordinal returns n written as a position, so 1 becomes 1st and 22 becomes
// 22nd.
//
//	str.Ordinal(1)    // 1st
//	str.Ordinal(11)   // 11th
//	str.Ordinal(23)   // 23rd
//
// The teens are the part worth having a function for. Eleven, twelve and
// thirteen take th rather than the st, nd and rd their last digit would suggest,
// and every hundred years the same three do it again.
//
// A negative n keeps its sign and is worded from its size, so -1 is -1st.
func Ordinal(n int) string {
	digits := strconv.Itoa(n)

	size := n
	if size < 0 {
		size = -size
	}

	suffix := "th"
	if size%100 < 11 || size%100 > 13 {
		switch size % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return digits + suffix
}

// Finish returns s with exactly one suffix on the end.
//
//	str.Finish("path/to", "/")     // path/to/
//	str.Finish("path/to/", "/")    // path/to/
//	str.Finish("path/to///", "/")  // path/to/
//
// Trailing copies of the suffix collapse into one, which is what makes this
// worth more than a HasSuffix check. Joining a base URL that may or may not end
// in a slash to a path that may or may not start with one is where this earns
// its place.
func Finish(s, suffix string) string {
	if suffix == "" {
		return s
	}
	return trimRepeats(s, suffix, false) + suffix
}

// Start returns s with exactly one prefix on the front.
//
//	str.Start("api/users", "/")   // /api/users
//	str.Start("//api", "/")       // /api
//
// See [Finish], which does the same thing at the other end.
func Start(s, prefix string) string {
	if prefix == "" {
		return s
	}
	return prefix + trimRepeats(s, prefix, true)
}

// trimRepeats takes repeated copies of affix off one end of s, leaving at most
// one. Cutting them one at a time is what keeps a string of slashes from
// needing a loop at the call site.
func trimRepeats(s, affix string, front bool) string {
	for {
		var next string
		if front {
			next = strings.TrimPrefix(s, affix)
		} else {
			next = strings.TrimSuffix(s, affix)
		}
		if next == s {
			return s
		}
		s = next
	}
}

// ReplaceLast returns s with the last occurrence of old replaced by new.
//
//	str.ReplaceLast("a/b/c", "/", " and ")   // a/b and c
//
// The first occurrence is strings.Replace with a count of one and the whole lot
// is [strings.ReplaceAll], so those are not here. The last one has nothing in
// [strings] to call.
//
// An empty old is left alone rather than being inserted at the end, since
// [strings.LastIndex] finds an empty string everywhere and matching that here
// would be a surprise rather than a convenience.
func ReplaceLast(s, old, new string) string {
	if old == "" {
		return s
	}

	at := strings.LastIndex(s, old)
	if at < 0 {
		return s
	}
	return s[:at] + new + s[at+len(old):]
}

// isASCII reports whether every byte of s is below 0x80.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
