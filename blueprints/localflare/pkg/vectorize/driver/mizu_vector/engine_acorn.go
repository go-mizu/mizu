package mizu_vector

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// ACORNEngine implements a simplified graph-based search.
// Uses flat single-level graph for simplicity and speed.
type ACORNEngine struct {
	distFunc DistanceFunc

	// Vector storage
	vectors []indexedVector

	// Graph structure - simple k-NN graph
	graph [][]int32

	// Parameters
	K        int // Neighbors per node
	efSearch int // Search beam width

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

	// Collect vectors
	e.vectors = make([]indexedVector, 0, n)
	for id, v := range vectors {
		e.vectors = append(e.vectors, indexedVector{id: id, values: v.Values})
	}

	// Build simple k-NN graph
	e.buildKNNGraph()

	e.needsRebuild = false
}

// buildKNNGraph builds a simple k-NN graph using sampling.
func (e *ACORNEngine) buildKNNGraph() {
	n := len(e.vectors)
	k := e.K
	if k > n-1 {
		k = n - 1
	}

	e.graph = make([][]int32, n)

	// Sample-based k-NN construction
	sampleSize := k * 4
	if sampleSize > n {
		sampleSize = n
	}

	for i := 0; i < n; i++ {
		// Sample random candidates
		candidates := make([]int, 0, sampleSize)
		for len(candidates) < sampleSize {
			j := e.rng.Intn(n)
			if j != i {
				candidates = append(candidates, j)
			}
		}

		// Find k nearest from sample
		type neighbor struct {
			idx  int32
			dist float32
		}
		neighbors := make([]neighbor, len(candidates))
		for j, c := range candidates {
			neighbors[j] = neighbor{
				idx:  int32(c),
				dist: e.distFunc(e.vectors[i].values, e.vectors[c].values),
			}
		}

		sort.Slice(neighbors, func(a, b int) bool {
			return neighbors[a].dist < neighbors[b].dist
		})

		if k > len(neighbors) {
			k = len(neighbors)
		}

		e.graph[i] = make([]int32, k)
		for j := 0; j < k; j++ {
			e.graph[i][j] = neighbors[j].idx
		}
	}

	// Make graph bidirectional
	for i := 0; i < n; i++ {
		for _, neighbor := range e.graph[i] {
			// Add reverse edge if not present
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

	visited := make([]bool, n)
	ef := e.efSearch
	if ef < k*2 {
		ef = k * 2
	}

	// Start from random entry points
	numStarts := 3
	if numStarts > n {
		numStarts = n
	}

	type candidate struct {
		idx  int32
		dist float32
	}

	results := make([]candidate, 0, ef)

	// Initialize with random starting points
	for s := 0; s < numStarts; s++ {
		start := int32(e.rng.Intn(n))
		if !visited[start] {
			visited[start] = true
			dist := e.distFunc(query, e.vectors[start].values)
			results = append(results, candidate{idx: start, dist: dist})
		}
	}

	// Greedy search with limited iterations
	maxIter := ef * 2
	for iter := 0; iter < maxIter && len(results) > 0; iter++ {
		// Find best unprocessed candidate
		sort.Slice(results, func(i, j int) bool {
			return results[i].dist < results[j].dist
		})

		// Process first unvisited neighbors of best result
		improved := false
		for _, curr := range results {
			for _, neighbor := range e.graph[curr.idx] {
				if visited[neighbor] {
					continue
				}
				visited[neighbor] = true
				improved = true

				dist := e.distFunc(query, e.vectors[neighbor].values)
				results = append(results, candidate{idx: neighbor, dist: dist})
			}
			if improved {
				break
			}
		}

		if !improved {
			break
		}

		// Keep only top ef
		if len(results) > ef {
			sort.Slice(results, func(i, j int) bool {
				return results[i].dist < results[j].dist
			})
			results = results[:ef]
		}
	}

	// Sort and return top k
	sort.Slice(results, func(i, j int) bool {
		return results[i].dist < results[j].dist
	})

	if k > len(results) {
		k = len(results)
	}

	output := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		output[i] = SearchResult{
			ID:       e.vectors[results[i].idx].id,
			Distance: results[i].dist,
		}
	}
	return output
}

func (e *ACORNEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *ACORNEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }

// Helper for level generation (unused but kept for interface consistency)
func (e *ACORNEngine) randomLevel() int {
	return int(-math.Log(e.rng.Float64()) / 1.386294)
}
