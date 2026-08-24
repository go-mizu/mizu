package crypt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/go-mizu/mizu/errs"
)

// hmacSize is what HMAC-SHA256 appends.
const hmacSize = sha256.Size

// signLabel separates the signing subkey from the key it comes out of. It is
// part of the format: changing it makes every signed value stop verifying.
const signLabel = "mizu1 sign"

// SignOverhead is how much longer a signed value is than the message in it: the
// header and the tag, 46 bytes.
const SignOverhead = prefixSize + hmacSize

// Sign returns message with a tag that says it came from here, bound to ad.
//
// The message stays readable. Signing is for values that are not secret but
// must not be changed by whoever is carrying them: a user id in a cookie, an
// unsubscribe link, a webhook payload going somewhere and coming back. When the
// value should not be readable either, encrypt it instead.
//
// The result is [SignOverhead] bytes longer than the message.
func (c *Crypt) Sign(message []byte, ad ...AD) []byte {
	out := prefix(make([]byte, 0, SignOverhead+len(message)), algHMAC, c.active)
	out = append(out, message...)

	tag := c.active.mac(out, ad)
	return append(out, tag[:]...)
}

// Verify returns the message inside a signed value, if it was signed by one of
// the keys and bound to the same ad.
//
// The error is [errs.Invalid] whatever went wrong, with the same codes
// [Crypt.Decrypt] uses: crypt.malformed, crypt.unknown_key and crypt.tampered.
//
// Reading the message out of a value without checking the tag is not something
// this package offers. A message nobody checked is a message somebody else
// wrote.
func (c *Crypt) Verify(b []byte, ad ...AD) ([]byte, error) {
	k, err := c.keyFor(b, signed)
	if err != nil {
		return nil, err
	}

	at := len(b) - hmacSize
	tag := k.mac(b[:at], ad)
	if !hmac.Equal(tag[:], b[at:]) {
		return nil, errs.New(errs.Invalid, "crypt.tampered", "crypt: this value does not carry a tag from the key it names")
	}
	return b[prefixSize:at], nil
}

// SignString is [Crypt.Sign] for text, and returns base64url without padding.
//
// The message is still readable to anybody who decodes it, which is the point
// of signing rather than encrypting.
func (c *Crypt) SignString(s string, ad ...AD) string {
	return base64.RawURLEncoding.EncodeToString(c.Sign([]byte(s), ad...))
}

// VerifyString is [Crypt.Verify] for what [Crypt.SignString] wrote.
func (c *Crypt) VerifyString(s string, ad ...AD) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", errs.New(errs.Invalid, "crypt.malformed", "crypt: this is not base64url text")
	}

	out, err := c.Verify(b, ad...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Seal is a value encrypted as JSON and written as text, which is what a
// session cookie or a signed URL parameter usually holds.
//
//	token, err := crypt.Seal(c, Session{User: id, Until: t}, crypt.AD("session"))
//
// It is a function rather than a method because Go methods cannot have their
// own type parameters.
//
// The error is the one from encoding the value, and it is [errs.Invalid] with
// the code crypt.unencodable. Nothing else here can fail.
func Seal[T any](c *Crypt, v T, ad ...AD) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", errs.Wrapf(err, errs.Invalid, "crypt.unencodable", "crypt: a %T does not encode as JSON", v)
	}
	return base64.RawURLEncoding.EncodeToString(c.Encrypt(b, ad...)), nil
}

// Unseal is the value inside what [Seal] wrote.
//
//	s, err := crypt.Unseal[Session](c, token, crypt.AD("session"))
//
// The error is whatever [Crypt.DecryptString] would have returned, or
// crypt.malformed if what came out is not the JSON of a T, which is what a value
// sealed before the type changed looks like.
func Unseal[T any](c *Crypt, s string, ad ...AD) (T, error) {
	var zero T

	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return zero, errs.New(errs.Invalid, "crypt.malformed", "crypt: this is not base64url text")
	}

	plain, err := c.Decrypt(b, ad...)
	if err != nil {
		return zero, err
	}

	var v T
	if err := json.Unmarshal(plain, &v); err != nil {
		return zero, errs.Wrapf(err, errs.Invalid, "crypt.malformed", "crypt: what this opened is not a %T", zero)
	}
	return v, nil
}

// mac is the tag over b and whatever the value is bound to.
func (k *keyed) mac(b []byte, ad []AD) [hmacSize]byte {
	h := hmac.New(sha256.New, k.signKey[:])
	h.Write(b)
	bind(h, ad)

	var tag [hmacSize]byte
	h.Sum(tag[:0])
	return tag
}
