package handlers

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/timelines"
)

// TimelineHandlers contains timeline-related handlers.
type TimelineHandlers struct {
	timelines    timelines.API
	getAccountID func(*mizu.Ctx) string
	optionalAuth func(*mizu.Ctx) string
}

// NewTimelineHandlers creates new timeline handlers.
func NewTimelineHandlers(
	timelines timelines.API,
	getAccountID func(*mizu.Ctx) string,
	optionalAuth func(*mizu.Ctx) string,
) *TimelineHandlers {
	return &TimelineHandlers{
		timelines:    timelines,
		getAccountID: getAccountID,
		optionalAuth: optionalAuth,
	}
}

// Home returns the home timeline.
func (h *TimelineHandlers) Home(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	limit := IntQuery(c, "limit", 20)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	postList, err := h.timelines.Home(c.Request().Context(), accountID, limit, maxID, sinceID)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

// Local returns the local timeline.
func (h *TimelineHandlers) Local(c *mizu.Ctx) error {
	viewerID := h.optionalAuth(c)
	limit := IntQuery(c, "limit", 20)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	postList, err := h.timelines.Local(c.Request().Context(), viewerID, limit, maxID, sinceID)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

// Hashtag returns the hashtag timeline.
func (h *TimelineHandlers) Hashtag(c *mizu.Ctx) error {
	tag := c.Param("tag")
	viewerID := h.optionalAuth(c)
	limit := IntQuery(c, "limit", 20)
	maxID := c.Query("max_id")
	sinceID := c.Query("since_id")

	postList, err := h.timelines.Hashtag(c.Request().Context(), tag, viewerID, limit, maxID, sinceID)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

// Bookmarks returns the user's bookmarks.
func (h *TimelineHandlers) Bookmarks(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	limit := IntQuery(c, "limit", 20)
	maxID := c.Query("max_id")

	postList, err := h.timelines.Bookmarks(c.Request().Context(), accountID, limit, maxID)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}
