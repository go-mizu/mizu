package api

import (
	"encoding/json"
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type ReviewHandler struct{ st store.Store }

func NewReviewHandler(st store.Store) *ReviewHandler { return &ReviewHandler{st: st} }

func (h *ReviewHandler) GetByBook(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	reviews, total, err := h.st.Review().GetByBook(c.Context(), bookID, page, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{"reviews": reviews, "total": total})
}

func (h *ReviewHandler) Create(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var review types.Review
	if err := c.BindJSON(&review, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	review.BookID = bookID
	if err := h.st.Review().Create(c.Context(), &review); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Add feed entry
	go func() {
		book, _ := h.st.Book().Get(c.Context(), bookID)
		title := ""
		if book != nil {
			title = book.Title
		}
		feedType := "rating"
		if review.Text != "" {
			feedType = "review"
		}
		data, _ := json.Marshal(map[string]any{"rating": review.Rating, "text": review.Text})
		h.st.Feed().Add(c.Context(), &types.FeedItem{
			Type:      feedType,
			BookID:    bookID,
			BookTitle: title,
			Data:      string(data),
		})
	}()

	return c.JSON(201, review)
}

func (h *ReviewHandler) Update(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var review types.Review
	if err := c.BindJSON(&review, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	review.ID = id
	if err := h.st.Review().Update(c.Context(), &review); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, review)
}

func (h *ReviewHandler) Delete(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.st.Review().Delete(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

func (h *ReviewHandler) GetProgress(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	progress, err := h.st.Progress().GetByBook(c.Context(), bookID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, progress)
}

func (h *ReviewHandler) UpdateProgress(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var p types.ReadingProgress
	if err := c.BindJSON(&p, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	p.BookID = bookID
	if err := h.st.Progress().Create(c.Context(), &p); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Add feed entry
	go func() {
		book, _ := h.st.Book().Get(c.Context(), bookID)
		title := ""
		if book != nil {
			title = book.Title
		}
		data, _ := json.Marshal(map[string]any{"page": p.Page, "percent": p.Percent})
		h.st.Feed().Add(c.Context(), &types.FeedItem{
			Type:      "progress",
			BookID:    bookID,
			BookTitle: title,
			Data:      string(data),
		})
	}()

	return c.JSON(201, p)
}
