package str_test

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/str"
)

func TestIsAscii(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"plain text", true},
		{"0123456789", true},
		{"!@#$%^&*()", true},
		{"a\tb\nc", true},
		{"\x7f", true},
		{"café", false},
		{"日本語", false},
		{"🇯🇵", false},
		{"almost ascii ", false},
	}

	for _, c := range cases {
		if got := str.IsAscii(c.in); got != c.want {
			t.Errorf("IsAscii(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestAsciiIsIsAscii ties the two together, since a converter whose output
// fails the matching test is one of them being wrong.
func TestAsciiIsIsAscii(t *testing.T) {
	inputs := []string{"", "plain", "crème brûlée", "Køge", "日本語", "Ωμέγα", "🇯🇵"}

	for _, in := range inputs {
		if got := str.Ascii(in); !str.IsAscii(got) {
			t.Errorf("Ascii(%q) = %q, which IsAscii says is not ascii", in, got)
		}
	}
}

func TestIsURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"the usual one", "https://example.com", true},
		{"with a path and a query", "http://example.com/a/b?c=1#d", true},
		{"with a port", "https://example.com:8080", true},
		{"with credentials", "https://user:pw@example.com/p", true},
		{"an address rather than a name", "http://[::1]:80/", true},
		{"a scheme that is not the web", "ftp://files.example.com/x", true},
		{"a host outside ascii", "http://例え.jp", true},
		{"a bare domain", "example.com", false},
		{"no scheme", "//example.com", false},
		{"a scheme with no host", "http://", false},
		{"mailto", "mailto:someone@example.com", false},
		{"javascript", "javascript:alert(1)", false},
		{"data", "data:text/plain,hi", false},
		{"a path", "/a/b", false},
		{"words", "not a url", false},
		{"nothing", "", false},
		{"a space in the host", "http://exa mple.com", false},
		{"a control character", "http://exam\nple.com", false},
	}

	for _, c := range cases {
		if got := str.IsURL(c.in); got != c.want {
			t.Errorf("%s: IsURL(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

func TestIsUUID(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"version 1", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"version 4", "f47ac10b-58cc-4372-a567-0e02b2c3d479", true},
		{"version 7", "01890a5d-ac96-774b-bcce-b302099a8057", true},
		{"uppercase", "F47AC10B-58CC-4372-A567-0E02B2C3D479", true},
		{"mixed case", "F47ac10B-58cc-4372-A567-0e02b2c3D479", true},
		{"the nil uuid", "00000000-0000-0000-0000-000000000000", true},
		{"a version nobody has defined", "f47ac10b-58cc-f372-a567-0e02b2c3d479", true},
		{"no hyphens", "f47ac10b58cc4372a5670e02b2c3d479", false},
		{"hyphens in the wrong places", "f47ac10b5-8cc-4372-a567-0e02b2c3d479", false},
		{"braced", "{f47ac10b-58cc-4372-a567-0e02b2c3d479}", false},
		{"a urn", "urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479", false},
		{"a character that is not hex", "z47ac10b-58cc-4372-a567-0e02b2c3d479", false},
		{"too short", "f47ac10b-58cc-4372-a567-0e02b2c3d47", false},
		{"too long", "f47ac10b-58cc-4372-a567-0e02b2c3d4799", false},
		{"nothing", "", false},
		{"thirty six of the wrong thing", strings.Repeat("x", 36), false},
	}

	for _, c := range cases {
		if got := str.IsUUID(c.in); got != c.want {
			t.Errorf("%s: IsUUID(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

func TestIsULID(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"the one from the specification", "01ARZ3NDEKTSV4RRFFQ69G5FAV", true},
		{"lowercase", "01arz3ndektsv4rrffq69g5fav", true},
		{"all zeroes", "00000000000000000000000000", true},
		{"the largest one there is", "7ZZZZZZZZZZZZZZZZZZZZZZZZZ", true},
		{"one past the largest", "8ZZZZZZZZZZZZZZZZZZZZZZZZZ", false},
		{"a letter in the first position", "ZZZZZZZZZZZZZZZZZZZZZZZZZZ", false},
		{"an i", "01ARZ3NDEKTSV4RRFFQ69G5FAI", false},
		{"an l", "01ARZ3NDEKTSV4RRFFQ69G5FAL", false},
		{"an o", "01ARZ3NDEKTSV4RRFFQ69G5FAO", false},
		{"a u", "01ARZ3NDEKTSV4RRFFQ69G5FAU", false},
		{"a hyphen", "01ARZ3NDEK-TSV4RRFFQ69G5FA", false},
		{"too short", "01ARZ3NDEKTSV4RRFFQ69G5FA", false},
		{"too long", "01ARZ3NDEKTSV4RRFFQ69G5FAVV", false},
		{"nothing", "", false},
		{"a uuid", "f47ac10b-58cc-4372-a567-0e", false},
	}

	for _, c := range cases {
		if got := str.IsULID(c.in); got != c.want {
			t.Errorf("%s: IsULID(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

// TestTheCrockfordHolesAreTheOnlyHoles walks the whole byte range so that a
// table with a letter missing, or one letter too many, cannot pass on the
// handful of cases above.
func TestTheCrockfordHolesAreTheOnlyHoles(t *testing.T) {
	const allowed = "0123456789ABCDEFGHJKMNPQRSTVWXYZabcdefghjkmnpqrstvwxyz"

	for c := range 256 {
		// The byte goes in second, since the first position has a range of its
		// own. It is built as a byte rather than as a rune, because a rune
		// above 0x7f is two bytes in a string and the test would be about the
		// length instead.
		in := "0" + string([]byte{byte(c)}) + strings.Repeat("0", 24)

		want := strings.IndexByte(allowed, byte(c)) >= 0
		if got := str.IsULID(in); got != want {
			t.Errorf("IsULID with %q in it = %v, want %v", byte(c), got, want)
		}
	}
}

// TestOnlyTheFirstCharacterIsBounded checks that the three bit limit is on the
// position it belongs to and not on every position.
func TestOnlyTheFirstCharacterIsBounded(t *testing.T) {
	for i := range 26 {
		in := []byte(strings.Repeat("0", 26))
		in[i] = 'Z'

		want := i != 0
		if got := str.IsULID(string(in)); got != want {
			t.Errorf("IsULID(%q) with a Z at %d = %v, want %v", in, i, got, want)
		}
	}
}
