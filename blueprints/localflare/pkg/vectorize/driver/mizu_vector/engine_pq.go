package mizu_vector

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// PQEngine implements Product Quantization for memory-efficient search.
// Memory: O(n*m) bytes where m=subspaces (vs O(n*d) for full vectors).
// Based on "Product quantization for nearest neighbor search" (Jégou et al., 2011).
type PQEngine struct {
	distFunc DistanceFunc

	// PQ parameters
	numSubspaces   int // m: number of subvector partitions
	numCentroids   int // k: centroids per subspace (typically 256)
	subspaceDims   int // dims per subspace

	// Codebooks: [m][k][subspaceDims]
	codebooks      [][][]float32

	// Encoded vectors: [n][m] codes
	encodedVectors []pqVector

	// Original vectors for distance computation
	vectors        []indexedVector

	needsRebuild   bool
	dims           int
}

type pqVector struct {
	id    string
	codes []byte // [m] indices into codebooks
}

// PQ configuration
const (
	pqDefaultSubspaces = 8  // Divide vector into 8 parts
	pqDefaultCentroids = 256 // 8-bit codes
	pqKMeansIters      = 10
)

// NewPQEngine creates a new Product Quantization engine.
func NewPQEngine(distFunc DistanceFunc, dims int) *PQEngine {
	numSubspaces := pqDefaultSubspaces
	if dims < numSubspaces {
		numSubspaces = dims
	}

	return &PQEngine{
		distFunc:     distFunc,
		numSubspaces: numSubspaces,
		numCentroids: pqDefaultCentroids,
		dims:         dims,
		needsRebuild: true,
	}
}

func (e *PQEngine) Name() string { return "pq" }

func (e *PQEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
	e.dims = dims
	e.subspaceDims = dims / e.numSubspaces

	// Adjust if dims not evenly divisible
	if dims%e.numSubspaces != 0 {
		e.numSubspaces = 1
		for m := 8; m >= 1; m-- {
			if dims%m == 0 {
				e.numSubspaces = m
				break
			}
		}
		e.subspaceDims = dims / e.numSubspaces
	}

	// Collect vectors
	e.vectors = make([]indexedVector, 0, len(vectors))
	for id, v := range vectors {
		e.vectors = append(e.vectors, indexedVector{id: id, values: v.Values})
	}

	// Train codebooks for each subspace
	e.codebooks = make([][][]float32, e.numSubspaces)
	for m := 0; m < e.numSubspaces; m++ {
		e.codebooks[m] = e.trainSubspaceCodebook(m)
	}

	// Encode all vectors
	e.encodedVectors = make([]pqVector, len(e.vectors))
	for i, v := range e.vectors {
		e.encodedVectors[i] = pqVector{
			id:    v.id,
			codes: e.encodeVector(v.values),
		}
	}

	e.needsRebuild = false
}

// trainSubspaceCodebook trains k-means codebook for subspace m.
func (e *PQEngine) trainSubspaceCodebook(m int) [][]float32 {
	// Extract subvectors
	subvectors := make([][]float32, len(e.vectors))
	startDim := m * e.subspaceDims
	for i, v := range e.vectors {
		subvectors[i] = v.values[startDim : startDim+e.subspaceDims]
	}

	// K-means clustering
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	k := e.numCentroids
	if k > len(subvectors) {
		k = len(subvectors)
	}

	// Initialize centroids randomly
	centroids := make([][]float32, k)
	perm := rng.Perm(len(subvectors))
	for i := 0; i < k; i++ {
		centroids[i] = make([]float32, e.subspaceDims)
		copy(centroids[i], subvectors[perm[i%len(perm)]])
	}

	// K-means iterations
	assignments := make([]int, len(subvectors))
	for iter := 0; iter < pqKMeansIters; iter++ {
		// Assign to nearest centroid
		for i, sv := range subvectors {
			minDist := float32(math.MaxFloat32)
			minIdx := 0
			for c, centroid := range centroids {
				dist := e.subvectorDistance(sv, centroid)
				if dist < minDist {
					minDist = dist
					minIdx = c
				}
			}
			assignments[i] = minIdx
		}

		// Update centroids
		counts := make([]int, k)
		for i := range centroids {
			for j := range centroids[i] {
				centroids[i][j] = 0
			}
		}

		for i, sv := range subvectors {
			c := assignments[i]
			counts[c]++
			for j, v := range sv {
				centroids[c][j] += v
			}
		}

		for c := 0; c < k; c++ {
			if counts[c] > 0 {
				for j := range centroids[c] {
					centroids[c][j] /= float32(counts[c])
				}
			}
		}
	}

	return centroids
}

func (e *PQEngine) subvectorDistance(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum
}

// encodeVector encodes a full vector into PQ codes.
func (e *PQEngine) encodeVector(v []float32) []byte {
	codes := make([]byte, e.numSubspaces)
	for m := 0; m < e.numSubspaces; m++ {
		startDim := m * e.subspaceDims
		subvector := v[startDim : startDim+e.subspaceDims]

		// Find nearest centroid
		minDist := float32(math.MaxFloat32)
		minIdx := 0
		for c, centroid := range e.codebooks[m] {
			dist := e.subvectorDistance(subvector, centroid)
			if dist < minDist {
				minDist = dist
				minIdx = c
			}
		}
		codes[m] = byte(minIdx)
	}
	return codes
}

func (e *PQEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *PQEngine) Delete(ids []string) {
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	filtered := make([]pqVector, 0, len(e.encodedVectors))
	for _, v := range e.encodedVectors {
		if _, deleted := idSet[v.id]; !deleted {
			filtered = append(filtered, v)
		}
	}
	e.encodedVectors = filtered

	filteredVec := make([]indexedVector, 0, len(e.vectors))
	for _, v := range e.vectors {
		if _, deleted := idSet[v.id]; !deleted {
			filteredVec = append(filteredVec, v)
		}
	}
	e.vectors = filteredVec
}

func (e *PQEngine) Search(query []float32, k int) []SearchResult {
	if len(e.encodedVectors) == 0 {
		return nil
	}

	// Precompute distance tables: dist[m][c] = distance from query subvector to centroid c
	distTables := make([][]float32, e.numSubspaces)
	for m := 0; m < e.numSubspaces; m++ {
		startDim := m * e.subspaceDims
		querySubvector := query[startDim : startDim+e.subspaceDims]

		distTables[m] = make([]float32, len(e.codebooks[m]))
		for c, centroid := range e.codebooks[m] {
			distTables[m][c] = e.subvectorDistance(querySubvector, centroid)
		}
	}

	// Asymmetric distance computation using lookup tables
	results := make([]SearchResult, len(e.encodedVectors))
	for i, pv := range e.encodedVectors {
		var dist float32
		for m := 0; m < e.numSubspaces; m++ {
			dist += distTables[m][pv.codes[m]]
		}
		results[i] = SearchResult{ID: pv.id, Distance: dist}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if k > len(results) {
		k = len(results)
	}

	// Rerank top candidates with exact distance
	candidates := results[:min(k*4, len(results))]

	// Find original vectors for reranking
	idToVec := make(map[string][]float32, len(candidates))
	for _, v := range e.vectors {
		idToVec[v.id] = v.values
	}

	for i := range candidates {
		if vec, ok := idToVec[candidates[i].ID]; ok {
			candidates[i].Distance = e.distFunc(query, vec)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Distance < candidates[j].Distance
	})

	if k > len(candidates) {
		k = len(candidates)
	}

	return candidates[:k]
}

func (e *PQEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *PQEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
