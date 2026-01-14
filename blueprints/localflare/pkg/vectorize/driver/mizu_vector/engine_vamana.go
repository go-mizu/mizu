package mizu_vector

import (
	"container/heap"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// VamanaEngine implements the Vamana graph algorithm from DiskANN.
// Time complexity: O(log n) per query with high recall.
// Based on "DiskANN: Fast Accurate Billion-point Nearest Neighbor Search on a Single Node" (Subramanya et al., 2019).
//
// Key differences from HNSW:
// - Single-level graph (simpler, disk-friendly)
// - Robust pruning with alpha parameter for diversity
// - Two-pass construction for better graph quality
type VamanaEngine struct {
	distFunc DistanceFunc

	// Vamana parameters
	R     int     // Max out-degree (connections per node)
	L     int     // Search list size during construction
	alpha float32 // Pruning parameter (1.0 = no pruning, 1.2 typical)

	// Graph structure
	nodes      map[string]*vamanaNode
	nodeList   []string // For random access
	medoid     string   // Entry point (medoid of dataset)

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

type vamanaNode struct {
	id       string
	values   []float32
	neighbors []string
}

// Vamana configuration
const (
	vamanaDefaultR     = 32
	vamanaDefaultL     = 100
	vamanaDefaultAlpha = 1.2
)

// NewVamanaEngine creates a new Vamana search engine.
func NewVamanaEngine(distFunc DistanceFunc) *VamanaEngine {
	return &VamanaEngine{
		distFunc:     distFunc,
		R:            vamanaDefaultR,
		L:            vamanaDefaultL,
		alpha:        vamanaDefaultAlpha,
		nodes:        make(map[string]*vamanaNode),
		needsRebuild: true,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *VamanaEngine) Name() string { return "vamana" }

func (e *VamanaEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
	e.mu.Lock()
	defer e.mu.Unlock()

	n := len(vectors)
	if n == 0 {
		e.needsRebuild = false
		return
	}

	// Initialize nodes
	e.nodes = make(map[string]*vamanaNode, n)
	e.nodeList = make([]string, 0, n)

	for id, v := range vectors {
		e.nodes[id] = &vamanaNode{
			id:       id,
			values:   v.Values,
			neighbors: make([]string, 0, e.R),
		}
		e.nodeList = append(e.nodeList, id)
	}

	// Find medoid (approximate centroid)
	e.medoid = e.findMedoid()

	// Initialize with random graph
	e.initializeRandomGraph()

	// Two-pass construction
	// Pass 1: Build initial graph
	e.buildPass()

	// Pass 2: Refine graph (with shuffled order)
	e.rng.Shuffle(len(e.nodeList), func(i, j int) {
		e.nodeList[i], e.nodeList[j] = e.nodeList[j], e.nodeList[i]
	})
	e.buildPass()

	e.needsRebuild = false
}

// findMedoid finds the approximate medoid of the dataset.
func (e *VamanaEngine) findMedoid() string {
	if len(e.nodeList) == 0 {
		return ""
	}

	// Sample-based medoid finding for efficiency
	sampleSize := 100
	if sampleSize > len(e.nodeList) {
		sampleSize = len(e.nodeList)
	}

	// Random sample
	sample := make([]string, sampleSize)
	perm := e.rng.Perm(len(e.nodeList))
	for i := 0; i < sampleSize; i++ {
		sample[i] = e.nodeList[perm[i]]
	}

	// Find vector with minimum total distance to others in sample
	minTotalDist := float32(math.MaxFloat32)
	medoid := sample[0]

	for _, id := range sample {
		node := e.nodes[id]
		var totalDist float32
		for _, otherId := range sample {
			if id != otherId {
				other := e.nodes[otherId]
				totalDist += e.distFunc(node.values, other.values)
			}
		}
		if totalDist < minTotalDist {
			minTotalDist = totalDist
			medoid = id
		}
	}

	return medoid
}

// initializeRandomGraph creates initial random connections.
func (e *VamanaEngine) initializeRandomGraph() {
	for id, node := range e.nodes {
		// Add random neighbors
		numNeighbors := e.R / 2
		if numNeighbors < 1 {
			numNeighbors = 1
		}

		perm := e.rng.Perm(len(e.nodeList))
		for i := 0; i < len(perm) && len(node.neighbors) < numNeighbors; i++ {
			neighborID := e.nodeList[perm[i]]
			if neighborID != id {
				node.neighbors = append(node.neighbors, neighborID)
			}
		}
	}
}

// buildPass performs one pass of graph construction.
func (e *VamanaEngine) buildPass() {
	for _, id := range e.nodeList {
		node := e.nodes[id]

		// GreedySearch from medoid
		candidates := e.greedySearch(node.values, e.L)

		// RobustPrune: select diverse neighbors
		newNeighbors := e.robustPrune(node.values, candidates, e.R)
		node.neighbors = newNeighbors

		// Add reverse edges
		for _, neighborID := range newNeighbors {
			if neighbor, ok := e.nodes[neighborID]; ok {
				// Check if already connected
				found := false
				for _, nid := range neighbor.neighbors {
					if nid == id {
						found = true
						break
					}
				}

				if !found {
					neighbor.neighbors = append(neighbor.neighbors, id)

					// Prune if over capacity
					if len(neighbor.neighbors) > e.R {
						neighbor.neighbors = e.robustPrune(neighbor.values, neighbor.neighbors, e.R)
					}
				}
			}
		}
	}
}

// greedySearch performs greedy beam search from medoid.
func (e *VamanaEngine) greedySearch(query []float32, L int) []string {
	if e.medoid == "" {
		return nil
	}

	visited := make(map[string]bool)
	candidates := &minDistHeap{}
	result := make([]candDist, 0, L)

	heap.Init(candidates)

	// Start from medoid
	medoidNode := e.nodes[e.medoid]
	dist := e.distFunc(query, medoidNode.values)
	heap.Push(candidates, candDist{id: e.medoid, dist: dist})
	visited[e.medoid] = true

	for candidates.Len() > 0 {
		curr := heap.Pop(candidates).(candDist)
		result = append(result, curr)

		// Stop if we have enough
		if len(result) >= L {
			break
		}

		// Explore neighbors
		if node, ok := e.nodes[curr.id]; ok {
			for _, neighborID := range node.neighbors {
				if visited[neighborID] {
					continue
				}
				visited[neighborID] = true

				if neighbor, ok := e.nodes[neighborID]; ok {
					dist := e.distFunc(query, neighbor.values)
					heap.Push(candidates, candDist{id: neighborID, dist: dist})
				}
			}
		}
	}

	// Extract IDs
	ids := make([]string, len(result))
	for i, c := range result {
		ids[i] = c.id
	}
	return ids
}

// robustPrune implements Vamana's robust pruning with alpha parameter.
func (e *VamanaEngine) robustPrune(query []float32, candidates []string, R int) []string {
	if len(candidates) <= R {
		return candidates
	}

	// Sort candidates by distance
	dists := make([]candDist, 0, len(candidates))
	for _, id := range candidates {
		if node, ok := e.nodes[id]; ok {
			dists = append(dists, candDist{id: id, dist: e.distFunc(query, node.values)})
		}
	}

	sort.Slice(dists, func(i, j int) bool {
		return dists[i].dist < dists[j].dist
	})

	// Greedy selection with diversity (alpha pruning)
	selected := make([]string, 0, R)

	for _, cand := range dists {
		if len(selected) >= R {
			break
		}

		// Check if cand is diverse enough from already selected
		keep := true
		candNode := e.nodes[cand.id]

		for _, selID := range selected {
			selNode := e.nodes[selID]
			distToSel := e.distFunc(candNode.values, selNode.values)

			// If distance to selected neighbor * alpha < distance to query, skip
			if distToSel*e.alpha < cand.dist {
				keep = false
				break
			}
		}

		if keep {
			selected = append(selected, cand.id)
		}
	}

	return selected
}

type candDist struct {
	id   string
	dist float32
}

type minDistHeap []candDist

func (h minDistHeap) Len() int           { return len(h) }
func (h minDistHeap) Less(i, j int) bool { return h[i].dist < h[j].dist }
func (h minDistHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minDistHeap) Push(x any)        { *h = append(*h, x.(candDist)) }
func (h *minDistHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

func (e *VamanaEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *VamanaEngine) Delete(ids []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
		delete(e.nodes, id)
	}

	// Clean up references
	for _, node := range e.nodes {
		filtered := make([]string, 0, len(node.neighbors))
		for _, nid := range node.neighbors {
			if _, deleted := idSet[nid]; !deleted {
				filtered = append(filtered, nid)
			}
		}
		node.neighbors = filtered
	}

	// Update node list
	filtered := make([]string, 0, len(e.nodeList))
	for _, id := range e.nodeList {
		if _, deleted := idSet[id]; !deleted {
			filtered = append(filtered, id)
		}
	}
	e.nodeList = filtered

	// Update medoid if deleted
	if _, deleted := idSet[e.medoid]; deleted {
		e.medoid = e.findMedoid()
	}
}

func (e *VamanaEngine) Search(query []float32, k int) []SearchResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.nodes) == 0 {
		return nil
	}

	L := e.L
	if L < k*2 {
		L = k * 2
	}

	candidates := e.greedySearch(query, L)

	// Compute exact distances and return top k
	results := make([]SearchResult, 0, len(candidates))
	for _, id := range candidates {
		if node, ok := e.nodes[id]; ok {
			dist := e.distFunc(query, node.values)
			results = append(results, SearchResult{ID: id, Distance: dist})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if k > len(results) {
		k = len(results)
	}

	return results[:k]
}

func (e *VamanaEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *VamanaEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }
