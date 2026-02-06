package fw2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CacheData holds all cached API responses.
type CacheData struct {
	Configs   []DatasetConfig          `json:"configs,omitempty"`
	Sizes     *DatasetSizeInfo         `json:"sizes,omitempty"`
	Files     map[string][]FileInfo    `json:"files,omitempty"`    // key: "lang/split"
	FetchedAt time.Time               `json:"fetched_at"`
}

// Cache provides disk-backed caching for HuggingFace API responses.
type Cache struct {
	path string
	ttl  time.Duration
}

// DefaultCacheTTL is how long cached data remains valid.
const DefaultCacheTTL = 24 * time.Hour

// NewCache creates a cache at ~/.cache/search/fw2.json.
func NewCache() *Cache {
	home, _ := os.UserHomeDir()
	return &Cache{
		path: filepath.Join(home, ".cache", "search", "fw2.json"),
		ttl:  DefaultCacheTTL,
	}
}

// Load reads the cache from disk. Returns nil if missing, expired, or corrupt.
func (c *Cache) Load() *CacheData {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil
	}
	var cd CacheData
	if err := json.Unmarshal(data, &cd); err != nil {
		return nil
	}
	if time.Since(cd.FetchedAt) > c.ttl {
		return nil
	}
	return &cd
}

// Save writes cache data to disk.
func (c *Cache) Save(cd *CacheData) error {
	cd.FetchedAt = time.Now()
	data, err := json.MarshalIndent(cd, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0644)
}

// Age returns how long ago the cache was written. Returns 0 if no cache.
func (c *Cache) Age() time.Duration {
	cd := c.Load()
	if cd == nil {
		return 0
	}
	return time.Since(cd.FetchedAt)
}

// Path returns the cache file path.
func (c *Cache) Path() string {
	return c.path
}
