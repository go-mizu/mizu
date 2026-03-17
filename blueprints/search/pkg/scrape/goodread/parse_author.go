package goodread

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// authorJSONLD is the JSON-LD structure on author pages.
type authorJSONLD struct {
	Type        string `json:"@type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       struct {
		URL string `json:"url"`
	} `json:"image"`
	URL     string `json:"url"`
	SameAs  string `json:"sameAs"`
	BirthDate string `json:"birthDate"`
	DeathDate string `json:"deathDate"`
}

// ParseAuthor parses a Goodreads author page.
func ParseAuthor(doc *goquery.Document, authorID, pageURL string) (*Author, error) {
	a := &Author{
		AuthorID:  authorID,
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// JSON-LD — use .Text() to get unescaped script content
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, sel *goquery.Selection) {
		if a.Name != "" {
			return
		}
		var ld authorJSONLD
		if err := json.Unmarshal([]byte(sel.Text()), &ld); err != nil {
			return
		}
		if ld.Type != "Person" {
			return
		}
		a.Name = ld.Name
		a.Bio = cleanHTML(ld.Description)
		a.PhotoURL = ld.Image.URL
		a.Website = ld.SameAs
		a.BornDate = ld.BirthDate
		a.DiedDate = ld.DeathDate
	})

	// HTML enrichment
	if a.Name == "" {
		a.Name = strings.TrimSpace(doc.Find("h1.authorName, h1[class*='authorName'], [data-testid='name']").First().Text())
	}
	if a.Name == "" {
		a.Name = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	// Bio
	if a.Bio == "" {
		a.Bio = strings.TrimSpace(doc.Find("[data-testid='description'], .authorShortBio, div.aboutAuthorInfo").Text())
	}

	// Photo
	if a.PhotoURL == "" {
		a.PhotoURL, _ = doc.Find("img.authorPhoto, img[itemprop='image']").First().Attr("src")
	}

	// Website
	if a.Website == "" {
		doc.Find("a[href*='http']").Each(func(_ int, sel *goquery.Selection) {
			if a.Website != "" {
				return
			}
			href, _ := sel.Attr("href")
			if !strings.Contains(href, "goodreads.com") && strings.HasPrefix(href, "http") {
				a.Website = href
			}
		})
	}

	// Hometown
	doc.Find("[class*='hometown'], [data-testid='hometown']").Each(func(_ int, sel *goquery.Selection) {
		a.Hometown = strings.TrimSpace(sel.Text())
	})

	// Genres
	doc.Find("a[href*='/genres/']").Each(func(_ int, sel *goquery.Selection) {
		g := strings.TrimSpace(sel.Text())
		if g != "" && !contains(a.Genres, g) {
			a.Genres = append(a.Genres, g)
		}
	})

	// Influences
	doc.Find("a[href*='/author/show/']").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if text != "" && text != a.Name && !contains(a.Influences, text) && len(a.Influences) < 10 {
			a.Influences = append(a.Influences, text)
		}
	})

	// Stats — avg rating / ratings count / books count
	doc.Find("[data-testid='ratingsCount'], [class*='ratingsCount']").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		text = strings.ReplaceAll(text, ",", "")
		n, _ := strconv.ParseInt(text, 10, 64)
		if n > 0 {
			a.RatingsCount = n
		}
	})

	doc.Find("[class*='avgRating'], [data-testid='avgRating']").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		f, _ := strconv.ParseFloat(text, 64)
		if f > 0 {
			a.AvgRating = f
		}
	})

	return a, nil
}
