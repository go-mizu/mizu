package mizu_vector

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/viterin/vek/vek32"
)

// HNSWEngine implements Hierarchical Navigable Small World graph.
// Time complexity: O(log n) per query.
// Based on "Efficient and robust approximate nearest neighbor search using HNSW" (Malkov & Yashunin, 2018).
//
// Optimized with:
// - SIMD-accelerated distance computation using viterin/vek
// - Contiguous memory layout for cache efficiency
// - Precomputed L2 norms for faster cosine distance
// - Typed heaps (no interface{} overhead)
// - Bitset for visited tracking (cache-friendly)
// - Lower Ml (0.25) to match external hnsw library
type HNSWEngine struct {
	distFunc DistanceFunc
	metric   vectorize.DistanceMetric

	// HNSW parameters
	M           int     // Max connections per node at layer 0
	Mmax        int     // Max connections per node at layers > 0
	Ml          float64 // Level generation factor (lower = fewer levels)
	efSearch    int     // Search beam width
	efConstruct int     // Construction beam width

	// Contiguous memory layout for cache efficiency
	vectorData  []float32 // All vector values: [v0d0, v0d1, ..., v1d0, ...]
	vectorIDs   []string  // Vector IDs indexed by int32
	vectorNorms []float32 // Precomputed L2 norms for cosine distance
	dims        int

	// Graph structure
	levels     []int         // Level for each vector
	neighbors  [][][]int32   // neighbors[level][nodeIdx] = list of neighbor indices
	entryPoint int32
	maxLevel   int

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

// HNSW configuration - tuned to match external hnsw library
const (
	hnswDefaultM        = 16
	hnswDefaultMmax     = 16    // Same as M for simplicity
	hnswDefaultMl       = 0.25  // Match external hnsw library
	hnswDefaultEfSearch = 64
	hnswDefaultEfConstr = 200
)

// NewHNSWEngine creates a new HNSW search engine.
func NewHNSWEngine(distFunc DistanceFunc) *HNSWEngine {
	return &HNSWEngine{
		distFunc:     distFunc,
		M:            hnswDefaultM,
		Mmax:         hnswDefaultMmax,
		Ml:           hnswDefaultMl,
		efSearch:     hnswDefaultEfSearch,
		efConstruct:  hnswDefaultEfConstr,
		entryPoint:   -1,
		needsRebuild: true,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *HNSWEngine) Name() string { return "hnsw" }

func (e *HNSWEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
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

	// Reset state
	e.levels = make([]int, 0, n)
	e.neighbors = make([][][]int32, 0)
	e.entryPoint = -1
	e.maxLevel = 0

	// Collect and insert all vectors
	for id, v := range vectors {
		e.vectorIDs = append(e.vectorIDs, id)
		e.vectorData = append(e.vectorData, v.Values...)
		e.vectorNorms = append(e.vectorNorms, vek32.Norm(v.Values))
	}

	// Insert all vectors into the graph
	for i := range e.vectorIDs {
		e.insertNode(int32(i))
	}

	e.needsRebuild = false
}

// getVector returns vector data at index using contiguous storage
func (e *HNSWEngine) getVector(idx int32) []float32 {
	start := int(idx) * e.dims
	return e.vectorData[start : start+e.dims]
}

// computeDistanceSIMD computes distance based on metric using SIMD
func (e *HNSWEngine) computeDistanceSIMD(aIdx int32, b []float32, bNorm float32) float32 {
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
func (e *HNSWEngine) computeDistanceBetween(aIdx, bIdx int32) float32 {
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

// randomLevel generates a random level for a new node.
// Using lower Ml produces fewer levels for faster search.
func (e *HNSWEngine) randomLevel() int {
	r := e.rng.Float64()
	level := int(-math.Log(r) * e.Ml)
	return level
}

// insertNode inserts a node into the HNSW graph.
func (e *HNSWEngine) insertNode(idx int32) {
	level := e.randomLevel()
	values := e.getVector(idx)
	valuesNorm := e.vectorNorms[idx]

	// Track level for this node
	e.levels = append(e.levels, level)

	// Ensure neighbors structure has enough levels
	for len(e.neighbors) <= level {
		e.neighbors = append(e.neighbors, make([][]int32, len(e.vectorIDs)))
	}
	// Extend each level's slice if needed
	for l := 0; l <= level; l++ {
		for len(e.neighbors[l]) <= int(idx) {
			e.neighbors[l] = append(e.neighbors[l], nil)
		}
		e.neighbors[l][idx] = make([]int32, 0, e.M)
	}

	if e.entryPoint < 0 {
		e.entryPoint = idx
		e.maxLevel = level
		return
	}

	// Search for entry point at each level
	currIdx := e.entryPoint
	currDist := e.computeDistanceSIMD(currIdx, values, valuesNorm)

	// Greedy search from top level to level+1
	for l := e.maxLevel; l > level; l-- {
		changed := true
		for changed {
			changed = false
			if l < len(e.neighbors) && int(currIdx) < len(e.neighbors[l]) {
				for _, friendIdx := range e.neighbors[l][currIdx] {
					dist := e.computeDistanceSIMD(friendIdx, values, valuesNorm)
					if dist < currDist {
						currIdx = friendIdx
						currDist = dist
						changed = true
					}
				}
			}
		}
	}

	// Insert at each level from level down to 0
	for l := min(level, e.maxLevel); l >= 0; l-- {
		// Search for neighbors at this level
		neighbors := e.searchLevelIdx(values, valuesNorm, currIdx, e.efConstruct, l)

		// Select M best neighbors
		M := e.M
		if l > 0 {
			M = e.Mmax
		}
		selected := e.selectNeighborsIdx(idx, neighbors, M)

		// Add bidirectional connections
		if l < len(e.neighbors) && int(idx) < len(e.neighbors[l]) {
			e.neighbors[l][idx] = selected
		}

		for _, neighborIdx := range selected {
			if l < len(e.neighbors) && int(neighborIdx) < len(e.neighbors[l]) {
				e.neighbors[l][neighborIdx] = append(e.neighbors[l][neighborIdx], idx)

				// Prune if too many connections
				Mmax := e.M * 2
				if l > 0 {
					Mmax = e.Mmax * 2
				}
				if len(e.neighbors[l][neighborIdx]) > Mmax {
					e.neighbors[l][neighborIdx] = e.selectNeighborsIdx(
						neighborIdx,
						e.neighbors[l][neighborIdx],
						M,
					)
				}
			}
		}

		if len(neighbors) > 0 {
			currIdx = neighbors[0]
		}
	}

	// Update entry point if new node has higher level
	if level > e.maxLevel {
		e.maxLevel = level
		e.entryPoint = idx
	}
}

// searchLevelIdx performs beam search at a specific level using indices.
func (e *HNSWEngine) searchLevelIdx(query []float32, queryNorm float32, entryIdx int32, ef, level int) []int32 {
	if level >= len(e.neighbors) {
		return nil
	}

	n := len(e.vectorIDs)
	visited := newBitset(n)

	// Use typed heaps - candidates is min-heap, result is max-heap
	candidates := make(minHeap32, 0, ef*2)
	result := make(maxHeap32, 0, ef)

	dist := e.computeDistanceSIMD(entryIdx, query, queryNorm)
	candidates.PushItem(distItem32{idx: entryIdx, dist: dist})
	result.PushItem(distItem32{idx: entryIdx, dist: dist})
	visited.Set(entryIdx)

	for len(candidates) > 0 {
		curr := candidates.PopItem()

		// Stop if current candidate is worse than worst result
		if len(result) >= ef && curr.dist > result[0].dist {
			break
		}

		// Explore neighbors
		if int(curr.idx) < len(e.neighbors[level]) {
			for _, neighborIdx := range e.neighbors[level][curr.idx] {
				if visited.Test(neighborIdx) {
					continue
				}
				visited.Set(neighborIdx)

				dist := e.computeDistanceSIMD(neighborIdx, query, queryNorm)

				if len(result) < ef {
					candidates.PushItem(distItem32{idx: neighborIdx, dist: dist})
					result.PushItem(distItem32{idx: neighborIdx, dist: dist})
				} else if dist < result[0].dist {
					candidates.PushItem(distItem32{idx: neighborIdx, dist: dist})
					result.PopItem()
					result.PushItem(distItem32{idx: neighborIdx, dist: dist})
				}
			}
		}
	}

	// Extract result indices sorted by distance
	results := make([]int32, len(result))
	for i := len(result) - 1; i >= 0; i-- {
		results[i] = result.PopItem().idx
	}

	return results
}

// selectNeighborsIdx selects the M best neighbors using simple heuristic.
func (e *HNSWEngine) selectNeighborsIdx(queryIdx int32, candidates []int32, M int) []int32 {
	if len(candidates) <= M {
		return candidates
	}

	// Use max-heap for efficient top-M selection
	h := make(maxHeap32, 0, M)

	for _, idx := range candidates {
		dist := e.computeDistanceBetween(queryIdx, idx)
		if len(h) < M {
			h.PushItem(distItem32{idx: idx, dist: dist})
		} else if dist < h[0].dist {
			h.PopItem()
			h.PushItem(distItem32{idx: idx, dist: dist})
		}
	}

	// Extract sorted by distance
	selected := make([]int32, len(h))
	for i := len(h) - 1; i >= 0; i-- {
		selected[i] = h.PopItem().idx
	}

	return selected
}

func (e *HNSWEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *HNSWEngine) Delete(ids []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Mark as needing rebuild - full rebuild is simpler for deletion
	e.needsRebuild = true
}

func (e *HNSWEngine) Search(query []float32, k int) []SearchResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.vectorIDs) == 0 || e.entryPoint < 0 {
		return nil
	}

	// Precompute query norm for cosine distance
	queryNorm := vek32.Norm(query)

	// Start from entry point
	currIdx := e.entryPoint
	currDist := e.computeDistanceSIMD(currIdx, query, queryNorm)

	// Greedy descent from top level to level 1
	for l := e.maxLevel; l > 0; l-- {
		changed := true
		for changed {
			changed = false
			if l < len(e.neighbors) && int(currIdx) < len(e.neighbors[l]) {
				for _, friendIdx := range e.neighbors[l][currIdx] {
					dist := e.computeDistanceSIMD(friendIdx, query, queryNorm)
					if dist < currDist {
						currIdx = friendIdx
						currDist = dist
						changed = true
					}
				}
			}
		}
	}

	// Search at level 0 with ef
	ef := e.efSearch
	if ef < k*2 {
		ef = k * 2
	}

	neighbors := e.searchLevelIdx(query, queryNorm, currIdx, ef, 0)

	// Return top k with distances
	results := make([]SearchResult, 0, k)
	for _, idx := range neighbors {
		if int(idx) >= len(e.vectorIDs) {
			continue
		}
		dist := e.computeDistanceSIMD(idx, query, queryNorm)
		results = append(results, SearchResult{
			ID:       e.vectorIDs[idx],
			Distance: dist,
		})
		if len(results) >= k {
			break
		}
	}

	return results
}

func (e *HNSWEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *HNSWEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
