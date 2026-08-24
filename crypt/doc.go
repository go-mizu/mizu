// Package crypt holds the keys an application keeps, the values it hides, and
// the random it draws.
//
// There are no algorithms to choose here and no options to get wrong. The
// standard library already has the pieces, and what is missing from an
// application is everything around them: where the key comes from, how it is
// named, what happens when it is replaced, and how a secret is kept out of the
// places values end up.
//
// # Keys
//
// A [Key] is 32 bytes, written as mizu1: and base64url:
//
//	key := crypt.GenerateKey()
//	fmt.Println(key.Reveal()) // mizu1:8Qc0kJ4gJx1TgQxKmFQ3s2LcQKp7Y3wZ8n0h0nAe1kU
//
// That text is what goes in the environment, and [ParseKey] reads it back. A
// Key is also a [encoding.TextUnmarshaler], so it works as a field of a
// configuration struct and arrives parsed.
//
// The bytes go no further. A Key prints, formats, logs and marshals as
// [Redacted] whatever anybody asks of it, and [Key.Reveal] is the single way
// back to the text. Reveal is a word to grep for in review, which is the point
// of it being a method rather than a field.
//
// [Key.ID] identifies a key without revealing it, as the first eight bytes of a
// hash. It is safe to log, safe to store next to the data, and it is what lets
// a ciphertext say which key it belongs to once an application has more than
// one.
//
// # Secrets
//
// A [Secret] is a string that does not print: a password, an API token, a
// webhook signing key.
//
//	type Config struct {
//		DatabaseURL string
//		StripeKey   crypt.Secret
//	}
//
//	fmt.Println(cfg) // {postgres://... [redacted]}
//
// The masking travels with the value, through every struct it is a field of and
// every handler it reaches, which is the part that a list of key names to
// redact cannot do. [Secret.Equal] compares in constant time, for the ones that
// arrive from outside and get checked.
//
// # Random
//
// Everything here draws from the operating system, through crypto/rand, so
// there is no seed to get wrong and no fast path that is not random enough.
//
//	crypt.Token(32)  // a session id, a reset link, an unguessable URL
//	crypt.String(20) // characters a person can read back to you
//	crypt.Digits(6)  // a code somebody types into a phone
//	crypt.Bytes(16)  // bytes for something else to use
//
// None of them return an error. The standard library's random source fills the
// slice or crashes the program, and a program that cannot get random bytes has
// nothing sensible left to do.
//
// [String], [Digits] and [Pick] draw characters by rejection, so every
// character of the alphabet is equally likely. Folding a random byte into the
// alphabet with a remainder is the usual way to get this wrong, and it makes
// the first few characters more likely than the rest.
//
// For a choice that does not have to be unguessable, such as which of three
// servers to try first, use math/rand/v2. It is faster and this is not what it
// is for.
package crypt
