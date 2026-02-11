package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type ListHandler struct{ st store.Store }

func NewListHandler(st store.Store) *ListHandler { return &ListHandler{st: st} }

func (h *ListHandler) GetAll(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	lists, total, err := h.st.List().GetAll(c.Context(), page, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{"lists": lists, "total": total})
}

func (h *ListHandler) Create(c *mizu.Ctx) error {
	var list types.BookList
	if err := c.BindJSON(&list, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	if err := h.st.List().Create(c.Context(), &list); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, list)
}

func (h *ListHandler) Get(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	list, err := h.st.List().Get(c.Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if list == nil {
		return c.JSON(404, map[string]string{"error": "list not found"})
	}
	return c.JSON(200, list)
}

func (h *ListHandler) AddBook(c *mizu.Ctx) error {
	listID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		BookID   int64 `json:"book_id"`
		Position int   `json:"position"`
	}
	if err := c.BindJSON(&body, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	if err := h.st.List().AddBook(c.Context(), listID, body.BookID, body.Position); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "added"})
}

func (h *ListHandler) Vote(c *mizu.Ctx) error {
	listID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	bookID, _ := strconv.ParseInt(c.Param("bookId"), 10, 64)
	if err := h.st.List().Vote(c.Context(), listID, bookID); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "voted"})
}
