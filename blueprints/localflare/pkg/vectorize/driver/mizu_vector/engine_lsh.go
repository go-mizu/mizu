package mizu_vector

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// LSHEngine implements Locality Sensitive Hashing with random hyperplane projections.
// Time complexity: O(L*K + candidates) where L=tables, K=hash functions.
// Based on "Approximate Nearest Neighbors: Towards Removing the Curse of Dimensionality" (Indyk & Motwani, 1998).
type LSHEngine struct {
	distFunc DistanceFunc

	// LSH parameters
	numTables    int         // Number of hash tables (L)
	numHashFuncs int         // Number of hash functions per table (K)
	hyperplanes  [][][]float32 // [L][K][dims] random hyperplanes

	// Index structures
	tables       []map[uint64][]indexedVector // Hash tables
	allVectors   []indexedVector              // For fallback

	needsRebuild bool
	dims         int
}

// LSH configuration
const (
	lshDefaultTables    = 8
	lshDefaultHashFuncs = 12
)

// NewLSHEngine creates a new LSH search engine.
func NewLSHEngine(distFunc DistanceFunc, dims int) *LSHEngine {
	return &LSHEngine{
		distFunc:     distFunc,
		numTables:    lshDefaultTables,
		numHashFuncs: lshDefaultHashFuncs,
		dims:         dims,
		needsRebuild: true,
	}
}

func (e *LSHEngine) Name() string { return "lsh" }

func (e *LSHEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
	e.dims = dims

	// Generate random hyperplanes
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	e.hyperplanes = make([][][]float32, e.numTables)
	for t := 0; t < e.numTables; t++ {
		e.hyperplanes[t] = make([][]float32, e.numHashFuncs)
		for h := 0; h < e.numHashFuncs; h++ {
			plane := make([]float32, dims)
			for d := 0; d < dims; d++ {
				plane[d] = float32(rng.NormFloat64())
			}
			// Normalize
			var norm float32
			for _, v := range plane {
				norm += v * v
			}
			norm = float32(math.Sqrt(float64(norm)))
			for d := range plane {
				plane[d] /= norm
			}
			e.hyperplanes[t][h] = plane
		}
	}

	// Build hash tables
	e.tables = make([]map[uint64][]indexedVector, e.numTables)
	for t := 0; t < e.numTables; t++ {
		e.tables[t] = make(map[uint64][]indexedVector)
	}

	e.allVectors = make([]indexedVector, 0, len(vectors))
	for id, v := range vectors {
		iv := indexedVector{id: id, values: v.Values}
		e.allVectors = append(e.allVectors, iv)

		// Hash into each table
		for t := 0; t < e.numTables; t++ {
			hash := e.computeHash(v.Values, t)
			e.tables[t][hash] = append(e.tables[t][hash], iv)
		}
	}

	e.needsRebuild = false
}

// computeHash computes the LSH hash for a vector in table t.
func (e *LSHEngine) computeHash(v []float32, t int) uint64 {
	var hash uint64
	for h := 0; h < e.numHashFuncs; h++ {
		// Compute dot product with hyperplane
		var dot float32
		for d := 0; d < len(v); d++ {
			dot += v[d] * e.hyperplanes[t][h][d]
		}
		// Set bit if positive
		if dot >= 0 {
			hash |= 1 << h
		}
	}
	return hash
}

func (e *LSHEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *LSHEngine) Delete(ids []string) {
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	// Remove from all tables
	for t := range e.tables {
		for hash, bucket := range e.tables[t] {
			filtered := make([]indexedVector, 0, len(bucket))
			for _, v := range bucket {
				if _, deleted := idSet[v.id]; !deleted {
					filtered = append(filtered, v)
				}
			}
			if len(filtered) == 0 {
				delete(e.tables[t], hash)
			} else {
				e.tables[t][hash] = filtered
			}
		}
	}

	// Remove from allVectors
	filtered := make([]indexedVector, 0, len(e.allVectors))
	for _, v := range e.allVectors {
		if _, deleted := idSet[v.id]; !deleted {
			filtered = append(filtered, v)
		}
	}
	e.allVectors = filtered
}

func (e *LSHEngine) Search(query []float32, k int) []SearchResult {
	if len(e.allVectors) == 0 {
		return nil
	}

	// Collect candidates from all tables
	candidateSet := make(map[string]indexedVector)

	for t := 0; t < e.numTables; t++ {
		hash := e.computeHash(query, t)

		// Check exact hash match
		if bucket, ok := e.tables[t][hash]; ok {
			for _, v := range bucket {
				candidateSet[v.id] = v
			}
		}

		// Multi-probe: check neighboring hashes (1-bit flips)
		for bit := 0; bit < e.numHashFuncs; bit++ {
			neighborHash := hash ^ (1 << bit)
			if bucket, ok := e.tables[t][neighborHash]; ok {
				for _, v := range bucket {
					candidateSet[v.id] = v
				}
			}
		}
	}

	// If too few candidates, search all
	if len(candidateSet) < k {
		return e.exhaustiveSearch(query, k)
	}

	// Compute exact distances for candidates
	results := make([]SearchResult, 0, len(candidateSet))
	for _, v := range candidateSet {
		dist := e.distFunc(query, v.values)
		results = append(results, SearchResult{ID: v.id, Distance: dist})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if k > len(results) {
		k = len(results)
	}
	return results[:k]
}

func (e *LSHEngine) exhaustiveSearch(query []float32, k int) []SearchResult {
	results := make([]SearchResult, 0, len(e.allVectors))
	for _, v := range e.allVectors {
		dist := e.distFunc(query, v.values)
		results = append(results, SearchResult{ID: v.id, Distance: dist})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if k > len(results) {
		k = len(results)
	}
	return results[:k]
}

func (e *LSHEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *LSHEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
