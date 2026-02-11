package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/pkg/openlibrary"
	"github.com/go-mizu/mizu/blueprints/book/store"
)

type AuthorHandler struct {
	st store.Store
	ol *openlibrary.Client
}

func NewAuthorHandler(st store.Store, ol *openlibrary.Client) *AuthorHandler {
	return &AuthorHandler{st: st, ol: ol}
}

func (h *AuthorHandler) Search(c *mizu.Ctx) error {
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	authors, err := h.st.Author().Search(c.Context(), q, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, authors)
}

func (h *AuthorHandler) Get(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	author, err := h.st.Author().Get(c.Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if author == nil {
		return c.JSON(404, map[string]string{"error": "author not found"})
	}
	return c.JSON(200, author)
}

func (h *AuthorHandler) Books(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	result, err := h.st.Author().GetBooks(c.Context(), id, page, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, result)
}
