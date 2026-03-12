package amazon

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseBrand parses an Amazon Brand Store page (/stores/<slug>).
func ParseBrand(doc *goquery.Document, pageURL string) (*Brand, error) {
	b := &Brand{
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// BrandID: slug after /stores/
	b.BrandID = extractPathSegmentAfter(pageURL, "/stores/")

	// Name: meta title, then h1, then #sc-brand-name
	if content, exists := doc.Find(`meta[name="title"]`).Attr("content"); exists {
		b.Name = strings.TrimSpace(content)
	}
	if b.Name == "" {
		b.Name = strings.TrimSpace(doc.Find("h1").First().Text())
	}
	if b.Name == "" {
		b.Name = strings.TrimSpace(doc.Find("#sc-brand-name").Text())
	}

	// Description: meta description, then .store-description
	if content, exists := doc.Find(`meta[name="description"]`).Attr("content"); exists {
		b.Description = strings.TrimSpace(content)
	}
	if b.Description == "" {
		b.Description = strings.TrimSpace(doc.Find(".store-description").Text())
	}

	// LogoURL: #sc-logo img src, then og:image
	if src, exists := doc.Find("#sc-logo img").Attr("src"); exists {
		b.LogoURL = strings.TrimSpace(src)
	}
	if b.LogoURL == "" {
		if content, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists {
			b.LogoURL = strings.TrimSpace(content)
		}
	}

	// BannerURL: first large hero image
	if src, exists := doc.Find("#sc-desktop-hero img").First().Attr("src"); exists {
		b.BannerURL = strings.TrimSpace(src)
	}

	// FeaturedASINs: all /dp/ links on page (deduplicated, max 20)
	seen := make(map[string]bool)
	doc.Find(`a[href*="/dp/"]`).Each(func(_ int, s *goquery.Selection) {
		if len(b.FeaturedASINs) >= 20 {
			return
		}
		href, _ := s.Attr("href")
		if a := ExtractASIN(href); a != "" && !seen[a] {
			seen[a] = true
			b.FeaturedASINs = append(b.FeaturedASINs, a)
		}
	})

	// FollowerCount: not publicly visible — leave as 0
	b.FollowerCount = 0

	return b, nil
}
