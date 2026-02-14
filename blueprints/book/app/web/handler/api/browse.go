package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
)

type BrowseHandler struct{ st store.Store }

func NewBrowseHandler(st store.Store) *BrowseHandler { return &BrowseHandler{st: st} }

func (h *BrowseHandler) ListGenres(c *mizu.Ctx) error {
	genres, err := h.st.Book().ListGenres(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, genres)
}

func (h *BrowseHandler) BooksByGenre(c *mizu.Ctx) error {
	genre := c.Param("genre")
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	result, err := h.st.Book().GetByGenre(c.Context(), genre, page, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, result)
}

func (h *BrowseHandler) NewReleases(c *mizu.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	books, err := h.st.Book().GetNewReleases(c.Context(), limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, books)
}

func (h *BrowseHandler) Popular(c *mizu.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	books, err := h.st.Book().GetPopular(c.Context(), limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, books)
}
