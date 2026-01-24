package search

import (
	"container/list"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine"
)

// MemoryCacheEntry represents an entry in the memory cache.
type MemoryCacheEntry struct {
	Hash      string
	Response  *engine.SearchResponse
	ExpiresAt time.Time
}

// MemoryCache provides an in-memory LRU cache for search results.
type MemoryCache struct {
	mu       sync.RWMutex
	capacity int
	ttl      time.Duration
	items    map[string]*list.Element
	lru      *list.List
}

// NewMemoryCache creates a new memory cache with the given capacity and TTL.
func NewMemoryCache(capacity int, ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// DefaultMemoryCacheCapacity is the default number of entries to cache.
const DefaultMemoryCacheCapacity = 500

// DefaultMemoryCacheTTL is the default TTL for memory cache entries.
const DefaultMemoryCacheTTL = 15 * time.Minute

// NewMemoryCacheWithDefaults creates a memory cache with default settings.
func NewMemoryCacheWithDefaults() *MemoryCache {
	return NewMemoryCache(DefaultMemoryCacheCapacity, DefaultMemoryCacheTTL)
}

// Get retrieves a cached response if it exists and is not expired.
func (c *MemoryCache) Get(hash string) (*engine.SearchResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[hash]
	if !ok {
		return nil, false
	}

	entry := elem.Value.(*MemoryCacheEntry)

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.removeElement(elem)
		return nil, false
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(elem)

	return entry.Response, true
}

// Set stores a response in the cache, evicting the least recently used entry if at capacity.
func (c *MemoryCache) Set(hash string, response *engine.SearchResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already exists, update and move to front
	if elem, ok := c.items[hash]; ok {
		entry := elem.Value.(*MemoryCacheEntry)
		entry.Response = response
		entry.ExpiresAt = time.Now().Add(c.ttl)
		c.lru.MoveToFront(elem)
		return
	}

	// Evict if at capacity
	for c.lru.Len() >= c.capacity {
		c.evictOldest()
	}

	// Add new entry
	entry := &MemoryCacheEntry{
		Hash:      hash,
		Response:  response,
		ExpiresAt: time.Now().Add(c.ttl),
	}
	elem := c.lru.PushFront(entry)
	c.items[hash] = elem
}

// Delete removes an entry from the cache.
func (c *MemoryCache) Delete(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[hash]; ok {
		c.removeElement(elem)
	}
}

// Clear removes all entries from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lru.Init()
}

// Len returns the number of entries in the cache.
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}

// evictOldest removes the least recently used entry.
func (c *MemoryCache) evictOldest() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes an element from the cache.
func (c *MemoryCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*MemoryCacheEntry)
	delete(c.items, entry.Hash)
	c.lru.Remove(elem)
}

// CleanupExpired removes all expired entries from the cache.
func (c *MemoryCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	// Iterate from back (oldest) to front
	for elem := c.lru.Back(); elem != nil; {
		entry := elem.Value.(*MemoryCacheEntry)
		prev := elem.Prev()
		if now.After(entry.ExpiresAt) {
			c.removeElement(elem)
			removed++
		}
		elem = prev
	}

	return removed
}
