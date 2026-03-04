package lotus

import "testing"

func TestFieldNorm_RoundTrip(t *testing.T) {
	for i := uint32(0); i <= 23; i++ {
		b := fieldNormEncode(i)
		got := fieldNormDecode(b)
		if got != i {
			t.Fatalf("lossless range: encode(%d)=%d, decode=%d", i, b, got)
		}
	}
	for i := uint32(24); i < 100000; i++ {
		b := fieldNormEncode(i)
		decoded := fieldNormDecode(b)
		b2 := fieldNormEncode(decoded)
		if b2 != b {
			t.Fatalf("idempotency failed at %d: encode=%d, decode=%d, re-encode=%d", i, b, decoded, b2)
		}
	}
}

func TestFieldNorm_Monotonic(t *testing.T) {
	prev := uint8(0)
	for i := uint32(0); i < 10000; i++ {
		b := fieldNormEncode(i)
		if b < prev {
			t.Fatalf("not monotonic at %d: byte %d < prev %d", i, b, prev)
		}
		prev = b
	}
}

func TestFieldNorm_BM25Table(t *testing.T) {
	table := buildFieldNormBM25Table(500.0)
	for i, v := range table {
		if v < 0 {
			t.Fatalf("table[%d] = %f < 0", i, v)
		}
	}
	for i := 1; i < 256; i++ {
		if table[i] < table[i-1] {
			t.Fatalf("table[%d]=%f < table[%d]=%f", i, table[i], i-1, table[i-1])
		}
	}
	if table[1] >= 1.2 {
		t.Fatalf("expected table[1] < 1.2, got %f", table[1])
	}
}
