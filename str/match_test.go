package str_test

import (
	"path"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/str"
)

func TestIs(t *testing.T) {
	cases := []struct {
		name        string
		pattern, in string
		want        bool
	}{
		{"no wildcard, the same", "user", "user", true},
		{"no wildcard, different", "user", "users", false},
		{"a trailing star", "user*", "users", true},
		{"a trailing star matching nothing", "user*", "user", true},
		{"a leading star", "*.jpg", "photo.jpg", true},
		{"a leading star matching nothing", "*.jpg", ".jpg", true},
		{"a star in the middle", "a*z", "abcz", true},
		{"a star in the middle matching nothing", "a*z", "az", true},
		{"two stars", "a*b*c", "a1b2c", true},
		{"two stars, out of order", "a*b*c", "a1c2b", false},
		{"a star on its own", "*", "anything at all", true},
		{"a star on its own against nothing", "*", "", true},
		{"the ends have to reach", "ab*ba", "aba", false},
		{"an empty pattern", "", "", true},
		{"an empty pattern against something", "", "x", false},
		{"an empty subject", "*", "", true},
		{"a repeated middle", "*a*a*", "xaya", true},
		{"a question mark is a literal", "v?", "v1", false},
		{"a question mark matches itself", "v?", "v?", true},
		{"a dot is a literal", "a.c", "abc", false},
		{"case matters", "User*", "user1", false},
		{"characters outside ascii", "caf*", "café", true},
	}

	for _, c := range cases {
		if got := str.Is(c.pattern, c.in); got != c.want {
			t.Errorf("%s: Is(%q, %q) = %v, want %v", c.name, c.pattern, c.in, got, c.want)
		}
	}
}

// TestAStarCrossesASlash is the difference from [path.Match] and the reason
// this exists, so it is checked against path.Match rather than described.
func TestAStarCrossesASlash(t *testing.T) {
	const pattern, subject = "admin/*", "admin/users/1"

	if !str.Is(pattern, subject) {
		t.Errorf("Is(%q, %q) = false, want a star to cross the slash", pattern, subject)
	}

	ok, err := path.Match(pattern, subject)
	if err != nil {
		t.Fatalf("path.Match: %v", err)
	}
	if ok {
		t.Fatalf("path.Match(%q, %q) is true, so there would be nothing to have here", pattern, subject)
	}
}

// TestIsAgreesWithTheObviousImplementation checks the pieces-and-anchors walk
// against what the pattern means, on inputs short enough to enumerate. The two
// are written differently on purpose: the reference is a backtracking match
// nobody would ship, and its only job is to disagree if the fast one is wrong.
func TestIsAgreesWithTheObviousImplementation(t *testing.T) {
	alphabet := []string{"a", "b", "*"}
	patterns := grow(alphabet, 4)
	subjects := grow([]string{"a", "b"}, 4)

	for _, pattern := range patterns {
		for _, subject := range subjects {
			got := str.Is(pattern, subject)
			want := backtrack(pattern, subject)
			if got != want {
				t.Fatalf("Is(%q, %q) = %v, want %v", pattern, subject, got, want)
			}
		}
	}
}

// grow returns every string of up to n characters over alphabet, including the
// empty one.
func grow(alphabet []string, n int) []string {
	out := []string{""}
	level := []string{""}
	for range n {
		var next []string
		for _, s := range level {
			for _, c := range alphabet {
				next = append(next, s+c)
			}
		}
		out = append(out, next...)
		level = next
	}
	return out
}

// backtrack answers the same question as [str.Is] by trying every place a star
// could stop.
func backtrack(pattern, s string) bool {
	if pattern == "" {
		return s == ""
	}
	if pattern[0] == '*' {
		for i := 0; i <= len(s); i++ {
			if backtrack(pattern[1:], s[i:]) {
				return true
			}
		}
		return false
	}
	return s != "" && s[0] == pattern[0] && backtrack(pattern[1:], s[1:])
}

func FuzzIs(f *testing.F) {
	f.Add("a*b", "axxb")
	f.Add("*", "")
	f.Add("ab*ba", "aba")
	f.Add("*a*a*", "xaya")

	f.Fuzz(func(t *testing.T, pattern, s string) {
		// The reference implementation is exponential in the number of stars,
		// so the fuzzer is kept to inputs it can answer.
		if len(pattern) > 12 || len(s) > 12 || strings.Count(pattern, "*") > 4 {
			t.Skip()
		}

		if got, want := str.Is(pattern, s), backtrack(pattern, s); got != want {
			t.Errorf("Is(%q, %q) = %v, want %v", pattern, s, got, want)
		}
	})
}
