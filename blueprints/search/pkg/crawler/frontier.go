package crawler

import (
	"container/heap"
	"sync"
	"time"
)

// Frontier is a thread-safe priority queue with per-domain rate limiting
// and a visited set for URL deduplication.
type Frontier struct {
	mu         sync.Mutex
	pq         priorityQueue
	visited    map[string]bool
	domainNext map[string]time.Time
	delay      time.Duration
	closed     bool
	cond       *sync.Cond
}

// NewFrontier creates a new frontier with the given per-domain delay.
func NewFrontier(delay time.Duration) *Frontier {
	f := &Frontier{
		visited:    make(map[string]bool),
		domainNext: make(map[string]time.Time),
		delay:      delay,
	}
	heap.Init(&f.pq)
	f.cond = sync.NewCond(&f.mu)
	return f
}

// Push adds a URL to the frontier if not already visited.
// Returns true if the URL was added.
func (f *Frontier) Push(entry URLEntry) bool {
	normalized, err := NormalizeURL(entry.URL)
	if err != nil {
		return false
	}
	entry.URL = normalized

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.visited[entry.URL] {
		return false
	}
	f.visited[entry.URL] = true
	heap.Push(&f.pq, &pqItem{entry: entry, priority: entry.Priority})
	f.cond.Signal()
	return true
}

// Pop removes and returns the highest-priority URL.
// It blocks until a URL is available or the frontier is closed.
// Returns the entry and false if the frontier is closed and empty.
func (f *Frontier) Pop() (URLEntry, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for f.pq.Len() == 0 && !f.closed {
		f.cond.Wait()
	}

	if f.pq.Len() == 0 {
		return URLEntry{}, false
	}

	item := heap.Pop(&f.pq).(*pqItem)
	return item.entry, true
}

// TryPop is a non-blocking version of Pop.
// Returns the entry and false if the frontier is empty.
func (f *Frontier) TryPop() (URLEntry, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.pq.Len() == 0 {
		return URLEntry{}, false
	}

	item := heap.Pop(&f.pq).(*pqItem)
	return item.entry, true
}

// WaitForDomain blocks until the domain is available for fetching,
// respecting the per-domain rate limit.
func (f *Frontier) WaitForDomain(domain string) {
	f.mu.Lock()
	next, ok := f.domainNext[domain]
	f.mu.Unlock()

	if ok {
		wait := time.Until(next)
		if wait > 0 {
			time.Sleep(wait)
		}
	}

	f.mu.Lock()
	f.domainNext[domain] = time.Now().Add(f.delay)
	f.mu.Unlock()
}

// SetDomainDelay overrides the delay for a specific domain.
func (f *Frontier) SetDomainDelay(domain string, delay time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.domainNext[domain] = time.Now().Add(delay)
}

// IsVisited checks if a URL has been visited.
func (f *Frontier) IsVisited(rawURL string) bool {
	normalized, err := NormalizeURL(rawURL)
	if err != nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.visited[normalized]
}

// Len returns the number of URLs in the queue (not visited).
func (f *Frontier) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pq.Len()
}

// VisitedCount returns the number of visited URLs.
func (f *Frontier) VisitedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.visited)
}

// Close signals that no more URLs will be added.
func (f *Frontier) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	f.cond.Broadcast()
}

// VisitedURLs returns a copy of all visited URLs.
func (f *Frontier) VisitedURLs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	urls := make([]string, 0, len(f.visited))
	for u := range f.visited {
		urls = append(urls, u)
	}
	return urls
}

// PendingEntries returns a copy of all pending entries.
func (f *Frontier) PendingEntries() []URLEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	entries := make([]URLEntry, f.pq.Len())
	for i, item := range f.pq {
		entries[i] = item.entry
	}
	return entries
}

// --- Priority Queue implementation ---

type pqItem struct {
	entry    URLEntry
	priority int
	index    int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Lower priority value = higher priority
	return pq[i].priority < pq[j].priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x any) {
	item := x.(*pqItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}
