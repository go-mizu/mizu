package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Vectorize handles Vectorize requests.
type Vectorize struct {
	store store.VectorizeStore
}

// NewVectorize creates a new Vectorize handler.
func NewVectorize(store store.VectorizeStore) *Vectorize {
	return &Vectorize{store: store}
}

// ListIndexes lists all vector indexes.
func (h *Vectorize) ListIndexes(c *mizu.Ctx) error {
	indexes, err := h.store.ListIndexes(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  indexes,
	})
}

// CreateVectorIndexInput is the input for creating an index.
type CreateVectorIndexInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Dimensions  int    `json:"dimensions"`
	Metric      string `json:"metric"` // cosine, euclidean, dot-product
}

// CreateIndex creates a new vector index.
func (h *Vectorize) CreateIndex(c *mizu.Ctx) error {
	var input CreateVectorIndexInput
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

	idx := &store.VectorIndex{
		ID:          ulid.Make().String(),
		Name:        input.Name,
		Description: input.Description,
		Dimensions:  input.Dimensions,
		Metric:      input.Metric,
		CreatedAt:   time.Now(),
	}

	if err := h.store.CreateIndex(c.Request().Context(), idx); err != nil {
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
	idx, err := h.store.GetIndex(c.Request().Context(), name)
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
	if err := h.store.DeleteIndex(c.Request().Context(), name); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{
		"success": true,
	})
}

// InsertVectorsInput is the input for inserting vectors.
type InsertVectorsInput struct {
	Vectors []struct {
		ID        string                 `json:"id"`
		Values    []float32              `json:"values"`
		Namespace string                 `json:"namespace"`
		Metadata  map[string]interface{} `json:"metadata"`
	} `json:"vectors"`
}

// InsertVectors inserts vectors into an index.
func (h *Vectorize) InsertVectors(c *mizu.Ctx) error {
	name := c.Param("name")

	var input InsertVectorsInput
	if err := c.BindJSON(&input, 100<<20); err != nil { // 100MB limit for bulk inserts
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	vectors := make([]*store.Vector, len(input.Vectors))
	for i, v := range input.Vectors {
		vectors[i] = &store.Vector{
			ID:        v.ID,
			Values:    v.Values,
			Namespace: v.Namespace,
			Metadata:  v.Metadata,
		}
	}

	if err := h.store.Insert(c.Request().Context(), name, vectors); err != nil {
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

	var input InsertVectorsInput
	if err := c.BindJSON(&input, 100<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	vectors := make([]*store.Vector, len(input.Vectors))
	for i, v := range input.Vectors {
		vectors[i] = &store.Vector{
			ID:        v.ID,
			Values:    v.Values,
			Namespace: v.Namespace,
			Metadata:  v.Metadata,
		}
	}

	if err := h.store.Upsert(c.Request().Context(), name, vectors); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success":      true,
		"mutationId":   ulid.Make().String(),
		"vectorsCount": len(vectors),
	})
}

// VectorQueryInput is the input for querying vectors.
type VectorQueryInput struct {
	Vector         []float32              `json:"vector"`
	TopK           int                    `json:"topK"`
	Namespace      string                 `json:"namespace"`
	Filter         map[string]interface{} `json:"filter"`
	ReturnValues   bool                   `json:"returnValues"`
	ReturnMetadata string                 `json:"returnMetadata"` // none, indexed, all
}

// Query queries the vector index.
func (h *Vectorize) Query(c *mizu.Ctx) error {
	name := c.Param("name")

	var input VectorQueryInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.TopK == 0 {
		input.TopK = 10
	}
	if input.ReturnMetadata == "" {
		input.ReturnMetadata = "none"
	}

	opts := &store.VectorQueryOptions{
		TopK:           input.TopK,
		Namespace:      input.Namespace,
		ReturnValues:   input.ReturnValues,
		ReturnMetadata: input.ReturnMetadata,
		Filter:         input.Filter,
	}

	results, err := h.store.Query(c.Request().Context(), name, input.Vector, opts)
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

// DeleteVectorsInput is the input for deleting vectors.
type DeleteVectorsInput struct {
	IDs       []string `json:"ids"`
	Namespace string   `json:"namespace"`
}

// DeleteVectors deletes vectors by ID.
func (h *Vectorize) DeleteVectors(c *mizu.Ctx) error {
	name := c.Param("name")

	var input DeleteVectorsInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if len(input.IDs) > 0 {
		if err := h.store.DeleteByIDs(c.Request().Context(), name, input.IDs); err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
	} else if input.Namespace != "" {
		if err := h.store.DeleteByNamespace(c.Request().Context(), name, input.Namespace); err != nil {
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

	vectors, err := h.store.GetByIDs(c.Request().Context(), name, input.IDs)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  vectors,
	})
}
