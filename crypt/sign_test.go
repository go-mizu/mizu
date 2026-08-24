package crypt

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

func TestSign(t *testing.T) {
	c := newTest(t)
	message := []byte("user:42")

	b := c.Sign(message)
	if len(b) != len(message)+SignOverhead {
		t.Errorf("a %d byte message came out %d bytes, want %d", len(message), len(b), len(message)+SignOverhead)
	}

	got, err := c.Verify(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, message) {
		t.Errorf("it came back as %q", got)
	}
}

// TestSignIsReadable is the difference from encryption, and it is the reason to
// reach for one rather than the other.
func TestSignIsReadable(t *testing.T) {
	c := newTest(t)
	message := []byte("user:42")

	if !bytes.Contains(c.Sign(message), message) {
		t.Error("a signed message is not in what came out")
	}
	if bytes.Contains(c.Encrypt(message), message) {
		t.Error("an encrypted message is in what came out")
	}
}

// TestSignIsDeterministic is the other difference. There is no nonce, so the
// same message signs the same way every time, which is what lets a signed value
// be compared or used as a key.
func TestSignIsDeterministic(t *testing.T) {
	c := newTest(t)

	first, second := c.Sign([]byte("user:42")), c.Sign([]byte("user:42"))
	if !bytes.Equal(first, second) {
		t.Error("the same message signed two different ways")
	}
	if bytes.Equal(first, c.Sign([]byte("user:43"))) {
		t.Error("two messages signed the same way")
	}
}

func TestSignSizes(t *testing.T) {
	c := newTest(t)

	for _, n := range []int{0, 1, 31, 32, 33, 1000} {
		message := bytes.Repeat([]byte{'a'}, n)

		got, err := c.Verify(c.Sign(message))
		if err != nil {
			t.Fatalf("%d bytes: %v", n, err)
		}
		if !bytes.Equal(got, message) {
			t.Errorf("%d bytes came back as %d", n, len(got))
		}
	}

	if _, err := c.Verify(c.Sign(nil)); err != nil {
		t.Errorf("a nil message: %v", err)
	}
}

// TestSignHeader pins the format down, the same way the ciphertext header is
// pinned. A signed value ends up in a cookie somebody still holds.
func TestSignHeader(t *testing.T) {
	key := GenerateKey()
	c, err := New(key)
	if err != nil {
		t.Fatal(err)
	}

	b := c.Sign([]byte("hello"))
	if got := string(b[:4]); got != "mizu" {
		t.Errorf("the magic is %q, want mizu", got)
	}
	if b[4] != 1 {
		t.Errorf("the version is %d, want 1", b[4])
	}
	if b[5] != 2 {
		t.Errorf("the algorithm is %d, want 2", b[5])
	}
	if got, want := [8]byte(b[6:14]), key.id(); got != want {
		t.Errorf("the key id is %x, want %x", got, want)
	}
	if got := string(b[14:19]); got != "hello" {
		t.Errorf("the message is %q, want hello", got)
	}
	if len(b[19:]) != 32 {
		t.Errorf("the tag is %d bytes, want 32", len(b[19:]))
	}
	if prefixSize != 14 || SignOverhead != 46 {
		t.Errorf("the prefix is %d bytes and the overhead %d, want 14 and 46", prefixSize, SignOverhead)
	}
}

// TestSignKey is the subkey HMAC uses, pinned for a key of all ones. Changing
// how it is derived would make every signed value anybody holds stop verifying,
// so it is written down here rather than only implied by the code.
//
// HKDF-Expand for 32 bytes of SHA-256 is one HMAC, keyed with the key, over the
// label and a counter byte, so the value below is:
//
//	python3 -c 'import hmac,hashlib;print(hmac.new(bytes([1])*32, b"mizu1 sign"+bytes([1]), hashlib.sha256).hexdigest())'
func TestSignKey(t *testing.T) {
	const want = "4f74a91c9d7fc32d5514712417338e82c617cc6d788b4b8344454a21f0e3279f"

	k := newKeyed(keyOf(t, bytes.Repeat([]byte{1}, KeySize)))
	if got := hex.EncodeToString(k.signKey[:]); got != want {
		t.Errorf("the signing key is %s, want %s", got, want)
	}

	// And it is not the key it came from, which is the point of deriving it.
	if bytes.Equal(k.signKey[:], k.key.b[:]) {
		t.Error("the signing key is the key")
	}
}

// TestSignHeaderIsAuthenticated is why the prefix goes into the tag. A value
// that lies about which key signed it has to stop verifying.
func TestSignHeaderIsAuthenticated(t *testing.T) {
	c := newTest(t)
	b := c.Sign([]byte("user:42"))

	for _, i := range []int{0, 4, 5, 6, 13} {
		changed := bytes.Clone(b)
		changed[i]++

		if _, err := c.Verify(changed); err == nil {
			t.Errorf("byte %d of the header changed and it still verified", i)
		}
	}
}

func TestSignAD(t *testing.T) {
	c := newTest(t)
	message := []byte("user:42")

	b := c.Sign(message, AD("session"))
	if _, err := c.Verify(b, AD("session")); err != nil {
		t.Fatalf("the value it was bound to: %v", err)
	}

	if _, err := c.Verify(b, AD("reset")); err == nil {
		t.Error("a value bound to one purpose verified under another")
	} else if errs.CodeOf(err) != "crypt.tampered" {
		t.Errorf("the error is %q, want crypt.tampered", errs.CodeOf(err))
	}
	if _, err := c.Verify(b); err == nil {
		t.Error("a bound value verified with nothing bound")
	}
	if _, err := c.Verify(c.Sign(message), AD("session")); err == nil {
		t.Error("an unbound value verified with something bound")
	}
}

// TestSignADIsUnambiguous is the same rule the ciphertext has: each piece goes
// in after its own length, so two of them cannot be rearranged into two others.
func TestSignADIsUnambiguous(t *testing.T) {
	c := newTest(t)

	b := c.Sign([]byte("x"), AD("user:4"), AD("2"))
	if _, err := c.Verify(b, AD("user:4"), AD("2")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Verify(b, AD("user:"), AD("42")); err == nil {
		t.Error("two bindings rearranged and the value still verified")
	}
	if _, err := c.Verify(b, AD("user:42")); err == nil {
		t.Error("two bindings joined into one and the value still verified")
	}

	empty := c.Sign([]byte("x"), AD(""))
	if _, err := c.Verify(empty); err == nil {
		t.Error("a value bound to nothing verified with no binding")
	}
	if _, err := c.Verify(empty, AD("")); err != nil {
		t.Errorf("a value bound to nothing: %v", err)
	}
}

// TestSignTampered walks every byte, message and tag alike. Changing any of
// them has to be caught, which is the whole promise of signing.
func TestSignTampered(t *testing.T) {
	c := newTest(t)
	b := c.Sign([]byte("user:42"))

	for i := range b {
		changed := bytes.Clone(b)
		changed[i] ^= 0x40

		if _, err := c.Verify(changed); err == nil {
			t.Fatalf("byte %d changed and the value still verified", i)
		}
	}

	for _, cut := range []int{0, 1, prefixSize, len(b) - 1} {
		if _, err := c.Verify(b[:cut]); err == nil {
			t.Errorf("a signed value cut to %d bytes verified", cut)
		}
	}
	if _, err := c.Verify(append(bytes.Clone(b), 'x')); err == nil {
		t.Error("a signed value with a byte on the end verified")
	}
}

func TestSignStrings(t *testing.T) {
	c := newTest(t)

	text := c.SignString("user:42", AD("session"))
	if strings.ContainsAny(text, "+/=") {
		t.Errorf("%q is not URL safe", text)
	}

	got, err := c.VerifyString(text, AD("session"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "user:42" {
		t.Errorf("it came back as %q", got)
	}

	if _, err := c.VerifyString("not base64!!", AD("session")); err == nil {
		t.Error("text that is not base64url verified")
	} else if errs.CodeOf(err) != "crypt.malformed" {
		t.Errorf("the error is %q, want crypt.malformed", errs.CodeOf(err))
	}
	if _, err := c.VerifyString(text); err == nil {
		t.Error("a bound value verified with nothing bound")
	}
}

// TestSignKeyring is a signed value outliving the key that wrote it, which is
// what a cookie somebody has not used for a month looks like after a rotation.
func TestSignKeyring(t *testing.T) {
	old := GenerateKey()
	before, err := New(old)
	if err != nil {
		t.Fatal(err)
	}
	b := before.Sign([]byte("signed under the old key"))

	after, err := before.Rotate(GenerateKey())
	if err != nil {
		t.Fatal(err)
	}
	got, err := after.Verify(b)
	if err != nil {
		t.Fatalf("a value signed under a previous key: %v", err)
	}
	if string(got) != "signed under the old key" {
		t.Errorf("it came back as %q", got)
	}
	if !after.NeedsRewrap(b) {
		t.Error("a signed value from before the rotation does not want rewrapping")
	}
	if after.NeedsRewrap(after.Sign(got)) {
		t.Error("a value signed with the active key wants rewrapping")
	}

	stranger := newTest(t)
	if _, err := stranger.Verify(b); err == nil {
		t.Error("a value verified against a keyring that never had the key")
	} else if errs.CodeOf(err) != "crypt.unknown_key" {
		t.Errorf("the error is %q, want crypt.unknown_key", errs.CodeOf(err))
	}
}

// TestSignAndEncryptDoNotMix is what the algorithm byte is there for. Neither
// one reads what the other wrote, and neither one reports it as tampering.
func TestSignAndEncryptDoNotMix(t *testing.T) {
	c := newTest(t)

	if _, err := c.Verify(c.Encrypt([]byte("x"))); err == nil {
		t.Error("a ciphertext verified")
	} else if errs.CodeOf(err) != "crypt.malformed" {
		t.Errorf("the error is %q, want crypt.malformed", errs.CodeOf(err))
	}

	// Long enough to be a ciphertext, so that the length is not what catches it.
	if _, err := c.Decrypt(c.Sign(bytes.Repeat([]byte{'x'}, 100))); err == nil {
		t.Error("a signed value decrypted")
	} else if errs.CodeOf(err) != "crypt.malformed" {
		t.Errorf("the error is %q, want crypt.malformed", errs.CodeOf(err))
	}
}

func TestVerifyMalformed(t *testing.T) {
	c := newTest(t)
	good := c.Sign([]byte("x"))

	wrongMagic := bytes.Clone(good)
	wrongMagic[0] = 'M'
	wrongVersion := bytes.Clone(good)
	wrongVersion[4] = 2
	wrongAlg := bytes.Clone(good)
	wrongAlg[5] = 9

	cases := map[string][]byte{
		"nothing":            nil,
		"short":              good[:SignOverhead-1],
		"not a signed value": wrongMagic,
		"another version":    wrongVersion,
		"another alg":        wrongAlg,
	}
	for name, b := range cases {
		got, err := c.Verify(b)
		if err == nil {
			t.Errorf("%s verified", name)
			continue
		}
		if got != nil {
			t.Errorf("%s: a message came back with the error", name)
		}
		if errs.KindOf(err) != errs.Invalid {
			t.Errorf("%s: the kind is %v, want invalid", name, errs.KindOf(err))
		}
		if code := errs.CodeOf(err); code != "crypt.malformed" {
			t.Errorf("%s: the code is %q, want crypt.malformed", name, code)
		}
	}
}

// session is what a sealed value looks like in practice.
type session struct {
	User  string `json:"user"`
	Admin bool   `json:"admin"`
}

func TestSeal(t *testing.T) {
	c := newTest(t)
	want := session{User: "42", Admin: true}

	token, err := Seal(c, want, AD("session"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(token, "42") || strings.ContainsAny(token, "+/=") {
		t.Errorf("the token is %q", token)
	}

	got, err := Unseal[session](c, token, AD("session"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("it came back as %+v, want %+v", got, want)
	}

	if _, err := Unseal[session](c, token); err == nil {
		t.Error("a bound token opened with nothing bound")
	}
}

func TestSealFailures(t *testing.T) {
	c := newTest(t)

	// A channel is the value encoding/json refuses, and the refusal is the
	// caller's mistake rather than anything about the key.
	if _, err := Seal(c, make(chan int)); err == nil {
		t.Error("a channel was sealed")
	} else if errs.CodeOf(err) != "crypt.unencodable" {
		t.Errorf("the error is %q, want crypt.unencodable", errs.CodeOf(err))
	}

	cases := map[string]string{
		"not base64":  "not base64!!",
		"not a value": c.EncryptString("not json"),
	}
	for name, token := range cases {
		got, err := Unseal[session](c, token)
		if err == nil {
			t.Errorf("%s was unsealed", name)
			continue
		}
		if got != (session{}) {
			t.Errorf("%s: a value came back with the error: %+v", name, got)
		}
		if code := errs.CodeOf(err); code != "crypt.malformed" {
			t.Errorf("%s: the code is %q, want crypt.malformed", name, code)
		}
	}

	// A token nobody in this keyring wrote reports what is actually wrong.
	stranger, err := Seal(newTest(t), session{User: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Unseal[session](c, stranger); errs.CodeOf(err) != "crypt.unknown_key" {
		t.Errorf("the error is %q, want crypt.unknown_key", errs.CodeOf(err))
	}
}
