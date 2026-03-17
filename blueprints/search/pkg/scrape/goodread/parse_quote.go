package goodread

import (
	"regexp"
	"strings"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reQuoteID = regexp.MustCompile(`/quotes/(\d+)`)
var reQuoteLikes = regexp.MustCompile(`(\d[\d,]*)\s+like`)

// ParseQuotes parses a Goodreads quotes page (/quotes/<id> or /author/quotes/<id>).
func ParseQuotes(doc *goquery.Document, pageURL string) ([]Quote, error) {
	var quotes []Quote

	doc.Find(".quoteText, [class*='quoteText'], [data-testid='quoteText']").Each(func(_ int, sel *goquery.Selection) {
		q := Quote{
			URL:       pageURL,
			FetchedAt: time.Now(),
		}

		// Get text — the quote itself is the direct text node
		q.Text = strings.TrimSpace(sel.Clone().Children().Remove().End().Text())
		if q.Text == "" {
			q.Text = strings.TrimSpace(sel.Text())
		}
		// Clean up leading/trailing quote marks
		q.Text = strings.Trim(q.Text, "\u201c\u201d\u2018\u2019\"'")
		q.Text = strings.TrimSpace(q.Text)

		if q.Text == "" {
			return
		}

		// Author and book from siblings
		parent := sel.Parent()
		parent.Find("a[href*='/author/show/']").First().Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			q.AuthorID = extractIDFromPath(href, "/author/show/")
			q.AuthorName = strings.TrimSpace(a.Text())
		})
		parent.Find("a[href*='/work/']").First().Each(func(_ int, a *goquery.Selection) {
			q.BookTitle = strings.TrimSpace(a.Text())
		})
		parent.Find("a[href*='/book/show/']").First().Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			q.BookID = extractIDFromPath(href, "/book/show/")
			if q.BookTitle == "" {
				q.BookTitle = strings.TrimSpace(a.Text())
			}
		})

		// Quote ID and likes from the quoteFooter block
		parent.Find("a[href*='/quotes/']").Each(func(_ int, a *goquery.Selection) {
			if q.QuoteID != "" {
				return
			}
			href, _ := a.Attr("href")
			if m := reQuoteID.FindStringSubmatch(href); len(m) > 1 {
				q.QuoteID = m[1]
				q.URL = BaseURL + href
			}
		})

		parent.Find("[class*='likes']").Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if m := reQuoteLikes.FindStringSubmatch(text); len(m) > 1 {
				n, _ := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
				q.LikesCount = n
			}
		})

		// Tags
		parent.Find("a[href*='/quotes/tag/'], a[href*='/quotes?tag=']").Each(func(_ int, a *goquery.Selection) {
			tag := strings.TrimSpace(a.Text())
			if tag != "" && !contains(q.Tags, tag) {
				q.Tags = append(q.Tags, tag)
			}
		})

		if q.QuoteID == "" {
			// Generate a pseudo-ID from the text hash
			q.QuoteID = "q" + hashStr(q.Text)[:12]
		}

		quotes = append(quotes, q)
	})

	return quotes, nil
}

func hashStr(s string) string {
	h := uint64(14695981039346656037)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return strconv.FormatUint(h, 16)
}
