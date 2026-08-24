package str

import "strings"

// PadLeft returns s padded on the left with with until it is n characters long.
//
//	str.PadLeft("7", 3, "0")     // 007
//	str.PadLeft("42", 7, "ab")   // ababa42
//
// A string already that long or longer comes back unchanged, so this never
// shortens anything. The padding repeats and is cut to fit, which is why the
// example above ends in a lone a.
//
// The counting is in grapheme clusters rather than bytes, so a column of names
// with accents in them lines up. It is not display width: a Japanese character
// is one character here and two columns wide in a terminal.
func PadLeft(s string, n int, with string) string {
	pad, ok := padding(s, n, with)
	if !ok {
		return s
	}
	return pad + s
}

// PadRight returns s padded on the right with with until it is n characters
// long.
//
//	str.PadRight("ana", 6, ".")   // ana...
//
// See [PadLeft] for what counts as a character and what happens when s is
// already long enough.
func PadRight(s string, n int, with string) string {
	pad, ok := padding(s, n, with)
	if !ok {
		return s
	}
	return s + pad
}

// Pad returns s padded on both sides with with until it is n characters long.
//
//	str.Pad("ana", 9, "-")   // ---ana---
//	str.Pad("ana", 8, "-")   // --ana---
//
// An odd amount of padding puts the extra character on the right, which is the
// same choice [fmt] makes and is what makes a centred column look right.
func Pad(s string, n int, with string) string {
	pad, ok := padding(s, n, with)
	if !ok {
		return s
	}

	half := Length(pad) / 2
	left := pad[:byteAt(pad, half)]
	return left + s + pad[byteAt(pad, half):]
}

// padding returns the run of padding needed to bring s up to n characters, and
// reports whether any is needed at all.
func padding(s string, n int, with string) (string, bool) {
	if with == "" {
		return "", false
	}

	short := n - Length(s)
	if short <= 0 {
		return "", false
	}

	// The filler is repeated past what is needed and then cut, because with can
	// be more than one character long.
	unit := Length(with)
	full := strings.Repeat(with, short/unit+1)
	return full[:byteAt(full, short)], true
}

// Wrap puts before on the front of s and after on the back.
//
//	str.Wrap("value", `"`, `"`)   // "value"
//	str.Wrap("b", "<i>", "</i>")  // <i>b</i>
//
// Nothing is checked and nothing is escaped. This is string handling, not
// markup, and anything going into HTML wants [html.EscapeString] instead.
func Wrap(s, before, after string) string { return before + s + after }

// Unwrap takes before off the front of s and after off the back, when they are
// both there.
//
//	str.Unwrap(`"value"`, `"`, `"`)   // value
//	str.Unwrap(`"value`, `"`, `"`)    // "value
//
// Both ends have to match or nothing is taken off, so a half-quoted string
// comes back as it was rather than half unwrapped.
func Unwrap(s, before, after string) string {
	if !strings.HasPrefix(s, before) || !strings.HasSuffix(s, after) {
		return s
	}
	// The two ends must not overlap, or a single quote would be read as both
	// its own opening and its own closing.
	if len(s) < len(before)+len(after) {
		return s
	}
	return s[len(before) : len(s)-len(after)]
}
