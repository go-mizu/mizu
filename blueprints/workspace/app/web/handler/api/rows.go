package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/comments"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/rows"
)

// Row handles database row endpoints.
type Row struct {
	rows      rows.API
	blocks    blocks.API
	comments  comments.API
	databases databases.API
	getUserID func(c *mizu.Ctx) string
}

// NewRow creates a new Row handler.
func NewRow(rows rows.API, blocks blocks.API, comments comments.API, databases databases.API, getUserID func(c *mizu.Ctx) string) *Row {
	return &Row{rows: rows, blocks: blocks, comments: comments, databases: databases, getUserID: getUserID}
}

// Create creates a new row in a database.
func (h *Row) Create(c *mizu.Ctx) error {
	dbID := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Properties map[string]interface{} `json:"properties"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Get database to find workspace ID
	db, err := h.databases.GetByID(c.Request().Context(), dbID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "database not found"})
	}

	row, err := h.rows.Create(c.Request().Context(), &rows.CreateIn{
		DatabaseID:  dbID,
		WorkspaceID: db.WorkspaceID,
		Properties:  in.Properties,
		CreatedBy:   userID,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, row)
}

// Get retrieves a single row.
func (h *Row) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	row, err := h.rows.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "row not found"})
	}

	return c.JSON(http.StatusOK, row)
}

// Update updates a row's properties.
func (h *Row) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Properties map[string]interface{} `json:"properties"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		slog.Error("Row.Update: failed to bind JSON", "error", err, "rowID", id)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request: " + err.Error()})
	}

	// Log the received properties for debugging
	propsJSON, _ := json.Marshal(in.Properties)
	slog.Info("Row.Update: received properties", "rowID", id, "properties", string(propsJSON))

	row, err := h.rows.Update(c.Request().Context(), id, &rows.UpdateIn{
		Properties: in.Properties,
		UpdatedBy:  userID,
	})
	if err != nil {
		slog.Error("Row.Update: failed to update", "error", err, "rowID", id)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	slog.Info("Row.Update: successfully updated", "rowID", id)
	return c.JSON(http.StatusOK, row)
}

// Delete deletes a row.
func (h *Row) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.rows.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// List lists rows in a database with optional filters and sorts.
func (h *Row) List(c *mizu.Ctx) error {
	dbID := c.Param("id")

	var filters []rows.Filter
	var sorts []rows.Sort

	// Parse filters from query param
	if filtersJSON := c.Query("filters"); filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid filters"})
		}
	}

	// Parse sorts from query param
	if sortsJSON := c.Query("sorts"); sortsJSON != "" {
		if err := json.Unmarshal([]byte(sortsJSON), &sorts); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid sorts"})
		}
	}

	cursor := c.Query("cursor")

	result, err := h.rows.List(c.Request().Context(), &rows.ListIn{
		DatabaseID: dbID,
		Filters:    filters,
		Sorts:      sorts,
		Cursor:     cursor,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// Duplicate creates a copy of a row.
func (h *Row) Duplicate(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	row, err := h.rows.DuplicateRow(c.Request().Context(), id, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, row)
}

// =====================================================
// Row Comments - using polymorphic comments
// =====================================================

// ListComments lists comments for a row.
func (h *Row) ListComments(c *mizu.Ctx) error {
	rowID := c.Param("id")

	// Get the row to find the workspace ID
	row, err := h.rows.GetByID(c.Request().Context(), rowID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "row not found"})
	}

	result, err := h.comments.ListByTarget(c.Request().Context(), row.WorkspaceID, comments.TargetDatabaseRow, rowID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// CreateComment creates a comment on a row.
func (h *Row) CreateComment(c *mizu.Ctx) error {
	rowID := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Content  []blocks.RichText `json:"content"`
		ParentID string            `json:"parent_id,omitempty"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Get the row to find the workspace ID
	row, err := h.rows.GetByID(c.Request().Context(), rowID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "row not found"})
	}

	comment, err := h.comments.Create(c.Request().Context(), &comments.CreateIn{
		WorkspaceID: row.WorkspaceID,
		TargetType:  comments.TargetDatabaseRow,
		TargetID:    rowID,
		ParentID:    in.ParentID,
		Content:     in.Content,
		AuthorID:    userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, comment)
}

// UpdateComment updates a row comment.
func (h *Row) UpdateComment(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		Content []blocks.RichText `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	comment, err := h.comments.Update(c.Request().Context(), id, in.Content)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, comment)
}

// DeleteComment deletes a row comment.
func (h *Row) DeleteComment(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// ResolveComment marks a comment as resolved.
func (h *Row) ResolveComment(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Resolve(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "resolved"})
}

// UnresolveComment marks a comment as unresolved.
func (h *Row) UnresolveComment(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Unresolve(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "unresolved"})
}

// =====================================================
// Row Content Blocks - using unified blocks table
// =====================================================

// ListBlocks lists content blocks for a row (row is a page).
func (h *Row) ListBlocks(c *mizu.Ctx) error {
	rowID := c.Param("id")

	// Row ID is the page ID since rows are pages
	result, err := h.blocks.GetByPage(c.Request().Context(), rowID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// CreateBlock creates a content block in a row.
func (h *Row) CreateBlock(c *mizu.Ctx) error {
	rowID := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Type     blocks.BlockType `json:"type"`
		Content  blocks.Content   `json:"content"`
		ParentID string           `json:"parent_id,omitempty"`
		Position int              `json:"position"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	block, err := h.blocks.Create(c.Request().Context(), &blocks.CreateIn{
		PageID:    rowID, // Row ID is the page ID
		ParentID:  in.ParentID,
		Type:      in.Type,
		Content:   in.Content,
		Position:  in.Position,
		CreatedBy: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, block)
}

// UpdateBlock updates a row content block.
func (h *Row) UpdateBlock(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in struct {
		Content blocks.Content `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	block, err := h.blocks.Update(c.Request().Context(), id, &blocks.UpdateIn{
		Content:   in.Content,
		UpdatedBy: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, block)
}

// DeleteBlock deletes a row content block.
func (h *Row) DeleteBlock(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.blocks.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// ReorderBlocks reorders blocks within a row.
func (h *Row) ReorderBlocks(c *mizu.Ctx) error {
	rowID := c.Param("id")

	var in struct {
		BlockIDs []string `json:"block_ids"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	ctx := c.Request().Context()

	// Use the Reorder method to update positions
	// ParentID is empty for top-level blocks in a row
	if err := h.blocks.Reorder(ctx, "", in.BlockIDs); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Return updated blocks
	result, err := h.blocks.GetByPage(ctx, rowID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}
