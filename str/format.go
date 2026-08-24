package str

import "net/url"

// IsAscii reports whether every character of s is ASCII.
//
//	str.IsAscii("plain text")   // true
//	str.IsAscii("café")         // false
//
// An empty string is ASCII, since there is nothing in it that is not.
func IsAscii(s string) bool { return isASCII(s) }

// IsURL reports whether s is an absolute URL with a scheme and a host.
//
//	str.IsURL("https://example.com/a?b=1")   // true
//	str.IsURL("example.com")                 // false
//	str.IsURL("mailto:someone@example.com")  // false
//
// The host is what the second and third examples turn on. A bare domain is not
// a URL, and neither is a scheme that addresses something other than a host,
// which is what rules out mailto, data and javascript. That last one is the
// reason this is worth having in front of a redirect or an href.
//
// This is a check on the shape of the string. It says nothing about whether the
// host resolves, whether anything answers, or whether the scheme is one worth
// following, and code that hands the result to a fetcher still has to decide
// which schemes it accepts.
func IsURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// IsUUID reports whether s is a UUID in the canonical 8-4-4-4-12 form.
//
//	str.IsUUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")   // true
//	str.IsUUID("6ba7b8109dad11d180b400c04fd430c8")       // false
//
// The hex digits may be either case. The braced and urn forms are not
// accepted, because a value written either of those ways came from somewhere
// that has not decided what it stores yet.
//
// Nothing is read out of the value, so the version and variant bits are not
// checked and the nil UUID passes. This answers whether a string is shaped like
// a UUID, which is the question a validator and a log redactor are asking. Code
// that needs the version wants a parser.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}

	for i := range len(s) {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			if !isHex(s[i]) {
				return false
			}
		}
	}
	return true
}

// IsULID reports whether s is a ULID: twenty six characters of Crockford base32
// spelling a 128 bit value.
//
//	str.IsULID("01ARZ3NDEKTSV4RRFFQ69G5FAV")   // true
//	str.IsULID("8ZZZZZZZZZZZZZZZZZZZZZZZZZ")   // false, it does not fit
//
// The alphabet is the digits and the Latin letters with I, L, O and U left out,
// so that nothing written in it can be misread as a digit or spell a word by
// accident. Either case is accepted.
//
// The first character has to be a digit under 8. Twenty six characters hold a
// hundred and thirty bits and the value is a hundred and twenty eight, so the
// first one carries three, and a string with anything else in that position
// cannot be a ULID rather than being merely unusual.
//
// Crockford's decoding substitutions are not accepted, so an O in place of a
// zero fails. A canonical encoder never emits one, and a string that needs the
// substitution to parse came from a person typing rather than from a program.
func IsULID(s string) bool {
	if len(s) != 26 || s[0] < '0' || s[0] > '7' {
		return false
	}

	for i := 1; i < len(s); i++ {
		if !isCrockford(s[i]) {
			return false
		}
	}
	return true
}

// isCrockford reports whether c is a character of the Crockford base32
// alphabet, in either case.
func isCrockford(c byte) bool {
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	if c >= '0' && c <= '9' {
		return true
	}
	return c >= 'A' && c <= 'Z' && c != 'I' && c != 'L' && c != 'O' && c != 'U'
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}
