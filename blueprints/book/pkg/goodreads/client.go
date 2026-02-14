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
	http    *http.Client
	baseURL string
}

// NewClient creates a new Goodreads scraper client.
func NewClient() *Client {
	return &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
	}
}

// GetBook fetches and parses a Goodreads book page by its numeric ID.
func (c *Client) GetBook(ctx context.Context, goodreadsID string) (*GoodreadsBook, error) {
	goodreadsID = strings.TrimSpace(goodreadsID)
	if goodreadsID == "" {
		return nil, fmt.Errorf("empty goodreads ID")
	}

	u := fmt.Sprintf("%s/book/show/%s", c.baseURL, goodreadsID)
	body, err := c.fetch(ctx, u)
	if err != nil {
		return nil, err
	}

	book, err := parseBookPage(body)
	if err != nil {
		return nil, fmt.Errorf("parse page: %w", err)
	}
	book.GoodreadsID = goodreadsID
	if book.URL == "" {
		book.URL = u
	}

	// Fetch quotes from the work quotes page
	if book.WorkID != "" {
		if quotes, err := c.GetQuotes(ctx, book.WorkID); err == nil && len(quotes) > 0 {
			book.Quotes = quotes
		}
	}

	return book, nil
}

// GetAuthor fetches and parses a Goodreads author page.
func (c *Client) GetAuthor(ctx context.Context, goodreadsID string) (*GoodreadsAuthor, error) {
	goodreadsID = strings.TrimSpace(goodreadsID)
	if goodreadsID == "" {
		return nil, fmt.Errorf("empty goodreads author ID")
	}

	u := fmt.Sprintf("%s/author/show/%s", c.baseURL, goodreadsID)
	body, err := c.fetch(ctx, u)
	if err != nil {
		return nil, err
	}

	author := parseAuthorPage(body)
	author.GoodreadsID = goodreadsID
	author.URL = u
	return author, nil
}

// GetList fetches and parses a Goodreads list page.
func (c *Client) GetList(ctx context.Context, urlOrID string) (*GoodreadsList, error) {
	u := urlOrID
	if !strings.Contains(u, "/") {
		u = fmt.Sprintf("%s/list/show/%s", c.baseURL, u)
	}

	body, err := c.fetch(ctx, u)
	if err != nil {
		return nil, err
	}

	return parseListPage(body), nil
}

// GetQuotes fetches quotes for a book by its Goodreads work ID.
func (c *Client) GetQuotes(ctx context.Context, workID string) ([]GoodreadsQuote, error) {
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return nil, fmt.Errorf("empty work ID")
	}
	u := fmt.Sprintf("%s/work/quotes/%s", c.baseURL, workID)
	body, err := c.fetch(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseQuotesPage(body), nil
}

// SearchBook searches Goodreads by title and returns the first matching book ID.
func (c *Client) SearchBook(ctx context.Context, title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("empty title")
	}
	u := fmt.Sprintf("%s/search?q=%s", c.baseURL, strings.ReplaceAll(title, " ", "+"))
	body, err := c.fetch(ctx, u)
	if err != nil {
		return "", err
	}
	// Extract first book ID from search results
	if m := reSearchBookID.FindStringSubmatch(body); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("no results found")
}

// GetPopularLists fetches Goodreads lists from a stable server-rendered source.
// Optional tag narrows discovery to `/list/tag/{tag}`.
func (c *Client) GetPopularLists(ctx context.Context, tag string) ([]GoodreadsListSummary, error) {
	tag = strings.TrimSpace(strings.ToLower(tag))
	urls := make([]string, 0, 3)
	if tag != "" {
		safeTag := strings.ReplaceAll(tag, " ", "-")
		urls = append(urls, fmt.Sprintf("%s/list/tag/%s", c.baseURL, safeTag))
	}
	// `/list` is a Next.js shell in many contexts; keep it as best-effort fallback.
	urls = append(urls,
		c.baseURL+"/list/popular_lists",
		c.baseURL+"/list?ref=nav_brws_lists",
	)

	var lastErr error
	for _, u := range urls {
		body, err := c.fetch(ctx, u)
		if err != nil {
			lastErr = err
			continue
		}
		lists := parseListsBrowse(body)
		if len(lists) > 0 {
			if tag != "" {
				for i := range lists {
					lists[i].Tag = tag
				}
			}
			return lists, nil
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return []GoodreadsListSummary{}, nil
}

// fetch retrieves a Goodreads page and returns the body as a string.
func (c *Client) fetch(ctx context.Context, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("goodreads returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(body), nil
}

// ParseGoodreadsURL extracts the numeric book ID from a Goodreads URL.
func ParseGoodreadsURL(input string) string {
	input = strings.TrimSpace(input)

	if !strings.Contains(input, "/") {
		return strings.Split(input, ".")[0]
	}

	if idx := strings.Index(input, "/book/show/"); idx >= 0 {
		path := input[idx+len("/book/show/"):]
		path = strings.Split(path, "?")[0]
		path = strings.Split(path, "#")[0]
		return strings.Split(path, ".")[0]
	}

	return input
}

// ParseGoodreadsAuthorURL extracts the numeric author ID from a Goodreads URL.
func ParseGoodreadsAuthorURL(input string) string {
	input = strings.TrimSpace(input)

	if !strings.Contains(input, "/") {
		return strings.Split(input, ".")[0]
	}

	if idx := strings.Index(input, "/author/show/"); idx >= 0 {
		path := input[idx+len("/author/show/"):]
		path = strings.Split(path, "?")[0]
		path = strings.Split(path, "#")[0]
		return strings.Split(path, ".")[0]
	}

	return input
}
