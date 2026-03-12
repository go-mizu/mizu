package goodread

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reJSONLD = regexp.MustCompile(`<script[^>]+type=["']application/ld\+json["'][^>]*>([\s\S]*?)</script>`)

// bookJSONLD is the JSON-LD structure embedded in Goodreads book pages.
// Author is json.RawMessage because Goodreads may send either an object or an array.
type bookJSONLD struct {
	Type            string          `json:"@type"`
	Name            string          `json:"name"`
	Author          json.RawMessage `json:"author"` // may be {} or [{}]
	Image           string          `json:"image"`
	Description     string          `json:"description"`
	ISBN            string          `json:"isbn"`
	NumberOfPages   int             `json:"numberOfPages"`
	InLanguage      string          `json:"inLanguage"`
	URL             string          `json:"url"`
	AggregateRating struct {
		RatingValue json.RawMessage `json:"ratingValue"`
		RatingCount json.RawMessage `json:"ratingCount"`
		ReviewCount json.RawMessage `json:"reviewCount"`
	} `json:"aggregateRating"`
	Publisher struct {
		Name string `json:"name"`
	} `json:"publisher"`
	DatePublished string `json:"datePublished"`
	BookFormat    string `json:"bookFormat"`
	ISBN13        string `json:"isbn13"`
	ASIN          string `json:"asin"`
}

// authorObj is a single author entry in JSON-LD.
type authorObj struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// parseAuthorField handles both a single object and an array of author objects.
func parseAuthorField(raw json.RawMessage) authorObj {
	if len(raw) == 0 {
		return authorObj{}
	}
	// Try array first
	var arr []authorObj
	if json.Unmarshal(raw, &arr) == nil && len(arr) > 0 {
		return arr[0]
	}
	// Try single object
	var obj authorObj
	json.Unmarshal(raw, &obj)
	return obj
}

// ParseBook parses a Goodreads book page and returns a Book struct.
func ParseBook(doc *goquery.Document, bookID, pageURL string) (*Book, error) {
	b := &Book{
		BookID:    bookID,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// Try JSON-LD first — use goquery Text() to get unescaped script content.
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, sel *goquery.Selection) {
		if b.AvgRating != 0 || b.Title != "" {
			return // already found a Book JSON-LD
		}
		text := sel.Text()
		var ld bookJSONLD
		if err := json.Unmarshal([]byte(text), &ld); err != nil {
			return
		}
		if ld.Type != "Book" {
			return
		}

		b.Title = ld.Name
		b.CoverURL = ld.Image
		b.Description = cleanHTML(ld.Description)
		b.ISBN = ld.ISBN
		b.ISBN13 = ld.ISBN13
		b.ASIN = ld.ASIN
		b.Pages = ld.NumberOfPages
		b.Language = ld.InLanguage
		b.Format = ld.BookFormat

		author := parseAuthorField(ld.Author)
		b.AuthorName = author.Name
		if author.URL != "" {
			b.AuthorID = extractIDFromPath(author.URL, "/author/show/")
		}

		if ld.Publisher.Name != "" {
			b.Publisher = ld.Publisher.Name
		}
		if ld.DatePublished != "" && len(ld.DatePublished) >= 4 {
			if y, err := strconv.Atoi(ld.DatePublished[:4]); err == nil {
				b.PublishedYear = y
			}
		}
		b.AvgRating = parseRawFloat(ld.AggregateRating.RatingValue)
		b.RatingsCount = parseRawInt64(ld.AggregateRating.RatingCount)
		b.ReviewsCount = parseRawInt64(ld.AggregateRating.ReviewCount)
	})

	// HTML fallback / enrichment
	if b.Title == "" {
		b.Title = strings.TrimSpace(doc.Find("h1[data-testid='bookTitle'], h1.Text__title1").First().Text())
	}
	if b.Title == "" {
		b.Title = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	// Title without series
	b.TitleWithoutSeries = b.Title
	if idx := strings.Index(b.Title, "("); idx > 0 {
		b.TitleWithoutSeries = strings.TrimSpace(b.Title[:idx])
	}

	// Author (from DOM if JSON-LD missing)
	if b.AuthorName == "" {
		b.AuthorName = strings.TrimSpace(doc.Find("[data-testid='name']").First().Text())
	}
	if b.AuthorID == "" {
		doc.Find("a[href*='/author/show/']").Each(func(_ int, sel *goquery.Selection) {
			if b.AuthorID != "" {
				return
			}
			href, _ := sel.Attr("href")
			if id := extractIDFromPath(href, "/author/show/"); id != "" {
				b.AuthorID = id
			}
		})
	}

	// Cover image fallback
	if b.CoverURL == "" {
		b.CoverURL, _ = doc.Find("img.ResponsiveImage").First().Attr("src")
	}

	// Genres
	doc.Find("a[href*='/genres/']").Each(func(_ int, sel *goquery.Selection) {
		g := strings.TrimSpace(sel.Text())
		if g != "" && !contains(b.Genres, g) {
			b.Genres = append(b.Genres, g)
		}
	})

	// Series
	doc.Find("a[href*='/series/']").Each(func(_ int, sel *goquery.Selection) {
		if b.SeriesName != "" {
			return
		}
		href, _ := sel.Attr("href")
		if id := extractIDFromPath(href, "/series/"); id != "" {
			b.SeriesID = id
			b.SeriesName = strings.TrimSpace(sel.Text())
		}
	})

	// Series position from title
	if seriesIdx := strings.Index(b.Title, "#"); seriesIdx > 0 {
		end := strings.Index(b.Title[seriesIdx:], ")")
		if end > 0 {
			b.SeriesPosition = strings.TrimSpace(b.Title[seriesIdx+1 : seriesIdx+end])
		}
	}

	// Similar books
	doc.Find("a[href*='/book/show/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if id := extractIDFromPath(href, "/book/show/"); id != "" && id != bookID {
			if !contains(b.SimilarBookIDs, id) && len(b.SimilarBookIDs) < 20 {
				b.SimilarBookIDs = append(b.SimilarBookIDs, id)
			}
		}
	})

	return b, nil
}

// parseRawFloat parses a JSON value that may be either a number or a quoted string.
func parseRawFloat(raw json.RawMessage) float64 {
	if len(raw) == 0 {
		return 0
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return f
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		f, _ = strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	}
	return f
}

// parseRawInt64 parses a JSON value that may be either a number or a quoted string.
func parseRawInt64(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		return n
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		s = strings.ReplaceAll(s, ",", "")
		n, _ = strconv.ParseInt(s, 10, 64)
	}
	return n
}

// extractIDFromPath extracts the ID portion after a path prefix.
// e.g. "/author/show/12345-name" → "12345"
func extractIDFromPath(href, prefix string) string {
	idx := strings.Index(href, prefix)
	if idx < 0 {
		return ""
	}
	rest := href[idx+len(prefix):]
	// Remove query/fragment
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	// Take numeric prefix only
	for i, c := range rest {
		if c == '-' || c == '/' {
			return rest[:i]
		}
		if c < '0' || c > '9' {
			if i == 0 {
				return rest
			}
			return rest[:i]
		}
	}
	return rest
}

// cleanHTML strips HTML tags from a string.
func cleanHTML(s string) string {
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.TrimSpace(s)
	return s
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
