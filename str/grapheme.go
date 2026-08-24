package str

import (
	"iter"
	"unicode"
	"unicode/utf8"
)

// Graphemes returns the grapheme clusters of s, one at a time.
//
//	for g := range str.Graphemes("héllo") {
//		fmt.Println(g)
//	}
//
// A grapheme cluster is what a reader calls a character: a base letter with
// everything hanging off it, an emoji with its skin tone, a flag made of two
// regional indicators, a family made of four people and three joiners. Ranging
// over a string with for range gives runes, and a rune is smaller than that, so
// a loop that slices on rune boundaries can cut a character in half.
//
// The clusters come back as substrings of s and no copying is done.
func Graphemes(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for s != "" {
			n := clusterLen(s)
			if !yield(s[:n]) {
				return
			}
			s = s[n:]
		}
	}
}

// Length returns the number of grapheme clusters in s.
//
//	str.Length("héllo")   // 5
//	str.Length("👨‍👩‍👧‍👦")       // 1
//
// This is the unit every function in this package counts in, so it is the
// number to compare against when [Limit] or [Substr] or [Pad] is involved. Use
// len(s) for bytes and [utf8.RuneCountInString] for runes.
func Length(s string) int {
	n := 0
	for s != "" {
		s = s[clusterLen(s):]
		n++
	}
	return n
}

// clusterLen returns the length in bytes of the first grapheme cluster in s.
// This is the Unicode text segmentation algorithm from annex 29, and the rest
// of the package is built on it.
func clusterLen(s string) int {
	if s == "" {
		return 0
	}

	// Text is mostly ASCII and no ASCII byte joins the one before it, with the
	// single exception of the newline after a carriage return. Taking that
	// straight off the front skips two table lookups per character, which on a
	// plain English string is nearly all of the work.
	if s[0] < utf8.RuneSelf && s[0] != '\r' && (len(s) == 1 || s[1] < utf8.RuneSelf) {
		return 1
	}

	r, size := utf8.DecodeRuneInString(s)
	prev := classOf(r)
	i := size

	// run counts the regional indicators seen so far in this cluster, which is
	// what decides whether the next one starts a new flag or finishes this one.
	run := 0
	if prev == clsRegional {
		run = 1
	}

	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		cur := classOf(r)
		if boundary(prev, cur, run) {
			break
		}

		i += size
		if cur == clsRegional {
			run++
		} else {
			run = 0
		}
		prev = cur
	}
	return i
}

// class is the grapheme cluster break property of a rune, the small set of
// classes annex 29 sorts every rune into before deciding where the breaks go.
type class uint8

const (
	clsOther class = iota
	clsCR
	clsLF
	clsControl
	clsExtend
	clsZWJ
	clsRegional
	clsPrepend
	clsSpacingMark
	clsHangulL
	clsHangulV
	clsHangulT
	clsHangulLV
	clsHangulLVT
)

const (
	zeroWidthJoiner    = 0x200D
	zeroWidthNonJoiner = 0x200C
)

func classOf(r rune) class {
	// The named runes come first because several of them fall in categories
	// that would otherwise send them somewhere else. The joiner is a format
	// character and the skin tones are modifier symbols.
	switch {
	case r == '\r':
		return clsCR
	case r == '\n':
		return clsLF
	case r == zeroWidthJoiner:
		return clsZWJ
	case r == zeroWidthNonJoiner:
		return clsExtend
	case r >= 0x1F1E6 && r <= 0x1F1FF:
		return clsRegional
	case r >= 0x1F3FB && r <= 0x1F3FF:
		return clsExtend
	case isPrepend(r):
		return clsPrepend
	}

	// Below the combining diacritics there is nothing that joins anything, so
	// all of Latin, Greek, Cyrillic and the modifier letters answer here rather
	// than searching the category tables. What is left over is the two control
	// blocks and the soft hyphen, and those have to keep their own class,
	// because a control character joins nothing on either side of it.
	if r < 0x0300 {
		if r < 0x20 || (r >= 0x7F && r <= 0x9F) || r == 0x00AD {
			return clsControl
		}
		return clsOther
	}

	switch {
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r):
		return clsExtend
	case unicode.Is(unicode.Mc, r):
		return clsSpacingMark
	case unicode.Is(unicode.Cc, r), unicode.Is(unicode.Cf, r),
		unicode.Is(unicode.Zl, r), unicode.Is(unicode.Zp, r):
		return clsControl
	}

	switch {
	case r >= 0x1100 && r <= 0x115F, r >= 0xA960 && r <= 0xA97C:
		return clsHangulL
	case r >= 0x1160 && r <= 0x11A7, r >= 0xD7B0 && r <= 0xD7C6:
		return clsHangulV
	case r >= 0x11A8 && r <= 0x11FF, r >= 0xD7CB && r <= 0xD7FB:
		return clsHangulT
	case r >= 0xAC00 && r <= 0xD7A3:
		// A precomposed syllable is LV when it has no trailing consonant and
		// LVT when it has one, and the arithmetic is how the block is laid out.
		if (r-0xAC00)%28 == 0 {
			return clsHangulLV
		}
		return clsHangulLVT
	}

	return clsOther
}

// boundary reports whether a cluster ends between prev and cur. run is the
// number of regional indicators already in the cluster.
//
// The cases are the rules from annex 29 in the order the annex gives them,
// which is the order they have to be tested in, since an earlier rule wins over
// a later one.
func boundary(prev, cur class, run int) bool {
	switch {
	case prev == clsCR && cur == clsLF:
		return false
	case prev == clsControl || prev == clsCR || prev == clsLF:
		return true
	case cur == clsControl || cur == clsCR || cur == clsLF:
		return true

	case prev == clsHangulL &&
		(cur == clsHangulL || cur == clsHangulV || cur == clsHangulLV || cur == clsHangulLVT):
		return false
	case (prev == clsHangulLV || prev == clsHangulV) &&
		(cur == clsHangulV || cur == clsHangulT):
		return false
	case (prev == clsHangulLVT || prev == clsHangulT) && cur == clsHangulT:
		return false

	case cur == clsExtend || cur == clsZWJ:
		return false
	case cur == clsSpacingMark:
		return false
	case prev == clsPrepend:
		return false

	case prev == clsZWJ:
		return false

	case prev == clsRegional && cur == clsRegional && run%2 == 1:
		return false
	}
	return true
}

// isPrepend reports whether r is one of the marks that attach to what follows
// them rather than to what came before. They are mostly Arabic number signs and
// a handful of Indic and Brahmi prefixes.
func isPrepend(r rune) bool {
	switch {
	case r >= 0x0600 && r <= 0x0605,
		r == 0x06DD, r == 0x070F, r == 0x0890, r == 0x0891, r == 0x08E2,
		r == 0x0D4E, r == 0x110BD, r == 0x110CD,
		r >= 0x111C2 && r <= 0x111C3,
		r == 0x1193F, r == 0x11941, r == 0x11A3A,
		r >= 0x11A84 && r <= 0x11A89,
		r == 0x11D46, r == 0x11F02:
		return true
	}
	return false
}
