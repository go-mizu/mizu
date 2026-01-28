package engines

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// GSA User Agents - Google Search App user agents that work better with Google
var gsaUserAgents = []string{
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 18_5_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; SM-S928B Build/UP1A.231005.007) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.230 Mobile Safari/537.36 GSA/15.3.36.28.arm64",
	"Mozilla/5.0 (Linux; Android 13; Pixel 7 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Mobile Safari/537.36 GSA/14.50.15.29.arm64",
}

// arcIDRange contains characters used for arc_id generation (like SearXNG)
const arcIDRange = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

// Google implements Google web search using SearXNG's bypass techniques.
type Google struct {
	*BaseEngine
	timeRangeMap map[TimeRange]string
	filterMap    map[SafeSearchLevel]string
	arcID        string
	arcIDTime    time.Time
	arcIDMu      sync.Mutex
}

// NewGoogle creates a new Google engine.
func NewGoogle() *Google {
	g := &Google{
		BaseEngine: NewBaseEngine("google", "g", []Category{CategoryGeneral, CategoryWeb}),
		timeRangeMap: map[TimeRange]string{
			TimeRangeDay:   "d",
			TimeRangeWeek:  "w",
			TimeRangeMonth: "m",
			TimeRangeYear:  "y",
		},
		filterMap: map[SafeSearchLevel]string{
			SafeSearchOff:      "off",
			SafeSearchModerate: "medium",
			SafeSearchStrict:   "high",
		},
	}

	g.SetPaging(true).
		SetMaxPage(50).
		SetTimeRangeSupport(true).
		SetSafeSearch(true).
		SetTimeout(10 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://www.google.com",
			WikidataID: "Q9366",
			Results:    "HTML",
		})

	// Set up traits
	g.traits.AllLocale = "ZZ"
	g.traits.Languages["en"] = "lang_en"
	g.traits.Languages["de"] = "lang_de"
	g.traits.Languages["fr"] = "lang_fr"
	g.traits.Languages["es"] = "lang_es"
	g.traits.Languages["it"] = "lang_it"
	g.traits.Languages["pt"] = "lang_pt"
	g.traits.Languages["ja"] = "lang_ja"
	g.traits.Languages["ko"] = "lang_ko"
	g.traits.Languages["zh"] = "lang_zh-CN"
	g.traits.Languages["ru"] = "lang_ru"

	g.traits.Regions["en-US"] = "US"
	g.traits.Regions["en-GB"] = "GB"
	g.traits.Regions["de-DE"] = "DE"
	g.traits.Regions["fr-FR"] = "FR"
	g.traits.Regions["es-ES"] = "ES"
	g.traits.Regions["it-IT"] = "IT"
	g.traits.Regions["pt-BR"] = "BR"
	g.traits.Regions["ja-JP"] = "JP"
	g.traits.Regions["ko-KR"] = "KR"
	g.traits.Regions["zh-CN"] = "CN"
	g.traits.Regions["ru-RU"] = "RU"

	g.traits.Custom["supported_domains"] = map[string]string{
		"US": "www.google.com",
		"GB": "www.google.co.uk",
		"DE": "www.google.de",
		"FR": "www.google.fr",
		"ES": "www.google.es",
		"IT": "www.google.it",
		"BR": "www.google.com.br",
		"JP": "www.google.co.jp",
		"KR": "www.google.co.kr",
		"CN": "www.google.com.hk",
		"RU": "www.google.ru",
	}

	return g
}

// getArcID generates or returns cached arc_id (changes hourly like SearXNG)
func (g *Google) getArcID() string {
	g.arcIDMu.Lock()
	defer g.arcIDMu.Unlock()

	// Regenerate if older than 1 hour (like SearXNG)
	if g.arcID == "" || time.Since(g.arcIDTime) > time.Hour {
		g.arcID = generateArcID()
		g.arcIDTime = time.Now()
	}
	return g.arcID
}

// generateArcID creates a 23-character random string (like SearXNG)
func generateArcID() string {
	b := make([]byte, 23)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(arcIDRange))))
		b[i] = arcIDRange[n.Int64()]
	}
	return string(b)
}

// uiAsync formats the async parameter for Google's UI request (like SearXNG)
// Format: arc_id:srp_<random_23_chars>_1<start:02>,use_ac:true,_fmt:prog
func (g *Google) uiAsync(start int) string {
	arcID := fmt.Sprintf("arc_id:srp_%s_1%02d", g.getArcID(), start)
	return fmt.Sprintf("%s,use_ac:true,_fmt:prog", arcID)
}

// getRandomGSAUserAgent returns a random GSA user agent
func getRandomGSAUserAgent() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(gsaUserAgents))))
	return gsaUserAgents[n.Int64()]
}

func (g *Google) Request(ctx context.Context, query string, params *RequestParams) error {
	start := (params.PageNo - 1) * 10
	asyncParam := g.uiAsync(start)

	// Get language and region info
	engLang := g.traits.GetLanguage(params.Locale, "lang_en")
	engRegion := g.traits.GetRegion(params.Locale, "US")
	langCode := strings.TrimPrefix(engLang, "lang_") // lang_zh-TW --> zh-TW

	// Get domain
	domains := g.traits.Custom["supported_domains"].(map[string]string)
	domain := domains[engRegion]
	if domain == "" {
		domain = "www.google.com"
	}

	// Build URL with async parameters (like SearXNG)
	// Parameter order matters for Google
	queryParams := url.Values{}
	queryParams.Set("q", query)

	// hl parameter: interface language (e.g., "en-US")
	queryParams.Set("hl", fmt.Sprintf("%s-%s", langCode, engRegion))

	// lr parameter: language restrict
	if params.Locale != "all" {
		queryParams.Set("lr", engLang)
	}

	// cr parameter: country restrict (only if region is specified)
	if engRegion != g.traits.AllLocale && strings.Contains(params.Locale, "-") {
		queryParams.Set("cr", "country"+engRegion)
	}

	queryParams.Set("ie", "utf8")
	queryParams.Set("oe", "utf8")
	queryParams.Set("filter", "0")
	queryParams.Set("start", fmt.Sprintf("%d", start))

	// Async arc request (key for bypassing rate limiting)
	queryParams.Set("asearch", "arc")
	queryParams.Set("async", asyncParam)

	// Time range
	if params.TimeRange != "" {
		if tr, ok := g.timeRangeMap[params.TimeRange]; ok {
			queryParams.Set("tbs", "qdr:"+tr)
		}
	}

	// Safe search
	if sf, ok := g.filterMap[params.SafeSearch]; ok {
		queryParams.Set("safe", sf)
	}

	params.URL = fmt.Sprintf("https://%s/search?%s", domain, queryParams.Encode())

	// Use GSA (Google Search App) user agent (like SearXNG)
	params.Headers.Set("User-Agent", getRandomGSAUserAgent())
	params.Headers.Set("Accept", "*/*")

	// CONSENT cookie to bypass cookie consent
	params.Cookies = append(params.Cookies, &http.Cookie{
		Name:  "CONSENT",
		Value: "YES+",
	})

	return nil
}

func (g *Google) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyStr := string(body)

	// Check for CAPTCHA/sorry page
	if strings.Contains(bodyStr, "sorry.google.com") || strings.Contains(bodyStr, "/sorry/") {
		return nil, fmt.Errorf("Google CAPTCHA detected")
	}

	results := NewEngineResults()

	doc, err := html.Parse(strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}

	// Parse results using SearXNG's approach
	// Look for div.MjjYud containers
	var findResults func(*html.Node)
	findResults = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			classes := getAttr(n, "class")

			// SearXNG uses MjjYud class for result containers
			if strings.Contains(classes, "MjjYud") {
				result := g.parseResultMjjYud(n)
				if result != nil && result.URL != "" && result.Title != "" {
					results.Add(*result)
				}
			} else if strings.Contains(classes, "g") && !strings.Contains(classes, "g-blk") {
				// Fallback to older class pattern
				result := g.parseResult(n)
				if result != nil && result.URL != "" && result.Title != "" {
					results.Add(*result)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findResults(c)
		}
	}
	findResults(doc)

	// Parse suggestions
	var findSuggestions func(*html.Node)
	findSuggestions = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			classes := getAttr(n, "class")
			if strings.Contains(classes, "ouy7Mc") {
				// Find suggestion links
				var findLinks func(*html.Node)
				findLinks = func(n *html.Node) {
					if n.Type == html.ElementNode && n.Data == "a" {
						text := extractText(n)
						if text != "" {
							results.AddSuggestion(text)
						}
					}
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						findLinks(c)
					}
				}
				findLinks(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findSuggestions(c)
		}
	}
	findSuggestions(doc)

	return results, nil
}

// parseResultMjjYud parses a result from MjjYud container (SearXNG approach)
func (g *Google) parseResultMjjYud(n *html.Node) *Result {
	result := &Result{}

	// Find title from div with role="link"
	var findTitle func(*html.Node) bool
	findTitle = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "div" {
			role := getAttr(n, "role")
			if role == "link" {
				result.Title = strings.TrimSpace(extractText(n))
				return true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findTitle(c) {
				return true
			}
		}
		return false
	}
	findTitle(n)

	// Find URL from first <a> tag
	var findURL func(*html.Node) bool
	findURL = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			if href != "" && !strings.HasPrefix(href, "#") {
				// Remove Google redirector: /url?q=...&sa=U
				if strings.HasPrefix(href, "/url?") {
					u, err := url.Parse(href)
					if err == nil {
						realURL := u.Query().Get("q")
						if realURL == "" {
							realURL = u.Query().Get("url")
						}
						if realURL != "" {
							// URL decode
							result.URL, _ = url.QueryUnescape(realURL)
							// Remove &sa=U suffix if present
							if idx := strings.Index(result.URL, "&sa=U"); idx > 0 {
								result.URL = result.URL[:idx]
							}
							return true
						}
					}
				} else if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
					result.URL = href
					return true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findURL(c) {
				return true
			}
		}
		return false
	}
	findURL(n)

	if result.URL == "" || result.Title == "" {
		return nil
	}

	// Filter out Google's internal URLs
	if strings.Contains(result.URL, "google.com") && !strings.Contains(result.URL, "translate.google") {
		return nil
	}

	// Find content from data-sncf="1" div
	var findContent func(*html.Node)
	findContent = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			dataSncf := getAttr(n, "data-sncf")
			if dataSncf == "1" {
				// Remove script tags before extracting text
				text := extractTextWithoutScripts(n)
				if text != "" && len(text) > len(result.Content) {
					result.Content = strings.TrimSpace(text)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findContent(c)
		}
	}
	findContent(n)

	result.ParsedURL, _ = url.Parse(result.URL)
	return result
}

func (g *Google) parseResult(n *html.Node) *Result {
	result := &Result{}

	// Find the first <a> tag with href
	var findLink func(*html.Node) bool
	findLink = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			if href != "" && !strings.HasPrefix(href, "#") && !strings.Contains(href, "google.com/search") {
				// Handle Google redirect URLs
				if strings.HasPrefix(href, "/url?") {
					u, err := url.Parse(href)
					if err == nil {
						realURL := u.Query().Get("q")
						if realURL == "" {
							realURL = u.Query().Get("url")
						}
						if realURL != "" {
							result.URL = realURL
						}
					}
				} else if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
					result.URL = href
				}

				if result.URL != "" {
					// Get title from h3 inside or nearby
					title := findH3Text(n)
					if title == "" {
						title = extractText(n)
					}
					result.Title = strings.TrimSpace(title)
					return true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findLink(c) {
				return true
			}
		}
		return false
	}
	findLink(n)

	if result.URL == "" {
		return nil
	}

	// Filter out Google's internal URLs
	if strings.Contains(result.URL, "google.com") && !strings.Contains(result.URL, "translate.google") {
		return nil
	}

	// Find content/snippet
	var findContent func(*html.Node)
	findContent = func(n *html.Node) {
		if n.Type == html.ElementNode {
			classes := getAttr(n, "class")
			dataSncf := getAttr(n, "data-sncf")
			// Various snippet class patterns Google uses
			if strings.Contains(classes, "VwiC3b") ||
				strings.Contains(classes, "IsZvec") ||
				strings.Contains(classes, "s3v9rd") ||
				dataSncf == "1" {
				text := extractText(n)
				if text != "" && len(text) > len(result.Content) {
					result.Content = strings.TrimSpace(text)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findContent(c)
		}
	}
	findContent(n)

	result.ParsedURL, _ = url.Parse(result.URL)
	return result
}

// extractTextWithoutScripts extracts text from node, skipping script tags
func extractTextWithoutScripts(n *html.Node) string {
	var sb strings.Builder
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			return // Skip script tags
		}
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return strings.TrimSpace(sb.String())
}

func findH3Text(n *html.Node) string {
	var result string
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h3" {
			result = extractText(n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(n)
	return result
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func extractText(n *html.Node) string {
	var sb strings.Builder
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return strings.TrimSpace(sb.String())
}

// GoogleImages implements Google image search using JSON API.
type GoogleImages struct {
	*BaseEngine
	parent *Google // Share arc_id with parent
}

// NewGoogleImages creates a new Google Images engine.
func NewGoogleImages() *GoogleImages {
	g := &GoogleImages{
		BaseEngine: NewBaseEngine("google images", "gi", []Category{CategoryImages}),
		parent:     NewGoogle(), // Create parent for arc_id sharing
	}

	g.SetPaging(true).
		SetMaxPage(50).
		SetSafeSearch(true).
		SetTimeout(10 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://images.google.com",
			WikidataID: "Q521550",
			Results:    "JSON",
		})

	return g
}

func (g *GoogleImages) Request(ctx context.Context, query string, params *RequestParams) error {
	// Use async JSON endpoint like SearXNG
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("tbm", "isch")
	queryParams.Set("asearch", "isch")
	queryParams.Set("hl", "en")
	queryParams.Set("safe", "off")

	// Zero-based pagination for images
	ijn := params.PageNo - 1
	queryParams.Set("async", fmt.Sprintf("_fmt:json,p:1,ijn:%d", ijn))

	params.URL = "https://www.google.com/search?" + queryParams.Encode()

	// Use GSA user agent
	params.Headers.Set("User-Agent", getRandomGSAUserAgent())
	params.Headers.Set("Accept", "*/*")

	params.Cookies = append(params.Cookies, &http.Cookie{
		Name:  "CONSENT",
		Value: "YES+",
	})

	return nil
}

func (g *GoogleImages) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()
	bodyStr := string(body)

	// Try to find JSON in the response
	// Google embeds JSON in various formats
	jsonStart := strings.Index(bodyStr, `{"ischj"`)
	if jsonStart == -1 {
		// Fallback: extract image URLs using regex
		re := regexp.MustCompile(`\["(https://[^"]+\.(?:jpg|jpeg|png|gif|webp)[^"]*)",(\d+),(\d+)\]`)
		matches := re.FindAllStringSubmatch(bodyStr, 50)

		for _, match := range matches {
			if len(match) >= 4 {
				imgURL := match[1]
				// Filter out small images (thumbnails)
				results.Add(Result{
					URL:        imgURL,
					ImageURL:   imgURL,
					Template:   "images",
					Resolution: fmt.Sprintf("%sx%s", match[2], match[3]),
				})
			}
		}
		return results, nil
	}

	// Find the end of JSON
	jsonEnd := strings.Index(bodyStr[jsonStart:], "\n")
	if jsonEnd == -1 {
		jsonEnd = len(bodyStr) - jsonStart
	}

	jsonStr := bodyStr[jsonStart : jsonStart+jsonEnd]

	// Parse JSON
	var data struct {
		Ischj struct {
			Metadata []struct {
				Result struct {
					ReferrerURL string `json:"referrer_url"`
					PageTitle   string `json:"page_title"`
					SiteTitle   string `json:"site_title"`
				} `json:"result"`
				OriginalImage struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"original_image"`
				Thumbnail struct {
					URL string `json:"url"`
				} `json:"thumbnail"`
			} `json:"metadata"`
		} `json:"ischj"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
		for _, item := range data.Ischj.Metadata {
			if item.OriginalImage.URL != "" {
				results.Add(Result{
					URL:          item.Result.ReferrerURL,
					Title:        item.Result.PageTitle,
					ImageURL:     item.OriginalImage.URL,
					ThumbnailURL: item.Thumbnail.URL,
					Source:       item.Result.SiteTitle,
					Template:     "images",
					Resolution:   fmt.Sprintf("%dx%d", item.OriginalImage.Width, item.OriginalImage.Height),
				})
			}
		}
	}

	return results, nil
}
