package cc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Cache provides persistent caching for Common Crawl API responses.
// Crawl list, manifests, and index metadata are cached locally to
// minimize network requests on limited connections.
type Cache struct {
	path string
	ttl  time.Duration
}

// CacheData holds all cached Common Crawl data.
type CacheData struct {
	// Volatile — expires after TTL
	Crawls    []Crawl           `json:"crawls,omitempty"`
	FetchedAt time.Time         `json:"fetched_at"`

	// Semi-permanent — manifests don't change for a given crawl
	Manifests map[string][]string `json:"manifests,omitempty"` // key: "CC-MAIN-2026-04:cc-index-table.paths.gz"
}

// NewCache creates a cache in the given data directory.
func NewCache(dataDir string) *Cache {
	return &Cache{
		path: filepath.Join(dataDir, "cache.json"),
		ttl:  24 * time.Hour,
	}
}

// Load reads cache from disk. Returns nil if cache is missing or corrupted.
func (c *Cache) Load() *CacheData {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil
	}
	var cd CacheData
	if err := json.Unmarshal(data, &cd); err != nil {
		return nil
	}
	return &cd
}

// Save persists cache to disk.
func (c *Cache) Save(cd *CacheData) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0644)
}

// IsFresh returns true if the cache data is within TTL.
func (c *Cache) IsFresh(cd *CacheData) bool {
	if cd == nil {
		return false
	}
	return time.Since(cd.FetchedAt) < c.ttl
}

// GetManifest returns cached manifest paths for a crawl+kind, or nil.
func (c *Cache) GetManifest(cd *CacheData, crawlID, kind string) []string {
	if cd == nil || cd.Manifests == nil {
		return nil
	}
	key := crawlID + ":" + kind
	return cd.Manifests[key]
}

// SetManifest caches manifest paths.
func (c *Cache) SetManifest(cd *CacheData, crawlID, kind string, paths []string) {
	if cd.Manifests == nil {
		cd.Manifests = make(map[string][]string)
	}
	cd.Manifests[crawlID+":"+kind] = paths
}
