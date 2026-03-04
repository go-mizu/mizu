package dahlia

import "testing"

func TestFieldNormLossless(t *testing.T) {
	for i := uint32(0); i <= 40; i++ {
		enc := encodeFieldNorm(i)
		dec := decodeFieldNorm(enc)
		if dec != i {
			t.Fatalf("lossless range: encode(%d)=%d, decode=%d", i, enc, dec)
		}
	}
}

func TestFieldNormQuantizationMatchesTantivyExamples(t *testing.T) {
	cases := []struct {
		dl   uint32
		want uint32
	}{
		{41, 40},
		{42, 42},
		{43, 42},
		{894, 856},
		{943, 920},
		{944, 920},
		{945, 920},
	}
	for _, tc := range cases {
		got := decodeFieldNorm(encodeFieldNorm(tc.dl))
		if got != tc.want {
			t.Fatalf("norm(%d)=%d, want %d", tc.dl, got, tc.want)
		}
	}
}

func TestFieldNormMonotonic(t *testing.T) {
	prev := uint32(0)
	for i := 0; i < 256; i++ {
		dl := decodeFieldNorm(uint8(i))
		if dl < prev {
			t.Fatalf("non-monotonic: decode(%d)=%d < decode(%d)=%d", i, dl, i-1, prev)
		}
		prev = dl
	}
}

func TestFieldNormIdempotent(t *testing.T) {
	for i := 0; i < 256; i++ {
		b := uint8(i)
		dl := decodeFieldNorm(b)
		b2 := encodeFieldNorm(dl)
		dl2 := decodeFieldNorm(b2)
		if dl2 != dl {
			t.Fatalf("not idempotent at %d: dl=%d, re-encoded=%d, re-decoded=%d", i, dl, b2, dl2)
		}
	}
}

func TestFieldNormBM25Table(t *testing.T) {
	table := buildFieldNormBM25Table(100.0)
	// Table should have entries for all 256 values
	// Shorter docs should have higher norm component (penalized less)
	// dl=0 should give k1*(1-b) = 1.2*0.25 = 0.3
	if table[0] < 0.29 || table[0] > 0.31 {
		t.Fatalf("table[0] (dl=0) = %f, want ~0.3", table[0])
	}
	// near avgdl should remain close to k1.
	normAtAvg := encodeFieldNorm(100) // decodes to 96.
	if table[normAtAvg] < 1.0 || table[normAtAvg] > 1.3 {
		t.Fatalf("table[norm(100)] = %f, want near [1.0,1.3]", table[normAtAvg])
	}
}
