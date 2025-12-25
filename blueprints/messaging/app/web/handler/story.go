package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/messaging/feature/stories"
)

// Story handles story endpoints.
type Story struct {
	stories   stories.API
	getUserID func(*mizu.Ctx) string
}

// NewStory creates a new Story handler.
func NewStory(stories stories.API, getUserID func(*mizu.Ctx) string) *Story {
	return &Story{stories: stories, getUserID: getUserID}
}

// List lists stories from contacts.
func (h *Story) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	storyList, err := h.stories.List(c.Request().Context(), userID)
	if err != nil {
		return InternalError(c, "Failed to list stories")
	}

	return Success(c, storyList)
}

// Create creates a new story.
func (h *Story) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	var in stories.CreateIn
	if err := c.BindJSON(&in, 10<<20); err != nil { // 10MB for stories with media
		return BadRequest(c, "Invalid request body")
	}

	story, err := h.stories.Create(c.Request().Context(), userID, &in)
	if err != nil {
		return InternalError(c, "Failed to create story")
	}

	return Created(c, story)
}

// Get retrieves a story.
func (h *Story) Get(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	storyID := c.Param("id")
	story, err := h.stories.GetByID(c.Request().Context(), storyID)
	if err != nil {
		if err == stories.ErrExpired {
			return NotFound(c, "Story expired")
		}
		return NotFound(c, "Story not found")
	}

	return Success(c, story)
}

// Delete deletes a story.
func (h *Story) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	storyID := c.Param("id")
	if err := h.stories.Delete(c.Request().Context(), storyID, userID); err != nil {
		if err == stories.ErrForbidden {
			return Forbidden(c, "Cannot delete this story")
		}
		return InternalError(c, "Failed to delete story")
	}

	return Success(c, nil)
}

// View records a story view.
func (h *Story) View(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	storyID := c.Param("id")
	if err := h.stories.View(c.Request().Context(), storyID, userID); err != nil {
		return InternalError(c, "Failed to record view")
	}

	return Success(c, nil)
}

// GetViewers gets story viewers.
func (h *Story) GetViewers(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	storyID := c.Param("id")
	limit, _ := strconv.Atoi(c.Query("limit"))

	viewers, err := h.stories.GetViewers(c.Request().Context(), storyID, limit)
	if err != nil {
		return InternalError(c, "Failed to get viewers")
	}

	return Success(c, viewers)
}
