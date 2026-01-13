package api

import (
	"time"

	"github.com/go-mizu/mizu"

	ae "github.com/go-mizu/blueprints/localflare/feature/analytics_engine"
)

// AnalyticsEngine handles Analytics Engine requests.
type AnalyticsEngine struct {
	svc ae.API
}

// NewAnalyticsEngine creates a new AnalyticsEngine handler.
func NewAnalyticsEngine(svc ae.API) *AnalyticsEngine {
	return &AnalyticsEngine{svc: svc}
}

// ListDatasets lists all datasets.
func (h *AnalyticsEngine) ListDatasets(c *mizu.Ctx) error {
	datasets, err := h.svc.ListDatasets(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"datasets": datasets,
		},
	})
}

// CreateDataset creates a new dataset.
func (h *AnalyticsEngine) CreateDataset(c *mizu.Ctx) error {
	var input ae.CreateDatasetIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "name is required"})
	}

	ds, err := h.svc.CreateDataset(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  ds,
	})
}

// GetDataset retrieves a dataset by name.
func (h *AnalyticsEngine) GetDataset(c *mizu.Ctx) error {
	name := c.Param("name")
	ds, err := h.svc.GetDataset(c.Request().Context(), name)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Dataset not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  ds,
	})
}

// DeleteDataset deletes a dataset.
func (h *AnalyticsEngine) DeleteDataset(c *mizu.Ctx) error {
	name := c.Param("name")
	if err := h.svc.DeleteDataset(c.Request().Context(), name); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
	})
}

// WriteDataPoints writes data points to a dataset.
func (h *AnalyticsEngine) WriteDataPoints(c *mizu.Ctx) error {
	name := c.Param("name")

	var input struct {
		DataPoints []struct {
			Timestamp time.Time `json:"timestamp"`
			Indexes   []string  `json:"indexes"`
			Doubles   []float64 `json:"doubles"`
			Blobs     [][]byte  `json:"blobs"`
		} `json:"data_points"`
	}
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	points := make([]*ae.DataPoint, len(input.DataPoints))
	for i, dp := range input.DataPoints {
		ts := dp.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		points[i] = &ae.DataPoint{
			Dataset:   name,
			Timestamp: ts,
			Indexes:   dp.Indexes,
			Doubles:   dp.Doubles,
			Blobs:     dp.Blobs,
		}
	}

	in := &ae.WriteDataPointsIn{DataPoints: points}
	if err := h.svc.WriteDataPoints(c.Request().Context(), name, in); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]int{
			"rows_written": len(points),
		},
	})
}

// Query executes a SQL query.
func (h *AnalyticsEngine) Query(c *mizu.Ctx) error {
	var input ae.QueryIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.SQL == "" {
		return c.JSON(400, map[string]string{"error": "query is required"})
	}

	results, err := h.svc.Query(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"data": results,
			"meta": map[string]any{
				"rows": len(results),
			},
		},
	})
}
