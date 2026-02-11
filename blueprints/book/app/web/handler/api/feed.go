package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
)

type FeedHandler struct{ st store.Store }

func NewFeedHandler(st store.Store) *FeedHandler { return &FeedHandler{st: st} }

func (h *FeedHandler) Recent(c *mizu.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	items, err := h.st.Feed().GetRecent(c.Context(), limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, items)
}
