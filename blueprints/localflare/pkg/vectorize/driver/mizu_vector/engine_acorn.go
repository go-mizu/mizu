package mizu_vector

import (
	"math/rand"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/viterin/vek/vek32"
)

// ACORNEngine implements a simplified graph-based search.
// Uses flat single-level graph for simplicity and speed.
//
// Optimized with:
// - SIMD-accelerated distance computation using viterin/vek
// - Contiguous memory layout for cache efficiency
// - Precomputed L2 norms for faster cosine distance
// - Object pooling for bitsets (zero-allocation search)
// - Medoid-based entry point (not random)
// - Typed heaps instead of sorting
// - Bitset for visited tracking
type ACORNEngine struct {
	distFunc DistanceFunc
	metric   vectorize.DistanceMetric

	// Contiguous memory layout for cache efficiency
	vectorData  []float32 // All vector values: [v0d0, v0d1, ..., v1d0, ...]
	vectorIDs   []string  // Vector IDs indexed by int32
	vectorNorms []float32 // Precomputed L2 norms for cosine distance
	dims        int

	// Graph structure - simple k-NN graph
	graph   [][]int32
	navNode int32 // Medoid for entry point

	// Parameters
	K        int // Neighbors per node
	efSearch int // Search beam width

	// Object pools for zero-allocation search
	bitsetPool sync.Pool // *bitset for visited tracking

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

const (
	acornDefaultK        = 32
	acornDefaultEfSearch = 64
)

// NewACORNEngine creates a new ACORN search engine.
func NewACORNEngine(distFunc DistanceFunc) *ACORNEngine {
	return &ACORNEngine{
		distFunc:     distFunc,
		K:            acornDefaultK,
		efSearch:     acornDefaultEfSearch,
		needsRebuild: true,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *ACORNEngine) Name() string { return "acorn" }

func (e *ACORNEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
	e.mu.Lock()
	defer e.mu.Unlock()

	n := len(vectors)
	if n == 0 {
		e.needsRebuild = false
		return
	}

	e.dims = dims
	e.metric = metric

	// Build contiguous storage for cache efficiency
	e.vectorData = make([]float32, 0, n*dims)
	e.vectorIDs = make([]string, 0, n)
	e.vectorNorms = make([]float32, 0, n)

	for id, v := range vectors {
		e.vectorIDs = append(e.vectorIDs, id)
		e.vectorData = append(e.vectorData, v.Values...)
		e.vectorNorms = append(e.vectorNorms, vek32.Norm(v.Values))
	}

	// Find medoid for entry point
	e.navNode = e.findMedoid(dims)

	// Build k-NN graph
	e.buildKNNGraph()

	e.needsRebuild = false
}

// getVector returns vector data at index using contiguous storage
func (e *ACORNEngine) getVector(idx int32) []float32 {
	start := int(idx) * e.dims
	return e.vectorData[start : start+e.dims]
}

// computeDistanceSIMD computes distance based on metric using SIMD
func (e *ACORNEngine) computeDistanceSIMD(aIdx int32, b []float32, bNorm float32) float32 {
	a := e.getVector(aIdx)
	aNorm := e.vectorNorms[aIdx]

	switch e.metric {
	case vectorize.Cosine:
		if aNorm == 0 || bNorm == 0 {
			return 1.0
		}
		dot := vek32.Dot(a, b)
		return 1.0 - dot/(aNorm*bNorm)
	case vectorize.Euclidean:
		return vek32.Distance(a, b)
	case vectorize.DotProduct:
		return -vek32.Dot(a, b)
	default:
		if aNorm == 0 || bNorm == 0 {
			return 1.0
		}
		dot := vek32.Dot(a, b)
		return 1.0 - dot/(aNorm*bNorm)
	}
}

// computeDistanceBetween computes distance between two indexed vectors
func (e *ACORNEngine) computeDistanceBetween(aIdx, bIdx int32) float32 {
	a := e.getVector(aIdx)
	b := e.getVector(bIdx)
	aNorm := e.vectorNorms[aIdx]
	bNorm := e.vectorNorms[bIdx]

	switch e.metric {
	case vectorize.Cosine:
		if aNorm == 0 || bNorm == 0 {
			return 1.0
		}
		dot := vek32.Dot(a, b)
		return 1.0 - dot/(aNorm*bNorm)
	case vectorize.Euclidean:
		return vek32.Distance(a, b)
	case vectorize.DotProduct:
		return -vek32.Dot(a, b)
	default:
		if aNorm == 0 || bNorm == 0 {
			return 1.0
		}
		dot := vek32.Dot(a, b)
		return 1.0 - dot/(aNorm*bNorm)
	}
}

// findMedoid finds the vector closest to centroid.
func (e *ACORNEngine) findMedoid(dims int) int32 {
	n := len(e.vectorIDs)

	// Compute centroid
	centroid := make([]float32, dims)
	for i := 0; i < n; i++ {
		vec := e.getVector(int32(i))
		for j := 0; j < dims; j++ {
			centroid[j] += vec[j]
		}
	}
	invN := 1.0 / float32(n)
	for j := 0; j < dims; j++ {
		centroid[j] *= invN
	}
	centroidNorm := vek32.Norm(centroid)

	// Find vector closest to centroid
	bestIdx := int32(0)
	bestDist := e.computeDistanceSIMD(0, centroid, centroidNorm)
	for i := 1; i < n; i++ {
		d := e.computeDistanceSIMD(int32(i), centroid, centroidNorm)
		if d < bestDist {
			bestDist = d
			bestIdx = int32(i)
		}
	}

	return bestIdx
}

// buildKNNGraph builds a k-NN graph using heap-based selection.
func (e *ACORNEngine) buildKNNGraph() {
	n := len(e.vectorIDs)
	k := e.K
	if k > n-1 {
		k = n - 1
	}

	e.graph = make([][]int32, n)

	// Build k-NN for each vector
	for i := 0; i < n; i++ {
		// Use heap for top-k selection
		h := make(maxHeap32, 0, k)

		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			dist := e.computeDistanceBetween(int32(i), int32(j))
			if len(h) < k {
				h.PushItem(distItem32{idx: int32(j), dist: dist})
			} else if dist < h[0].dist {
				h.PopItem()
				h.PushItem(distItem32{idx: int32(j), dist: dist})
			}
		}

		// Extract neighbors
		e.graph[i] = make([]int32, len(h))
		for idx := len(h) - 1; idx >= 0; idx-- {
			e.graph[i][idx] = h.PopItem().idx
		}
	}

	// Make graph bidirectional
	for i := 0; i < n; i++ {
		for _, neighbor := range e.graph[i] {
			// Check if reverse edge exists
			found := false
			for _, existing := range e.graph[neighbor] {
				if existing == int32(i) {
					found = true
					break
				}
			}
			if !found && len(e.graph[neighbor]) < k*2 {
				e.graph[neighbor] = append(e.graph[neighbor], int32(i))
			}
		}
	}
}

func (e *ACORNEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *ACORNEngine) Delete(ids []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.needsRebuild = true
}

func (e *ACORNEngine) Search(query []float32, k int) []SearchResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	n := len(e.vectorIDs)
	if n == 0 {
		return nil
	}

	queryNorm := vek32.Norm(query)

	// Get pooled bitset or create new one
	var visited *bitset
	if pooled := e.bitsetPool.Get(); pooled != nil {
		visited = pooled.(*bitset)
		visited.Clear()
		if visited.Size() < n {
			visited = newBitset(n)
		}
	} else {
		visited = newBitset(n)
	}
	defer e.bitsetPool.Put(visited)

	ef := e.efSearch
	if ef < k*2 {
		ef = k * 2
	}

	// Use heaps for efficient search
	candidates := make(minHeap32, 0, ef*2)
	result := make(maxHeap32, 0, ef)

	// Start from medoid
	startDist := e.computeDistanceSIMD(e.navNode, query, queryNorm)
	candidates.PushItem(distItem32{idx: e.navNode, dist: startDist})
	result.PushItem(distItem32{idx: e.navNode, dist: startDist})
	visited.Set(e.navNode)

	// Beam search with prefetching
	for len(candidates) > 0 {
		curr := candidates.PopItem()

		// Stop if current candidate is worse than worst in result
		if len(result) >= ef && curr.dist > result[0].dist {
			break
		}

		// Get neighbors and prefetch
		neighbors := e.graph[curr.idx]
		if len(neighbors) > 0 {
			_ = e.vectorData[int(neighbors[0])*e.dims]
		}

		// Explore neighbors
		for _, neighbor := range neighbors {
			if visited.Test(neighbor) {
				continue
			}
			visited.Set(neighbor)

			dist := e.computeDistanceSIMD(neighbor, query, queryNorm)

			if len(result) < ef {
				candidates.PushItem(distItem32{idx: neighbor, dist: dist})
				result.PushItem(distItem32{idx: neighbor, dist: dist})
			} else if dist < result[0].dist {
				candidates.PushItem(distItem32{idx: neighbor, dist: dist})
				result.PopItem()
				result.PushItem(distItem32{idx: neighbor, dist: dist})
			}
		}
	}

	// Extract top k results
	if k > len(result) {
		k = len(result)
	}

	// Get all results sorted
	allResults := make([]distItem32, len(result))
	for i := len(result) - 1; i >= 0; i-- {
		allResults[i] = result.PopItem()
	}

	results := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		results[i] = SearchResult{
			ID:       e.vectorIDs[allResults[i].idx],
			Distance: allResults[i].dist,
		}
	}

	return results
}

func (e *ACORNEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *ACORNEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
