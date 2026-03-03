package dahlia

import (
	"bytes"
	"testing"
)

func TestVIntRoundTrip(t *testing.T) {
	values := []uint32{0, 1, 127, 128, 255, 256, 16383, 16384, 1<<21 - 1, 1 << 21, 1<<28 - 1, 1 << 28, ^uint32(0)}
	for _, v := range values {
		var buf bytes.Buffer
		n, err := vintPut(&buf, v)
		if err != nil {
			t.Fatalf("vintPut(%d): %v", v, err)
		}
		if n != vintSize(v) {
			t.Fatalf("vintPut(%d) wrote %d bytes, vintSize says %d", v, n, vintSize(v))
		}
		got, consumed := vintGet(buf.Bytes())
		if got != v {
			t.Fatalf("vintGet: got %d, want %d", got, v)
		}
		if consumed != n {
			t.Fatalf("vintGet consumed %d, written %d", consumed, n)
		}
	}
}

func TestVIntMultiple(t *testing.T) {
	var buf bytes.Buffer
	values := []uint32{100, 200, 300, 0, ^uint32(0)}
	for _, v := range values {
		vintPut(&buf, v)
	}
	data := buf.Bytes()
	off := 0
	for i, want := range values {
		got, n := vintGet(data[off:])
		if got != want {
			t.Fatalf("value %d: got %d, want %d", i, got, want)
		}
		off += n
	}
	if off != len(data) {
		t.Fatalf("consumed %d bytes, total %d", off, len(data))
	}
}

func TestVIntSizes(t *testing.T) {
	tests := []struct {
		v    uint32
		size int
	}{
		{0, 1}, {127, 1}, {128, 2}, {16383, 2}, {16384, 3},
		{1<<21 - 1, 3}, {1 << 21, 4}, {1<<28 - 1, 4}, {1 << 28, 5}, {^uint32(0), 5},
	}
	for _, tt := range tests {
		if got := vintSize(tt.v); got != tt.size {
			t.Fatalf("vintSize(%d) = %d, want %d", tt.v, got, tt.size)
		}
	}
}
