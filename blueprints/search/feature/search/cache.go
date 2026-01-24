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

// CacheStore defines the interface for cache storage operations.
type CacheStore interface {
	Get(ctx context.Context, hash string) (*CacheEntry, error)
	Set(ctx context.Context, entry *CacheEntry) error
	Delete(ctx context.Context, hash string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// CacheEntry represents a cached search result.
type CacheEntry struct {
	Hash        string    `json:"hash"`
	Query       string    `json:"query"`
	Category    string    `json:"category"`
	ResultsJSON string    `json:"results_json"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Cache provides query-hash based caching for search results.
type Cache struct {
	store CacheStore
	ttl   time.Duration
}

// NewCache creates a new cache with the given store and TTL.
func NewCache(store CacheStore, ttl time.Duration) *Cache {
	return &Cache{
		store: store,
		ttl:   ttl,
	}
}

// DefaultTTL is the default cache TTL (1 hour).
const DefaultTTL = 1 * time.Hour

// NewCacheWithDefaultTTL creates a new cache with the default TTL.
func NewCacheWithDefaultTTL(store CacheStore) *Cache {
	return NewCache(store, DefaultTTL)
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

// Get retrieves a cached search response if it exists and is not expired.
func (c *Cache) Get(ctx context.Context, query string, category engine.Category, opts engine.SearchOptions) (*engine.SearchResponse, bool) {
	hash := CacheKey(query, category, opts)

	entry, err := c.store.Get(ctx, hash)
	if err != nil {
		return nil, false
	}

	if entry == nil {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Delete expired entry
		_ = c.store.Delete(ctx, hash)
		return nil, false
	}

	// Unmarshal results
	var response engine.SearchResponse
	if err := json.Unmarshal([]byte(entry.ResultsJSON), &response); err != nil {
		return nil, false
	}

	return &response, true
}

// Set stores a search response in the cache.
func (c *Cache) Set(ctx context.Context, query string, category engine.Category, opts engine.SearchOptions, response *engine.SearchResponse) error {
	hash := CacheKey(query, category, opts)

	// Marshal results
	resultsJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	entry := &CacheEntry{
		Hash:        hash,
		Query:       query,
		Category:    string(category),
		ResultsJSON: string(resultsJSON),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(c.ttl),
	}

	return c.store.Set(ctx, entry)
}

// Invalidate removes a cached entry.
func (c *Cache) Invalidate(ctx context.Context, query string, category engine.Category, opts engine.SearchOptions) error {
	hash := CacheKey(query, category, opts)
	return c.store.Delete(ctx, hash)
}

// Cleanup removes all expired entries from the cache.
func (c *Cache) Cleanup(ctx context.Context) (int64, error) {
	return c.store.DeleteExpired(ctx)
}

// TTL returns the cache TTL.
func (c *Cache) TTL() time.Duration {
	return c.ttl
}
