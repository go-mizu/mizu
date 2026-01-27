package engines

import (
	"context"
	"encoding/base64"
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
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:         "https://www.bing.com",
			WikidataID:      "Q182496",
			OfficialAPIDocs: "https://www.microsoft.com/en-us/bing/apis/bing-web-search-api",
			Results:         "HTML",
		})

	// Bing market codes
	b.traits.AllLocale = "clear"
	b.traits.Languages["en"] = "en-us"
	b.traits.Languages["de"] = "de-de"
	b.traits.Languages["fr"] = "fr-fr"
	b.traits.Languages["es"] = "es-es"
	b.traits.Languages["it"] = "it-it"
	b.traits.Languages["pt"] = "pt-pt"
	b.traits.Languages["ja"] = "ja-jp"
	b.traits.Languages["ko"] = "ko-kr"
	b.traits.Languages["zh"] = "zh-cn"
	b.traits.Languages["ru"] = "ru-ru"

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

	// Pagination
	if params.PageNo > 1 {
		first := (params.PageNo-1)*10 + 1
		queryParams.Set("first", fmt.Sprintf("%d", first))
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
				// Year needs special handling
				unixDay := int(time.Now().Unix() / 86400)
				queryParams.Set("filters", fmt.Sprintf("ex1:\"ez%s_%d_%d\"", tr, unixDay-365, unixDay))
			} else {
				queryParams.Set("filters", fmt.Sprintf("ex1:\"ez%s\"", tr))
			}
		}
	}

	params.URL = "https://www.bing.com/search?" + queryParams.Encode()
	params.AllowRedirects = true

	// Set cookies for language/region
	params.Cookies = append(params.Cookies,
		&http.Cookie{Name: "_EDGE_CD", Value: fmt.Sprintf("m=%s&u=%s", engRegion, engLang)},
		&http.Cookie{Name: "_EDGE_S", Value: fmt.Sprintf("mkt=%s&ui=%s", engRegion, engLang)},
	)

	return nil
}

func (b *Bing) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// Find result list
	var findResults func(*html.Node)
	findResults = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "li" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "b_algo") {
					result := b.parseResult(n)
					if result != nil {
						results.Add(*result)
					}
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
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href := attr.Val
					// Bing uses redirect URLs
					if strings.HasPrefix(href, "https://www.bing.com/ck/a?") {
						result.URL = b.decodeBingURL(href)
					} else if strings.HasPrefix(href, "http") {
						result.URL = href
					}
				}
			}
			if result.URL != "" {
				result.Title = extractText(n)
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

	// Find content in <p>
	var findContent func(*html.Node)
	findContent = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			text := extractText(n)
			if text != "" && len(text) > len(result.Content) {
				result.Content = text
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

	// Remove "a1" prefix
	if len(paramU) > 2 {
		encoded := paramU[2:]
		// Add padding if needed
		padding := 4 - (len(encoded) % 4)
		if padding < 4 {
			encoded += strings.Repeat("=", padding)
		}
		// Decode base64
		decoded, err := base64.URLEncoding.DecodeString(encoded)
		if err == nil {
			return string(decoded)
		}
	}

	return bingURL
}

// BingImages implements Bing image search.
type BingImages struct {
	*BaseEngine
}

// NewBingImages creates a new Bing Images engine.
func NewBingImages() *BingImages {
	b := &BingImages{
		BaseEngine: NewBaseEngine("bing images", "bi", []Category{CategoryImages}),
	}

	b.SetPaging(true).
		SetSafeSearch(true).
		SetTimeout(5 * time.Second).
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
	queryParams.Set("qft", "+filterui:photo-photo")

	if params.PageNo > 1 {
		queryParams.Set("first", fmt.Sprintf("%d", (params.PageNo-1)*35+1))
	}

	params.URL = "https://www.bing.com/images/search?" + queryParams.Encode()
	params.Headers.Set("Accept", "text/html")

	return nil
}

func (b *BingImages) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	// Parse image URLs from data attributes
	re := regexp.MustCompile(`murl&quot;:&quot;([^&]+)&quot;`)
	matches := re.FindAllStringSubmatch(string(body), 35)

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

	return results, nil
}

// BingNews implements Bing news search.
type BingNews struct {
	*BaseEngine
}

// NewBingNews creates a new Bing News engine.
func NewBingNews() *BingNews {
	b := &BingNews{
		BaseEngine: NewBaseEngine("bing news", "bn", []Category{CategoryNews}),
	}

	b.SetPaging(true).
		SetTimeRangeSupport(true).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://www.bing.com/news",
			WikidataID: "Q182496",
			Results:    "HTML",
		})

	return b
}

func (b *BingNews) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)

	if params.PageNo > 1 {
		queryParams.Set("first", fmt.Sprintf("%d", (params.PageNo-1)*10+1))
	}

	params.URL = "https://www.bing.com/news/search?" + queryParams.Encode()
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

	// Find news cards
	var findNews func(*html.Node)
	findNews = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "news-card") {
					result := b.parseNewsResult(n)
					if result != nil {
						result.Template = "news"
						results.Add(*result)
					}
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

	// Find link
	var findLink func(*html.Node) bool
	findLink = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.HasPrefix(attr.Val, "http") {
					result.URL = attr.Val
					result.Title = extractText(n)
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

	// Find snippet
	var findSnippet func(*html.Node)
	findSnippet = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "snippet") {
					result.Content = extractText(n)
					return
				}
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
