package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cfBaseURLTemplate = "https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering"
	DefaultProxyURL   = "https://browser.go-mizu.workers.dev/api"
)

// Credentials holds CF account credentials.
type Credentials struct {
	AccountID string `json:"account_id"`
	APIToken  string `json:"api_token"`
}

// LoadCredentials reads credentials from $HOME/data/cloudflare/cloudflare.json.
func LoadCredentials() (Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Credentials{}, fmt.Errorf("get home dir: %w", err)
	}
	path := filepath.Join(home, "data", "cloudflare", "cloudflare.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, fmt.Errorf("read %s: %w", path, err)
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return Credentials{}, fmt.Errorf("parse cloudflare.json: %w", err)
	}
	if creds.AccountID == "" {
		return Credentials{}, fmt.Errorf("cloudflare.json missing account_id")
	}
	if creds.APIToken == "" {
		return Credentials{}, fmt.Errorf("cloudflare.json missing api_token")
	}
	return creds, nil
}

// ProxyConfig holds configuration for the self-hosted browser worker proxy
// (tools/browser deployed at DefaultProxyURL).
type ProxyConfig struct {
	URL   string // worker base URL, e.g. DefaultProxyURL
	Token string // AUTH_TOKEN secret
}

// LoadProxyConfig reads MIZU_TOKEN from the environment and returns a
// ProxyConfig using DefaultProxyURL. Set BROWSER_PROXY_URL to override the URL.
func LoadProxyConfig() (ProxyConfig, error) {
	token := os.Getenv("MIZU_TOKEN")
	if token == "" {
		return ProxyConfig{}, fmt.Errorf("MIZU_TOKEN env var not set")
	}
	u := os.Getenv("BROWSER_PROXY_URL")
	if u == "" {
		u = DefaultProxyURL
	}
	return ProxyConfig{URL: u, Token: token}, nil
}

// Client sends requests to the CF Browser Rendering REST API.
//
// When a ProxyConfig is set (via NewClientWithProxy), every request is routed
// through the self-hosted worker first. The worker caches results in D1 and
// has its own fallback layer — so most calls never reach CF.
//
// If the proxy fails or is unavailable for an endpoint, the client falls back
// to the direct CF API. When CF returns 429 the reset time is cached in-memory
// and subsequent calls skip CF until the window expires.
type Client struct {
	creds     Credentials
	proxy     ProxyConfig
	http      *http.Client
	cfBaseURL string

	mu               sync.RWMutex
	cfRateLimitUntil time.Time
}

// NewClient creates a Client that calls CF directly (no proxy).
func NewClient(creds Credentials) *Client {
	return &Client{
		creds:     creds,
		http:      &http.Client{Timeout: 60 * time.Second},
		cfBaseURL: fmt.Sprintf(cfBaseURLTemplate, creds.AccountID),
	}
}

// NewClientWithProxy creates a Client that routes all requests through the
// proxy worker first, falling back to CF when the proxy cannot serve a request.
func NewClientWithProxy(creds Credentials, proxy ProxyConfig) *Client {
	c := NewClient(creds)
	c.proxy = proxy
	return c
}

// CFRateLimitUntil returns when direct CF calls can resume (zero = not limited).
func (c *Client) CFRateLimitUntil() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfRateLimitUntil
}

func (c *Client) setCFRateLimit(d time.Duration) {
	c.mu.Lock()
	c.cfRateLimitUntil = time.Now().Add(d)
	c.mu.Unlock()
}

func (c *Client) cfIsRateLimited() bool {
	c.mu.RLock()
	until := c.cfRateLimitUntil
	c.mu.RUnlock()
	return !until.IsZero() && time.Now().Before(until)
}

// ── apiResponse envelope ──────────────────────────────────────────────────────

type apiResponse struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []apiError      `json:"errors"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── Low-level HTTP helpers ────────────────────────────────────────────────────

// doPostJSON POSTs to baseURL+path using token, decodes the CF envelope.
// Returns retryAfter > 0 when the server responds with 429.
func (c *Client) doPostJSON(baseURL, token, path string, req, out any) (retryAfter time.Duration, err error) {
	body, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("POST %s%s: %w", baseURL, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		ra := parseRetryAfter(resp.Header.Get("Retry-After"))
		return ra, fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}

	var env apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return 0, fmt.Errorf("decode response (HTTP %d): %w", resp.StatusCode, err)
	}
	if !env.Success {
		if len(env.Errors) > 0 {
			return 0, fmt.Errorf("API error %d: %s", env.Errors[0].Code, env.Errors[0].Message)
		}
		return 0, fmt.Errorf("API returned success=false (HTTP %d)", resp.StatusCode)
	}
	if out != nil {
		return 0, json.Unmarshal(env.Result, out)
	}
	return 0, nil
}

// doPostBinary POSTs to baseURL+path using token.
// If the response Content-Type is binary (image/* or application/pdf) the raw
// bytes are returned. Otherwise the response is decoded as a JSON error envelope.
// Returns retryAfter > 0 on 429.
func (c *Client) doPostBinary(baseURL, token, path string, req any) (data []byte, retryAfter time.Duration, err error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("POST %s%s: %w", baseURL, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		ra := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, ra, fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}

	if isBinaryContentType(resp.Header.Get("Content-Type")) {
		b, err := io.ReadAll(resp.Body)
		return b, 0, err
	}

	// JSON response — decode as error envelope.
	var env apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, 0, fmt.Errorf("unexpected response (HTTP %d): %w", resp.StatusCode, err)
	}
	if !env.Success {
		if len(env.Errors) > 0 {
			return nil, 0, fmt.Errorf("API error %d: %s", env.Errors[0].Code, env.Errors[0].Message)
		}
		return nil, 0, fmt.Errorf("API returned success=false (HTTP %d)", resp.StatusCode)
	}
	return nil, 0, fmt.Errorf("expected binary response, got JSON success (HTTP %d)", resp.StatusCode)
}

// doGetJSON sends GET to rawURL with token and decodes the CF envelope.
func (c *Client) doGetJSON(rawURL, token string, out any) (retryAfter time.Duration, err error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		ra := parseRetryAfter(resp.Header.Get("Retry-After"))
		return ra, fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}

	var env apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return 0, fmt.Errorf("decode response (HTTP %d): %w", resp.StatusCode, err)
	}
	if !env.Success {
		if len(env.Errors) > 0 {
			return 0, fmt.Errorf("API error %d: %s", env.Errors[0].Code, env.Errors[0].Message)
		}
		return 0, fmt.Errorf("API returned success=false (HTTP %d)", resp.StatusCode)
	}
	if out != nil {
		return 0, json.Unmarshal(env.Result, out)
	}
	return 0, nil
}

// doDeleteHTTP sends DELETE to rawURL with token.
func (c *Client) doDeleteHTTP(rawURL, token string) error {
	req, err := http.NewRequest(http.MethodDelete, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("DELETE: HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}

// ── Routing helpers (proxy → CF with rate-limit cache) ────────────────────────

// postJSON routes to proxy first, then CF.
func (c *Client) postJSON(path string, req, out any) error {
	if c.proxy.URL != "" {
		if _, err := c.doPostJSON(c.proxy.URL, c.proxy.Token, path, req, out); err == nil {
			return nil
		}
		// proxy error → fall through to CF
	}
	if c.cfIsRateLimited() {
		until := c.CFRateLimitUntil()
		return fmt.Errorf("CF rate limited until %s (%.0f min remaining)",
			until.Format(time.RFC3339), time.Until(until).Minutes())
	}
	ra, err := c.doPostJSON(c.cfBaseURL, c.creds.APIToken, path, req, out)
	if ra > 0 {
		c.setCFRateLimit(ra)
	}
	return err
}

// postBinary routes to proxy first (accepts binary response), then CF.
func (c *Client) postBinary(path string, req any) ([]byte, error) {
	if c.proxy.URL != "" {
		if data, _, err := c.doPostBinary(c.proxy.URL, c.proxy.Token, path, req); err == nil {
			return data, nil
		}
		// proxy error (e.g. 503 "no browser") → fall through to CF
	}
	if c.cfIsRateLimited() {
		until := c.CFRateLimitUntil()
		return nil, fmt.Errorf("CF rate limited until %s (%.0f min remaining)",
			until.Format(time.RFC3339), time.Until(until).Minutes())
	}
	data, ra, err := c.doPostBinary(c.cfBaseURL, c.creds.APIToken, path, req)
	if ra > 0 {
		c.setCFRateLimit(ra)
	}
	return data, err
}

// crawlBase returns the base URL + token for crawl GET/DELETE operations.
// Proxy is preferred when configured.
func (c *Client) crawlBase() (baseURL, token string) {
	if c.proxy.URL != "" {
		return c.proxy.URL, c.proxy.Token
	}
	return c.cfBaseURL, c.creds.APIToken
}

// getJSON and doDelete are called from crawl.go with full URLs built via crawlBase.
func (c *Client) getJSON(rawURL string, out any) error {
	_, token := c.crawlBase()
	_, err := c.doGetJSON(rawURL, token, out)
	return err
}

func (c *Client) doDelete(rawURL string) error {
	_, token := c.crawlBase()
	return c.doDeleteHTTP(rawURL, token)
}

// ── Utilities ─────────────────────────────────────────────────────────────────

func parseRetryAfter(s string) time.Duration {
	s = strings.TrimSpace(s)
	secs, err := strconv.Atoi(s)
	if err != nil || secs <= 0 {
		return 60 * time.Second
	}
	return time.Duration(secs) * time.Second
}

func isBinaryContentType(ct string) bool {
	ct = strings.ToLower(strings.SplitN(ct, ";", 2)[0])
	return strings.HasPrefix(ct, "image/") || ct == "application/pdf"
}

// ── Shared option types ───────────────────────────────────────────────────────

// GotoOptions controls page navigation behaviour.
type GotoOptions struct {
	WaitUntil string `json:"waitUntil,omitempty"` // load|domcontentloaded|networkidle0|networkidle2
	Timeout   int    `json:"timeout,omitempty"`   // ms
}

// WaitForSelector waits for a CSS selector before returning.
type WaitForSelector struct {
	Selector string `json:"selector"`
	Timeout  int    `json:"timeout,omitempty"` // ms
	Visible  bool   `json:"visible,omitempty"`
}

// Viewport sets the browser viewport size.
type Viewport struct {
	Width             int     `json:"width,omitempty"`
	Height            int     `json:"height,omitempty"`
	DeviceScaleFactor float64 `json:"deviceScaleFactor,omitempty"`
}

// Cookie sets a browser cookie.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	URL   string `json:"url,omitempty"`
}

// Authenticate provides HTTP Basic Auth credentials.
type Authenticate struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ScriptTag injects a JavaScript tag.
type ScriptTag struct {
	URL     string `json:"url,omitempty"`
	Content string `json:"content,omitempty"`
}

// StyleTag injects a CSS tag.
type StyleTag struct {
	URL     string `json:"url,omitempty"`
	Content string `json:"content,omitempty"`
}

// CommonRequest holds fields shared across most Browser Rendering endpoints.
// At least one of URL or HTML must be set.
type CommonRequest struct {
	URL                  string            `json:"url,omitempty"`
	HTML                 string            `json:"html,omitempty"`
	UserAgent            string            `json:"userAgent,omitempty"`
	Authenticate         *Authenticate     `json:"authenticate,omitempty"`
	Cookies              []Cookie          `json:"cookies,omitempty"`
	GotoOptions          *GotoOptions      `json:"gotoOptions,omitempty"`
	WaitForSelector      *WaitForSelector  `json:"waitForSelector,omitempty"`
	SetExtraHTTPHeaders  map[string]string `json:"setExtraHTTPHeaders,omitempty"`
	RejectResourceTypes  []string          `json:"rejectResourceTypes,omitempty"`
	RejectRequestPattern []string          `json:"rejectRequestPattern,omitempty"`
}

// ── Endpoint request/response types ──────────────────────────────────────────

// ContentRequest fetches rendered HTML.
type ContentRequest struct{ CommonRequest }

// MarkdownRequest extracts Markdown from a page.
type MarkdownRequest struct{ CommonRequest }

// LinksRequest extracts hyperlinks from a page.
type LinksRequest struct {
	CommonRequest
	VisibleLinksOnly     bool `json:"visibleLinksOnly,omitempty"`
	ExcludeExternalLinks bool `json:"excludeExternalLinks,omitempty"`
}

// ScreenshotOptions controls screenshot capture.
type ScreenshotOptions struct {
	OmitBackground        bool   `json:"omitBackground,omitempty"`
	FullPage              bool   `json:"fullPage,omitempty"`
	Quality               int    `json:"quality,omitempty"`
	Type                  string `json:"type,omitempty"` // png (default) | jpeg
	CaptureBeyondViewport bool   `json:"captureBeyondViewport,omitempty"`
}

// ScreenshotRequest captures a screenshot.
type ScreenshotRequest struct {
	CommonRequest
	ScreenshotOptions *ScreenshotOptions `json:"screenshotOptions,omitempty"`
	Viewport          *Viewport          `json:"viewport,omitempty"`
	Selector          string             `json:"selector,omitempty"`
	AddScriptTag      []ScriptTag        `json:"addScriptTag,omitempty"`
	AddStyleTag       []StyleTag         `json:"addStyleTag,omitempty"`
}

// PDFOptions controls PDF rendering.
type PDFOptions struct {
	Format          string  `json:"format,omitempty"`
	Landscape       bool    `json:"landscape,omitempty"`
	Scale           float64 `json:"scale,omitempty"`
	PrintBackground bool    `json:"printBackground,omitempty"`
}

// PDFRequest renders a page as PDF.
type PDFRequest struct {
	CommonRequest
	PDFOptions          *PDFOptions `json:"pdfOptions,omitempty"`
	Viewport            *Viewport   `json:"viewport,omitempty"`
	DisplayHeaderFooter bool        `json:"displayHeaderFooter,omitempty"`
	HeaderTemplate      string      `json:"headerTemplate,omitempty"`
	FooterTemplate      string      `json:"footerTemplate,omitempty"`
	AddStyleTag         []StyleTag  `json:"addStyleTag,omitempty"`
}

// SnapshotRequest captures both screenshot and HTML in one call.
type SnapshotRequest struct {
	CommonRequest
	ScreenshotOptions    *ScreenshotOptions `json:"screenshotOptions,omitempty"`
	Viewport             *Viewport          `json:"viewport,omitempty"`
	AddScriptTag         []ScriptTag        `json:"addScriptTag,omitempty"`
	SetJavaScriptEnabled *bool              `json:"setJavaScriptEnabled,omitempty"`
}

// SnapshotResult contains a base64 screenshot and the page HTML.
type SnapshotResult struct {
	Screenshot string `json:"screenshot"` // base64-encoded PNG
	Content    string `json:"content"`    // HTML
}

// ScrapeElement specifies a CSS selector to scrape.
type ScrapeElement struct {
	Selector string `json:"selector"`
}

// ScrapeRequest extracts HTML elements matching CSS selectors.
type ScrapeRequest struct {
	CommonRequest
	Elements []ScrapeElement `json:"elements"`
}

// ScrapedAttribute is a single HTML attribute key/value pair.
type ScrapedAttribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ScrapedElement holds the extracted data for one DOM element.
type ScrapedElement struct {
	Text       string             `json:"text"`
	HTML       string             `json:"html"`
	Attributes []ScrapedAttribute `json:"attributes"`
	Height     float64            `json:"height"`
	Width      float64            `json:"width"`
	Top        float64            `json:"top"`
	Left       float64            `json:"left"`
}

// ScrapeSelector groups results for one CSS selector.
type ScrapeSelector struct {
	Selector string           `json:"selector"`
	Results  []ScrapedElement `json:"results"`
}

// JSONRequest extracts structured data from a page using AI.
// At least one of Prompt or ResponseFormat must be set.
type JSONRequest struct {
	CommonRequest
	Prompt         string         `json:"prompt,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

// ── Synchronous endpoint methods ─────────────────────────────────────────────

// Content fetches the rendered HTML of a page.
func (c *Client) Content(req ContentRequest) (string, error) {
	var result string
	return result, c.postJSON("/content", req, &result)
}

// Markdown extracts Markdown from a page.
func (c *Client) Markdown(req MarkdownRequest) (string, error) {
	var result string
	return result, c.postJSON("/markdown", req, &result)
}

// Links extracts hyperlinks from a page.
func (c *Client) Links(req LinksRequest) ([]string, error) {
	var result []string
	return result, c.postJSON("/links", req, &result)
}

// Screenshot captures a screenshot of a page and returns raw PNG/JPEG bytes.
func (c *Client) Screenshot(req ScreenshotRequest) ([]byte, error) {
	return c.postBinary("/screenshot", req)
}

// PDF renders a page as PDF and returns raw PDF bytes.
func (c *Client) PDF(req PDFRequest) ([]byte, error) {
	return c.postBinary("/pdf", req)
}

// Snapshot captures both a screenshot and the HTML in one request.
func (c *Client) Snapshot(req SnapshotRequest) (*SnapshotResult, error) {
	var result SnapshotResult
	return &result, c.postJSON("/snapshot", req, &result)
}

// Scrape extracts HTML elements matching CSS selectors.
func (c *Client) Scrape(req ScrapeRequest) ([]ScrapeSelector, error) {
	var result []ScrapeSelector
	return result, c.postJSON("/scrape", req, &result)
}

// JSON extracts structured data from a page using AI.
func (c *Client) JSON(req JSONRequest) (map[string]any, error) {
	var result map[string]any
	return result, c.postJSON("/json", req, &result)
}
