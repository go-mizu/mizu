package dcrawler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
)

// PinterestClient fetches Pinterest data via the internal Resource API.
// No browser or login required for public content.
type PinterestClient struct {
	client  *http.Client
	cookies []*http.Cookie
}

// NewPinterestClient creates a client and warms up the session.
func NewPinterestClient(ctx context.Context) (*PinterestClient, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	// Warm-up: GET pinterest.com to collect session cookies
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.pinterest.com/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", pinterestUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pinterest warmup: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	u, _ := url.Parse("https://www.pinterest.com/")
	cookies := jar.Cookies(u)

	return &PinterestClient{client: client, cookies: cookies}, nil
}

const pinterestUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// PinResult holds a single pin from the Pinterest API.
type PinResult struct {
	ID          string
	Title       string
	Description string
	ImageURL    string // highest-res image URL
	PinURL      string // full pinterest URL to this pin
	Width       int
	Height      int
	DomainURL   string // original source URL
	AltText     string
}

// Search queries Pinterest for pins matching the query.
// Returns all results across all pages (follows bookmarks).
func (pc *PinterestClient) Search(ctx context.Context, query string, maxPins int) ([]PinResult, error) {
	var allPins []PinResult
	var bookmark string

	for page := 1; ; page++ {
		if ctx.Err() != nil {
			return allPins, ctx.Err()
		}

		pins, nextBookmark, err := pc.searchPage(ctx, query, bookmark)
		if err != nil {
			return allPins, fmt.Errorf("page %d: %w", page, err)
		}

		allPins = append(allPins, pins...)
		fmt.Printf("\r  Page %d: %s pins so far", page, fmtInt(len(allPins)))

		if maxPins > 0 && len(allPins) >= maxPins {
			allPins = allPins[:maxPins]
			break
		}

		if isEndBookmark(nextBookmark) || len(pins) == 0 {
			break
		}
		bookmark = nextBookmark

		// Small delay between pages to avoid rate limiting
		select {
		case <-ctx.Done():
			return allPins, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	fmt.Println()

	return allPins, nil
}

func (pc *PinterestClient) searchPage(ctx context.Context, query string, bookmark string) ([]PinResult, string, error) {
	sourceURL := fmt.Sprintf("/search/pins/?q=%s&rs=typed", url.QueryEscape(query))

	options := map[string]any{
		"query":     query,
		"scope":     "pins",
		"rs":        "typed",
		"page_size": 25,
	}
	if bookmark != "" {
		options["bookmarks"] = []string{bookmark}
	}

	data := map[string]any{
		"options": options,
		"context": map[string]any{},
	}

	dataJSON, _ := json.Marshal(data)

	params := url.Values{}
	params.Set("source_url", sourceURL)
	params.Set("data", string(dataJSON))
	params.Set("_", fmt.Sprintf("%d", time.Now().UnixMilli()))

	apiURL := "https://www.pinterest.com/resource/BaseSearchResource/get/?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("User-Agent", pinterestUA)
	req.Header.Set("Accept", "application/json, text/javascript, */*, q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-Pinterest-Appstate", "active")
	req.Header.Set("X-Pinterest-Pws-Handler", "www/search/[scope].js")
	req.Header.Set("X-Pinterest-Source-Url", sourceURL)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Referer", "https://www.pinterest.com"+sourceURL)

	// Set CSRF token from cookies
	for _, c := range pc.cookies {
		if c.Name == "csrftoken" {
			req.Header.Set("X-CSRFToken", c.Value)
			break
		}
	}

	resp, err := pc.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, "", err
	}

	return parsePinterestResponse(body)
}

// pinterestImage holds a single image variant.
type pinterestImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// pinterestPin is a pin object from the API.
type pinterestPin struct {
	ID          string                    `json:"id"`
	Type        string                    `json:"type"`
	Title       string                    `json:"title"`
	GridTitle   string                    `json:"grid_title"`
	Description string                    `json:"description"`
	AutoAltText string                    `json:"auto_alt_text"`
	DomainURL   string                    `json:"link"`
	Images      map[string]pinterestImage `json:"images"`
}

// pinterestResponse matches the actual API response structure.
// resource_response.data is a map with "results" array inside.
// resource_response.bookmark is singular (not plural).
type pinterestResponse struct {
	ResourceResponse struct {
		Data struct {
			Results []pinterestPin `json:"results"`
		} `json:"data"`
		Bookmark string `json:"bookmark"`
	} `json:"resource_response"`
}

func parsePinterestResponse(body []byte) ([]PinResult, string, error) {
	var resp pinterestResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("parse response: %w", err)
	}

	var results []PinResult
	for _, p := range resp.ResourceResponse.Data.Results {
		if p.ID == "" || p.Type != "pin" {
			continue
		}
		imgURL, w, h := bestImage(p.Images)
		if imgURL == "" {
			continue
		}

		title := p.GridTitle
		if title == "" {
			title = p.Title
		}
		alt := p.AutoAltText
		if alt == "" {
			alt = title
		}

		results = append(results, PinResult{
			ID:          p.ID,
			Title:       title,
			Description: p.Description,
			ImageURL:    imgURL,
			PinURL:      fmt.Sprintf("https://www.pinterest.com/pin/%s/", p.ID),
			Width:       w,
			Height:      h,
			DomainURL:   p.DomainURL,
			AltText:     alt,
		})
	}

	return results, resp.ResourceResponse.Bookmark, nil
}

// bestImage returns the highest-resolution image URL from a pin's images map.
// Preference: orig > 736x > 474x > 236x > anything else
func bestImage(images map[string]pinterestImage) (string, int, int) {
	priority := []string{"orig", "736x", "474x", "236x"}
	for _, key := range priority {
		if img, ok := images[key]; ok && img.URL != "" {
			return img.URL, img.Width, img.Height
		}
	}
	// Fallback: pick the largest
	var bestURL string
	var bestW, bestH int
	for _, img := range images {
		if img.Width*img.Height > bestW*bestH {
			bestURL = img.URL
			bestW = img.Width
			bestH = img.Height
		}
	}
	return bestURL, bestW, bestH
}

func isEndBookmark(bookmark string) bool {
	return bookmark == "" || bookmark == "-end-" || strings.HasPrefix(bookmark, "Y2JOb25lO")
}

// IsPinterestDomain checks if the domain is Pinterest.
func IsPinterestDomain(domain string) bool {
	d := NormalizeDomain(domain)
	return d == "pinterest.com" || strings.HasSuffix(d, ".pinterest.com")
}

// ExtractPinterestQuery extracts the search query from a Pinterest search URL.
// Returns empty string if not a search URL.
func ExtractPinterestQuery(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(u.Path, "/search/") {
		return ""
	}
	return u.Query().Get("q")
}

// RunPinterestSearch runs a Pinterest search crawl using the internal API.
// Stores results in the ResultDB and optionally downloads images.
func RunPinterestSearch(ctx context.Context, c *Crawler, query string) error {
	fmt.Printf("  Pinterest API mode (no browser needed)\n")
	fmt.Printf("  Query: %q\n\n", query)

	pc, err := NewPinterestClient(ctx)
	if err != nil {
		return fmt.Errorf("pinterest client: %w", err)
	}

	maxPins := c.config.MaxPages
	if maxPins <= 0 {
		maxPins = 500
	}

	fmt.Printf("  Searching (max %s pins)...\n", fmtInt(maxPins))
	pins, err := pc.Search(ctx, query, maxPins)
	if err != nil && len(pins) == 0 {
		return fmt.Errorf("search: %w", err)
	}
	fmt.Printf("  Found %s pins\n", fmtInt(len(pins)))

	// Store results in ResultDB
	rdb, err := NewResultDB(c.config.ResultDir(), c.config.ShardCount, c.config.BatchSize)
	if err != nil {
		return fmt.Errorf("result db: %w", err)
	}
	defer rdb.Close()

	rdb.SetMeta("domain", "pinterest.com")
	rdb.SetMeta("query", query)

	now := time.Now()
	for i, pin := range pins {
		// Store pin as a page
		result := Result{
			URL:         pin.PinURL,
			URLHash:     xxhash.Sum64String(pin.PinURL),
			Depth:       0,
			StatusCode:  200,
			ContentType: "application/json",
			Title:       pin.Title,
			Description: pin.Description,
			CrawledAt:   now,
		}
		rdb.AddPage(result)

		// Store image as a link
		var links []Link
		if pin.ImageURL != "" {
			links = append(links, Link{
				TargetURL:  pin.ImageURL,
				AnchorText: truncate(pin.AltText, 200),
				Rel:        "image",
				IsInternal: false,
			})
		}
		if pin.DomainURL != "" {
			links = append(links, Link{
				TargetURL:  pin.DomainURL,
				AnchorText: truncate(pin.Title, 200),
				Rel:        "source",
				IsInternal: false,
			})
		}
		if len(links) > 0 {
			rdb.AddLinks(result.URLHash, links)
		}

		if (i+1)%50 == 0 {
			fmt.Printf("\r  Stored: %s / %s pins", fmtInt(i+1), fmtInt(len(pins)))
		}
	}
	fmt.Printf("\r  Stored: %s pins with images          \n", fmtInt(len(pins)))

	c.resultDB = rdb
	return nil
}
