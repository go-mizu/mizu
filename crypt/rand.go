package crypt

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
)

// Alphanumeric is the alphabet [String] draws from: the digits and both cases
// of the Latin letters, in that order.
const Alphanumeric = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Digits10 is the alphabet [Digits] draws from.
const Digits10 = "0123456789"

// Bytes returns n random bytes from the operating system's random source.
//
// There is no error to handle. The standard library's random source fills the
// slice or crashes the program, on the grounds that a program that cannot get
// random bytes has nothing sensible left to do. It panics if n is negative.
func Bytes(n int) []byte {
	if n < 0 {
		panic("crypt: Bytes with a negative length")
	}
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// String returns n random characters from [Alphanumeric].
//
// Each character carries a little under six bits, so a twenty character string
// is about 119 bits. For anything an attacker gets to guess at, twenty is a
// reasonable floor.
//
// This is the one to reach for when the value ends up somewhere a person reads
// it back to you or types it in. [Token] is shorter for the same entropy and is
// the one for a value that travels in a URL or a header.
func String(n int) string { return Pick(n, Alphanumeric) }

// Token returns a URL safe token holding n bytes of entropy, written in
// base64url without padding.
//
// The string is longer than n: four characters for every three bytes, so
// Token(32) is 43 characters. Ask for bytes of entropy rather than a length,
// since the bytes are the part that matters. 32 is a good default for a session
// id, a password reset link or an unguessable URL.
func Token(n int) string {
	return base64.RawURLEncoding.EncodeToString(Bytes(n))
}

// Digits returns n random decimal digits, for a code somebody reads off a
// screen and types into a phone.
//
// A six digit code is a million values, which is only unguessable because the
// thing checking it counts attempts and stops. Rate limit it and expire it.
func Digits(n int) string { return Pick(n, Digits10) }

// Pick returns n characters drawn from alphabet, each one equally likely.
//
// The alphabet is bytes, not runes, so it is for ASCII. It panics if n is
// negative, if the alphabet is empty, or if it is longer than 256 bytes.
//
// Characters are drawn by rejection: a random byte that falls in the last,
// short run of the alphabet is thrown away rather than folded back to the
// start, which is what would make the first few characters of the alphabet more
// likely than the rest.
func Pick(n int, alphabet string) string {
	if n < 0 {
		panic("crypt: Pick with a negative length")
	}
	if len(alphabet) == 0 || len(alphabet) > 256 {
		panic("crypt: Pick wants an alphabet of 1 to 256 bytes")
	}
	if n == 0 {
		return ""
	}

	size := len(alphabet)
	limit := 256 - 256%size // the first byte value with no whole run under it

	out := make([]byte, 0, n)

	// Room for the bytes that get rejected, so the common case reads once. The
	// worst alphabet, 129 characters, throws away a little under half.
	buf := make([]byte, n+n/2+8)
	for {
		rand.Read(buf)
		for _, b := range buf {
			if int(b) >= limit {
				continue
			}
			out = append(out, alphabet[int(b)%size])
			if len(out) == n {
				return string(out)
			}
		}
	}
}

// Choice returns a random element of s, and panics if s is empty.
func Choice[S ~[]E, E any](s S) E {
	if len(s) == 0 {
		panic("crypt: Choice from an empty slice")
	}
	return s[Intn(len(s))]
}

// Intn returns a random int in [0, n), and panics if n is not positive.
//
// It is [math/rand/v2.IntN] with the operating system's random source behind
// it, for the times when the choice has to be one nobody can predict: which
// server to send a request to is math/rand, which of the tokens somebody
// guessed is not.
func Intn(n int) int {
	if n <= 0 {
		panic("crypt: Intn with a non positive bound")
	}

	// Values at the top of the range are thrown away rather than folded, since
	// folding would make the low values more likely. The bound is the largest
	// value with a whole run of n under it: 2^64 mod n is what has to go, and
	// negating an unsigned n is how to write 2^64 - n without overflowing.
	un := uint64(n)
	limit := ^uint64(0) - (-un)%un

	var buf [8]byte
	for {
		rand.Read(buf[:])
		if v := binary.BigEndian.Uint64(buf[:]); v <= limit {
			return int(v % un)
		}
	}
}
