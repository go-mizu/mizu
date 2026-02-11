package goodreads

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://www.goodreads.com"

// Client scrapes book data from Goodreads HTML pages.
type Client struct {
	http *http.Client
}

// NewClient creates a new Goodreads scraper client.
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetBook fetches and parses a Goodreads book page by its numeric ID.
func (c *Client) GetBook(ctx context.Context, goodreadsID string) (*GoodreadsBook, error) {
	goodreadsID = strings.TrimSpace(goodreadsID)
	if goodreadsID == "" {
		return nil, fmt.Errorf("empty goodreads ID")
	}

	u := fmt.Sprintf("%s/book/show/%s", baseURL, goodreadsID)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch goodreads page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("goodreads returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	book, err := parseBookPage(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse page: %w", err)
	}
	book.GoodreadsID = goodreadsID
	return book, nil
}

// ParseGoodreadsURL extracts the numeric book ID from a Goodreads URL.
// Accepts formats like:
//   - https://www.goodreads.com/book/show/112247
//   - https://www.goodreads.com/book/show/112247.The_Art_of_Computer_Programming_Volume_1
//   - 112247
func ParseGoodreadsURL(input string) string {
	input = strings.TrimSpace(input)

	// Direct ID
	if !strings.Contains(input, "/") {
		return strings.Split(input, ".")[0]
	}

	// URL format: /book/show/112247 or /book/show/112247.Title
	if idx := strings.Index(input, "/book/show/"); idx >= 0 {
		path := input[idx+len("/book/show/"):]
		path = strings.Split(path, "?")[0]
		path = strings.Split(path, "#")[0]
		return strings.Split(path, ".")[0]
	}

	return input
}
