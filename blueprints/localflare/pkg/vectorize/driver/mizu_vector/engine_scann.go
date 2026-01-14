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

// ScaNNEngine implements a simplified ScaNN-style search.
// Uses partitioning + asymmetric distance computation.
//
// Optimized with:
// - SIMD-accelerated distance computation using viterin/vek
// - Contiguous memory layout for cache efficiency
// - Precomputed L2 norms for faster cosine distance
// - Parallel cluster search for large datasets
// - Typed heaps instead of sorting
type ScaNNEngine struct {
	distFunc DistanceFunc
	metric   vectorize.DistanceMetric

	// Contiguous memory layout for cache efficiency
	vectorData  []float32 // All vector values: [v0d0, v0d1, ..., v1d0, ...]
	vectorIDs   []string  // Vector IDs indexed by int32
	vectorNorms []float32 // Precomputed L2 norms for cosine distance
	dims        int

	// Partitioner (K-means clustering)
	centroids     [][]float32
	centroidNorms []float32
	clusters      [][]int32

	// Parameters
	nClusters int
	nProbe    int

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

// ScaNN configuration
const (
	scannDefaultClusters = 64
	scannDefaultNprobe   = 10 // Increased for better recall
	scannParallelThresh  = 200
)

// NewScaNNEngine creates a new ScaNN search engine.
func NewScaNNEngine(distFunc DistanceFunc) *ScaNNEngine {
	return &ScaNNEngine{
		distFunc:     distFunc,
		nClusters:    scannDefaultClusters,
		nProbe:       scannDefaultNprobe,
		needsRebuild: true,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *ScaNNEngine) Name() string { return "scann" }

func (e *ScaNNEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
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

	// Build partitioner
	numClusters := e.nClusters
	if numClusters > n {
		numClusters = n
	}
	if numClusters < 1 {
		numClusters = 1
	}

	e.buildPartitioner(dims, numClusters)
	e.needsRebuild = false
}

// getVector returns vector data at index using contiguous storage
func (e *ScaNNEngine) getVector(idx int) []float32 {
	start := idx * e.dims
	return e.vectorData[start : start+e.dims]
}

// computeDistanceSIMD computes distance based on metric using SIMD
func (e *ScaNNEngine) computeDistanceSIMD(a []float32, normA float32, b []float32, normB float32) float32 {
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

// buildPartitioner creates K-means clustering with parallel assignment.
func (e *ScaNNEngine) buildPartitioner(dims, numClusters int) {
	n := len(e.vectorIDs)

	// K-means++ initialization using SIMD
	e.centroids = make([][]float32, numClusters)
	used := make([]bool, n)

	// First centroid: random
	first := e.rng.Intn(n)
	e.centroids[0] = make([]float32, dims)
	copy(e.centroids[0], e.getVector(first))
	used[first] = true

	// Remaining centroids using D² sampling
	minDists := make([]float32, n)
	for i := range minDists {
		minDists[i] = float32(math.MaxFloat32)
	}

	for c := 1; c < numClusters; c++ {
		// Update min distances using SIMD
		prevCentroid := e.centroids[c-1]
		var totalDist float32
		for i := 0; i < n; i++ {
			if used[i] {
				minDists[i] = 0
				continue
			}
			d := vek32.Distance(e.getVector(i), prevCentroid)
			if d < minDists[i] {
				minDists[i] = d
			}
			totalDist += minDists[i]
		}

		if totalDist == 0 {
			break
		}

		// Sample proportional to distance
		target := e.rng.Float32() * totalDist
		var cumsum float32
		selectedIdx := 0
		for i := 0; i < n; i++ {
			cumsum += minDists[i]
			if cumsum >= target {
				selectedIdx = i
				break
			}
		}

		e.centroids[c] = make([]float32, dims)
		copy(e.centroids[c], e.getVector(selectedIdx))
		used[selectedIdx] = true
	}

	// Remove nil centroids
	validCentroids := make([][]float32, 0, numClusters)
	for _, c := range e.centroids {
		if c != nil {
			validCentroids = append(validCentroids, c)
		}
	}
	e.centroids = validCentroids
	numClusters = len(e.centroids)

	// Parallel k-means iterations
	assignments := make([]int, n)
	for iter := 0; iter < 10; iter++ {
		e.parallelAssign(assignments, numClusters)
		e.updateCentroids(assignments, numClusters, dims)
	}

	// Precompute centroid norms
	e.centroidNorms = make([]float32, numClusters)
	for i, c := range e.centroids {
		e.centroidNorms[i] = vek32.Norm(c)
	}

	// Build cluster lists
	e.clusters = make([][]int32, numClusters)
	for c := range e.clusters {
		e.clusters[c] = make([]int32, 0)
	}
	for i := 0; i < n; i++ {
		c := assignments[i]
		e.clusters[c] = append(e.clusters[c], int32(i))
	}
}

func (e *ScaNNEngine) parallelAssign(assignments []int, numClusters int) {
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
				bestC := 0
				bestDist := vek32.Distance(vec, e.centroids[0])
				for c := 1; c < numClusters; c++ {
					d := vek32.Distance(vec, e.centroids[c])
					if d < bestDist {
						bestDist = d
						bestC = c
					}
				}
				assignments[i] = bestC
			}
		}(start, end)
	}
	wg.Wait()
}

func (e *ScaNNEngine) updateCentroids(assignments []int, numClusters, dims int) {
	counts := make([]int, numClusters)
	newCentroids := make([][]float32, numClusters)
	for c := range newCentroids {
		newCentroids[c] = make([]float32, dims)
	}

	for i := range e.vectorIDs {
		c := assignments[i]
		counts[c]++
		vec := e.getVector(i)
		for j := 0; j < dims; j++ {
			newCentroids[c][j] += vec[j]
		}
	}

	for c := 0; c < numClusters; c++ {
		if counts[c] > 0 {
			invCount := 1.0 / float32(counts[c])
			for j := 0; j < dims; j++ {
				e.centroids[c][j] = newCentroids[c][j] * invCount
			}
		}
	}
}

func (e *ScaNNEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *ScaNNEngine) Delete(ids []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.needsRebuild = true
}

func (e *ScaNNEngine) Search(query []float32, k int) []SearchResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.vectorIDs) == 0 {
		return nil
	}

	// Precompute query norm
	queryNorm := vek32.Norm(query)

	// Find nearest clusters using SIMD distance
	nprobe := e.nProbe
	if nprobe > len(e.centroids) {
		nprobe = len(e.centroids)
	}

	// Compute all centroid distances at once
	centroidDists := make([]float32, len(e.centroids))
	for i, c := range e.centroids {
		centroidDists[i] = e.computeDistanceSIMD(query, queryNorm, c, e.centroidNorms[i])
	}

	// Select nprobe nearest centroids
	topClusters := make(maxHeap32, 0, nprobe)
	for i, dist := range centroidDists {
		if len(topClusters) < nprobe {
			topClusters.PushItem(distItem32{idx: int32(i), dist: dist})
		} else if dist < topClusters[0].dist {
			topClusters.PopItem()
			topClusters.PushItem(distItem32{idx: int32(i), dist: dist})
		}
	}

	// Count total vectors
	totalVecs := 0
	clusterIndices := make([]int32, len(topClusters))
	for i := range topClusters {
		clusterIndices[i] = topClusters[i].idx
		totalVecs += len(e.clusters[topClusters[i].idx])
	}

	// Use parallel search for large datasets
	if totalVecs > scannParallelThresh*2 && len(clusterIndices) >= 2 {
		return e.searchParallel(clusterIndices, query, queryNorm, k)
	}

	// Serial search for smaller datasets
	return e.searchSerial(clusterIndices, query, queryNorm, k)
}

func (e *ScaNNEngine) searchSerial(clusterIndices []int32, query []float32, queryNorm float32, k int) []SearchResult {
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

func (e *ScaNNEngine) searchParallel(clusterIndices []int32, query []float32, queryNorm float32, k int) []SearchResult {
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

	// Merge results
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

func (e *ScaNNEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *ScaNNEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
