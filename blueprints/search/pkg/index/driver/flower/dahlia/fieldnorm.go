package dahlia

import "math"

// encodeFieldNorm encodes a field length to a single byte using Lucene SmallFloat.
// Values 0-23 are stored losslessly. Larger values use 4-bit mantissa + exponent.
func encodeFieldNorm(dl uint32) uint8 {
	if dl <= 23 {
		return uint8(dl)
	}
	// Find highest set bit position
	shift := 0
	v := dl
	for v > 0x1F { // 5 bits (1 implicit + 4 mantissa)
		v >>= 1
		shift++
	}
	// Encode: exponent in upper bits, mantissa in lower 4 bits
	mantissa := (dl >> uint(shift)) & 0x0F
	encoded := uint8((shift << 4) | int(mantissa))
	if encoded < 24 {
		encoded = 24
	}
	if encoded > 255 {
		return 255
	}
	return encoded
}

// decodeFieldNorm decodes a SmallFloat byte back to approximate field length.
func decodeFieldNorm(b uint8) uint32 {
	if b <= 23 {
		return uint32(b)
	}
	shift := uint(b >> 4)
	mantissa := uint32(b&0x0F) | 0x10 // implicit leading 1
	return mantissa << shift
}

// buildFieldNormBM25Table precomputes the BM25 denominator component
// k1 * (1 - b + b * dl / avgdl) for all 256 possible norm byte values.
func buildFieldNormBM25Table(avgDocLen float64) [256]float32 {
	var table [256]float32
	if avgDocLen <= 0 {
		avgDocLen = 1
	}
	for i := 0; i < 256; i++ {
		dl := float64(decodeFieldNorm(uint8(i)))
		table[i] = float32(bm25K1 * (1.0 - bm25B + bm25B*dl/avgDocLen))
	}
	return table
}

// fieldNormBM25Score computes BM25+ TF component using precomputed norm table.
func fieldNormBM25Score(tf float64, normComponent float32) float64 {
	return (tf * (bm25K1 + 1.0)) / (tf + float64(normComponent))
}

// fieldNormUpperBound returns the maximum possible BM25 TF score for a block,
// given the max TF and the shortest document norm in the block.
func fieldNormUpperBound(maxTF uint32, shortestNorm uint8, normTable [256]float32) float64 {
	tf := float64(maxTF)
	normComp := float64(normTable[shortestNorm])
	if normComp < 0 {
		normComp = 0
	}
	score := (tf*(bm25K1+1.0))/(tf+normComp) + bm25Delta
	if math.IsInf(score, 1) || math.IsNaN(score) {
		return 0
	}
	return score
}
