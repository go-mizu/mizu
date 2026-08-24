package str_test

import (
	"testing"

	"github.com/go-mizu/mizu/str"
)

func TestBefore(t *testing.T) {
	cases := []struct{ s, search, want string }{
		{"user@example.com", "@", "user"},
		{"a/b/c", "/", "a"},
		{"nothing here", "@", "nothing here"},
		{"", "@", ""},
		{"abc", "", "abc"},
		{"@leading", "@", ""},
	}

	for _, c := range cases {
		if got := str.Before(c.s, c.search); got != c.want {
			t.Errorf("Before(%q, %q) = %q, want %q", c.s, c.search, got, c.want)
		}
	}
}

func TestBeforeLast(t *testing.T) {
	cases := []struct{ s, search, want string }{
		{"a/b/c.txt", "/", "a/b"},
		{"one", "/", "one"},
		{"", "/", ""},
		{"abc", "", "abc"},
	}

	for _, c := range cases {
		if got := str.BeforeLast(c.s, c.search); got != c.want {
			t.Errorf("BeforeLast(%q, %q) = %q, want %q", c.s, c.search, got, c.want)
		}
	}
}

func TestAfter(t *testing.T) {
	cases := []struct{ s, search, want string }{
		{"user@example.com", "@", "example.com"},
		{"a/b/c", "/", "b/c"},
		{"nothing here", "@", "nothing here"},
		{"trailing@", "@", ""},
		{"abc", "", "abc"},
	}

	for _, c := range cases {
		if got := str.After(c.s, c.search); got != c.want {
			t.Errorf("After(%q, %q) = %q, want %q", c.s, c.search, got, c.want)
		}
	}
}

func TestAfterLast(t *testing.T) {
	cases := []struct{ s, search, want string }{
		{"a/b/c.txt", "/", "c.txt"},
		{"archive.tar.gz", ".", "gz"},
		{"one", "/", "one"},
		{"abc", "", "abc"},
	}

	for _, c := range cases {
		if got := str.AfterLast(c.s, c.search); got != c.want {
			t.Errorf("AfterLast(%q, %q) = %q, want %q", c.s, c.search, got, c.want)
		}
	}
}

// TestTheCuttingFunctionsHandBackTheWholeStringWhenTheyFindNothing is the one
// surprise in this family, so it is pinned on its own rather than left as a row
// in four separate tables.
func TestTheCuttingFunctionsHandBackTheWholeStringWhenTheyFindNothing(t *testing.T) {
	const s = "no separator in here"

	fns := map[string]func(string, string) string{
		"Before":     str.Before,
		"BeforeLast": str.BeforeLast,
		"After":      str.After,
		"AfterLast":  str.AfterLast,
	}

	for name, fn := range fns {
		if got := fn(s, "@"); got != s {
			t.Errorf("%s found nothing and gave %q, want the whole string", name, got)
		}
	}
}

// TestASearchLongerThanTheString covers the bounds check that a naive index
// plus length would get wrong.
func TestASearchLongerThanTheString(t *testing.T) {
	if got := str.After("ab", "abcdef"); got != "ab" {
		t.Errorf("After gave %q, want the whole string", got)
	}
	if got := str.Before("ab", "abcdef"); got != "ab" {
		t.Errorf("Before gave %q, want the whole string", got)
	}
}

func TestBetween(t *testing.T) {
	cases := []struct{ s, from, to, want string }{
		{"[a] and [b]", "[", "]", "a] and [b"},
		{"<p>text</p>", "<p>", "</p>", "text"},
		{"nothing", "[", "]", "nothing"},
		{"[unclosed", "[", "]", "unclosed"},
	}

	for _, c := range cases {
		if got := str.Between(c.s, c.from, c.to); got != c.want {
			t.Errorf("Between(%q, %q, %q) = %q, want %q", c.s, c.from, c.to, got, c.want)
		}
	}
}

func TestBetweenFirst(t *testing.T) {
	cases := []struct{ s, from, to, want string }{
		{"[a] and [b]", "[", "]", "a"},
		{"<p>text</p>", "<p>", "</p>", "text"},
		{"nothing", "[", "]", "nothing"},
	}

	for _, c := range cases {
		if got := str.BetweenFirst(c.s, c.from, c.to); got != c.want {
			t.Errorf("BetweenFirst(%q, %q, %q) = %q, want %q", c.s, c.from, c.to, got, c.want)
		}
	}
}

// TestBetweenAndBetweenFirstDisagree is what the two are for, so a change that
// made them agree would be a change that lost one of them.
func TestBetweenAndBetweenFirstDisagree(t *testing.T) {
	const s = "[a][b]"

	greedy, lazy := str.Between(s, "[", "]"), str.BetweenFirst(s, "[", "]")
	if greedy == lazy {
		t.Errorf("both gave %q, want Between to reach further", greedy)
	}
	if greedy != "a][b" {
		t.Errorf("Between gave %q, want a][b", greedy)
	}
	if lazy != "a" {
		t.Errorf("BetweenFirst gave %q, want a", lazy)
	}
}
