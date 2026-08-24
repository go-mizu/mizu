package hash

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkHash is the number that decides the parameters, and the one
// hash:tune reads. What a password costs to check here is what it costs an
// attacker to guess, so this is not a benchmark to make smaller.
//
// The default is the OWASP recommendation. Everything below it is what a
// machine that can afford more looks like.
func BenchmarkHash(b *testing.B) {
	cases := []struct {
		name          string
		memory, lanes int
	}{
		{"owasp", 19456, 1},
		{"owasp/4 lanes", 19456, 4},
		{"64MiB", 65536, 1},
		{"cheap", 64, 1},
	}

	for _, c := range cases {
		h, err := New(Params{Memory: c.memory, Passes: 2, Lanes: c.lanes})
		if err != nil {
			b.Fatalf("New: %v", err)
		}

		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				sinkString, sinkErr = h.Hash(b.Context(), "correct horse battery staple")
			}
		})
	}
}

// BenchmarkVerify is the one that runs on every sign in, and it should cost the
// same as hashing, since it does the same work.
func BenchmarkVerify(b *testing.B) {
	h := Default()

	encoded, err := h.Hash(b.Context(), "correct horse battery staple")
	if err != nil {
		b.Fatalf("Hash: %v", err)
	}

	b.ReportAllocs()
	for b.Loop() {
		sinkBool, sinkErr = h.Verify(b.Context(), "correct horse battery staple", encoded)
	}
}

// BenchmarkParse is what a verify pays before it starts, and what NeedsRehash
// pays on its own. It runs once per sign in and it should not show up at all
// next to the hash.
func BenchmarkParse(b *testing.B) {
	const encoded = "$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8"

	b.ReportAllocs()
	for b.Loop() {
		sinkPHC, sinkErr = parse(encoded)
	}
}

func BenchmarkEncode(b *testing.B) {
	p := phc{memory: 19456, passes: 2, lanes: 1, salt: counting(16), tag: counting(32)}

	b.ReportAllocs()
	for b.Loop() {
		sinkString = p.encode()
	}
}

// BenchmarkGate is the cost of the limit on a hasher that is not busy, which is
// what it is almost always doing. It is a channel send and a receive, and it
// has to stay far enough below a hash to not be worth thinking about.
func BenchmarkGate(b *testing.B) {
	for _, n := range []int{1, 8, 64} {
		g := newGate(n, time.Second)

		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				release, err := g.enter(b.Context())
				if err != nil {
					b.Fatal(err)
				}
				release()
			}
		})
	}
}

// BenchmarkGateContended is the same gate with more goroutines than slots,
// which is what a burst of logins looks like from the inside.
func BenchmarkGateContended(b *testing.B) {
	g := newGate(4, time.Minute)
	ctx := b.Context()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			release, err := g.enter(ctx)
			if err != nil {
				b.Error(err)
				return
			}
			release()
		}
	})
}

var (
	sinkString string
	sinkBool   bool
	sinkErr    error
	sinkPHC    phc
)
