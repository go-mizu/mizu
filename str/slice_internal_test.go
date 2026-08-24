package str

import "testing"

// TestByteAtPastTheEnd covers the contract every caller in the package leans on
// without ever reaching it, since they all clamp against Length first. The
// guard is what makes byteAt safe to call without that clamp, so it is checked
// from inside rather than left as a promise the comment makes and nothing
// keeps.
func TestByteAtPastTheEnd(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want int
	}{
		{"", 0, 0},
		{"", 5, 0},
		{"abc", 0, 0},
		{"abc", 2, 2},
		{"abc", 3, 3},
		{"abc", 10, 3},
		{"a\U0001F1EF\U0001F1F5", 1, 1},
		{"a\U0001F1EF\U0001F1F5", 2, 9},
		{"a\U0001F1EF\U0001F1F5", 99, 9},
	}

	for _, c := range cases {
		if got := byteAt(c.s, c.n); got != c.want {
			t.Errorf("byteAt(%q, %d) = %d, want %d", c.s, c.n, got, c.want)
		}
	}
}
