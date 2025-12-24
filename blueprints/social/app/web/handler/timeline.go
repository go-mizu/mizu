package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/timelines"
)

// Timeline handles timeline endpoints.
type Timeline struct {
	timelines    timelines.API
	getAccountID func(*mizu.Ctx) string
	optionalAuth func(*mizu.Ctx) string
}

// NewTimeline creates a new timeline handler.
func NewTimeline(timelinesSvc timelines.API, getAccountID func(*mizu.Ctx) string, optionalAuth func(*mizu.Ctx) string) *Timeline {
	return &Timeline{
		timelines:    timelinesSvc,
		getAccountID: getAccountID,
		optionalAuth: optionalAuth,
	}
}

// Home handles GET /api/v1/timelines/home
func (h *Timeline) Home(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	opts := h.parseOpts(c)

	posts, err := h.timelines.Home(c.Request().Context(), accountID, opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, posts)
}

// Public handles GET /api/v1/timelines/public
func (h *Timeline) Public(c *mizu.Ctx) error {
	opts := h.parseOpts(c)

	posts, err := h.timelines.Public(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, posts)
}

// Hashtag handles GET /api/v1/timelines/tag/:tag
func (h *Timeline) Hashtag(c *mizu.Ctx) error {
	tag := c.Param("tag")
	opts := h.parseOpts(c)

	posts, err := h.timelines.Hashtag(c.Request().Context(), tag, opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, posts)
}

// List handles GET /api/v1/timelines/list/:id
func (h *Timeline) List(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	listID := c.Param("id")
	opts := h.parseOpts(c)

	posts, err := h.timelines.List(c.Request().Context(), accountID, listID, opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, posts)
}

// Bookmarks handles GET /api/v1/bookmarks
func (h *Timeline) Bookmarks(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	opts := h.parseOpts(c)

	posts, err := h.timelines.Bookmarks(c.Request().Context(), accountID, opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, posts)
}

func (h *Timeline) parseOpts(c *mizu.Ctx) timelines.TimelineOpts {
	limit := IntQuery(c, "limit", 20)
	return timelines.TimelineOpts{
		Limit:     limit,
		MaxID:     c.Query("max_id"),
		MinID:     c.Query("min_id"),
		SinceID:   c.Query("since_id"),
		OnlyMedia: BoolQuery(c, "only_media", false),
	}
}
