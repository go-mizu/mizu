package goodread

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reGenreSlug = regexp.MustCompile(`/genres/([^/?#]+)`)
var reBooksCount = regexp.MustCompile(`([\d,]+)\s+book`)

// ParseGenre parses a Goodreads genre page (/genres/<slug>).
func ParseGenre(doc *goquery.Document, slug, pageURL string) (*Genre, []string, error) {
	g := &Genre{
		Slug:      slug,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// Name: typically the h1
	g.Name = strings.TrimSpace(doc.Find("h1, [data-testid='genreTitle']").First().Text())
	if g.Name == "" {
		// Fall back to slug prettify
		g.Name = strings.ReplaceAll(strings.Title(slug), "-", " ")
	}

	// Description
	g.Description = strings.TrimSpace(doc.Find("[data-testid='description'], .genreDescription, .mediumText").First().Text())

	// Books count from page text
	doc.Find("div, span, p").Each(func(_ int, sel *goquery.Selection) {
		if g.BooksCount > 0 {
			return
		}
		text := strings.TrimSpace(sel.Text())
		if m := reBooksCount.FindStringSubmatch(text); len(m) > 1 {
			n, _ := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
			if n > 0 {
				g.BooksCount = n
			}
		}
	})

	// Top book IDs on the genre page
	var bookIDs []string
	seen := map[string]bool{}
	doc.Find("a[href*='/book/show/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if id := extractIDFromPath(href, "/book/show/"); id != "" && !seen[id] {
			seen[id] = true
			bookIDs = append(bookIDs, id)
		}
	})

	return g, bookIDs, nil
}
