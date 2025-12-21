package handlers

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/feature/search"
	"github.com/go-mizu/blueprints/microblog/feature/trending"
)

// SearchHandlers contains search and trending-related handlers.
type SearchHandlers struct {
	search       search.API
	trending     trending.API
	posts        posts.API
	optionalAuth func(*mizu.Ctx) string
}

// NewSearchHandlers creates new search handlers.
func NewSearchHandlers(
	search search.API,
	trending trending.API,
	posts posts.API,
	optionalAuth func(*mizu.Ctx) string,
) *SearchHandlers {
	return &SearchHandlers{
		search:       search,
		trending:     trending,
		posts:        posts,
		optionalAuth: optionalAuth,
	}
}

// Search performs a search.
func (h *SearchHandlers) Search(c *mizu.Ctx) error {
	query := c.Query("q")
	limit := IntQuery(c, "limit", 25)
	viewerID := h.optionalAuth(c)

	results, err := h.search.Search(c.Request().Context(), query, nil, limit, viewerID)
	if err != nil {
		return c.JSON(500, ErrorResponse("SEARCH_FAILED", err.Error()))
	}

	// Group results by type
	var accountResults, hashtagResults, postResults []*search.Result
	for _, r := range results {
		switch r.Type {
		case search.ResultTypeAccount:
			accountResults = append(accountResults, r)
		case search.ResultTypeHashtag:
			hashtagResults = append(hashtagResults, r)
		case search.ResultTypePost:
			postResults = append(postResults, r)
		}
	}

	return c.JSON(200, map[string]any{
		"data": map[string]any{
			"accounts": accountResults,
			"hashtags": hashtagResults,
			"posts":    postResults,
		},
	})
}

// TrendingTags returns trending hashtags.
func (h *SearchHandlers) TrendingTags(c *mizu.Ctx) error {
	limit := IntQuery(c, "limit", 10)
	tags, err := h.trending.Tags(c.Request().Context(), limit)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]any{"data": tags})
}

// TrendingPosts returns trending posts.
func (h *SearchHandlers) TrendingPosts(c *mizu.Ctx) error {
	limit := IntQuery(c, "limit", 20)
	viewerID := h.optionalAuth(c)

	ids, err := h.trending.Posts(c.Request().Context(), limit)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	var postList []*posts.Post
	for _, id := range ids {
		if p, err := h.posts.GetByID(c.Request().Context(), id, viewerID); err == nil {
			postList = append(postList, p)
		}
	}

	return c.JSON(200, map[string]any{"data": postList})
}
