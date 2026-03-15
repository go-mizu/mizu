package amazon

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseSeller parses an Amazon seller profile page (/sp?seller=<id>).
func ParseSeller(doc *goquery.Document, sellerID, pageURL string) (*Seller, error) {
	s := &Seller{
		SellerID:  sellerID,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// SellerID fallback from URL
	if s.SellerID == "" {
		s.SellerID = extractQueryParam(pageURL, "seller")
	}

	// Name: #sellerName, then first h1
	s.Name = strings.TrimSpace(doc.Find("#sellerName").Text())
	if s.Name == "" {
		s.Name = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	// Rating: try #effective-timeframe-12month star class
	doc.Find("#effective-timeframe-12month [class*='a-icon-star']").Each(func(_ int, star *goquery.Selection) {
		if s.Rating != 0 {
			return
		}
		cls, _ := star.Attr("class")
		if r := starRatingFromClass(cls); r > 0 {
			s.Rating = r
		}
	})
	// Fallback: .seller-feedback-rating .a-size-large text
	if s.Rating == 0 {
		ratingText := strings.TrimSpace(doc.Find(".seller-feedback-rating .a-size-large").First().Text())
		if ratingText != "" {
			s.Rating = parseFloatStr(strings.Fields(ratingText)[0])
		}
	}

	// RatingCount
	doc.Find("#effective-timeframe-12month").Each(func(_ int, sel *goquery.Selection) {
		if s.RatingCount != 0 {
			return
		}
		t := strings.TrimSpace(sel.Text())
		if v := parseInt64Digits(t); v > 0 {
			s.RatingCount = int(v)
		}
	})
	if s.RatingCount == 0 {
		t := strings.TrimSpace(doc.Find(".total-ratings-count").First().Text())
		s.RatingCount = int(parseInt64Digits(t))
	}

	// Feedback percentages: rows in seller feedback table
	// Typical structure: rows for Positive, Neutral, Negative each containing
	// a .feedback-percentage span.
	rowTexts := []string{"positive", "neutral", "negative"}
	doc.Find(".feedback-percentage").Each(func(i int, sel *goquery.Selection) {
		pctText := strings.TrimSpace(sel.Text())
		pctText = strings.ReplaceAll(pctText, "%", "")
		v := parseFloatStr(pctText)
		switch {
		case i < len(rowTexts) && rowTexts[i] == "positive":
			s.PositivePct = v
		case i < len(rowTexts) && rowTexts[i] == "neutral":
			s.NeutralPct = v
		case i < len(rowTexts) && rowTexts[i] == "negative":
			s.NegativePct = v
		}
	})

	// If percentages weren't found via class, try table row text matching
	if s.PositivePct == 0 && s.NeutralPct == 0 && s.NegativePct == 0 {
		doc.Find("table tr, .feedback-row").Each(func(_ int, row *goquery.Selection) {
			text := strings.ToLower(row.Text())
			pctText := ""
			row.Find("td, span").Each(func(_ int, cell *goquery.Selection) {
				t := strings.TrimSpace(cell.Text())
				if strings.HasSuffix(t, "%") {
					pctText = strings.ReplaceAll(t, "%", "")
				}
			})
			v := parseFloatStr(pctText)
			if strings.Contains(text, "positive") && s.PositivePct == 0 {
				s.PositivePct = v
			} else if strings.Contains(text, "neutral") && s.NeutralPct == 0 {
				s.NeutralPct = v
			} else if strings.Contains(text, "negative") && s.NegativePct == 0 {
				s.NegativePct = v
			}
		})
	}

	return s, nil
}
