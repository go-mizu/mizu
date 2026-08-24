package crypt

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-mizu/mizu/errs"
)

// KeySize is how many bytes a key holds.
const KeySize = 32

// keyPrefix is what the text form of a key starts with. The version is part of
// it so that a key written by a later release fails to parse here with
// something to read, rather than decrypting nothing and saying the ciphertext
// was bad.
const keyPrefix = "mizu1:"

// keyLabel separates the hash behind [Key.ID] from any other use of the same
// bytes, so that publishing an id says nothing about a signature made with the
// same key.
const keyLabel = "mizu1 key id"

// Key is the secret an application encrypts and signs with.
//
// A key is 32 bytes. It is written as mizu1: and the bytes in base64url, which
// is what [GenerateKey] and the mizu key:generate command produce and what
// [ParseKey] reads back:
//
//	mizu1:8Qc0kJ4gJx1TgQxKmFQ3s2LcQKp7Y3wZ8n0h0nAe1kU
//
// The bytes stay inside this package. A Key prints, formats, logs and marshals
// as [Redacted] whatever is asked of it, so a key that lands in a struct
// somebody dumps to a log does not land in the log. [Key.Reveal] is the one way
// back to the text, and it is one word to grep for in review.
//
// The zero Key is not a key. It is 32 zero bytes, which would encrypt, so
// everything here that takes a key rejects it. [Key.IsZero] is how a program
// asks whether one was configured.
type Key struct {
	// b is an array rather than a slice so that a Key copies by value, and so
	// that nothing outside this package ends up holding the same bytes.
	b [KeySize]byte
}

// GenerateKey returns a new key from the operating system's random source.
//
// This is what mizu key:generate calls. A program can call it too, for a key
// that lives as long as the process and never reaches disk.
func GenerateKey() Key {
	var k Key

	// crypto/rand.Read fills the slice or crashes the program, so there is no
	// error to return and no half filled key to guard against.
	rand.Read(k.b[:])
	return k
}

// ParseKey reads the text form of a key.
//
// Surrounding whitespace is ignored, since a key usually arrives from an
// environment variable or a file that ends in a newline. Nothing else is: one
// format in, one format out.
func ParseKey(s string) (Key, error) {
	var k Key

	text, ok := strings.CutPrefix(strings.TrimSpace(s), keyPrefix)
	if !ok {
		return k, errs.Invalidf("crypt: a key starts with %q", keyPrefix)
	}

	// The error from the decoder names the byte it stopped at, and the text it
	// stopped in is the key, so it is not passed on.
	b, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return k, errs.Invalidf("crypt: a key is base64url text after %q", keyPrefix)
	}
	if len(b) != KeySize {
		return k, errs.Invalidf("crypt: a key is %d bytes, this one is %d", KeySize, len(b))
	}

	// 32 bytes need 43 characters, which carry 258 bits, so the last character
	// has two bits nothing uses. The decoder throws them away rather than
	// insisting they are zero, which would leave every key with four spellings.
	// Re-encoding says whether this is the one.
	if base64.RawURLEncoding.EncodeToString(b) != text {
		return k, errs.Invalidf("crypt: a key is base64url with no unused bits set")
	}

	copy(k.b[:], b)
	return k, nil
}

// MustParseKey is [ParseKey] for a key written in the source, in a test or an
// example. It panics if the key does not parse.
func MustParseKey(s string) Key {
	k, err := ParseKey(s)
	if err != nil {
		panic(err)
	}
	return k
}

// Reveal returns the text form of the key, the one [ParseKey] reads.
//
// This is the only method that returns the key. Everything else redacts, so a
// search for Reveal finds every place a key can leave the program.
func (k Key) Reveal() string {
	return keyPrefix + base64.RawURLEncoding.EncodeToString(k.b[:])
}

// String returns the prefix and [Redacted], so that a key printed by accident
// looks like a key and is not one.
func (k Key) String() string { return keyPrefix + Redacted }

// ID identifies a key without revealing it: the first eight bytes of SHA-256
// over a fixed label and the key, as hex.
//
// It goes in the header of everything this package encrypts, so that a keyring
// holding several keys knows which one a ciphertext belongs to without trying
// them all. It is safe to log and safe to store next to the data.
func (k Key) ID() string {
	id := k.id()
	return hex.EncodeToString(id[:])
}

// id is [Key.ID] as the bytes that go in a header.
func (k Key) id() [8]byte {
	var buf [len(keyLabel) + KeySize]byte
	copy(buf[:], keyLabel)
	copy(buf[len(keyLabel):], k.b[:])

	sum := sha256.Sum256(buf[:])
	clear(buf[:])

	var id [8]byte
	copy(id[:], sum[:])
	return id
}

// Equal is whether two keys are the same, in constant time.
//
// Two keys also compare with ==, which is not constant time. Use this one where
// the answer is about a key somebody else supplied.
func (k Key) Equal(other Key) bool {
	return subtle.ConstantTimeCompare(k.b[:], other.b[:]) == 1
}

// IsZero is whether this is the zero Key, which is what a Key field holds when
// nothing configured it.
func (k Key) IsZero() bool {
	var zero [KeySize]byte
	return subtle.ConstantTimeCompare(k.b[:], zero[:]) == 1
}

// LogValue is the redacted form, so a key logged as an attribute is masked in
// every handler, including the ones in the standard library.
func (k Key) LogValue() slog.Value { return slog.StringValue(k.String()) }

// Format writes the redacted form for every verb, including %x and the ones
// that print the fields of a struct, which is what this type exists to stop.
func (k Key) Format(f fmt.State, verb rune) { format(f, verb, k.String()) }

// MarshalText writes the redacted form. JSON goes through it too, since a
// [encoding.TextMarshaler] needs nothing else.
//
// A key survives a round trip through text only as far as the program that
// wrote it, which is the point: marshalling a configuration to show somebody
// does not put the key in it. A configuration file gets its key from wherever
// secrets come from, not from a struct that was marshalled earlier.
func (k Key) MarshalText() ([]byte, error) { return []byte(k.String()), nil }

// UnmarshalText reads the text form, so a Key works as a field in a
// configuration struct.
func (k *Key) UnmarshalText(b []byte) error {
	parsed, err := ParseKey(string(b))
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}
