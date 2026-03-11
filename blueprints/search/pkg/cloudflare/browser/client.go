package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const baseURLTemplate = "https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering"

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

// Client sends requests to the CF Browser Rendering REST API.
type Client struct {
	creds   Credentials
	http    *http.Client
	baseURL string
}

// NewClient creates a new Client.
func NewClient(creds Credentials) *Client {
	return &Client{
		creds:   creds,
		http:    &http.Client{Timeout: 60 * time.Second},
		baseURL: fmt.Sprintf(baseURLTemplate, creds.AccountID),
	}
}

// apiResponse is the CF envelope for JSON endpoints.
type apiResponse struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []apiError      `json:"errors"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// postJSON POSTs JSON and decodes the envelope, placing result into out.
func (c *Client) postJSON(path string, req, out any) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.creds.APIToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}

	var env apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !env.Success {
		if len(env.Errors) > 0 {
			return fmt.Errorf("API error %d: %s", env.Errors[0].Code, env.Errors[0].Message)
		}
		return fmt.Errorf("API returned success=false")
	}
	if out != nil {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}

// postBinary POSTs JSON and returns the raw binary response body.
func (c *Client) postBinary(path string, req any) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.creds.APIToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("POST %s: HTTP %d: %s", path, resp.StatusCode, b)
	}
	return io.ReadAll(resp.Body)
}

// getJSON sends GET to rawURL and decodes the envelope into out.
func (c *Client) getJSON(rawURL string, out any) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.creds.APIToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limited (429); retry after %s", resp.Header.Get("Retry-After"))
	}

	var env apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !env.Success {
		if len(env.Errors) > 0 {
			return fmt.Errorf("API error %d: %s", env.Errors[0].Code, env.Errors[0].Message)
		}
		return fmt.Errorf("API returned success=false")
	}
	if out != nil {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}

// doDelete sends DELETE to rawURL.
func (c *Client) doDelete(rawURL string) error {
	req, err := http.NewRequest(http.MethodDelete, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.creds.APIToken)

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

// --- Shared option types ---

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
	RejectResourceTypes  []string          `json:"rejectResourceTypes,omitempty"`  // image|media|font|stylesheet
	RejectRequestPattern []string          `json:"rejectRequestPattern,omitempty"` // regex
}

// --- Endpoint request/response types ---

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
	Format          string  `json:"format,omitempty"` // A4|Letter|etc
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

// --- Synchronous endpoint methods ---

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
