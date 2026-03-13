package ebay

import (
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseSearch parses an eBay search results page.
func ParseSearch(doc *goquery.Document, query string, page int, pageURL string) (SearchResult, []string, string, error) {
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if strings.Contains(strings.ToLower(title), "pardon our interruption") {
		return SearchResult{}, nil, "", fmt.Errorf("challenge page")
	}

	sr := SearchResult{
		SearchID:  searchID(query, page),
		Query:     query,
		Page:      page,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	seen := make(map[string]struct{})
	var itemURLs []string

	collectLink := func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		itemURL := NormalizeItemURL(absoluteURL(BaseURL, href))
		itemID := ExtractItemID(itemURL)
		if itemID == "" {
			return
		}
		if _, exists := seen[itemID]; exists {
			return
		}
		seen[itemID] = struct{}{}
		sr.ResultItemIDs = append(sr.ResultItemIDs, itemID)
		itemURLs = append(itemURLs, itemURL)
	}

	doc.Find("ul.srp-results a[href*=\"/itm/\"]").Each(collectLink)
	if len(sr.ResultItemIDs) == 0 {
		doc.Find("a.s-item__link[href*=\"/itm/\"], a[href*=\"/itm/\"]").Each(collectLink)
	}

	if sr.TotalResults == "" {
		sr.TotalResults = normalizeSpace(doc.Find("h1.srp-controls__count-heading span").First().Text())
	}
	if sr.TotalResults == "" {
		doc.Find(".srp-controls__count-heading span").Each(func(_ int, s *goquery.Selection) {
			text := normalizeSpace(s.Text())
			if text != "" && sr.TotalResults == "" {
				sr.TotalResults = text
			}
		})
	}

	nextPageURL := ""
	if href, ok := doc.Find("a.pagination__next").First().Attr("href"); ok {
		nextPageURL = absoluteURL(BaseURL, href)
	}
	sr.NextPageURL = nextPageURL

	if len(sr.ResultItemIDs) == 0 {
		return sr, nil, nextPageURL, fmt.Errorf("no item results found")
	}

	return sr, itemURLs, nextPageURL, nil
}
