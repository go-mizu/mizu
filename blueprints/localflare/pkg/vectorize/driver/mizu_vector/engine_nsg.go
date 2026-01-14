package mizu_vector

import (
	"math/rand"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// NSGEngine implements Navigating Spreading-out Graph.
// Based on "Fast Approximate Nearest Neighbor Search With The Navigating Spreading-out Graph" (VLDB 2019).
//
// Optimized with:
// - Index-based storage (no string lookups)
// - Typed heaps instead of sorting
// - Bitset for visited tracking
type NSGEngine struct {
	distFunc DistanceFunc

	// Index-based storage
	vectors []nsgVector

	// Graph structure
	graph   [][]int32 // Adjacency list using indices
	navNode int32     // Navigating node (medoid)

	// NSG parameters
	R int // Max out-degree (default: 32)
	L int // Search list size (default: 50)

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

type nsgVector struct {
	id     string
	values []float32
}

// NSG configuration
const (
	nsgDefaultR = 32
	nsgDefaultL = 50
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

	// Collect vectors with index-based storage
	e.vectors = make([]nsgVector, 0, n)
	for id, v := range vectors {
		e.vectors = append(e.vectors, nsgVector{id: id, values: v.Values})
	}

	// Find navigating node (closest to centroid)
	e.navNode = e.findNavigatingNode(dims)

	// Build graph using greedy construction
	e.buildGraph()

	e.needsRebuild = false
}

// findNavigatingNode finds the medoid (vector closest to centroid).
func (e *NSGEngine) findNavigatingNode(dims int) int32 {
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

// buildGraph builds the NSG graph using incremental insertion.
func (e *NSGEngine) buildGraph() {
	n := len(e.vectors)
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
		neighbors := e.searchForNeighbors(i)

		// Select neighbors using MRNG-like pruning
		selected := e.selectNeighbors(i, neighbors)
		e.graph[i] = selected

		// Add reverse edges
		for _, neighbor := range selected {
			e.graph[neighbor] = append(e.graph[neighbor], int32(i))
			// Prune if too many
			if len(e.graph[neighbor]) > e.R*2 {
				e.graph[neighbor] = e.selectNeighbors(int(neighbor), e.graph[neighbor])
			}
		}
	}
}

// searchForNeighbors finds candidate neighbors for a node using heap-based search.
func (e *NSGEngine) searchForNeighbors(queryIdx int) []int32 {
	query := e.vectors[queryIdx].values
	n := len(e.vectors)

	visited := newBitset(n)
	visited.Set(int32(queryIdx))

	// Use heaps for efficient search
	candidates := make(minHeap32, 0, e.L*2)
	result := make(maxHeap32, 0, e.L*2)

	// Start from navigating node
	startDist := e.distFunc(query, e.vectors[e.navNode].values)
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

			dist := e.distFunc(query, e.vectors[neighbor].values)

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
func (e *NSGEngine) selectNeighbors(queryIdx int, candidates []int32) []int32 {
	if len(candidates) == 0 {
		return nil
	}

	query := e.vectors[queryIdx].values

	// Build sorted candidate list using heap
	h := make(minHeap32, 0, len(candidates))
	for _, c := range candidates {
		if c != int32(queryIdx) {
			h.PushItem(distItem32{idx: c, dist: e.distFunc(query, e.vectors[c].values)})
		}
	}

	// Select using MRNG criteria (simplified)
	selected := make([]int32, 0, e.R)
	for len(h) > 0 && len(selected) < e.R {
		cand := h.PopItem()

		// Check if candidate is occluded by any selected neighbor
		occluded := false
		candVec := e.vectors[cand.idx].values

		for _, s := range selected {
			selVec := e.vectors[s].values
			distCandSel := e.distFunc(candVec, selVec)

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

	n := len(e.vectors)
	if n == 0 {
		return nil
	}

	visited := newBitset(n)
	L := e.L
	if L < k*2 {
		L = k * 2
	}

	// Use heaps for efficient search
	candidates := make(minHeap32, 0, L*2)
	result := make(maxHeap32, 0, L)

	// Start from navigating node
	startDist := e.distFunc(query, e.vectors[e.navNode].values)
	candidates.PushItem(distItem32{idx: e.navNode, dist: startDist})
	result.PushItem(distItem32{idx: e.navNode, dist: startDist})
	visited.Set(e.navNode)

	// Beam search
	for len(candidates) > 0 {
		curr := candidates.PopItem()

		// Stop if current candidate is worse than worst in result
		if len(result) >= L && curr.dist > result[0].dist {
			break
		}

		// Explore neighbors
		for _, neighbor := range e.graph[curr.idx] {
			if visited.Test(neighbor) {
				continue
			}
			visited.Set(neighbor)

			dist := e.distFunc(query, e.vectors[neighbor].values)

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
			ID:       e.vectors[allResults[i].idx].id,
			Distance: allResults[i].dist,
		}
	}

	return results
}

func (e *NSGEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *NSGEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
