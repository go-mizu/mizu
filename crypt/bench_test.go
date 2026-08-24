package crypt

import (
	"crypto/cipher"
	"fmt"
	"testing"
)

func BenchmarkGenerateKey(b *testing.B) {
	for b.Loop() {
		sinkKey = GenerateKey()
	}
}

func BenchmarkParseKey(b *testing.B) {
	text := GenerateKey().Reveal()
	b.ReportAllocs()

	for b.Loop() {
		sinkKey, _ = ParseKey(text)
	}
}

// BenchmarkKeyID matters because the id goes in the header of every ciphertext,
// so it happens once per encryption.
func BenchmarkKeyID(b *testing.B) {
	key := GenerateKey()
	b.ReportAllocs()

	for b.Loop() {
		sinkBytes8 = key.id()
	}
}

func BenchmarkKeyEqual(b *testing.B) {
	a, c := GenerateKey(), GenerateKey()
	b.ReportAllocs()

	for b.Loop() {
		sinkBool = a.Equal(c)
	}
}

func BenchmarkSecretEqual(b *testing.B) {
	s := Secret("t0ps3cr3t-not-a-real-token")
	b.ReportAllocs()

	for b.Loop() {
		sinkBool = s.Equal("t0ps3cr3t-not-a-real-token")
	}
}

func BenchmarkBytes(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkSlice = Bytes(32)
	}
}

func BenchmarkToken(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkString = Token(32)
	}
}

func BenchmarkString(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkString = String(20)
	}
}

func BenchmarkDigits(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkString = Digits(6)
	}
}

func BenchmarkIntn(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkInt = Intn(7)
	}
}

// sizes are a session cookie, a row of a table and something that arrived as a
// file. The first is where most of the time is the derivation rather than the
// message.
var sizes = []int{64, 1024, 65536}

func BenchmarkEncrypt(b *testing.B) {
	c := benchCrypt(b)

	for _, size := range sizes {
		plaintext := make([]byte, size)
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()

			for b.Loop() {
				sinkSlice = c.Encrypt(plaintext)
			}
		})
	}
}

func BenchmarkDecrypt(b *testing.B) {
	c := benchCrypt(b)

	for _, size := range sizes {
		ciphertext := c.Encrypt(make([]byte, size))
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()

			for b.Loop() {
				sinkSlice, sinkErr = c.Decrypt(ciphertext)
			}
		})
	}
}

// BenchmarkEncryptAD is the cost of binding, which is one more allocation and a
// copy of the header.
func BenchmarkEncryptAD(b *testing.B) {
	c := benchCrypt(b)
	plaintext := make([]byte, 64)
	b.SetBytes(64)
	b.ReportAllocs()

	for b.Loop() {
		sinkSlice = c.Encrypt(plaintext, AD("user:42"))
	}
}

func BenchmarkEncryptString(b *testing.B) {
	c := benchCrypt(b)
	plaintext := string(make([]byte, 64))
	b.SetBytes(64)
	b.ReportAllocs()

	for b.Loop() {
		sinkString = c.EncryptString(plaintext)
	}
}

func BenchmarkDecryptString(b *testing.B) {
	c := benchCrypt(b)
	ciphertext := c.EncryptString(string(make([]byte, 64)))
	b.SetBytes(64)
	b.ReportAllocs()

	for b.Loop() {
		sinkString, sinkErr = c.DecryptString(ciphertext)
	}
}

// BenchmarkAEAD is the XAES part on its own: three AES calls and a GCM setup,
// which happens once per message and is what a short message mostly costs.
func BenchmarkAEAD(b *testing.B) {
	k := newKeyed(GenerateKey())
	nonce := Bytes(nonceSize)
	b.ReportAllocs()

	for b.Loop() {
		sinkAEAD, sinkSlice = k.aead(nonce)
	}
}

// BenchmarkNeedsRewrap is a header read with no decryption, which is what a
// rewrap job does to every row it looks at.
func BenchmarkNeedsRewrap(b *testing.B) {
	c := benchCrypt(b)
	ciphertext := c.Encrypt(make([]byte, 64))
	b.ReportAllocs()

	for b.Loop() {
		sinkBool = c.NeedsRewrap(ciphertext)
	}
}

func benchCrypt(b *testing.B) *Crypt {
	b.Helper()

	c, err := New(GenerateKey())
	if err != nil {
		b.Fatal(err)
	}
	return c
}

var (
	sinkKey    Key
	sinkBytes8 [8]byte
	sinkBool   bool
	sinkSlice  []byte
	sinkString string
	sinkInt    int
	sinkErr    error
	sinkAEAD   cipher.AEAD
)
