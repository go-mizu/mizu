package api

import (
	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/feature/vectorize"
)

// Vectorize handles Vectorize requests.
type Vectorize struct {
	svc vectorize.API
}

// NewVectorize creates a new Vectorize handler.
func NewVectorize(svc vectorize.API) *Vectorize {
	return &Vectorize{svc: svc}
}

// ListIndexes lists all vector indexes.
func (h *Vectorize) ListIndexes(c *mizu.Ctx) error {
	indexes, err := h.svc.ListIndexes(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  indexes,
	})
}

// CreateIndex creates a new vector index.
func (h *Vectorize) CreateIndex(c *mizu.Ctx) error {
	var input vectorize.CreateIndexIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "name is required"})
	}

	// Set defaults
	if input.Dimensions == 0 {
		input.Dimensions = 768
	}
	if input.Metric == "" {
		input.Metric = "cosine"
	}

	idx, err := h.svc.CreateIndex(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  idx,
	})
}

// GetIndex retrieves an index by name.
func (h *Vectorize) GetIndex(c *mizu.Ctx) error {
	name := c.Param("name")
	idx, err := h.svc.GetIndex(c.Request().Context(), name)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Index not found"})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  idx,
	})
}

// DeleteIndex deletes an index.
func (h *Vectorize) DeleteIndex(c *mizu.Ctx) error {
	name := c.Param("name")
	if err := h.svc.DeleteIndex(c.Request().Context(), name); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
	})
}

// InsertVectors inserts vectors into an index.
func (h *Vectorize) InsertVectors(c *mizu.Ctx) error {
	name := c.Param("name")

	var input struct {
		Vectors []struct {
			ID        string                 `json:"id"`
			Values    []float32              `json:"values"`
			Namespace string                 `json:"namespace"`
			Metadata  map[string]interface{} `json:"metadata"`
		} `json:"vectors"`
	}
	if err := c.BindJSON(&input, 100<<20); err != nil { // 100MB limit for bulk inserts
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	vectors := make([]*vectorize.Vector, len(input.Vectors))
	for i, v := range input.Vectors {
		vectors[i] = &vectorize.Vector{
			ID:        v.ID,
			Values:    v.Values,
			Namespace: v.Namespace,
			Metadata:  v.Metadata,
		}
	}

	if err := h.svc.InsertVectors(c.Request().Context(), name, vectors); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success":      true,
		"mutationId":   ulid.Make().String(),
		"vectorsCount": len(vectors),
	})
}

// UpsertVectors upserts vectors into an index.
func (h *Vectorize) UpsertVectors(c *mizu.Ctx) error {
	name := c.Param("name")

	var input struct {
		Vectors []struct {
			ID        string                 `json:"id"`
			Values    []float32              `json:"values"`
			Namespace string                 `json:"namespace"`
			Metadata  map[string]interface{} `json:"metadata"`
		} `json:"vectors"`
	}
	if err := c.BindJSON(&input, 100<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	vectors := make([]*vectorize.Vector, len(input.Vectors))
	for i, v := range input.Vectors {
		vectors[i] = &vectorize.Vector{
			ID:        v.ID,
			Values:    v.Values,
			Namespace: v.Namespace,
			Metadata:  v.Metadata,
		}
	}

	if err := h.svc.UpsertVectors(c.Request().Context(), name, vectors); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success":      true,
		"mutationId":   ulid.Make().String(),
		"vectorsCount": len(vectors),
	})
}

// Query queries the vector index.
func (h *Vectorize) Query(c *mizu.Ctx) error {
	name := c.Param("name")

	var input vectorize.QueryIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.TopK == 0 {
		input.TopK = 10
	}
	if input.ReturnMetadata == "" {
		input.ReturnMetadata = "none"
	}

	results, err := h.svc.Query(c.Request().Context(), name, &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Format response
	matches := make([]map[string]any, len(results))
	for i, r := range results {
		match := map[string]any{
			"id":    r.ID,
			"score": r.Score,
		}
		if input.ReturnValues && r.Values != nil {
			match["values"] = r.Values
		}
		if input.ReturnMetadata != "none" && r.Metadata != nil {
			match["metadata"] = r.Metadata
		}
		matches[i] = match
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"count":   len(matches),
			"matches": matches,
		},
	})
}

// DeleteVectors deletes vectors by ID.
func (h *Vectorize) DeleteVectors(c *mizu.Ctx) error {
	name := c.Param("name")

	var input struct {
		IDs       []string `json:"ids"`
		Namespace string   `json:"namespace"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if len(input.IDs) > 0 {
		if err := h.svc.DeleteByIDs(c.Request().Context(), name, input.IDs); err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(200, map[string]any{
		"success":    true,
		"mutationId": ulid.Make().String(),
	})
}

// GetByIDs retrieves vectors by their IDs.
func (h *Vectorize) GetByIDs(c *mizu.Ctx) error {
	name := c.Param("name")

	var input struct {
		IDs []string `json:"ids"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	vectors, err := h.svc.GetByIDs(c.Request().Context(), name, input.IDs)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  vectors,
	})
}
