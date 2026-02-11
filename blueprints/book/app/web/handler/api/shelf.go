package api

import (
	"encoding/json"
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type ShelfHandler struct{ st store.Store }

func NewShelfHandler(st store.Store) *ShelfHandler { return &ShelfHandler{st: st} }

func (h *ShelfHandler) List(c *mizu.Ctx) error {
	shelves, err := h.st.Shelf().List(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, shelves)
}

func (h *ShelfHandler) Create(c *mizu.Ctx) error {
	var shelf types.Shelf
	if err := c.BindJSON(&shelf, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	if err := h.st.Shelf().Create(c.Context(), &shelf); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, shelf)
}

func (h *ShelfHandler) Update(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var shelf types.Shelf
	if err := c.BindJSON(&shelf, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	shelf.ID = id
	if err := h.st.Shelf().Update(c.Context(), &shelf); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, shelf)
}

func (h *ShelfHandler) Delete(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.st.Shelf().Delete(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

func (h *ShelfHandler) GetBooks(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	sort := c.Query("sort")
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	books, total, err := h.st.Shelf().GetBooks(c.Context(), id, sort, page, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{"books": books, "total": total, "page": page})
}

func (h *ShelfHandler) AddBook(c *mizu.Ctx) error {
	shelfID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		BookID int64 `json:"book_id"`
	}
	if err := c.BindJSON(&body, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	if err := h.st.Shelf().AddBook(c.Context(), shelfID, body.BookID); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Add feed entry
	go func() {
		book, _ := h.st.Book().Get(c.Context(), body.BookID)
		shelf, _ := h.st.Shelf().Get(c.Context(), shelfID)
		title := ""
		if book != nil {
			title = book.Title
		}
		shelfName := ""
		if shelf != nil {
			shelfName = shelf.Name
		}
		data, _ := json.Marshal(map[string]any{"shelf_name": shelfName})
		h.st.Feed().Add(c.Context(), &types.FeedItem{
			Type:      "shelve",
			BookID:    body.BookID,
			BookTitle: title,
			Data:      string(data),
		})
	}()

	return c.JSON(200, map[string]string{"status": "added"})
}

func (h *ShelfHandler) RemoveBook(c *mizu.Ctx) error {
	shelfID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	bookID, _ := strconv.ParseInt(c.Param("bookId"), 10, 64)
	if err := h.st.Shelf().RemoveBook(c.Context(), shelfID, bookID); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "removed"})
}
