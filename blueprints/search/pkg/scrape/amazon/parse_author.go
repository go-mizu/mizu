package amazon

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseAuthor parses an Amazon Author Central page (/author/<slug>).
func ParseAuthor(doc *goquery.Document, pageURL string) (*Author, error) {
	a := &Author{
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// AuthorID: slug after /author/
	a.AuthorID = extractPathSegmentAfter(pageURL, "/author/")

	// Name
	a.Name = strings.TrimSpace(doc.Find("#ap_author_name").Text())
	if a.Name == "" {
		a.Name = strings.TrimSpace(doc.Find("h1.author-name").Text())
	}
	if a.Name == "" {
		a.Name = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	// Bio
	a.Bio = strings.TrimSpace(doc.Find("#ap_author_bio").Text())
	if a.Bio == "" {
		a.Bio = strings.TrimSpace(doc.Find(".author-bio").Text())
	}

	// PhotoURL
	if src, exists := doc.Find("#ap_author_image img").Attr("src"); exists {
		a.PhotoURL = strings.TrimSpace(src)
	}
	if a.PhotoURL == "" {
		if src, exists := doc.Find(".author-image img").Attr("src"); exists {
			a.PhotoURL = strings.TrimSpace(src)
		}
	}

	// Website
	if href, exists := doc.Find("#ap_author_website a").Attr("href"); exists {
		a.Website = strings.TrimSpace(href)
	}

	// Twitter handle
	if href, exists := doc.Find("#ap_author_twitter a").Attr("href"); exists {
		href = strings.TrimSpace(href)
		// extract handle from URL, e.g. "https://twitter.com/handle"
		parts := strings.Split(strings.TrimRight(href, "/"), "/")
		if len(parts) > 0 {
			handle := parts[len(parts)-1]
			if handle != "" && !strings.HasPrefix(handle, "http") {
				a.Twitter = "@" + strings.TrimPrefix(handle, "@")
			}
		}
	}

	// BookASINs from author book list
	seen := make(map[string]bool)
	doc.Find(`#author-book-list-template-1 .a-link-normal[href*="/dp/"]`).Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if asin := ExtractASIN(href); asin != "" && !seen[asin] {
			seen[asin] = true
			a.BookASINs = append(a.BookASINs, asin)
		}
	})

	// FollowerCount: not visible
	a.FollowerCount = 0

	return a, nil
}
