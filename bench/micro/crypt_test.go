package micro

import (
	"bytes"
	"testing"

	"github.com/go-mizu/mizu/crypt"
)

func init() {
	register("crypt/seal/1kb", benchCryptSeal)
	register("crypt/open/1kb", benchCryptOpen)
	register("sign/hmac", benchSignHMAC)
}

// benchKey is fixed rather than generated so that two runs seal with the same
// key. It is written here in the open because it is a benchmark fixture and
// nothing outside this file has ever encrypted anything with it. A real key
// comes from the environment and never appears in a repository.
const benchKey = "mizu1:6aP7QlJC-lcgIXy1SBMCT3FmdkhWhhTzKROmHhewgXc"

// benchAD is the associated data a real caller passes: what the ciphertext is
// for, so that a session cookie cannot be replayed as a signed URL. Checking it
// is part of the open, which is why the open budget is above the seal budget.
var benchAD = crypt.AD("bench:crypt")

// plaintext1KB is built rather than read from testdata, because an AEAD costs
// the same for every 1 KB of input and a file would pin a length that a
// bytes.Repeat states more clearly.
var plaintext1KB = bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog\n"), 1024/44+1)[:1024]

func benchCryptSeal(b *testing.B) {
	c, err := crypt.New(crypt.MustParseKey(benchKey))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(plaintext1KB)))
	for b.Loop() {
		out := c.Encrypt(plaintext1KB, benchAD)
		_ = out
	}
}

func benchCryptOpen(b *testing.B) {
	c, err := crypt.New(crypt.MustParseKey(benchKey))
	if err != nil {
		b.Fatal(err)
	}
	sealed := c.Encrypt(plaintext1KB, benchAD)

	b.ReportAllocs()
	b.SetBytes(int64(len(plaintext1KB)))
	for b.Loop() {
		out, err := c.Decrypt(sealed, benchAD)
		if err != nil {
			b.Fatal(err)
		}
		_ = out
	}
}

// benchSignHMAC is the signed cookie and the signed URL, which are the two
// places a value goes out to a browser and comes back expected to be unchanged.
// The message is short because those values are short.
func benchSignHMAC(b *testing.B) {
	c, err := crypt.New(crypt.MustParseKey(benchKey))
	if err != nil {
		b.Fatal(err)
	}
	message := []byte("user=4211&expires=1774224000")

	b.ReportAllocs()
	for b.Loop() {
		out := c.Sign(message, benchAD)
		_ = out
	}
}
