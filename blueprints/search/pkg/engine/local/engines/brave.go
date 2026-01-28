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

// Brave implements Brave web search.
type Brave struct {
	*BaseEngine
	timeRangeMap map[TimeRange]string
}

// NewBrave creates a new Brave engine.
func NewBrave() *Brave {
	b := &Brave{
		BaseEngine: NewBaseEngine("brave", "br", []Category{CategoryGeneral, CategoryWeb}),
		timeRangeMap: map[TimeRange]string{
			TimeRangeDay:   "pd",
			TimeRangeWeek:  "pw",
			TimeRangeMonth: "pm",
			TimeRangeYear:  "py",
		},
	}

	b.SetPaging(true).
		SetTimeRangeSupport(true).
		SetSafeSearch(true).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://search.brave.com",
			WikidataID: "Q107463951",
			Results:    "HTML",
		})

	return b
}

func (b *Brave) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("source", "web")

	// Pagination
	if params.PageNo > 1 {
		queryParams.Set("offset", fmt.Sprintf("%d", params.PageNo-1))
	}

	// Time range
	if params.TimeRange != "" {
		if tr, ok := b.timeRangeMap[params.TimeRange]; ok {
			queryParams.Set("tf", tr)
		}
	}

	params.URL = "https://search.brave.com/search?" + queryParams.Encode()
	params.Headers.Set("Accept", "text/html")
	params.Headers.Set("Accept-Language", "en-US,en;q=0.9")

	// Safe search cookie
	safeValue := "moderate"
	if params.SafeSearch == SafeSearchStrict {
		safeValue = "strict"
	} else if params.SafeSearch == SafeSearchOff {
		safeValue = "off"
	}
	params.Cookies = append(params.Cookies, &http.Cookie{Name: "safesearch", Value: safeValue})

	return nil
}

func (b *Brave) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// Find search results
	var findResults func(*html.Node)
	findResults = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "snippet") {
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

func (b *Brave) parseResult(n *html.Node) *Result {
	result := &Result{}

	var findTitle func(*html.Node) bool
	findTitle = func(n *html.Node) bool {
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
			if findTitle(c) {
				return true
			}
		}
		return false
	}
	findTitle(n)

	if result.URL == "" || result.Title == "" {
		return nil
	}

	// Find content
	var findContent func(*html.Node)
	findContent = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "content") {
					text := extractText(n)
					if len(text) > len(result.Content) {
						result.Content = text
					}
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
