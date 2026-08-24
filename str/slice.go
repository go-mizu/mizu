package str

import (
	"strings"
	"unicode"
)

// Substr returns length characters of s starting at start.
//
//	str.Substr("hello world", 0, 5)   // hello
//	str.Substr("hello world", 6, 5)   // world
//	str.Substr("hello world", -5, 5)  // world
//
// A negative start counts back from the end, so -5 is the fifth character from
// the right. A negative length leaves off that many characters from the end
// instead of taking a fixed number, so Substr(s, 1, -1) drops the first and the
// last. Asking for more than is there gives what is there rather than an error.
//
// The counting is in grapheme clusters, so this cannot cut a character in half
// the way slicing a string by bytes can.
func Substr(s string, start, length int) string {
	// A negative start or length is counted from the end, so those are the only
	// two cases that have to measure the whole string first. When both are
	// positive the walk stops as soon as it has what was asked for, which is
	// what keeps taking the first few characters off a long string cheap.
	if start < 0 || length < 0 {
		total := Length(s)

		if start < 0 {
			start += total
			if start < 0 {
				start = 0
			}
		}
		if start >= total {
			return ""
		}
		if length < 0 {
			length = total + length - start
		}
	}
	if length <= 0 {
		return ""
	}

	// The second walk carries on from where the first one stopped rather than
	// starting over at the front of the string.
	from := byteAt(s, start)
	return s[from : from+byteAt(s[from:], length)]
}

// Take returns the first n characters of s, or the last n when n is negative.
//
//	str.Take("hello world", 5)    // hello
//	str.Take("hello world", -5)   // world
//
// Asking for more than is there gives the whole string. [Limit] is the one that
// marks a string it had to cut.
func Take(s string, n int) string {
	if n < 0 {
		return Substr(s, n, -n)
	}
	return Substr(s, 0, n)
}

// Limit cuts s down to n characters and puts end on the tail if it had to cut.
//
//	str.Limit("the quick brown fox", 9, "...")   // the quick...
//	str.Limit("short", 9, "...")                 // short
//
// The result is n characters plus end, not n characters in total, which is what
// people mean when they ask for a preview of the first n. Pass "" for end to
// cut with no mark.
//
// Counting is in grapheme clusters, so cutting a string with an emoji in it
// leaves the emoji whole or leaves it out.
func Limit(s string, n int, end string) string {
	if n < 0 {
		n = 0
	}

	// Walking to the cut point answers both questions at once: where to cut,
	// and whether there was anything to cut off. Measuring the string first
	// would cross all of it to learn something the first n characters settle.
	at := byteAt(s, n)
	if at == len(s) {
		return s
	}
	return s[:at] + end
}

// Words cuts s down to n words and puts end on the tail if it had to cut.
//
//	str.Words("the quick brown fox", 3, "...")   // the quick brown...
//
// A word is a run of anything that is not a space, the same as
// [strings.Fields]. The spacing inside what is kept is the spacing s had, so
// this is for trimming a paragraph rather than tidying one up. Tidying one up
// is strings.Join(strings.Fields(s), " ").
func Words(s string, n int, end string) string {
	if n < 0 {
		n = 0
	}

	count := 0
	inWord := false

	for i, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if !inWord {
			inWord = true
			count++
			if count > n {
				// i is the first character of the word past the limit, so
				// everything before it is the n words and the space after them.
				return strings.TrimRightFunc(s[:i], unicode.IsSpace) + end
			}
		}
	}
	return s
}

// Excerpt returns the part of s around the first phrase, with radius characters
// on either side and omission on whichever end it had to cut.
//
//	str.Excerpt("the quick brown fox jumps", "brown", 6, "...")
//	// ...uick brown fox j...
//
// When phrase is not in s, the result is empty, which is the difference from
// the cutting functions in this package that hand back the whole string. An
// excerpt of something that is not there is nothing, not everything.
func Excerpt(s, phrase string, radius int, omission string) string {
	if phrase == "" {
		return ""
	}

	at := strings.Index(s, phrase)
	if at < 0 {
		return ""
	}
	if radius < 0 {
		radius = 0
	}

	// The offsets are in bytes and the radius is in characters, so both edges
	// are found by counting clusters out from the match.
	before := Length(s[:at])
	start := before - radius
	if start < 0 {
		start = 0
	}
	end := before + Length(phrase) + radius

	out := Substr(s, start, end-start)
	if start > 0 {
		out = omission + out
	}
	if end < Length(s) {
		out += omission
	}
	return out
}

// Mask replaces every character of s except the last keep with the rune with.
//
//	str.Mask("4111111111111111", '*', 4)   // ************1111
//	str.Mask("secret", '*', 0)             // ******
//
// This is the shape a card number or a token wants, where the tail is what
// makes it recognisable and the rest is what has to go. A keep of 0 or less
// hides all of it, and a keep longer than the string leaves it alone, which
// means a short value is never partly revealed by asking for too much.
//
// The mask is one rune per character, so a family emoji becomes one asterisk
// rather than seven.
func Mask(s string, with rune, keep int) string {
	if keep < 0 {
		keep = 0
	}

	total := Length(s)
	if keep >= total {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	for range total - keep {
		b.WriteRune(with)
	}
	b.WriteString(s[byteAt(s, total-keep):])
	return b.String()
}

// Reverse returns s with its characters back to front.
//
//	str.Reverse("hello")   // olleh
//
// The unit is the grapheme cluster, so an accent stays on the letter it belongs
// to and an emoji built out of several code points comes back whole. Reversing
// by runes takes both of those apart.
func Reverse(s string) string {
	// The clusters are read forwards, because that is the only direction they
	// can be read in, and copied into the answer from the back. Collecting them
	// into a slice first and walking that backwards costs a string header per
	// character, which on a long string is more space than the answer.
	out := make([]byte, len(s))
	at := len(out)
	for g := range Graphemes(s) {
		at -= len(g)
		copy(out[at:], g)
	}
	return string(out)
}

// byteAt returns the byte offset where character n starts, or len(s) when there
// are not that many characters.
func byteAt(s string, n int) int {
	i := 0
	for range n {
		if i >= len(s) {
			return len(s)
		}
		i += clusterLen(s[i:])
	}
	return i
}
