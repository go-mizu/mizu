package lotus

import "math/bits"

const numFreeValues = 24

// fieldNormEncode encodes a document length to a single byte (Lucene SmallFloat).
func fieldNormEncode(i uint32) uint8 {
	if i < numFreeValues {
		return uint8(i)
	}
	return uint8(numFreeValues) + longToInt4(i-numFreeValues)
}

// fieldNormDecode decodes a fieldnorm byte back to the approximate token count.
func fieldNormDecode(b uint8) uint32 {
	if b < numFreeValues {
		return uint32(b)
	}
	return numFreeValues + int4ToLong(b-numFreeValues)
}

func longToInt4(i uint32) uint8 {
	if i == 0 {
		return 0
	}
	numBits := 32 - bits.LeadingZeros32(i)
	if numBits < 4 {
		return uint8(i)
	}
	shift := uint(numBits - 4)
	encoded := (i >> shift) & 0x07
	encoded |= (uint32(shift) + 1) << 3
	return uint8(encoded)
}

func int4ToLong(b uint8) uint32 {
	mantissa := uint32(b) & 0x07
	shift := int(b>>3) - 1
	if shift < 0 {
		return mantissa
	}
	return (mantissa | 0x08) << uint(shift)
}

// buildFieldNormBM25Table precomputes BM25 length normalization for all 256 byte values.
func buildFieldNormBM25Table(avgDocLen float64) [256]float32 {
	const k1, b = 1.2, 0.75
	var table [256]float32
	for i := 0; i < 256; i++ {
		dl := float64(fieldNormDecode(uint8(i)))
		table[i] = float32(k1 * (1.0 - b + b*dl/avgDocLen))
	}
	return table
}
