package goodread

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reListID = regexp.MustCompile(`/list/show/(\d+)`)
var reListBookID = regexp.MustCompile(`/book/show/(\d+)`)

// ParseList parses a Goodreads listopia list page.
func ParseList(doc *goquery.Document, listID, pageURL string) (*List, []ListBook, error) {
	l := &List{
		ListID:    listID,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// List name
	l.Name = strings.TrimSpace(doc.Find("h1.listTitle, h1[class*='listTitle'], h1").First().Text())

	// Description
	l.Description = strings.TrimSpace(doc.Find(".listDescription, [data-testid='description']").First().Text())

	// Creator
	doc.Find("a[href*='/user/show/'], a[href*='/profile/']").First().Each(func(_ int, sel *goquery.Selection) {
		l.CreatedByUser = strings.TrimSpace(sel.Text())
	})

	// Votes/voters count
	doc.Find("[class*='smallText'], .smallText").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if strings.Contains(text, "voter") {
			parts := strings.Fields(text)
			if len(parts) > 0 {
				n, _ := strconv.Atoi(strings.ReplaceAll(parts[0], ",", ""))
				if n > 0 {
					l.VotersCount = n
				}
			}
		}
	})

	// Tags
	doc.Find("a[href*='/list/tag/']").Each(func(_ int, sel *goquery.Selection) {
		tag := strings.TrimSpace(sel.Text())
		if tag != "" && !contains(l.Tags, tag) {
			l.Tags = append(l.Tags, tag)
		}
	})

	// Books in list with rank and votes
	var books []ListBook
	rank := 1

	doc.Find("tr.bookContainer, tr[itemtype*='Book'], .listWithDividers__item").Each(func(_ int, row *goquery.Selection) {
		var bookID string
		row.Find("a[href*='/book/show/']").Each(func(_ int, a *goquery.Selection) {
			if bookID != "" {
				return
			}
			href, _ := a.Attr("href")
			if m := reListBookID.FindStringSubmatch(href); len(m) > 1 {
				bookID = m[1]
			}
		})

		if bookID == "" {
			return
		}

		lb := ListBook{
			ListID: listID,
			BookID: bookID,
			Rank:   rank,
		}

		// Votes
		row.Find("[class*='smallText']").Each(func(_ int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if strings.Contains(text, "vote") {
				n, _ := strconv.Atoi(strings.ReplaceAll(strings.Fields(text)[0], ",", ""))
				lb.Votes = n
			}
		})

		books = append(books, lb)
		rank++
	})

	// Fallback: find books via links
	if len(books) == 0 {
		seen := map[string]bool{}
		doc.Find("a[href*='/book/show/']").Each(func(_ int, sel *goquery.Selection) {
			href, _ := sel.Attr("href")
			if m := reListBookID.FindStringSubmatch(href); len(m) > 1 {
				bookID := m[1]
				if !seen[bookID] {
					seen[bookID] = true
					books = append(books, ListBook{
						ListID: listID,
						BookID: bookID,
						Rank:   rank,
					})
					rank++
				}
			}
		})
	}

	l.BooksCount = len(books)
	return l, books, nil
}
