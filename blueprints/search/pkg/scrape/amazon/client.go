package amazon

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	cfg        Config
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) SearchURL(query string, page int) string {
	v := url.Values{}
	v.Set("k", query)
	if page > 1 {
		v.Set("page", fmt.Sprintf("%d", page))
	}
	if strings.TrimSpace(c.cfg.SortBy) != "" {
		v.Set("s", c.cfg.SortBy)
	}
	return fmt.Sprintf("https://%s/s?%s", c.cfg.Market, v.Encode())
}

func (c *Client) FetchSearchPage(ctx context.Context, query string, page int) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.SearchURL(query, page), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("amazon status=%d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func sleepRate(rate float64) {
	if rate <= 0 {
		return
	}
	time.Sleep(time.Duration(float64(time.Second) / rate))
}
