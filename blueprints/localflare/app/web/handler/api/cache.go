package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Cache handles cache-related requests.
type Cache struct {
	store store.CacheStore
}

// NewCache creates a new Cache handler.
func NewCache(store store.CacheStore) *Cache {
	return &Cache{store: store}
}

// GetSettings retrieves cache settings for a zone.
func (h *Cache) GetSettings(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	settings, err := h.store.GetSettings(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  settings,
	})
}

// UpdateCacheSettingsInput is the input for updating cache settings.
type UpdateCacheSettingsInput struct {
	CacheLevel      string `json:"cache_level"`
	BrowserTTL      int    `json:"browser_ttl"`
	EdgeTTL         int    `json:"edge_ttl"`
	DevelopmentMode bool   `json:"development_mode"`
	AlwaysOnline    bool   `json:"always_online"`
}

// UpdateSettings updates cache settings for a zone.
func (h *Cache) UpdateSettings(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input UpdateCacheSettingsInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	settings := &store.CacheSettings{
		ZoneID:          zoneID,
		CacheLevel:      input.CacheLevel,
		BrowserTTL:      input.BrowserTTL,
		EdgeTTL:         input.EdgeTTL,
		DevelopmentMode: input.DevelopmentMode,
		AlwaysOnline:    input.AlwaysOnline,
	}

	if settings.CacheLevel == "" {
		settings.CacheLevel = "standard"
	}

	if err := h.store.UpdateSettings(c.Request().Context(), settings); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  settings,
	})
}

// ListRules lists all cache rules for a zone.
func (h *Cache) ListRules(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	rules, err := h.store.ListRules(c.Request().Context(), zoneID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  rules,
	})
}

// CreateCacheRuleInput is the input for creating a cache rule.
type CreateCacheRuleInput struct {
	Name        string `json:"name"`
	Expression  string `json:"expression"`
	CacheLevel  string `json:"cache_level"`
	EdgeTTL     int    `json:"edge_ttl"`
	BrowserTTL  int    `json:"browser_ttl"`
	BypassCache bool   `json:"bypass_cache"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// CreateRule creates a new cache rule.
func (h *Cache) CreateRule(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input CreateCacheRuleInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" || input.Expression == "" {
		return c.JSON(400, map[string]string{"error": "Name and expression are required"})
	}

	rule := &store.CacheRule{
		ID:          ulid.Make().String(),
		ZoneID:      zoneID,
		Name:        input.Name,
		Expression:  input.Expression,
		CacheLevel:  input.CacheLevel,
		EdgeTTL:     input.EdgeTTL,
		BrowserTTL:  input.BrowserTTL,
		BypassCache: input.BypassCache,
		Priority:    input.Priority,
		Enabled:     input.Enabled,
		CreatedAt:   time.Now(),
	}

	if err := h.store.CreateRule(c.Request().Context(), rule); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  rule,
	})
}

// DeleteRule deletes a cache rule.
func (h *Cache) DeleteRule(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteRule(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// PurgeInput is the input for purging cache.
type PurgeInput struct {
	PurgeEverything bool     `json:"purge_everything"`
	Files           []string `json:"files"`
	Tags            []string `json:"tags"`
	Prefixes        []string `json:"prefixes"`
}

// Purge purges the cache for a zone.
func (h *Cache) Purge(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")

	var input PurgeInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.PurgeEverything {
		if err := h.store.PurgeAll(c.Request().Context(), zoneID); err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
	} else if len(input.Files) > 0 {
		if err := h.store.PurgeURLs(c.Request().Context(), zoneID, input.Files); err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]string{
			"id": zoneID,
		},
	})
}
