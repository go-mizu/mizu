package googlebooks

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

const apiBase = "https://www.googleapis.com/books/v1"

// Client is a Google Books API client
type Client struct {
	http   *http.Client
	apiKey string
}

// NewClient creates a new Google Books client
func NewClient(apiKey string) *Client {
	return &Client{
		http:   &http.Client{Timeout: 15 * time.Second},
		apiKey: apiKey,
	}
}

// Search searches Google Books
func (c *Client) Search(ctx context.Context, query string, limit int) ([]types.Book, error) {
	if limit <= 0 {
		limit = 10
	}

	u := fmt.Sprintf("%s/volumes?q=%s&maxResults=%d", apiBase, url.QueryEscape(query), limit)
	if c.apiKey != "" {
		u += "&key=" + c.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Google Books API returned %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var books []types.Book
	for _, item := range result.Items {
		books = append(books, volumeToBook(item))
	}
	return books, nil
}

// GetVolume fetches a single volume by ID
func (c *Client) GetVolume(ctx context.Context, id string) (*types.Book, error) {
	u := fmt.Sprintf("%s/volumes/%s", apiBase, id)
	if c.apiKey != "" {
		u += "?key=" + c.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Google Books API returned %d", resp.StatusCode)
	}

	var vol Volume
	if err := json.NewDecoder(resp.Body).Decode(&vol); err != nil {
		return nil, err
	}

	book := volumeToBook(vol)
	return &book, nil
}

func volumeToBook(vol Volume) types.Book {
	vi := vol.VolumeInfo

	var isbn10, isbn13 string
	for _, id := range vi.IndustryIdentifiers {
		switch id.Type {
		case "ISBN_10":
			isbn10 = id.Identifier
		case "ISBN_13":
			isbn13 = id.Identifier
		}
	}

	coverURL := ""
	if vi.ImageLinks != nil {
		coverURL = vi.ImageLinks.Thumbnail
		// Use https
		coverURL = strings.Replace(coverURL, "http://", "https://", 1)
	}

	year := 0
	if len(vi.PublishedDate) >= 4 {
		fmt.Sscanf(vi.PublishedDate[:4], "%d", &year)
	}

	return types.Book{
		GoogleID:      vol.ID,
		Title:         vi.Title,
		Subtitle:      vi.Subtitle,
		Description:   vi.Description,
		AuthorNames:   strings.Join(vi.Authors, ", "),
		CoverURL:      coverURL,
		ISBN10:        isbn10,
		ISBN13:        isbn13,
		Publisher:     vi.Publisher,
		PublishDate:   vi.PublishedDate,
		PublishYear:   year,
		PageCount:     vi.PageCount,
		Language:      vi.Language,
		Subjects:      vi.Categories,
		AverageRating: vi.AverageRating,
		RatingsCount:  vi.RatingsCount,
	}
}
