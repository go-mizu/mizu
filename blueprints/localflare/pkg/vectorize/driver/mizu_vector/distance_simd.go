package mizu_vector

import (
	"math"

	"github.com/viterin/vek/vek32"
)

// SIMD-optimized distance functions using viterin/vek.
// These provide 4-8x speedup over manual loop unrolling on AVX2-capable CPUs.

// DotProductSIMD computes dot product using SIMD instructions.
func DotProductSIMD(a, b []float32) float32 {
	return vek32.Dot(a, b)
}

// NegDotProductSIMD returns negative dot product using SIMD.
func NegDotProductSIMD(a, b []float32) float32 {
	return -vek32.Dot(a, b)
}

// EuclideanDistanceSIMD computes squared L2 distance using SIMD.
func EuclideanDistanceSIMD(a, b []float32) float32 {
	return vek32.Distance(a, b)
}

// CosineDistanceSIMD computes 1 - cosine_similarity using SIMD.
func CosineDistanceSIMD(a, b []float32) float32 {
	dot := vek32.Dot(a, b)
	normA := vek32.Norm(a)
	normB := vek32.Norm(b)

	if normA == 0 || normB == 0 {
		return 1.0
	}

	return 1.0 - dot/(normA*normB)
}

// L2NormSIMD computes L2 norm using SIMD.
func L2NormSIMD(v []float32) float32 {
	return vek32.Norm(v)
}

// NormalizeSIMD normalizes a vector in-place using SIMD.
func NormalizeSIMD(v []float32) {
	norm := vek32.Norm(v)
	if norm == 0 {
		return
	}
	invNorm := 1.0 / norm
	for i := range v {
		v[i] *= invNorm
	}
}

// BatchDotProduct computes dot products of query against multiple vectors.
// Optimized for cache locality by processing vectors sequentially.
func BatchDotProduct(query []float32, vectors [][]float32, results []float32) {
	for i, v := range vectors {
		results[i] = vek32.Dot(query, v)
	}
}

// BatchCosineDistance computes cosine distances of query against multiple vectors.
func BatchCosineDistance(query []float32, queryNorm float32, vectors [][]float32, norms []float32, results []float32) {
	for i, v := range vectors {
		if queryNorm == 0 || norms[i] == 0 {
			results[i] = 1.0
			continue
		}
		dot := vek32.Dot(query, v)
		results[i] = 1.0 - dot/(queryNorm*norms[i])
	}
}

// SoAVectorStore provides Structure of Arrays storage for cache-efficient access.
// Vectors are stored contiguously: [v0d0, v0d1, ..., v1d0, v1d1, ...]
type SoAVectorStore struct {
	ids    []string
	data   []float32 // Contiguous vector data
	norms  []float32 // Precomputed L2 norms
	dims   int
	idToIdx map[string]int32
}

// NewSoAVectorStore creates a new Structure of Arrays vector store.
func NewSoAVectorStore(dims int) *SoAVectorStore {
	return &SoAVectorStore{
		ids:     make([]string, 0),
		data:    make([]float32, 0),
		norms:   make([]float32, 0),
		dims:    dims,
		idToIdx: make(map[string]int32),
	}
}

// Add adds a vector to the store.
func (s *SoAVectorStore) Add(id string, values []float32) {
	idx := int32(len(s.ids))
	s.idToIdx[id] = idx
	s.ids = append(s.ids, id)
	s.data = append(s.data, values...)
	s.norms = append(s.norms, vek32.Norm(values))
}

// Get retrieves a vector by index.
func (s *SoAVectorStore) Get(idx int) []float32 {
	start := idx * s.dims
	return s.data[start : start+s.dims]
}

// GetByID retrieves a vector by ID.
func (s *SoAVectorStore) GetByID(id string) ([]float32, bool) {
	idx, ok := s.idToIdx[id]
	if !ok {
		return nil, false
	}
	return s.Get(int(idx)), true
}

// Len returns the number of vectors.
func (s *SoAVectorStore) Len() int {
	return len(s.ids)
}

// ComputeDistances computes distances from query to all vectors using SIMD.
func (s *SoAVectorStore) ComputeDistances(query []float32, distFunc DistanceFunc) []float32 {
	n := s.Len()
	results := make([]float32, n)
	for i := 0; i < n; i++ {
		results[i] = distFunc(query, s.Get(i))
	}
	return results
}

// ComputeCosineDistances computes cosine distances efficiently using precomputed norms.
func (s *SoAVectorStore) ComputeCosineDistances(query []float32) []float32 {
	n := s.Len()
	results := make([]float32, n)
	queryNorm := vek32.Norm(query)

	if queryNorm == 0 {
		for i := 0; i < n; i++ {
			results[i] = 1.0
		}
		return results
	}

	for i := 0; i < n; i++ {
		if s.norms[i] == 0 {
			results[i] = 1.0
			continue
		}
		dot := vek32.Dot(query, s.Get(i))
		results[i] = 1.0 - dot/(queryNorm*s.norms[i])
	}
	return results
}

// CompressedGraph stores graph edges using int32 indices instead of string IDs.
// This reduces memory usage and improves cache locality.
type CompressedGraph struct {
	idToIdx map[string]int32
	idxToID []string
	edges   [][]int32
}

// NewCompressedGraph creates a new compressed graph.
func NewCompressedGraph(capacity int) *CompressedGraph {
	return &CompressedGraph{
		idToIdx: make(map[string]int32, capacity),
		idxToID: make([]string, 0, capacity),
		edges:   make([][]int32, 0, capacity),
	}
}

// AddNode adds a node to the graph and returns its index.
func (g *CompressedGraph) AddNode(id string) int32 {
	if idx, ok := g.idToIdx[id]; ok {
		return idx
	}
	idx := int32(len(g.idxToID))
	g.idToIdx[id] = idx
	g.idxToID = append(g.idxToID, id)
	g.edges = append(g.edges, nil)
	return idx
}

// SetEdges sets the edges for a node.
func (g *CompressedGraph) SetEdges(idx int32, neighbors []int32) {
	g.edges[idx] = neighbors
}

// GetEdges returns the edges for a node.
func (g *CompressedGraph) GetEdges(idx int32) []int32 {
	if int(idx) >= len(g.edges) {
		return nil
	}
	return g.edges[idx]
}

// GetID returns the ID for an index.
func (g *CompressedGraph) GetID(idx int32) string {
	if int(idx) >= len(g.idxToID) {
		return ""
	}
	return g.idxToID[idx]
}

// GetIdx returns the index for an ID.
func (g *CompressedGraph) GetIdx(id string) (int32, bool) {
	idx, ok := g.idToIdx[id]
	return idx, ok
}

// Len returns the number of nodes.
func (g *CompressedGraph) Len() int {
	return len(g.idxToID)
}

// PrefetchVector prefetches a vector's memory into cache.
// This is a hint to the CPU and may improve performance for sequential access.
func PrefetchVector(data []float32) {
	// Go doesn't have direct prefetch intrinsics, but accessing the first
	// element can help bring the cache line into L1
	if len(data) > 0 {
		_ = data[0]
	}
}

// MinHeap for top-k selection using indices.
type IndexedMinHeap struct {
	indices   []int32
	distances []float32
	k         int
}

// NewIndexedMinHeap creates a min-heap for top-k selection.
func NewIndexedMinHeap(k int) *IndexedMinHeap {
	return &IndexedMinHeap{
		indices:   make([]int32, 0, k+1),
		distances: make([]float32, 0, k+1),
		k:         k,
	}
}

// Push adds an item to the heap.
func (h *IndexedMinHeap) Push(idx int32, dist float32) {
	if len(h.indices) < h.k {
		h.indices = append(h.indices, idx)
		h.distances = append(h.distances, dist)
		h.siftUp(len(h.indices) - 1)
	} else if dist < h.distances[0] {
		h.indices[0] = idx
		h.distances[0] = dist
		h.siftDown(0)
	}
}

// MaxDist returns the maximum distance in the heap.
func (h *IndexedMinHeap) MaxDist() float32 {
	if len(h.distances) == 0 {
		return math.MaxFloat32
	}
	return h.distances[0]
}

// Results returns the indices and distances sorted by distance.
func (h *IndexedMinHeap) Results() ([]int32, []float32) {
	// Sort by distance (heap to sorted array)
	n := len(h.indices)
	indices := make([]int32, n)
	distances := make([]float32, n)
	copy(indices, h.indices)
	copy(distances, h.distances)

	// Simple insertion sort for small k
	for i := 1; i < n; i++ {
		j := i
		for j > 0 && distances[j] < distances[j-1] {
			distances[j], distances[j-1] = distances[j-1], distances[j]
			indices[j], indices[j-1] = indices[j-1], indices[j]
			j--
		}
	}
	return indices, distances
}

func (h *IndexedMinHeap) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h.distances[i] > h.distances[parent] {
			h.distances[i], h.distances[parent] = h.distances[parent], h.distances[i]
			h.indices[i], h.indices[parent] = h.indices[parent], h.indices[i]
			i = parent
		} else {
			break
		}
	}
}

func (h *IndexedMinHeap) siftDown(i int) {
	n := len(h.indices)
	for {
		largest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && h.distances[left] > h.distances[largest] {
			largest = left
		}
		if right < n && h.distances[right] > h.distances[largest] {
			largest = right
		}

		if largest == i {
			break
		}

		h.distances[i], h.distances[largest] = h.distances[largest], h.distances[i]
		h.indices[i], h.indices[largest] = h.indices[largest], h.indices[i]
		i = largest
	}
}
