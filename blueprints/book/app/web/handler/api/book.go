package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/pkg/openlibrary"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type BookHandler struct {
	st store.Store
	ol *openlibrary.Client
}

func NewBookHandler(st store.Store, ol *openlibrary.Client) *BookHandler {
	return &BookHandler{st: st, ol: ol}
}

func (h *BookHandler) Search(c *mizu.Ctx) error {
	q := c.Query("q")
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	// Search local DB first
	result, err := h.st.Book().Search(c.Context(), q, page, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// If few local results, supplement from Open Library
	if result.TotalCount < limit && q != "" {
		olBooks, err := h.ol.Search(c.Context(), q, limit-result.TotalCount)
		if err == nil {
			for _, ob := range olBooks {
				// Skip if already in results
				found := false
				for _, rb := range result.Books {
					if rb.OLKey == ob.OLKey || (rb.ISBN13 != "" && rb.ISBN13 == ob.ISBN13) {
						found = true
						break
					}
				}
				if !found {
					result.Books = append(result.Books, ob)
					result.TotalCount++
				}
			}
		}
	}

	return c.JSON(200, result)
}

func (h *BookHandler) Get(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	book, err := h.st.Book().Get(c.Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if book == nil {
		return c.JSON(404, map[string]string{"error": "book not found"})
	}

	// Get user's review/rating
	if review, _ := h.st.Review().GetUserReview(c.Context(), id); review != nil {
		book.UserRating = review.Rating
	}

	// Get shelf
	if shelves, _ := h.st.Shelf().GetBookShelves(c.Context(), id); len(shelves) > 0 {
		for _, sh := range shelves {
			if sh.IsExclusive {
				book.UserShelf = sh.Slug
				break
			}
		}
	}

	return c.JSON(200, book)
}

func (h *BookHandler) Create(c *mizu.Ctx) error {
	var book types.Book
	if err := c.BindJSON(&book, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	if err := h.st.Book().Create(c.Context(), &book); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, book)
}

func (h *BookHandler) Similar(c *mizu.Ctx) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 10
	}
	books, err := h.st.Book().GetSimilar(c.Context(), id, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, books)
}

func (h *BookHandler) Trending(c *mizu.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	books, err := h.st.Book().GetTrending(c.Context(), limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, books)
}
