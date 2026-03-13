package ebay

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	pricePattern         = regexp.MustCompile(`([0-9][0-9,]*\.?[0-9]{0,2})`)
	positivePctPattern   = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)%\s*positive`)
	feedbackScorePattern = regexp.MustCompile(`([0-9][0-9,]*)\s+feedback`)
)

type productJSONLD struct {
	Type        string `json:"@type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       any    `json:"image"`
	Offers      any    `json:"offers"`
	Category    string `json:"category"`
	Brand       any    `json:"brand"`
}

type offerJSONLD struct {
	Price         string `json:"price"`
	PriceCurrency string `json:"priceCurrency"`
	Availability  string `json:"availability"`
	ItemCondition string `json:"itemCondition"`
}

// ParseItem parses an eBay item page and discovers related item URLs.
func ParseItem(doc *goquery.Document, pageURL string) (*Item, []string, error) {
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if strings.Contains(strings.ToLower(title), "pardon our interruption") {
		return nil, nil, fmt.Errorf("challenge page")
	}

	item := &Item{
		URL:           pageURL,
		ItemID:        ExtractItemID(pageURL),
		ItemSpecifics: map[string]string{},
		FetchedAt:     time.Now(),
	}

	if canonical, ok := doc.Find(`link[rel="canonical"]`).Attr("href"); ok && canonical != "" {
		item.URL = NormalizeItemURL(canonical)
		if item.ItemID == "" {
			item.ItemID = ExtractItemID(item.URL)
		}
	}

	if product, rawJSONLD := extractProductJSONLD(doc); product != nil {
		item.RawJSONLD = rawJSONLD
		if item.Title == "" {
			item.Title = normalizeSpace(product.Name)
		}
		if item.Description == "" {
			item.Description = normalizeSpace(stripHTML(product.Description))
		}
		item.ImageURLs = appendUniqueStrings(item.ImageURLs, imagesFromAny(product.Image)...)
		if item.Price == 0 || item.Currency == "" || item.Availability == "" || item.Condition == "" {
			if offer, ok := firstOffer(product.Offers); ok {
				if item.Price == 0 {
					item.Price = parsePrice(offer.Price)
				}
				if item.Currency == "" {
					item.Currency = strings.TrimSpace(offer.PriceCurrency)
				}
				if item.Availability == "" {
					item.Availability = simplifySchemaValue(offer.Availability)
				}
				if item.Condition == "" {
					item.Condition = simplifySchemaValue(offer.ItemCondition)
				}
			}
		}
		if product.Category != "" {
			item.CategoryPath = appendUniqueStrings(item.CategoryPath, product.Category)
		}
	}

	if item.Title == "" {
		item.Title = firstNonEmpty(
			attrValue(doc, `meta[property="og:title"]`, "content"),
			textValue(doc, "h1.x-item-title__mainTitle span.ux-textspans"),
			textValue(doc, `div[data-testid="x-item-title"] span.ux-textspans`),
			strings.TrimSuffix(title, " | eBay"),
		)
	}
	item.Title = normalizeTitle(item.Title)

	if item.Subtitle == "" {
		item.Subtitle = firstNonEmpty(
			textValue(doc, ".x-item-title__subTitle span.ux-textspans"),
			textValue(doc, `div[data-testid="x-item-subtitle"] span.ux-textspans`),
		)
	}
	if item.Description == "" {
		item.Description = firstNonEmpty(
			attrValue(doc, `meta[name="description"]`, "content"),
			textValue(doc, ".d-item-description"),
		)
	}
	if item.Price == 0 {
		item.Price = parsePrice(firstNonEmpty(
			attrValue(doc, `meta[itemprop="price"]`, "content"),
			textValue(doc, ".x-price-primary span.ux-textspans"),
			textValue(doc, `div[data-testid="x-price-primary"] span.ux-textspans`),
			textValue(doc, ".display-price"),
		))
	}
	if item.Currency == "" {
		item.Currency = firstNonEmpty(
			attrValue(doc, `meta[itemprop="priceCurrency"]`, "content"),
			attrValue(doc, `meta[property="product:price:currency"]`, "content"),
		)
	}
	if item.OriginalPrice == 0 {
		item.OriginalPrice = parsePrice(firstNonEmpty(
			textValue(doc, ".ux-textspans--STRIKETHROUGH"),
			textValue(doc, ".x-price-transparency span.ux-textspans"),
		))
	}
	if item.Condition == "" {
		item.Condition = firstNonEmpty(
			textValue(doc, ".x-item-condition-text span.ux-textspans"),
			textValue(doc, ".u-flL.condText"),
		)
	}

	textBlobs := collectTextBlobs(doc)
	for _, blob := range textBlobs {
		lower := strings.ToLower(blob)
		switch {
		case item.ShippingText == "" && (strings.Contains(lower, "shipping") || strings.Contains(lower, "delivery")):
			item.ShippingText = blob
		case item.ReturnsText == "" && strings.Contains(lower, "return"):
			item.ReturnsText = blob
		case item.Location == "" && strings.HasPrefix(lower, "located in:"):
			item.Location = strings.TrimSpace(strings.TrimPrefix(blob, "Located in:"))
		}
		if item.SellerPositivePct == 0 {
			if m := positivePctPattern.FindStringSubmatch(lower); len(m) == 2 {
				item.SellerPositivePct, _ = strconv.ParseFloat(m[1], 64)
			}
		}
		if item.SellerFeedbackScore == 0 {
			if m := feedbackScorePattern.FindStringSubmatch(lower); len(m) == 2 {
				item.SellerFeedbackScore = parseInt64(m[1])
			}
		}
	}

	if item.SellerName == "" || item.SellerURL == "" {
		doc.Find(`a[href*="/str/"], a[href*="feedback_profile"], a[href*="/usr/"]`).Each(func(_ int, s *goquery.Selection) {
			if item.SellerName != "" && item.SellerURL != "" {
				return
			}
			name := normalizeSpace(s.Text())
			href, _ := s.Attr("href")
			if name == "" || href == "" {
				return
			}
			item.SellerName = name
			item.SellerURL = absoluteURL(BaseURL, href)
		})
	}

	if len(item.ImageURLs) == 0 {
		item.ImageURLs = appendUniqueStrings(item.ImageURLs,
			attrValue(doc, `meta[property="og:image"]`, "content"),
			attrValue(doc, `link[rel="image_src"]`, "href"),
		)
	}

	doc.Find(`nav[aria-label*="Listing"] a, .seo-breadcrumbs a`).Each(func(_ int, s *goquery.Selection) {
		text := normalizeSpace(s.Text())
		if text != "" {
			item.CategoryPath = appendUniqueStrings(item.CategoryPath, text)
		}
	})

	doc.Find(".ux-labels-values").Each(func(_ int, s *goquery.Selection) {
		key := normalizeSpace(s.Find(".ux-labels-values__labels-content .ux-textspans").First().Text())
		val := normalizeSpace(s.Find(".ux-labels-values__values-content .ux-textspans").Text())
		if key == "" || val == "" {
			return
		}
		key = strings.TrimSuffix(key, ":")
		if _, exists := item.ItemSpecifics[key]; !exists {
			item.ItemSpecifics[key] = val
		}
	})

	if item.ItemID == "" {
		return nil, nil, fmt.Errorf("could not extract item ID")
	}
	if item.Title == "" {
		return nil, nil, fmt.Errorf("could not extract item title")
	}

	related := collectRelatedItemURLs(doc, item.ItemID)
	return item, related, nil
}

func extractProductJSONLD(doc *goquery.Document) (*productJSONLD, string) {
	var (
		result *productJSONLD
		raw    string
	)
	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		payload := strings.TrimSpace(s.Text())
		if payload == "" {
			return true
		}
		var anyVal any
		if err := json.Unmarshal([]byte(payload), &anyVal); err != nil {
			return true
		}
		if p := findProduct(anyVal); p != nil {
			result = p
			raw = payload
			return false
		}
		return true
	})
	return result, raw
}

func findProduct(v any) *productJSONLD {
	switch x := v.(type) {
	case map[string]any:
		if typ := strings.ToLower(fmt.Sprint(x["@type"])); typ == "product" {
			var p productJSONLD
			b, _ := json.Marshal(x)
			if err := json.Unmarshal(b, &p); err == nil {
				return &p
			}
		}
		if graph, ok := x["@graph"]; ok {
			return findProduct(graph)
		}
	case []any:
		for _, el := range x {
			if p := findProduct(el); p != nil {
				return p
			}
		}
	}
	return nil
}

func firstOffer(v any) (offerJSONLD, bool) {
	switch x := v.(type) {
	case map[string]any:
		var offer offerJSONLD
		b, _ := json.Marshal(x)
		if err := json.Unmarshal(b, &offer); err == nil {
			return offer, true
		}
	case []any:
		for _, el := range x {
			if offer, ok := firstOffer(el); ok {
				return offer, true
			}
		}
	}
	return offerJSONLD{}, false
}

func imagesFromAny(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		out := make([]string, 0, len(x))
		for _, el := range x {
			if s, ok := el.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func simplifySchemaValue(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "/"); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.ReplaceAll(s, "_", " ")
	return normalizeSpace(s)
}

func collectRelatedItemURLs(doc *goquery.Document, selfID string) []string {
	seen := map[string]struct{}{}
	var out []string
	doc.Find(`a[href*="/itm/"]`).Each(func(_ int, s *goquery.Selection) {
		if len(out) >= 20 {
			return
		}
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		u := NormalizeItemURL(absoluteURL(BaseURL, href))
		id := ExtractItemID(u)
		if id == "" || id == selfID {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		out = append(out, u)
	})
	return out
}

func collectTextBlobs(doc *goquery.Document) []string {
	var out []string
	doc.Find(".ux-textspans, .ux-labels-values__values-content, .ux-layout-section__textual-display").Each(func(_ int, s *goquery.Selection) {
		text := normalizeSpace(s.Text())
		if text != "" {
			out = append(out, text)
		}
	})
	return out
}

func textValue(doc *goquery.Document, selector string) string {
	return normalizeSpace(doc.Find(selector).First().Text())
}

func attrValue(doc *goquery.Document, selector, attr string) string {
	v, _ := doc.Find(selector).First().Attr(attr)
	return strings.TrimSpace(v)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parsePrice(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	m := pricePattern.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0
	}
	f, _ := strconv.ParseFloat(m[1], 64)
	return f
}

func parseInt64(s string) int64 {
	s = strings.ReplaceAll(s, ",", "")
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func appendUniqueStrings(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, v := range dst {
		seen[v] = struct{}{}
	}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, exists := seen[v]; exists {
			continue
		}
		seen[v] = struct{}{}
		dst = append(dst, v)
	}
	return dst
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func normalizeTitle(s string) string {
	s = normalizeSpace(strings.TrimSuffix(s, "| eBay"))
	return strings.TrimSuffix(s, " | eBay")
}

func stripHTML(s string) string {
	r := strings.NewReplacer("<br>", " ", "<br/>", " ", "<br />", " ")
	s = r.Replace(s)
	for {
		start := strings.IndexByte(s, '<')
		if start < 0 {
			break
		}
		end := strings.IndexByte(s[start:], '>')
		if end < 0 {
			break
		}
		s = s[:start] + " " + s[start+end+1:]
	}
	return normalizeSpace(s)
}
