package dahlia

import "io"

// vintPut writes a variable-byte encoded uint32 to w.
// Returns bytes written and any error.
func vintPut(w io.Writer, v uint32) (int, error) {
	var buf [5]byte
	n := vintEncode(buf[:], v)
	return w.Write(buf[:n])
}

// vintEncode encodes v into buf, returns bytes written.
func vintEncode(buf []byte, v uint32) int {
	i := 0
	for v >= 0x80 {
		buf[i] = byte(v) | 0x80
		v >>= 7
		i++
	}
	buf[i] = byte(v)
	return i + 1
}

// vintGet decodes a variable-byte uint32 from buf.
// Returns the value and number of bytes consumed.
func vintGet(buf []byte) (uint32, int) {
	var v uint32
	var shift uint
	for i, b := range buf {
		v |= uint32(b&0x7F) << shift
		if b < 0x80 {
			return v, i + 1
		}
		shift += 7
		if shift >= 35 {
			break
		}
	}
	return v, len(buf)
}

// vintSize returns the encoded size of v in bytes.
func vintSize(v uint32) int {
	if v < 1<<7 {
		return 1
	}
	if v < 1<<14 {
		return 2
	}
	if v < 1<<21 {
		return 3
	}
	if v < 1<<28 {
		return 4
	}
	return 5
}
