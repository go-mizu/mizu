package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// vqdCache stores VQD tokens for queries (like SearXNG's EngineCache)
var (
	vqdCache   = make(map[string]vqdEntry)
	vqdCacheMu sync.RWMutex
)

type vqdEntry struct {
	value   string
	expires time.Time
}

// DuckDuckGo implements DuckDuckGo web search using HTML endpoint.
// NOTE: DuckDuckGo HTML web search has aggressive CAPTCHA protection.
// The JSON APIs (images, news, videos) work without CAPTCHA - use those instead.
// SearXNG handles this by using proxy rotation and session management.
type DuckDuckGo struct {
	*BaseEngine
	timeRangeMap map[TimeRange]string
}

// NewDuckDuckGo creates a new DuckDuckGo engine.
// WARNING: This engine is disabled due to CAPTCHA protection.
// Use DuckDuckGoImages, DuckDuckGoNews, or DuckDuckGoVideos instead.
func NewDuckDuckGo() *DuckDuckGo {
	d := &DuckDuckGo{
		BaseEngine: NewBaseEngine("duckduckgo", "ddg", []Category{CategoryGeneral, CategoryWeb}),
		timeRangeMap: map[TimeRange]string{
			TimeRangeDay:   "d",
			TimeRangeWeek:  "w",
			TimeRangeMonth: "m",
			TimeRangeYear:  "y",
		},
	}

	d.SetPaging(true).
		SetTimeRangeSupport(true).
		SetSafeSearch(true).
		SetTimeout(10 * time.Second).
		SetDisabled(true). // CAPTCHA protection - use JSON APIs (images/news/videos) instead
		SetAbout(EngineAbout{
			Website:    "https://duckduckgo.com",
			WikidataID: "Q12805",
			Results:    "HTML",
		})

	// DuckDuckGo uses region codes
	d.traits.AllLocale = "wt-wt"
	d.traits.Regions["en-US"] = "us-en"
	d.traits.Regions["en-GB"] = "uk-en"
	d.traits.Regions["de-DE"] = "de-de"
	d.traits.Regions["fr-FR"] = "fr-fr"
	d.traits.Regions["es-ES"] = "es-es"
	d.traits.Regions["it-IT"] = "it-it"
	d.traits.Regions["ja-JP"] = "jp-jp"
	d.traits.Regions["ko-KR"] = "kr-kr"
	d.traits.Regions["zh-CN"] = "cn-zh"
	d.traits.Regions["ru-RU"] = "ru-ru"

	return d
}

// getVQD fetches or returns cached VQD token for a query (like SearXNG)
// VQD is required for DDG's bot protection
func getVQD(query, region string, forceRequest bool) (string, error) {
	cacheKey := fmt.Sprintf("%s//%s", query, region)

	// Check cache first (unless forcing refresh)
	if !forceRequest {
		vqdCacheMu.RLock()
		if entry, ok := vqdCache[cacheKey]; ok && time.Now().Before(entry.expires) {
			vqdCacheMu.RUnlock()
			return entry.value, nil
		}
		vqdCacheMu.RUnlock()
	}

	// Fetch VQD from DuckDuckGo (like SearXNG: https://duckduckgo.com/?q={query})
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Don't follow redirects to avoid CAPTCHA redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	reqURL := fmt.Sprintf("https://duckduckgo.com/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	// Browser headers (simpler set that works - Go handles gzip automatically)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DDG VQD request failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract VQD using regex (like SearXNG's extr function: vqd="...")
	vqdRe := regexp.MustCompile(`vqd="([^"]+)"`)
	matches := vqdRe.FindSubmatch(body)
	if len(matches) < 2 {
		// Try single quote pattern
		vqdRe2 := regexp.MustCompile(`vqd='([^']+)'`)
		matches = vqdRe2.FindSubmatch(body)
		if len(matches) < 2 {
			return "", fmt.Errorf("could not extract VQD from DDG response")
		}
	}

	vqd := string(matches[1])

	// Cache the VQD for 1 hour (like SearXNG: expire=3600)
	vqdCacheMu.Lock()
	vqdCache[cacheKey] = vqdEntry{
		value:   vqd,
		expires: time.Now().Add(time.Hour),
	}
	vqdCacheMu.Unlock()

	return vqd, nil
}

// setVQD stores a VQD token in the cache (from response)
func setVQD(query, region, vqd string) {
	cacheKey := fmt.Sprintf("%s//%s", query, region)
	vqdCacheMu.Lock()
	vqdCache[cacheKey] = vqdEntry{
		value:   vqd,
		expires: time.Now().Add(time.Hour),
	}
	vqdCacheMu.Unlock()
}

func (d *DuckDuckGo) Request(ctx context.Context, query string, params *RequestParams) error {
	// Limit query length (DDG does not accept queries with more than 499 chars)
	if len(query) >= 500 {
		return fmt.Errorf("DDG does not accept queries with more than 499 characters")
	}

	engRegion := d.traits.GetRegion(params.Locale, d.traits.AllLocale)

	// DuckDuckGo lite HTML endpoint (like SearXNG: https://html.duckduckgo.com/html/)
	params.URL = "https://html.duckduckgo.com/html/"
	params.Method = "POST"

	// Build form data (matching SearXNG order: q, b, s, nextParams, v, o, dc, api, vqd, kl, df)
	params.Data = url.Values{}
	params.Data.Set("q", query)

	if params.PageNo == 1 {
		// First page: only need q parameter with empty b (like SearXNG)
		params.Data.Set("b", "")
	} else {
		// Pagination (page 2+): Page 2 = offset 10, Page 3+ = 10 + (n-2)*15 (like SearXNG)
		offset := 10 + (params.PageNo-2)*15
		params.Data.Set("s", fmt.Sprintf("%d", offset))
		params.Data.Set("nextParams", "")
		params.Data.Set("v", "l")
		params.Data.Set("o", "json")
		params.Data.Set("dc", fmt.Sprintf("%d", offset+1))
		params.Data.Set("api", "d.js")

		// VQD is required for page 2+ (DDG bot protection)
		vqd, err := getVQD(query, engRegion, false)
		if err != nil {
			// Without VQD, don't make the request to avoid IP blocking
			return fmt.Errorf("DDG pagination requires VQD token: %w", err)
		}
		params.Data.Set("vqd", vqd)
	}

	// Put empty kl in form data if language/region set to all
	if engRegion == "wt-wt" {
		params.Data.Set("kl", "")
	} else {
		params.Data.Set("kl", engRegion)
	}

	// Time range filter (df is always set, empty string if no time range)
	params.Data.Set("df", "")
	if params.TimeRange != "" {
		if tr, ok := d.timeRangeMap[params.TimeRange]; ok {
			params.Data.Set("df", tr)
		}
	}

	// Required headers for DuckDuckGo bot detection (matching SearXNG exactly)
	params.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	params.Headers.Set("Referer", "https://html.duckduckgo.com/html/")
	params.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	params.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")

	// Accept-Language: SearXNG uses send_accept_language_header = True
	// DuckDuckGo-Lite tries to guess user's preferred language from Accept-Language
	params.Headers.Set("Accept-Language", "en-US,en;q=0.9")

	// Critical Sec-Fetch headers for DDG bot detection (SearXNG notes: at least Mode is used)
	params.Headers.Set("Sec-Fetch-Dest", "document")
	params.Headers.Set("Sec-Fetch-Mode", "navigate")
	params.Headers.Set("Sec-Fetch-Site", "same-origin")
	params.Headers.Set("Sec-Fetch-User", "?1")
	params.Headers.Set("Upgrade-Insecure-Requests", "1")

	// Cookies for region (like SearXNG: params['cookies']['kl'] = eng_region)
	params.Cookies = append(params.Cookies, &http.Cookie{
		Name:  "kl",
		Value: engRegion,
	})

	// Add df cookie if time range is set (like SearXNG)
	if params.TimeRange != "" {
		if tr, ok := d.timeRangeMap[params.TimeRange]; ok {
			params.Cookies = append(params.Cookies, &http.Cookie{
				Name:  "df",
				Value: tr,
			})
		}
	}

	return nil
}

func (d *DuckDuckGo) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyStr := string(body)
	results := NewEngineResults()

	// Check for CAPTCHA
	if strings.Contains(bodyStr, "challenge-form") || strings.Contains(bodyStr, "Unfortunately, bots") {
		return nil, fmt.Errorf("DuckDuckGo CAPTCHA detected")
	}

	doc, err := html.Parse(strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}

	// Extract VQD from response for caching (like SearXNG)
	engRegion := d.traits.GetRegion(params.Locale, d.traits.AllLocale)
	d.extractAndCacheVQD(doc, params.Data.Get("q"), engRegion)

	// Find result divs (like SearXNG: //div[@id="links"]/div[contains(@class, "web-result")])
	var findResults func(*html.Node)
	findResults = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			id := getAttr(n, "id")
			if id == "links" {
				// Found the links container, now find web-result divs
				d.parseLinksContainer(n, results)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findResults(c)
		}
	}
	findResults(doc)

	// Find zero-click info (instant answers)
	var findZeroClick func(*html.Node)
	findZeroClick = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			id := getAttr(n, "id")
			if id == "zero_click_abstract" {
				text := extractText(n)
				// Filter out IP/user agent info
				if text != "" &&
					!strings.Contains(text, "Your IP address") &&
					!strings.Contains(text, "Your user agent") &&
					!strings.Contains(text, "URL Decoded") {
					results.AddAnswer(Answer{Answer: text})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findZeroClick(c)
		}
	}
	findZeroClick(doc)

	return results, nil
}

// extractAndCacheVQD extracts VQD from the response form and caches it
func (d *DuckDuckGo) extractAndCacheVQD(doc *html.Node, query, region string) {
	var findVQD func(*html.Node)
	findVQD = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			name := getAttr(n, "name")
			if name == "vqd" {
				value := getAttr(n, "value")
				if value != "" {
					setVQD(query, region, value)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findVQD(c)
		}
	}
	findVQD(doc)
}

// parseLinksContainer parses results from the #links div
func (d *DuckDuckGo) parseLinksContainer(n *html.Node, results *EngineResults) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			classes := getAttr(c, "class")
			// Look for web-result class but exclude ads
			if strings.Contains(classes, "web-result") && !strings.Contains(classes, "result--ad") {
				result := d.parseResult(c)
				if result != nil {
					results.Add(*result)
				}
			}
		}
	}
}

func (d *DuckDuckGo) parseResult(n *html.Node) *Result {
	result := &Result{}

	// Find link in h2 > a (like SearXNG: .//h2/a)
	var findH2Link func(*html.Node) bool
	findH2Link = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "h2" {
			// Find <a> inside h2
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "a" {
					href := getAttr(c, "href")
					if href != "" {
						// DuckDuckGo uses direct URLs or uddg redirect
						if strings.HasPrefix(href, "//duckduckgo.com/l/?uddg=") {
							// Parse redirect URL
							u, err := url.Parse("https:" + href)
							if err == nil {
								uddg := u.Query().Get("uddg")
								if uddg != "" {
									result.URL, _ = url.QueryUnescape(uddg)
								}
							}
						} else if strings.HasPrefix(href, "http") {
							result.URL = href
						}

						if result.URL != "" {
							result.Title = strings.TrimSpace(extractText(c))
							return true
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findH2Link(c) {
				return true
			}
		}
		return false
	}
	findH2Link(n)

	if result.URL == "" || result.Title == "" {
		return nil
	}

	// Find snippet (like SearXNG: .//a[contains(@class, "result__snippet")])
	var findSnippet func(*html.Node)
	findSnippet = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			classes := getAttr(n, "class")
			if strings.Contains(classes, "result__snippet") {
				result.Content = strings.TrimSpace(extractText(n))
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findSnippet(c)
		}
	}
	findSnippet(n)

	result.ParsedURL, _ = url.Parse(result.URL)
	return result
}

// DuckDuckGoImages implements DuckDuckGo image search using JSON API.
// Unlike the HTML web search, the JSON APIs for images/news/videos work without CAPTCHA.
type DuckDuckGoImages struct {
	*BaseEngine
}

// NewDuckDuckGoImages creates a new DuckDuckGo Images engine.
// Uses the JSON API (i.js) which works without CAPTCHA.
func NewDuckDuckGoImages() *DuckDuckGoImages {
	d := &DuckDuckGoImages{
		BaseEngine: NewBaseEngine("duckduckgo images", "ddi", []Category{CategoryImages}),
	}

	d.SetPaging(true).
		SetSafeSearch(true).
		SetTimeout(10 * time.Second).
		SetDisabled(false). // JSON API works!
		SetAbout(EngineAbout{
			Website:    "https://duckduckgo.com/?iax=images",
			WikidataID: "Q12805",
			Results:    "JSON",
		})

	// DuckDuckGo uses region codes
	d.traits.AllLocale = "wt-wt"
	d.traits.Regions["en-US"] = "us-en"
	d.traits.Regions["en-GB"] = "uk-en"
	d.traits.Regions["de-DE"] = "de-de"
	d.traits.Regions["fr-FR"] = "fr-fr"

	return d
}

func (d *DuckDuckGoImages) Request(ctx context.Context, query string, params *RequestParams) error {
	engRegion := d.traits.GetRegion(params.Locale, d.traits.AllLocale)

	// Get VQD token first (required by DDG JSON APIs)
	// SearXNG uses force_request=True for extra APIs
	vqd, err := getVQD(query, engRegion, true)
	if err != nil {
		return fmt.Errorf("DDG Images requires VQD token: %w", err)
	}

	// Build image search URL (like SearXNG duckduckgo_extra.py)
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("o", "json")
	queryParams.Set("l", engRegion)
	queryParams.Set("f", ",,,,,")
	queryParams.Set("vqd", vqd)

	// Pagination
	if params.PageNo > 1 {
		queryParams.Set("s", fmt.Sprintf("%d", (params.PageNo-1)*100))
	}

	// SafeSearch
	switch params.SafeSearch {
	case SafeSearchOff:
		queryParams.Set("p", "-1")
	case SafeSearchStrict:
		queryParams.Set("p", "1")
	}

	params.URL = "https://duckduckgo.com/i.js?" + queryParams.Encode()

	// Required headers to prevent rate limiting (from SearXNG)
	params.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	params.Headers.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	params.Headers.Set("Referer", "https://duckduckgo.com/")
	params.Headers.Set("X-Requested-With", "XMLHttpRequest")

	// Cookies for language/region (like SearXNG)
	params.Cookies = append(params.Cookies,
		&http.Cookie{Name: "l", Value: engRegion},
		&http.Cookie{Name: "ah", Value: engRegion},
	)

	return nil
}

func (d *DuckDuckGoImages) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	// Parse JSON response
	type ddgImageResult struct {
		Image     string `json:"image"`
		Thumbnail string `json:"thumbnail"`
		Title     string `json:"title"`
		URL       string `json:"url"`
		Source    string `json:"source"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}

	type ddgImagesResponse struct {
		Results []ddgImageResult `json:"results"`
		Next    string           `json:"next"`
	}

	var data ddgImagesResponse
	bodyStr := string(body)

	// Find JSON start (DDG sometimes wraps response)
	jsonStart := strings.Index(bodyStr, "{")
	if jsonStart == -1 {
		return results, nil
	}

	if err := json.Unmarshal([]byte(bodyStr[jsonStart:]), &data); err != nil {
		return results, nil
	}

	for _, item := range data.Results {
		if item.Image != "" {
			result := Result{
				URL:          item.URL,
				Title:        item.Title,
				ImageURL:     item.Image,
				ThumbnailURL: item.Thumbnail,
				Source:       item.Source,
				Template:     "images",
				Resolution:   fmt.Sprintf("%dx%d", item.Width, item.Height),
			}
			result.ParsedURL, _ = url.Parse(result.URL)
			results.Add(result)
		}
	}

	return results, nil
}

// DuckDuckGoNews implements DuckDuckGo news search using JSON API.
type DuckDuckGoNews struct {
	*BaseEngine
}

// NewDuckDuckGoNews creates a new DuckDuckGo News engine.
func NewDuckDuckGoNews() *DuckDuckGoNews {
	d := &DuckDuckGoNews{
		BaseEngine: NewBaseEngine("duckduckgo news", "ddn", []Category{CategoryNews}),
	}

	d.SetPaging(true).
		SetTimeout(10 * time.Second).
		SetDisabled(false). // JSON API works!
		SetAbout(EngineAbout{
			Website:    "https://duckduckgo.com/?iar=news",
			WikidataID: "Q12805",
			Results:    "JSON",
		})

	d.traits.AllLocale = "wt-wt"
	d.traits.Regions["en-US"] = "us-en"
	d.traits.Regions["en-GB"] = "uk-en"
	d.traits.Regions["de-DE"] = "de-de"
	d.traits.Regions["fr-FR"] = "fr-fr"

	return d
}

func (d *DuckDuckGoNews) Request(ctx context.Context, query string, params *RequestParams) error {
	engRegion := d.traits.GetRegion(params.Locale, d.traits.AllLocale)

	vqd, err := getVQD(query, engRegion, true)
	if err != nil {
		return fmt.Errorf("DDG News requires VQD token: %w", err)
	}

	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("o", "json")
	queryParams.Set("l", engRegion)
	queryParams.Set("f", ",,,,,")
	queryParams.Set("vqd", vqd)

	if params.PageNo > 1 {
		queryParams.Set("s", fmt.Sprintf("%d", (params.PageNo-1)*30))
	}

	params.URL = "https://duckduckgo.com/news.js?" + queryParams.Encode()

	params.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	params.Headers.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	params.Headers.Set("Referer", "https://duckduckgo.com/")
	params.Headers.Set("X-Requested-With", "XMLHttpRequest")

	params.Cookies = append(params.Cookies,
		&http.Cookie{Name: "l", Value: engRegion},
	)

	return nil
}

func (d *DuckDuckGoNews) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	type ddgNewsResult struct {
		URL          string `json:"url"`
		Title        string `json:"title"`
		Excerpt      string `json:"excerpt"`
		Source       string `json:"source"`
		Image        string `json:"image"`
		Date         int64  `json:"date"`
		RelativeTime string `json:"relative_time"`
	}

	type ddgNewsResponse struct {
		Results []ddgNewsResult `json:"results"`
		Next    string          `json:"next"`
	}

	var data ddgNewsResponse
	bodyStr := string(body)

	jsonStart := strings.Index(bodyStr, "{")
	if jsonStart == -1 {
		return results, nil
	}

	if err := json.Unmarshal([]byte(bodyStr[jsonStart:]), &data); err != nil {
		return results, nil
	}

	for _, item := range data.Results {
		result := Result{
			URL:          item.URL,
			Title:        item.Title,
			Content:      item.Excerpt,
			Source:       item.Source,
			ThumbnailURL: item.Image,
			Template:     "news",
		}
		if item.Date > 0 {
			result.PublishedAt = time.Unix(item.Date, 0)
		}
		result.ParsedURL, _ = url.Parse(result.URL)
		results.Add(result)
	}

	return results, nil
}

// DuckDuckGoVideos implements DuckDuckGo video search using JSON API.
type DuckDuckGoVideos struct {
	*BaseEngine
}

// NewDuckDuckGoVideos creates a new DuckDuckGo Videos engine.
func NewDuckDuckGoVideos() *DuckDuckGoVideos {
	d := &DuckDuckGoVideos{
		BaseEngine: NewBaseEngine("duckduckgo videos", "ddv", []Category{CategoryVideos}),
	}

	d.SetPaging(true).
		SetSafeSearch(true).
		SetTimeout(10 * time.Second).
		SetDisabled(false). // JSON API works!
		SetAbout(EngineAbout{
			Website:    "https://duckduckgo.com/?iax=videos",
			WikidataID: "Q12805",
			Results:    "JSON",
		})

	d.traits.AllLocale = "wt-wt"
	d.traits.Regions["en-US"] = "us-en"
	d.traits.Regions["en-GB"] = "uk-en"
	d.traits.Regions["de-DE"] = "de-de"
	d.traits.Regions["fr-FR"] = "fr-fr"

	return d
}

func (d *DuckDuckGoVideos) Request(ctx context.Context, query string, params *RequestParams) error {
	engRegion := d.traits.GetRegion(params.Locale, d.traits.AllLocale)

	vqd, err := getVQD(query, engRegion, true)
	if err != nil {
		return fmt.Errorf("DDG Videos requires VQD token: %w", err)
	}

	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("o", "json")
	queryParams.Set("l", engRegion)
	queryParams.Set("f", ",,,,,")
	queryParams.Set("vqd", vqd)

	if params.PageNo > 1 {
		queryParams.Set("s", fmt.Sprintf("%d", (params.PageNo-1)*60))
	}

	switch params.SafeSearch {
	case SafeSearchOff:
		queryParams.Set("p", "-1")
	case SafeSearchStrict:
		queryParams.Set("p", "1")
	}

	params.URL = "https://duckduckgo.com/v.js?" + queryParams.Encode()

	params.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	params.Headers.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	params.Headers.Set("Referer", "https://duckduckgo.com/")
	params.Headers.Set("X-Requested-With", "XMLHttpRequest")

	params.Cookies = append(params.Cookies,
		&http.Cookie{Name: "l", Value: engRegion},
	)

	return nil
}

func (d *DuckDuckGoVideos) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	type ddgVideoImages struct {
		Small  string `json:"small"`
		Medium string `json:"medium"`
		Large  string `json:"large"`
	}

	type ddgVideoResult struct {
		Content     string         `json:"content"` // URL
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Duration    string         `json:"duration"`
		Provider    string         `json:"provider"`
		Uploader    string         `json:"uploader"`
		Published   string         `json:"published"`
		Images      ddgVideoImages `json:"images"`
		EmbedURL    string         `json:"embed_url"`
	}

	type ddgVideosResponse struct {
		Results []ddgVideoResult `json:"results"`
		Next    string           `json:"next"`
	}

	var data ddgVideosResponse
	bodyStr := string(body)

	jsonStart := strings.Index(bodyStr, "{")
	if jsonStart == -1 {
		return results, nil
	}

	if err := json.Unmarshal([]byte(bodyStr[jsonStart:]), &data); err != nil {
		return results, nil
	}

	for _, item := range data.Results {
		thumbnail := item.Images.Medium
		if thumbnail == "" {
			thumbnail = item.Images.Small
		}
		if thumbnail == "" {
			thumbnail = item.Images.Large
		}

		content := item.Description
		if item.Uploader != "" && content != "" {
			content = fmt.Sprintf("by %s - %s", item.Uploader, content)
		} else if item.Uploader != "" {
			content = "by " + item.Uploader
		}

		result := Result{
			URL:          item.Content,
			Title:        item.Title,
			Content:      content,
			Duration:     item.Duration,
			Source:       item.Provider,
			ThumbnailURL: thumbnail,
			EmbedURL:     item.EmbedURL,
			Template:     "videos",
		}
		result.ParsedURL, _ = url.Parse(result.URL)
		results.Add(result)
	}

	return results, nil
}
