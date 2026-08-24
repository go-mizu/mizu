package crypt

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"

	"github.com/go-mizu/mizu/errs"
)

// A ciphertext is a header and then AES-256-GCM output:
//
//	magic 4 | version 1 | alg 1 | key id 8 | nonce 24 | ciphertext | tag 16
//
// The header is in the clear, since a program has to read the key id before it
// can decrypt anything, and it is authenticated: it goes in as additional data,
// so changing the key id or the version of a stored value makes it stop
// opening rather than open as something else.
const (
	version = 1

	// algXAES is XAES-256-GCM, which is what everything here writes. The byte
	// is in the format so that a later release can add one and still read what
	// this one wrote.
	algXAES = 1

	keyIDSize  = 8
	headerSize = 4 + 1 + 1 + keyIDSize + nonceSize

	// Overhead is how much longer a ciphertext is than the message in it: the
	// header and the tag, 54 bytes. A column that holds encrypted text needs
	// room for it.
	Overhead = headerSize + tagSize
)

// magic is the first four bytes of every ciphertext, so that something looking
// at a column or a file can tell what it is holding.
var magic = [4]byte{'m', 'i', 'z', 'u'}

// AD is data a ciphertext is bound to without being encrypted.
//
// It is the answer to a value being moved somewhere it does not belong: a
// session cookie replayed against another account, a row copied from one tenant
// to another. Encrypt with the id, decrypt with the id, and a value that
// arrives under a different one fails to open.
//
//	b := c.Encrypt(card, crypt.AD("user:"+userID))
//	card, err := c.Decrypt(b, crypt.AD("user:"+userID))
//
// The value has to be there at decryption time, so it is something the caller
// already knows, not something it reads out of the ciphertext.
type AD []byte

// Crypt encrypts and decrypts with a keyring: one key that writes, and any
// number of older ones that still read.
//
//	c, err := crypt.New(cfg.AppKey, cfg.OldKeys...)
//	if err != nil {
//		return err
//	}
//	b := c.Encrypt([]byte("4111 1111 1111 1111"))
//
// There is nothing to choose. Every ciphertext is XAES-256-GCM with a random
// 192 bit nonce, which is authenticated encryption: a value that was changed by
// anybody along the way fails to open rather than opening as something else.
//
// A Crypt is safe for concurrent use and holds no state between messages.
type Crypt struct {
	active *keyed
	byID   map[[keyIDSize]byte]*keyed
}

// New returns a Crypt that encrypts with active and decrypts with any of the
// keys given.
//
// The previous keys are the ones a rotation retired. Ciphertexts written under
// them keep opening, and everything written from here on uses the active key,
// which is what makes a rotation something a program can do while it is
// running. See [Crypt.NeedsRewrap] for the other half of it.
//
// A key that is the zero [Key] is an error, since that is what a configuration
// field holds when nothing filled it in.
func New(active Key, previous ...Key) (*Crypt, error) {
	if active.IsZero() {
		return nil, errs.New(errs.Invalid, "crypt.no_key", "crypt: the active key is not set")
	}

	c := &Crypt{
		active: newKeyed(active),
		byID:   make(map[[keyIDSize]byte]*keyed, len(previous)+1),
	}
	c.byID[active.id()] = c.active

	for i, k := range previous {
		if k.IsZero() {
			return nil, errs.Newf(errs.Invalid, "crypt.no_key", "crypt: previous key %d is not set", i)
		}
		if _, dup := c.byID[k.id()]; dup {
			return nil, errs.Newf(errs.Invalid, "crypt.duplicate_key", "crypt: key %s was given twice", k.ID())
		}
		c.byID[k.id()] = newKeyed(k)
	}
	return c, nil
}

// ActiveID is the id of the key everything is being encrypted with, which is
// worth logging once at startup so that a ciphertext can be traced back to the
// process that wrote it.
func (c *Crypt) ActiveID() string { return c.active.key.ID() }

// Encrypt returns the ciphertext of plaintext, bound to ad.
//
// There is nothing to fail here, so there is no error to check. The key was
// checked when the Crypt was made, and the random source fills the nonce or
// crashes the program.
//
// The result is [Overhead] bytes longer than the plaintext and is not text. Use
// [Crypt.EncryptString] for something that goes in a cookie, a URL or a column
// of characters.
func (c *Crypt) Encrypt(plaintext []byte, ad ...AD) []byte {
	out := make([]byte, headerSize, headerSize+len(plaintext)+tagSize)

	copy(out, magic[:])
	out[4] = version
	out[5] = algXAES
	id := c.active.key.id()
	copy(out[6:], id[:])

	nonce := out[6+keyIDSize : headerSize]
	rand.Read(nonce)

	gcm, nx := c.active.aead(nonce)
	return gcm.Seal(out, nx, plaintext, additional(out, ad))
}

// Decrypt returns the plaintext of a ciphertext that was encrypted with one of
// the keys and bound to the same ad.
//
// The error is [errs.Invalid] whatever went wrong, with a code saying which:
// crypt.malformed for something that is not a ciphertext this package wrote,
// crypt.unknown_key for one written under a key this Crypt does not have, and
// crypt.tampered for one that will not open, which means it was changed, or the
// ad does not match, or the key is not the one it says it is.
//
// None of that says anything about the plaintext, and it is all the caller gets
// to know: a value that does not open is a value that never existed.
func (c *Crypt) Decrypt(b []byte, ad ...AD) ([]byte, error) {
	k, err := c.keyFor(b)
	if err != nil {
		return nil, err
	}

	gcm, nx := k.aead(b[6+keyIDSize : headerSize])
	out, err := gcm.Open(nil, nx, b[headerSize:], additional(b[:headerSize], ad))
	if err != nil {
		return nil, errs.New(errs.Invalid, "crypt.tampered", "crypt: this ciphertext does not open with the key it names")
	}
	return out, nil
}

// EncryptString is [Crypt.Encrypt] for text, and returns base64url without
// padding, which is safe in a URL, a cookie, a header and a filename.
func (c *Crypt) EncryptString(s string, ad ...AD) string {
	return base64.RawURLEncoding.EncodeToString(c.Encrypt([]byte(s), ad...))
}

// DecryptString is [Crypt.Decrypt] for what [Crypt.EncryptString] wrote.
func (c *Crypt) DecryptString(s string, ad ...AD) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", errs.New(errs.Invalid, "crypt.malformed", "crypt: this is not base64url text")
	}

	out, err := c.Decrypt(b, ad...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Rotate returns a Crypt that encrypts with next and still decrypts everything
// this one could.
//
// Rotating costs nothing and changes nothing that is already written. What is
// written stays readable until the key that wrote it is dropped from the list,
// which is a thing to do once [Crypt.NeedsRewrap] stops saying yes.
func (c *Crypt) Rotate(next Key) (*Crypt, error) {
	if _, have := c.byID[next.id()]; have {
		return nil, errs.Newf(errs.Invalid, "crypt.duplicate_key", "crypt: key %s is already in this keyring", next.ID())
	}

	previous := make([]Key, 0, len(c.byID))
	for _, k := range c.byID {
		previous = append(previous, k.key)
	}
	return New(next, previous...)
}

// NeedsRewrap is whether a ciphertext was written under a key that is no longer
// the active one, which is how a program finds the rows still holding the old
// key after a rotation.
//
//	if c.NeedsRewrap(row.Card) {
//		plain, err := c.Decrypt(row.Card)
//		...
//		row.Card = c.Encrypt(plain)
//	}
//
// Something that is not a ciphertext this package wrote answers false. Nothing
// is going to fix that by encrypting it again.
func (c *Crypt) NeedsRewrap(b []byte) bool {
	if err := check(b); err != nil {
		return false
	}
	return [keyIDSize]byte(b[6:6+keyIDSize]) != c.active.key.id()
}

// keyFor is the key a ciphertext names, once the header holds together.
func (c *Crypt) keyFor(b []byte) (*keyed, error) {
	if err := check(b); err != nil {
		return nil, err
	}

	id := [keyIDSize]byte(b[6 : 6+keyIDSize])
	k, ok := c.byID[id]
	if !ok {
		return nil, errs.Newf(errs.Invalid, "crypt.unknown_key", "crypt: no key %x in this keyring", id)
	}
	return k, nil
}

// check is whether b starts like a ciphertext this package wrote and is long
// enough to hold one.
func check(b []byte) error {
	const code = "crypt.malformed"
	switch {
	case len(b) < Overhead:
		return errs.New(errs.Invalid, code, "crypt: this is too short to be a ciphertext")
	case [4]byte(b[:4]) != magic:
		return errs.New(errs.Invalid, code, "crypt: this is not a ciphertext")
	case b[4] != version:
		return errs.Newf(errs.Invalid, code, "crypt: this ciphertext is version %d and this is version %d", b[4], version)
	case b[5] != algXAES:
		return errs.Newf(errs.Invalid, code, "crypt: this ciphertext uses algorithm %d, which this version does not have", b[5])
	}
	return nil
}

// additional is what the tag covers besides the message: the header, and then
// whatever the caller bound the value to.
//
// Each piece is written after its own length, so that two of them cannot be
// rearranged into the same bytes as two others. With nothing bound, this is the
// header itself and costs no copy.
func additional(header []byte, ad []AD) []byte {
	if len(ad) == 0 {
		return header
	}

	size := len(header)
	for _, a := range ad {
		size += binary.MaxVarintLen64 + len(a)
	}

	out := make([]byte, 0, size)
	out = append(out, header...)
	for _, a := range ad {
		out = binary.AppendUvarint(out, uint64(len(a)))
		out = append(out, a...)
	}
	return out
}
