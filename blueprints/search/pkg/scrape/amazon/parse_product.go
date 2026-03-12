package amazon

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseProduct parses an Amazon product detail page.
// Tries JSON-LD first, falls back to HTML selectors.
// Returns error only if both Title and ASIN are empty.
func ParseProduct(doc *goquery.Document, asin, pageURL string) (*Product, error) {
	p := &Product{
		ASIN:      asin,
		URL:       pageURL,
		FetchedAt: time.Now(),
		Specs:     make(map[string]string),
	}

	// 1. Try JSON-LD
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, s *goquery.Selection) {
		if p.Title != "" {
			return
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(s.Text()), &raw); err != nil {
			return
		}
		if raw["@type"] != "Product" {
			return
		}
		if v, ok := raw["name"].(string); ok {
			p.Title = strings.TrimSpace(v)
		}
		if v, ok := raw["description"].(string); ok {
			p.Description = strings.TrimSpace(v)
		}
		if brand, ok := raw["brand"].(map[string]interface{}); ok {
			if v, ok := brand["name"].(string); ok {
				p.Brand = strings.TrimSpace(v)
			}
		}
		if offers, ok := raw["offers"].(map[string]interface{}); ok {
			if v, ok := offers["price"].(string); ok {
				p.Price = parsePrice(v)
			} else if v, ok := offers["price"].(float64); ok {
				p.Price = v
			}
			if v, ok := offers["priceCurrency"].(string); ok {
				p.Currency = v
			}
		}
		if agg, ok := raw["aggregateRating"].(map[string]interface{}); ok {
			if v, ok := agg["ratingValue"].(string); ok {
				p.Rating = parseFloatStr(v)
			} else if v, ok := agg["ratingValue"].(float64); ok {
				p.Rating = v
			}
			if v, ok := agg["reviewCount"].(string); ok {
				p.RatingsCount = parseInt64Str(v)
			} else if v, ok := agg["reviewCount"].(float64); ok {
				p.RatingsCount = int64(v)
			}
		}
		if img, ok := raw["image"].(string); ok && img != "" {
			p.Images = append(p.Images, img)
		}
	})

	// 2. HTML fallbacks
	if p.Title == "" {
		p.Title = strings.TrimSpace(doc.Find("#productTitle").Text())
	}

	if p.Price == 0 {
		priceStr := strings.TrimSpace(doc.Find(".a-price .a-offscreen").First().Text())
		p.Price = parsePrice(priceStr)
		if p.Currency == "" && strings.HasPrefix(priceStr, "$") {
			p.Currency = "USD"
		}
	}

	if p.ListPrice == 0 {
		lp := strings.TrimSpace(doc.Find(".basisPrice .a-offscreen").First().Text())
		if lp == "" {
			lp = strings.TrimSpace(doc.Find(".a-text-strike").First().Text())
		}
		p.ListPrice = parsePrice(lp)
	}

	if p.Rating == 0 {
		ratingAttr, _ := doc.Find("#acrPopover").Attr("title")
		// e.g. "4.5 out of 5 stars"
		if parts := strings.Fields(ratingAttr); len(parts) > 0 {
			p.Rating = parseFloatStr(parts[0])
		}
	}

	if p.RatingsCount == 0 {
		rcText := strings.TrimSpace(doc.Find("#acrCustomerReviewText").First().Text())
		p.RatingsCount = parseInt64Digits(rcText)
	}

	if p.ReviewsCount == 0 {
		rvText := strings.TrimSpace(doc.Find("#acrCustomerReviewLink").First().Text())
		if rvText == "" {
			rvText = strings.TrimSpace(doc.Find("#acrCustomerReviewText").First().Text())
		}
		p.ReviewsCount = parseInt64Digits(rvText)
	}

	// Answered questions count
	qaText := strings.TrimSpace(doc.Find("#askATFLink").Text())
	if qaText != "" {
		p.AnsweredQs = int(parseInt64Digits(qaText))
	}

	// Availability
	p.Availability = strings.TrimSpace(doc.Find("#availability span").First().Text())

	// Description (HTML fallback)
	if p.Description == "" {
		p.Description = strings.TrimSpace(doc.Find("#productDescription p").Text())
		if p.Description == "" {
			p.Description = strings.TrimSpace(doc.Find("#feature-bullets").Text())
		}
	}

	// Bullet points
	doc.Find("#feature-bullets ul li span.a-list-item").Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t == "" || strings.EqualFold(t, "About this item") {
			return
		}
		p.BulletPoints = append(p.BulletPoints, t)
	})

	// Specs from tech spec table
	doc.Find("#productDetails_techSpec_section_1 tr").Each(func(_ int, s *goquery.Selection) {
		key := strings.TrimSpace(s.Find("th").Text())
		val := strings.TrimSpace(s.Find("td").Text())
		if key != "" && val != "" {
			p.Specs[key] = val
		}
	})
	// Specs from detail bullets
	doc.Find("#detailBullets_feature_div li").Each(func(_ int, s *goquery.Selection) {
		spans := s.Find("span")
		if spans.Length() >= 2 {
			key := strings.TrimSpace(spans.Eq(0).Text())
			val := strings.TrimSpace(spans.Eq(1).Text())
			key = strings.Trim(key, ": \u200f\u200e")
			if key != "" && val != "" {
				p.Specs[key] = val
			}
		}
	})

	// Images from imgBlkFront data-a-dynamic-image
	if p.Images == nil {
		imgData, exists := doc.Find("#imgBlkFront").Attr("data-a-dynamic-image")
		if !exists {
			imgData, _ = doc.Find("#landingImage").Attr("data-a-dynamic-image")
		}
		if imgData != "" {
			var imgMap map[string]interface{}
			if err := json.Unmarshal([]byte(imgData), &imgMap); err == nil {
				for k := range imgMap {
					p.Images = append(p.Images, k)
				}
			}
		}
	}

	// Category path / breadcrumb
	doc.Find("#wayfinding-breadcrumbs_feature_div li a").Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			p.CategoryPath = append(p.CategoryPath, t)
		}
	})

	// Seller info
	sellerSel := doc.Find("#sellerProfileTriggerId")
	p.SellerName = strings.TrimSpace(sellerSel.Text())
	if href, exists := sellerSel.Attr("href"); exists {
		p.SellerID = extractQueryParam(href, "seller")
	}

	// SoldBy / FulfilledBy
	soldByText := strings.TrimSpace(doc.Find("#merchant-info").Text())
	if soldByText == "" {
		soldByText = strings.TrimSpace(doc.Find("#shipsFromSoldBy").Text())
	}
	p.SoldBy = soldByText

	// Best sellers rank
	doc.Find("#SalesRank").Each(func(_ int, s *goquery.Selection) {
		t := s.Text()
		p.Rank, p.RankCategory = parseBSR(t)
	})
	if p.Rank == 0 {
		doc.Find("#detailBullets_feature_div li").Each(func(_ int, s *goquery.Selection) {
			t := s.Text()
			if strings.Contains(t, "Best Sellers Rank") {
				p.Rank, p.RankCategory = parseBSR(t)
			}
		})
	}

	// Variants from swatch selectors
	seen := make(map[string]bool)
	doc.Find("li.swatchSelect a[href*='/dp/']").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if a := ExtractASIN(href); a != "" && a != asin && !seen[a] {
			seen[a] = true
			p.VariantASINs = append(p.VariantASINs, a)
		}
	})
	doc.Find(`div[data-dp-url*="/dp/"]`).Each(func(_ int, s *goquery.Selection) {
		dpURL, _ := s.Attr("data-dp-url")
		if a := ExtractASIN(dpURL); a != "" && a != asin && !seen[a] {
			seen[a] = true
			p.VariantASINs = append(p.VariantASINs, a)
		}
	})

	// Parent ASIN
	if val, exists := doc.Find("#landingAsin").Attr("value"); exists && val != "" {
		p.ParentASIN = val
	}
	if p.ParentASIN == "" {
		if val, exists := doc.Find("#ppd").Attr("data-asin"); exists && val != "" && val != asin {
			p.ParentASIN = val
		}
	}

	// Similar ASINs
	seenSim := make(map[string]bool)
	doc.Find(`#similarity-widget .a-link-normal[href*="/dp/"]`).Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if a := ExtractASIN(href); a != "" && a != asin && !seenSim[a] && len(p.SimilarASINs) < 6 {
			seenSim[a] = true
			p.SimilarASINs = append(p.SimilarASINs, a)
		}
	})

	// BrowseNodeIDs from nav-subnav
	if nodeAttr, exists := doc.Find("#nav-subnav").Attr("data-node"); exists && nodeAttr != "" {
		p.BrowseNodeIDs = []string{nodeAttr}
	}

	// ASIN fallback from page itself
	if p.ASIN == "" {
		if val, exists := doc.Find("#ASIN").Attr("value"); exists && val != "" {
			p.ASIN = val
		}
	}

	if p.Title == "" && p.ASIN == "" {
		return nil, fmt.Errorf("product page: both title and ASIN are empty")
	}

	return p, nil
}

// parseBSR extracts rank integer and category string from Best Sellers Rank text.
// e.g. "#1,234 in Books" → (1234, "Books")
func parseBSR(text string) (int, string) {
	idx := strings.Index(text, "#")
	if idx < 0 {
		return 0, ""
	}
	rest := text[idx+1:]
	// collect digits (and comma) up to first space
	var rankStr strings.Builder
	for _, ch := range rest {
		if ch >= '0' && ch <= '9' {
			rankStr.WriteRune(ch)
		} else if ch == ',' {
			// skip commas
		} else {
			break
		}
	}
	rank := int(parseInt64Str(rankStr.String()))

	// category: text after " in "
	cat := ""
	if inIdx := strings.Index(text, " in "); inIdx >= 0 {
		cat = strings.TrimSpace(text[inIdx+4:])
		// trim parenthetical / trailing content
		if pIdx := strings.Index(cat, "("); pIdx > 0 {
			cat = strings.TrimSpace(cat[:pIdx])
		}
	}
	return rank, cat
}
