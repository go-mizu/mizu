package local

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// Cache provides caching functionality for engines.
type Cache interface {
	// Get retrieves a cached value.
	Get(key string) (any, bool)

	// Set stores a value with expiration.
	Set(key string, value any, expire time.Duration) error

	// Delete removes a cached value.
	Delete(key string) error

	// SecretHash creates a secure hash of a value.
	SecretHash(value string) string
}

// cacheEntry represents a cached item.
type cacheEntry struct {
	value      any
	expiration time.Time
}

// MemoryCache is an in-memory cache implementation.
type MemoryCache struct {
	mu      sync.RWMutex
	items   map[string]cacheEntry
	secret  string
	ttl     time.Duration
	cleanup *time.Ticker
	done    chan struct{}
}

// NewMemoryCache creates a new memory cache.
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	mc := &MemoryCache{
		items:  make(map[string]cacheEntry),
		secret: generateSecret(),
		ttl:    defaultTTL,
		done:   make(chan struct{}),
	}

	// Start cleanup goroutine
	mc.cleanup = time.NewTicker(time.Minute)
	go mc.cleanupLoop()

	return mc
}

func (mc *MemoryCache) cleanupLoop() {
	for {
		select {
		case <-mc.cleanup.C:
			mc.deleteExpired()
		case <-mc.done:
			mc.cleanup.Stop()
			return
		}
	}
}

func (mc *MemoryCache) deleteExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	for key, entry := range mc.items {
		if now.After(entry.expiration) {
			delete(mc.items, key)
		}
	}
}

// Get retrieves a cached value.
func (mc *MemoryCache) Get(key string) (any, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, ok := mc.items[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiration) {
		return nil, false
	}

	return entry.value, true
}

// Set stores a value with expiration.
func (mc *MemoryCache) Set(key string, value any, expire time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if expire == 0 {
		expire = mc.ttl
	}

	mc.items[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(expire),
	}
	return nil
}

// Delete removes a cached value.
func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.items, key)
	return nil
}

// SecretHash creates a secure hash of a value.
func (mc *MemoryCache) SecretHash(value string) string {
	h := sha256.New()
	h.Write([]byte(mc.secret))
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

// Close stops the cleanup goroutine.
func (mc *MemoryCache) Close() {
	close(mc.done)
}

func generateSecret() string {
	// In production, this should be read from config or environment
	return "searxng-go-secret-key"
}

// EngineCache provides per-engine caching.
type EngineCache struct {
	cache      Cache
	engineName string
}

// NewEngineCache creates a new engine cache.
func NewEngineCache(cache Cache, engineName string) *EngineCache {
	return &EngineCache{
		cache:      cache,
		engineName: engineName,
	}
}

// Get retrieves a cached value for this engine.
func (ec *EngineCache) Get(key string) (any, bool) {
	return ec.cache.Get(ec.prefixKey(key))
}

// Set stores a value for this engine.
func (ec *EngineCache) Set(key string, value any, expire time.Duration) error {
	return ec.cache.Set(ec.prefixKey(key), value, expire)
}

// Delete removes a cached value for this engine.
func (ec *EngineCache) Delete(key string) error {
	return ec.cache.Delete(ec.prefixKey(key))
}

// SecretHash creates a secure hash for this engine.
func (ec *EngineCache) SecretHash(value string) string {
	return ec.cache.SecretHash(ec.engineName + ":" + value)
}

func (ec *EngineCache) prefixKey(key string) string {
	return ec.engineName + ":" + key
}
