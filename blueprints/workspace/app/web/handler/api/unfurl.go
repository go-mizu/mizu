package api

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Unfurl handles URL metadata fetching for bookmarks.
type Unfurl struct {
	client *http.Client
}

// NewUnfurl creates a new unfurl handler.
func NewUnfurl() *Unfurl {
	return &Unfurl{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// UnfurlResponse is the response format for URL metadata.
type UnfurlResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
	SiteName    string `json:"siteName,omitempty"`
}

// Get fetches metadata for a URL.
func (h *Unfurl) Get(c *mizu.Ctx) error {
	rawURL := c.Query("url")
	if rawURL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url parameter required"})
	}

	// Validate URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid URL"})
	}

	// Fetch the URL
	req, err := http.NewRequestWithContext(c.Request().Context(), "GET", rawURL, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create request"})
	}

	// Set a browser-like User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WorkspaceBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := h.client.Do(req)
	if err != nil {
		// Return basic metadata on error
		return c.JSON(http.StatusOK, h.fallbackMetadata(parsedURL))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.JSON(http.StatusOK, h.fallbackMetadata(parsedURL))
	}

	// Read body (limit to 1MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return c.JSON(http.StatusOK, h.fallbackMetadata(parsedURL))
	}

	html := string(body)
	metadata := h.extractMetadata(html, parsedURL)

	return c.JSON(http.StatusOK, metadata)
}

// extractMetadata extracts Open Graph and standard meta tags from HTML.
func (h *Unfurl) extractMetadata(html string, parsedURL *url.URL) *UnfurlResponse {
	result := &UnfurlResponse{
		SiteName: strings.TrimPrefix(parsedURL.Host, "www."),
	}

	// Extract title
	if title := h.extractTag(html, `<title[^>]*>([^<]+)</title>`); title != "" {
		result.Title = strings.TrimSpace(title)
	}

	// Open Graph tags take priority
	if ogTitle := h.extractMeta(html, "og:title"); ogTitle != "" {
		result.Title = ogTitle
	}
	if ogDesc := h.extractMeta(html, "og:description"); ogDesc != "" {
		result.Description = ogDesc
	} else if desc := h.extractMeta(html, "description"); desc != "" {
		result.Description = desc
	}
	if ogImage := h.extractMeta(html, "og:image"); ogImage != "" {
		result.Image = h.resolveURL(ogImage, parsedURL)
	}
	if ogSiteName := h.extractMeta(html, "og:site_name"); ogSiteName != "" {
		result.SiteName = ogSiteName
	}

	// Twitter cards as fallback
	if result.Title == "" {
		if twTitle := h.extractMeta(html, "twitter:title"); twTitle != "" {
			result.Title = twTitle
		}
	}
	if result.Description == "" {
		if twDesc := h.extractMeta(html, "twitter:description"); twDesc != "" {
			result.Description = twDesc
		}
	}
	if result.Image == "" {
		if twImage := h.extractMeta(html, "twitter:image"); twImage != "" {
			result.Image = h.resolveURL(twImage, parsedURL)
		}
	}

	// Extract favicon
	result.Favicon = h.extractFavicon(html, parsedURL)

	// Fallback title
	if result.Title == "" {
		result.Title = result.SiteName
	}

	return result
}

// extractTag extracts content using a regex pattern.
func (h *Unfurl) extractTag(html, pattern string) string {
	re := regexp.MustCompile(`(?i)` + pattern)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return h.decodeHTMLEntities(matches[1])
	}
	return ""
}

// extractMeta extracts a meta tag content by property or name.
func (h *Unfurl) extractMeta(html, name string) string {
	// Try property attribute (Open Graph)
	patterns := []string{
		`<meta[^>]+property=["']` + regexp.QuoteMeta(name) + `["'][^>]+content=["']([^"']+)["']`,
		`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']` + regexp.QuoteMeta(name) + `["']`,
		// Try name attribute
		`<meta[^>]+name=["']` + regexp.QuoteMeta(name) + `["'][^>]+content=["']([^"']+)["']`,
		`<meta[^>]+content=["']([^"']+)["'][^>]+name=["']` + regexp.QuoteMeta(name) + `["']`,
	}

	for _, pattern := range patterns {
		if content := h.extractTag(html, pattern); content != "" {
			return content
		}
	}
	return ""
}

// extractFavicon extracts the favicon URL.
func (h *Unfurl) extractFavicon(html string, parsedURL *url.URL) string {
	// Try link tags
	patterns := []string{
		`<link[^>]+rel=["'](?:shortcut )?icon["'][^>]+href=["']([^"']+)["']`,
		`<link[^>]+href=["']([^"']+)["'][^>]+rel=["'](?:shortcut )?icon["']`,
		`<link[^>]+rel=["']apple-touch-icon["'][^>]+href=["']([^"']+)["']`,
	}

	for _, pattern := range patterns {
		if favicon := h.extractTag(html, pattern); favicon != "" {
			return h.resolveURL(favicon, parsedURL)
		}
	}

	// Fallback to Google's favicon service
	return "https://www.google.com/s2/favicons?domain=" + parsedURL.Host + "&sz=32"
}

// resolveURL resolves a potentially relative URL.
func (h *Unfurl) resolveURL(rawURL string, base *url.URL) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	return base.ResolveReference(parsed).String()
}

// decodeHTMLEntities decodes common HTML entities.
func (h *Unfurl) decodeHTMLEntities(s string) string {
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&#39;":  "'",
		"&apos;": "'",
		"&nbsp;": " ",
	}

	for entity, char := range replacements {
		s = strings.ReplaceAll(s, entity, char)
	}

	return s
}

// fallbackMetadata returns basic metadata when fetching fails.
func (h *Unfurl) fallbackMetadata(parsedURL *url.URL) *UnfurlResponse {
	domain := strings.TrimPrefix(parsedURL.Host, "www.")
	return &UnfurlResponse{
		Title:    domain,
		SiteName: domain,
		Favicon:  "https://www.google.com/s2/favicons?domain=" + parsedURL.Host + "&sz=32",
	}
}
