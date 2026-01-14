package mizu_vector

// Typed min/max heaps for efficient top-K selection without interface{} overhead.

// distItem32 represents a distance item with int32 index.
type distItem32 struct {
	idx  int32
	dist float32
}

// minHeap32 is a min-heap of distItem32 (closest at top).
type minHeap32 []distItem32

func (h minHeap32) Len() int            { return len(h) }
func (h minHeap32) Less(i, j int) bool  { return h[i].dist < h[j].dist }
func (h minHeap32) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap32) Push(x distItem32)  { *h = append(*h, x) }
func (h *minHeap32) Pop() distItem32 {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// heapifyUp restores heap property after Push.
func (h minHeap32) heapifyUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h[parent].dist <= h[i].dist {
			break
		}
		h[parent], h[i] = h[i], h[parent]
		i = parent
	}
}

// heapifyDown restores heap property after Pop.
func (h minHeap32) heapifyDown(i int) {
	n := len(h)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2
		if left < n && h[left].dist < h[smallest].dist {
			smallest = left
		}
		if right < n && h[right].dist < h[smallest].dist {
			smallest = right
		}
		if smallest == i {
			break
		}
		h[i], h[smallest] = h[smallest], h[i]
		i = smallest
	}
}

// PushItem adds an item and maintains heap property.
func (h *minHeap32) PushItem(item distItem32) {
	*h = append(*h, item)
	h.heapifyUp(len(*h) - 1)
}

// PopItem removes and returns the minimum item.
func (h *minHeap32) PopItem() distItem32 {
	old := *h
	n := len(old)
	item := old[0]
	old[0] = old[n-1]
	*h = old[:n-1]
	if len(*h) > 0 {
		h.heapifyDown(0)
	}
	return item
}

// PeekItem returns the minimum item without removing it.
func (h minHeap32) PeekItem() distItem32 {
	return h[0]
}

// maxHeap32 is a max-heap of distItem32 (farthest at top).
type maxHeap32 []distItem32

func (h maxHeap32) Len() int            { return len(h) }
func (h maxHeap32) Less(i, j int) bool  { return h[i].dist > h[j].dist }
func (h maxHeap32) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }

func (h maxHeap32) heapifyUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h[parent].dist >= h[i].dist {
			break
		}
		h[parent], h[i] = h[i], h[parent]
		i = parent
	}
}

func (h maxHeap32) heapifyDown(i int) {
	n := len(h)
	for {
		largest := i
		left := 2*i + 1
		right := 2*i + 2
		if left < n && h[left].dist > h[largest].dist {
			largest = left
		}
		if right < n && h[right].dist > h[largest].dist {
			largest = right
		}
		if largest == i {
			break
		}
		h[i], h[largest] = h[largest], h[i]
		i = largest
	}
}

// PushItem adds an item and maintains heap property.
func (h *maxHeap32) PushItem(item distItem32) {
	*h = append(*h, item)
	h.heapifyUp(len(*h) - 1)
}

// PopItem removes and returns the maximum item.
func (h *maxHeap32) PopItem() distItem32 {
	old := *h
	n := len(old)
	item := old[0]
	old[0] = old[n-1]
	*h = old[:n-1]
	if len(*h) > 0 {
		h.heapifyDown(0)
	}
	return item
}

// PeekItem returns the maximum item without removing it.
func (h maxHeap32) PeekItem() distItem32 {
	return h[0]
}

// topKHeap maintains the K smallest items using a max-heap.
// Items smaller than the max are added, maintaining size K.
type topKHeap struct {
	items maxHeap32
	k     int
}

// newTopKHeap creates a new top-K heap.
func newTopKHeap(k int) *topKHeap {
	return &topKHeap{
		items: make(maxHeap32, 0, k),
		k:     k,
	}
}

// TryAdd attempts to add an item. Returns true if added.
func (h *topKHeap) TryAdd(idx int32, dist float32) bool {
	if len(h.items) < h.k {
		h.items.PushItem(distItem32{idx: idx, dist: dist})
		return true
	}
	if dist < h.items[0].dist {
		h.items[0] = distItem32{idx: idx, dist: dist}
		h.items.heapifyDown(0)
		return true
	}
	return false
}

// MaxDist returns the maximum distance in the heap.
func (h *topKHeap) MaxDist() float32 {
	if len(h.items) == 0 {
		return 1e30
	}
	return h.items[0].dist
}

// Full returns true if the heap has K items.
func (h *topKHeap) Full() bool {
	return len(h.items) >= h.k
}

// Results returns items sorted by distance (ascending).
func (h *topKHeap) Results() []distItem32 {
	// Copy and sort
	result := make([]distItem32, len(h.items))
	copy(result, h.items)
	// Simple insertion sort for small k
	for i := 1; i < len(result); i++ {
		j := i
		for j > 0 && result[j].dist < result[j-1].dist {
			result[j], result[j-1] = result[j-1], result[j]
			j--
		}
	}
	return result
}

// Len returns the number of items.
func (h *topKHeap) Len() int {
	return len(h.items)
}
