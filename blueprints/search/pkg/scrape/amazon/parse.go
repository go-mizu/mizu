package amazon

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var nonNumeric = regexp.MustCompile(`[^0-9.]`)

func ParseSearchResults(baseURL string, query string, page int, html []byte) ([]Product, bool, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return nil, false, err
	}

	products := make([]Product, 0, 24)
	position := 0
	doc.Find("div.s-main-slot div[data-component-type='s-search-result']").Each(func(_ int, s *goquery.Selection) {
		asin, _ := s.Attr("data-asin")
		asin = strings.TrimSpace(asin)
		if asin == "" {
			return
		}

		title := strings.TrimSpace(s.Find("h2 span").First().Text())
		href, _ := s.Find("h2 a").Attr("href")
		img, _ := s.Find("img.s-image").Attr("src")
		priceWhole := strings.TrimSpace(s.Find("span.a-price > span.a-offscreen").First().Text())
		if priceWhole == "" {
			whole := strings.TrimSpace(s.Find("span.a-price-whole").First().Text())
			frac := strings.TrimSpace(s.Find("span.a-price-fraction").First().Text())
			if whole != "" {
				priceWhole = whole
				if frac != "" {
					priceWhole += "." + frac
				}
			}
		}

		rating := parseFloat(s.Find("span.a-icon-alt").First().Text())
		reviews := parseCount(s.Find("span.a-size-base.s-underline-text").First().Text())
		badge := strings.TrimSpace(s.Find("span.a-badge-text").First().Text())
		lowerAll := strings.ToLower(s.Text())
		isPrime := strings.Contains(lowerAll, "prime") || s.Find("i[aria-label='Amazon Prime']").Length() > 0
		isSponsored := strings.Contains(lowerAll, "sponsored")

		u := resolveURL(baseURL, href)
		position++
		products = append(products, Product{
			Query:        query,
			ASIN:         asin,
			Title:        title,
			URL:          u,
			ImageURL:     img,
			PriceText:    priceWhole,
			PriceValue:   parsePrice(priceWhole),
			Currency:     detectCurrency(priceWhole),
			Rating:       rating,
			ReviewCount:  reviews,
			IsPrime:      isPrime,
			IsSponsored:  isSponsored,
			Badge:        badge,
			Position:     position,
			ResultPage:   page,
			RawContainer: strings.TrimSpace(s.Text()),
		})
	})

	hasNext := doc.Find("a.s-pagination-next").Length() > 0 && !strings.Contains(doc.Find("a.s-pagination-next").AttrOr("class", ""), "s-pagination-disabled")
	return products, hasNext, nil
}

func resolveURL(base, href string) string {
	if strings.TrimSpace(href) == "" {
		return ""
	}
	u, err := url.Parse(href)
	if err == nil && u.IsAbs() {
		return u.String()
	}
	b, err := url.Parse(base)
	if err != nil {
		return href
	}
	return b.ResolveReference(u).String()
}

func parsePrice(raw string) float64 {
	v := nonNumeric.ReplaceAllString(raw, "")
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

func parseFloat(raw string) float64 {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) == 0 {
		return 0
	}
	f, _ := strconv.ParseFloat(nonNumeric.ReplaceAllString(parts[0], ""), 64)
	return f
}

func parseCount(raw string) int {
	raw = strings.ReplaceAll(raw, ",", "")
	v := nonNumeric.ReplaceAllString(raw, "")
	n, _ := strconv.Atoi(v)
	return n
}

func detectCurrency(raw string) string {
	switch {
	case strings.Contains(raw, "$"):
		return "USD"
	case strings.Contains(raw, "£"):
		return "GBP"
	case strings.Contains(raw, "€"):
		return "EUR"
	case strings.Contains(raw, "¥"):
		return "JPY"
	default:
		return ""
	}
}

func ValidateSearchHTML(html []byte) error {
	if len(html) == 0 {
		return fmt.Errorf("empty response body")
	}
	if strings.Contains(strings.ToLower(string(html)), "captcha") {
		return fmt.Errorf("captcha detected")
	}
	return nil
}
