package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type QuoteHandler struct{ st store.Store }

func NewQuoteHandler(st store.Store) *QuoteHandler { return &QuoteHandler{st: st} }

func (h *QuoteHandler) GetAll(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	quotes, total, err := h.st.Quote().GetAll(c.Context(), page, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{"quotes": quotes, "total": total})
}

func (h *QuoteHandler) Create(c *mizu.Ctx) error {
	var q types.Quote
	if err := c.BindJSON(&q, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	if err := h.st.Quote().Create(c.Context(), &q); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, q)
}

func (h *QuoteHandler) GetByBook(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	quotes, err := h.st.Quote().GetByBook(c.Context(), bookID, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, quotes)
}
