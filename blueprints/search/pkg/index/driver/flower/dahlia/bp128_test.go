package dahlia

import (
	"math/rand"
	"testing"
)

func TestBP128RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		gen  func() [blockSize]uint32
	}{
		{"zeros", func() [blockSize]uint32 { return [blockSize]uint32{} }},
		{"small_deltas", func() [blockSize]uint32 {
			var v [blockSize]uint32
			for i := range v {
				v[i] = uint32(i % 10)
			}
			return v
		}},
		{"large_values", func() [blockSize]uint32 {
			var v [blockSize]uint32
			for i := range v {
				v[i] = uint32(i * 100000)
			}
			return v
		}},
		{"max_bits", func() [blockSize]uint32 {
			var v [blockSize]uint32
			for i := range v {
				v[i] = ^uint32(0)
			}
			return v
		}},
		{"random", func() [blockSize]uint32 {
			var v [blockSize]uint32
			rng := rand.New(rand.NewSource(42))
			for i := range v {
				v[i] = rng.Uint32()
			}
			return v
		}},
		{"single_bit", func() [blockSize]uint32 {
			var v [blockSize]uint32
			for i := range v {
				v[i] = uint32(i & 1)
			}
			return v
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals := tt.gen()
			packed := bp128Pack(vals[:])
			var out [blockSize]uint32
			n := bp128Unpack(packed, out[:])
			if n != len(packed) {
				t.Fatalf("consumed %d bytes, packed %d", n, len(packed))
			}
			for i := 0; i < blockSize; i++ {
				if out[i] != vals[i] {
					t.Fatalf("mismatch at %d: got %d, want %d", i, out[i], vals[i])
				}
			}
		})
	}
}

func TestBP128ZeroSize(t *testing.T) {
	var vals [blockSize]uint32
	packed := bp128Pack(vals[:])
	if len(packed) != 1 {
		t.Fatalf("zero block should be 1 byte, got %d", len(packed))
	}
	if packed[0] != 0 {
		t.Fatalf("zero block header should be 0, got %d", packed[0])
	}
}

func TestBP128MaxSize(t *testing.T) {
	var vals [blockSize]uint32
	for i := range vals {
		vals[i] = ^uint32(0)
	}
	packed := bp128Pack(vals[:])
	// 1 header + 32*16 = 513 bytes
	if len(packed) != 513 {
		t.Fatalf("max block should be 513 bytes, got %d", len(packed))
	}
}

func TestBP128DocBlock(t *testing.T) {
	var docs [blockSize]uint32
	for i := range docs {
		docs[i] = uint32(i * 3) // 0, 3, 6, 9, ...
	}
	packed := bp128DocBlock(docs[:], 0)
	var deltas [blockSize]uint32
	bp128Unpack(packed, deltas[:])
	// All deltas should be 3 (except first which is 0)
	if deltas[0] != 0 {
		t.Fatalf("first delta should be 0, got %d", deltas[0])
	}
	for i := 1; i < blockSize; i++ {
		if deltas[i] != 3 {
			t.Fatalf("delta[%d] should be 3, got %d", i, deltas[i])
		}
	}
}

func BenchmarkBP128Pack(b *testing.B) {
	var vals [blockSize]uint32
	rng := rand.New(rand.NewSource(42))
	for i := range vals {
		vals[i] = rng.Uint32() >> 16 // 16-bit values
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp128Pack(vals[:])
	}
}

func BenchmarkBP128Unpack(b *testing.B) {
	var vals [blockSize]uint32
	rng := rand.New(rand.NewSource(42))
	for i := range vals {
		vals[i] = rng.Uint32() >> 16
	}
	packed := bp128Pack(vals[:])
	var out [blockSize]uint32
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp128Unpack(packed, out[:])
	}
}
