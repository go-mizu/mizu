package handler

import (
	"github.com/go-mizu/blueprints/forum/feature/forums"
	"github.com/go-mizu/mizu"
)

// Forum contains forum-related handlers.
type Forum struct {
	forums       forums.API
	getAccountID func(*mizu.Ctx) string
	optionalAuth func(*mizu.Ctx) string
}

// NewForum creates new forum handlers.
func NewForum(
	forums forums.API,
	getAccountID func(*mizu.Ctx) string,
	optionalAuth func(*mizu.Ctx) string,
) *Forum {
	return &Forum{
		forums:       forums,
		getAccountID: getAccountID,
		optionalAuth: optionalAuth,
	}
}

// List returns all forums.
func (h *Forum) List(c *mizu.Ctx) error {
	viewerID := h.optionalAuth(c)
	parentID := c.Query("parent_id")

	var forumsList []*forums.Forum
	var err error

	if parentID != "" {
		forumsList, err = h.forums.List(c.Request().Context(), parentID, viewerID)
	} else {
		forumsList, err = h.forums.ListAll(c.Request().Context(), viewerID)
	}

	if err != nil {
		return c.JSON(500, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"forums": forumsList,
		"total":  len(forumsList),
	}))
}

// Get returns a specific forum.
func (h *Forum) Get(c *mizu.Ctx) error {
	idOrSlug := c.Param("id")
	viewerID := h.optionalAuth(c)

	// Try by ID first
	forum, err := h.forums.GetByID(c.Request().Context(), idOrSlug, viewerID)
	if err != nil {
		// Try by slug
		forum, err = h.forums.GetBySlug(c.Request().Context(), idOrSlug, viewerID)
		if err != nil {
			return c.JSON(404, ErrorResponse("Forum not found"))
		}
	}

	return c.JSON(200, DataResponse(map[string]any{
		"forum": forum,
	}))
}

// Create creates a new forum.
func (h *Forum) Create(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)

	var in forums.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	forum, err := h.forums.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(201, DataResponse(map[string]any{
		"forum": forum,
	}))
}

// Update updates a forum.
func (h *Forum) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	var in forums.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	forum, err := h.forums.Update(c.Request().Context(), id, accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"forum": forum,
	}))
}

// Delete deletes a forum.
func (h *Forum) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.forums.Delete(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Forum deleted successfully",
	}))
}

// Join joins a forum.
func (h *Forum) Join(c *mizu.Ctx) error {
	forumID := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.forums.Join(c.Request().Context(), forumID, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Joined forum successfully",
	}))
}

// Leave leaves a forum.
func (h *Forum) Leave(c *mizu.Ctx) error {
	forumID := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.forums.Leave(c.Request().Context(), forumID, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Left forum successfully",
	}))
}

// AddModerator adds a moderator to a forum.
func (h *Forum) AddModerator(c *mizu.Ctx) error {
	forumID := c.Param("id")
	accountID := h.getAccountID(c)

	var in struct {
		AccountID string `json:"account_id"`
		Role      string `json:"role"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	if err := h.forums.AddModerator(c.Request().Context(), forumID, accountID, in.AccountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Moderator added successfully",
	}))
}

// RemoveModerator removes a moderator from a forum.
func (h *Forum) RemoveModerator(c *mizu.Ctx) error {
	forumID := c.Param("id")
	accountID := h.getAccountID(c)
	moderatorID := c.Param("account_id")

	if err := h.forums.RemoveModerator(c.Request().Context(), forumID, accountID, moderatorID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Moderator removed successfully",
	}))
}
