package vectorize

import "math"

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value in [-1, 1] where 1 means identical direction.
// Returns 0 if either vector has zero magnitude or different lengths.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProd, normA, normB float32
	for i := range a {
		dotProd += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProd / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// EuclideanDistance computes the L2 (Euclidean) distance between two vectors.
// Returns a value in [0, inf) where 0 means identical vectors.
// Returns 0 if vectors have different lengths.
func EuclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum)))
}

// InnerProduct computes the inner product (dot product) of two vectors.
// For normalized vectors, higher values indicate more similarity.
// Returns 0 if vectors have different lengths.
func InnerProduct(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}

	return sum
}

// Normalize returns a unit vector (magnitude 1) in the same direction.
// Returns nil for zero vectors or empty input.
func Normalize(v []float32) []float32 {
	if len(v) == 0 {
		return nil
	}

	var magnitude float32
	for _, val := range v {
		magnitude += val * val
	}

	if magnitude == 0 {
		return nil
	}

	magnitude = float32(math.Sqrt(float64(magnitude)))
	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = val / magnitude
	}

	return result
}

// Magnitude returns the L2 norm (magnitude) of a vector.
func Magnitude(v []float32) float32 {
	if len(v) == 0 {
		return 0
	}

	var sum float32
	for _, val := range v {
		sum += val * val
	}

	return float32(math.Sqrt(float64(sum)))
}

// ComputeScore computes the similarity score based on the specified metric.
// For Cosine and DotProduct, higher is more similar.
// For Euclidean, the score is converted to 1/(1+distance) so higher is more similar.
func ComputeScore(a, b []float32, metric DistanceMetric) float32 {
	switch metric {
	case Cosine:
		return CosineSimilarity(a, b)
	case Euclidean:
		// Convert distance to similarity: 1/(1+d) gives range (0, 1]
		return 1.0 / (1.0 + EuclideanDistance(a, b))
	case DotProduct:
		return InnerProduct(a, b)
	default:
		return CosineSimilarity(a, b)
	}
}
