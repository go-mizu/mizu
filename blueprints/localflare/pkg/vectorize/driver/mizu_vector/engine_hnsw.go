package mizu_vector

import (
	"container/heap"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// HNSWEngine implements Hierarchical Navigable Small World graph.
// Time complexity: O(log n) per query.
// Based on "Efficient and robust approximate nearest neighbor search using HNSW" (Malkov & Yashunin, 2018).
type HNSWEngine struct {
	distFunc DistanceFunc

	// HNSW parameters
	M        int     // Max connections per node
	Ml       float64 // Level generation factor
	efSearch int     // Search queue size

	// Graph structure
	nodes      map[string]*hnswNode
	entryPoint string
	maxLevel   int

	needsRebuild bool
	rng          *rand.Rand
	mu           sync.RWMutex
}

type hnswNode struct {
	id      string
	values  []float32
	level   int
	friends [][]string // friends[level] = list of neighbor IDs
}

// HNSW configuration
const (
	hnswDefaultM        = 16
	hnswDefaultMl       = 1.0 / math.Ln2
	hnswDefaultEfSearch = 64
	hnswDefaultEfConstr = 200
)

// NewHNSWEngine creates a new HNSW search engine.
func NewHNSWEngine(distFunc DistanceFunc) *HNSWEngine {
	return &HNSWEngine{
		distFunc:     distFunc,
		M:            hnswDefaultM,
		Ml:           hnswDefaultMl,
		efSearch:     hnswDefaultEfSearch,
		nodes:        make(map[string]*hnswNode),
		needsRebuild: true,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *HNSWEngine) Name() string { return "hnsw" }

func (e *HNSWEngine) Build(vectors map[string]*vectorize.Vector, dims int, metric vectorize.DistanceMetric) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.nodes = make(map[string]*hnswNode)
	e.entryPoint = ""
	e.maxLevel = 0

	// Insert all vectors
	for id, v := range vectors {
		e.insertNode(id, v.Values)
	}

	e.needsRebuild = false
}

// randomLevel generates a random level for a new node.
func (e *HNSWEngine) randomLevel() int {
	r := e.rng.Float64()
	return int(-math.Log(r) * e.Ml)
}

// insertNode inserts a node into the HNSW graph.
func (e *HNSWEngine) insertNode(id string, values []float32) {
	level := e.randomLevel()

	node := &hnswNode{
		id:      id,
		values:  values,
		level:   level,
		friends: make([][]string, level+1),
	}
	for i := range node.friends {
		node.friends[i] = make([]string, 0, e.M)
	}

	e.nodes[id] = node

	if e.entryPoint == "" {
		e.entryPoint = id
		e.maxLevel = level
		return
	}

	// Search for entry point at each level
	currNode := e.entryPoint
	currDist := e.distFunc(values, e.nodes[currNode].values)

	// Greedy search from top level to level+1
	for l := e.maxLevel; l > level; l-- {
		changed := true
		for changed {
			changed = false
			if node, ok := e.nodes[currNode]; ok && l < len(node.friends) {
				for _, friendID := range node.friends[l] {
					if friend, ok := e.nodes[friendID]; ok {
						dist := e.distFunc(values, friend.values)
						if dist < currDist {
							currNode = friendID
							currDist = dist
							changed = true
						}
					}
				}
			}
		}
	}

	// Insert at each level from level down to 0
	for l := min(level, e.maxLevel); l >= 0; l-- {
		// Search for neighbors at this level
		neighbors := e.searchLevel(values, currNode, hnswDefaultEfConstr, l)

		// Select M best neighbors
		selected := e.selectNeighbors(values, neighbors, e.M)

		// Add bidirectional connections
		node.friends[l] = selected
		for _, neighborID := range selected {
			if neighbor, ok := e.nodes[neighborID]; ok && l < len(neighbor.friends) {
				neighbor.friends[l] = append(neighbor.friends[l], id)

				// Prune if too many connections
				if len(neighbor.friends[l]) > e.M*2 {
					neighbor.friends[l] = e.selectNeighbors(neighbor.values, neighbor.friends[l], e.M)
				}
			}
		}

		if len(neighbors) > 0 {
			currNode = neighbors[0]
		}
	}

	// Update entry point if new node has higher level
	if level > e.maxLevel {
		e.maxLevel = level
		e.entryPoint = id
	}
}

// searchLevel performs beam search at a specific level.
func (e *HNSWEngine) searchLevel(query []float32, entryID string, ef, level int) []string {
	visited := make(map[string]bool)
	candidates := &distHeap{}
	result := &distHeap{}

	heap.Init(candidates)
	heap.Init(result)

	if entry, ok := e.nodes[entryID]; ok {
		dist := e.distFunc(query, entry.values)
		heap.Push(candidates, distItem{id: entryID, dist: dist, isMax: false})
		heap.Push(result, distItem{id: entryID, dist: dist, isMax: true})
		visited[entryID] = true
	}

	for candidates.Len() > 0 {
		curr := heap.Pop(candidates).(distItem)

		// Stop if current candidate is worse than worst result
		if result.Len() >= ef {
			worst := (*result)[0]
			if curr.dist > worst.dist {
				break
			}
		}

		// Explore neighbors
		if node, ok := e.nodes[curr.id]; ok && level < len(node.friends) {
			for _, neighborID := range node.friends[level] {
				if visited[neighborID] {
					continue
				}
				visited[neighborID] = true

				if neighbor, ok := e.nodes[neighborID]; ok {
					dist := e.distFunc(query, neighbor.values)

					if result.Len() < ef {
						heap.Push(candidates, distItem{id: neighborID, dist: dist, isMax: false})
						heap.Push(result, distItem{id: neighborID, dist: dist, isMax: true})
					} else if dist < (*result)[0].dist {
						heap.Push(candidates, distItem{id: neighborID, dist: dist, isMax: false})
						heap.Pop(result)
						heap.Push(result, distItem{id: neighborID, dist: dist, isMax: true})
					}
				}
			}
		}
	}

	// Extract result IDs sorted by distance
	results := make([]string, result.Len())
	for i := result.Len() - 1; i >= 0; i-- {
		results[i] = heap.Pop(result).(distItem).id
	}

	return results
}

// selectNeighbors selects the M best neighbors using simple heuristic.
func (e *HNSWEngine) selectNeighbors(query []float32, candidates []string, M int) []string {
	if len(candidates) <= M {
		return candidates
	}

	// Use heap for efficient top-M selection
	h := &maxDistHeapHNSW{}
	heap.Init(h)

	for _, id := range candidates {
		if node, ok := e.nodes[id]; ok {
			dist := e.distFunc(query, node.values)
			if h.Len() < M {
				heap.Push(h, hnswDistItem{id: id, dist: dist})
			} else if dist < (*h)[0].dist {
				heap.Pop(h)
				heap.Push(h, hnswDistItem{id: id, dist: dist})
			}
		}
	}

	selected := make([]string, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		selected[i] = heap.Pop(h).(hnswDistItem).id
	}

	return selected
}

type hnswDistItem struct {
	id   string
	dist float32
}

type maxDistHeapHNSW []hnswDistItem

func (h maxDistHeapHNSW) Len() int           { return len(h) }
func (h maxDistHeapHNSW) Less(i, j int) bool { return h[i].dist > h[j].dist } // max-heap
func (h maxDistHeapHNSW) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *maxDistHeapHNSW) Push(x any)        { *h = append(*h, x.(hnswDistItem)) }
func (h *maxDistHeapHNSW) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

func (e *HNSWEngine) Insert(vectors []*vectorize.Vector) {
	e.needsRebuild = true
}

func (e *HNSWEngine) Delete(ids []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, id := range ids {
		delete(e.nodes, id)
	}

	// Clean up references
	for _, node := range e.nodes {
		for l := range node.friends {
			filtered := make([]string, 0, len(node.friends[l]))
			for _, friendID := range node.friends[l] {
				if _, exists := e.nodes[friendID]; exists {
					filtered = append(filtered, friendID)
				}
			}
			node.friends[l] = filtered
		}
	}
}

func (e *HNSWEngine) Search(query []float32, k int) []SearchResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.nodes) == 0 || e.entryPoint == "" {
		return nil
	}

	// Start from entry point
	currNode := e.entryPoint
	currDist := e.distFunc(query, e.nodes[currNode].values)

	// Greedy descent from top level to level 1
	for l := e.maxLevel; l > 0; l-- {
		changed := true
		for changed {
			changed = false
			if node, ok := e.nodes[currNode]; ok && l < len(node.friends) {
				for _, friendID := range node.friends[l] {
					if friend, ok := e.nodes[friendID]; ok {
						dist := e.distFunc(query, friend.values)
						if dist < currDist {
							currNode = friendID
							currDist = dist
							changed = true
						}
					}
				}
			}
		}
	}

	// Search at level 0 with ef
	ef := e.efSearch
	if ef < k {
		ef = k * 2
	}

	neighbors := e.searchLevel(query, currNode, ef, 0)

	// Return top k
	results := make([]SearchResult, 0, k)
	for _, id := range neighbors {
		if node, ok := e.nodes[id]; ok {
			dist := e.distFunc(query, node.values)
			results = append(results, SearchResult{ID: id, Distance: dist})
		}
		if len(results) >= k {
			break
		}
	}

	return results
}

func (e *HNSWEngine) NeedsRebuild() bool     { return e.needsRebuild }
func (e *HNSWEngine) SetNeedsRebuild(v bool) { e.needsRebuild = v }

// distItem for priority queue
type distItem struct {
	id    string
	dist  float32
	isMax bool // true for max-heap (worst at top), false for min-heap
}

// distHeap implements a priority queue
type distHeap []distItem

func (h distHeap) Len() int { return len(h) }

func (h distHeap) Less(i, j int) bool {
	if h[i].isMax {
		return h[i].dist > h[j].dist // max-heap
	}
	return h[i].dist < h[j].dist // min-heap
}

func (h distHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *distHeap) Push(x any) {
	*h = append(*h, x.(distItem))
}

func (h *distHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
