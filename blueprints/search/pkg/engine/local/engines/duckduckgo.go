package engines

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// DuckDuckGo implements DuckDuckGo web search.
type DuckDuckGo struct {
	*BaseEngine
	timeRangeMap map[TimeRange]string
}

// NewDuckDuckGo creates a new DuckDuckGo engine.
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
		SetTimeout(5 * time.Second).
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

func (d *DuckDuckGo) Request(ctx context.Context, query string, params *RequestParams) error {
	engRegion := d.traits.GetRegion(params.Locale, d.traits.AllLocale)

	// DuckDuckGo lite HTML endpoint
	params.URL = "https://html.duckduckgo.com/html/"
	params.Method = "POST"

	params.Data = url.Values{}
	params.Data.Set("q", query)
	params.Data.Set("kl", engRegion)

	if params.TimeRange != "" {
		if tr, ok := d.timeRangeMap[params.TimeRange]; ok {
			params.Data.Set("df", tr)
		}
	}

	// For pagination
	if params.PageNo > 1 {
		offset := (params.PageNo - 1) * 25
		params.Data.Set("s", fmt.Sprintf("%d", offset))
		params.Data.Set("dc", fmt.Sprintf("%d", offset+1))
	}

	params.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	params.Headers.Set("Referer", "https://html.duckduckgo.com/")

	return nil
}

func (d *DuckDuckGo) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
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
				if attr.Key == "class" && strings.Contains(attr.Val, "web-result") {
					result := d.parseResult(n)
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

	// Find zero-click info (instant answers)
	var findZeroClick func(*html.Node)
	findZeroClick = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "zero_click_abstract" {
					text := extractText(n)
					if text != "" && !strings.Contains(text, "Your IP address") {
						results.AddAnswer(Answer{Answer: text})
					}
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

func (d *DuckDuckGo) parseResult(n *html.Node) *Result {
	result := &Result{}

	// Find link in h2 > a
	var findLink func(*html.Node) bool
	findLink = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					// DuckDuckGo uses direct URLs or uddg redirect
					href := attr.Val
					if strings.HasPrefix(href, "//duckduckgo.com/l/?uddg=") {
						// Parse redirect URL
						u, err := url.Parse("https:" + href)
						if err == nil {
							result.URL = u.Query().Get("uddg")
						}
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

	// Find snippet
	var findSnippet func(*html.Node)
	findSnippet = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "result__snippet") {
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

// DuckDuckGoImages implements DuckDuckGo image search.
type DuckDuckGoImages struct {
	*BaseEngine
}

// NewDuckDuckGoImages creates a new DuckDuckGo Images engine.
func NewDuckDuckGoImages() *DuckDuckGoImages {
	d := &DuckDuckGoImages{
		BaseEngine: NewBaseEngine("duckduckgo images", "ddi", []Category{CategoryImages}),
	}

	d.SetPaging(true).
		SetSafeSearch(true).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://duckduckgo.com/?iax=images",
			WikidataID: "Q12805",
			Results:    "JSON",
		})

	return d
}

func (d *DuckDuckGoImages) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("iax", "images")
	queryParams.Set("ia", "images")

	params.URL = "https://duckduckgo.com/?" + queryParams.Encode()
	params.Headers.Set("Accept", "text/html")

	return nil
}

func (d *DuckDuckGoImages) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	// DuckDuckGo images requires JavaScript to load
	// This is a placeholder - real implementation would need to handle API
	return NewEngineResults(), nil
}
