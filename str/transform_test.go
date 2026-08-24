package str_test

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/str"
)

func TestAscii(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"nothing to do", "plain ascii", "plain ascii"},
		{"accents come off", "crème brûlée", "creme brulee"},
		{"a precomposed letter", "café", "cafe"},
		{"a letter and a mark", "café", "cafe"},
		{"the Nordic letters", "Køge Æblegrød", "Koge AEblegrod"},
		{"a sharp s", "Straße", "Strasse"},
		{"a Polish l", "Łódź", "Lodz"},
		{"an Icelandic thorn", "Þórsmörk", "Thorsmork"},
		{"typographic quotes", "“hello”", `"hello"`},
		{"an apostrophe", "it’s", "it's"},
		{"a long dash", "a—b", "a-b"},
		{"an ellipsis", "wait…", "wait..."},
		{"a non-breaking space", "10 kg", "10 kg"},
		{"a script with no ascii", "日本語", ""},
		{"a mix of scripts", "hello 日本", "hello "},
		{"an empty string", "", ""},
	}

	for _, c := range cases {
		if got := str.Ascii(c.in); got != c.want {
			t.Errorf("%s: Ascii(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestAsciiIsAscii is the promise in the name, checked on everything in the
// table at once rather than trusted case by case.
func TestAsciiIsAscii(t *testing.T) {
	inputs := []string{"crème brûlée", "Køge Æblegrød", "日本語", "Ωμέγα", "Привет", "🇯🇵", "á̂̃"}

	for _, in := range inputs {
		got := str.Ascii(in)
		for i := 0; i < len(got); i++ {
			if got[i] >= 0x80 {
				t.Errorf("Ascii(%q) = %q, which has a byte %#x outside ascii", in, got, got[i])
				break
			}
		}
	}
}

func TestSlug(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"a title", "Hello, World!", "hello-world"},
		{"accents", "crème brûlée 2026", "creme-brulee-2026"},
		{"runs of punctuation", "a -- b __ c", "a-b-c"},
		{"leading and trailing junk", "  !!hello!!  ", "hello"},
		{"already a slug", "already-a-slug", "already-a-slug"},
		{"digits", "version 2 point 0", "version-2-point-0"},
		{"an underscore", "user_id", "user-id"},
		{"nothing but punctuation", "!!!", ""},
		{"a script with no ascii", "日本語", ""},
		{"an empty string", "", ""},
	}

	for _, c := range cases {
		if got := str.Slug(c.in); got != c.want {
			t.Errorf("%s: Slug(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestSlugIsSafeInAPath is the reason the function exists, so it is checked as
// a property over the table rather than read off the expected values.
func TestSlugIsSafeInAPath(t *testing.T) {
	inputs := []string{"Hello, World!", "crème brûlée", "a/b/c", "?query=1&x=2", "  spaces  ", "日本語", "a..b", "%2e%2e"}

	for _, in := range inputs {
		got := str.Slug(in)
		for i := 0; i < len(got); i++ {
			c := got[i]
			ok := c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-'
			if !ok {
				t.Errorf("Slug(%q) = %q, which has %q in it", in, got, c)
				break
			}
		}
		if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") || strings.Contains(got, "--") {
			t.Errorf("Slug(%q) = %q, want no hyphen at either end and none doubled", in, got)
		}
	}
}

func TestOrdinal(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0th"},
		{1, "1st"}, {2, "2nd"}, {3, "3rd"}, {4, "4th"},
		{10, "10th"},
		{11, "11th"}, {12, "12th"}, {13, "13th"}, {14, "14th"},
		{21, "21st"}, {22, "22nd"}, {23, "23rd"}, {24, "24th"},
		{100, "100th"},
		{101, "101st"}, {102, "102nd"}, {103, "103rd"},
		{111, "111th"}, {112, "112th"}, {113, "113th"},
		{1000, "1000th"},
		{1011, "1011th"},
		{1021, "1021st"},
		{-1, "-1st"},
		{-11, "-11th"},
		{-22, "-22nd"},
	}

	for _, c := range cases {
		if got := str.Ordinal(c.n); got != c.want {
			t.Errorf("Ordinal(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

// TestTheTeensAlwaysTakeTh walks every hundred so that a rule written against
// the last digit alone cannot pass.
func TestTheTeensAlwaysTakeTh(t *testing.T) {
	for hundred := 0; hundred < 1000; hundred += 100 {
		for _, n := range []int{11, 12, 13} {
			if got := str.Ordinal(hundred + n); !strings.HasSuffix(got, "th") {
				t.Errorf("Ordinal(%d) = %q, want it to end in th", hundred+n, got)
			}
		}
	}
}

func TestFinish(t *testing.T) {
	cases := []struct {
		s, suffix, want string
	}{
		{"path/to", "/", "path/to/"},
		{"path/to/", "/", "path/to/"},
		{"path/to///", "/", "path/to/"},
		{"", "/", "/"},
		{"/", "/", "/"},
		{"///", "/", "/"},
		{"file", ".go", "file.go"},
		{"file.go", ".go", "file.go"},
		{"file.go.go", ".go", "file.go"},
		{"anything", "", "anything"},
	}

	for _, c := range cases {
		if got := str.Finish(c.s, c.suffix); got != c.want {
			t.Errorf("Finish(%q, %q) = %q, want %q", c.s, c.suffix, got, c.want)
		}
	}
}

func TestStart(t *testing.T) {
	cases := []struct {
		s, prefix, want string
	}{
		{"api/users", "/", "/api/users"},
		{"/api/users", "/", "/api/users"},
		{"//api", "/", "/api"},
		{"", "/", "/"},
		{"///", "/", "/"},
		{"name", "get", "getname"},
		{"getname", "get", "getname"},
		{"anything", "", "anything"},
	}

	for _, c := range cases {
		if got := str.Start(c.s, c.prefix); got != c.want {
			t.Errorf("Start(%q, %q) = %q, want %q", c.s, c.prefix, got, c.want)
		}
	}
}

// TestFinishAndStartSettleAfterOneCall is the property that makes them safe to
// call on a value that has already been through them, which is the whole
// reason they collapse repeats rather than checking for one.
func TestFinishAndStartSettleAfterOneCall(t *testing.T) {
	inputs := []string{"", "/", "///", "a", "a/", "a///", "/a", "///a"}

	for _, in := range inputs {
		once := str.Finish(in, "/")
		if twice := str.Finish(once, "/"); twice != once {
			t.Errorf("Finish(%q) settled on %q then moved to %q", in, once, twice)
		}

		once = str.Start(in, "/")
		if twice := str.Start(once, "/"); twice != once {
			t.Errorf("Start(%q) settled on %q then moved to %q", in, once, twice)
		}
	}
}

func TestReplaceLast(t *testing.T) {
	cases := []struct {
		name              string
		s, old, new, want string
	}{
		{"the last of several", "a/b/c", "/", " and ", "a/b and c"},
		{"the only one", "a/b", "/", "-", "a-b"},
		{"not there", "abc", "/", "-", "abc"},
		{"an empty replacement", "a/b/c", "/", "", "a/bc"},
		{"an empty search", "abc", "", "-", "abc"},
		{"an empty string", "", "/", "-", ""},
		{"a longer search", "one two one", "one", "1", "one two 1"},
		{"overlapping", "aaa", "aa", "b", "ab"},
	}

	for _, c := range cases {
		if got := str.ReplaceLast(c.s, c.old, c.new); got != c.want {
			t.Errorf("%s: ReplaceLast(%q, %q, %q) = %q, want %q", c.name, c.s, c.old, c.new, got, c.want)
		}
	}
}

// TestReplaceLastIsNotReplaceFirst is what the function is for, so it is worth
// one test that would fail if someone reached for strings.Replace with a count
// of one by mistake.
func TestReplaceLastIsNotReplaceFirst(t *testing.T) {
	const s = "one two one two"

	last := str.ReplaceLast(s, "one", "1")
	first := strings.Replace(s, "one", "1", 1)

	if last == first {
		t.Fatalf("both gave %q, want them to touch different occurrences", last)
	}
	if last != "one two 1 two" {
		t.Errorf("ReplaceLast gave %q, want the second one replaced", last)
	}
}
