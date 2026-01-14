package mizu_vector

import (
	"container/heap"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// ACORNEngine implements ACORN-1 filter-aware HNSW traversal.
// Based on Elasticsearch's implementation for filtered vector search.
//
// Key innovation: Instead of post-filtering, integrate filter into graph traversal.
// When many neighbors don't match the filter, explore extended neighborhoods.
//
// Standard HNSW: explore 32 neighbors → filter → often 0 results
// ACORN-1: explore 32 neighbors → if >10% filtered, explore neighbors' neighbors
type ACORNEngine struct {
	distFunc DistanceFunc

	// HNSW graph structure using compressed indices
	graph      *CompressedGraph
	store      *SoAVectorStore
	levels     []int    // Level of each node
	entryPoint int32    // Entry point node
	maxLevel   int

	// ACORN parameters
	M              int     // Max connections per layer
	Ml             float64 // Level generation factor
	efConstruction int     // Construction queue size
	efSearch       int     // Search queue size
	filterRatio    float32 // Threshold for extended neighborhood (default 0.1)

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

// ACORN configuration
const (
	acornDefaultM              = 16
	acornDefaultMl             = 1.0 / 1.386294 // 1/ln(2)
	acornDefaultEfConstruction = 128
	acornDefaultEfSearch       = 64
	acornDefaultFilterRatio    = 0.1
)

// NewACORNEngine creates a new ACORN-1 search engine.
func NewACORNEngine(distFunc DistanceFunc) *ACORNEngine {
	return &ACORNEngine{
		distFunc:       distFunc,
		M:              acornDefaultM,
		Ml:             acornDefaultMl,
		efConstruction: acornDefaultEfConstruction,
		efSearch:       acornDefaultEfSearch,
		filterRatio:    acornDefaultFilterRatio,
		needsRebuild:   true,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
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

	// Initialize stores
	e.store = NewSoAVectorStore(dims)
	e.graph = NewCompressedGraph(n)
	e.levels = make([]int, 0, n)
	e.maxLevel = 0
	e.entryPoint = -1

	// Add all vectors and build graph incrementally
	for id, v := range vectors {
		e.insertVector(id, v.Values)
	}

	e.needsRebuild = false
}

func (e *ACORNEngine) insertVector(id string, values []float32) {
	// Add to stores
	e.store.Add(id, values)
	idx := e.graph.AddNode(id)

	// Generate random level
	level := e.randomLevel()
	e.levels = append(e.levels, level)

	if e.entryPoint < 0 {
		e.entryPoint = idx
		e.maxLevel = level
		// Initialize empty neighbor lists for each level
		for l := 0; l <= level; l++ {
			e.graph.SetEdges(idx, nil)
		}
		return
	}

	// Find entry point at top level
	currNode := e.entryPoint
	currDist := e.distFunc(values, e.store.Get(int(currNode)))

	// Greedy search from top to insertion level
	for l := e.maxLevel; l > level; l-- {
		changed := true
		for changed {
			changed = false
			for _, neighbor := range e.graph.GetEdges(currNode) {
				dist := e.distFunc(values, e.store.Get(int(neighbor)))
				if dist < currDist {
					currNode = neighbor
					currDist = dist
					changed = true
				}
			}
		}
	}

	// Insert at each level from level down to 0
	for l := min(level, e.maxLevel); l >= 0; l-- {
		// Search for neighbors at this level
		neighbors := e.searchLevelACORN(values, currNode, e.efConstruction, l, nil)

		// Select M best neighbors using simple heuristic
		selected := e.selectNeighborsSimple(values, neighbors, e.M)

		// Connect bidirectionally
		for _, neighbor := range selected {
			// Add neighbor to current node
			currEdges := e.graph.GetEdges(idx)
			currEdges = append(currEdges, neighbor)
			e.graph.SetEdges(idx, currEdges)

			// Add current node to neighbor
			neighborEdges := e.graph.GetEdges(neighbor)
			neighborEdges = append(neighborEdges, idx)
			e.graph.SetEdges(neighbor, neighborEdges)

			// Prune neighbor if too many connections
			if len(neighborEdges) > e.M*2 {
				neighborVec := e.store.Get(int(neighbor))
				neighborEdgeIDs := make([]int32, len(neighborEdges))
				copy(neighborEdgeIDs, neighborEdges)
				pruned := e.selectNeighborsSimple(neighborVec, neighborEdgeIDs, e.M)
				e.graph.SetEdges(neighbor, pruned)
			}
		}

		if len(neighbors) > 0 {
			currNode = neighbors[0]
		}
	}

	// Update entry point if new node has higher level
	if level > e.maxLevel {
		e.maxLevel = level
		e.entryPoint = idx
	}
}

func (e *ACORNEngine) randomLevel() int {
	r := e.rng.Float64()
	return int(-math.Log(r) * e.Ml)
}

// searchLevelACORN performs search at a specific level with optional filter.
func (e *ACORNEngine) searchLevelACORN(query []float32, entry int32, ef, level int, filter func(int32) bool) []int32 {
	visited := make(map[int32]bool)
	candidates := &acornHeap{}
	results := &acornHeap{}

	heap.Init(candidates)
	heap.Init(results)

	entryDist := e.distFunc(query, e.store.Get(int(entry)))
	heap.Push(candidates, acornItem{idx: entry, dist: entryDist, isMax: false})
	heap.Push(results, acornItem{idx: entry, dist: entryDist, isMax: true})
	visited[entry] = true

	for candidates.Len() > 0 {
		curr := heap.Pop(candidates).(acornItem)

		// Stop if current candidate is worse than worst result
		if results.Len() >= ef {
			worst := (*results)[0]
			if curr.dist > worst.dist {
				break
			}
		}

		// Get neighbors
		neighbors := e.graph.GetEdges(curr.idx)
		matchCount := 0
		totalNeighbors := len(neighbors)

		for _, neighbor := range neighbors {
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true

			// Apply filter if provided
			if filter != nil && !filter(neighbor) {
				continue
			}
			matchCount++

			dist := e.distFunc(query, e.store.Get(int(neighbor)))

			if results.Len() < ef {
				heap.Push(candidates, acornItem{idx: neighbor, dist: dist, isMax: false})
				heap.Push(results, acornItem{idx: neighbor, dist: dist, isMax: true})
			} else if dist < (*results)[0].dist {
				heap.Push(candidates, acornItem{idx: neighbor, dist: dist, isMax: false})
				heap.Pop(results)
				heap.Push(results, acornItem{idx: neighbor, dist: dist, isMax: true})
			}
		}

		// ACORN-1: If too many neighbors filtered out, explore extended neighborhood
		if filter != nil && totalNeighbors > 0 {
			filterRatio := 1.0 - float32(matchCount)/float32(totalNeighbors)
			if filterRatio > e.filterRatio {
				e.exploreExtendedNeighborhood(query, neighbors, visited, candidates, results, ef, filter)
			}
		}
	}

	// Extract results
	result := make([]int32, results.Len())
	for i := results.Len() - 1; i >= 0; i-- {
		result[i] = heap.Pop(results).(acornItem).idx
	}

	return result
}

// exploreExtendedNeighborhood explores neighbors' neighbors when filter ratio is high.
func (e *ACORNEngine) exploreExtendedNeighborhood(query []float32, neighbors []int32, visited map[int32]bool,
	candidates, results *acornHeap, ef int, filter func(int32) bool) {

	for _, neighbor := range neighbors {
		for _, nn := range e.graph.GetEdges(neighbor) {
			if visited[nn] {
				continue
			}
			visited[nn] = true

			if filter != nil && !filter(nn) {
				continue
			}

			dist := e.distFunc(query, e.store.Get(int(nn)))

			if results.Len() < ef {
				heap.Push(candidates, acornItem{idx: nn, dist: dist, isMax: false})
				heap.Push(results, acornItem{idx: nn, dist: dist, isMax: true})
			} else if dist < (*results)[0].dist {
				heap.Push(candidates, acornItem{idx: nn, dist: dist, isMax: false})
				heap.Pop(results)
				heap.Push(results, acornItem{idx: nn, dist: dist, isMax: true})
			}
		}
	}
}

func (e *ACORNEngine) selectNeighborsSimple(query []float32, candidates []int32, M int) []int32 {
	if len(candidates) <= M {
		return candidates
	}

	// Sort by distance
	type candDist struct {
		idx  int32
		dist float32
	}
	sorted := make([]candDist, len(candidates))
	for i, c := range candidates {
		sorted[i] = candDist{idx: c, dist: e.distFunc(query, e.store.Get(int(c)))}
	}

	// Partial sort using heap for efficiency
	h := &acornMaxHeap{}
	heap.Init(h)
	for _, cd := range sorted {
		if h.Len() < M {
			heap.Push(h, cd)
		} else if cd.dist < (*h)[0].dist {
			heap.Pop(h)
			heap.Push(h, cd)
		}
	}

	// Extract results
	selected := make([]int32, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		selected[i] = heap.Pop(h).(candDist).idx
	}

	return selected
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

	if e.graph == nil || e.graph.Len() == 0 || e.entryPoint < 0 {
		return nil
	}

	// Greedy descent from top level
	currNode := e.entryPoint
	currDist := e.distFunc(query, e.store.Get(int(currNode)))

	for l := e.maxLevel; l > 0; l-- {
		changed := true
		for changed {
			changed = false
			for _, neighbor := range e.graph.GetEdges(currNode) {
				dist := e.distFunc(query, e.store.Get(int(neighbor)))
				if dist < currDist {
					currNode = neighbor
					currDist = dist
					changed = true
				}
			}
		}
	}

	// Search at level 0
	ef := e.efSearch
	if ef < k {
		ef = k * 2
	}

	// No filter for basic search
	neighbors := e.searchLevelACORN(query, currNode, ef, 0, nil)

	// Return top k
	results := make([]SearchResult, 0, k)
	for _, idx := range neighbors {
		if len(results) >= k {
			break
		}
		dist := e.distFunc(query, e.store.Get(int(idx)))
		results = append(results, SearchResult{
			ID:       e.graph.GetID(idx),
			Distance: dist,
		})
	}

	return results
}

// SearchWithFilter performs filtered search using ACORN-1 algorithm.
func (e *ACORNEngine) SearchWithFilter(query []float32, k int, filter func(id string) bool) []SearchResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.graph == nil || e.graph.Len() == 0 || e.entryPoint < 0 {
		return nil
	}

	// Create index-based filter
	idxFilter := func(idx int32) bool {
		return filter(e.graph.GetID(idx))
	}

	// Greedy descent from top level
	currNode := e.entryPoint
	currDist := e.distFunc(query, e.store.Get(int(currNode)))

	for l := e.maxLevel; l > 0; l-- {
		changed := true
		for changed {
			changed = false
			for _, neighbor := range e.graph.GetEdges(currNode) {
				dist := e.distFunc(query, e.store.Get(int(neighbor)))
				if dist < currDist {
					currNode = neighbor
					currDist = dist
					changed = true
				}
			}
		}
	}

	// Search at level 0 with filter
	ef := e.efSearch * 2 // Larger ef for filtered search
	if ef < k*4 {
		ef = k * 4
	}

	neighbors := e.searchLevelACORN(query, currNode, ef, 0, idxFilter)

	// Return top k
	results := make([]SearchResult, 0, k)
	for _, idx := range neighbors {
		if len(results) >= k {
			break
		}
		id := e.graph.GetID(idx)
		if !filter(id) {
			continue
		}
		dist := e.distFunc(query, e.store.Get(int(idx)))
		results = append(results, SearchResult{
			ID:       id,
			Distance: dist,
		})
	}

	return results
}

func (e *ACORNEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *ACORNEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }

// Heap implementations for ACORN
type acornItem struct {
	idx   int32
	dist  float32
	isMax bool
}

type acornHeap []acornItem

func (h acornHeap) Len() int { return len(h) }
func (h acornHeap) Less(i, j int) bool {
	if h[i].isMax {
		return h[i].dist > h[j].dist // max-heap
	}
	return h[i].dist < h[j].dist // min-heap
}
func (h acornHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *acornHeap) Push(x any)         { *h = append(*h, x.(acornItem)) }
func (h *acornHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type acornMaxHeap []struct {
	idx  int32
	dist float32
}

func (h acornMaxHeap) Len() int           { return len(h) }
func (h acornMaxHeap) Less(i, j int) bool { return h[i].dist > h[j].dist }
func (h acornMaxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *acornMaxHeap) Push(x any) {
	*h = append(*h, x.(struct {
		idx  int32
		dist float32
	}))
}
func (h *acornMaxHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
