package facebook

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Client struct {
	http       *http.Client
	userAgents []string
	delay      time.Duration
	lastReq    time.Time
	cookies    string
	preferMB   bool
	mu         sync.Mutex
}

func NewClient(cfg Config) (*Client, error) {
	cookies := strings.TrimSpace(cfg.Cookies)
	if cookies == "" {
		cookies = strings.TrimSpace(os.Getenv("FACEBOOK_COOKIE"))
	}

	cookieFile := strings.TrimSpace(cfg.CookiesFile)
	if cookieFile == "" {
		cookieFile = strings.TrimSpace(os.Getenv("FACEBOOK_COOKIE_FILE"))
	}
	if cookies == "" && cookieFile != "" {
		b, err := os.ReadFile(cookieFile)
		if err != nil {
			return nil, fmt.Errorf("read cookie file: %w", err)
		}
		cookies = strings.TrimSpace(string(b))
	}

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
		cookies:    cookies,
		preferMB:   cfg.PreferMBasic,
	}, nil
}

func (c *Client) Fetch(ctx context.Context, rawURL string) ([]byte, int, error) {
	const maxAttempts = 3
	target := NormalizeURL(rawURL, c.preferMB)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c.rateLimit()

		body, code, err := c.doGet(ctx, target)
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
		if code == 429 || code == 503 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("facebook rate limited/challenged with HTTP %d", code)
			}
			time.Sleep(time.Duration(attempt*attempt) * 4 * time.Second)
			continue
		}
		if code >= 500 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("server error HTTP %d", code)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}
		if isLoginWall(body) {
			return body, code, fmt.Errorf("facebook login wall detected; provide cookies for more coverage")
		}
		if isErrorShell(body) {
			return body, code, fmt.Errorf("facebook served an error shell page; anonymous access is limited for this target")
		}
		return body, code, nil
	}
	return nil, 0, fmt.Errorf("all attempts failed")
}

func (c *Client) FetchHTML(ctx context.Context, rawURL string) (*goquery.Document, int, error) {
	body, code, err := c.Fetch(ctx, rawURL)
	if err != nil {
		return nil, code, err
	}
	if code == 404 {
		return nil, code, nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, code, fmt.Errorf("parse html: %w", err)
	}
	return doc, code, nil
}

func (c *Client) doGet(ctx context.Context, rawURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.userAgents[rand.Intn(len(c.userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", BaseURL+"/")
	if c.cookies != "" {
		req.Header.Set("Cookie", c.cookies)
	}

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

func isLoginWall(body []byte) bool {
	s := strings.ToLower(string(body))
	return strings.Contains(s, "login to facebook") ||
		strings.Contains(s, "log in to facebook") ||
		strings.Contains(s, "you must log in") ||
		strings.Contains(s, "enter mobile number or email")
}

func isErrorShell(body []byte) bool {
	s := strings.ToLower(string(body))
	return strings.Contains(s, "error facebook") ||
		strings.Contains(s, "facebook is not available on this browser") ||
		strings.Contains(s, "sorry, something went wrong") ||
		strings.Contains(s, "the link you followed may be broken") ||
		strings.Contains(s, "\"error_code\"") ||
		strings.Contains(s, "www.facebook.com/common/scribe_endpoint.php")
}
