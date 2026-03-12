package amazon

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseReviews parses a /product-reviews/<ASIN>?pageNumber=N page.
// Returns all reviews found on the page and the next page URL (empty if last page).
func ParseReviews(doc *goquery.Document, asin, pageURL string) ([]Review, string, error) {
	var reviews []Review
	now := time.Now()

	doc.Find(`#cm_cr-review_list div[data-hook="review"]`).Each(func(_ int, s *goquery.Selection) {
		r := Review{
			ASIN:         asin,
			FetchedAt:    now,
			VariantAttrs: make(map[string]string),
		}

		// ReviewID from container id attribute
		r.ReviewID, _ = s.Attr("id")

		// ReviewerName
		r.ReviewerName = strings.TrimSpace(s.Find(`[data-hook="review-author"]`).Text())

		// ReviewerID from author link href — extract amzn1.account.* segment
		if authorHref, exists := s.Find(`[data-hook="review-author"] a`).Attr("href"); exists {
			r.ReviewerID = extractReviewerID(authorHref)
		}

		// Rating: from star class on rating span
		ratingClass := ""
		s.Find(`[data-hook="review-star-rating"] span, [data-hook="rating-out-of-five"] span`).Each(func(_ int, span *goquery.Selection) {
			if ratingClass != "" {
				return
			}
			cls, exists := span.Attr("class")
			if exists && strings.Contains(cls, "a-star-") {
				ratingClass = cls
			}
		})
		if ratingClass != "" {
			for _, part := range strings.Fields(ratingClass) {
				if strings.HasPrefix(part, "a-star-") && !strings.Contains(part, "small") {
					suffix := strings.TrimPrefix(part, "a-star-")
					segs := strings.Split(suffix, "-")
					if len(segs) == 1 {
						r.Rating = int(parseInt64Str(segs[0]))
					}
					break
				}
			}
		}

		// Title: the review title span (not the stars span)
		s.Find(`[data-hook="review-title"] span`).Each(func(_ int, span *goquery.Selection) {
			if r.Title != "" {
				return
			}
			t := strings.TrimSpace(span.Text())
			// skip the "N.N out of 5 stars" span
			if strings.Contains(t, "out of") && strings.Contains(t, "stars") {
				return
			}
			if t != "" {
				r.Title = t
			}
		})

		// Text
		r.Text = strings.TrimSpace(s.Find(`[data-hook="review-body"] span`).Text())

		// DatePosted
		dateText := strings.TrimSpace(s.Find(`[data-hook="review-date"]`).Text())
		r.DatePosted = parseReviewDate(dateText)

		// VerifiedPurchase
		r.VerifiedPurchase = s.Find(`[data-hook="avp-badge"]`).Length() > 0

		// HelpfulVotes from "N people found this helpful"
		helpfulText := strings.TrimSpace(s.Find(`[data-hook="helpful-vote-statement"]`).Text())
		if helpfulText != "" {
			r.HelpfulVotes = int(parseInt64Digits(helpfulText))
		}

		// Images: review image tiles
		s.Find(".review-image-tile").Each(func(_ int, img *goquery.Selection) {
			src, _ := img.Attr("src")
			if src != "" {
				r.Images = append(r.Images, src)
			}
		})

		// VariantAttrs: "Color: Red  Size: Large"
		formatText := strings.TrimSpace(s.Find(`[data-hook="format-strip"]`).Text())
		if formatText != "" {
			r.VariantAttrs = parseVariantAttrs(formatText)
		}

		// URL from review title link
		if titleHref, exists := s.Find(`[data-hook="review-title"] a`).Attr("href"); exists {
			r.URL = absoluteURL(BaseURL, titleHref)
		}

		reviews = append(reviews, r)
	})

	// Next page URL: last pagination item that is not disabled
	nextPageURL := ""
	doc.Find("li.a-last").Each(func(_ int, s *goquery.Selection) {
		if nextPageURL != "" {
			return
		}
		if s.HasClass("a-disabled") {
			return
		}
		href, exists := s.Find("a").Attr("href")
		if exists && href != "" {
			nextPageURL = absoluteURL(BaseURL, href)
		}
	})

	return reviews, nextPageURL, nil
}

// parseReviewDate parses the date from "Reviewed in X on Month DD, YYYY".
func parseReviewDate(text string) time.Time {
	const prefix = " on "
	idx := strings.Index(text, prefix)
	if idx < 0 {
		// try parsing the full string
		t, err := time.Parse("January 2, 2006", strings.TrimSpace(text))
		if err == nil {
			return t
		}
		return time.Time{}
	}
	dateStr := strings.TrimSpace(text[idx+len(prefix):])
	t, err := time.Parse("January 2, 2006", dateStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

// extractReviewerID extracts the Amazon account ID from a profile URL.
// e.g. "/gp/profile/amzn1.account.ABC123/ref=..." → "amzn1.account.ABC123"
func extractReviewerID(href string) string {
	const marker = "/profile/"
	idx := strings.Index(href, marker)
	if idx < 0 {
		return ""
	}
	rest := href[idx+len(marker):]
	end := strings.IndexAny(rest, "/?#")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// parseVariantAttrs converts "Color: Red  Size: Large" into a map.
func parseVariantAttrs(text string) map[string]string {
	attrs := make(map[string]string)
	// Split on two or more spaces, or on explicit delimiter
	// Normalise: replace multiple spaces with a single "|"
	parts := strings.FieldsFunc(text, func(r rune) bool { return r == '|' })
	if len(parts) == 1 {
		// Try splitting on two+ spaces
		parts = strings.Split(text, "  ")
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if colon := strings.Index(part, ":"); colon > 0 {
			k := strings.TrimSpace(part[:colon])
			v := strings.TrimSpace(part[colon+1:])
			if k != "" {
				attrs[k] = v
			}
		}
	}
	return attrs
}
