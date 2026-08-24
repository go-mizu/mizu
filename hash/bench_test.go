package hash

import (
	"fmt"
	"testing"
)

func BenchmarkBlake2b(b *testing.B) {
	for _, size := range []int{64, 1024, 65536} {
		in := counting(size)
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()

			for b.Loop() {
				sinkSlice = blake2bSum(maxDigest, in)
			}
		})
	}
}

// BenchmarkArgon2id is the number that decides the parameters. The default is
// the OWASP recommendation, 19 MiB over two passes, and what it costs on the
// machine running it is the whole question hash:tune answers.
func BenchmarkArgon2id(b *testing.B) {
	password := []byte("correct horse battery staple")
	salt := counting(16)

	cases := []struct {
		name         string
		time, memory uint32
		lanes        uint8
	}{
		{"owasp", 2, 19456, 1},
		{"owasp/4 lanes", 2, 19456, 4},
		{"64MiB", 3, 65536, 1},
		{"cheap", 1, 64, 1},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				sinkSlice = argon2id(password, salt, nil, nil, c.time, c.memory, c.lanes, 32)
			}
		})
	}
}

// BenchmarkFillBlock is the compression function on its own, which is where
// every one of those milliseconds goes.
func BenchmarkFillBlock(b *testing.B) {
	var prev, ref, next block
	for i := range prev {
		prev[i] = uint64(i)
		ref[i] = uint64(i) * 7
	}
	b.SetBytes(argonBlockSize)
	b.ReportAllocs()

	for b.Loop() {
		fillBlock(&prev, &ref, &next, false)
	}
}

var sinkSlice []byte
