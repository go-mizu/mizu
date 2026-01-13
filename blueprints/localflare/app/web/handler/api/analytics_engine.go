package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// AnalyticsEngine handles Analytics Engine requests.
type AnalyticsEngine struct {
	store store.AnalyticsEngineStore
}

// NewAnalyticsEngine creates a new AnalyticsEngine handler.
func NewAnalyticsEngine(store store.AnalyticsEngineStore) *AnalyticsEngine {
	return &AnalyticsEngine{store: store}
}

// ListDatasets lists all datasets.
func (h *AnalyticsEngine) ListDatasets(c *mizu.Ctx) error {
	datasets, err := h.store.ListDatasets(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  datasets,
	})
}

// CreateAEDatasetInput is the input for creating a dataset.
type CreateAEDatasetInput struct {
	Name string `json:"name"`
}

// CreateDataset creates a new dataset.
func (h *AnalyticsEngine) CreateDataset(c *mizu.Ctx) error {
	var input CreateAEDatasetInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "name is required"})
	}

	ds := &store.AnalyticsEngineDataset{
		ID:        ulid.Make().String(),
		Name:      input.Name,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateDataset(c.Request().Context(), ds); err != nil {
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
	ds, err := h.store.GetDataset(c.Request().Context(), name)
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
	if err := h.store.DeleteDataset(c.Request().Context(), name); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
	})
}

// WriteDataPointsInput is the input for writing data points.
type WriteDataPointsInput struct {
	DataPoints []struct {
		Timestamp time.Time  `json:"timestamp"`
		Indexes   []string   `json:"indexes"`
		Doubles   []float64  `json:"doubles"`
		Blobs     [][]byte   `json:"blobs"`
	} `json:"data_points"`
}

// WriteDataPoints writes data points to a dataset.
func (h *AnalyticsEngine) WriteDataPoints(c *mizu.Ctx) error {
	name := c.Param("name")

	var input WriteDataPointsInput
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	points := make([]*store.AnalyticsEngineDataPoint, len(input.DataPoints))
	for i, dp := range input.DataPoints {
		ts := dp.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		points[i] = &store.AnalyticsEngineDataPoint{
			Dataset:   name,
			Timestamp: ts,
			Indexes:   dp.Indexes,
			Doubles:   dp.Doubles,
			Blobs:     dp.Blobs,
		}
	}

	if err := h.store.WriteBatch(c.Request().Context(), points); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]int{
			"rows_written": len(points),
		},
	})
}

// AEQueryInput is the input for querying a dataset.
type AEQueryInput struct {
	Query string `json:"query"`
}

// Query executes a SQL query.
func (h *AnalyticsEngine) Query(c *mizu.Ctx) error {
	var input AEQueryInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Query == "" {
		return c.JSON(400, map[string]string{"error": "query is required"})
	}

	results, err := h.store.Query(c.Request().Context(), input.Query)
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
