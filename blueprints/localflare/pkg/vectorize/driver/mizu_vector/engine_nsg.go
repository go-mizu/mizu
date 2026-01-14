package mizu_vector

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// NSGEngine implements Navigating Spreading-out Graph.
// Based on "Fast Approximate Nearest Neighbor Search With The Navigating Spreading-out Graph" (VLDB 2019).
// Uses simplified construction for better build performance.
type NSGEngine struct {
	distFunc DistanceFunc

	// Vector storage
	vectors []indexedVector

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

	// Collect vectors
	e.vectors = make([]indexedVector, 0, n)
	for id, v := range vectors {
		e.vectors = append(e.vectors, indexedVector{id: id, values: v.Values})
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

	// Initialize with navigating node having no neighbors
	e.graph[e.navNode] = make([]int32, 0, e.R)

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

// searchForNeighbors finds candidate neighbors for a node.
func (e *NSGEngine) searchForNeighbors(queryIdx int) []int32 {
	query := e.vectors[queryIdx].values
	n := len(e.vectors)

	visited := make([]bool, n)
	visited[queryIdx] = true

	// Start from navigating node
	candidates := make([]int32, 0, e.L*2)
	candidates = append(candidates, e.navNode)
	visited[e.navNode] = true

	// Greedy expansion
	for len(candidates) < e.L*2 {
		improved := false

		// Try to expand from best unvisited candidates
		for _, candIdx := range candidates {
			for _, neighbor := range e.graph[candIdx] {
				if visited[neighbor] {
					continue
				}
				visited[neighbor] = true
				candidates = append(candidates, neighbor)
				improved = true
			}
		}

		if !improved {
			break
		}

		// Sort by distance and keep top L*2
		sort.Slice(candidates, func(i, j int) bool {
			return e.distFunc(query, e.vectors[candidates[i]].values) <
				e.distFunc(query, e.vectors[candidates[j]].values)
		})
		if len(candidates) > e.L*2 {
			candidates = candidates[:e.L*2]
		}
	}

	return candidates
}

// selectNeighbors selects R neighbors using MRNG-style pruning.
func (e *NSGEngine) selectNeighbors(queryIdx int, candidates []int32) []int32 {
	if len(candidates) == 0 {
		return nil
	}

	query := e.vectors[queryIdx].values

	// Sort candidates by distance
	type candDist struct {
		idx  int32
		dist float32
	}
	sorted := make([]candDist, 0, len(candidates))
	for _, c := range candidates {
		if c != int32(queryIdx) {
			sorted = append(sorted, candDist{idx: c, dist: e.distFunc(query, e.vectors[c].values)})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].dist < sorted[j].dist
	})

	// Select using MRNG criteria (simplified)
	selected := make([]int32, 0, e.R)
	for _, cand := range sorted {
		if len(selected) >= e.R {
			break
		}

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

	visited := make([]bool, n)
	L := e.L
	if L < k*2 {
		L = k * 2
	}

	// Start from navigating node
	type candidate struct {
		idx  int32
		dist float32
	}
	candidates := make([]candidate, 0, L*2)
	startDist := e.distFunc(query, e.vectors[e.navNode].values)
	candidates = append(candidates, candidate{idx: e.navNode, dist: startDist})
	visited[e.navNode] = true

	// Greedy search
	processed := 0
	for processed < len(candidates) && processed < L {
		// Get best unprocessed candidate
		bestIdx := processed
		for i := processed + 1; i < len(candidates); i++ {
			if candidates[i].dist < candidates[bestIdx].dist {
				bestIdx = i
			}
		}
		// Swap to front of unprocessed
		candidates[processed], candidates[bestIdx] = candidates[bestIdx], candidates[processed]
		curr := candidates[processed]
		processed++

		// Explore neighbors
		for _, neighbor := range e.graph[curr.idx] {
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true

			dist := e.distFunc(query, e.vectors[neighbor].values)
			candidates = append(candidates, candidate{idx: neighbor, dist: dist})
		}
	}

	// Sort by distance
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dist < candidates[j].dist
	})

	// Return top k
	if k > len(candidates) {
		k = len(candidates)
	}

	results := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		results[i] = SearchResult{
			ID:       e.vectors[candidates[i].idx].id,
			Distance: candidates[i].dist,
		}
	}

	return results
}

func (e *NSGEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *NSGEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
