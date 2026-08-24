package crypt

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

// newTest is a Crypt over freshly generated keys.
func newTest(t *testing.T, previous ...Key) *Crypt {
	t.Helper()
	c, err := New(GenerateKey(), previous...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCrypt(t *testing.T) {
	c := newTest(t)
	plaintext := []byte("4111 1111 1111 1111")

	b := c.Encrypt(plaintext)
	if bytes.Contains(b, plaintext) {
		t.Fatal("the plaintext is in the ciphertext")
	}
	if len(b) != len(plaintext)+Overhead {
		t.Errorf("a %d byte message came out %d bytes, want %d", len(plaintext), len(b), len(plaintext)+Overhead)
	}

	got, err := c.Decrypt(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("it came back as %q", got)
	}
}

// TestCryptSizes covers the ends: nothing at all, and something longer than one
// block.
func TestCryptSizes(t *testing.T) {
	c := newTest(t)
	for _, n := range []int{0, 1, 15, 16, 17, 1000} {
		plaintext := bytes.Repeat([]byte{'a'}, n)

		got, err := c.Decrypt(c.Encrypt(plaintext))
		if err != nil {
			t.Fatalf("%d bytes: %v", n, err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Errorf("%d bytes came back as %d", n, len(got))
		}
	}

	if _, err := c.Decrypt(c.Encrypt(nil)); err != nil {
		t.Errorf("a nil message: %v", err)
	}
}

// TestCryptNonce is what the 192 bit nonce is for: the same message under the
// same key comes out different every time, without anything having to count.
func TestCryptNonce(t *testing.T) {
	c := newTest(t)
	plaintext := []byte("the same message")

	seen := map[string]bool{}
	for range 100 {
		b := c.Encrypt(plaintext)
		body := string(b[headerSize:])
		if seen[body] {
			t.Fatal("two encryptions of the same message came out the same")
		}
		seen[body] = true
	}
}

// TestCryptHeader pins the format down. It is on disk in other people's
// databases, so it is not a thing to change quietly.
func TestCryptHeader(t *testing.T) {
	key := GenerateKey()
	c, err := New(key)
	if err != nil {
		t.Fatal(err)
	}

	b := c.Encrypt([]byte("x"))
	if got := string(b[:4]); got != "mizu" {
		t.Errorf("the magic is %q, want mizu", got)
	}
	if b[4] != 1 {
		t.Errorf("the version is %d, want 1", b[4])
	}
	if b[5] != 1 {
		t.Errorf("the algorithm is %d, want 1", b[5])
	}
	if got, want := [8]byte(b[6:14]), key.id(); got != want {
		t.Errorf("the key id is %x, want %x", got, want)
	}
	if nonce := b[14:38]; allZero(nonce) {
		t.Error("the nonce is all zeroes")
	}
	if headerSize != 38 || Overhead != 54 {
		t.Errorf("the header is %d bytes and the overhead %d, want 38 and 54", headerSize, Overhead)
	}
}

// TestCryptHeaderIsAuthenticated is the reason the header goes in as additional
// data. Changing what a ciphertext says about itself has to break it.
func TestCryptHeaderIsAuthenticated(t *testing.T) {
	c := newTest(t)
	b := c.Encrypt([]byte("x"))

	for _, i := range []int{6, 13, 14, 37} {
		changed := bytes.Clone(b)
		changed[i]++

		if _, err := c.Decrypt(changed); err == nil {
			t.Errorf("byte %d of the header changed and it still opened", i)
		}
	}
}

func TestCryptAD(t *testing.T) {
	c := newTest(t)
	plaintext := []byte("4111 1111 1111 1111")

	b := c.Encrypt(plaintext, AD("user:42"))
	if _, err := c.Decrypt(b, AD("user:42")); err != nil {
		t.Fatalf("the value it was bound to: %v", err)
	}

	// The point of it: the same ciphertext under somebody else's id.
	if _, err := c.Decrypt(b, AD("user:43")); err == nil {
		t.Error("a value bound to one user opened under another")
	}
	if errs.CodeOf(mustFail(t, c, b, AD("user:43"))) != "crypt.tampered" {
		t.Error("the wrong binding is not reported as tampering")
	}
	if _, err := c.Decrypt(b); err == nil {
		t.Error("a bound value opened with nothing bound")
	}
	if _, err := c.Decrypt(c.Encrypt(plaintext), AD("user:42")); err == nil {
		t.Error("an unbound value opened with something bound")
	}
}

// TestCryptADIsUnambiguous is why each piece goes in after its own length.
// Without that, two bindings could be rearranged into the same bytes as two
// others and a value would open under the wrong one.
func TestCryptADIsUnambiguous(t *testing.T) {
	c := newTest(t)

	b := c.Encrypt([]byte("x"), AD("user:4"), AD("2"))
	if _, err := c.Decrypt(b, AD("user:4"), AD("2")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Decrypt(b, AD("user:"), AD("42")); err == nil {
		t.Error("two bindings rearranged and the value still opened")
	}
	if _, err := c.Decrypt(b, AD("user:42")); err == nil {
		t.Error("two bindings joined into one and the value still opened")
	}

	// An empty binding is a binding, and is not the same as no binding at all.
	empty := c.Encrypt([]byte("x"), AD(""))
	if _, err := c.Decrypt(empty); err == nil {
		t.Error("a value bound to nothing opened with no binding")
	}
	if _, err := c.Decrypt(empty, AD("")); err != nil {
		t.Errorf("a value bound to nothing: %v", err)
	}
}

// TestCryptTampered walks every byte of a ciphertext. Not one of them is
// allowed to change without the value refusing to open.
func TestCryptTampered(t *testing.T) {
	c := newTest(t)
	b := c.Encrypt([]byte("4111 1111 1111 1111"))

	for i := range b {
		changed := bytes.Clone(b)
		changed[i] ^= 0x40

		if _, err := c.Decrypt(changed); err == nil {
			t.Fatalf("byte %d changed and the value still opened", i)
		}
	}

	for _, cut := range []int{0, 1, headerSize, len(b) - 1} {
		if _, err := c.Decrypt(b[:cut]); err == nil {
			t.Errorf("a ciphertext cut to %d bytes opened", cut)
		}
	}
	if _, err := c.Decrypt(append(bytes.Clone(b), 'x')); err == nil {
		t.Error("a ciphertext with a byte on the end opened")
	}
}

func TestCryptStrings(t *testing.T) {
	c := newTest(t)

	text := c.EncryptString("hello", AD("greeting"))
	if strings.Contains(text, "hello") {
		t.Fatalf("the plaintext is in %q", text)
	}
	if strings.ContainsAny(text, "+/=") {
		t.Errorf("%q is not URL safe", text)
	}

	got, err := c.DecryptString(text, AD("greeting"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("it came back as %q", got)
	}

	if _, err := c.DecryptString("not base64!!", AD("greeting")); err == nil {
		t.Error("text that is not base64url decrypted")
	} else if errs.CodeOf(err) != "crypt.malformed" {
		t.Errorf("the error is %q, want crypt.malformed", errs.CodeOf(err))
	}
	if _, err := c.DecryptString(text); err == nil {
		t.Error("a bound value opened with nothing bound")
	}
}

// TestCryptKeyring is the whole reason a Crypt holds more than one key: a value
// written before a rotation still opens.
func TestCryptKeyring(t *testing.T) {
	old, next := GenerateKey(), GenerateKey()

	before, err := New(old)
	if err != nil {
		t.Fatal(err)
	}
	b := before.Encrypt([]byte("written under the old key"))

	after, err := New(next, old)
	if err != nil {
		t.Fatal(err)
	}
	got, err := after.Decrypt(b)
	if err != nil {
		t.Fatalf("a value written under a previous key: %v", err)
	}
	if string(got) != "written under the old key" {
		t.Errorf("it came back as %q", got)
	}

	// And a Crypt that never had the key says so, rather than saying the value
	// was tampered with.
	stranger := newTest(t)
	err = mustFail(t, stranger, b)
	if errs.CodeOf(err) != "crypt.unknown_key" {
		t.Errorf("the error is %q, want crypt.unknown_key", errs.CodeOf(err))
	}
	if !strings.Contains(err.Error(), old.ID()) {
		t.Errorf("the error does not name the key: %v", err)
	}
}

func TestCryptRotate(t *testing.T) {
	old := GenerateKey()
	before, err := New(old)
	if err != nil {
		t.Fatal(err)
	}
	b := before.Encrypt([]byte("older than the rotation"))

	next := GenerateKey()
	after, err := before.Rotate(next)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := after.Decrypt(b); err != nil {
		t.Errorf("the value from before the rotation: %v", err)
	}
	if after.ActiveID() != next.ID() {
		t.Errorf("the active key is %s, want %s", after.ActiveID(), next.ID())
	}
	if id := [8]byte(after.Encrypt([]byte("x"))[6:14]); id != next.id() {
		t.Error("something written after the rotation used the old key")
	}

	// Rotating twice keeps everything, and rotating to a key that is already
	// in the ring is a mistake worth naming.
	third, err := after.Rotate(GenerateKey())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := third.Decrypt(b); err != nil {
		t.Errorf("two rotations later: %v", err)
	}
	if _, err := third.Rotate(old); err == nil {
		t.Error("a key already in the ring was rotated in again")
	} else if errs.CodeOf(err) != "crypt.duplicate_key" {
		t.Errorf("the error is %q, want crypt.duplicate_key", errs.CodeOf(err))
	}
}

func TestCryptNeedsRewrap(t *testing.T) {
	old := GenerateKey()
	before, err := New(old)
	if err != nil {
		t.Fatal(err)
	}

	b := before.Encrypt([]byte("older than the rotation"))
	if before.NeedsRewrap(b) {
		t.Error("a value written with the active key wants rewrapping")
	}

	after, err := before.Rotate(GenerateKey())
	if err != nil {
		t.Fatal(err)
	}
	if !after.NeedsRewrap(b) {
		t.Error("a value written before the rotation does not want rewrapping")
	}

	// The rewrap itself, which is what a background job does.
	plain, err := after.Decrypt(b)
	if err != nil {
		t.Fatal(err)
	}
	if again := after.Encrypt(plain); after.NeedsRewrap(again) {
		t.Error("a rewrapped value still wants rewrapping")
	}

	// Nothing that is not a ciphertext is going to be fixed by rewrapping it.
	for _, b := range [][]byte{nil, []byte("hello"), bytes.Repeat([]byte{'x'}, 100)} {
		if after.NeedsRewrap(b) {
			t.Errorf("%q wants rewrapping", b)
		}
	}
}

func TestCryptMalformed(t *testing.T) {
	c := newTest(t)
	good := c.Encrypt([]byte("x"))

	short := bytes.Clone(good)
	wrongMagic := bytes.Clone(good)
	wrongMagic[0] = 'M'
	wrongVersion := bytes.Clone(good)
	wrongVersion[4] = 2
	wrongAlg := bytes.Clone(good)
	wrongAlg[5] = 9

	cases := map[string][]byte{
		"nothing":          nil,
		"short":            short[:Overhead-1],
		"not a ciphertext": wrongMagic,
		"another version":  wrongVersion,
		"another alg":      wrongAlg,
	}
	for name, b := range cases {
		err := mustFail(t, c, b)
		if errs.KindOf(err) != errs.Invalid {
			t.Errorf("%s: the kind is %v, want invalid", name, errs.KindOf(err))
		}
		if got := errs.CodeOf(err); got != "crypt.malformed" {
			t.Errorf("%s: the code is %q, want crypt.malformed", name, got)
		}
	}
}

func TestNewFailures(t *testing.T) {
	var zero Key
	key := GenerateKey()

	cases := map[string]func() (*Crypt, error){
		"no active key":   func() (*Crypt, error) { return New(zero) },
		"a zero previous": func() (*Crypt, error) { return New(key, zero) },
		"the same key twice": func() (*Crypt, error) {
			return New(key, GenerateKey(), key)
		},
	}
	for name, build := range cases {
		c, err := build()
		if err == nil {
			t.Errorf("%s was accepted", name)
			continue
		}
		if errs.KindOf(err) != errs.Invalid {
			t.Errorf("%s: the kind is %v, want invalid", name, errs.KindOf(err))
		}
		if c != nil {
			t.Errorf("%s: a Crypt came back with the error", name)
		}
	}
}

// TestCryptConcurrent is what the documentation promises. A Crypt holds nothing
// that changes, and this is the test that says so under the race detector.
func TestCryptConcurrent(t *testing.T) {
	c := newTest(t)

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 50 {
				plaintext := []byte(strings.Repeat("x", i*j%64))

				got, err := c.Decrypt(c.Encrypt(plaintext, AD("user:42")), AD("user:42"))
				if err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(got, plaintext) {
					t.Errorf("a %d byte message came back as %d", len(plaintext), len(got))
					return
				}
			}
		}()
	}
	wg.Wait()
}

// mustFail is a decryption that has to fail, and the error it failed with.
func mustFail(t *testing.T, c *Crypt, b []byte, ad ...AD) error {
	t.Helper()

	got, err := c.Decrypt(b, ad...)
	if err == nil {
		t.Fatalf("%q opened as %q", b, got)
	}
	if got != nil {
		t.Errorf("a plaintext came back with the error: %q", got)
	}
	return err
}
