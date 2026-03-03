package lotus

import "math/bits"

// bp128Pack packs exactly 128 uint32 values into a bitpacked byte slice.
// Format: [numBits: 1 byte] [data: numBits×16 bytes]
func bp128Pack(vals []uint32) []byte {
	var maxVal uint32
	for _, v := range vals[:128] {
		if v > maxVal {
			maxVal = v
		}
	}
	numBits := bitsNeeded(maxVal)
	size := 1 + int(numBits)*16
	buf := make([]byte, size)
	buf[0] = numBits
	if numBits == 0 {
		return buf
	}
	data := buf[1:]
	bitPos := uint(0)
	nb := uint(numBits)
	for _, v := range vals[:128] {
		byteOff := bitPos >> 3
		bitOff := bitPos & 7
		remaining := nb
		val := uint64(v)
		for remaining > 0 {
			space := 8 - bitOff
			if space > remaining {
				space = remaining
			}
			mask := uint64((1 << space) - 1)
			data[byteOff] |= byte((val & mask) << bitOff)
			val >>= space
			remaining -= space
			bitOff = 0
			byteOff++
		}
		bitPos += nb
	}
	return buf
}

// bp128Unpack decodes 128 uint32 values from a bitpacked byte slice into out.
func bp128Unpack(buf []byte, out []uint32) {
	numBits := uint(buf[0])
	if numBits == 0 {
		for i := range out[:128] {
			out[i] = 0
		}
		return
	}
	data := buf[1:]
	bitPos := uint(0)
	for i := 0; i < 128; i++ {
		byteOff := bitPos >> 3
		bitOff := bitPos & 7
		remaining := numBits
		var val uint64
		shift := uint(0)
		for remaining > 0 {
			space := 8 - bitOff
			if space > remaining {
				space = remaining
			}
			mask := uint64((1 << space) - 1)
			val |= (uint64(data[byteOff]) >> bitOff & mask) << shift
			shift += space
			remaining -= space
			bitOff = 0
			byteOff++
		}
		out[i] = uint32(val)
		bitPos += numBits
	}
}

// bitsNeeded returns the minimum bits to represent v (0 for v==0).
func bitsNeeded(v uint32) uint8 {
	if v == 0 {
		return 0
	}
	return uint8(32 - bits.LeadingZeros32(v))
}
