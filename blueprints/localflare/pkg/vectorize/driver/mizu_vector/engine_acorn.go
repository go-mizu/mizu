package mizu_vector

import (
	"math/rand"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// ACORNEngine implements a simplified graph-based search.
// Uses flat single-level graph for simplicity and speed.
//
// Optimized with:
// - Index-based storage (no string lookups)
// - Medoid-based entry point (not random)
// - Typed heaps instead of sorting
// - Bitset for visited tracking
type ACORNEngine struct {
	distFunc DistanceFunc

	// Index-based storage
	vectors []acornVector

	// Graph structure - simple k-NN graph
	graph   [][]int32
	navNode int32 // Medoid for entry point

	// Parameters
	K        int // Neighbors per node
	efSearch int // Search beam width

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

type acornVector struct {
	id     string
	values []float32
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

	// Collect vectors with index-based storage
	e.vectors = make([]acornVector, 0, n)
	for id, v := range vectors {
		e.vectors = append(e.vectors, acornVector{id: id, values: v.Values})
	}

	// Find medoid for entry point
	e.navNode = e.findMedoid(dims)

	// Build k-NN graph
	e.buildKNNGraph()

	e.needsRebuild = false
}

// findMedoid finds the vector closest to centroid.
func (e *ACORNEngine) findMedoid(dims int) int32 {
	n := len(e.vectors)

	// Compute centroid
	centroid := make([]float32, dims)
	for i := 0; i < n; i++ {
		for j := 0; j < dims; j++ {
			centroid[j] += e.vectors[i].values[j]
		}
	}
	invN := 1.0 / float32(n)
	for j := 0; j < dims; j++ {
		centroid[j] *= invN
	}

	// Find vector closest to centroid
	bestIdx := int32(0)
	bestDist := e.distFunc(centroid, e.vectors[0].values)
	for i := 1; i < n; i++ {
		d := e.distFunc(centroid, e.vectors[i].values)
		if d < bestDist {
			bestDist = d
			bestIdx = int32(i)
		}
	}

	return bestIdx
}

// buildKNNGraph builds a k-NN graph using heap-based selection.
func (e *ACORNEngine) buildKNNGraph() {
	n := len(e.vectors)
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
			dist := e.distFunc(e.vectors[i].values, e.vectors[j].values)
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

	n := len(e.vectors)
	if n == 0 {
		return nil
	}

	visited := newBitset(n)
	ef := e.efSearch
	if ef < k*2 {
		ef = k * 2
	}

	// Use heaps for efficient search
	candidates := make(minHeap32, 0, ef*2)
	result := make(maxHeap32, 0, ef)

	// Start from medoid
	startDist := e.distFunc(query, e.vectors[e.navNode].values)
	candidates.PushItem(distItem32{idx: e.navNode, dist: startDist})
	result.PushItem(distItem32{idx: e.navNode, dist: startDist})
	visited.Set(e.navNode)

	// Beam search
	for len(candidates) > 0 {
		curr := candidates.PopItem()

		// Stop if current candidate is worse than worst in result
		if len(result) >= ef && curr.dist > result[0].dist {
			break
		}

		// Explore neighbors
		for _, neighbor := range e.graph[curr.idx] {
			if visited.Test(neighbor) {
				continue
			}
			visited.Set(neighbor)

			dist := e.distFunc(query, e.vectors[neighbor].values)

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
			ID:       e.vectors[allResults[i].idx].id,
			Distance: allResults[i].dist,
		}
	}

	return results
}

func (e *ACORNEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *ACORNEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
