package crypt

import (
	"bytes"
	"crypto/sha3"
	"encoding/hex"
	"testing"
)

// The vectors come from the specification, https://c2sp.org/XAES-256-GCM. They
// are the reason this file exists: a derivation that is close but not the same
// still encrypts and still decrypts, and would be a private algorithm nobody
// else can read.
var vectors = []struct {
	name       string
	key        []byte
	nonce      string
	l          string
	k1         string
	kx         string
	nx         string
	plaintext  string
	ad         string
	ciphertext string
}{
	{
		name:       "the top bit of L clear",
		key:        bytes.Repeat([]byte{0x01}, KeySize),
		nonce:      "ABCDEFGHIJKLMNOPQRSTUVWX",
		l:          "7298caa565031eadc6ce23d23ea66378",
		k1:         "e531954aca063d5b8d9c47a47d4cc6f0",
		kx:         "c8612c9ed53fe43e8e005b828a1631a0bbcb6ab2f46514ec4f439fcfd0fa969b",
		nx:         "4d4e4f505152535455565758",
		plaintext:  "XAES-256-GCM",
		ad:         "",
		ciphertext: "ce546ef63c9cc60765923609b33a9a1974e96e52daf2fcf7075e2271",
	},
	{
		// The other branch of the doubling, where the polynomial goes back in.
		name:       "the top bit of L set",
		key:        bytes.Repeat([]byte{0x03}, KeySize),
		nonce:      "ABCDEFGHIJKLMNOPQRSTUVWX",
		l:          "91c08762876dccf9ba204a33768fa5fe",
		k1:         "23810ec50edb99f374409466ed1f4b7b",
		kx:         "e9c621d4cdd9b11b00a6427ad7e559aeedd66b3857646677748f8ca796cb3fd8",
		nx:         "4d4e4f505152535455565758",
		plaintext:  "XAES-256-GCM",
		ad:         "c2sp.org/XAES-256-GCM",
		ciphertext: "986ec1832593df5443a179437fd083bf3fdb41abd740a21f71eb769d",
	},
}

func TestXAESVectors(t *testing.T) {
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			key := keyOf(t, v.key)
			k := newKeyed(key)

			// L is not kept, since only K1 is used again, so it is derived here
			// the same way the code does and checked on the way past.
			var l [16]byte
			k.block.Encrypt(l[:], l[:])
			if got := hex.EncodeToString(l[:]); got != v.l {
				t.Errorf("L is %s, want %s", got, v.l)
			}
			if got := hex.EncodeToString(k.k1[:]); got != v.k1 {
				t.Errorf("K1 is %s, want %s", got, v.k1)
			}

			gcm, nx := k.aead([]byte(v.nonce))
			if got := hex.EncodeToString(nx); got != v.nx {
				t.Errorf("Nx is %s, want %s", got, v.nx)
			}

			// The derived key is not handed back, so it is checked by what it
			// produces: the ciphertext of the vector, tag and all.
			got := gcm.Seal(nil, nx, []byte(v.plaintext), []byte(v.ad))
			if hex.EncodeToString(got) != v.ciphertext {
				t.Errorf("the ciphertext is %x, want %s", got, v.ciphertext)
			}

			back, err := gcm.Open(nil, nx, got, []byte(v.ad))
			if err != nil {
				t.Fatalf("the vector does not open: %v", err)
			}
			if string(back) != v.plaintext {
				t.Errorf("the vector opened as %q", back)
			}
		})
	}
}

// TestXAESDerivedKey is the one intermediate the code does not hand back, taken
// apart here so that a wrong Kx is not covered up by the encryption being
// self consistent.
func TestXAESDerivedKey(t *testing.T) {
	for _, v := range vectors {
		k := newKeyed(keyOf(t, v.key))
		nonce := []byte(v.nonce)

		// The derivation, spelled out from the specification rather than
		// reusing the code under test.
		var m1, m2 [16]byte
		m1[1], m1[2] = 0x01, 0x58
		m2[1], m2[2] = 0x02, 0x58
		copy(m1[4:], nonce[:12])
		copy(m2[4:], nonce[:12])
		for i := range m1 {
			m1[i] ^= k.k1[i]
			m2[i] ^= k.k1[i]
		}

		kx := make([]byte, 32)
		k.block.Encrypt(kx[:16], m1[:])
		k.block.Encrypt(kx[16:], m2[:])
		if got := hex.EncodeToString(kx); got != v.kx {
			t.Errorf("%s: Kx is %s, want %s", v.name, got, v.kx)
		}
	}
}

// TestXAESAccumulated is the specification's randomized test. Keys, nonces,
// plaintexts and additional data come out of one SHAKE-128 stream with no
// input, every ciphertext goes into another, and the digest of that stream is
// what the specification publishes.
//
// It catches the mistakes the two vectors above cannot: a length that only goes
// wrong for an empty plaintext, a nonce byte that only matters sometimes.
func TestXAESAccumulated(t *testing.T) {
	const (
		iterations = 10000
		want       = "e6b9edf2df6cec60c8cbd864e2211b597fb69a529160cd040d56c0c210081939"
	)
	if got := accumulate(t, iterations); got != want {
		t.Errorf("%d iterations hash to %s, want %s", iterations, got, want)
	}
}

// TestXAESAccumulatedLong is the same test at the other size the specification
// publishes. It takes a few seconds, which is what -short is for.
func TestXAESAccumulatedLong(t *testing.T) {
	const (
		iterations = 1000000
		want       = "2163ae1445985a30b60585ee67daa55674df06901b890593e824b8a7c885ab15"
	)
	if testing.Short() {
		t.Skip("a million messages")
	}
	if got := accumulate(t, iterations); got != want {
		t.Errorf("%d iterations hash to %s, want %s", iterations, got, want)
	}
}

func accumulate(t *testing.T, iterations int) string {
	t.Helper()

	rng, digest := sha3.NewSHAKE128(), sha3.NewSHAKE128()

	key := make([]byte, KeySize)
	nonce := make([]byte, nonceSize)
	length := make([]byte, 1)

	for range iterations {
		rng.Read(key)
		rng.Read(nonce)

		rng.Read(length)
		plaintext := make([]byte, length[0])
		rng.Read(plaintext)

		rng.Read(length)
		ad := make([]byte, length[0])
		rng.Read(ad)

		gcm, nx := newKeyed(keyOf(t, key)).aead(nonce)
		digest.Write(gcm.Seal(nil, nx, plaintext, ad))
	}

	sum := make([]byte, 32)
	digest.Read(sum)
	return hex.EncodeToString(sum)
}

// TestDouble covers the shift on its own, at the two edges the vectors reach
// and at the ends of a block.
func TestDouble(t *testing.T) {
	cases := map[string]string{
		"00000000000000000000000000000000": "00000000000000000000000000000000",
		"00000000000000000000000000000001": "00000000000000000000000000000002",
		"80000000000000000000000000000000": "00000000000000000000000000000087",
		"ffffffffffffffffffffffffffffffff": "ffffffffffffffffffffffffffffff79",
		"7298caa565031eadc6ce23d23ea66378": "e531954aca063d5b8d9c47a47d4cc6f0",
		"91c08762876dccf9ba204a33768fa5fe": "23810ec50edb99f374409466ed1f4b7b",
	}
	for in, want := range cases {
		var l [16]byte
		b, err := hex.DecodeString(in)
		if err != nil {
			t.Fatal(err)
		}
		copy(l[:], b)

		if got := hex.EncodeToString(doubled(l)); got != want {
			t.Errorf("double(%s) is %s, want %s", in, got, want)
		}
	}
}

func doubled(l [16]byte) []byte {
	out := double(l)
	return out[:]
}

// keyOf is a Key from bytes a test has, which is the way in that the public
// interface does not offer.
func keyOf(t testing.TB, b []byte) Key {
	t.Helper()
	if len(b) != KeySize {
		t.Fatalf("a key is %d bytes, not %d", KeySize, len(b))
	}

	var k Key
	copy(k.b[:], b)
	return k
}
