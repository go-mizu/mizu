package mizu_vector

import (
	"runtime"
	"sort"
	"sync"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// FlatEngine implements brute-force exact search.
// Time complexity: O(n*d) per query where n=vectors, d=dimensions.
// This serves as the baseline for accuracy comparison.
type FlatEngine struct {
	distFunc     DistanceFunc
	vectors      []indexedVector
	needsRebuild bool
}

type indexedVector struct {
	id     string
	values []float32
}

// NewFlatEngine creates a new brute-force search engine.
func NewFlatEngine(distFunc DistanceFunc) *FlatEngine {
	return &FlatEngine{
		distFunc:     distFunc,
		needsRebuild: true,
	}
}

func (e *FlatEngine) Name() string { return "flat" }

func (e *FlatEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
	e.vectors = make([]indexedVector, 0, len(vectors))
	for id, v := range vectors {
		e.vectors = append(e.vectors, indexedVector{
			id:     id,
			values: v.Values,
		})
	}
	e.needsRebuild = false
}

func (e *FlatEngine) Insert(vectors []*vectorize.Vector) {
	// Flat engine rebuilds on search, just mark dirty
	e.needsRebuild = true
}

func (e *FlatEngine) Delete(ids []string) {
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	filtered := make([]indexedVector, 0, len(e.vectors))
	for _, v := range e.vectors {
		if _, deleted := idSet[v.id]; !deleted {
			filtered = append(filtered, v)
		}
	}
	e.vectors = filtered
}

func (e *FlatEngine) Search(query []float32, k int) []SearchResult {
	if len(e.vectors) == 0 {
		return nil
	}

	// Parallel distance computation
	nWorkers := runtime.NumCPU()
	chunkSize := (len(e.vectors) + nWorkers - 1) / nWorkers

	type distResult struct {
		id   string
		dist float32
	}

	resultsChan := make(chan []distResult, nWorkers)
	var wg sync.WaitGroup

	for w := 0; w < nWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(e.vectors) {
			end = len(e.vectors)
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			results := make([]distResult, 0, end-start)
			for i := start; i < end; i++ {
				dist := e.distFunc(query, e.vectors[i].values)
				results = append(results, distResult{e.vectors[i].id, dist})
			}
			resultsChan <- results
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect and sort
	allResults := make([]distResult, 0, len(e.vectors))
	for results := range resultsChan {
		allResults = append(allResults, results...)
	}

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].dist < allResults[j].dist
	})

	if k > len(allResults) {
		k = len(allResults)
	}

	results := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		results[i] = SearchResult{
			ID:       allResults[i].id,
			Distance: allResults[i].dist,
		}
	}

	return results
}

func (e *FlatEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *FlatEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
