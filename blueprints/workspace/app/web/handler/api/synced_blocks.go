package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/synced_blocks"
)

// SyncedBlocks handles synced block API endpoints.
type SyncedBlocks struct {
	syncedBlocks synced_blocks.API
	getUserID    func(c *mizu.Ctx) string
}

// NewSyncedBlocks creates a new synced blocks handler.
func NewSyncedBlocks(syncedBlocks synced_blocks.API, getUserID func(c *mizu.Ctx) string) *SyncedBlocks {
	return &SyncedBlocks{
		syncedBlocks: syncedBlocks,
		getUserID:    getUserID,
	}
}

// Get retrieves a synced block by ID.
func (h *SyncedBlocks) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	sb, err := h.syncedBlocks.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "synced block not found"})
	}

	// Map to frontend expected format
	response := map[string]interface{}{
		"id":          sb.ID,
		"pageId":      sb.PageID,
		"pageName":    sb.PageName,
		"lastUpdated": sb.LastUpdated,
		"content":     mapBlocksToContent(sb.Content),
	}

	return c.JSON(http.StatusOK, response)
}

// Create creates a new synced block.
func (h *SyncedBlocks) Create(c *mizu.Ctx) error {
	var in synced_blocks.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	in.CreatedBy = h.getUserID(c)

	sb, err := h.syncedBlocks.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, sb)
}

// Update updates a synced block.
func (h *SyncedBlocks) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in synced_blocks.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	sb, err := h.syncedBlocks.Update(c.Request().Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, sb)
}

// Delete removes a synced block.
func (h *SyncedBlocks) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.syncedBlocks.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// List returns synced blocks for a page.
func (h *SyncedBlocks) ListByPage(c *mizu.Ctx) error {
	pageID := c.Param("pageID")

	sbs, err := h.syncedBlocks.ListByPage(c.Request().Context(), pageID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"synced_blocks": sbs})
}

// ListByWorkspace returns all synced blocks in a workspace.
func (h *SyncedBlocks) ListByWorkspace(c *mizu.Ctx) error {
	workspaceID := c.Param("workspaceID")

	sbs, err := h.syncedBlocks.ListByWorkspace(c.Request().Context(), workspaceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"synced_blocks": sbs})
}

// mapBlocksToContent converts blocks to frontend content format.
func mapBlocksToContent(blks []blocks.Block) []map[string]interface{} {
	var result []map[string]interface{}
	for _, b := range blks {
		content := map[string]interface{}{
			"id":   b.ID,
			"type": string(b.Type),
		}

		// Map content fields
		if b.Content.RichText != nil {
			richText := make([]map[string]interface{}, len(b.Content.RichText))
			for i, rt := range b.Content.RichText {
				richText[i] = map[string]interface{}{
					"text":        rt.Text,
					"annotations": rt.Annotations,
				}
			}
			content["content"] = map[string]interface{}{
				"rich_text": richText,
				"checked":   b.Content.Checked,
				"icon":      b.Content.Icon,
				"color":     b.Content.Color,
				"language":  b.Content.Language,
				"url":       b.Content.URL,
			}
		}

		result = append(result, content)
	}
	return result
}
