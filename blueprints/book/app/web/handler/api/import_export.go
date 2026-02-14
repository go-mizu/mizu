package api

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type ImportExportHandler struct{ st store.Store }

func NewImportExportHandler(st store.Store) *ImportExportHandler {
	return &ImportExportHandler{st: st}
}

func (h *ImportExportHandler) ImportCSV(c *mizu.Ctx) error {
	file, _, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "no file uploaded"})
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid CSV"})
	}
	if len(records) < 2 {
		return c.JSON(400, map[string]string{"error": "CSV is empty"})
	}

	header := records[0]
	cols := make(map[string]int)
	for i, h := range header {
		cols[strings.TrimSpace(h)] = i
	}

	imported := 0
	for _, row := range records[1:] {
		book := &types.Book{}
		if i, ok := cols["Title"]; ok && i < len(row) {
			book.Title = row[i]
		}
		if i, ok := cols["Author"]; ok && i < len(row) {
			book.AuthorNames = row[i]
		}
		if i, ok := cols["ISBN13"]; ok && i < len(row) {
			book.ISBN13 = strings.Trim(row[i], "=\"")
		}
		if i, ok := cols["ISBN"]; ok && i < len(row) {
			book.ISBN10 = strings.Trim(row[i], "=\"")
		}
		if i, ok := cols["Number of Pages"]; ok && i < len(row) {
			book.PageCount, _ = strconv.Atoi(row[i])
		}
		if i, ok := cols["Publisher"]; ok && i < len(row) {
			book.Publisher = row[i]
		}
		if i, ok := cols["Year Published"]; ok && i < len(row) {
			book.PublishYear, _ = strconv.Atoi(row[i])
		}
		if i, ok := cols["Average Rating"]; ok && i < len(row) {
			book.AverageRating, _ = strconv.ParseFloat(row[i], 64)
		}
		if i, ok := cols["Book Id"]; ok && i < len(row) {
			book.GoodreadsID = strings.TrimSpace(row[i])
		}

		if book.Title == "" {
			continue
		}

		if err := h.st.Book().Create(c.Context(), book); err != nil {
			continue
		}

		if i, ok := cols["Exclusive Shelf"]; ok && i < len(row) {
			if shelf, _ := h.st.Shelf().GetBySlug(c.Context(), row[i]); shelf != nil {
				h.st.Shelf().AddBook(c.Context(), shelf.ID, book.ID)
			}
		}

		if i, ok := cols["My Rating"]; ok && i < len(row) {
			if rating, _ := strconv.Atoi(row[i]); rating > 0 {
				review := &types.Review{BookID: book.ID, Rating: rating}
				if i, ok := cols["My Review"]; ok && i < len(row) {
					review.Text = row[i]
				}
				if i, ok := cols["Date Read"]; ok && i < len(row) && row[i] != "" {
					if t, err := time.Parse("2006/01/02", row[i]); err == nil {
						review.FinishedAt = &t
					}
				}
				h.st.Review().Create(c.Context(), review)
			}
		}

		imported++
	}

	return c.JSON(200, map[string]any{"imported": imported})
}

func (h *ImportExportHandler) ExportCSV(c *mizu.Ctx) error {
	c.Header().Set("Content-Type", "text/csv")
	c.Header().Set("Content-Disposition", "attachment; filename=book_export.csv")

	w := csv.NewWriter(c.Writer())
	defer w.Flush()

	w.Write([]string{
		"Title", "Author", "ISBN", "ISBN13", "My Rating", "Average Rating",
		"Publisher", "Number of Pages", "Year Published", "Date Read",
		"Exclusive Shelf", "My Review",
	})

	result, err := h.st.Book().Search(c.Context(), "", 1, 10000)
	if err != nil {
		return err
	}

	for _, book := range result.Books {
		rating := ""
		shelf := ""
		reviewText := ""
		dateRead := ""

		if review, _ := h.st.Review().GetUserReview(c.Context(), book.ID); review != nil {
			if review.Rating > 0 {
				rating = strconv.Itoa(review.Rating)
			}
			reviewText = review.Text
			if review.FinishedAt != nil {
				dateRead = review.FinishedAt.Format("2006/01/02")
			}
		}

		if shelves, _ := h.st.Shelf().GetBookShelves(c.Context(), book.ID); len(shelves) > 0 {
			for _, sh := range shelves {
				if sh.IsExclusive {
					shelf = sh.Slug
					break
				}
			}
		}

		w.Write([]string{
			book.Title, book.AuthorNames, book.ISBN10, book.ISBN13,
			rating, fmt.Sprintf("%.2f", book.AverageRating),
			book.Publisher, strconv.Itoa(book.PageCount),
			strconv.Itoa(book.PublishYear), dateRead, shelf, reviewText,
		})
	}

	return nil
}
