package hash

import (
	"context"
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

// BenchmarkVerifyBcrypt is what a sign in costs on the day an application
// starts reading somebody else's password column, at the cost Laravel writes.
// It is the number to hold next to BenchmarkVerify when deciding whether a
// migration changes what a login costs, and it says that at cost 12 bcrypt is
// in the same range as argon2id at the OWASP defaults.
func BenchmarkVerifyBcrypt(b *testing.B) {
	for name, encoded := range bcryptHashes {
		b.Run(name, func(b *testing.B) {
			var v Bcrypt
			ctx := b.Context()

			b.ReportAllocs()
			for b.Loop() {
				sinkBool, sinkErr = v.Verify(ctx, "hunter2", encoded)
			}
		})
	}
}

// BenchmarkMigrationVerify is what the composition costs on top of the hash
// itself, which had better be nothing measurable. Every sign in in an
// application that is migrating goes through it.
func BenchmarkMigrationVerify(b *testing.B) {
	h := Migrating(cheap(b), Bcrypt{MaxCost: 4})
	ctx := b.Context()

	encoded, err := h.Hash(ctx, "hunter2")
	if err != nil {
		b.Fatal(err)
	}

	b.Run("argon2id", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkBool, sinkErr = h.Verify(ctx, "hunter2", encoded)
		}
	})
	b.Run("bcrypt", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkBool, sinkErr = h.Verify(ctx, "hunter2", bcryptHashes["cost 4"])
		}
	})
}

// BenchmarkReads is the dispatch on its own, since it runs on every sign in
// before anything expensive happens and it is the one part of a Migration that
// a caller pays for whether or not there is an old hash to read.
func BenchmarkReads(b *testing.B) {
	var v Bcrypt
	const encoded = "$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8"

	b.ReportAllocs()
	for b.Loop() {
		sinkBool = v.Reads(encoded)
	}
}

// BenchmarkTune is the search and not the hashing, so it says what the command
// costs beyond the hashes it cannot avoid. A machine that answers instantly
// leaves only the arithmetic.
func BenchmarkTune(b *testing.B) {
	instant := func(ctx context.Context, p Params) (time.Duration, error) {
		return time.Duration(p.Memory) * time.Microsecond, nil
	}
	ctx := b.Context()

	b.ReportAllocs()
	for b.Loop() {
		sinkTuning, sinkErr = tune(ctx, Target{}, instant)
	}
}

var (
	sinkString string
	sinkBool   bool
	sinkErr    error
	sinkPHC    phc
	sinkTuning Tuning
)
