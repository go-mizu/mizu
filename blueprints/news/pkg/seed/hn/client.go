package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/pkg/seed"
)

const (
	baseURL        = "https://hacker-news.firebaseio.com/v0"
	defaultUA      = "NewsSeeder/1.0"
	defaultWorkers = 10
)

// Client is a Hacker News API client.
type Client struct {
	httpClient  *http.Client
	userAgent   string
	rateLimit   time.Duration
	lastReq     time.Time
	mu          sync.Mutex
	concurrency int
}

// NewClient creates a new Hacker News client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent:   defaultUA,
		rateLimit:   100 * time.Millisecond,
		concurrency: defaultWorkers,
	}
}

// WithUserAgent sets the user agent.
func (c *Client) WithUserAgent(ua string) *Client {
	c.userAgent = ua
	return c
}

// WithRateLimit sets the rate limit between requests.
func (c *Client) WithRateLimit(d time.Duration) *Client {
	c.rateLimit = d
	return c
}

// WithConcurrency sets the number of concurrent workers for batch fetches.
func (c *Client) WithConcurrency(n int) *Client {
	if n > 0 {
		c.concurrency = n
	}
	return c
}

// Name returns the source name.
func (c *Client) Name() string {
	return "hn"
}

// FetchFeed fetches story IDs from a specific feed.
func (c *Client) FetchFeed(ctx context.Context, feed FeedType) ([]int, error) {
	url := fmt.Sprintf("%s/%s.json", baseURL, feed.Endpoint())

	var ids []int
	if err := c.get(ctx, url, &ids); err != nil {
		return nil, err
	}

	return ids, nil
}

// FetchItem fetches a single item by ID.
func (c *Client) FetchItem(ctx context.Context, id int) (*Item, error) {
	url := fmt.Sprintf("%s/item/%d.json", baseURL, id)

	var item Item
	if err := c.get(ctx, url, &item); err != nil {
		return nil, err
	}

	// Check for null response (deleted item)
	if item.ID == 0 {
		return nil, nil
	}

	return &item, nil
}

// FetchItems fetches multiple items concurrently.
func (c *Client) FetchItems(ctx context.Context, ids []int) ([]*Item, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	type result struct {
		index int
		item  *Item
		err   error
	}

	results := make(chan result, len(ids))
	sem := make(chan struct{}, c.concurrency)

	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(idx, itemID int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			item, err := c.FetchItem(ctx, itemID)
			results <- result{index: idx, item: item, err: err}
		}(i, id)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	items := make([]*Item, len(ids))
	var firstErr error
	for r := range results {
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		if r.item != nil {
			items[r.index] = r.item
		}
	}

	// Filter nil items
	filtered := make([]*Item, 0, len(items))
	for _, item := range items {
		if item != nil {
			filtered = append(filtered, item)
		}
	}

	return filtered, firstErr
}

// FetchStories implements seed.Source - fetches stories from HN.
func (c *Client) FetchStories(ctx context.Context, opts seed.FetchOpts) ([]*seed.StoryData, error) {
	// Determine feed type from SortBy
	feed := FeedTop
	switch opts.SortBy {
	case "new":
		feed = FeedNew
	case "best":
		feed = FeedBest
	case "ask":
		feed = FeedAsk
	case "show":
		feed = FeedShow
	case "jobs":
		feed = FeedJobs
	}

	// Fetch story IDs
	ids, err := c.FetchFeed(ctx, feed)
	if err != nil {
		return nil, err
	}

	// Apply limit
	limit := opts.Limit
	if limit <= 0 {
		limit = 30
	}
	if limit > len(ids) {
		limit = len(ids)
	}
	ids = ids[:limit]

	// Fetch items concurrently
	items, err := c.FetchItems(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Convert to StoryData
	stories := make([]*seed.StoryData, 0, len(items))
	for _, item := range items {
		if item == nil || item.IsDeleted() || !item.IsStory() {
			continue
		}

		story := c.itemToStory(item)
		stories = append(stories, story)
	}

	return stories, nil
}

// FetchComments implements seed.Source - fetches comments for a story.
func (c *Client) FetchComments(ctx context.Context, storyID string) ([]*seed.CommentData, error) {
	id, err := strconv.Atoi(storyID)
	if err != nil {
		return nil, fmt.Errorf("invalid story ID: %s", storyID)
	}

	// Fetch the story to get comment IDs
	story, err := c.FetchItem(ctx, id)
	if err != nil {
		return nil, err
	}
	if story == nil {
		return nil, nil
	}

	if len(story.Kids) == 0 {
		return nil, nil
	}

	// Fetch and build comment tree
	return c.fetchCommentTree(ctx, story.Kids, storyID, "", 0)
}

// fetchCommentTree recursively fetches comments and their replies.
func (c *Client) fetchCommentTree(ctx context.Context, ids []int, storyID, parentID string, depth int) ([]*seed.CommentData, error) {
	if len(ids) == 0 || depth > 10 { // Limit depth to prevent deep recursion
		return nil, nil
	}

	items, err := c.FetchItems(ctx, ids)
	if err != nil {
		return nil, err
	}

	comments := make([]*seed.CommentData, 0, len(items))
	for _, item := range items {
		if item == nil || item.IsDeleted() || !item.IsComment() {
			continue
		}

		comment := c.itemToComment(item, storyID, parentID, depth)

		// Recursively fetch replies
		if len(item.Kids) > 0 {
			replies, err := c.fetchCommentTree(ctx, item.Kids, storyID, strconv.Itoa(item.ID), depth+1)
			if err == nil {
				comment.Replies = replies
			}
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

// itemToStory converts an HN Item to seed.StoryData.
func (c *Client) itemToStory(item *Item) *seed.StoryData {
	// Determine if it's a text post (Ask HN, etc.) or a link
	isSelf := item.URL == ""
	content := c.htmlToText(item.Text)

	return &seed.StoryData{
		ExternalID:   strconv.Itoa(item.ID),
		Title:        item.Title,
		Content:      content,
		URL:          item.URL,
		Author:       item.By,
		Score:        int64(item.Score),
		CommentCount: int64(item.Descendants),
		CreatedAt:    item.CreatedTime(),
		IsSelf:       isSelf,
		Domain:       c.extractDomain(item.URL),
	}
}

// itemToComment converts an HN Item to seed.CommentData.
func (c *Client) itemToComment(item *Item, storyID, parentID string, depth int) *seed.CommentData {
	return &seed.CommentData{
		ExternalID:       strconv.Itoa(item.ID),
		ExternalParentID: parentID,
		ExternalStoryID:  storyID,
		Author:           item.By,
		Content:          c.htmlToText(item.Text),
		Score:            int64(item.Score),
		CreatedAt:        item.CreatedTime(),
		Depth:            depth,
	}
}

// htmlToText converts HN HTML content to plain text.
func (c *Client) htmlToText(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	// Decode HTML entities
	text := html.UnescapeString(htmlContent)

	// Replace <p> tags with newlines
	text = strings.ReplaceAll(text, "<p>", "\n\n")
	text = strings.ReplaceAll(text, "</p>", "")

	// Replace <br> tags with newlines
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")

	// Convert links to markdown format
	linkRegex := regexp.MustCompile(`<a[^>]+href="([^"]+)"[^>]*>([^<]*)</a>`)
	text = linkRegex.ReplaceAllString(text, "[$2]($1)")

	// Remove remaining HTML tags
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	text = tagRegex.ReplaceAllString(text, "")

	// Clean up whitespace
	text = strings.TrimSpace(text)

	return text
}

// extractDomain extracts the domain from a URL.
func (c *Client) extractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	// Simple domain extraction
	url := strings.TrimPrefix(rawURL, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")

	if idx := strings.Index(url, "/"); idx > 0 {
		url = url[:idx]
	}

	return url
}

// get performs an HTTP GET request with rate limiting.
func (c *Client) get(ctx context.Context, url string, v any) error {
	// Rate limiting
	c.mu.Lock()
	if !c.lastReq.IsZero() {
		elapsed := time.Since(c.lastReq)
		if elapsed < c.rateLimit {
			time.Sleep(c.rateLimit - elapsed)
		}
	}
	c.lastReq = time.Now()
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

// Ensure Client implements seed.Source
var _ seed.Source = (*Client)(nil)
