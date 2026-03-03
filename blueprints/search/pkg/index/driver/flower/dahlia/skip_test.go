package dahlia

import "testing"

func TestSkipEntryRoundTrip(t *testing.T) {
	entries := []skipEntry{
		{0, 0, 0, 0, 0, 0},
		{127, 100, 200, 300, 5, 10},
		{1000000, 999999, 888888, 777777, 42, 255},
		{^uint32(0), ^uint32(0), ^uint32(0), ^uint32(0), ^uint32(0), 255},
	}
	var buf [skipEntrySize]byte
	for _, e := range entries {
		encodeSkipEntry(buf[:], e)
		got := decodeSkipEntry(buf[:])
		if got != e {
			t.Fatalf("round-trip failed: got %+v, want %+v", got, e)
		}
	}
}

func TestSkipEntrySize(t *testing.T) {
	// Verify the constant matches actual encoding size
	if skipEntrySize != 21 {
		t.Fatalf("skipEntrySize = %d, want 21", skipEntrySize)
	}
}

func TestSkipIndexFindBlock(t *testing.T) {
	idx := skipIndex{
		{lastDoc: 127},  // block 0: docs 0-127
		{lastDoc: 255},  // block 1: docs 128-255
		{lastDoc: 383},  // block 2: docs 256-383
		{lastDoc: 500},  // block 3: docs 384-500
	}

	tests := []struct {
		target uint32
		want   int
	}{
		{0, 0},
		{50, 0},
		{127, 0},
		{128, 1},
		{255, 1},
		{256, 2},
		{383, 2},
		{384, 3},
		{500, 3},
		{501, -1},
	}
	for _, tt := range tests {
		got := idx.findBlock(tt.target)
		if got != tt.want {
			t.Errorf("findBlock(%d) = %d, want %d", tt.target, got, tt.want)
		}
	}
}
