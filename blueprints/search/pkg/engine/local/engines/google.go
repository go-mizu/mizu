package engines

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Google implements Google web search.
type Google struct {
	*BaseEngine
	timeRangeMap map[TimeRange]string
	filterMap    map[SafeSearchLevel]string
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
		SetTimeout(5 * time.Second).
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

func (g *Google) Request(ctx context.Context, query string, params *RequestParams) error {
	start := (params.PageNo - 1) * 10

	// Get language and region info
	engLang := g.traits.GetLanguage(params.Locale, "lang_en")
	engRegion := g.traits.GetRegion(params.Locale, "US")

	// Get domain
	domains := g.traits.Custom["supported_domains"].(map[string]string)
	domain := domains[engRegion]
	if domain == "" {
		domain = "www.google.com"
	}

	// Build URL
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("hl", strings.TrimPrefix(engLang, "lang_"))
	queryParams.Set("lr", engLang)
	if engRegion != g.traits.AllLocale {
		queryParams.Set("cr", "country"+engRegion)
	}
	queryParams.Set("ie", "utf8")
	queryParams.Set("oe", "utf8")
	queryParams.Set("filter", "0")

	if start > 0 {
		queryParams.Set("start", fmt.Sprintf("%d", start))
	}

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
	params.Headers.Set("Accept", "*/*")
	params.Headers.Set("Accept-Language", "en-US,en;q=0.9")

	return nil
}

func (g *Google) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// Find result divs
	var findResults func(*html.Node)
	findResults = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "g") {
					result := g.parseResult(n)
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

func (g *Google) parseResult(n *html.Node) *Result {
	result := &Result{}

	// Find link
	var findLink func(*html.Node) bool
	findLink = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.HasPrefix(attr.Val, "/url?") {
					// Parse Google redirect URL
					u, err := url.Parse(attr.Val)
					if err == nil {
						result.URL = u.Query().Get("q")
						if result.URL == "" {
							result.URL = u.Query().Get("url")
						}
					}
				} else if attr.Key == "href" && (strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://")) {
					result.URL = attr.Val
				}
			}
			if result.URL != "" {
				// Get title from link text
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
	findLink(n)

	if result.URL == "" || result.Title == "" {
		return nil
	}

	// Find content/snippet
	var findContent func(*html.Node)
	findContent = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && (strings.Contains(attr.Val, "VwiC3b") || strings.Contains(attr.Val, "IsZvec")) {
					result.Content = extractText(n)
					return
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

// GoogleImages implements Google image search.
type GoogleImages struct {
	*BaseEngine
}

// NewGoogleImages creates a new Google Images engine.
func NewGoogleImages() *GoogleImages {
	g := &GoogleImages{
		BaseEngine: NewBaseEngine("google images", "gi", []Category{CategoryImages}),
	}

	g.SetPaging(true).
		SetMaxPage(50).
		SetSafeSearch(true).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://images.google.com",
			WikidataID: "Q521550",
			Results:    "HTML",
		})

	return g
}

func (g *GoogleImages) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("tbm", "isch")
	queryParams.Set("hl", "en")

	if params.PageNo > 1 {
		queryParams.Set("start", fmt.Sprintf("%d", (params.PageNo-1)*20))
	}

	params.URL = "https://www.google.com/search?" + queryParams.Encode()
	params.Headers.Set("Accept", "text/html")

	return nil
}

func (g *GoogleImages) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	// Parse image results from data embedded in page
	// This is a simplified implementation
	text := string(body)

	// Find image URLs using regex (simplified approach)
	re := regexp.MustCompile(`\["(https://[^"]+\.(?:jpg|jpeg|png|gif|webp))",\d+,\d+\]`)
	matches := re.FindAllStringSubmatch(text, 20)

	for _, match := range matches {
		if len(match) >= 2 {
			imgURL := match[1]
			results.Add(Result{
				URL:      imgURL,
				ImageURL: imgURL,
				Template: "images",
			})
		}
	}

	return results, nil
}
