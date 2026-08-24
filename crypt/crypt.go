package crypt

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"io"

	"github.com/go-mizu/mizu/errs"
)

// Everything this package writes starts with the same prefix, and then differs
// by what the algorithm needs:
//
//	magic 4 | version 1 | alg 1 | key id 8 | nonce 24 | ciphertext | tag 16
//	magic 4 | version 1 | alg 1 | key id 8 | message              | tag 32
//
// The prefix is in the clear, since a program has to read the key id before it
// can do anything else, and it is authenticated, so changing the key id or the
// version of a stored value makes it stop opening rather than open as something
// else.
const (
	version = 1

	// algXAES is XAES-256-GCM and algHMAC is HMAC-SHA256. The byte is in the
	// format so that a later release can add an algorithm and still read what
	// this one wrote.
	algXAES = 1
	algHMAC = 2

	keyIDSize  = 8
	prefixSize = 4 + 1 + 1 + keyIDSize
	headerSize = prefixSize + nonceSize

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
	out := prefix(make([]byte, 0, headerSize+len(plaintext)+tagSize), algXAES, c.active)
	out = out[:headerSize]

	nonce := out[prefixSize:headerSize]
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
	k, err := c.keyFor(b, encrypted)
	if err != nil {
		return nil, err
	}

	gcm, nx := k.aead(b[prefixSize:headerSize])
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

// NeedsRewrap is whether a value was written under a key that is no longer the
// active one, which is how a program finds the rows still holding the old key
// after a rotation.
//
//	if c.NeedsRewrap(row.Card) {
//		plain, err := c.Decrypt(row.Card)
//		...
//		row.Card = c.Encrypt(plain)
//	}
//
// Ciphertexts and signed values both answer. Something this package did not
// write answers false, since nothing is going to fix that by encrypting it
// again.
func (c *Crypt) NeedsRewrap(b []byte) bool {
	for _, f := range []form{encrypted, signed} {
		if f.check(b) == nil {
			return [keyIDSize]byte(b[6:6+keyIDSize]) != c.active.key.id()
		}
	}
	return false
}

// keyFor is the key a value names, once its prefix holds together.
func (c *Crypt) keyFor(b []byte, f form) (*keyed, error) {
	if err := f.check(b); err != nil {
		return nil, err
	}

	id := [keyIDSize]byte(b[6 : 6+keyIDSize])
	k, ok := c.byID[id]
	if !ok {
		return nil, errs.Newf(errs.Invalid, "crypt.unknown_key", "crypt: no key %x in this keyring", id)
	}
	return k, nil
}

// A form is one of the two things this package writes. They differ in the
// algorithm byte, in how much they add to the message, and in what to call them
// when one of them is wrong.
type form struct {
	alg      byte
	overhead int
	noun     string
}

var (
	encrypted = form{algXAES, Overhead, "a ciphertext"}
	signed    = form{algHMAC, SignOverhead, "a signed value"}
)

// check is whether b starts like something this package wrote in this form and
// is long enough to hold it.
func (f form) check(b []byte) error {
	const code = "crypt.malformed"
	switch {
	case len(b) < f.overhead:
		return errs.Newf(errs.Invalid, code, "crypt: this is too short to be %s", f.noun)
	case [4]byte(b[:4]) != magic:
		return errs.Newf(errs.Invalid, code, "crypt: this is not %s", f.noun)
	case b[4] != version:
		return errs.Newf(errs.Invalid, code, "crypt: this value is version %d and this is version %d", b[4], version)
	case b[5] != f.alg:
		return errs.Newf(errs.Invalid, code, "crypt: this value uses algorithm %d, and %s is algorithm %d", b[5], f.noun, f.alg)
	}
	return nil
}

// prefix is the bytes at the front of a value: the magic number, the version,
// the algorithm and the id of the key that wrote it.
func prefix(dst []byte, alg byte, k *keyed) []byte {
	dst = append(dst, magic[:]...)
	dst = append(dst, version, alg)
	id := k.key.id()
	return append(dst, id[:]...)
}

// additional is what the tag covers besides the message: the header, and then
// whatever the caller bound the value to. With nothing bound, this is the header
// itself and costs no copy.
func additional(header []byte, ad []AD) []byte {
	if len(ad) == 0 {
		return header
	}

	size := len(header)
	for _, a := range ad {
		size += binary.MaxVarintLen64 + len(a)
	}

	buf := bytes.NewBuffer(make([]byte, 0, size))
	buf.Write(header)
	bind(buf, ad)
	return buf.Bytes()
}

// bind writes what a value is tied to, each piece after its own length, so that
// two of them cannot be rearranged into the same bytes as two others.
func bind(w io.Writer, ad []AD) {
	var size [binary.MaxVarintLen64]byte
	for _, a := range ad {
		w.Write(binary.AppendUvarint(size[:0], uint64(len(a))))
		w.Write(a)
	}
}
