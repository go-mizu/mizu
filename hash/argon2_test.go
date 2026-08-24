package hash

import (
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/argon2"
)

// The argon2id itself comes from golang.org/x/crypto, so what is left to test
// here is that this package hands it the right things: the parameters out of
// the encoded hash rather than the ones configured, the salt as bytes rather
// than as the base64 it was stored in, and the tag at the length the stored
// hash says.
//
// Getting any of those wrong produces a hash that verifies against itself and
// against nothing else in the world, which is a bug that a round trip test
// cannot see. So the vectors are stated here as bytes.

// TestVectors are answers from the argon2 reference implementation, by way of
// golang.org/x/crypto. They go through Verify rather than through the hash
// function, so what they check is the whole path from an encoded string to a
// yes or a no.
func TestVectors(t *testing.T) {
	h := cheap(t)

	cases := []struct {
		password string
		encoded  string
	}{
		// One lane, one pass, the smallest memory anything uses.
		{"password", "$argon2id$v=19$m=64,t=1,p=1$c29tZXNhbHQxMjM0NTY3OA$KovyxkqWI8NiIbnVTsHJGtWda07axUqoJVIuPotqICc"},

		// Four lanes, which is the parameter most likely to be dropped on the
		// way through, since it changes the answer and nothing else.
		{"password", "$argon2id$v=19$m=32,t=3,p=4$c29tZXNhbHQxMjM0NTY3OA$zTQtuG1WWMjjJnuhwywU9mvZXEnKLIQtv/DimNN4F8k"},

		// A 16 byte tag, to say the length comes from the stored hash and not
		// from what this package writes.
		{"password", "$argon2id$v=19$m=8,t=1,p=1$c29tZXNhbHQ$9zpz0nGFnSHvRUZdoSvfhg"},

		// A 64 byte tag, on the other side of a single BLAKE2b digest.
		{"password", "$argon2id$v=19$m=1024,t=4,p=2$c29tZXNhbHQxMjM0NTY3OA$sZgRawlIzzdZzyE+rnDLEZMKJwDMzU0QZVppr2OlwR5+/O6g8RYF0SQ4WfWn/FEPAdm7nXl2lq9AZ/422uekPw"},
	}

	for _, c := range cases {
		ok, err := h.Verify(t.Context(), c.password, c.encoded)
		if err != nil {
			t.Errorf("%s: %v", c.encoded, err)
			continue
		}
		if !ok {
			t.Errorf("%s: does not verify against %q", c.encoded, c.password)
		}
		if ok, _ := h.Verify(t.Context(), c.password+"x", c.encoded); ok {
			t.Errorf("%s: verifies against the wrong password too", c.encoded)
		}
	}
}

// TestHashMatchesTheLibrary is the other direction. What Hash writes is what
// argon2.IDKey returns for the salt in the encoded string, so a hash from here
// is a hash anything else that speaks the format agrees with.
func TestHashMatchesTheLibrary(t *testing.T) {
	h, err := New(Params{Memory: 128, Passes: 3, Lanes: 2})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	encoded, err := h.Hash(t.Context(), "correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	p, err := parse(encoded)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	want := argon2.IDKey([]byte("correct horse battery staple"), p.salt, 3, 128, 2, keySize)
	if hex.EncodeToString(p.tag) != hex.EncodeToString(want) {
		t.Errorf("the tag is %x, want %x", p.tag, want)
	}
}

// counting is n bytes that are not all the same, so that a wrong byte order
// shows up.
func counting(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}
