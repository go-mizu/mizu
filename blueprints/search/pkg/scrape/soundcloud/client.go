package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var clientIDRe = regexp.MustCompile(`"hydratable":"apiClient","data":{"id":"([A-Za-z0-9]{32})"`)

type Client struct {
	http     *http.Client
	delay    time.Duration
	pageSize int
	mu       sync.Mutex
	lastReq  time.Time
	clientID string
}

func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     cfg.Workers + 2,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		delay:    cfg.Delay,
		pageSize: cfg.PageSize,
	}
}

func (c *Client) Fetch(ctx context.Context, rawURL string) ([]byte, int, error) {
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c.rateLimit()
		body, code, err := c.doGet(ctx, rawURL, "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		if err == nil && code < 500 && code != 429 {
			return body, code, nil
		}
		if attempt == maxAttempts {
			if err != nil {
				return nil, code, err
			}
			return body, code, fmt.Errorf("unexpected HTTP %d for %s", code, rawURL)
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return nil, 0, fmt.Errorf("unreachable")
}

func (c *Client) FetchHTML(ctx context.Context, rawURL string) (*goquery.Document, []byte, int, error) {
	body, code, err := c.Fetch(ctx, rawURL)
	if err != nil {
		return nil, nil, code, err
	}
	if code == 404 {
		return nil, nil, code, nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, body, code, fmt.Errorf("parse HTML: %w", err)
	}
	return doc, body, code, nil
}

func (c *Client) Search(ctx context.Context, query, kind string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = c.pageSize
	}
	clientID, err := c.ensureClientID(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := APIBaseURL + "/search"
	switch strings.ToLower(kind) {
	case "tracks", EntityTrack:
		endpoint += "/tracks"
	case "playlists", "sets", EntityPlaylist:
		endpoint += "/playlists"
	case "users", "people", EntityUser:
		endpoint += "/users"
	}

	var results []SearchResult
	offset := 0
	pageSize := min(limit, c.pageSize)
	for len(results) < limit {
		u, _ := url.Parse(endpoint)
		q := u.Query()
		q.Set("q", query)
		q.Set("client_id", clientID)
		q.Set("limit", fmt.Sprintf("%d", pageSize))
		q.Set("offset", fmt.Sprintf("%d", offset))
		q.Set("linked_partitioning", "1")
		u.RawQuery = q.Encode()

		var resp searchResponse
		if err := c.fetchJSON(ctx, u.String(), &resp); err != nil {
			return results, err
		}

		for _, raw := range resp.Collection {
			r, ok := parseSearchResult(query, raw)
			if !ok {
				continue
			}
			results = append(results, r)
			if len(results) >= limit {
				break
			}
		}
		if len(resp.Collection) == 0 || resp.NextHref == "" {
			break
		}
		offset += pageSize
	}
	return results, nil
}

func (c *Client) ensureClientID(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.clientID != "" {
		id := c.clientID
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	body, _, err := c.Fetch(ctx, BaseURL+"/")
	if err != nil {
		return "", err
	}
	m := clientIDRe.FindSubmatch(body)
	if len(m) != 2 {
		return "", fmt.Errorf("extract soundcloud client_id: not found")
	}

	c.mu.Lock()
	c.clientID = string(m[1])
	id := c.clientID
	c.mu.Unlock()
	return id, nil
}

func (c *Client) fetchJSON(ctx context.Context, rawURL string, dst any) error {
	c.rateLimit()
	body, code, err := c.doGet(ctx, rawURL, "application/json, text/plain, */*")
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("unexpected HTTP %d for %s", code, rawURL)
	}
	if strings.Contains(string(body), "geo.captcha-delivery.com") {
		return fmt.Errorf("soundcloud datadome challenge for %s", rawURL)
	}
	return json.Unmarshal(body, dst)
}

func (c *Client) doGet(ctx context.Context, rawURL, accept string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", BaseURL+"/")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func (c *Client) rateLimit() {
	if c.delay <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if since := time.Since(c.lastReq); since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}

type searchResponse struct {
	Collection []json.RawMessage `json:"collection"`
	NextHref   string            `json:"next_href"`
}
