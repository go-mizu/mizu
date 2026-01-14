package mizu_vector

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// IVFEngine implements Inverted File Index with k-means clustering.
// Time complexity: O(sqrt(n)*d) per query.
// Based on "Product quantization for nearest neighbor search" (Jégou et al., 2011).
//
// Optimized with:
// - Index-based storage (no string lookups in hot path)
// - Typed top-K heap instead of full sorting
// - Inline search for small clusters (no goroutine overhead)
type IVFEngine struct {
	distFunc DistanceFunc

	// Index-based storage
	vectors   []ivfVector     // Vector data indexed by int32
	centroids [][]float32     // [nClusters][dims]
	clusters  [][]int32       // Vector indices in each cluster
	nProbe    int             // Number of clusters to search

	needsRebuild bool
}

type ivfVector struct {
	id     string
	values []float32
}

// IVF configuration
const (
	ivfMinVectors      = 256
	ivfClustersPerSqrt = 4
	ivfMaxClusters     = 256
	ivfKMeansIters     = 10
	ivfDefaultNProbe   = 8
	ivfParallelThresh  = 500 // Only use goroutines if cluster has > this many vectors
)

// NewIVFEngine creates a new IVF search engine.
func NewIVFEngine(distFunc DistanceFunc) *IVFEngine {
	return &IVFEngine{
		distFunc:     distFunc,
		nProbe:       ivfDefaultNProbe,
		needsRebuild: true,
	}
}

func (e *IVFEngine) Name() string { return "ivf" }

func (e *IVFEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
	n := len(vectors)
	if n == 0 {
		e.needsRebuild = false
		return
	}

	// Collect vectors with index-based storage
	e.vectors = make([]ivfVector, 0, n)
	for id, v := range vectors {
		e.vectors = append(e.vectors, ivfVector{id: id, values: v.Values})
	}

	if n < ivfMinVectors {
		// Fall back to flat storage for small datasets
		e.buildFlat()
		return
	}

	// Determine number of clusters
	nClusters := int(math.Sqrt(float64(n)) * ivfClustersPerSqrt)
	if nClusters > ivfMaxClusters {
		nClusters = ivfMaxClusters
	}
	if nClusters < 2 {
		nClusters = 2
	}

	// K-means++ initialization
	centroids := e.kmeansppInit(nClusters, dims)

	// Run k-means
	assignments := make([]int, n)
	for iter := 0; iter < ivfKMeansIters; iter++ {
		e.assignToCentroids(centroids, assignments)
		centroids = e.updateCentroids(assignments, nClusters, dims)
	}

	// Build cluster structure
	clusters := make([][]int32, nClusters)
	for i := range clusters {
		clusters[i] = make([]int32, 0)
	}

	for i := range e.vectors {
		cluster := assignments[i]
		clusters[cluster] = append(clusters[cluster], int32(i))
	}

	e.centroids = centroids
	e.clusters = clusters
	e.needsRebuild = false
}

func (e *IVFEngine) buildFlat() {
	// Single cluster for small datasets
	e.centroids = nil
	n := len(e.vectors)
	indices := make([]int32, n)
	for i := 0; i < n; i++ {
		indices[i] = int32(i)
	}
	e.clusters = [][]int32{indices}
	e.needsRebuild = false
}

func (e *IVFEngine) kmeansppInit(k, dims int) [][]float32 {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := len(e.vectors)
	centroids := make([][]float32, k)

	// First centroid: random
	firstIdx := rng.Intn(n)
	centroids[0] = make([]float32, dims)
	copy(centroids[0], e.vectors[firstIdx].values)

	distances := make([]float32, n)

	for i := 1; i < k; i++ {
		var totalDist float32

		// Distance to nearest centroid
		for j := 0; j < n; j++ {
			minDist := float32(math.MaxFloat32)
			for c := 0; c < i; c++ {
				d := e.distFunc(e.vectors[j].values, centroids[c])
				if d < minDist {
					minDist = d
				}
			}
			distances[j] = minDist * minDist
			totalDist += distances[j]
		}

		// Sample proportional to distance squared
		if totalDist > 0 {
			target := rng.Float32() * totalDist
			var cumulative float32
			for j, d := range distances {
				cumulative += d
				if cumulative >= target {
					centroids[i] = make([]float32, dims)
					copy(centroids[i], e.vectors[j].values)
					break
				}
			}
		}

		if centroids[i] == nil {
			centroids[i] = make([]float32, dims)
			copy(centroids[i], e.vectors[rng.Intn(n)].values)
		}
	}

	return centroids
}

func (e *IVFEngine) assignToCentroids(centroids [][]float32, assignments []int) {
	n := len(e.vectors)
	nWorkers := runtime.NumCPU()
	chunkSize := (n + nWorkers - 1) / nWorkers

	var wg sync.WaitGroup
	for w := 0; w < nWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > n {
			end = n
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				minDist := float32(math.MaxFloat32)
				minIdx := 0
				for c, centroid := range centroids {
					d := e.distFunc(e.vectors[i].values, centroid)
					if d < minDist {
						minDist = d
						minIdx = c
					}
				}
				assignments[i] = minIdx
			}
		}(start, end)
	}
	wg.Wait()
}

func (e *IVFEngine) updateCentroids(assignments []int, k, dims int) [][]float32 {
	newCentroids := make([][]float32, k)
	counts := make([]int, k)

	for i := range newCentroids {
		newCentroids[i] = make([]float32, dims)
	}

	for i, v := range e.vectors {
		c := assignments[i]
		counts[c]++
		for d := 0; d < dims; d++ {
			newCentroids[c][d] += v.values[d]
		}
	}

	for c := 0; c < k; c++ {
		if counts[c] > 0 {
			for d := 0; d < dims; d++ {
				newCentroids[c][d] /= float32(counts[c])
			}
		}
	}

	return newCentroids
}

func (e *IVFEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *IVFEngine) Delete(ids []string) {
	e.needsRebuild = true
}

func (e *IVFEngine) Search(query []float32, k int) []SearchResult {
	if len(e.clusters) == 0 {
		return nil
	}

	// No centroids = flat search
	if e.centroids == nil {
		return e.searchClusterTopK(e.clusters[0], query, k)
	}

	// Find nearest clusters using heap-based selection
	nProbe := e.nProbe
	if nProbe > len(e.centroids) {
		nProbe = len(e.centroids)
	}

	// Use max-heap to find nProbe nearest centroids
	topClusters := make(maxHeap32, 0, nProbe)
	for i, centroid := range e.centroids {
		dist := e.distFunc(query, centroid)
		if len(topClusters) < nProbe {
			topClusters.PushItem(distItem32{idx: int32(i), dist: dist})
		} else if dist < topClusters[0].dist {
			topClusters.PopItem()
			topClusters.PushItem(distItem32{idx: int32(i), dist: dist})
		}
	}

	// Count total vectors to search
	totalVecs := 0
	for len(topClusters) > 0 {
		item := topClusters.PopItem()
		totalVecs += len(e.clusters[item.idx])
	}

	// Rebuild topClusters for iteration
	topClusters = make(maxHeap32, 0, nProbe)
	for i, centroid := range e.centroids {
		dist := e.distFunc(query, centroid)
		if len(topClusters) < nProbe {
			topClusters.PushItem(distItem32{idx: int32(i), dist: dist})
		} else if dist < topClusters[0].dist {
			topClusters.PopItem()
			topClusters.PushItem(distItem32{idx: int32(i), dist: dist})
		}
	}

	// Use parallel search only for large total vector counts
	if totalVecs > ivfParallelThresh*2 {
		return e.searchClustersParallel(topClusters, query, k)
	}

	// Serial search for smaller datasets
	return e.searchClustersMerged(topClusters, query, k)
}

// searchClustersMerged searches clusters serially and merges results using top-K heap.
func (e *IVFEngine) searchClustersMerged(topClusters maxHeap32, query []float32, k int) []SearchResult {
	resultHeap := make(maxHeap32, 0, k)

	for len(topClusters) > 0 {
		item := topClusters.PopItem()
		cluster := e.clusters[item.idx]

		for _, vecIdx := range cluster {
			dist := e.distFunc(query, e.vectors[vecIdx].values)
			if len(resultHeap) < k {
				resultHeap.PushItem(distItem32{idx: vecIdx, dist: dist})
			} else if dist < resultHeap[0].dist {
				resultHeap.PopItem()
				resultHeap.PushItem(distItem32{idx: vecIdx, dist: dist})
			}
		}
	}

	// Extract sorted results
	results := make([]SearchResult, len(resultHeap))
	for i := len(resultHeap) - 1; i >= 0; i-- {
		item := resultHeap.PopItem()
		results[i] = SearchResult{
			ID:       e.vectors[item.idx].id,
			Distance: item.dist,
		}
	}

	return results
}

// searchClustersParallel searches clusters in parallel and merges results.
func (e *IVFEngine) searchClustersParallel(topClusters maxHeap32, query []float32, k int) []SearchResult {
	nClusters := len(topClusters)
	resultsChan := make(chan []distItem32, nClusters)
	var wg sync.WaitGroup

	for len(topClusters) > 0 {
		item := topClusters.PopItem()
		cluster := e.clusters[item.idx]

		if len(cluster) == 0 {
			continue
		}

		wg.Add(1)
		go func(cluster []int32) {
			defer wg.Done()
			// Use top-K heap for each cluster
			clusterHeap := make(maxHeap32, 0, k)
			for _, vecIdx := range cluster {
				dist := e.distFunc(query, e.vectors[vecIdx].values)
				if len(clusterHeap) < k {
					clusterHeap.PushItem(distItem32{idx: vecIdx, dist: dist})
				} else if dist < clusterHeap[0].dist {
					clusterHeap.PopItem()
					clusterHeap.PushItem(distItem32{idx: vecIdx, dist: dist})
				}
			}
			resultsChan <- clusterHeap
		}(cluster)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Merge results using top-K heap
	finalHeap := make(maxHeap32, 0, k)
	for clusterResults := range resultsChan {
		for _, item := range clusterResults {
			if len(finalHeap) < k {
				finalHeap.PushItem(item)
			} else if item.dist < finalHeap[0].dist {
				finalHeap.PopItem()
				finalHeap.PushItem(item)
			}
		}
	}

	// Extract sorted results
	results := make([]SearchResult, len(finalHeap))
	for i := len(finalHeap) - 1; i >= 0; i-- {
		item := finalHeap.PopItem()
		results[i] = SearchResult{
			ID:       e.vectors[item.idx].id,
			Distance: item.dist,
		}
	}

	return results
}

// searchClusterTopK searches a single cluster using top-K heap.
func (e *IVFEngine) searchClusterTopK(cluster []int32, query []float32, k int) []SearchResult {
	resultHeap := make(maxHeap32, 0, k)

	for _, vecIdx := range cluster {
		dist := e.distFunc(query, e.vectors[vecIdx].values)
		if len(resultHeap) < k {
			resultHeap.PushItem(distItem32{idx: vecIdx, dist: dist})
		} else if dist < resultHeap[0].dist {
			resultHeap.PopItem()
			resultHeap.PushItem(distItem32{idx: vecIdx, dist: dist})
		}
	}

	// Extract sorted results
	results := make([]SearchResult, len(resultHeap))
	for i := len(resultHeap) - 1; i >= 0; i-- {
		item := resultHeap.PopItem()
		results[i] = SearchResult{
			ID:       e.vectors[item.idx].id,
			Distance: item.dist,
		}
	}

	return results
}

func (e *IVFEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *IVFEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
