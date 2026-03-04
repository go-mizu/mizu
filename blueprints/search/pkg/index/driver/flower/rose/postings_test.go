package rose

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Task 1: VByte codec tests
// ---------------------------------------------------------------------------

func TestVByte_RoundTrip(t *testing.T) {
	cases := []uint32{0, 1, 127, 128, 16383, 16384, 2097151, 2097152, 1<<28 - 1, math.MaxUint32}
	for _, v := range cases {
		buf := vbyteEncode(nil, v)
		got, _, err := vbyteDecode(buf, 0)
		if err != nil {
			t.Errorf("vbyte(%d): unexpected error: %v", v, err)
		}
		if got != v {
			t.Errorf("vbyte(%d): got %d", v, got)
		}
	}
}

func TestVByte_Sizes(t *testing.T) {
	sizes := []struct {
		v    uint32
		want int
	}{
		{127, 1},
		{128, 2},
		{16383, 2},
		{16384, 3},
	}
	for _, tc := range sizes {
		buf := vbyteEncode(nil, tc.v)
		if len(buf) != tc.want {
			t.Errorf("vbyteEncode(%d): len=%d, want %d", tc.v, len(buf), tc.want)
		}
	}
}

func TestVByte_Sequence(t *testing.T) {
	vals := []uint32{5, 200, 130, 16000, 0, 300}
	var buf []byte
	for _, v := range vals {
		buf = vbyteEncode(buf, v)
	}
	pos := 0
	for _, want := range vals {
		got, newPos, err := vbyteDecode(buf, pos)
		if err != nil {
			t.Fatalf("decode pos %d: unexpected error: %v", pos, err)
		}
		if got != want {
			t.Fatalf("decode pos %d: got %d, want %d", pos, got, want)
		}
		pos = newPos
	}
}

// ---------------------------------------------------------------------------
// Task 2: Block pack / unpack tests
// ---------------------------------------------------------------------------

func TestBlock_RoundTrip(t *testing.T) {
	docIDs := []uint32{10, 25, 130, 300, 500}
	impacts := []uint8{200, 50, 180, 10, 255}
	data, bmi := packBlock(docIDs, impacts, 0)
	if bmi != 255 {
		t.Errorf("BlockMaxImpact: got %d, want 255", bmi)
	}
	gotIDs, gotImp, _, err := unpackBlock(data, 0, len(docIDs))
	if err != nil {
		t.Fatalf("unpackBlock: %v", err)
	}
	for i, want := range docIDs {
		if gotIDs[i] != want {
			t.Errorf("docID[%d]: got %d, want %d", i, gotIDs[i], want)
		}
	}
	for i, want := range impacts {
		if gotImp[i] != want {
			t.Errorf("impact[%d]: got %d, want %d", i, gotImp[i], want)
		}
	}
}

func TestBlock_MaxImpact(t *testing.T) {
	_, bmi := packBlock([]uint32{1, 2, 3}, []uint8{10, 200, 50}, 0)
	if bmi != 200 {
		t.Errorf("got %d, want 200", bmi)
	}
}

func TestBlock_Full128(t *testing.T) {
	ids := make([]uint32, 128)
	imp := make([]uint8, 128)
	for i := range ids {
		ids[i] = uint32(i * 10)
		imp[i] = uint8(i + 1)
	}
	data, _ := packBlock(ids, imp, 0)
	gotIDs, gotImp, _, err := unpackBlock(data, 0, 128)
	if err != nil {
		t.Fatalf("%v", err)
	}
	for i := range ids {
		if gotIDs[i] != ids[i] || gotImp[i] != imp[i] {
			t.Errorf("mismatch at %d", i)
		}
	}
}

func TestBlock_WithNonZeroBase(t *testing.T) {
	// blockBase=100, docIDs=[110, 150, 200] → deltas=[10, 40, 50]
	docIDs := []uint32{110, 150, 200}
	data, _ := packBlock(docIDs, []uint8{1, 2, 3}, 100)
	gotIDs, _, _, err := unpackBlock(data, 100, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range docIDs {
		if gotIDs[i] != want {
			t.Errorf("[%d]: got %d, want %d", i, gotIDs[i], want)
		}
	}
}

func TestBlock_UnpackZero(t *testing.T) {
	ids, imp, consumed, err := unpackBlock(nil, 0, 0)
	if err != nil || ids != nil || imp != nil || consumed != 0 {
		t.Errorf("unpackBlock n=0: got ids=%v imp=%v consumed=%d err=%v, want nil nil 0 nil", ids, imp, consumed, err)
	}
}

func TestVByte_TruncatedDecode(t *testing.T) {
	// Encode a value that requires 2 bytes (e.g. 128), then pass only 1 byte.
	full := vbyteEncode(nil, 128)
	if len(full) != 2 {
		t.Fatalf("expected 2-byte encoding for 128, got %d bytes", len(full))
	}
	// Pass only the first byte (continuation byte with MSB set).
	_, _, err := vbyteDecode(full[:1], 0)
	if err == nil {
		t.Error("expected error on truncated buffer, got nil")
	}
}

func TestBlock_TruncatedData(t *testing.T) {
	docIDs := []uint32{10, 25, 130}
	impacts := []uint8{1, 2, 3}
	data, _ := packBlock(docIDs, impacts, 0)
	// Truncate data so that it cannot hold all deltas.
	_, _, _, err := unpackBlock(data[:1], 0, len(docIDs))
	if err == nil {
		t.Error("expected error on truncated block data, got nil")
	}
}
