package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/feature/search"
	"github.com/go-mizu/blueprints/social/feature/trending"
)

// Search handles search endpoints.
type Search struct {
	search       search.API
	trending     trending.API
	posts        posts.API
	optionalAuth func(*mizu.Ctx) string
}

// NewSearch creates a new search handler.
func NewSearch(searchSvc search.API, trendingSvc trending.API, postsSvc posts.API, optionalAuth func(*mizu.Ctx) string) *Search {
	return &Search{
		search:       searchSvc,
		trending:     trendingSvc,
		posts:        postsSvc,
		optionalAuth: optionalAuth,
	}
}

// Search handles GET /api/v1/search
func (h *Search) Search(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return BadRequest(c, "query is required")
	}

	limit := IntQuery(c, "limit", 20)
	offset := IntQuery(c, "offset", 0)
	searchType := c.Query("type")

	opts := search.SearchOpts{
		Query:  query,
		Type:   searchType,
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.search.Search(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, result)
}

// TrendingTags handles GET /api/v1/trends/tags
func (h *Search) TrendingTags(c *mizu.Ctx) error {
	limit := IntQuery(c, "limit", 10)
	offset := IntQuery(c, "offset", 0)

	opts := trending.TrendingOpts{
		Limit:  limit,
		Offset: offset,
	}

	tags, err := h.trending.GetTrendingTags(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, tags)
}

// TrendingPosts handles GET /api/v1/trends/posts
func (h *Search) TrendingPosts(c *mizu.Ctx) error {
	limit := IntQuery(c, "limit", 10)
	offset := IntQuery(c, "offset", 0)

	opts := trending.TrendingOpts{
		Limit:  limit,
		Offset: offset,
	}

	ps, err := h.trending.GetTrendingPosts(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, ps)
}
