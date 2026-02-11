package openlibrary

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

const (
	baseURL   = "https://openlibrary.org"
	coversURL = "https://covers.openlibrary.org"
)

// Client is an Open Library API client
type Client struct {
	http *http.Client
}

// NewClient creates a new Open Library client
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// Search searches Open Library for books
func (c *Client) Search(ctx context.Context, query string, limit int) ([]types.Book, error) {
	if limit <= 0 {
		limit = 10
	}

	u := fmt.Sprintf("%s/search.json?q=%s&limit=%d&fields=key,title,author_name,author_key,first_publish_year,number_of_pages_median,isbn,cover_i,subject,publisher,language,ratings_average,ratings_count",
		baseURL, url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BookManager/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search API returned %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var books []types.Book
	for _, doc := range result.Docs {
		book := docToBook(doc)
		books = append(books, book)
	}
	return books, nil
}

// GetWork fetches a work by its Open Library key
func (c *Client) GetWork(ctx context.Context, olKey string) (*types.Book, error) {
	u := fmt.Sprintf("%s%s.json", baseURL, olKey)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BookManager/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("works API returned %d", resp.StatusCode)
	}

	var work WorkResponse
	if err := json.NewDecoder(resp.Body).Decode(&work); err != nil {
		return nil, err
	}

	book := &types.Book{
		OLKey:    work.Key,
		Title:    work.Title,
		Subjects: work.Subjects,
	}

	// Handle description which can be string or object
	switch d := work.Description.(type) {
	case string:
		book.Description = d
	case map[string]interface{}:
		if v, ok := d["value"].(string); ok {
			book.Description = v
		}
	}

	// Set cover
	if len(work.Covers) > 0 && work.Covers[0] > 0 {
		book.CoverID = work.Covers[0]
		book.CoverURL = CoverURL(work.Covers[0], "L")
	}

	return book, nil
}

// GetAuthor fetches an author by their Open Library key
func (c *Client) GetAuthor(ctx context.Context, olKey string) (*types.Author, error) {
	u := fmt.Sprintf("%s%s.json", baseURL, olKey)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BookManager/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("authors API returned %d", resp.StatusCode)
	}

	var ar AuthorResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}

	author := &types.Author{
		OLKey:     ar.Key,
		Name:      ar.Name,
		BirthDate: ar.BirthDate,
		DeathDate: ar.DeathDate,
	}

	switch b := ar.Bio.(type) {
	case string:
		author.Bio = b
	case map[string]interface{}:
		if v, ok := b["value"].(string); ok {
			author.Bio = v
		}
	}

	if len(ar.Photos) > 0 && ar.Photos[0] > 0 {
		author.PhotoURL = fmt.Sprintf("%s/a/id/%d-M.jpg", coversURL, ar.Photos[0])
	}

	return author, nil
}

// CoverURL returns a cover image URL
func CoverURL(coverID int, size string) string {
	if coverID <= 0 {
		return ""
	}
	return fmt.Sprintf("%s/b/id/%d-%s.jpg", coversURL, coverID, size)
}

func docToBook(doc SearchDoc) types.Book {
	authorNames := strings.Join(doc.AuthorName, ", ")

	var isbn13, isbn10 string
	for _, isbn := range doc.ISBN {
		clean := strings.ReplaceAll(isbn, "-", "")
		if len(clean) == 13 && isbn13 == "" {
			isbn13 = clean
		} else if len(clean) == 10 && isbn10 == "" {
			isbn10 = clean
		}
		if isbn13 != "" && isbn10 != "" {
			break
		}
	}

	coverURL := ""
	if doc.CoverI > 0 {
		coverURL = CoverURL(doc.CoverI, "M")
	}

	publisher := ""
	if len(doc.Publisher) > 0 {
		publisher = doc.Publisher[0]
	}

	lang := "en"
	if len(doc.Language) > 0 {
		lang = doc.Language[0]
	}

	// Limit subjects to top 10
	subjects := doc.Subject
	if len(subjects) > 10 {
		subjects = subjects[:10]
	}

	return types.Book{
		OLKey:         doc.Key,
		Title:         doc.Title,
		AuthorNames:   authorNames,
		CoverURL:      coverURL,
		CoverID:       doc.CoverI,
		ISBN10:        isbn10,
		ISBN13:        isbn13,
		Publisher:     publisher,
		PublishYear:   doc.FirstPublishYear,
		PageCount:     doc.NumberOfPages,
		Language:      lang,
		Subjects:      subjects,
		AverageRating: doc.RatingsAverage,
		RatingsCount:  doc.RatingsCount,
	}
}
