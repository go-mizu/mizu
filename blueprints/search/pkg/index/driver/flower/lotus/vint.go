package lotus

import "io"

// vintPut writes v as a variable-byte integer to w.
func vintPut(w io.ByteWriter, v uint32) {
	for v >= 0x80 {
		w.WriteByte(byte(v&0x7F) | 0x80)
		v >>= 7
	}
	w.WriteByte(byte(v))
}

// vintGet reads a variable-byte integer from buf.
func vintGet(buf []byte) (uint32, int) {
	var v uint32
	for i, b := range buf {
		if i >= 5 {
			break
		}
		v |= uint32(b&0x7F) << (7 * uint(i))
		if b&0x80 == 0 {
			return v, i + 1
		}
	}
	return v, len(buf)
}

// vintSize returns the number of bytes needed to encode v.
func vintSize(v uint32) int {
	n := 1
	for v >= 0x80 {
		n++
		v >>= 7
	}
	return n
}
