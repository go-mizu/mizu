package mizu_vector

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// ScaNNEngine implements a simplified ScaNN-style search.
// Uses partitioning + asymmetric distance computation.
type ScaNNEngine struct {
	distFunc DistanceFunc

	// Vector storage
	vectors []indexedVector

	// Partitioner (K-means clustering)
	centroids [][]float32
	clusters  [][]int32

	// Parameters
	nClusters int
	nProbe    int
	reorderK  int

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

// ScaNN configuration
const (
	scannDefaultClusters = 64
	scannDefaultNprobe   = 8
	scannDefaultReorder  = 50
)

// NewScaNNEngine creates a new ScaNN search engine.
func NewScaNNEngine(distFunc DistanceFunc) *ScaNNEngine {
	return &ScaNNEngine{
		distFunc:     distFunc,
		nClusters:    scannDefaultClusters,
		nProbe:       scannDefaultNprobe,
		reorderK:     scannDefaultReorder,
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

	// Collect vectors
	e.vectors = make([]indexedVector, 0, n)
	for id, v := range vectors {
		e.vectors = append(e.vectors, indexedVector{id: id, values: v.Values})
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

// buildPartitioner creates K-means clustering.
func (e *ScaNNEngine) buildPartitioner(dims, numClusters int) {
	n := len(e.vectors)

	// K-means++ initialization
	e.centroids = make([][]float32, numClusters)
	used := make([]bool, n)

	// First centroid: random
	first := e.rng.Intn(n)
	e.centroids[0] = make([]float32, dims)
	copy(e.centroids[0], e.vectors[first].values)
	used[first] = true

	// Remaining centroids using D² sampling
	minDists := make([]float32, n)
	for i := range minDists {
		minDists[i] = float32(math.MaxFloat32)
	}

	for c := 1; c < numClusters; c++ {
		// Update min distances
		prevCentroid := e.centroids[c-1]
		var totalDist float32
		for i := 0; i < n; i++ {
			if used[i] {
				minDists[i] = 0
				continue
			}
			d := e.distFunc(e.vectors[i].values, prevCentroid)
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
		copy(e.centroids[c], e.vectors[selectedIdx].values)
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

	// K-means iterations
	assignments := make([]int, n)
	for iter := 0; iter < 10; iter++ {
		// Assign vectors to nearest centroid
		for i := 0; i < n; i++ {
			bestC := 0
			bestDist := e.distFunc(e.vectors[i].values, e.centroids[0])
			for c := 1; c < numClusters; c++ {
				d := e.distFunc(e.vectors[i].values, e.centroids[c])
				if d < bestDist {
					bestDist = d
					bestC = c
				}
			}
			assignments[i] = bestC
		}

		// Update centroids
		counts := make([]int, numClusters)
		newCentroids := make([][]float32, numClusters)
		for c := range newCentroids {
			newCentroids[c] = make([]float32, dims)
		}

		for i := 0; i < n; i++ {
			c := assignments[i]
			counts[c]++
			for j := 0; j < dims; j++ {
				newCentroids[c][j] += e.vectors[i].values[j]
			}
		}

		for c := 0; c < numClusters; c++ {
			if counts[c] > 0 {
				for j := 0; j < dims; j++ {
					e.centroids[c][j] = newCentroids[c][j] / float32(counts[c])
				}
			}
		}
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

	if len(e.vectors) == 0 {
		return nil
	}

	// Find nearest clusters
	type clusterDist struct {
		idx  int
		dist float32
	}
	clusterDists := make([]clusterDist, len(e.centroids))
	for i, c := range e.centroids {
		clusterDists[i] = clusterDist{idx: i, dist: e.distFunc(query, c)}
	}
	sort.Slice(clusterDists, func(i, j int) bool {
		return clusterDists[i].dist < clusterDists[j].dist
	})

	// Collect candidates from top clusters
	nprobe := e.nProbe
	if nprobe > len(clusterDists) {
		nprobe = len(clusterDists)
	}

	type candidate struct {
		idx  int32
		dist float32
	}
	candidates := make([]candidate, 0)

	for p := 0; p < nprobe; p++ {
		clusterIdx := clusterDists[p].idx
		for _, vecIdx := range e.clusters[clusterIdx] {
			dist := e.distFunc(query, e.vectors[vecIdx].values)
			candidates = append(candidates, candidate{idx: vecIdx, dist: dist})
		}
	}

	// Sort and return top k
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dist < candidates[j].dist
	})

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

func (e *ScaNNEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *ScaNNEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
