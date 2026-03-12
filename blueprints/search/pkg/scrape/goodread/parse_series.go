package goodread

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reSeriesBooks = regexp.MustCompile(`/book/show/(\d+)`)

// ParseSeries parses a Goodreads series page.
func ParseSeries(doc *goquery.Document, seriesID, pageURL string) (*Series, []SeriesBook, error) {
	s := &Series{
		SeriesID:  seriesID,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// Series name
	s.Name = strings.TrimSpace(doc.Find("h1.gr-h1--serif, [data-testid='seriesTitle'], h1").First().Text())
	s.Name = strings.TrimSuffix(s.Name, " Series")

	// Description
	s.Description = strings.TrimSpace(doc.Find("[data-testid='description'], .seriesDesc").First().Text())

	// Total books / primary work count from text like "3 primary works • 7 total works"
	doc.Find("div, span, p").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if strings.Contains(text, "primary work") {
			parts := strings.Fields(text)
			for i, p := range parts {
				if strings.Contains(p, "primary") && i > 0 {
					n, _ := strconv.Atoi(parts[i-1])
					s.PrimaryWorkCount = n
				}
				if strings.Contains(p, "total") && i > 0 {
					n, _ := strconv.Atoi(parts[i-1])
					s.TotalBooks = n
				}
			}
		}
	})

	// Collect books in series
	var books []SeriesBook
	seen := map[string]bool{}
	pos := 1

	doc.Find("a[href*='/book/show/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if m := reSeriesBooks.FindStringSubmatch(href); len(m) > 1 {
			bookID := m[1]
			if seen[bookID] {
				return
			}
			seen[bookID] = true
			books = append(books, SeriesBook{
				SeriesID: seriesID,
				BookID:   bookID,
				Position: pos,
			})
			pos++
		}
	})

	if s.TotalBooks == 0 {
		s.TotalBooks = len(books)
	}

	return s, books, nil
}
