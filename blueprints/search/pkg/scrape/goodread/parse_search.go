package goodread

import (
	"encoding/json"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseSearchAutocomplete parses the Goodreads autocomplete JSON response.
// Endpoint: GET /book/auto_complete?format=json&q=<query>
//
// Note: this endpoint returns up to ~20 results and does not support pagination.
// For more results, provide a more specific query.
func ParseSearchAutocomplete(body []byte) []SearchResult {
	var items []struct {
		BookID  string `json:"bookId"`
		BookURL string `json:"bookUrl"`
		Title   string `json:"title"`
		Author  struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			ProfileURL string `json:"profileUrl"`
		} `json:"author"`
	}
	if err := json.Unmarshal(body, &items); err != nil {
		return nil
	}

	seen := map[string]bool{}
	var results []SearchResult

	for _, item := range items {
		if item.BookURL == "" {
			continue
		}
		bookURL := item.BookURL
		if strings.HasPrefix(bookURL, "/") {
			bookURL = BaseURL + bookURL
		}
		if !seen[bookURL] {
			seen[bookURL] = true
			results = append(results, SearchResult{
				URL:        bookURL,
				EntityType: "book",
				Title:      item.Title,
			})
		}
		// Also surface the author.
		if item.Author.ProfileURL != "" {
			authorURL := item.Author.ProfileURL
			if strings.HasPrefix(authorURL, "/") {
				authorURL = BaseURL + authorURL
			}
			if !seen[authorURL] {
				seen[authorURL] = true
				results = append(results, SearchResult{
					URL:        authorURL,
					EntityType: "author",
					Title:      item.Author.Name,
				})
			}
		}
	}

	return results
}

// ParseSearchHTML parses book search results from a Goodreads search HTML page.
// Works with or without authentication.
//
// Goodreads uses tr[itemtype="http://schema.org/Book"] for result rows.
// The old tr.bookContainer selector is no longer used.
func ParseSearchHTML(doc *goquery.Document) []SearchResult {
	seen := map[string]bool{}
	var results []SearchResult

	doc.Find(`tr[itemtype="http://schema.org/Book"]`).Each(func(_ int, s *goquery.Selection) {
		a := s.Find("a.bookTitle")
		href, _ := a.Attr("href")
		title := strings.TrimSpace(a.Text())
		if href == "" {
			return
		}
		bookURL := href
		if strings.HasPrefix(bookURL, "/") {
			bookURL = BaseURL + bookURL
		}
		if !seen[bookURL] {
			seen[bookURL] = true
			results = append(results, SearchResult{
				URL:        bookURL,
				EntityType: "book",
				Title:      title,
			})
		}
		// Also surface the author.
		authorA := s.Find("a.authorName")
		authorHref, _ := authorA.Attr("href")
		authorName := strings.TrimSpace(authorA.Text())
		if authorHref != "" {
			authorURL := authorHref
			if strings.HasPrefix(authorURL, "/") {
				authorURL = BaseURL + authorURL
			}
			if !seen[authorURL] {
				seen[authorURL] = true
				results = append(results, SearchResult{
					URL:        authorURL,
					EntityType: "author",
					Title:      authorName,
				})
			}
		}
	})
	return results
}

// ParseSearchHTMLNextPage extracts the "next page" URL from a Goodreads search results page.
func ParseSearchHTMLNextPage(doc *goquery.Document) string {
	href, exists := doc.Find("a.next_page").Attr("href")
	if !exists || href == "" {
		return ""
	}
	if strings.HasPrefix(href, "/") {
		return BaseURL + href
	}
	return href
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
