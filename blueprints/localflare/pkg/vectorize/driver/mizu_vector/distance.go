package mizu_vector

import "math"

// DistanceFunc is a function that computes distance between two vectors.
type DistanceFunc func(a, b []float32) float32

// CosineDistance computes 1 - cosine_similarity with SIMD-friendly loop unrolling.
// Returns 0 for identical vectors, 2 for opposite vectors.
func CosineDistance(a, b []float32) float32 {
	var dot, normA, normB float32

	// Process 8 elements at a time for better CPU pipeline utilization
	n := len(a)
	i := 0
	for ; i <= n-8; i += 8 {
		dot += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]
		normA += a[i]*a[i] + a[i+1]*a[i+1] + a[i+2]*a[i+2] + a[i+3]*a[i+3] +
			a[i+4]*a[i+4] + a[i+5]*a[i+5] + a[i+6]*a[i+6] + a[i+7]*a[i+7]
		normB += b[i]*b[i] + b[i+1]*b[i+1] + b[i+2]*b[i+2] + b[i+3]*b[i+3] +
			b[i+4]*b[i+4] + b[i+5]*b[i+5] + b[i+6]*b[i+6] + b[i+7]*b[i+7]
	}

	// Handle remaining elements
	for ; i < n; i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 1.0
	}

	sim := dot / float32(math.Sqrt(float64(normA*normB)))
	return 1.0 - sim
}

// EuclideanDistance computes squared L2 distance with loop unrolling.
func EuclideanDistance(a, b []float32) float32 {
	var sum float32
	n := len(a)
	i := 0

	// Process 8 elements at a time
	for ; i <= n-8; i += 8 {
		d0, d1 := a[i]-b[i], a[i+1]-b[i+1]
		d2, d3 := a[i+2]-b[i+2], a[i+3]-b[i+3]
		d4, d5 := a[i+4]-b[i+4], a[i+5]-b[i+5]
		d6, d7 := a[i+6]-b[i+6], a[i+7]-b[i+7]
		sum += d0*d0 + d1*d1 + d2*d2 + d3*d3 + d4*d4 + d5*d5 + d6*d6 + d7*d7
	}

	for ; i < n; i++ {
		d := a[i] - b[i]
		sum += d * d
	}

	return sum
}

// NegDotProduct returns negative dot product (for similarity search).
// Lower values = more similar.
func NegDotProduct(a, b []float32) float32 {
	var sum float32
	n := len(a)
	i := 0

	for ; i <= n-8; i += 8 {
		sum += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]
	}

	for ; i < n; i++ {
		sum += a[i] * b[i]
	}

	return -sum
}

// DotProduct computes dot product.
func DotProduct(a, b []float32) float32 {
	var sum float32
	n := len(a)
	i := 0

	for ; i <= n-8; i += 8 {
		sum += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]
	}

	for ; i < n; i++ {
		sum += a[i] * b[i]
	}

	return sum
}

// L2Norm computes the L2 norm of a vector.
func L2Norm(v []float32) float32 {
	var sum float32
	for _, x := range v {
		sum += x * x
	}
	return float32(math.Sqrt(float64(sum)))
}

// Normalize normalizes a vector in-place.
func Normalize(v []float32) {
	norm := L2Norm(v)
	if norm == 0 {
		return
	}
	for i := range v {
		v[i] /= norm
	}
}

// NormalizedCopy returns a normalized copy of the vector.
func NormalizedCopy(v []float32) []float32 {
	result := make([]float32, len(v))
	copy(result, v)
	Normalize(result)
	return result
}

// HammingDistance computes Hamming distance between two bit vectors (as uint64 slices).
func HammingDistance(a, b []uint64) int {
	var dist int
	for i := range a {
		dist += popcount(a[i] ^ b[i])
	}
	return dist
}

// popcount counts the number of 1 bits in x.
func popcount(x uint64) int {
	// Brian Kernighan's algorithm
	var count int
	for x != 0 {
		x &= x - 1
		count++
	}
	return count
}

// AsymmetricDistance computes distance between a query vector and a quantized vector.
// Used by PQ and RaBitQ for fast distance computation.
func AsymmetricDistance(queryDists [][]float32, codes []byte, m int) float32 {
	var sum float32
	for i := 0; i < m; i++ {
		sum += queryDists[i][codes[i]]
	}
	return sum
}
