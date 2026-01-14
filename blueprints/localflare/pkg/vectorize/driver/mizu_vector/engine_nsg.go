package mizu_vector

import (
	"math/rand"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/viterin/vek/vek32"
)

// NSGEngine implements Navigating Spreading-out Graph.
// Based on "Fast Approximate Nearest Neighbor Search With The Navigating Spreading-out Graph" (VLDB 2019).
//
// Optimized with:
// - SIMD-accelerated distance computation using viterin/vek
// - Contiguous memory layout for cache efficiency
// - Precomputed L2 norms for faster cosine distance
// - Object pooling for bitsets (zero-allocation search)
// - Typed heaps instead of sorting
// - Bitset for visited tracking
type NSGEngine struct {
	distFunc DistanceFunc
	metric   vectorize.DistanceMetric

	// Contiguous memory layout for cache efficiency
	vectorData  []float32 // All vector values: [v0d0, v0d1, ..., v1d0, ...]
	vectorIDs   []string  // Vector IDs indexed by int32
	vectorNorms []float32 // Precomputed L2 norms for cosine distance
	dims        int

	// Graph structure
	graph   [][]int32 // Adjacency list using indices
	navNode int32     // Navigating node (medoid)

	// NSG parameters
	R int // Max out-degree (default: 32)
	L int // Search list size (default: 50)

	// Object pools for zero-allocation search
	bitsetPool sync.Pool // *bitset for visited tracking

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

// NSG configuration
const (
	nsgDefaultR = 32
	nsgDefaultL = 64 // Increased for better recall
)

// NewNSGEngine creates a new NSG search engine.
func NewNSGEngine(distFunc DistanceFunc) *NSGEngine {
	return &NSGEngine{
		distFunc:     distFunc,
		R:            nsgDefaultR,
		L:            nsgDefaultL,
		needsRebuild: true,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *NSGEngine) Name() string { return "nsg" }

func (e *NSGEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
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

	// Find navigating node (closest to centroid)
	e.navNode = e.findNavigatingNode(dims)

	// Build graph using greedy construction
	e.buildGraph()

	e.needsRebuild = false
}

// getVector returns vector data at index using contiguous storage
func (e *NSGEngine) getVector(idx int32) []float32 {
	start := int(idx) * e.dims
	return e.vectorData[start : start+e.dims]
}

// computeDistanceSIMD computes distance based on metric using SIMD
func (e *NSGEngine) computeDistanceSIMD(aIdx int32, b []float32, bNorm float32) float32 {
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
func (e *NSGEngine) computeDistanceBetween(aIdx, bIdx int32) float32 {
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

// findNavigatingNode finds the medoid (vector closest to centroid).
func (e *NSGEngine) findNavigatingNode(dims int) int32 {
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

// buildGraph builds the NSG graph using incremental insertion.
func (e *NSGEngine) buildGraph() {
	n := len(e.vectorIDs)
	e.graph = make([][]int32, n)

	// Initialize all nodes with empty neighbor lists
	for i := 0; i < n; i++ {
		e.graph[i] = make([]int32, 0, e.R)
	}

	// Insert each vector incrementally
	for i := 0; i < n; i++ {
		if int32(i) == e.navNode {
			continue
		}

		// Search for nearest neighbors starting from navNode
		neighbors := e.searchForNeighbors(int32(i))

		// Select neighbors using MRNG-like pruning
		selected := e.selectNeighbors(int32(i), neighbors)
		e.graph[i] = selected

		// Add reverse edges
		for _, neighbor := range selected {
			e.graph[neighbor] = append(e.graph[neighbor], int32(i))
			// Prune if too many
			if len(e.graph[neighbor]) > e.R*2 {
				e.graph[neighbor] = e.selectNeighbors(neighbor, e.graph[neighbor])
			}
		}
	}
}

// searchForNeighbors finds candidate neighbors for a node using heap-based search.
func (e *NSGEngine) searchForNeighbors(queryIdx int32) []int32 {
	query := e.getVector(queryIdx)
	queryNorm := e.vectorNorms[queryIdx]
	n := len(e.vectorIDs)

	visited := newBitset(n)
	visited.Set(queryIdx)

	// Use heaps for efficient search
	candidates := make(minHeap32, 0, e.L*2)
	result := make(maxHeap32, 0, e.L*2)

	// Start from navigating node
	startDist := e.computeDistanceSIMD(e.navNode, query, queryNorm)
	candidates.PushItem(distItem32{idx: e.navNode, dist: startDist})
	result.PushItem(distItem32{idx: e.navNode, dist: startDist})
	visited.Set(e.navNode)

	// Beam search
	for len(candidates) > 0 {
		curr := candidates.PopItem()

		// Stop if current candidate is worse than worst in result
		if len(result) >= e.L*2 && curr.dist > result[0].dist {
			break
		}

		// Explore neighbors
		for _, neighbor := range e.graph[curr.idx] {
			if visited.Test(neighbor) {
				continue
			}
			visited.Set(neighbor)

			dist := e.computeDistanceSIMD(neighbor, query, queryNorm)

			if len(result) < e.L*2 {
				candidates.PushItem(distItem32{idx: neighbor, dist: dist})
				result.PushItem(distItem32{idx: neighbor, dist: dist})
			} else if dist < result[0].dist {
				candidates.PushItem(distItem32{idx: neighbor, dist: dist})
				result.PopItem()
				result.PushItem(distItem32{idx: neighbor, dist: dist})
			}
		}
	}

	// Extract result indices
	results := make([]int32, len(result))
	for i := len(result) - 1; i >= 0; i-- {
		results[i] = result.PopItem().idx
	}

	return results
}

// selectNeighbors selects R neighbors using MRNG-style pruning.
func (e *NSGEngine) selectNeighbors(queryIdx int32, candidates []int32) []int32 {
	if len(candidates) == 0 {
		return nil
	}

	// Build sorted candidate list using heap
	h := make(minHeap32, 0, len(candidates))
	for _, c := range candidates {
		if c != queryIdx {
			h.PushItem(distItem32{idx: c, dist: e.computeDistanceBetween(queryIdx, c)})
		}
	}

	// Select using MRNG criteria (simplified)
	selected := make([]int32, 0, e.R)
	for len(h) > 0 && len(selected) < e.R {
		cand := h.PopItem()

		// Check if candidate is occluded by any selected neighbor
		occluded := false

		for _, s := range selected {
			distCandSel := e.computeDistanceBetween(cand.idx, s)
			if distCandSel < cand.dist {
				occluded = true
				break
			}
		}

		if !occluded {
			selected = append(selected, cand.idx)
		}
	}

	return selected
}

func (e *NSGEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *NSGEngine) Delete(ids []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.needsRebuild = true
}

func (e *NSGEngine) Search(query []float32, k int) []SearchResult {
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

	L := e.L
	if L < k*2 {
		L = k * 2
	}

	// Use heaps for efficient search
	candidates := make(minHeap32, 0, L*2)
	result := make(maxHeap32, 0, L)

	// Start from navigating node
	startDist := e.computeDistanceSIMD(e.navNode, query, queryNorm)
	candidates.PushItem(distItem32{idx: e.navNode, dist: startDist})
	result.PushItem(distItem32{idx: e.navNode, dist: startDist})
	visited.Set(e.navNode)

	// Beam search with prefetching
	for len(candidates) > 0 {
		curr := candidates.PopItem()

		// Stop if current candidate is worse than worst in result
		if len(result) >= L && curr.dist > result[0].dist {
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

			if len(result) < L {
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

func (e *NSGEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *NSGEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
