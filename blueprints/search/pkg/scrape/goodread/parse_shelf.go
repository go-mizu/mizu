package goodread

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reShelfBookID = regexp.MustCompile(`/book/show/(\d+)`)
var reDateRead = regexp.MustCompile(`(\d{4})`)

// ParseShelf parses a Goodreads user shelf page (/review/list/<user_id>?shelf=...).
func ParseShelf(doc *goquery.Document, userID, shelfName, pageURL string) (*Shelf, []ShelfBook, error) {
	shelfID := userID + "/" + shelfName
	s := &Shelf{
		ShelfID:   shelfID,
		UserID:    userID,
		Name:      shelfName,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// Shelf display name
	label := strings.TrimSpace(doc.Find("h2, [data-testid='shelfTitle'], .h2Container").First().Text())
	if label != "" {
		s.Name = label
	}

	// Books count
	doc.Find("[class*='headerCount'], .headerBooksCount").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		text = regexp.MustCompile(`[^\d]`).ReplaceAllString(text, "")
		if n, err := strconv.Atoi(text); err == nil && n > 0 {
			s.BooksCount = n
		}
	})

	// Books on this page
	var books []ShelfBook

	doc.Find("tr.bookalike, tr[id^='review_']").Each(func(_ int, row *goquery.Selection) {
		var bookID string
		row.Find("a[href*='/book/show/']").Each(func(_ int, a *goquery.Selection) {
			if bookID != "" {
				return
			}
			href, _ := a.Attr("href")
			if m := reShelfBookID.FindStringSubmatch(href); len(m) > 1 {
				bookID = m[1]
			}
		})
		if bookID == "" {
			return
		}

		sb := ShelfBook{
			ShelfID: shelfID,
			UserID:  userID,
			BookID:  bookID,
		}

		// Rating: count stars
		sb.Rating = row.Find("[class*='staticStar'][class*='p10']").Length()

		// Date added
		row.Find("[class*='date_added'] .date_started_value, [class*='date_added'] .date_added").Each(func(_ int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if t, err := time.Parse("Jan 02, 2006", text); err == nil {
				sb.DateAdded = t
			}
		})

		// Date read
		row.Find("[class*='date_read'] .date_read_value").Each(func(_ int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if t, err := time.Parse("Jan 02, 2006", text); err == nil {
				sb.DateRead = t
			} else if m := reDateRead.FindString(text); m != "" {
				if t, err := time.Parse("2006", m); err == nil {
					sb.DateRead = t
				}
			}
		})

		books = append(books, sb)
	})

	if s.BooksCount == 0 {
		s.BooksCount = len(books)
	}

	return s, books, nil
}

// ParseShelfNextPage returns the URL for the next page of shelf results, or "".
func ParseShelfNextPage(doc *goquery.Document) string {
	var next string
	doc.Find("a[href*='page=']").Each(func(_ int, sel *goquery.Selection) {
		text := strings.ToLower(strings.TrimSpace(sel.Text()))
		if text == "next" || text == "next »" || text == "next page" {
			href, _ := sel.Attr("href")
			if href != "" {
				if strings.HasPrefix(href, "/") {
					next = BaseURL + href
				} else {
					next = href
				}
			}
		}
	})
	return next
}
