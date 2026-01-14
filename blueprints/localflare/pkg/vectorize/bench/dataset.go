package bench

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// Dataset holds generated test data.
type Dataset struct {
	Vectors      []*vectorize.Vector
	QueryVectors [][]float32
	Dimensions   int
}

// GenerateDataset creates a reproducible test dataset.
func GenerateDataset(size, dimensions, numQueries int, seed int64) *Dataset {
	rng := rand.New(rand.NewSource(seed))

	vectors := make([]*vectorize.Vector, size)
	for i := 0; i < size; i++ {
		vectors[i] = &vectorize.Vector{
			ID:        generateUUID(seed, i),
			Values:    generateNormalizedVector(rng, dimensions),
			Namespace: fmt.Sprintf("ns_%d", i%10), // 10 namespaces
			Metadata: map[string]any{
				"category": fmt.Sprintf("cat_%d", i%5),
				"index":    i,
			},
		}
	}

	queries := make([][]float32, numQueries)
	for i := 0; i < numQueries; i++ {
		queries[i] = generateNormalizedVector(rng, dimensions)
	}

	return &Dataset{
		Vectors:      vectors,
		QueryVectors: queries,
		Dimensions:   dimensions,
	}
}

// generateNormalizedVector creates a random unit vector.
func generateNormalizedVector(rng *rand.Rand, dimensions int) []float32 {
	vec := make([]float32, dimensions)
	var magnitude float64

	for i := 0; i < dimensions; i++ {
		// Use Gaussian distribution for more realistic embeddings
		val := float32(rng.NormFloat64())
		vec[i] = val
		magnitude += float64(val * val)
	}

	// Normalize to unit vector
	magnitude = math.Sqrt(magnitude)
	if magnitude > 0 {
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / magnitude)
		}
	}

	return vec
}

// Batches splits vectors into batches.
func (d *Dataset) Batches(batchSize int) [][]*vectorize.Vector {
	var batches [][]*vectorize.Vector
	for i := 0; i < len(d.Vectors); i += batchSize {
		end := i + batchSize
		if end > len(d.Vectors) {
			end = len(d.Vectors)
		}
		batches = append(batches, d.Vectors[i:end])
	}
	return batches
}

// VectorIDs returns all vector IDs.
func (d *Dataset) VectorIDs() []string {
	ids := make([]string, len(d.Vectors))
	for i, v := range d.Vectors {
		ids[i] = v.ID
	}
	return ids
}

// SampleIDs returns a random sample of vector IDs.
func (d *Dataset) SampleIDs(count int, seed int64) []string {
	rng := rand.New(rand.NewSource(seed))
	if count > len(d.Vectors) {
		count = len(d.Vectors)
	}

	indices := rng.Perm(len(d.Vectors))[:count]
	ids := make([]string, count)
	for i, idx := range indices {
		ids[i] = d.Vectors[idx].ID
	}
	return ids
}

// generateUUID creates a deterministic UUID v4 format string based on seed and index.
func generateUUID(seed int64, index int) string {
	// Create a deterministic hash from seed and index
	input := fmt.Sprintf("benchmark_vector_%d_%d", seed, index)
	hash := md5.Sum([]byte(input))
	hexStr := hex.EncodeToString(hash[:])

	// Format as UUID v4 (xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx)
	// Set version (4) and variant bits
	return fmt.Sprintf("%s-%s-4%s-%s%s-%s",
		hexStr[0:8],
		hexStr[8:12],
		hexStr[13:16],
		hexStr[16:17], // variant should be 8, 9, a, or b but we'll leave it
		hexStr[17:20],
		hexStr[20:32],
	)
}
