package lotus

import (
	"math/rand"
	"testing"
)

func TestBP128_RoundTrip_SmallDeltas(t *testing.T) {
	vals := make([]uint32, 128)
	for i := range vals {
		vals[i] = uint32(i % 8)
	}
	buf := bp128Pack(vals)
	if len(buf) != 1+3*16 {
		t.Fatalf("expected 49 bytes, got %d", len(buf))
	}
	got := make([]uint32, 128)
	bp128Unpack(buf, got)
	for i, v := range got {
		if v != vals[i] {
			t.Fatalf("mismatch at %d: got %d want %d", i, v, vals[i])
		}
	}
}

func TestBP128_RoundTrip_LargeValues(t *testing.T) {
	vals := make([]uint32, 128)
	for i := range vals {
		vals[i] = uint32(rand.Intn(1 << 20))
	}
	buf := bp128Pack(vals)
	got := make([]uint32, 128)
	bp128Unpack(buf, got)
	for i, v := range got {
		if v != vals[i] {
			t.Fatalf("mismatch at %d: got %d want %d", i, v, vals[i])
		}
	}
}

func TestBP128_ZeroBits(t *testing.T) {
	vals := make([]uint32, 128)
	buf := bp128Pack(vals)
	if len(buf) != 1 {
		t.Fatalf("expected 1 byte for all-zero block, got %d", len(buf))
	}
	got := make([]uint32, 128)
	bp128Unpack(buf, got)
	for i, v := range got {
		if v != 0 {
			t.Fatalf("mismatch at %d: got %d want 0", i, v)
		}
	}
}

func TestBP128_MaxBits(t *testing.T) {
	vals := make([]uint32, 128)
	vals[0] = 0xFFFFFFFF
	buf := bp128Pack(vals)
	if len(buf) != 1+32*16 {
		t.Fatalf("expected %d bytes, got %d", 1+32*16, len(buf))
	}
	got := make([]uint32, 128)
	bp128Unpack(buf, got)
	if got[0] != 0xFFFFFFFF {
		t.Fatalf("got %x want %x", got[0], uint32(0xFFFFFFFF))
	}
}

func TestBP128_AllOnes(t *testing.T) {
	vals := make([]uint32, 128)
	for i := range vals {
		vals[i] = 1
	}
	buf := bp128Pack(vals)
	if len(buf) != 1+1*16 {
		t.Fatalf("expected 17 bytes, got %d", len(buf))
	}
	got := make([]uint32, 128)
	bp128Unpack(buf, got)
	for i, v := range got {
		if v != 1 {
			t.Fatalf("mismatch at %d: got %d want 1", i, v)
		}
	}
}

func BenchmarkBP128_Pack(b *testing.B) {
	vals := make([]uint32, 128)
	for i := range vals {
		vals[i] = uint32(rand.Intn(256))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp128Pack(vals)
	}
}

func BenchmarkBP128_Unpack(b *testing.B) {
	vals := make([]uint32, 128)
	for i := range vals {
		vals[i] = uint32(rand.Intn(256))
	}
	buf := bp128Pack(vals)
	out := make([]uint32, 128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp128Unpack(buf, out)
	}
}
