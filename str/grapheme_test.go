package str_test

import (
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/go-mizu/mizu/str"
)

// The strings the segmenter has to get right, written out with escapes so that
// an editor cannot quietly normalise them into something easier.
const (
	combined = "é"                                          // e with a combining acute
	family   = "\U0001F468‍\U0001F469‍\U0001F467‍\U0001F466" // a family of four
	flagJP   = "\U0001F1EF\U0001F1F5"                        // the flag of Japan
	thumbsUp = "\U0001F44D\U0001F3FD"                        // a thumbs up with a skin tone
	hindi    = "क्षि"                                        // ksi in Devanagari
	hangul   = "각"                                         // gak, spelled out as three jamo
)

func TestLength(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{combined, 1},
		{family, 1},
		{flagJP, 1},
		{thumbsUp, 1},
		{hindi, 2},
		{hangul, 1},
		{"a" + family + "b", 3},
		{flagJP + flagJP, 2},
	}

	for _, c := range cases {
		if got := str.Length(c.in); got != c.want {
			t.Errorf("Length(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestLengthAgainstRunesAndBytes is the point of the whole file: for anything
// interesting the three counts differ, and only one of them is what a reader
// would say.
func TestLengthAgainstRunesAndBytes(t *testing.T) {
	if got := str.Length(family); got != 1 {
		t.Errorf("Length(family) = %d, want 1", got)
	}
	if got := utf8.RuneCountInString(family); got != 7 {
		t.Errorf("the family is %d runes, want 7", got)
	}
	if got := len(family); got != 25 {
		t.Errorf("the family is %d bytes, want 25", got)
	}
}

func TestGraphemes(t *testing.T) {
	got := slices.Collect(str.Graphemes("a" + combined + family))
	want := []string{"a", combined, family}

	if !slices.Equal(got, want) {
		t.Errorf("Graphemes gave %q, want %q", got, want)
	}
}

func TestGraphemesOfAnEmptyString(t *testing.T) {
	if got := slices.Collect(str.Graphemes("")); len(got) != 0 {
		t.Errorf("Graphemes of nothing gave %q, want nothing", got)
	}
}

// TestGraphemesStopsEarly is the rule every sequence in this tree has to obey.
func TestGraphemesStopsEarly(t *testing.T) {
	seen := 0
	for range str.Graphemes("abcdef") {
		seen++
		if seen == 2 {
			break
		}
	}

	if seen != 2 {
		t.Errorf("the loop ran %d times after breaking at 2", seen)
	}
}

// TestGraphemesPutBackTogether checks nothing is lost or added, on every string
// in the table at once.
func TestGraphemesPutBackTogether(t *testing.T) {
	for _, in := range []string{"", "hello", combined, family, flagJP, thumbsUp, hindi, hangul, "a\r\nb"} {
		out := ""
		for g := range str.Graphemes(in) {
			out += g
		}
		if out != in {
			t.Errorf("Graphemes(%q) put back together gave %q", in, out)
		}
	}
}

// TestCarriageReturnAndNewline is a rule of its own in the annex, because a
// Windows line ending is one character and not two.
func TestCarriageReturnAndNewline(t *testing.T) {
	if got := str.Length("a\r\nb"); got != 3 {
		t.Errorf("Length of a CRLF b = %d, want 3", got)
	}
	if got := str.Length("a\n\rb"); got != 4 {
		t.Errorf("Length of a LF CR b = %d, want 4, since only CR LF joins", got)
	}
}

// TestFlagsPairUp covers the rule that counts regional indicators, where the
// third one starts a new flag rather than joining the first two.
func TestFlagsPairUp(t *testing.T) {
	japan := "\U0001F1EF\U0001F1F5"
	lone := "\U0001F1EF"

	if got := str.Length(japan + lone); got != 2 {
		t.Errorf("Length of a flag and a spare indicator = %d, want 2", got)
	}
	if got := str.Length(japan + japan + japan); got != 3 {
		t.Errorf("Length of three flags = %d, want 3", got)
	}
}

// TestControlCharactersStandAlone is the other side of the rule that keeps
// marks attached: a control character joins nothing.
// TestZeroWidthNonJoiner covers the other invisible joiner, the one Persian
// and Hindi use to stop two letters running together. It attaches to the letter
// before it rather than standing on its own.
func TestZeroWidthNonJoiner(t *testing.T) {
	if got := str.Length("a\u200Cb"); got != 2 {
		t.Errorf("Length of a ZWNJ b = %d, want 2", got)
	}
}

func TestControlCharactersStandAlone(t *testing.T) {
	if got := str.Length("a\tb"); got != 3 {
		t.Errorf("Length of a tab b = %d, want 3", got)
	}
	// The first character here is not ASCII, so this goes the long way round
	// the classifier rather than down the fast path.
	if got := str.Length("é\tb"); got != 3 {
		t.Errorf("Length of an accented letter, a tab and b = %d, want 3", got)
	}
	if got := str.Length("é\u2028b"); got != 3 {
		t.Errorf("Length across a line separator = %d, want 3", got)
	}
	if got := str.Length("́"); got != 1 {
		t.Errorf("Length of a lone combining accent = %d, want 1", got)
	}
}

// TestAControlCharacterDoesNotAbsorbAMark is the case a fast path in the
// classifier got wrong once: a tab is a control character, so a combining
// accent after one stands on its own rather than attaching to it.
func TestAControlCharacterDoesNotAbsorbAMark(t *testing.T) {
	if got := str.Length("a\t\u0301"); got != 3 {
		t.Errorf("Length of a, tab, combining acute = %d, want 3", got)
	}
	if got := str.Length("a\u0301"); got != 1 {
		t.Errorf("Length of a and a combining acute = %d, want 1", got)
	}
}

// TestPrependMarksAttachForwards covers the Arabic number sign, which is the
// one class of mark that hangs onto what comes after it.
func TestPrependMarksAttachForwards(t *testing.T) {
	if got := str.Length("؀" + "7"); got != 1 {
		t.Errorf("Length of a number sign and a digit = %d, want 1", got)
	}
}

func TestHangulSyllables(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"three jamo", "각", 1},
		{"lead and vowel", "가", 1},
		{"precomposed", "각", 1},
		{"precomposed and a trailing jamo", "각", 1},
		{"a full syllable and another trailing jamo", "각ᆨ", 1},
		{"two lead jamo", "ᄀᄀ", 1},
		{"a vowel then a lead", "ᅡᄀ", 2},
	}

	for _, c := range cases {
		if got := str.Length(c.in); got != c.want {
			t.Errorf("%s: Length = %d, want %d", c.name, got, c.want)
		}
	}
}

// TestInvalidUTF8 pins that broken input comes back rather than hanging or
// panicking, since strings in Go are bytes and not all of them are text.
func TestInvalidUTF8(t *testing.T) {
	in := "a\xffb"

	if got := str.Length(in); got != 3 {
		t.Errorf("Length of a string with a bad byte = %d, want 3", got)
	}

	out := ""
	for g := range str.Graphemes(in) {
		out += g
	}
	if out != in {
		t.Errorf("Graphemes of a string with a bad byte gave %q, want %q", out, in)
	}
}
