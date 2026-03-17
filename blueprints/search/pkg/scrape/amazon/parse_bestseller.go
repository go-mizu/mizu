package amazon

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseBestseller parses /bestsellers/<cat>, /new-releases/<cat>, etc.
// Returns the list record and individual ranked entries.
func ParseBestseller(doc *goquery.Document, listType, category, nodeID, pageURL string) (*BestsellerList, []BestsellerEntry, error) {
	today := time.Now().Format("2006-01-02")

	list := &BestsellerList{
		ListID:       listType + "/" + category + "/" + today,
		ListType:     listType,
		Category:     category,
		NodeID:       nodeID,
		SnapshotDate: today,
		URL:          pageURL,
		FetchedAt:    time.Now(),
	}

	var entries []BestsellerEntry

	// Try #zg-ordered-list li (classic layout), then #gridItemRoot divs (grid layout)
	doc.Find("#zg-ordered-list li, li.zg-item-immersion").Each(func(_ int, s *goquery.Selection) {
		entry := parseBestsellerItem(s, list.ListID)
		if entry.ASIN != "" || entry.Rank > 0 {
			entries = append(entries, entry)
		}
	})

	if len(entries) == 0 {
		// Grid layout: each grid cell is a direct div
		doc.Find(`div[id^="gridItemRoot"]`).Each(func(_ int, s *goquery.Selection) {
			entry := parseBestsellerItem(s, list.ListID)
			if entry.ASIN != "" || entry.Rank > 0 {
				entries = append(entries, entry)
			}
		})
	}

	return list, entries, nil
}

// parseBestsellerItem extracts one ranked entry from its container element.
func parseBestsellerItem(s *goquery.Selection, listID string) BestsellerEntry {
	entry := BestsellerEntry{ListID: listID}

	// Rank: .zg-badge-text or .zg-bdg-text, strip leading "#"
	rankText := strings.TrimSpace(s.Find(".zg-badge-text, .zg-bdg-text").First().Text())
	rankText = strings.TrimPrefix(rankText, "#")
	rankText = strings.ReplaceAll(rankText, ",", "")
	entry.Rank = int(parseInt64Str(rankText))

	// ASIN from first /dp/ link
	s.Find(`a[href*="/dp/"]`).Each(func(_ int, a *goquery.Selection) {
		if entry.ASIN != "" {
			return
		}
		href, _ := a.Attr("href")
		entry.ASIN = ExtractASIN(href)
	})

	// Title: try multiple selectors in order of specificity
	titleSels := []string{
		".p13n-sc-truncate-desktop-type2",
		".p13n-sc-line-clamp-2",
		"._cDEzb_p13n-sc-css-line-clamp-3_g3dy1",
		"a.a-link-normal",
	}
	for _, sel := range titleSels {
		t := strings.TrimSpace(s.Find(sel).First().Text())
		if t != "" {
			entry.Title = t
			break
		}
	}

	// Price: .a-price .a-offscreen
	priceStr := strings.TrimSpace(s.Find(".a-price .a-offscreen").First().Text())
	entry.Price = parsePrice(priceStr)

	// Rating: from star class, e.g. a-star-small-4-5
	s.Find("[class*='a-star-']").Each(func(_ int, star *goquery.Selection) {
		if entry.Rating != 0 {
			return
		}
		cls, _ := star.Attr("class")
		if r := starRatingFromClass(cls); r > 0 {
			entry.Rating = r
		}
	})

	// RatingsCount: small link text with a number
	s.Find(".a-size-small .a-link-normal, .a-size-small a").Each(func(_ int, a *goquery.Selection) {
		if entry.RatingsCount != 0 {
			return
		}
		t := strings.TrimSpace(a.Text())
		t = strings.ReplaceAll(t, ",", "")
		if v := parseInt64Str(t); v > 0 {
			entry.RatingsCount = v
		}
	})

	return entry
}
