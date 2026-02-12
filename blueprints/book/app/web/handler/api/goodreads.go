package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
			mergeGoodreadsData(existing, grBook)
			h.st.Book().Update(c.Context(), existing)
			h.importGoodreadsContent(c, existing.ID, grBook)
			return c.JSON(200, existing)
		}
	}

	if err := h.st.Book().Create(c.Context(), &book); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	h.importGoodreadsContent(c, book.ID, grBook)
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

	h.importGoodreadsContent(c, book.ID, grBook)
	return c.JSON(201, book)
}

// ImportAuthor fetches an author from Goodreads and imports them.
func (h *GoodreadsHandler) ImportAuthor(c *mizu.Ctx) error {
	rawID := c.Param("id")
	grID := goodreads.ParseGoodreadsAuthorURL(rawID)
	if grID == "" {
		return c.JSON(400, map[string]string{"error": "invalid Goodreads author ID"})
	}

	// Check if already imported
	if existing, _ := h.st.Author().GetByGoodreadsID(c.Context(), grID); existing != nil {
		return c.JSON(200, existing)
	}

	grAuthor, err := h.gr.GetAuthor(c.Context(), grID)
	if err != nil {
		return c.JSON(502, map[string]string{"error": err.Error()})
	}

	author := goodreadsToAuthor(grAuthor)
	if err := h.st.Author().Create(c.Context(), &author); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, author)
}

// ImportList fetches a Goodreads list by URL and imports it.
func (h *GoodreadsHandler) ImportList(c *mizu.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BindJSON(&req, 1<<16); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}

	grList, err := h.gr.GetList(c.Context(), req.URL)
	if err != nil {
		return c.JSON(502, map[string]string{"error": err.Error()})
	}

	list := types.BookList{
		Title:        grList.Title,
		Description:  grList.Description,
		GoodreadsURL: req.URL,
		VoterCount:   grList.VoterCount,
	}
	if err := h.st.List().Create(c.Context(), &list); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Import books from the list
	for i, grItem := range grList.Books {
		book := types.Book{
			Title:         grItem.Title,
			AuthorNames:   grItem.AuthorName,
			CoverURL:      grItem.CoverURL,
			AverageRating: grItem.AverageRating,
			RatingsCount:  grItem.RatingsCount,
		}
		if err := h.st.Book().Create(c.Context(), &book); err != nil {
			continue
		}
		h.st.List().AddBook(c.Context(), list.ID, book.ID, i+1)
	}

	return c.JSON(201, list)
}

// EnrichBook enriches an existing book with Goodreads data (reviews, quotes, metadata).
func (h *GoodreadsHandler) EnrichBook(c *mizu.Ctx) error {
	rawID := c.Param("id")
	var bookID int64
	if _, err := fmt.Sscanf(rawID, "%d", &bookID); err != nil || bookID == 0 {
		return c.JSON(400, map[string]string{"error": "invalid book ID"})
	}

	book, err := h.st.Book().Get(c.Context(), bookID)
	if err != nil || book == nil {
		return c.JSON(404, map[string]string{"error": "book not found"})
	}

	// Find on Goodreads: by goodreads_id, ISBN, or title search
	var grID string
	if book.GoodreadsID != "" {
		grID = book.GoodreadsID
	} else if book.ISBN13 != "" {
		if id, err := h.gr.SearchBook(c.Context(), book.ISBN13); err == nil {
			grID = id
		}
	}
	if grID == "" {
		if id, err := h.gr.SearchBook(c.Context(), book.Title); err == nil {
			grID = id
		}
	}
	if grID == "" {
		return c.JSON(404, map[string]string{"error": "book not found on Goodreads"})
	}

	grBook, err := h.gr.GetBook(c.Context(), grID)
	if err != nil {
		return c.JSON(502, map[string]string{"error": err.Error()})
	}

	// Update book metadata
	mergeGoodreadsData(book, grBook)
	h.st.Book().Update(c.Context(), book)

	// Import reviews and quotes
	h.importGoodreadsContent(c, book.ID, grBook)

	return c.JSON(200, book)
}

// BrowseLists fetches popular lists from Goodreads.
func (h *GoodreadsHandler) BrowseLists(c *mizu.Ctx) error {
	lists, err := h.gr.GetPopularLists(c.Context())
	if err != nil {
		return c.JSON(502, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, lists)
}

// importGoodreadsContent imports reviews and quotes from a scraped Goodreads book.
func (h *GoodreadsHandler) importGoodreadsContent(c *mizu.Ctx, bookID int64, gr *goodreads.GoodreadsBook) {
	// Import reviews
	for _, r := range gr.Reviews {
		review := types.Review{
			BookID:       bookID,
			Rating:       r.Rating,
			Text:         r.Text,
			LikesCount:   r.LikesCount,
			ReviewerName: r.ReviewerName,
			Source:       "goodreads",
		}
		if t := parseGoodreadsDate(r.Date); t != nil {
			review.CreatedAt = *t
			review.UpdatedAt = *t
		}
		h.st.Review().Create(c.Context(), &review)
	}

	// Import quotes
	for _, q := range gr.Quotes {
		quote := types.Quote{
			BookID:     bookID,
			AuthorName: q.AuthorName,
			Text:       q.Text,
			LikesCount: q.LikesCount,
		}
		h.st.Quote().Create(c.Context(), &quote)
	}
}

func parseGoodreadsDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	layouts := []string{
		"January 2, 2006",
		"January 2 2006",
		"Jan 2, 2006",
		"Jan 2 2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t
		}
	}
	return nil
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

func goodreadsToAuthor(gr *goodreads.GoodreadsAuthor) types.Author {
	return types.Author{
		Name:        gr.Name,
		Bio:         gr.Bio,
		PhotoURL:    gr.PhotoURL,
		BirthDate:   gr.BornDate,
		DeathDate:   gr.DiedDate,
		WorksCount:  gr.WorksCount,
		GoodreadsID: gr.GoodreadsID,
		Followers:   gr.Followers,
		Genres:      gr.Genres,
		Influences:  gr.Influences,
	}
}
