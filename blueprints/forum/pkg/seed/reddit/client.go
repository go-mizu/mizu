package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed"
)

const (
	baseURL     = "https://www.reddit.com"
	defaultUA   = "ForumSeeder/1.0"
	rateLimitMS = 2000 // 2 seconds between requests
)

// Client is a Reddit API client.
type Client struct {
	httpClient *http.Client
	userAgent  string
	rateLimit  time.Duration
	lastReq    time.Time
	mu         sync.Mutex
}

// NewClient creates a new Reddit client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: defaultUA,
		rateLimit: rateLimitMS * time.Millisecond,
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

// Name returns the source name.
func (c *Client) Name() string {
	return "reddit"
}

// FetchSubreddit fetches metadata for a subreddit.
func (c *Client) FetchSubreddit(ctx context.Context, name string) (*seed.SubredditData, error) {
	url := fmt.Sprintf("%s/r/%s/about.json", baseURL, name)
	var about SubredditAbout
	if err := c.get(ctx, url, &about); err != nil {
		return nil, err
	}

	return &seed.SubredditData{
		Name:        about.Data.DisplayName,
		Title:       about.Data.DisplayName,
		Description: about.Data.PublicDescription,
		Subscribers: about.Data.Subscribers,
	}, nil
}

// FetchThreads fetches threads from a subreddit.
func (c *Client) FetchThreads(ctx context.Context, subreddit string, opts seed.FetchOpts) ([]*seed.ThreadData, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	url := fmt.Sprintf("%s/r/%s.json?limit=%d", baseURL, subreddit, limit)
	if opts.After != "" {
		url += "&after=" + opts.After
	}

	var listing Listing
	if err := c.get(ctx, url, &listing); err != nil {
		return nil, err
	}

	var threads []*seed.ThreadData
	for _, child := range listing.Data.Children {
		if child.Kind != "t3" { // t3 = link/post
			continue
		}
		d := child.Data

		// Skip removed/deleted posts
		if d.Author == "[deleted]" || d.Author == "[removed]" {
			continue
		}

		threads = append(threads, &seed.ThreadData{
			ExternalID:    d.ID,
			Title:         d.Title,
			Content:       d.Selftext,
			URL:           d.URL,
			Author:        d.Author,
			Score:         d.Score,
			UpvoteCount:   d.Ups,
			DownvoteCount: d.Downs,
			CommentCount:  d.NumComments,
			CreatedAt:     d.CreatedTime(),
			IsNSFW:        d.Over18,
			IsSpoiler:     d.Spoiler,
			IsSelf:        d.IsSelf,
			Domain:        d.Domain,
			Permalink:     d.Permalink,
		})
	}

	return threads, nil
}

// FetchComments fetches comments for a thread.
func (c *Client) FetchComments(ctx context.Context, subreddit, threadID string) ([]*seed.CommentData, error) {
	url := fmt.Sprintf("%s/r/%s/comments/%s.json?limit=200&depth=10", baseURL, subreddit, threadID)

	var listings []Listing
	if err := c.get(ctx, url, &listings); err != nil {
		return nil, err
	}

	// The response is [post, comments]
	if len(listings) < 2 {
		return nil, nil
	}

	return c.parseComments(listings[1].Data.Children), nil
}

func (c *Client) parseComments(children []Thing) []*seed.CommentData {
	var comments []*seed.CommentData

	for _, child := range children {
		if child.Kind != "t1" { // t1 = comment
			continue
		}
		d := child.Data

		// Skip deleted/removed
		if d.Author == "[deleted]" || d.Author == "[removed]" || d.Body == "[deleted]" || d.Body == "[removed]" {
			continue
		}

		comment := &seed.CommentData{
			ExternalID:       d.ID,
			ExternalParentID: c.extractParentID(d.ParentID),
			ExternalThreadID: c.extractThreadID(d.LinkID),
			Author:           d.Author,
			Content:          d.Body,
			Score:            d.Score,
			UpvoteCount:      d.Ups,
			DownvoteCount:    d.Downs,
			CreatedAt:        d.CreatedTime(),
			Depth:            d.Depth,
		}

		// Parse nested replies
		comment.Replies = c.parseRepliesFromAny(d.Replies)

		comments = append(comments, comment)
	}

	return comments
}

func (c *Client) parseRepliesFromAny(replies any) []*seed.CommentData {
	if replies == nil {
		return nil
	}

	// Empty string means no replies
	if _, ok := replies.(string); ok {
		return nil
	}

	// It's a map that we need to parse as a Listing
	replyMap, ok := replies.(map[string]any)
	if !ok {
		return nil
	}

	data, ok := replyMap["data"].(map[string]any)
	if !ok {
		return nil
	}

	children, ok := data["children"].([]any)
	if !ok {
		return nil
	}

	var things []Thing
	for _, child := range children {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}

		kind, _ := childMap["kind"].(string)
		if kind != "t1" {
			continue
		}

		dataMap, ok := childMap["data"].(map[string]any)
		if !ok {
			continue
		}

		thing := Thing{Kind: kind}
		thing.Data = c.parseThingData(dataMap)
		things = append(things, thing)
	}

	return c.parseComments(things)
}

func (c *Client) parseThingData(m map[string]any) ThingData {
	d := ThingData{}

	if v, ok := m["id"].(string); ok {
		d.ID = v
	}
	if v, ok := m["name"].(string); ok {
		d.Name = v
	}
	if v, ok := m["author"].(string); ok {
		d.Author = v
	}
	if v, ok := m["score"].(float64); ok {
		d.Score = int64(v)
	}
	if v, ok := m["ups"].(float64); ok {
		d.Ups = int64(v)
	}
	if v, ok := m["downs"].(float64); ok {
		d.Downs = int64(v)
	}
	if v, ok := m["created_utc"].(float64); ok {
		d.CreatedUTC = v
	}
	if v, ok := m["body"].(string); ok {
		d.Body = v
	}
	if v, ok := m["parent_id"].(string); ok {
		d.ParentID = v
	}
	if v, ok := m["link_id"].(string); ok {
		d.LinkID = v
	}
	if v, ok := m["depth"].(float64); ok {
		d.Depth = int(v)
	}
	if v, ok := m["replies"]; ok {
		d.Replies = v
	}

	return d
}

func (c *Client) extractParentID(fullID string) string {
	// Parent ID is like "t3_abc123" (thread) or "t1_xyz789" (comment)
	// We want to return empty for thread parents, and the ID for comment parents
	if strings.HasPrefix(fullID, "t3_") {
		return "" // Top-level comment
	}
	return strings.TrimPrefix(fullID, "t1_")
}

func (c *Client) extractThreadID(fullID string) string {
	return strings.TrimPrefix(fullID, "t3_")
}

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

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

// Ensure Client implements seed.Source
var _ seed.Source = (*Client)(nil)
