package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Client struct {
	http       *http.Client
	userAgents []string
	delay      time.Duration
	lastReq    time.Time
}

func NewClient(cfg Config) *Client {
	return &Client{
		http: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxConnsPerHost:     cfg.Workers + 2,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		userAgents: userAgents,
		delay:      cfg.Delay,
	}
}

func (c *Client) Fetch(ctx context.Context, url string) ([]byte, int, error) {
	c.rateLimit()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.userAgents[rand.Intn(len(c.userAgents))])
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.AddCookie(&http.Cookie{Name: "CONSENT", Value: "YES+"})
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (c *Client) FetchHTML(ctx context.Context, url string) (*goquery.Document, int, error) {
	body, code, err := c.Fetch(ctx, url)
	if err != nil {
		return nil, code, err
	}
	if code == 404 {
		return nil, code, nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, code, err
	}
	return doc, code, nil
}

func (c *Client) FetchPageData(ctx context.Context, url string) (*PageData, int, error) {
	body, code, err := c.Fetch(ctx, url)
	if err != nil {
		return nil, code, err
	}
	if code == 404 {
		return nil, code, nil
	}
	html := string(body)
	data := &PageData{
		HTML:          html,
		InitialData:   parseJSONAny(extractJSONVar(html, "var ytInitialData = ")),
		PlayerResp:    parseJSONAny(extractJSONVar(html, "var ytInitialPlayerResponse = ")),
		YTCFG:         parseJSONObject(extractJSONCall(html, "ytcfg.set(")),
		APIKey:        extractQuotedConfig(html, "INNERTUBE_API_KEY"),
		ClientVersion: extractQuotedConfig(html, "INNERTUBE_CLIENT_VERSION"),
		VisitorData:   extractQuotedConfig(html, "VISITOR_DATA"),
	}
	if data.APIKey == "" && data.YTCFG != nil {
		data.APIKey = stringValue(data.YTCFG["INNERTUBE_API_KEY"])
	}
	if data.ClientVersion == "" && data.YTCFG != nil {
		data.ClientVersion = stringValue(data.YTCFG["INNERTUBE_CLIENT_VERSION"])
	}
	if data.VisitorData == "" && data.YTCFG != nil {
		data.VisitorData = stringValue(data.YTCFG["VISITOR_DATA"])
	}
	return data, code, nil
}

func (c *Client) FetchTranscript(ctx context.Context, url string) (string, error) {
	body, code, err := c.Fetch(ctx, url)
	if err != nil {
		return "", err
	}
	if code != 200 {
		return "", fmt.Errorf("caption track returned HTTP %d", code)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	var parts []string
	doc.Find("text").Each(func(_ int, s *goquery.Selection) {
		text := cleanWhitespace(s.Text())
		if text != "" {
			parts = append(parts, text)
		}
	})
	return strings.Join(parts, "\n"), nil
}

func (c *Client) rateLimit() {
	if c.delay <= 0 {
		return
	}
	if since := time.Since(c.lastReq); since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}

func parseJSONAny(raw string) any {
	if raw == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil
	}
	return v
}

func parseJSONObject(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil
	}
	return v
}
