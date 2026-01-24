package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is an HTTP client for the SearXNG API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new SearXNG client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithHTTP creates a new SearXNG client with a custom HTTP client.
func NewClientWithHTTP(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// SearchRequest represents a search request to SearXNG.
type SearchRequest struct {
	Query      string
	Categories string // comma-separated list of categories
	Engines    string // comma-separated list of engines
	Language   string
	PageNo     int
	TimeRange  string // day, week, month, year
	SafeSearch int    // 0=off, 1=moderate, 2=strict
	Format     string // json or html
}

// SearchResponse represents the raw response from SearXNG.
type SearchResponse struct {
	Query           string        `json:"query"`
	NumberOfResults int64         `json:"number_of_results"`
	Results         []RawResult   `json:"results"`
	Answers         []any         `json:"answers"` // Can be strings or objects
	Corrections     []string      `json:"corrections"`
	Infoboxes       []RawInfobox  `json:"infoboxes"`
	Suggestions     []string      `json:"suggestions"`
	UnresponsiveEngines [][]any   `json:"unresponsive_engines"`
}

// RawResult represents a raw search result from SearXNG.
type RawResult struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Engine        string   `json:"engine"`
	Engines       []string `json:"engines"`
	Category      string   `json:"category"`
	Score         float64  `json:"score"`
	ParsedURL     []string `json:"parsed_url"`
	Template      string   `json:"template"`
	PublishedDate string   `json:"publishedDate"`
	PubDate       string   `json:"pubdate"`

	// Image fields
	Thumbnail    string `json:"thumbnail"`
	ThumbnailSrc string `json:"thumbnail_src"`
	ImgSrc       string `json:"img_src"`
	ImgFormat    string `json:"img_format"`
	Source       string `json:"source"`
	Resolution   string `json:"resolution"`

	// Video fields
	Duration  string `json:"duration"`
	Length    any    `json:"length"` // Can be string or number
	EmbedURL  string `json:"embed_url"`
	IframeSrc string `json:"iframe_src"`

	// Music fields
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Track  string `json:"track"`

	// File/Torrent fields
	FileSize   string `json:"filesize"`
	MagnetLink string `json:"magnetlink"`
	Seed       int    `json:"seed"`
	Leech      int    `json:"leech"`

	// Science fields
	DOI         string `json:"doi"`
	ISSN        any    `json:"issn"` // Can be string or array
	ISBN        any    `json:"isbn"` // Can be string or array
	Authors     any    `json:"authors"` // Can be array of strings or array of objects
	Publisher   string `json:"publisher"`
	Journal     string `json:"journal"`
	Type        string `json:"type"`
	AccessRight string `json:"access_right"`

	// Map fields
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	BoundingBox []string `json:"boundingbox"`
	Geojson     any      `json:"geojson"`
	Address     *RawAddress `json:"address"`
	OSMType     string   `json:"osm_type"`
	OSMID       string   `json:"osm_id"`
}

// RawAddress represents an address from SearXNG.
type RawAddress struct {
	Name        string `json:"name"`
	Road        string `json:"road"`
	Locality    string `json:"locality"`
	PostCode    string `json:"postcode"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	HouseNumber string `json:"house_number"`
}

// RawInfobox represents an infobox from SearXNG.
type RawInfobox struct {
	ID         string `json:"id"`
	Infobox    string `json:"infobox"`
	Content    string `json:"content"`
	ImgSrc     string `json:"img_src"`
	URLs       []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"urls"`
	Attributes []struct {
		Label string `json:"label"`
		Value string `json:"value"`
	} `json:"attributes"`
	Engine string `json:"engine"`
}

// Search performs a search request to SearXNG.
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	// Build URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = "/search"

	// Build query parameters
	q := u.Query()
	q.Set("q", req.Query)
	q.Set("format", "json")

	if req.Categories != "" {
		q.Set("categories", req.Categories)
	}
	if req.Engines != "" {
		q.Set("engines", req.Engines)
	}
	if req.Language != "" {
		q.Set("language", req.Language)
	}
	if req.PageNo > 1 {
		q.Set("pageno", strconv.Itoa(req.PageNo))
	}
	if req.TimeRange != "" {
		q.Set("time_range", req.TimeRange)
	}
	if req.SafeSearch > 0 {
		q.Set("safesearch", strconv.Itoa(req.SafeSearch))
	}

	u.RawQuery = q.Encode()

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "SearchBlueprint/1.0")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &searchResp, nil
}

// Healthz checks if SearXNG is healthy.
func (c *Client) Healthz(ctx context.Context) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = "/healthz"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
