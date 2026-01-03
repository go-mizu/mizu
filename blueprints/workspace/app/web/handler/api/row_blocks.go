package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/rowblocks"
)

// RowBlock handles row content block endpoints.
type RowBlock struct {
	blocks rowblocks.API
}

// NewRowBlock creates a new RowBlock handler.
func NewRowBlock(blocks rowblocks.API) *RowBlock {
	return &RowBlock{blocks: blocks}
}

// Create creates a new content block for a row.
func (h *RowBlock) Create(c *mizu.Ctx) error {
	rowID := c.Param("id")

	var in struct {
		Type       string                 `json:"type"`
		Content    string                 `json:"content"`
		ParentID   string                 `json:"parent_id,omitempty"`
		AfterID    string                 `json:"after_id,omitempty"`
		Properties map[string]interface{} `json:"properties,omitempty"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if in.Type == "" {
		in.Type = "paragraph"
	}

	block, err := h.blocks.Create(c.Request().Context(), &rowblocks.CreateIn{
		RowID:      rowID,
		Type:       rowblocks.BlockType(in.Type),
		Content:    in.Content,
		ParentID:   in.ParentID,
		AfterID:    in.AfterID,
		Properties: in.Properties,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, block)
}

// List lists all content blocks for a row.
func (h *RowBlock) List(c *mizu.Ctx) error {
	rowID := c.Param("id")

	blocks, err := h.blocks.ListByRow(c.Request().Context(), rowID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if blocks == nil {
		blocks = []*rowblocks.Block{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"blocks": blocks})
}

// Get retrieves a single block.
func (h *RowBlock) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	block, err := h.blocks.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "block not found"})
	}

	return c.JSON(http.StatusOK, block)
}

// Update updates a content block.
func (h *RowBlock) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		Content    string                 `json:"content,omitempty"`
		Properties map[string]interface{} `json:"properties,omitempty"`
		Checked    *bool                  `json:"checked,omitempty"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	block, err := h.blocks.Update(c.Request().Context(), id, &rowblocks.UpdateIn{
		Content:    in.Content,
		Properties: in.Properties,
		Checked:    in.Checked,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, block)
}

// Delete deletes a content block.
func (h *RowBlock) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.blocks.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// Reorder updates the order of blocks in a row.
func (h *RowBlock) Reorder(c *mizu.Ctx) error {
	rowID := c.Param("id")

	var in struct {
		BlockIDs []string `json:"block_ids"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if err := h.blocks.Reorder(c.Request().Context(), rowID, &rowblocks.ReorderIn{
		BlockIDs: in.BlockIDs,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "reordered"})
}
