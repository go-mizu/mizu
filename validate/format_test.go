package validate

import (
	"strings"
	"testing"
)

// check runs one predicate over a list of strings that pass and a list that
// does not, and says which string it was looking at when it disagreed.
func check(t *testing.T, name string, fn func(string) bool, good, bad []string) {
	t.Helper()

	for _, s := range good {
		if !fn(s) {
			t.Errorf("%s(%q) = false, want true", name, s)
		}
	}
	for _, s := range bad {
		if fn(s) {
			t.Errorf("%s(%q) = true, want false", name, s)
		}
	}
}

func TestIsEmail(t *testing.T) {
	check(t, "IsEmail", IsEmail,
		[]string{
			"a@b.co",
			"user@example.com",
			"first.last@example.com",
			"user+tag@sub.example.co.uk",
			"!#$%&'*+-/=?^_`{|}~@example.com",
			"USER@EXAMPLE.COM",
			"a@x-1.example",
		},
		[]string{
			"",
			"@example.com",
			"user@",
			"user",
			"user@localhost",
			"user@example.c",
			"user@example.c0m",
			"user@-example.com",
			"user@example..com",
			".user@example.com",
			"user.@example.com",
			"us..er@example.com",
			"us er@example.com",
			`"us er"@example.com`,
			"Name <user@example.com>",
			strings.Repeat("a", 65) + "@example.com",
		},
	)
}

// The last @ wins, so an address with one in the local part is refused on the
// local part rather than read as a shorter address with a longer domain.
func TestIsEmailSplitsOnTheLastAt(t *testing.T) {
	if IsEmail("a@b@example.com") {
		t.Error("IsEmail read a@b as a local part")
	}
}

func TestIsURL(t *testing.T) {
	check(t, "IsURL", IsURL,
		[]string{
			"http://example.com",
			"https://example.com/",
			"https://example.com:8443/path?q=1#top",
			"http://localhost:3000",
			"http://192.0.2.1/health",
			"http://[2001:db8::1]/health",
			"http://[2001:db8::1]:8080/",
			"https://user:pass@example.com/",
		},
		[]string{
			"",
			"/path",
			"example.com",
			"ftp://example.com",
			"mailto:user@example.com",
			"javascript:alert(1)",
			"http://",
			"http://exa mple.com",
			"http://example.com\n",
			"http://-example.com",
			"http://example.com:0",
			"http://example.com:99999",
			"http://[example]/",
			":no-scheme",
		},
	)
}

func TestIsURI(t *testing.T) {
	check(t, "IsURI", IsURI,
		[]string{
			"http://example.com",
			"ftp://files.example.com/pub",
			"mailto:user@example.com",
			"urn:isbn:0451450523",
			"tel:+15551234567",
			"postgres://localhost/app?sslmode=disable",
		},
		[]string{
			"",
			"/path",
			"example.com",
			"//example.com",
			"urn isbn:0451450523",
			":no-scheme",
		},
	)
}

// A URL is a URI, so anything the stricter rule takes the looser one takes too.
func TestEveryURLIsAURI(t *testing.T) {
	for _, s := range []string{"http://example.com", "https://a.example:8080/x", "http://[2001:db8::1]/"} {
		if !IsURI(s) {
			t.Errorf("IsURL(%q) is true and IsURI is not", s)
		}
	}
}

// net/url refuses a host with an unmatched bracket or junk where the port goes,
// so IsURL never sees one. isHost still answers for them, because a function
// that only works on the inputs one caller happens to send is one that breaks
// when a second caller arrives.
func TestIsHostAnswersForWhatURLParseWouldHaveRejected(t *testing.T) {
	check(t, "isHost", isHost,
		[]string{"[2001:db8::1]", "[2001:db8::1]:80", "example.com", "10.0.0.1:443"},
		[]string{"[2001:db8::1", "[2001:db8::1]x", "[2001:db8::1]:", "[example]", "", "example.com:"},
	)
}

func TestIsHostname(t *testing.T) {
	check(t, "IsHostname", IsHostname,
		[]string{
			"localhost",
			"example.com",
			"sub.domain.example.co.uk",
			"x",
			"1",
			"a-b.example",
			"192.0.2.1",
			strings.Repeat("a", 63) + ".example.com",
		},
		[]string{
			"",
			".",
			"example..com",
			".example.com",
			"example.com.",
			"-example.com",
			"example-.com",
			"exa_mple.com",
			"exa mple.com",
			strings.Repeat("a", 64) + ".example.com",
			strings.Repeat("a.", 130) + "com",
		},
	)
}

func TestIsIP(t *testing.T) {
	check(t, "IsIP", IsIP,
		[]string{"0.0.0.0", "192.0.2.1", "255.255.255.255", "::1", "2001:db8::1", "::ffff:192.0.2.1"},
		[]string{"", "192.0.2", "192.0.2.256", "192.0.2.1/32", "fe80::1%eth0", "example.com"},
	)
}

// An IPv4 address written as one is IPv4 and an IPv6 address that holds an
// IPv4 one is IPv6, so the two rules never both pass.
func TestIPv4AndIPv6DoNotOverlap(t *testing.T) {
	check(t, "IsIPv4", IsIPv4,
		[]string{"192.0.2.1", "0.0.0.0"},
		[]string{"::1", "2001:db8::1", "::ffff:192.0.2.1", "", "example.com"},
	)
	check(t, "IsIPv6", IsIPv6,
		[]string{"::1", "2001:db8::1", "::ffff:192.0.2.1"},
		[]string{"192.0.2.1", "0.0.0.0", "", "example.com", "fe80::1%eth0"},
	)
}

func TestIsCIDR(t *testing.T) {
	check(t, "IsCIDR", IsCIDR,
		[]string{"10.0.0.0/8", "10.0.0.7/8", "192.0.2.1/32", "2001:db8::/32", "::/0"},
		[]string{"", "10.0.0.0", "10.0.0.0/33", "10.0.0.0/-1", "2001:db8::/129", "fe80::1%eth0/64"},
	)
}

func TestIsMAC(t *testing.T) {
	check(t, "IsMAC", IsMAC,
		[]string{
			"00:1b:44:11:3a:b7",
			"00-1B-44-11-3A-B7",
			"00:1b:44:11:3a:b7:00:00",
			"00-1b-44-11-3a-b7-00-00",
			"0011.2233.4455",
			"0011.2233.4455.6677",
		},
		[]string{
			"",
			"001b44113ab7",
			"00:1b:44:11:3a",
			"00:1b:44:11:3a:b7:00",
			"00:1b:44:11:3a:bg",
			"00:1b:44:11:3a:b7:",
			"00-1b:44:11:3a:b7",
			"001:1:44:11:3a:b7",
			"0011.2233",
			"001.2233.4455",
			"0011.223.34455",
			"0011.2233.445g",
		},
	)
}

func TestIsPort(t *testing.T) {
	check(t, "IsPort", IsPort,
		[]string{"1", "80", "8080", "65535"},
		[]string{"", "0", "00", "080", "65536", "99999", "123456", "-1", "80 ", "http"},
	)
}

func TestIsUUID(t *testing.T) {
	check(t, "IsUUID", IsUUID,
		[]string{
			"00000000-0000-0000-0000-000000000000",
			"f47ac10b-58cc-4372-a567-0e02b2c3d479",
			"F47AC10B-58CC-4372-A567-0E02B2C3D479",
			"f47ac10b-58cc-9372-e567-0e02b2c3d479",
		},
		[]string{
			"",
			"f47ac10b58cc4372a5670e02b2c3d479",
			"f47ac10b-58cc-4372-a567-0e02b2c3d47",
			"f47ac10b-58cc-4372-a567-0e02b2c3d4790",
			"f47ac10b_58cc-4372-a567-0e02b2c3d479",
			"f47ac10b-58cc-4372-a567-0e02b2c3d47g",
			"{f47ac10b-58cc-4372-a567-0e02b2c3d479}",
			"urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479",
		},
	)
}

func TestIsULID(t *testing.T) {
	check(t, "IsULID", IsULID,
		[]string{
			"01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"01arz3ndektsv4rrffq69g5fav",
			"00000000000000000000000000",
			"7ZZZZZZZZZZZZZZZZZZZZZZZZZ",
		},
		[]string{
			"",
			"01ARZ3NDEKTSV4RRFFQ69G5FA",
			"01ARZ3NDEKTSV4RRFFQ69G5FAVV",
			"8ZZZZZZZZZZZZZZZZZZZZZZZZZ",
			"01ARZ3NDEKTSV4RRFFQ69G5FAI",
			"01ARZ3NDEKTSV4RRFFQ69G5FAL",
			"01ARZ3NDEKTSV4RRFFQ69G5FAO",
			"01ARZ3NDEKTSV4RRFFQ69G5FAU",
			"I1ARZ3NDEKTSV4RRFFQ69G5FAV",
			"01ARZ3NDEKTSV4RRFFQ69G5FA-",
		},
	)
}

func TestIsE164(t *testing.T) {
	check(t, "IsE164", IsE164,
		[]string{"+15551234567", "+442071838750", "+12", "+123456789012345"},
		[]string{
			"",
			"+",
			"+1",
			"+0123456789",
			"+1234567890123456",
			"15551234567",
			"+1 555 123 4567",
			"+1-555-123-4567",
			"+1555123456a",
		},
	)
}

// Every rule this file ships has a sentence to go with it, because a rule with
// no entry falls back to "X is not valid.", which is what somebody sees when
// they cannot work out what to type.
func TestEveryFormatHasAMessage(t *testing.T) {
	for name := range formats {
		tpl, ok := templates[name]
		if !ok {
			t.Errorf("format %q has no English message", name)
			continue
		}
		if tpl.params != 0 {
			t.Errorf("format %q wants %d parameters, and a format check has none", name, tpl.params)
		}
	}
}

// A rule name is what somebody writes in a struct tag, so it is one lower case
// word with nothing in it that needs quoting.
func TestFormatNamesAreTagSafe(t *testing.T) {
	for name := range formats {
		if name == "" {
			t.Error("a format has no name")
			continue
		}
		for i := range len(name) {
			if c := name[i]; !isDigit(c) && !(c >= 'a' && c <= 'z') {
				t.Errorf("format %q is not a lower case word", name)
				break
			}
		}
	}
}

// The table is what the interpreter will read, so the entry for a name has to
// be the function that name is about. A copied line that points at the
// neighbour above it passes every other test in this file.
func TestFormatsPointAtTheRightFunction(t *testing.T) {
	cases := []struct {
		name  string
		takes string
	}{
		{"cidr", "10.0.0.0/8"},
		{"e164", "+15551234567"},
		{"email", "user@example.com"},
		{"hostname", "example.com"},
		{"ip", "2001:db8::1"},
		{"ipv4", "192.0.2.1"},
		{"ipv6", "2001:db8::1"},
		{"mac", "00:1b:44:11:3a:b7"},
		{"port", "8080"},
		{"ulid", "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		{"uri", "urn:isbn:0451450523"},
		{"url", "https://example.com/"},
		{"uuid", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
	}

	if len(cases) != len(formats) {
		t.Fatalf("%d formats and %d cases, add the new one here", len(formats), len(cases))
	}
	for _, c := range cases {
		fn, ok := formats[c.name]
		if !ok {
			t.Errorf("no format named %q", c.name)
			continue
		}
		if !fn(c.takes) {
			t.Errorf("formats[%q](%q) = false", c.name, c.takes)
		}
	}
}

// The alphabet is the thing a ULID is read against, and a typo in it would let
// through the four characters it exists to keep out.
func TestCrockfordLeavesOutTheLettersThatLookLikeDigits(t *testing.T) {
	if len(crockford) != 32 {
		t.Fatalf("crockford has %d characters, want 32", len(crockford))
	}
	for _, c := range "ILOU" {
		if strings.ContainsRune(crockford, c) {
			t.Errorf("crockford contains %q", c)
		}
	}
	seen := map[byte]bool{}
	for i := range len(crockford) {
		if seen[crockford[i]] {
			t.Errorf("crockford repeats %q", crockford[i])
		}
		seen[crockford[i]] = true
	}
}

// FuzzFormats says the checks answer rather than panic, and that the ones that
// are stricter versions of each other stay that way.
func FuzzFormats(f *testing.F) {
	for _, s := range []string{
		"", "a@b.co", "http://example.com:8080/x", "2001:db8::1", "10.0.0.0/8",
		"00:1b:44:11:3a:b7", "01ARZ3NDEKTSV4RRFFQ69G5FAV", "+15551234567",
		"f47ac10b-58cc-4372-a567-0e02b2c3d479", "[::1]", "urn:isbn:1",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		for _, fn := range formats {
			_ = fn(s)
		}

		if IsURL(s) && !IsURI(s) {
			t.Errorf("IsURL(%q) and not IsURI", s)
		}
		if IsIPv4(s) && !IsIP(s) {
			t.Errorf("IsIPv4(%q) and not IsIP", s)
		}
		if IsIPv6(s) && !IsIP(s) {
			t.Errorf("IsIPv6(%q) and not IsIP", s)
		}
		if IsIPv4(s) && IsIPv6(s) {
			t.Errorf("%q is both families", s)
		}
		if IsEmail(s) {
			at := strings.LastIndexByte(s, '@')
			if !IsHostname(s[at+1:]) {
				t.Errorf("IsEmail(%q) and its domain is not a hostname", s)
			}
		}
	})
}
