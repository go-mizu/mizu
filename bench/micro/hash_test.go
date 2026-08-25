package micro

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/bench"
	"github.com/go-mizu/mizu/hash"
)

func init() {
	register("hash/argon2id/verify", benchArgon2idVerify)
	register("hash/bcrypt/verify", benchBcryptVerify)
}

// The password behind every hash in the corpus, and the hashes themselves. The
// parameters that decide what a verify costs are written into each hash, so
// they come from a versioned file rather than being chosen here.
const benchPassword = "correct horse battery staple"

var benchHashes = bench.Values("hashes.txt")

// benchArgon2idVerify is the most expensive single thing the framework does on
// a request, on exactly one route. Forty milliseconds and 19 MiB per sign in is
// deliberate, and this is here so that a change to the default parameters is
// seen rather than discovered under load.
//
// It runs one at a time, so it does not measure the concurrency limit that
// keeps a burst of logins from being an out of memory kill. That is a load test
// in bench/macro, not a number per operation.
func benchArgon2idVerify(b *testing.B) {
	h := hash.Default()
	encoded := benchHashes["argon2id"]
	ctx := context.Background()

	if ok, err := h.Verify(ctx, benchPassword, encoded); err != nil || !ok {
		b.Fatalf("the corpus hash does not verify: ok=%v err=%v", ok, err)
	}

	b.ReportAllocs()
	for b.Loop() {
		ok, err := h.Verify(ctx, benchPassword, encoded)
		if err != nil || !ok {
			b.Fatalf("verify: ok=%v err=%v", ok, err)
		}
	}
}

// benchBcryptVerify is the path an application arrives with rather than one it
// chooses, and it is slower than argon2id at a cost that buys less. It is
// measured because migration happens one sign in at a time, so the old path is
// live for as long as people keep coming back.
func benchBcryptVerify(b *testing.B) {
	var h hash.Bcrypt
	encoded := benchHashes["bcrypt"]
	ctx := context.Background()

	if ok, err := h.Verify(ctx, benchPassword, encoded); err != nil || !ok {
		b.Fatalf("the corpus hash does not verify: ok=%v err=%v", ok, err)
	}

	b.ReportAllocs()
	for b.Loop() {
		ok, err := h.Verify(ctx, benchPassword, encoded)
		if err != nil || !ok {
			b.Fatalf("verify: ok=%v err=%v", ok, err)
		}
	}
}
