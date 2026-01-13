package api

import (
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Analytics handles analytics requests.
type Analytics struct {
	store store.AnalyticsStore
}

// NewAnalytics creates a new Analytics handler.
func NewAnalytics(store store.AnalyticsStore) *Analytics {
	return &Analytics{store: store}
}

// Traffic returns traffic analytics for a zone.
func (h *Analytics) Traffic(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	period := c.Query("period")
	if period == "" {
		period = "24h"
	}

	var since time.Time
	switch period {
	case "6h":
		since = time.Now().Add(-6 * time.Hour)
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	default:
		since = time.Now().Add(-24 * time.Hour)
	}

	data, err := h.store.Query(c.Request().Context(), zoneID, since, time.Now())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	summary, err := h.store.GetSummary(c.Request().Context(), zoneID, period)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"totals": map[string]interface{}{
				"requests":      summary.Requests,
				"bandwidth":     summary.Bandwidth,
				"page_views":    summary.PageViews,
				"unique_visits": summary.UniqueVisits,
			},
			"timeseries": data,
		},
	})
}

// Security returns security analytics for a zone.
func (h *Analytics) Security(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	period := c.Query("period")
	if period == "" {
		period = "24h"
	}

	summary, err := h.store.GetSummary(c.Request().Context(), zoneID, period)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"totals": map[string]interface{}{
				"threats": summary.Threats,
			},
			"threats_by_type": map[string]int64{
				"bad_browser": summary.Threats / 3,
				"bad_ip":      summary.Threats / 3,
				"rate_limit":  summary.Threats / 3,
			},
			"threats_by_country": map[string]int64{
				"US": summary.Threats / 4,
				"CN": summary.Threats / 4,
				"RU": summary.Threats / 4,
				"XX": summary.Threats / 4,
			},
		},
	})
}

// Cache returns cache analytics for a zone.
func (h *Analytics) Cache(c *mizu.Ctx) error {
	zoneID := c.Param("zoneID")
	period := c.Query("period")
	if period == "" {
		period = "24h"
	}

	summary, err := h.store.GetSummary(c.Request().Context(), zoneID, period)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	total := summary.CacheHits + summary.CacheMisses
	hitRatio := float64(0)
	if total > 0 {
		hitRatio = float64(summary.CacheHits) / float64(total) * 100
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"totals": map[string]interface{}{
				"cache_hits":   summary.CacheHits,
				"cache_misses": summary.CacheMisses,
				"hit_ratio":    hitRatio,
			},
			"bandwidth_saved": summary.CacheHits * 1024, // Simulated bandwidth saved
		},
	})
}
