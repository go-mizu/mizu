package lotus

import (
	"bytes"
	"testing"
)

func TestVInt_RoundTrip(t *testing.T) {
	cases := []uint32{0, 1, 127, 128, 16383, 16384, 1<<21 - 1, 1 << 28, 0xFFFFFFFF}
	for _, v := range cases {
		var buf bytes.Buffer
		vintPut(&buf, v)
		got, n := vintGet(buf.Bytes())
		if got != v {
			t.Fatalf("roundtrip %d: got %d", v, got)
		}
		if n != buf.Len() {
			t.Fatalf("bytes consumed %d != written %d for value %d", n, buf.Len(), v)
		}
	}
}

func TestVInt_MultipleValues(t *testing.T) {
	vals := []uint32{300, 0, 1, 100000, 0xFFFFFFFF}
	var buf bytes.Buffer
	for _, v := range vals {
		vintPut(&buf, v)
	}
	data := buf.Bytes()
	off := 0
	for i, want := range vals {
		got, n := vintGet(data[off:])
		if got != want {
			t.Fatalf("value[%d]: got %d want %d", i, got, want)
		}
		off += n
	}
	if off != len(data) {
		t.Fatalf("consumed %d bytes, total %d", off, len(data))
	}
}

func TestVInt_ByteCount(t *testing.T) {
	cases := []struct {
		val  uint32
		want int
	}{
		{0, 1}, {127, 1},
		{128, 2}, {16383, 2},
		{16384, 3}, {1<<21 - 1, 3},
		{1 << 21, 4}, {1<<28 - 1, 4},
		{1 << 28, 5}, {0xFFFFFFFF, 5},
	}
	for _, c := range cases {
		got := vintSize(c.val)
		if got != c.want {
			t.Fatalf("vintSize(%d) = %d, want %d", c.val, got, c.want)
		}
		var buf bytes.Buffer
		vintPut(&buf, c.val)
		if buf.Len() != c.want {
			t.Fatalf("vintPut(%d) wrote %d bytes, want %d", c.val, buf.Len(), c.want)
		}
	}
}
