package api

import (
	"encoding/json"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/pkg/goodreads"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type GoodreadsHandler struct {
	st store.Store
	gr *goodreads.Client
}

func NewGoodreadsHandler(st store.Store) *GoodreadsHandler {
	return &GoodreadsHandler{st: st, gr: goodreads.NewClient()}
}

// GetByGoodreadsID fetches a book from Goodreads by its ID, imports it, and returns it.
func (h *GoodreadsHandler) GetByGoodreadsID(c *mizu.Ctx) error {
	rawID := c.Param("id")
	grID := goodreads.ParseGoodreadsURL(rawID)

	// Check if already imported
	if existing, _ := h.st.Book().GetByGoodreadsID(c.Context(), grID); existing != nil {
		return c.JSON(200, existing)
	}

	// Fetch from Goodreads
	grBook, err := h.gr.GetBook(c.Context(), grID)
	if err != nil {
		return c.JSON(502, map[string]string{"error": err.Error()})
	}

	book := goodreadsToBook(grBook)

	// Check by ISBN too
	if book.ISBN13 != "" {
		if existing, _ := h.st.Book().GetByISBN(c.Context(), book.ISBN13); existing != nil {
			// Update existing with Goodreads data
			mergeGoodreadsData(existing, grBook)
			h.st.Book().Update(c.Context(), existing)
			return c.JSON(200, existing)
		}
	}

	if err := h.st.Book().Create(c.Context(), &book); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Import reviews as quotes (community content)
	for _, q := range grBook.Quotes {
		quote := types.Quote{
			BookID:     book.ID,
			AuthorName: q.AuthorName,
			Text:       q.Text,
			LikesCount: q.LikesCount,
		}
		h.st.Quote().Create(c.Context(), &quote)
	}

	return c.JSON(200, book)
}

// ImportFromURL imports a book from a Goodreads URL.
func (h *GoodreadsHandler) ImportFromURL(c *mizu.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BindJSON(&req, 1<<16); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}

	grID := goodreads.ParseGoodreadsURL(req.URL)
	if grID == "" {
		return c.JSON(400, map[string]string{"error": "invalid Goodreads URL"})
	}

	// Check if already imported
	if existing, _ := h.st.Book().GetByGoodreadsID(c.Context(), grID); existing != nil {
		return c.JSON(200, existing)
	}

	grBook, err := h.gr.GetBook(c.Context(), grID)
	if err != nil {
		return c.JSON(502, map[string]string{"error": err.Error()})
	}

	book := goodreadsToBook(grBook)
	if err := h.st.Book().Create(c.Context(), &book); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, book)
}

func goodreadsToBook(gr *goodreads.GoodreadsBook) types.Book {
	subj, _ := json.Marshal(gr.Genres)
	rdist, _ := json.Marshal(gr.RatingDist)

	return types.Book{
		GoodreadsID:      gr.GoodreadsID,
		Title:            gr.Title,
		AuthorNames:      gr.AuthorName,
		Description:      gr.Description,
		ISBN10:           gr.ISBN,
		ISBN13:           gr.ISBN13,
		ASIN:             gr.ASIN,
		PageCount:        gr.PageCount,
		Format:           gr.Format,
		Publisher:        gr.Publisher,
		PublishDate:      gr.PublishDate,
		FirstPublished:   gr.FirstPublished,
		Language:         gr.Language,
		CoverURL:         gr.CoverURL,
		Series:           gr.Series,
		AverageRating:    gr.AverageRating,
		RatingsCount:     gr.RatingsCount,
		ReviewsCount:     gr.ReviewsCount,
		CurrentlyReading: gr.CurrentlyReading,
		WantToRead:       gr.WantToRead,
		RatingDist:       gr.RatingDist,
		RatingDistJSON:   string(rdist),
		Subjects:         gr.Genres,
		SubjectsJSON:     string(subj),
	}
}

func mergeGoodreadsData(book *types.Book, gr *goodreads.GoodreadsBook) {
	book.GoodreadsID = gr.GoodreadsID
	if book.Description == "" {
		book.Description = gr.Description
	}
	if book.CoverURL == "" {
		book.CoverURL = gr.CoverURL
	}
	book.ASIN = gr.ASIN
	book.Series = gr.Series
	book.AverageRating = gr.AverageRating
	book.RatingsCount = gr.RatingsCount
	book.ReviewsCount = gr.ReviewsCount
	book.CurrentlyReading = gr.CurrentlyReading
	book.WantToRead = gr.WantToRead
	book.RatingDist = gr.RatingDist
	book.FirstPublished = gr.FirstPublished
	if len(gr.Genres) > 0 {
		book.Subjects = gr.Genres
	}
}
