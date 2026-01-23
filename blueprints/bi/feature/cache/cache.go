// Package cache provides query result caching for data sources.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/bi/drivers"
)

// Cache defines the interface for query result caching.
type Cache interface {
	// Get retrieves a cached result by key.
	Get(ctx context.Context, key string) (*CachedResult, error)

	// Set stores a result with the given TTL.
	Set(ctx context.Context, key string, result *CachedResult, ttl time.Duration) error

	// Delete removes a cached result.
	Delete(ctx context.Context, key string) error

	// Clear removes all cached results for a data source.
	Clear(ctx context.Context, datasourceID string) error

	// Stats returns cache statistics for a data source.
	Stats(ctx context.Context, datasourceID string) (*CacheStats, error)
}

// CachedResult represents a cached query result.
type CachedResult struct {
	DataSourceID string               `json:"datasource_id"`
	QueryHash    string               `json:"query_hash"`
	Result       *drivers.QueryResult `json:"result"`
	CachedAt     time.Time            `json:"cached_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	HitCount     int64                `json:"hit_count"`
}

// CacheStats contains cache statistics.
type CacheStats struct {
	DataSourceID string  `json:"datasource_id"`
	Entries      int64   `json:"entries"`
	Hits         int64   `json:"hits"`
	Misses       int64   `json:"misses"`
	HitRate      float64 `json:"hit_rate"`
	MemoryBytes  int64   `json:"memory_bytes"`
}

// MemoryCache implements an in-memory cache.
type MemoryCache struct {
	entries      map[string]*cacheEntry
	stats        map[string]*cacheStatsInternal
	mu           sync.RWMutex
	maxSize      int
	cleanupEvery time.Duration
	stopCh       chan struct{}
}

type cacheEntry struct {
	result    *CachedResult
	size      int64
	expiresAt time.Time
}

type cacheStatsInternal struct {
	entries int64
	hits    int64
	misses  int64
	bytes   int64
}

// MemoryCacheOption is a functional option for MemoryCache.
type MemoryCacheOption func(*MemoryCache)

// WithMaxSize sets the maximum number of entries.
func WithMaxSize(size int) MemoryCacheOption {
	return func(c *MemoryCache) {
		c.maxSize = size
	}
}

// WithCleanupInterval sets the cleanup interval.
func WithCleanupInterval(d time.Duration) MemoryCacheOption {
	return func(c *MemoryCache) {
		c.cleanupEvery = d
	}
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache(opts ...MemoryCacheOption) *MemoryCache {
	c := &MemoryCache{
		entries:      make(map[string]*cacheEntry),
		stats:        make(map[string]*cacheStatsInternal),
		maxSize:      10000,
		cleanupEvery: time.Minute,
		stopCh:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c
}

// Get retrieves a cached result.
func (c *MemoryCache) Get(ctx context.Context, key string) (*CachedResult, error) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		c.recordMiss(key)
		return nil, nil
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		c.Delete(ctx, key)
		c.recordMiss(key)
		return nil, nil
	}

	// Update hit count
	c.mu.Lock()
	entry.result.HitCount++
	c.mu.Unlock()

	c.recordHit(key)
	return entry.result, nil
}

// Set stores a result in the cache.
func (c *MemoryCache) Set(ctx context.Context, key string, result *CachedResult, ttl time.Duration) error {
	// Calculate size
	size := c.estimateSize(result)

	entry := &cacheEntry{
		result:    result,
		size:      size,
		expiresAt: time.Now().Add(ttl),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict entries
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = entry

	// Update stats
	dsID := result.DataSourceID
	if c.stats[dsID] == nil {
		c.stats[dsID] = &cacheStatsInternal{}
	}
	c.stats[dsID].entries++
	c.stats[dsID].bytes += size

	return nil
}

// Delete removes a cached result.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}

	dsID := entry.result.DataSourceID
	if stats, ok := c.stats[dsID]; ok {
		stats.entries--
		stats.bytes -= entry.size
	}

	delete(c.entries, key)
	return nil
}

// Clear removes all cached results for a data source.
func (c *MemoryCache) Clear(ctx context.Context, datasourceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	keysToDelete := make([]string, 0)
	for key, entry := range c.entries {
		if entry.result.DataSourceID == datasourceID {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(c.entries, key)
	}

	// Reset stats for this data source
	c.stats[datasourceID] = &cacheStatsInternal{}

	return nil
}

// Stats returns cache statistics.
func (c *MemoryCache) Stats(ctx context.Context, datasourceID string) (*CacheStats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats[datasourceID]
	if stats == nil {
		return &CacheStats{
			DataSourceID: datasourceID,
		}, nil
	}

	hitRate := float64(0)
	total := stats.hits + stats.misses
	if total > 0 {
		hitRate = float64(stats.hits) / float64(total)
	}

	return &CacheStats{
		DataSourceID: datasourceID,
		Entries:      stats.entries,
		Hits:         stats.hits,
		Misses:       stats.misses,
		HitRate:      hitRate,
		MemoryBytes:  stats.bytes,
	}, nil
}

// Close stops the cache cleanup goroutine.
func (c *MemoryCache) Close() error {
	close(c.stopCh)
	return nil
}

// cleanup periodically removes expired entries.
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(c.cleanupEvery)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.removeExpired()
		}
	}
}

// removeExpired removes all expired entries.
func (c *MemoryCache) removeExpired() {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			dsID := entry.result.DataSourceID
			if stats, ok := c.stats[dsID]; ok {
				stats.entries--
				stats.bytes -= entry.size
			}
			delete(c.entries, key)
		}
	}
}

// evictOldest removes the oldest entries to make room.
func (c *MemoryCache) evictOldest() {
	// Simple eviction: remove entries that are closest to expiration
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		entry := c.entries[oldestKey]
		dsID := entry.result.DataSourceID
		if stats, ok := c.stats[dsID]; ok {
			stats.entries--
			stats.bytes -= entry.size
		}
		delete(c.entries, oldestKey)
	}
}

// recordHit records a cache hit.
func (c *MemoryCache) recordHit(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.entries[key]
	if entry == nil {
		return
	}

	dsID := entry.result.DataSourceID
	if c.stats[dsID] == nil {
		c.stats[dsID] = &cacheStatsInternal{}
	}
	c.stats[dsID].hits++
}

// recordMiss records a cache miss.
func (c *MemoryCache) recordMiss(key string) {
	// Extract datasource ID from key if possible
	// For now, we'll skip this since we don't have the datasource ID in misses
	// In a real implementation, we'd parse the key or track misses differently
}

// estimateSize estimates the memory size of a cached result.
func (c *MemoryCache) estimateSize(result *CachedResult) int64 {
	if result == nil || result.Result == nil {
		return 0
	}

	// Rough estimate: JSON size
	data, err := json.Marshal(result)
	if err != nil {
		return 1024 // Default size
	}
	return int64(len(data))
}

// GenerateCacheKey generates a cache key for a query.
func GenerateCacheKey(datasourceID, query string, params []any) string {
	h := sha256.New()
	h.Write([]byte(datasourceID))
	h.Write([]byte(query))

	for _, p := range params {
		h.Write([]byte(fmt.Sprintf("%v", p)))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// CachedExecutor wraps a driver with caching.
type CachedExecutor struct {
	driver       drivers.Driver
	cache        Cache
	datasourceID string
	ttl          time.Duration
}

// NewCachedExecutor creates a new cached executor.
func NewCachedExecutor(driver drivers.Driver, cache Cache, datasourceID string, ttl time.Duration) *CachedExecutor {
	return &CachedExecutor{
		driver:       driver,
		cache:        cache,
		datasourceID: datasourceID,
		ttl:          ttl,
	}
}

// Execute runs a query with caching.
func (e *CachedExecutor) Execute(ctx context.Context, query string, params ...any) (*drivers.QueryResult, error) {
	if e.ttl <= 0 || e.cache == nil {
		// Caching disabled
		return e.driver.Execute(ctx, query, params...)
	}

	// Generate cache key
	key := GenerateCacheKey(e.datasourceID, query, params)

	// Try to get from cache
	cached, err := e.cache.Get(ctx, key)
	if err == nil && cached != nil {
		result := cached.Result
		result.Cached = true
		return result, nil
	}

	// Execute query
	result, err := e.driver.Execute(ctx, query, params...)
	if err != nil {
		return nil, err
	}

	// Store in cache
	cachedResult := &CachedResult{
		DataSourceID: e.datasourceID,
		QueryHash:    key,
		Result:       result,
		CachedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(e.ttl),
	}
	e.cache.Set(ctx, key, cachedResult, e.ttl)

	return result, nil
}
