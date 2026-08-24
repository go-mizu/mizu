package str_test

import (
	"testing"

	"github.com/go-mizu/mizu/str"
)

func TestPadLeft(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		with string
		want string
	}{
		{"7", 3, "0", "007"},
		{"42", 6, "ab", "abab42"}, // the filler is cut to fit
		{"42", 7, "ab", "ababa42"},
		{"already long", 3, "-", "already long"},
		{"exact", 5, "-", "exact"},
		{"", 3, "-", "---"},
		{"x", 3, "", "x"},
		{"x", -1, "-", "x"},
	}

	for _, c := range cases {
		if got := str.PadLeft(c.s, c.n, c.with); got != c.want {
			t.Errorf("PadLeft(%q, %d, %q) = %q, want %q", c.s, c.n, c.with, got, c.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		with string
		want string
	}{
		{"ana", 6, ".", "ana..."},
		{"ana", 3, ".", "ana"},
		{"", 2, "xy", "xy"},
		{"a", 6, "xy", "axyxyx"},
	}

	for _, c := range cases {
		if got := str.PadRight(c.s, c.n, c.with); got != c.want {
			t.Errorf("PadRight(%q, %d, %q) = %q, want %q", c.s, c.n, c.with, got, c.want)
		}
	}
}

func TestPad(t *testing.T) {
	cases := []struct {
		name string
		s    string
		n    int
		with string
		want string
	}{
		{"an even amount", "ana", 9, "-", "---ana---"},
		{"an odd amount", "ana", 8, "-", "--ana---"},
		{"one short", "ana", 4, "-", "ana-"},
		{"already long", "ana", 2, "-", "ana"},
		{"nothing to pad with", "ana", 9, "", "ana"},
	}

	for _, c := range cases {
		if got := str.Pad(c.s, c.n, c.with); got != c.want {
			t.Errorf("%s: Pad(%q, %d, %q) = %q, want %q", c.name, c.s, c.n, c.with, got, c.want)
		}
	}
}

// TestPaddingCountsCharacters is the reason these are here rather than a format
// verb: fmt pads by bytes, so a name with an accent in it comes out short.
func TestPaddingCountsCharacters(t *testing.T) {
	const name = "José" // five bytes, four characters

	got := str.PadRight(name, 6, ".")
	if want := "José.."; got != want {
		t.Errorf("PadRight(%q, 6) = %q, want %q", name, got, want)
	}
	if str.Length(got) != 6 {
		t.Errorf("the result is %d characters, want 6", str.Length(got))
	}
}

// TestPaddingReachesExactlyTheLengthAsked runs every function over a range of
// lengths, since an off by one in the filler would only show at some of them.
func TestPaddingReachesExactlyTheLengthAsked(t *testing.T) {
	fns := map[string]func(string, int, string) string{
		"PadLeft":  str.PadLeft,
		"PadRight": str.PadRight,
		"Pad":      str.Pad,
	}

	for name, fn := range fns {
		for _, with := range []string{"-", "ab", "xyz"} {
			for n := range 12 {
				got := fn("ana", n, with)
				want := max(n, 3)
				if str.Length(got) != want {
					t.Errorf("%s(%q, %d, %q) = %q, which is %d characters, want %d",
						name, "ana", n, with, got, str.Length(got), want)
				}
			}
		}
	}
}

func TestWrap(t *testing.T) {
	if got := str.Wrap("value", `"`, `"`); got != `"value"` {
		t.Errorf("Wrap gave %q", got)
	}
	if got := str.Wrap("b", "<i>", "</i>"); got != "<i>b</i>" {
		t.Errorf("Wrap gave %q", got)
	}
	if got := str.Wrap("", "[", "]"); got != "[]" {
		t.Errorf("Wrap gave %q, want []", got)
	}
}

func TestUnwrap(t *testing.T) {
	cases := []struct {
		name             string
		s, before, after string
		want             string
	}{
		{"both ends", `"value"`, `"`, `"`, "value"},
		{"tags", "<i>b</i>", "<i>", "</i>", "b"},
		{"only the front", `"value`, `"`, `"`, `"value`},
		{"only the back", `value"`, `"`, `"`, `value"`},
		{"neither", "value", `"`, `"`, "value"},
		{"nothing inside", `""`, `"`, `"`, ""},
		{"a single mark that would be read as both", `"`, `"`, `"`, `"`},
		{"an empty string", "", `"`, `"`, ""},
	}

	for _, c := range cases {
		if got := str.Unwrap(c.s, c.before, c.after); got != c.want {
			t.Errorf("%s: Unwrap(%q) = %q, want %q", c.name, c.s, got, c.want)
		}
	}
}

func TestWrapAndUnwrapComeBack(t *testing.T) {
	for _, s := range []string{"", "value", `has "quotes" inside`} {
		if got := str.Unwrap(str.Wrap(s, `"`, `"`), `"`, `"`); got != s {
			t.Errorf("wrapping and unwrapping %q gave %q", s, got)
		}
	}
}
