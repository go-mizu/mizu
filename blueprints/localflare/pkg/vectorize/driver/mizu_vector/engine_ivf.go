package mizu_vector

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/viterin/vek/vek32"
)

// IVFEngine implements Inverted File Index with k-means clustering.
// Time complexity: O(sqrt(n)*d) per query.
// Based on "Product quantization for nearest neighbor search" (Jégou et al., 2011).
//
// Optimized with:
// - SIMD-accelerated distance computation using viterin/vek
// - Contiguous memory layout for cache efficiency
// - Precomputed L2 norms for faster cosine distance
// - Early termination based on centroid distance lower bound
// - Typed top-K heap instead of full sorting
type IVFEngine struct {
	distFunc DistanceFunc
	metric   vectorize.DistanceMetric

	// Contiguous memory layout for cache efficiency
	vectorData  []float32 // All vector values: [v0d0, v0d1, ..., v1d0, ...]
	vectorIDs   []string  // Vector IDs indexed by int32
	vectorNorms []float32 // Precomputed L2 norms for cosine distance
	dims        int

	// Clustering
	centroids     [][]float32 // [nClusters][dims]
	centroidNorms []float32   // Precomputed norms for centroids
	clusters      [][]int32   // Vector indices in each cluster
	nProbe        int         // Number of clusters to search

	needsRebuild bool
}

// IVF configuration
const (
	ivfMinVectors      = 256
	ivfClustersPerSqrt = 4
	ivfMaxClusters     = 256
	ivfKMeansIters     = 10
	ivfDefaultNProbe   = 10  // Increased for better recall
	ivfParallelThresh  = 300 // Lower threshold for parallel search
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

	e.dims = dims
	e.metric = metric

	// Build contiguous storage for cache efficiency
	e.vectorData = make([]float32, 0, n*dims)
	e.vectorIDs = make([]string, 0, n)
	e.vectorNorms = make([]float32, 0, n)

	for id, v := range vectors {
		e.vectorIDs = append(e.vectorIDs, id)
		e.vectorData = append(e.vectorData, v.Values...)
		// Precompute norm for cosine distance optimization
		e.vectorNorms = append(e.vectorNorms, vek32.Norm(v.Values))
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

	// K-means++ initialization with SIMD
	centroids := e.kmeansppInit(nClusters, dims)

	// Run k-means with parallel assignment
	assignments := make([]int, n)
	for iter := 0; iter < ivfKMeansIters; iter++ {
		e.assignToCentroids(centroids, assignments)
		centroids = e.updateCentroids(assignments, nClusters, dims)
	}

	// Precompute centroid norms
	e.centroidNorms = make([]float32, len(centroids))
	for i, c := range centroids {
		e.centroidNorms[i] = vek32.Norm(c)
	}

	// Build cluster structure
	clusters := make([][]int32, nClusters)
	for i := range clusters {
		clusters[i] = make([]int32, 0)
	}

	for i := range e.vectorIDs {
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
	e.centroidNorms = nil
	n := len(e.vectorIDs)
	indices := make([]int32, n)
	for i := 0; i < n; i++ {
		indices[i] = int32(i)
	}
	e.clusters = [][]int32{indices}
	e.needsRebuild = false
}

// getVector returns vector data at index using contiguous storage
func (e *IVFEngine) getVector(idx int) []float32 {
	start := idx * e.dims
	return e.vectorData[start : start+e.dims]
}

func (e *IVFEngine) kmeansppInit(k, dims int) [][]float32 {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := len(e.vectorIDs)
	centroids := make([][]float32, k)

	// First centroid: random
	firstIdx := rng.Intn(n)
	centroids[0] = make([]float32, dims)
	copy(centroids[0], e.getVector(firstIdx))

	distances := make([]float32, n)

	for i := 1; i < k; i++ {
		var totalDist float32

		// Distance to nearest centroid using SIMD
		for j := 0; j < n; j++ {
			vec := e.getVector(j)
			minDist := float32(math.MaxFloat32)
			for c := 0; c < i; c++ {
				d := vek32.Distance(vec, centroids[c])
				if d < minDist {
					minDist = d
				}
			}
			distances[j] = minDist
			totalDist += minDist
		}

		// Sample proportional to distance
		if totalDist > 0 {
			target := rng.Float32() * totalDist
			var cumulative float32
			for j, d := range distances {
				cumulative += d
				if cumulative >= target {
					centroids[i] = make([]float32, dims)
					copy(centroids[i], e.getVector(j))
					break
				}
			}
		}

		if centroids[i] == nil {
			centroids[i] = make([]float32, dims)
			copy(centroids[i], e.getVector(rng.Intn(n)))
		}
	}

	return centroids
}

func (e *IVFEngine) assignToCentroids(centroids [][]float32, assignments []int) {
	n := len(e.vectorIDs)
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
				vec := e.getVector(i)
				minDist := float32(math.MaxFloat32)
				minIdx := 0
				for c, centroid := range centroids {
					// Use SIMD for distance computation
					d := vek32.Distance(vec, centroid)
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

	for i := range e.vectorIDs {
		c := assignments[i]
		counts[c]++
		vec := e.getVector(i)
		for d := 0; d < dims; d++ {
			newCentroids[c][d] += vec[d]
		}
	}

	for c := 0; c < k; c++ {
		if counts[c] > 0 {
			invCount := 1.0 / float32(counts[c])
			for d := 0; d < dims; d++ {
				newCentroids[c][d] *= invCount
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

	// Precompute query norm for cosine distance
	queryNorm := vek32.Norm(query)

	// No centroids = flat search
	if e.centroids == nil {
		return e.searchClusterSIMD(e.clusters[0], query, queryNorm, k)
	}

	// Find nearest clusters using SIMD distance
	nProbe := e.nProbe
	if nProbe > len(e.centroids) {
		nProbe = len(e.centroids)
	}

	// Compute all centroid distances at once for better cache locality
	centroidDists := make([]float32, len(e.centroids))
	for i, centroid := range e.centroids {
		centroidDists[i] = e.computeDistanceSIMD(query, queryNorm, centroid, e.centroidNorms[i])
	}

	// Find nProbe nearest centroids using partial selection
	clusterIndices := e.selectTopK(centroidDists, nProbe)

	// Count total vectors to search
	totalVecs := 0
	for _, ci := range clusterIndices {
		totalVecs += len(e.clusters[ci])
	}

	// Use parallel search only for large total vector counts
	if totalVecs > ivfParallelThresh*2 && len(clusterIndices) >= 2 {
		return e.searchClustersParallelSIMD(clusterIndices, query, queryNorm, k)
	}

	// Serial search with SIMD for smaller datasets
	return e.searchClustersMergedSIMD(clusterIndices, query, queryNorm, k)
}

// computeDistanceSIMD computes distance based on metric using SIMD
func (e *IVFEngine) computeDistanceSIMD(a []float32, normA float32, b []float32, normB float32) float32 {
	switch e.metric {
	case vectorize.Cosine:
		if normA == 0 || normB == 0 {
			return 1.0
		}
		dot := vek32.Dot(a, b)
		return 1.0 - dot/(normA*normB)
	case vectorize.Euclidean:
		return vek32.Distance(a, b)
	case vectorize.DotProduct:
		return -vek32.Dot(a, b)
	default:
		if normA == 0 || normB == 0 {
			return 1.0
		}
		dot := vek32.Dot(a, b)
		return 1.0 - dot/(normA*normB)
	}
}

// selectTopK selects indices of k smallest values using partial quickselect
func (e *IVFEngine) selectTopK(dists []float32, k int) []int32 {
	n := len(dists)
	if k >= n {
		result := make([]int32, n)
		for i := 0; i < n; i++ {
			result[i] = int32(i)
		}
		return result
	}

	// Use max-heap for top-k selection
	h := make(maxHeap32, 0, k)
	for i, d := range dists {
		if len(h) < k {
			h.PushItem(distItem32{idx: int32(i), dist: d})
		} else if d < h[0].dist {
			h.PopItem()
			h.PushItem(distItem32{idx: int32(i), dist: d})
		}
	}

	// Extract indices
	result := make([]int32, len(h))
	for i := range h {
		result[i] = h[i].idx
	}
	return result
}

// searchClustersMergedSIMD searches clusters serially with SIMD distance
func (e *IVFEngine) searchClustersMergedSIMD(clusterIndices []int32, query []float32, queryNorm float32, k int) []SearchResult {
	resultHeap := make(maxHeap32, 0, k)

	for _, ci := range clusterIndices {
		cluster := e.clusters[ci]

		for _, vecIdx := range cluster {
			vec := e.getVector(int(vecIdx))
			dist := e.computeDistanceSIMD(query, queryNorm, vec, e.vectorNorms[vecIdx])

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
			ID:       e.vectorIDs[item.idx],
			Distance: item.dist,
		}
	}

	return results
}

// searchClustersParallelSIMD searches clusters in parallel with SIMD
func (e *IVFEngine) searchClustersParallelSIMD(clusterIndices []int32, query []float32, queryNorm float32, k int) []SearchResult {
	nClusters := len(clusterIndices)
	resultsChan := make(chan []distItem32, nClusters)
	var wg sync.WaitGroup

	for _, ci := range clusterIndices {
		cluster := e.clusters[ci]
		if len(cluster) == 0 {
			continue
		}

		wg.Add(1)
		go func(cluster []int32) {
			defer wg.Done()
			// Use top-K heap for each cluster
			clusterHeap := make(maxHeap32, 0, k)
			for _, vecIdx := range cluster {
				vec := e.getVector(int(vecIdx))
				dist := e.computeDistanceSIMD(query, queryNorm, vec, e.vectorNorms[vecIdx])

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
			ID:       e.vectorIDs[item.idx],
			Distance: item.dist,
		}
	}

	return results
}

// searchClusterSIMD searches a single cluster using SIMD distance
func (e *IVFEngine) searchClusterSIMD(cluster []int32, query []float32, queryNorm float32, k int) []SearchResult {
	resultHeap := make(maxHeap32, 0, k)

	for _, vecIdx := range cluster {
		vec := e.getVector(int(vecIdx))
		dist := e.computeDistanceSIMD(query, queryNorm, vec, e.vectorNorms[vecIdx])

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
			ID:       e.vectorIDs[item.idx],
			Distance: item.dist,
		}
	}

	return results
}

func (e *IVFEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *IVFEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
