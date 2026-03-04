package dahlia

import (
	"encoding/binary"
	"math/bits"
)

// bp128Pack packs exactly 128 uint32 values using bit-packing.
// Format: [numBits:1 byte][packed data: numBits*16 bytes]
// Uses a uint64 accumulator for fast byte-level flushing.
func bp128Pack(vals []uint32) []byte {
	if len(vals) != blockSize {
		panic("bp128Pack: need exactly 128 values")
	}
	var maxVal uint32
	for _, v := range vals {
		if v > maxVal {
			maxVal = v
		}
	}
	numBits := uint(0)
	if maxVal > 0 {
		numBits = uint(bits.Len32(maxVal))
	}
	if numBits == 0 {
		return []byte{0}
	}
	dataBytes := int(numBits) * 16
	out := make([]byte, 1+dataBytes)
	out[0] = byte(numBits)

	var acc uint64
	var accBits uint
	off := 1
	for _, v := range vals {
		acc |= uint64(v) << accBits
		accBits += numBits
		for accBits >= 8 {
			out[off] = byte(acc)
			acc >>= 8
			accBits -= 8
			off++
		}
	}
	if accBits > 0 {
		out[off] = byte(acc)
	}
	return out
}

// bp128Unpack reads 128 uint32 values from a bitpacked buffer.
// Returns the number of bytes consumed.
// Uses a uint64 accumulator for fast byte-level reading.
func bp128Unpack(buf []byte, out []uint32) int {
	if len(buf) == 0 {
		panic("bp128Unpack: empty buffer")
	}
	numBits := uint(buf[0])
	if numBits == 0 {
		for i := range out[:blockSize] {
			out[i] = 0
		}
		return 1
	}
	dataBytes := int(numBits) * 16
	mask := uint64((1 << numBits) - 1)
	var acc uint64
	var accBits uint
	off := 1

	for i := 0; i < blockSize; i++ {
		for accBits < numBits {
			acc |= uint64(buf[off]) << accBits
			accBits += 8
			off++
		}
		out[i] = uint32(acc & mask)
		acc >>= numBits
		accBits -= numBits
	}
	return 1 + dataBytes
}

// bp128PackedSize returns the byte size of a packed block for the given max value.
func bp128PackedSize(maxVal uint32) int {
	if maxVal == 0 {
		return 1
	}
	numBits := bits.Len32(maxVal)
	return 1 + numBits*16
}

// bp128DocBlock packs 128 doc IDs as deltas and returns the packed bytes.
func bp128DocBlock(docs []uint32, prevDoc uint32) []byte {
	var deltas [blockSize]uint32
	prev := prevDoc
	for i, d := range docs[:blockSize] {
		deltas[i] = d - prev
		prev = d
	}
	return bp128Pack(deltas[:])
}

// bp128FreqBlock packs 128 frequencies.
func bp128FreqBlock(freqs []uint32) []byte {
	return bp128Pack(freqs[:blockSize])
}

// appendU32LE appends a uint32 in little-endian.
func appendU32LE(dst []byte, v uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	return append(dst, buf[:]...)
}

// readU32LE reads a little-endian uint32.
func readU32LE(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}
