package hash

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// The vectors are from RFC 7693 appendix A and from the BLAKE2 reference
// implementation. A hash that is close but not the same would still be a hash,
// and Argon2 built on it would still produce something that verifies against
// itself, so this is the test that says the answer is BLAKE2b.
func TestBlake2b(t *testing.T) {
	cases := []struct {
		size  int
		in    []byte
		want  string
		about string
	}{
		{64, nil, "786a02f742015903c6c6fd852552d272912f4740e15847618a86e217f71f5419d25e1031afee585313896444934eb04b903a685b1448b755d56f701afe9be2ce", "nothing at all"},
		{64, []byte("abc"), "ba80a53f981c4d0d6a2797b69f12f6e94c212f14685ac4b74b12bb6fdbffa2d17d87c5392aab792dc252d5de4533cc9518d38aa8dbf1925ab92386edd4009923", "RFC 7693 appendix A"},
		{64, []byte("The quick brown fox jumps over the lazy dog"), "a8add4bdddfd93e4877d2746e62817b116364a1fa7bc148d95090bc7333b3673f82401cf7aa2e4cb1ecd90296e3f14cb5413f8ed77be73045b13914cdcd6a918", "one block and a bit"},
		{32, []byte("abc"), "bddd813c634239723171ef3fee98579b94964e3bb1cb3e427262c8c068d52319", "a shorter digest is a different hash, not a truncation"},
		{64, counting(300), "d9cf5983dc6b34c0fa1f0226926855ad3eccd2bcdcd8f8053b9a80664d33b5afcc32fd21c70ea14f4ef50ca97c3203c4d1803159f0e01bb6cb1d1c83db52b63c", "three blocks"},
	}

	for _, c := range cases {
		if got := hex.EncodeToString(blake2bSum(c.size, c.in)); got != c.want {
			t.Errorf("%s: %s, want %s", c.about, got, c.want)
		}
	}
}

// TestBlake2bWrites is the part a single call cannot reach: a block is held
// back until something follows it, because the last one is compressed with a
// flag the others do not have.
func TestBlake2bWrites(t *testing.T) {
	in := counting(600)
	want := blake2bSum(maxDigest, in)

	// Every split of the input, one byte at a time, and the ends of a block.
	for _, at := range []int{0, 1, 127, 128, 129, 255, 256, 257, 511, 512, 599, 600} {
		d := newBlake2b(maxDigest)
		d.write(in[:at])
		d.write(in[at:])

		if got := d.sum(nil); !bytes.Equal(got, want) {
			t.Errorf("split at %d gives %x, want %x", at, got, want)
		}
	}

	// And one byte at a time the whole way, which is the path where the buffer
	// fills and empties on every call.
	d := newBlake2b(maxDigest)
	for i := range in {
		d.write(in[i : i+1])
	}
	if got := d.sum(nil); !bytes.Equal(got, want) {
		t.Errorf("a byte at a time gives %x, want %x", got, want)
	}
}

// TestBlake2bSumAppends is what the argument is for. The digest goes on the end
// of what is already there.
func TestBlake2bSumAppends(t *testing.T) {
	d := newBlake2b(32)
	d.write([]byte("abc"))

	got := d.sum([]byte("prefix"))
	if string(got[:6]) != "prefix" {
		t.Errorf("sum overwrote what it was given: %q", got)
	}
	if len(got) != 6+32 {
		t.Errorf("sum appended %d bytes, want 32", len(got)-6)
	}
	if hex.EncodeToString(got[6:]) != "bddd813c634239723171ef3fee98579b94964e3bb1cb3e427262c8c068d52319" {
		t.Errorf("the digest is %x", got[6:])
	}
}

// TestBlake2bLong covers H' at the sizes Argon2 asks for: the 1024 byte blocks
// it fills memory with, the 64 bytes and under it uses for the tag, and the
// boundary in between where the chaining starts.
func TestBlake2bLong(t *testing.T) {
	in := []byte("abc")

	// Up to 64 bytes it is the plain hash of the length and the input, which is
	// stated here rather than derived from the code under test.
	for _, size := range []int{1, 32, 64} {
		want := blake2bSum(size, []byte{byte(size), 0, 0, 0}, in)
		if got := blake2bLong(size, in); !bytes.Equal(got, want) {
			t.Errorf("%d bytes: %x, want %x", size, got, want)
		}
	}

	// Past that it chains, and the first 32 bytes are the first half of the
	// first digest whatever the total length is.
	first := blake2bSum(maxDigest, []byte{65, 0, 0, 0}, in)
	if got := blake2bLong(65, in); !bytes.Equal(got[:32], first[:32]) {
		t.Errorf("the first link is %x, want %x", got[:32], first[:32])
	}

	for _, size := range []int{65, 96, 128, argonBlockSize} {
		if got := len(blake2bLong(size, in)); got != size {
			t.Errorf("asked for %d bytes and got %d", size, got)
		}
	}

	// Two lengths never give the same bytes, since the length is hashed in.
	a, b := blake2bLong(96, in), blake2bLong(128, in)
	if bytes.Equal(a[:96], b[:96]) {
		t.Error("two lengths agree on their common prefix")
	}
}

// counting is n bytes that are not all the same, so that a wrong word order
// shows up.
func counting(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}
