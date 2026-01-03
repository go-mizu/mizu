package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// Page handles page endpoints.
type Page struct {
	pages     pages.API
	blocks    blocks.API
	getUserID func(c *mizu.Ctx) string
}

// NewPage creates a new Page handler.
func NewPage(pages pages.API, blocks blocks.API, getUserID func(c *mizu.Ctx) string) *Page {
	return &Page{pages: pages, blocks: blocks, getUserID: getUserID}
}

// Create creates a new page.
func (h *Page) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in pages.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	in.CreatedBy = userID
	page, err := h.pages.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, page)
}

// Get retrieves a page.
func (h *Page) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	page, err := h.pages.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "page not found"})
	}

	return c.JSON(http.StatusOK, page)
}

// Update updates a page.
func (h *Page) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in pages.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	in.UpdatedBy = userID
	page, err := h.pages.Update(c.Request().Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, page)
}

// Delete deletes a page.
func (h *Page) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.pages.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// List lists pages in a workspace.
func (h *Page) List(c *mizu.Ctx) error {
	workspaceID := c.Param("workspaceID")

	list, err := h.pages.ListByWorkspace(c.Request().Context(), workspaceID, pages.ListOpts{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, list)
}

// GetBlocks retrieves blocks for a page.
func (h *Page) GetBlocks(c *mizu.Ctx) error {
	pageID := c.Param("id")

	blocksList, err := h.blocks.GetByPage(c.Request().Context(), pageID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, blocksList)
}

// Archive archives a page.
func (h *Page) Archive(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.pages.Archive(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "archived"})
}

// Restore restores an archived page.
func (h *Page) Restore(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.pages.Restore(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "restored"})
}

// Duplicate duplicates a page.
func (h *Page) Duplicate(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		ParentID string `json:"parent_id"`
	}
	c.BindJSON(&in, 1<<20)

	page, err := h.pages.Duplicate(c.Request().Context(), id, in.ParentID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, page)
}

// UpdateBlocks updates all blocks for a page.
func (h *Page) UpdateBlocks(c *mizu.Ctx) error {
	pageID := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Blocks []blocks.UpdateIn `json:"blocks"`
	}
	if err := c.BindJSON(&in, 10<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Set updater for each block
	for i := range in.Blocks {
		in.Blocks[i].UpdatedBy = userID
	}

	// Get existing blocks for this page to determine creates vs updates
	existing, err := h.blocks.GetByPage(c.Request().Context(), pageID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	existingIDs := make(map[string]bool)
	for _, b := range existing {
		existingIDs[b.ID] = true
	}

	// Separate into creates and updates
	var creates []*blocks.CreateIn
	var updates []*blocks.UpdateIn
	incomingIDs := make(map[string]bool)

	for i := range in.Blocks {
		block := &in.Blocks[i]
		incomingIDs[block.ID] = true

		if existingIDs[block.ID] {
			updates = append(updates, block)
		} else {
			creates = append(creates, &blocks.CreateIn{
				PageID:    pageID,
				Type:      block.Type,
				Content:   block.Content,
				CreatedBy: userID,
			})
		}
	}

	// Delete blocks that are no longer present
	var deletes []string
	for _, b := range existing {
		if !incomingIDs[b.ID] {
			deletes = append(deletes, b.ID)
		}
	}

	ctx := c.Request().Context()

	if len(deletes) > 0 {
		if err := h.blocks.BatchDelete(ctx, deletes); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	if len(updates) > 0 {
		if err := h.blocks.BatchUpdate(ctx, updates); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	if len(creates) > 0 {
		if _, err := h.blocks.BatchCreate(ctx, creates); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	// Return the updated blocks
	result, err := h.blocks.GetByPage(ctx, pageID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}
