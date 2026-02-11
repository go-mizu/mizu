package goodreads

import (
	"encoding/json"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reJSONLD         = regexp.MustCompile(`<script\s+type="application/ld\+json"[^>]*>([\s\S]*?)</script>`)
	reDescription    = regexp.MustCompile(`<div[^>]*data-testid="description"[^>]*>([\s\S]*?)</div>`)
	reDescSpan       = regexp.MustCompile(`<span[^>]*>([\s\S]*?)</span>`)
	reGenre          = regexp.MustCompile(`<a[^>]*href="/genres/[^"]*"[^>]*>([^<]+)</a>`)
	reCurrentReading = regexp.MustCompile(`([\d,]+)\s*people?\s*(?:are\s+)?currently\s+reading`)
	reWantToRead     = regexp.MustCompile(`([\d,]+)\s*people?\s*want\s+to\s+read`)
	reRatingBar      = regexp.MustCompile(`(?i)(\d)\s*(?:star|Stars)[^<]*?([\d,]+)`)
	reSeries         = regexp.MustCompile(`<a[^>]*href="/series/[^"]*"[^>]*>([^<]+)</a>`)
	rePublisher      = regexp.MustCompile(`(?i)(?:published|publisher)[^<]*?(?:by\s+)?([A-Z][^<,\n]{2,40})`)
	reFirstPub       = regexp.MustCompile(`(?i)first\s+published?\s+([^<\n]+?)(?:\)|<)`)
	reASIN           = regexp.MustCompile(`(?i)ASIN[:\s]+([A-Z0-9]{10})`)
	reCoverImg       = regexp.MustCompile(`<img[^>]*class="[^"]*ResponsiveImage[^"]*"[^>]*src="([^"]+)"`)
	reReviewBlock    = regexp.MustCompile(`(?s)<article[^>]*class="[^"]*ReviewCard[^"]*"[^>]*>([\s\S]*?)</article>`)
	reReviewerName   = regexp.MustCompile(`<a[^>]*class="[^"]*ReviewerProfile[^"]*"[^>]*>[\s\S]*?<span[^>]*>([^<]+)</span>`)
	reReviewDate     = regexp.MustCompile(`<span[^>]*class="[^"]*Text__body3[^"]*"[^>]*>([A-Z][a-z]+\s+\d{1,2},?\s+\d{4})</span>`)
	reReviewText     = regexp.MustCompile(`<span[^>]*class="[^"]*Formatted[^"]*"[^>]*>([\s\S]*?)</span>`)
	reReviewLikes    = regexp.MustCompile(`(\d+)\s*like`)
	reReviewStars    = regexp.MustCompile(`Rating\s+(\d)\s+out\s+of\s+5`)
	reRatingDist5    = regexp.MustCompile(`(?i)5\s*(?:star|Stars)\s*[^0-9]*([\d,]+)`)
	reRatingDist4    = regexp.MustCompile(`(?i)4\s*(?:star|Stars)\s*[^0-9]*([\d,]+)`)
	reRatingDist3    = regexp.MustCompile(`(?i)3\s*(?:star|Stars)\s*[^0-9]*([\d,]+)`)
	reRatingDist2    = regexp.MustCompile(`(?i)2\s*(?:star|Stars)\s*[^0-9]*([\d,]+)`)
	reRatingDist1    = regexp.MustCompile(`(?i)1\s*(?:star|Stars)\s*[^0-9]*([\d,]+)`)
	reStripTags      = regexp.MustCompile(`<[^>]*>`)
)

func parseBookPage(body string) (*GoodreadsBook, error) {
	book := &GoodreadsBook{}

	// 1. Parse JSON-LD structured data
	parseJSONLD(body, book)

	// 2. Parse description
	parseDescription(body, book)

	// 3. Parse genres
	parseGenres(body, book)

	// 4. Parse stats (currently reading, want to read)
	parseStats(body, book)

	// 5. Parse rating distribution
	parseRatingDist(body, book)

	// 6. Parse series
	parseSeries(body, book)

	// 7. Parse additional metadata
	parseMetadata(body, book)

	// 8. Parse cover image
	parseCover(body, book)

	// 9. Parse reviews
	parseReviews(body, book)

	return book, nil
}

func parseJSONLD(body string, book *GoodreadsBook) {
	matches := reJSONLD.FindAllStringSubmatch(body, -1)
	for _, m := range matches {
		var ld jsonLD
		if err := json.Unmarshal([]byte(m[1]), &ld); err != nil {
			continue
		}
		if ld.Type != "Book" {
			continue
		}
		book.Title = ld.Name
		book.CoverURL = ld.Image
		book.Format = ld.BookFormat
		book.PageCount = ld.NumberOfPages
		book.Language = ld.InLanguage
		book.ISBN13 = ld.ISBN
		if len(ld.ISBN) == 10 {
			book.ISBN = ld.ISBN
			book.ISBN13 = ""
		}
		book.AverageRating = ld.AggregateRating.RatingValue
		book.RatingsCount = ld.AggregateRating.RatingCount
		book.ReviewsCount = ld.AggregateRating.ReviewCount
		if len(ld.Author) > 0 {
			names := make([]string, len(ld.Author))
			for i, a := range ld.Author {
				names[i] = a.Name
			}
			book.AuthorName = strings.Join(names, ", ")
		}
		break
	}
}

func parseDescription(body string, book *GoodreadsBook) {
	match := reDescription.FindStringSubmatch(body)
	if match == nil {
		return
	}
	content := match[1]
	// Extract from inner spans (Goodreads wraps in nested spans)
	spans := reDescSpan.FindAllStringSubmatch(content, -1)
	if len(spans) > 0 {
		// Take the longest span (full description vs truncated)
		longest := ""
		for _, s := range spans {
			text := stripTags(s[1])
			if len(text) > len(longest) {
				longest = text
			}
		}
		if longest != "" {
			book.Description = strings.TrimSpace(html.UnescapeString(longest))
			return
		}
	}
	book.Description = strings.TrimSpace(html.UnescapeString(stripTags(content)))
}

func parseGenres(body string, book *GoodreadsBook) {
	matches := reGenre.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	for _, m := range matches {
		genre := strings.TrimSpace(m[1])
		if genre == "" || seen[genre] {
			continue
		}
		seen[genre] = true
		book.Genres = append(book.Genres, genre)
	}
}

func parseStats(body string, book *GoodreadsBook) {
	if m := reCurrentReading.FindStringSubmatch(body); m != nil {
		book.CurrentlyReading = parseCommaInt(m[1])
	}
	if m := reWantToRead.FindStringSubmatch(body); m != nil {
		book.WantToRead = parseCommaInt(m[1])
	}
}

func parseRatingDist(body string, book *GoodreadsBook) {
	// Try individual star patterns
	if m := reRatingDist5.FindStringSubmatch(body); m != nil {
		book.RatingDist[0] = parseCommaInt(m[1])
	}
	if m := reRatingDist4.FindStringSubmatch(body); m != nil {
		book.RatingDist[1] = parseCommaInt(m[1])
	}
	if m := reRatingDist3.FindStringSubmatch(body); m != nil {
		book.RatingDist[2] = parseCommaInt(m[1])
	}
	if m := reRatingDist2.FindStringSubmatch(body); m != nil {
		book.RatingDist[3] = parseCommaInt(m[1])
	}
	if m := reRatingDist1.FindStringSubmatch(body); m != nil {
		book.RatingDist[4] = parseCommaInt(m[1])
	}

	// Fallback: generic N star pattern
	if book.RatingDist == [5]int{} {
		matches := reRatingBar.FindAllStringSubmatch(body, -1)
		for _, m := range matches {
			star, _ := strconv.Atoi(m[1])
			count := parseCommaInt(m[2])
			if star >= 1 && star <= 5 {
				book.RatingDist[5-star] = count
			}
		}
	}
}

func parseSeries(body string, book *GoodreadsBook) {
	if m := reSeries.FindStringSubmatch(body); m != nil {
		book.Series = strings.TrimSpace(html.UnescapeString(m[1]))
	}
}

func parseMetadata(body string, book *GoodreadsBook) {
	if m := reFirstPub.FindStringSubmatch(body); m != nil {
		book.FirstPublished = strings.TrimSpace(m[1])
	}
	if m := reASIN.FindStringSubmatch(body); m != nil {
		book.ASIN = m[1]
	}
	if m := rePublisher.FindStringSubmatch(body); m != nil && book.Publisher == "" {
		book.Publisher = strings.TrimSpace(m[1])
	}
}

func parseCover(body string, book *GoodreadsBook) {
	if book.CoverURL != "" {
		return
	}
	if m := reCoverImg.FindStringSubmatch(body); m != nil {
		book.CoverURL = m[1]
	}
}

func parseReviews(body string, book *GoodreadsBook) {
	blocks := reReviewBlock.FindAllStringSubmatch(body, 30)
	for _, block := range blocks {
		content := block[1]
		review := GoodreadsReview{}

		if m := reReviewerName.FindStringSubmatch(content); m != nil {
			review.ReviewerName = strings.TrimSpace(m[1])
		}
		if m := reReviewStars.FindStringSubmatch(content); m != nil {
			review.Rating, _ = strconv.Atoi(m[1])
		}
		if m := reReviewDate.FindStringSubmatch(content); m != nil {
			review.Date = strings.TrimSpace(m[1])
		}
		if m := reReviewText.FindStringSubmatch(content); m != nil {
			review.Text = strings.TrimSpace(html.UnescapeString(stripTags(m[1])))
		}
		if m := reReviewLikes.FindStringSubmatch(content); m != nil {
			review.LikesCount, _ = strconv.Atoi(m[1])
		}

		if review.ReviewerName != "" || review.Text != "" {
			book.Reviews = append(book.Reviews, review)
		}
	}
}

func stripTags(s string) string {
	return strings.TrimSpace(reStripTags.ReplaceAllString(s, ""))
}

func parseCommaInt(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	n, _ := strconv.Atoi(s)
	return n
}
