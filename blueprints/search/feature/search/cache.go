package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine"
)

// CacheStore defines the interface for persistent cache storage operations.
type CacheStore interface {
	Get(ctx context.Context, hash string) (*CacheEntry, error)
	GetVersion(ctx context.Context, hash string, version int) (*CacheEntry, error)
	GetVersions(ctx context.Context, hash string) ([]*CacheEntry, error)
	Set(ctx context.Context, entry *CacheEntry) error
	Delete(ctx context.Context, hash string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// CacheEntry represents a cached search result with versioning.
type CacheEntry struct {
	Hash        string    `json:"hash"`
	Query       string    `json:"query"`
	Category    string    `json:"category"`
	OptionsJSON string    `json:"options_json"`
	ResultsJSON string    `json:"results_json"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}

// CacheStats contains cache statistics.
type CacheStats struct {
	TotalEntries   int64 `json:"total_entries"`
	UniqueQueries  int64 `json:"unique_queries"`
	TotalSizeBytes int64 `json:"total_size_bytes"`
	MemoryEntries  int   `json:"memory_entries"`
}

// CacheOptions contains options for cache operations.
type CacheOptions struct {
	Refetch bool // Force fetch from engine, bypassing cache
	Version int  // Specific version to retrieve (0 = latest)
}

// Cache provides two-tier caching for search results.
// L1: In-memory LRU cache for hot queries (fast, limited capacity, TTL-based)
// L2: SQLite persistent store with versioning (slow, unlimited, no TTL)
type Cache struct {
	memory *MemoryCache
	store  CacheStore
}

// NewCache creates a new two-tier cache.
func NewCache(store CacheStore, memory *MemoryCache) *Cache {
	return &Cache{
		memory: memory,
		store:  store,
	}
}

// NewCacheWithDefaults creates a new cache with default memory settings.
func NewCacheWithDefaults(store CacheStore) *Cache {
	return &Cache{
		memory: NewMemoryCacheWithDefaults(),
		store:  store,
	}
}

// CacheKey generates a cache key from query and options.
func CacheKey(query string, category engine.Category, opts engine.SearchOptions) string {
	data := fmt.Sprintf("%s:%s:%d:%d:%s:%s:%s:%d",
		query,
		category,
		opts.Page,
		opts.PerPage,
		opts.TimeRange,
		opts.Language,
		opts.Region,
		opts.SafeSearch,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Get retrieves a cached search response.
// First checks L1 (memory), then L2 (SQLite).
// If found in L2, populates L1 for future requests.
func (c *Cache) Get(ctx context.Context, query string, category engine.Category, opts engine.SearchOptions, cacheOpts CacheOptions) (*engine.SearchResponse, bool) {
	// If refetch requested, skip cache entirely
	if cacheOpts.Refetch {
		return nil, false
	}

	hash := CacheKey(query, category, opts)

	// If specific version requested, go directly to store
	if cacheOpts.Version > 0 {
		return c.getVersion(ctx, hash, cacheOpts.Version)
	}

	// Check L1 (memory cache)
	if c.memory != nil {
		if response, ok := c.memory.Get(hash); ok {
			return response, true
		}
	}

	// Check L2 (SQLite store)
	entry, err := c.store.Get(ctx, hash)
	if err != nil || entry == nil {
		return nil, false
	}

	// Unmarshal results
	var response engine.SearchResponse
	if err := json.Unmarshal([]byte(entry.ResultsJSON), &response); err != nil {
		return nil, false
	}

	// Populate L1 cache for future requests
	if c.memory != nil {
		c.memory.Set(hash, &response)
	}

	return &response, true
}

// getVersion retrieves a specific version from the store.
func (c *Cache) getVersion(ctx context.Context, hash string, version int) (*engine.SearchResponse, bool) {
	// Version queries only supported in store interface that has GetVersion
	type versionedStore interface {
		GetVersion(ctx context.Context, hash string, version int) (*CacheEntry, error)
	}

	vs, ok := c.store.(versionedStore)
	if !ok {
		return nil, false
	}

	entry, err := vs.GetVersion(ctx, hash, version)
	if err != nil || entry == nil {
		return nil, false
	}

	var response engine.SearchResponse
	if err := json.Unmarshal([]byte(entry.ResultsJSON), &response); err != nil {
		return nil, false
	}

	return &response, true
}

// Set stores a search response in both L1 and L2 caches.
func (c *Cache) Set(ctx context.Context, query string, category engine.Category, opts engine.SearchOptions, response *engine.SearchResponse) error {
	hash := CacheKey(query, category, opts)

	// Marshal results
	resultsJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	// Marshal options for storage
	optsJSON, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}

	// Store in L2 (SQLite) - creates new version
	entry := &CacheEntry{
		Hash:        hash,
		Query:       query,
		Category:    string(category),
		OptionsJSON: string(optsJSON),
		ResultsJSON: string(resultsJSON),
		CreatedAt:   time.Now(),
	}

	if err := c.store.Set(ctx, entry); err != nil {
		return err
	}

	// Store in L1 (memory)
	if c.memory != nil {
		c.memory.Set(hash, response)
	}

	return nil
}

// Invalidate removes a cached entry from both L1 and L2.
func (c *Cache) Invalidate(ctx context.Context, query string, category engine.Category, opts engine.SearchOptions) error {
	hash := CacheKey(query, category, opts)

	// Remove from L1
	if c.memory != nil {
		c.memory.Delete(hash)
	}

	// Remove from L2
	return c.store.Delete(ctx, hash)
}

// InvalidateMemory removes an entry from memory cache only.
func (c *Cache) InvalidateMemory(query string, category engine.Category, opts engine.SearchOptions) {
	if c.memory != nil {
		hash := CacheKey(query, category, opts)
		c.memory.Delete(hash)
	}
}

// ClearMemory clears all entries from the memory cache.
func (c *Cache) ClearMemory() {
	if c.memory != nil {
		c.memory.Clear()
	}
}

// GetVersions retrieves all versions of a cached query.
func (c *Cache) GetVersions(ctx context.Context, query string, category engine.Category, opts engine.SearchOptions) ([]*CacheEntry, error) {
	type versionedStore interface {
		GetVersions(ctx context.Context, hash string) ([]*CacheEntry, error)
	}

	vs, ok := c.store.(versionedStore)
	if !ok {
		return nil, fmt.Errorf("store does not support versioning")
	}

	hash := CacheKey(query, category, opts)
	return vs.GetVersions(ctx, hash)
}

// Stats returns cache statistics.
func (c *Cache) Stats(ctx context.Context) (*CacheStats, error) {
	type statsStore interface {
		GetStats(ctx context.Context) (*CacheStats, error)
	}

	ss, ok := c.store.(statsStore)
	if !ok {
		return &CacheStats{
			MemoryEntries: c.memory.Len(),
		}, nil
	}

	stats, err := ss.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	if c.memory != nil {
		stats.MemoryEntries = c.memory.Len()
	}

	return stats, nil
}

// Cleanup performs maintenance on the cache.
func (c *Cache) Cleanup(ctx context.Context) error {
	// Cleanup expired memory entries
	if c.memory != nil {
		c.memory.CleanupExpired()
	}

	return nil
}
