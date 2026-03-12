package goodread

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var reReviewID = regexp.MustCompile(`/review/show/(\d+)`)

// reviewJSONLD is the structure for embedded reviews in JSON-LD.
type reviewJSONLD struct {
	Type         string `json:"@type"`
	ReviewRating struct {
		RatingValue string `json:"ratingValue"`
	} `json:"reviewRating"`
	Author struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"author"`
	ReviewBody  string `json:"reviewBody"`
	DateCreated string `json:"dateCreated"`
	URL         string `json:"url"`
}

// ParseReviews extracts embedded reviews from a book page.
// Goodreads embeds the first ~30 reviews in JSON-LD and HTML.
func ParseReviews(doc *goquery.Document, bookID string) []Review {
	var reviews []Review

	// Try to extract from JSON-LD — use .Text() to get unescaped script content
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, sel *goquery.Selection) {
		// The main Book JSON-LD may contain a "review" array
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(sel.Text()), &raw); err != nil {
			return
		}
		reviewsRaw, ok := raw["review"]
		if !ok {
			return
		}

		var jsonReviews []reviewJSONLD
		if err := json.Unmarshal(reviewsRaw, &jsonReviews); err != nil {
			// single review?
			var single reviewJSONLD
			if err2 := json.Unmarshal(reviewsRaw, &single); err2 == nil {
				jsonReviews = []reviewJSONLD{single}
			}
		}

		for _, r := range jsonReviews {
			review := Review{
				BookID:    bookID,
				Text:      cleanHTML(r.ReviewBody),
				FetchedAt: time.Now(),
			}
			review.Rating, _ = strconv.Atoi(r.ReviewRating.RatingValue)
			review.UserName = r.Author.Name
			review.UserID = extractIDFromPath(r.Author.URL, "/user/show/")
			review.URL = r.URL
			if review.ReviewID == "" && review.URL != "" {
				if m2 := reReviewID.FindStringSubmatch(review.URL); len(m2) > 1 {
					review.ReviewID = m2[1]
				}
			}
			if r.DateCreated != "" {
				review.DateAdded, _ = time.Parse("2006-01-02", r.DateCreated)
			}
			if review.ReviewID != "" {
				reviews = append(reviews, review)
			}
		}
	})

	// HTML fallback: parse review cards from DOM
	if len(reviews) == 0 {
		doc.Find("[data-testid='review'], .ReviewCard, .review").Each(func(_ int, sel *goquery.Selection) {
			review := Review{
				BookID:    bookID,
				FetchedAt: time.Now(),
			}

			// Review URL / ID
			sel.Find("a[href*='/review/show/']").Each(func(_ int, a *goquery.Selection) {
				if review.ReviewID != "" {
					return
				}
				href, _ := a.Attr("href")
				if m := reReviewID.FindStringSubmatch(href); len(m) > 1 {
					review.ReviewID = m[1]
					review.URL = BaseURL + href
				}
			})

			// Author
			review.UserName = strings.TrimSpace(sel.Find("[data-testid='name'], .ReviewerProfile__name").First().Text())
			sel.Find("a[href*='/user/show/']").Each(func(_ int, a *goquery.Selection) {
				if review.UserID != "" {
					return
				}
				href, _ := a.Attr("href")
				review.UserID = extractIDFromPath(href, "/user/show/")
			})

			// Rating: count filled stars
			review.Rating = sel.Find("[data-testid='ratingStars'] [class*='filled'], .star.off + .star, svg.star").Length()
			if review.Rating == 0 {
				sel.Find("[aria-label]").Each(func(_ int, s *goquery.Selection) {
					label, _ := s.Attr("aria-label")
					if strings.Contains(label, "stars") {
						n, _ := strconv.Atoi(strings.Fields(label)[0])
						review.Rating = n
					}
				})
			}

			// Text
			review.Text = strings.TrimSpace(sel.Find("[data-testid='reviewText'], .ReviewText__content, .reviewText").Text())

			if review.ReviewID != "" && review.Text != "" {
				reviews = append(reviews, review)
			}
		})
	}

	return reviews
}
