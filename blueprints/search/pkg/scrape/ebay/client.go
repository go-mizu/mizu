package ebay

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Client handles HTTP requests to ebay.com with rate limiting,
// User-Agent rotation, and challenge-page detection.
type Client struct {
	http       *http.Client
	userAgents []string
	delay      time.Duration
	lastReq    time.Time
	mu         sync.Mutex
}

// NewClient creates a new eBay HTTP client.
func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     cfg.Workers + 2,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}

	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		userAgents: userAgents,
		delay:      cfg.Delay,
	}
}

// Fetch fetches raw bytes of a URL. Returns (body, statusCode, error).
func (c *Client) Fetch(ctx context.Context, rawURL string) ([]byte, int, error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c.rateLimit()

		body, code, err := c.doGet(ctx, rawURL)
		if err != nil {
			if attempt == maxAttempts {
				return nil, 0, err
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if code == 404 {
			return nil, code, nil
		}
		if code == 429 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("rate limited (HTTP 429)")
			}
			time.Sleep(time.Duration(attempt*attempt) * 5 * time.Second)
			continue
		}
		if code >= 500 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("server error HTTP %d", code)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}
		if isChallengePage(body) {
			return nil, code, fmt.Errorf("ebay challenge page served")
		}

		return body, code, nil
	}

	return nil, 0, fmt.Errorf("all attempts failed for %s", rawURL)
}

// FetchHTML fetches a URL and parses it as HTML.
func (c *Client) FetchHTML(ctx context.Context, rawURL string) (*goquery.Document, int, error) {
	body, code, err := c.Fetch(ctx, rawURL)
	if err != nil {
		return nil, code, err
	}
	if code == 404 {
		return nil, code, nil
	}
	if code != 200 {
		return nil, code, fmt.Errorf("unexpected HTTP %d for %s", code, rawURL)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, code, fmt.Errorf("parse HTML: %w", err)
	}
	return doc, code, nil
}

func (c *Client) doGet(ctx context.Context, rawURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("User-Agent", c.userAgents[rand.Intn(len(c.userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
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
	since := time.Since(c.lastReq)
	if since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}

func isChallengePage(body []byte) bool {
	s := string(body)
	return strings.Contains(s, "Pardon Our Interruption") ||
		strings.Contains(s, "ChallengeGet") ||
		strings.Contains(s, "challenge-Tfl") ||
		strings.Contains(s, "splashui")
}
