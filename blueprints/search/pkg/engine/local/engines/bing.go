package engines

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Bing implements Bing web search.
type Bing struct {
	*BaseEngine
	timeRangeMap map[TimeRange]string
}

// NewBing creates a new Bing engine.
func NewBing() *Bing {
	b := &Bing{
		BaseEngine: NewBaseEngine("bing", "b", []Category{CategoryGeneral, CategoryWeb}),
		timeRangeMap: map[TimeRange]string{
			TimeRangeDay:   "1",
			TimeRangeWeek:  "2",
			TimeRangeMonth: "3",
			TimeRangeYear:  "5",
		},
	}

	b.SetPaging(true).
		SetMaxPage(200).
		SetTimeRangeSupport(true).
		SetSafeSearch(true).
		SetTimeout(10 * time.Second).
		SetAbout(EngineAbout{
			Website:         "https://www.bing.com",
			WikidataID:      "Q182496",
			OfficialAPIDocs: "https://www.microsoft.com/en-us/bing/apis/bing-web-search-api",
			Results:         "HTML",
		})

	// Bing market codes
	b.traits.AllLocale = "clear"
	b.traits.Languages["en"] = "en"
	b.traits.Languages["de"] = "de"
	b.traits.Languages["fr"] = "fr"
	b.traits.Languages["es"] = "es"
	b.traits.Languages["it"] = "it"
	b.traits.Languages["pt"] = "pt"
	b.traits.Languages["ja"] = "ja"
	b.traits.Languages["ko"] = "ko"
	b.traits.Languages["zh"] = "zh"
	b.traits.Languages["ru"] = "ru"

	b.traits.Regions["en-US"] = "en-us"
	b.traits.Regions["en-GB"] = "en-gb"
	b.traits.Regions["de-DE"] = "de-de"
	b.traits.Regions["fr-FR"] = "fr-fr"
	b.traits.Regions["es-ES"] = "es-es"
	b.traits.Regions["it-IT"] = "it-it"
	b.traits.Regions["pt-BR"] = "pt-br"
	b.traits.Regions["ja-JP"] = "ja-jp"
	b.traits.Regions["ko-KR"] = "ko-kr"
	b.traits.Regions["zh-CN"] = "zh-cn"
	b.traits.Regions["ru-RU"] = "ru-ru"

	return b
}

func (b *Bing) Request(ctx context.Context, query string, params *RequestParams) error {
	engRegion := b.traits.GetRegion(params.Locale, "en-us")
	engLang := b.traits.GetLanguage(params.Locale, "en")

	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("pq", query)

	// Pagination - Bing uses 1-indexed 'first' parameter
	if params.PageNo > 1 {
		first := (params.PageNo-1)*10 + 1
		queryParams.Set("first", fmt.Sprintf("%d", first))
		// FORM parameter changes based on page number
		if params.PageNo == 2 {
			queryParams.Set("FORM", "PERE")
		} else {
			queryParams.Set("FORM", fmt.Sprintf("PERE%d", params.PageNo-2))
		}
	}

	// Time range
	if params.TimeRange != "" {
		if tr, ok := b.timeRangeMap[params.TimeRange]; ok {
			if params.TimeRange == TimeRangeYear {
				// Year needs special handling with Unix days
				unixDay := int(time.Now().Unix() / 86400)
				queryParams.Set("filters", fmt.Sprintf("ex1:\"ez%s_%d_%d\"", tr, unixDay-365, unixDay))
			} else {
				queryParams.Set("filters", fmt.Sprintf("ex1:\"ez%s\"", tr))
			}
		}
	}

	params.URL = "https://www.bing.com/search?" + queryParams.Encode()
	params.AllowRedirects = true

	// Set proper headers
	params.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	params.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	params.Headers.Set("Accept-Language", "en-US,en;q=0.9")
	params.Headers.Set("DNT", "1")
	params.Headers.Set("Upgrade-Insecure-Requests", "1")

	// Set cookies for language/region (critical for Bing)
	params.Cookies = append(params.Cookies,
		&http.Cookie{Name: "_EDGE_CD", Value: fmt.Sprintf("m=%s&u=%s", engRegion, engLang)},
		&http.Cookie{Name: "_EDGE_S", Value: fmt.Sprintf("mkt=%s&ui=%s", engRegion, engLang)},
		&http.Cookie{Name: "SRCHHPGUSR", Value: fmt.Sprintf("SRCHLANG=%s", engLang)},
	)

	return nil
}

func (b *Bing) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()
	bodyStr := string(body)

	doc, err := html.Parse(strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}

	// Find result list - Bing uses ol#b_results > li.b_algo
	var findResults func(*html.Node)
	findResults = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "li" {
			classes := getAttr(n, "class")
			if strings.Contains(classes, "b_algo") {
				result := b.parseResult(n)
				if result != nil {
					results.Add(*result)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findResults(c)
		}
	}
	findResults(doc)

	return results, nil
}

func (b *Bing) parseResult(n *html.Node) *Result {
	result := &Result{}

	// Find link in h2 > a
	var findLink func(*html.Node) bool
	findLink = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			if href != "" {
				// Bing uses redirect URLs with base64 encoded real URL
				if strings.HasPrefix(href, "https://www.bing.com/ck/a?") {
					result.URL = b.decodeBingURL(href)
				} else if strings.HasPrefix(href, "http") {
					result.URL = href
				}
			}
			if result.URL != "" {
				result.Title = strings.TrimSpace(extractText(n))
				return true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findLink(c) {
				return true
			}
		}
		return false
	}

	// Find h2 first
	var findH2 func(*html.Node) bool
	findH2 = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "h2" {
			return findLink(n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findH2(c) {
				return true
			}
		}
		return false
	}
	findH2(n)

	if result.URL == "" || result.Title == "" {
		return nil
	}

	// Find content in <p> or caption div
	var findContent func(*html.Node)
	findContent = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "p" || (n.Data == "div" && strings.Contains(getAttr(n, "class"), "b_caption")) {
				text := strings.TrimSpace(extractText(n))
				if text != "" && len(text) > len(result.Content) {
					result.Content = text
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

func (b *Bing) decodeBingURL(bingURL string) string {
	u, err := url.Parse(bingURL)
	if err != nil {
		return bingURL
	}

	// Get the 'u' parameter which contains the encoded URL
	paramU := u.Query().Get("u")
	if paramU == "" {
		return bingURL
	}

	// Remove "a1" prefix (base64 URL encoding marker)
	if len(paramU) > 2 && strings.HasPrefix(paramU, "a1") {
		encoded := paramU[2:]
		// URL-safe base64 uses - and _ instead of + and /
		encoded = strings.ReplaceAll(encoded, "-", "+")
		encoded = strings.ReplaceAll(encoded, "_", "/")
		// Add padding if needed
		padding := 4 - (len(encoded) % 4)
		if padding < 4 {
			encoded += strings.Repeat("=", padding)
		}
		// Decode base64
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil {
			return string(decoded)
		}
	}

	return bingURL
}

// BingImages implements Bing image search using async endpoint.
type BingImages struct {
	*BaseEngine
	timeRangeMap map[TimeRange]int
}

// NewBingImages creates a new Bing Images engine.
func NewBingImages() *BingImages {
	b := &BingImages{
		BaseEngine: NewBaseEngine("bing images", "bi", []Category{CategoryImages}),
		timeRangeMap: map[TimeRange]int{
			TimeRangeDay:   60 * 24,        // 1440 minutes
			TimeRangeWeek:  60 * 24 * 7,    // 10080 minutes
			TimeRangeMonth: 60 * 24 * 31,   // 44640 minutes
			TimeRangeYear:  60 * 24 * 365,  // 525600 minutes
		},
	}

	b.SetPaging(true).
		SetTimeRangeSupport(true).
		SetSafeSearch(true).
		SetTimeout(10 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://www.bing.com/images",
			WikidataID: "Q182496",
			Results:    "HTML",
		})

	return b
}

func (b *BingImages) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("async", "1")
	queryParams.Set("count", "35")

	// Pagination - 35 results per page
	first := 1
	if params.PageNo > 1 {
		first = (params.PageNo-1)*35 + 1
	}
	queryParams.Set("first", fmt.Sprintf("%d", first))

	// Time range filter
	if params.TimeRange != "" {
		if minutes, ok := b.timeRangeMap[params.TimeRange]; ok {
			queryParams.Set("qft", fmt.Sprintf("filterui:age-lt%d", minutes))
		}
	}

	params.URL = "https://www.bing.com/images/async?" + queryParams.Encode()

	params.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	params.Headers.Set("Accept", "text/html")

	return nil
}

func (b *BingImages) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()
	bodyStr := string(body)

	// Parse image metadata from JSON in HTML attributes
	// Bing embeds image data in the 'm' attribute of anchor tags
	re := regexp.MustCompile(`class="iusc"[^>]*m="([^"]+)"`)
	matches := re.FindAllStringSubmatch(bodyStr, 50)

	for _, match := range matches {
		if len(match) >= 2 {
			// Decode HTML entities
			jsonStr := strings.ReplaceAll(match[1], "&quot;", `"`)
			jsonStr = strings.ReplaceAll(jsonStr, "&amp;", "&")

			var metadata struct {
				Purl string `json:"purl"` // Page URL
				Murl string `json:"murl"` // Media/image URL
				Turl string `json:"turl"` // Thumbnail URL
				Desc string `json:"desc"` // Description
				T    string `json:"t"`    // Title
			}

			if err := json.Unmarshal([]byte(jsonStr), &metadata); err == nil {
				if metadata.Murl != "" {
					results.Add(Result{
						URL:          metadata.Purl,
						Title:        metadata.T,
						ImageURL:     metadata.Murl,
						ThumbnailURL: metadata.Turl,
						Content:      metadata.Desc,
						Template:     "images",
					})
				}
			}
		}
	}

	// Fallback: extract image URLs using simpler regex
	if len(results.Results) == 0 {
		re = regexp.MustCompile(`murl&quot;:&quot;([^&]+)&quot;`)
		matches = re.FindAllStringSubmatch(bodyStr, 35)

		for _, match := range matches {
			if len(match) >= 2 {
				imgURL, _ := url.QueryUnescape(match[1])
				if imgURL != "" {
					results.Add(Result{
						URL:      imgURL,
						ImageURL: imgURL,
						Template: "images",
					})
				}
			}
		}
	}

	return results, nil
}

// BingNews implements Bing news search using infinite scroll endpoint.
type BingNews struct {
	*BaseEngine
	timeRangeMap map[TimeRange]string
}

// NewBingNews creates a new Bing News engine.
func NewBingNews() *BingNews {
	b := &BingNews{
		BaseEngine: NewBaseEngine("bing news", "bn", []Category{CategoryNews}),
		timeRangeMap: map[TimeRange]string{
			TimeRangeDay:   "4", // Last 24 hours
			TimeRangeWeek:  "7", // Last week
			TimeRangeMonth: "9", // Last month
		},
	}

	b.SetPaging(true).
		SetTimeRangeSupport(true).
		SetTimeout(10 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://www.bing.com/news",
			WikidataID: "Q182496",
			Results:    "HTML",
		})

	return b
}

func (b *BingNews) Request(ctx context.Context, query string, params *RequestParams) error {
	// Extract language and country from locale
	lang := "en"
	country := "us"
	if params.Locale != "" {
		parts := strings.Split(strings.ToLower(params.Locale), "-")
		if len(parts) >= 1 {
			lang = parts[0]
		}
		if len(parts) >= 2 {
			country = parts[1]
		}
	}

	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("InfiniteScroll", "1")
	queryParams.Set("form", "PTFTNR")
	queryParams.Set("setlang", lang)
	queryParams.Set("cc", country)

	// Pagination
	first := 1
	sfx := 0
	if params.PageNo > 1 {
		first = (params.PageNo-1)*10 + 1
		sfx = params.PageNo - 1
	}
	queryParams.Set("first", fmt.Sprintf("%d", first))
	queryParams.Set("SFX", fmt.Sprintf("%d", sfx))

	// Time range
	if params.TimeRange != "" {
		if interval, ok := b.timeRangeMap[params.TimeRange]; ok {
			queryParams.Set("qft", fmt.Sprintf("interval=\"%s\"", interval))
		}
	}

	params.URL = "https://www.bing.com/news/infinitescrollajax?" + queryParams.Encode()

	params.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	params.Headers.Set("Accept", "text/html")

	return nil
}

func (b *BingNews) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// Find news items
	var findNews func(*html.Node)
	findNews = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			classes := getAttr(n, "class")
			if strings.Contains(classes, "newsitem") || strings.Contains(classes, "news-card") {
				result := b.parseNewsResult(n)
				if result != nil {
					result.Template = "news"
					results.Add(*result)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findNews(c)
		}
	}
	findNews(doc)

	return results, nil
}

func (b *BingNews) parseNewsResult(n *html.Node) *Result {
	result := &Result{}

	// Find title link
	var findLink func(*html.Node) bool
	findLink = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			classes := getAttr(n, "class")
			href := getAttr(n, "href")
			if (strings.Contains(classes, "title") || classes == "") && strings.HasPrefix(href, "http") {
				result.URL = href
				result.Title = strings.TrimSpace(extractText(n))
				return result.Title != ""
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

	// Find snippet
	var findSnippet func(*html.Node)
	findSnippet = func(n *html.Node) {
		if n.Type == html.ElementNode {
			classes := getAttr(n, "class")
			if strings.Contains(classes, "snippet") || strings.Contains(classes, "summary") {
				text := strings.TrimSpace(extractText(n))
				if text != "" && len(text) > len(result.Content) {
					result.Content = text
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findSnippet(c)
		}
	}
	findSnippet(n)

	// Find thumbnail
	var findThumbnail func(*html.Node)
	findThumbnail = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			src := getAttr(n, "src")
			if src != "" && !strings.Contains(src, "data:image") {
				if !strings.HasPrefix(src, "http") {
					src = "https://www.bing.com" + src
				}
				result.ThumbnailURL = src
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findThumbnail(c)
		}
	}
	findThumbnail(n)

	// Find source
	var findSource func(*html.Node)
	findSource = func(n *html.Node) {
		if n.Type == html.ElementNode {
			classes := getAttr(n, "class")
			ariaLabel := getAttr(n, "aria-label")
			if strings.Contains(classes, "source") || ariaLabel != "" {
				text := strings.TrimSpace(extractText(n))
				if text != "" && result.Source == "" {
					result.Source = text
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findSource(c)
		}
	}
	findSource(n)

	result.ParsedURL, _ = url.Parse(result.URL)
	return result
}
