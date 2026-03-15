package amazon

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseSearch parses an Amazon search results page (/s?k=...).
// Returns the SearchResult for this page, a list of product page URLs for
// enqueuing, the next page URL (empty if last page), and any error.
func ParseSearch(doc *goquery.Document, query string, page int, pageURL string) (SearchResult, []string, string, error) {
	sr := SearchResult{
		SearchID:  searchID(query, page),
		Query:     query,
		Page:      page,
		FetchedAt: time.Now(),
	}

	// ResultASINs from search result cards
	doc.Find(`[data-component-type="s-search-result"][data-asin]`).Each(func(_ int, s *goquery.Selection) {
		asin, _ := s.Attr("data-asin")
		asin = strings.TrimSpace(asin)
		if len(asin) == 10 {
			sr.ResultASINs = append(sr.ResultASINs, asin)
		}
	})

	// TotalResults: first info-bar span containing "results"
	doc.Find(`[data-component-type="s-result-info-bar"] .a-size-base`).Each(func(_ int, s *goquery.Selection) {
		if sr.TotalResults != "" {
			return
		}
		t := strings.TrimSpace(s.Text())
		if strings.Contains(strings.ToLower(t), "result") {
			sr.TotalResults = t
		}
	})

	// Product URLs for enqueuing
	var productURLs []string
	for _, asin := range sr.ResultASINs {
		productURLs = append(productURLs, BaseURL+"/dp/"+asin)
	}

	// Next page URL
	nextPageURL := ""
	doc.Find(`.s-pagination-next`).Each(func(_ int, s *goquery.Selection) {
		if nextPageURL != "" {
			return
		}
		// disabled pagination item has class s-pagination-disabled
		if s.HasClass("s-pagination-disabled") {
			return
		}
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}
		nextPageURL = absoluteURL(BaseURL, href)
	})

	return sr, productURLs, nextPageURL, nil
}

// searchID returns an MD5-based hex ID for a query+page combination.
func searchID(query string, page int) string {
	h := md5.Sum([]byte(query + strconv.Itoa(page)))
	return fmt.Sprintf("%x", h)
}
