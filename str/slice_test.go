package str_test

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/str"
)

func TestSubstr(t *testing.T) {
	cases := []struct {
		name          string
		s             string
		start, length int
		want          string
	}{
		{"from the front", "hello world", 0, 5, "hello"},
		{"from the middle", "hello world", 6, 5, "world"},
		{"a negative start", "hello world", -5, 5, "world"},
		{"a negative length", "hello world", 1, -1, "ello worl"},
		{"past the end", "hello", 2, 100, "llo"},
		{"a start past the end", "hello", 10, 3, ""},
		{"a start before the front", "hello", -100, 2, "he"},
		{"nothing wanted", "hello", 1, 0, ""},
		{"a negative length that eats it all", "hello", 0, -10, ""},
		{"a start past the end with a negative length", "hello", 10, -1, ""},
		{"an empty string", "", 0, 5, ""},
		{"an empty string counted from the end", "", -1, -1, ""},
	}

	for _, c := range cases {
		if got := str.Substr(c.s, c.start, c.length); got != c.want {
			t.Errorf("%s: Substr(%q, %d, %d) = %q, want %q", c.name, c.s, c.start, c.length, got, c.want)
		}
	}
}

// TestSubstrCountsCharacters is the reason this is not string slicing: the
// answer is in characters and the offsets in the string are not.
func TestSubstrCountsCharacters(t *testing.T) {
	const s = "a\U0001F1EF\U0001F1F5b" // a, the flag of Japan, b

	if got := str.Substr(s, 1, 1); got != "\U0001F1EF\U0001F1F5" {
		t.Errorf("Substr gave %q, want the flag whole", got)
	}
	if got := str.Substr(s, 0, 2); got != "a\U0001F1EF\U0001F1F5" {
		t.Errorf("Substr gave %q, want a and the flag", got)
	}
	if got := str.Substr(s, -1, 1); got != "b" {
		t.Errorf("Substr gave %q, want b", got)
	}
}

func TestTake(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"hello world", 5, "hello"},
		{"hello world", -5, "world"},
		{"hello", 100, "hello"},
		{"hello", -100, "hello"},
		{"hello", 0, ""},
		{"", 5, ""},
	}

	for _, c := range cases {
		if got := str.Take(c.s, c.n); got != c.want {
			t.Errorf("Take(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestLimit(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		end  string
		want string
	}{
		{"the quick brown fox", 9, "...", "the quick..."},
		{"short", 9, "...", "short"},
		{"exactly9!", 9, "...", "exactly9!"},
		{"the quick", 4, "", "the "},
		{"anything", -1, "...", "..."},
		{"", 5, "...", ""},
	}

	for _, c := range cases {
		if got := str.Limit(c.s, c.n, c.end); got != c.want {
			t.Errorf("Limit(%q, %d, %q) = %q, want %q", c.s, c.n, c.end, got, c.want)
		}
	}
}

// TestLimitNeverCutsACharacterInHalf is the promise the whole package is built
// on, checked on the string most likely to break it.
func TestLimitNeverCutsACharacterInHalf(t *testing.T) {
	const s = "hi \U0001F468‍\U0001F469‍\U0001F467‍\U0001F466 there"

	for n := range str.Length(s) + 2 {
		got := str.Limit(s, n, "")
		if !strings.HasPrefix(s, got) {
			t.Fatalf("Limit(s, %d) = %q, which is not a prefix of the input", n, got)
		}
		if want := min(n, str.Length(s)); str.Length(got) != want {
			t.Errorf("Limit(s, %d) is %d characters, want %d", n, str.Length(got), want)
		}
	}
}

func TestWords(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		end  string
		want string
	}{
		{"the quick brown fox", 3, "...", "the quick brown..."},
		{"the quick", 5, "...", "the quick"},
		{"the quick brown", 3, "...", "the quick brown"},
		{"one", 0, "...", "..."},
		{"  leading spaces here", 2, "", "  leading spaces"},
		{"", 3, "...", ""},
		{"one two", -1, "!", "!"},
	}

	for _, c := range cases {
		if got := str.Words(c.s, c.n, c.end); got != c.want {
			t.Errorf("Words(%q, %d, %q) = %q, want %q", c.s, c.n, c.end, got, c.want)
		}
	}
}

// TestWordsKeepsTheSpacingItWasGiven is the difference from splitting on
// whitespace and joining the pieces back with single spaces.
func TestWordsKeepsTheSpacingItWasGiven(t *testing.T) {
	if got := str.Words("one  two  three", 2, ""); got != "one  two" {
		t.Errorf("Words gave %q, want the double space kept", got)
	}
}

func TestExcerpt(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		phrase   string
		radius   int
		omission string
		want     string
	}{
		{"in the middle", "the quick brown fox jumps", "brown", 6, "...", "...quick brown fox j..."},
		{"at the front", "brown fox jumps", "brown", 3, "...", "brown fo..."},
		{"at the back", "the quick brown", "brown", 3, "...", "...ck brown"},
		{"the whole string", "brown", "brown", 10, "...", "brown"},
		{"no radius", "the brown fox", "brown", 0, "|", "|brown|"},
		{"not found", "the quick fox", "brown", 5, "...", ""},
		{"an empty phrase", "anything", "", 5, "...", ""},
		{"a negative radius", "the brown fox", "brown", -3, "|", "|brown|"},
	}

	for _, c := range cases {
		got := str.Excerpt(c.s, c.phrase, c.radius, c.omission)
		if got != c.want {
			t.Errorf("%s: Excerpt = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestExcerptOfSomethingMissingIsNothing is where this parts company with the
// cutting functions, which hand the whole string back.
func TestExcerptOfSomethingMissingIsNothing(t *testing.T) {
	if got := str.Excerpt("a long piece of text", "absent", 5, "..."); got != "" {
		t.Errorf("Excerpt gave %q, want nothing", got)
	}
	if got := str.After("a long piece of text", "absent"); got == "" {
		t.Error("After gave nothing, want the whole string, which is the contrast")
	}
}

func TestMask(t *testing.T) {
	cases := []struct {
		s    string
		with rune
		keep int
		want string
	}{
		{"4111111111111111", '*', 4, "************1111"},
		{"secret", '*', 0, "******"},
		{"secret", '*', -3, "******"},
		{"secret", '*', 100, "secret"},
		{"secret", '*', 6, "secret"},
		{"", '*', 4, ""},
		{"token", '#', 2, "###en"},
	}

	for _, c := range cases {
		if got := str.Mask(c.s, c.with, c.keep); got != c.want {
			t.Errorf("Mask(%q, %q, %d) = %q, want %q", c.s, c.with, c.keep, got, c.want)
		}
	}
}

// TestMaskHidesAWholeCharacter checks the count is in characters on both sides,
// so a masked emoji is one mark and not seven.
func TestMaskHidesAWholeCharacter(t *testing.T) {
	const s = "\U0001F468‍\U0001F469‍\U0001F467‍\U0001F466ab"

	if got := str.Mask(s, '*', 2); got != "*ab" {
		t.Errorf("Mask gave %q, want one asterisk and ab", got)
	}
}

// TestMaskOfAShortStringNeverRevealsPartOfIt is the safety property, since the
// whole point is hiding a secret.
func TestMaskOfAShortStringNeverRevealsPartOfIt(t *testing.T) {
	for _, s := range []string{"", "a", "ab", "abc", "abcd"} {
		got := str.Mask(s, '*', 4)
		if len(s) > 4 {
			continue
		}
		if got != s {
			t.Errorf("Mask(%q, keep 4) = %q, want the string unchanged when it is no longer than the tail", s, got)
		}
	}
}

func TestReverse(t *testing.T) {
	cases := []struct{ s, want string }{
		{"", ""},
		{"a", "a"},
		{"hello", "olleh"},
		{"日本語", "語本日"},
	}

	for _, c := range cases {
		if got := str.Reverse(c.s); got != c.want {
			t.Errorf("Reverse(%q) = %q, want %q", c.s, got, c.want)
		}
	}
}

// TestReverseKeepsCharactersWhole is the reason this is not a loop over runes,
// which would leave the accent on the wrong letter and the flag inside out.
func TestReverseKeepsCharactersWhole(t *testing.T) {
	const in = "ab́c" // a, b with a combining acute, c

	got := str.Reverse(in)
	if want := "cb́a"; got != want {
		t.Errorf("Reverse(%q) = %q, want %q", in, got, want)
	}

	const flag = "a\U0001F1EF\U0001F1F5"
	if got := str.Reverse(flag); got != "\U0001F1EF\U0001F1F5a" {
		t.Errorf("Reverse gave %q, want the flag the right way round", got)
	}
}

func TestReverseTwiceComesBack(t *testing.T) {
	for _, s := range []string{"", "hello", "日本語", "áb", "\U0001F1EF\U0001F1F5"} {
		if got := str.Reverse(str.Reverse(s)); got != s {
			t.Errorf("reversing %q twice gave %q", s, got)
		}
	}
}
