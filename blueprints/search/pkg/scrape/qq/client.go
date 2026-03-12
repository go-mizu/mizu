package qq

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrArticleDeleted indicates the article has been removed (302 → babygohome).
var ErrArticleDeleted = errors.New("article deleted")

// errNoRedirect is returned by CheckRedirect to stop following redirects.
var errNoRedirect = errors.New("no redirect")

// Client handles HTTP requests to news.qq.com APIs.
type Client struct {
	http        *http.Client
	articleHTTP *http.Client // no-redirect client for article fetches
	userAgent   string
}

// NewClient creates a new QQ News API client.
func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxConnsPerHost:     cfg.Workers + 5,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// Article client does NOT follow redirects — we detect 302 → babygohome as "deleted"
	articleHTTP := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errNoRedirect
		},
	}

	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		articleHTTP: articleHTTP,
		userAgent:   cfg.UserAgent,
	}
}

// FetchSitemapIndex fetches and parses the sitemap index XML.
func (c *Client) FetchSitemapIndex(ctx context.Context) (*SitemapIndex, error) {
	body, err := c.get(ctx, SitemapIndexURL)
	if err != nil {
		return nil, fmt.Errorf("fetch sitemap index: %w", err)
	}

	var idx SitemapIndex
	if err := xml.Unmarshal(body, &idx); err != nil {
		return nil, fmt.Errorf("parse sitemap index: %w", err)
	}

	return &idx, nil
}

// FetchSitemap fetches and parses an individual sitemap XML.
func (c *Client) FetchSitemap(ctx context.Context, url string) (*URLSet, error) {
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch sitemap %s: %w", url, err)
	}

	var urlSet URLSet
	if err := xml.Unmarshal(body, &urlSet); err != nil {
		return nil, fmt.Errorf("parse sitemap %s: %w", url, err)
	}

	return &urlSet, nil
}

// FetchArticlePage fetches the full HTML of an article page.
// Returns ErrArticleDeleted if the article redirects to babygohome.
func (c *Client) FetchArticlePage(ctx context.Context, articleID string) (string, int, error) {
	url := ArticleBaseURL + articleID
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := c.articleHTTP.Do(req)
	if err != nil {
		// Check if this was a redirect we blocked
		if resp != nil {
			resp.Body.Close()
			if resp.StatusCode >= 300 && resp.StatusCode < 400 {
				loc := resp.Header.Get("Location")
				if strings.Contains(loc, "babygohome") || strings.Contains(loc, "qq.com/baby") {
					return "", resp.StatusCode, ErrArticleDeleted
				}
				return "", resp.StatusCode, fmt.Errorf("redirect to %s", loc)
			}
		}
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}

	return string(body), resp.StatusCode, nil
}

// FetchHotRanking fetches the hot ranking list.
func (c *Client) FetchHotRanking(ctx context.Context) ([]HotNewsItem, error) {
	body, err := c.get(ctx, HotRankingURL+"?page_size=50")
	if err != nil {
		return nil, fmt.Errorf("fetch hot ranking: %w", err)
	}

	var resp HotRankingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse hot ranking: %w", err)
	}

	var items []HotNewsItem
	for _, list := range resp.IDList {
		// Skip first item (metadata header)
		for i, item := range list.NewsList {
			if i == 0 {
				continue
			}
			if item.ID != "" {
				items = append(items, item)
			}
		}
	}

	return items, nil
}

// FetchChannelFeed fetches articles from a channel feed.
func (c *Client) FetchChannelFeed(ctx context.Context, channelID string) ([]FeedNewsItem, error) {
	reqBody := FeedRequest{
		Forward:   "2",
		BaseReq:   FeedBaseReq{From: "pc"},
		ChannelID: channelID,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", FeedAPIURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch channel %s: %w", channelID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feedResp FeedResponse
	if err := json.Unmarshal(body, &feedResp); err != nil {
		return nil, fmt.Errorf("parse channel %s: %w", channelID, err)
	}

	var items []FeedNewsItem
	for _, dataItem := range feedResp.Data {
		// Top-level items
		if dataItem.ID != "" {
			items = append(items, FeedNewsItem{
				ID:          dataItem.ID,
				Title:       dataItem.Title,
				ArticleType: dataItem.ArticleType,
			})
		}
		// Nested sub_items (grouped articles)
		for _, sub := range dataItem.SubItems {
			if sub.ID != "" && sub.ID != dataItem.ID {
				items = append(items, FeedNewsItem{
					ID:          sub.ID,
					Title:       sub.Title,
					ArticleType: sub.ArticleType,
				})
			}
		}
	}

	return items, nil
}

// ProbeSitemapStatus checks if a sitemap URL exists via HEAD request.
// Returns the HTTP status code (200=exists, 404=not found, 501=WAF).
func (c *Client) ProbeSitemapStatus(ctx context.Context, url string) int {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0
	}
	resp.Body.Close()

	return resp.StatusCode
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
