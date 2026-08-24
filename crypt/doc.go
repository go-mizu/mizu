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
// # Encryption
//
// A [Crypt] encrypts and decrypts with a keyring: one key that writes, and any
// number of retired ones that still read.
//
//	c, err := crypt.New(cfg.AppKey, cfg.OldKeys...)
//	if err != nil {
//		return err
//	}
//	b := c.Encrypt([]byte("4111 1111 1111 1111"))
//	card, err := c.Decrypt(b)
//
// There is nothing to select. Every ciphertext is XAES-256-GCM with a random
// 192 bit nonce, which is authenticated encryption: a value that anybody changed
// along the way fails to open rather than opening as something else. The nonce
// is long enough that drawing it at random is the whole of the answer, so there
// is no counter to keep and nothing that goes wrong when a process restarts.
//
// [Crypt.EncryptString] is the same thing for values that have to be text, and
// returns base64url with no padding, which is safe in a URL, a cookie, a header
// and a filename.
//
// A ciphertext starts with a header holding the version, the algorithm and
// [Key.ID], so a stored value says which key opens it. The header is
// authenticated along with the message. [Overhead] is how much longer a
// ciphertext is than what went into it, which is what a column has to have room
// for.
//
// # Binding
//
// [AD] is data a ciphertext is tied to without being encrypted, and it is the
// answer to a value being moved somewhere it does not belong: a session cookie
// replayed against another account, a row copied from one tenant to another.
//
//	b := c.Encrypt(card, crypt.AD("user:"+userID))
//	card, err := c.Decrypt(b, crypt.AD("user:"+userID))
//
// A value that arrives under a different id fails to open. The binding is
// something the caller already knows at decryption time, not something it reads
// out of the ciphertext, which is what makes it worth anything.
//
// # Signing
//
// [Crypt.Sign] leaves the message readable and adds a tag that says it came from
// here, which is what a value that is not secret but must not be changed needs:
// a user id in a cookie, an unsubscribe link, a payload that goes out to a
// service and comes back.
//
//	b := c.Sign([]byte("user:42"))
//	who, err := c.Verify(b)
//
// [Crypt.Verify] is the only way to read one. A message nobody checked the tag
// on is a message somebody else may have written, so there is no call here that
// hands it back without checking.
//
// Signing is HMAC-SHA256 under a subkey derived from the key, so the bytes that
// sign are never the bytes that encrypt. It has no nonce, so the same message
// signs the same way every time.
//
// # Values
//
// [Seal] and [Unseal] are a value in and a value out, encoded as JSON and
// encrypted, which is what a session cookie or a signed URL parameter usually
// holds.
//
//	token, err := crypt.Seal(c, Session{User: id}, crypt.AD("session"))
//	s, err := crypt.Unseal[Session](c, token, crypt.AD("session"))
//
// They are functions rather than methods because Go methods cannot have their
// own type parameters. A token that no longer decodes as a T fails the same way
// a changed one does, which is what a cookie from before a deploy looks like.
//
// # Rotation
//
// [Crypt.Rotate] returns a Crypt that writes with a new key and still reads
// everything the old one could, so replacing a key is not a migration that has
// to finish before the application starts again.
//
//	c, err = c.Rotate(next)
//
// [Crypt.NeedsRewrap] finds what is still holding an old key, which is how a
// background job works through stored rows until the retired key can be dropped
// from the list.
//
// # Random
//
// Everything here draws from the operating system, through crypto/rand, so
// there is no seed to get wrong and no fast path that is not random enough.
//
//	crypt.Token(32)    // a session id, a reset link, an unguessable URL
//	crypt.String(20)   // characters a person can read back to you
//	crypt.Digits(6)    // a code somebody types into a phone
//	crypt.Password(24) // a first password for an account somebody else opened
//	crypt.Bytes(16)    // bytes for something else to use
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
