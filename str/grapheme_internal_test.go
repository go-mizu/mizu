package str

import "testing"

// clusterLen is the one function the rest of the package is built on, so the
// empty case is checked from inside rather than left to the callers that all
// happen to guard against it. A version of it that returned anything other than
// zero here would turn Graphemes into a loop that never ends.
func TestClusterLenOfAnEmptyString(t *testing.T) {
	if got := clusterLen(""); got != 0 {
		t.Errorf("clusterLen of an empty string = %d, want 0", got)
	}
}

func TestClassOf(t *testing.T) {
	cases := []struct {
		name string
		r    rune
		want class
	}{
		{"a letter", 'a', clsOther},
		{"a carriage return", '\r', clsCR},
		{"a newline", '\n', clsLF},
		{"a tab", '\t', clsControl},
		{"a combining acute", 0x0301, clsExtend},
		{"a zero width joiner", zeroWidthJoiner, clsZWJ},
		{"a zero width non joiner", zeroWidthNonJoiner, clsExtend},
		{"a regional indicator", 0x1F1EF, clsRegional},
		{"a skin tone", 0x1F3FD, clsExtend},
		{"an Arabic number sign", 0x0600, clsPrepend},
		{"a Devanagari vowel sign", 0x093F, clsSpacingMark},
		{"a line separator", 0x2028, clsControl},
		{"a byte order mark", 0xFEFF, clsControl},
		{"a Hangul lead", 0x1100, clsHangulL},
		{"a Hangul vowel", 0x1161, clsHangulV},
		{"a Hangul trail", 0x11A8, clsHangulT},
		{"a syllable with no trail", 0xAC00, clsHangulLV},
		{"a syllable with a trail", 0xAC01, clsHangulLVT},
	}

	for _, c := range cases {
		if got := classOf(c.r); got != c.want {
			t.Errorf("%s: classOf(U+%04X) = %d, want %d", c.name, c.r, got, c.want)
		}
	}
}

// TestIsPrependCoversItsWholeTable walks the marks that attach forwards, since
// a typo in one of the ranges would otherwise go unnoticed.
func TestIsPrependCoversItsWholeTable(t *testing.T) {
	yes := []rune{
		0x0600, 0x0605, 0x06DD, 0x070F, 0x0890, 0x0891, 0x08E2, 0x0D4E,
		0x110BD, 0x110CD, 0x111C2, 0x111C3, 0x1193F, 0x11941, 0x11A3A,
		0x11A84, 0x11A89, 0x11D46, 0x11F02,
	}
	for _, r := range yes {
		if !isPrepend(r) {
			t.Errorf("isPrepend(U+%04X) = false, want true", r)
		}
	}

	no := []rune{'a', 0x05FF, 0x0606, 0x11A8B, 0x11F03}
	for _, r := range no {
		if isPrepend(r) {
			t.Errorf("isPrepend(U+%04X) = true, want false", r)
		}
	}
}
