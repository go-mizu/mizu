package goodread

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseSearch parses a Goodreads search results page and returns discovered URLs.
func ParseSearch(doc *goquery.Document) []SearchResult {
	var results []SearchResult
	seen := map[string]bool{}

	// Books in search results
	doc.Find("tr.bookContainer, [itemtype*='Book'], [data-testid='bookTitle']").Each(func(_ int, sel *goquery.Selection) {
		var r SearchResult
		sel.Find("a[href*='/book/show/']").First().Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			if id := extractIDFromPath(href, "/book/show/"); id != "" {
				r.URL = BaseURL + "/book/show/" + id
				r.EntityType = "book"
				r.Title = strings.TrimSpace(a.Text())
			}
		})
		if r.URL != "" && !seen[r.URL] {
			seen[r.URL] = true
			results = append(results, r)
		}
	})

	// Authors in search results
	doc.Find("a[href*='/author/show/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if id := extractIDFromPath(href, "/author/show/"); id != "" {
			url := BaseURL + "/author/show/" + id
			if !seen[url] {
				seen[url] = true
				results = append(results, SearchResult{
					URL:        url,
					EntityType: "author",
					Title:      strings.TrimSpace(sel.Text()),
				})
			}
		}
	})

	return results
}

// ParseSearchNextPage returns the next page URL for search results, or "".
func ParseSearchNextPage(doc *goquery.Document) string {
	var next string
	doc.Find("a.next_page, a[href*='page='][rel='next']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if href != "" {
			if strings.HasPrefix(href, "/") {
				next = BaseURL + href
			} else {
				next = href
			}
		}
	})
	return next
}

// InferEntityType infers the entity type from a Goodreads URL path.
func InferEntityType(url string) string {
	switch {
	case strings.Contains(url, "/book/show/"):
		return "book"
	case strings.Contains(url, "/author/show/"):
		return "author"
	case strings.Contains(url, "/series/"):
		return "series"
	case strings.Contains(url, "/list/show/"):
		return "list"
	case strings.Contains(url, "/user/show/"):
		return "user"
	case strings.Contains(url, "/quotes/"):
		return "quote"
	case strings.Contains(url, "/genres/"):
		return "genre"
	default:
		return "book"
	}
}
