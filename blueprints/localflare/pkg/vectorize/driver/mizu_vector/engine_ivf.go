package mizu_vector

import (
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// IVFEngine implements Inverted File Index with k-means clustering.
// Time complexity: O(sqrt(n)*d) per query.
// Based on "Product quantization for nearest neighbor search" (Jégou et al., 2011).
type IVFEngine struct {
	distFunc DistanceFunc

	// Index structures
	centroids [][]float32       // [nClusters][dims]
	clusters  [][]indexedVector // Vectors in each cluster
	nProbe    int               // Number of clusters to search

	needsRebuild bool
}

// IVF configuration
const (
	ivfMinVectors     = 256
	ivfClustersPerSqrt = 4
	ivfMaxClusters    = 256
	ivfKMeansIters    = 10
	ivfDefaultNProbe  = 8
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
	if n < ivfMinVectors {
		// Fall back to flat storage for small datasets
		e.buildFlat(vectors)
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

	// Collect vectors
	vectorList := make([]indexedVector, 0, n)
	for id, v := range vectors {
		vectorList = append(vectorList, indexedVector{id: id, values: v.Values})
	}

	// K-means++ initialization
	centroids := e.kmeansppInit(vectorList, nClusters, dims)

	// Run k-means
	assignments := make([]int, n)
	for iter := 0; iter < ivfKMeansIters; iter++ {
		e.assignToCentroids(vectorList, centroids, assignments)
		centroids = e.updateCentroids(vectorList, assignments, nClusters, dims)
	}

	// Build cluster structure
	clusters := make([][]indexedVector, nClusters)
	for i := range clusters {
		clusters[i] = make([]indexedVector, 0)
	}

	for i, v := range vectorList {
		cluster := assignments[i]
		clusters[cluster] = append(clusters[cluster], v)
	}

	e.centroids = centroids
	e.clusters = clusters
	e.needsRebuild = false
}

func (e *IVFEngine) buildFlat(vectors map[string]*vectorize.Vector) {
	// Single cluster for small datasets
	e.centroids = nil
	e.clusters = [][]indexedVector{make([]indexedVector, 0, len(vectors))}
	for id, v := range vectors {
		e.clusters[0] = append(e.clusters[0], indexedVector{id: id, values: v.Values})
	}
	e.needsRebuild = false
}

func (e *IVFEngine) kmeansppInit(vectors []indexedVector, k, dims int) [][]float32 {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	centroids := make([][]float32, k)

	// First centroid: random
	firstIdx := rng.Intn(len(vectors))
	centroids[0] = make([]float32, dims)
	copy(centroids[0], vectors[firstIdx].values)

	distances := make([]float32, len(vectors))

	for i := 1; i < k; i++ {
		var totalDist float32

		// Distance to nearest centroid
		for j, v := range vectors {
			minDist := float32(math.MaxFloat32)
			for c := 0; c < i; c++ {
				d := e.distFunc(v.values, centroids[c])
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
					copy(centroids[i], vectors[j].values)
					break
				}
			}
		}

		if centroids[i] == nil {
			centroids[i] = make([]float32, dims)
			copy(centroids[i], vectors[rng.Intn(len(vectors))].values)
		}
	}

	return centroids
}

func (e *IVFEngine) assignToCentroids(vectors []indexedVector, centroids [][]float32, assignments []int) {
	nWorkers := runtime.NumCPU()
	chunkSize := (len(vectors) + nWorkers - 1) / nWorkers

	var wg sync.WaitGroup
	for w := 0; w < nWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(vectors) {
			end = len(vectors)
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
					d := e.distFunc(vectors[i].values, centroid)
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

func (e *IVFEngine) updateCentroids(vectors []indexedVector, assignments []int, k, dims int) [][]float32 {
	newCentroids := make([][]float32, k)
	counts := make([]int, k)

	for i := range newCentroids {
		newCentroids[i] = make([]float32, dims)
	}

	for i, v := range vectors {
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
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	for i := range e.clusters {
		filtered := make([]indexedVector, 0, len(e.clusters[i]))
		for _, v := range e.clusters[i] {
			if _, deleted := idSet[v.id]; !deleted {
				filtered = append(filtered, v)
			}
		}
		e.clusters[i] = filtered
	}
}

func (e *IVFEngine) Search(query []float32, k int) []SearchResult {
	if len(e.clusters) == 0 {
		return nil
	}

	// No centroids = flat search
	if e.centroids == nil {
		return e.searchCluster(e.clusters[0], query, k)
	}

	// Find nearest clusters
	nProbe := e.nProbe
	if nProbe > len(e.centroids) {
		nProbe = len(e.centroids)
	}

	type clusterDist struct {
		idx  int
		dist float32
	}

	clusterDists := make([]clusterDist, len(e.centroids))
	for i, centroid := range e.centroids {
		clusterDists[i] = clusterDist{i, e.distFunc(query, centroid)}
	}

	sort.Slice(clusterDists, func(i, j int) bool {
		return clusterDists[i].dist < clusterDists[j].dist
	})

	// Search in parallel across top clusters
	resultsChan := make(chan []SearchResult, nProbe)
	var wg sync.WaitGroup

	for i := 0; i < nProbe; i++ {
		cluster := e.clusters[clusterDists[i].idx]
		if len(cluster) == 0 {
			continue
		}

		wg.Add(1)
		go func(cluster []indexedVector) {
			defer wg.Done()
			resultsChan <- e.searchCluster(cluster, query, k)
		}(cluster)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Merge results
	allResults := make([]SearchResult, 0)
	for results := range resultsChan {
		allResults = append(allResults, results...)
	}

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Distance < allResults[j].Distance
	})

	if k > len(allResults) {
		k = len(allResults)
	}

	return allResults[:k]
}

func (e *IVFEngine) searchCluster(cluster []indexedVector, query []float32, k int) []SearchResult {
	results := make([]SearchResult, 0, len(cluster))
	for _, v := range cluster {
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

func (e *IVFEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *IVFEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
