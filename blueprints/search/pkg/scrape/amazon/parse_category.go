package amazon

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseCategory parses an Amazon browse node / category page (/b?node=<id>).
func ParseCategory(doc *goquery.Document, pageURL string) (*Category, error) {
	c := &Category{
		URL:       pageURL,
		FetchedAt: time.Now(),
	}

	// NodeID from URL query param "node"
	c.NodeID = extractNodeID(pageURL)

	// Name: first nav-subnav item, then h1, then #zg_listTitle
	c.Name = strings.TrimSpace(doc.Find("#nav-subnav .nav-a-content").First().Text())
	if c.Name == "" {
		c.Name = strings.TrimSpace(doc.Find("h1").First().Text())
	}
	if c.Name == "" {
		c.Name = strings.TrimSpace(doc.Find("#zg_listTitle").Text())
	}

	// ParentNodeID from nav-subnav attribute
	if val, exists := doc.Find("#nav-subnav").Attr("data-parent-node-id"); exists && val != "" {
		c.ParentNodeID = strings.TrimSpace(val)
	}
	// Fallback: last breadcrumb link's node= param
	if c.ParentNodeID == "" {
		links := doc.Find("#wayfinding-breadcrumbs_feature_div li a")
		links.Each(func(i int, s *goquery.Selection) {
			if i == links.Length()-1 {
				href, _ := s.Attr("href")
				if n := extractNodeID(href); n != "" && n != c.NodeID {
					c.ParentNodeID = n
				}
			}
		})
	}

	// Breadcrumb
	doc.Find("#wayfinding-breadcrumbs_feature_div li a").Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			c.Breadcrumb = append(c.Breadcrumb, t)
		}
	})

	// ChildNodeIDs from nav-subnav links (exclude current node)
	seen := make(map[string]bool)
	doc.Find(`#nav-subnav .nav-a[href*="node="]`).Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		n := extractNodeID(href)
		if n == "" || n == c.NodeID || seen[n] {
			return
		}
		seen[n] = true
		c.ChildNodeIDs = append(c.ChildNodeIDs, n)
	})

	// TopASINs from search result cards (first 12)
	doc.Find(`[data-component-type="s-search-result"]`).Each(func(_ int, s *goquery.Selection) {
		if len(c.TopASINs) >= 12 {
			return
		}
		if asin, exists := s.Attr("data-asin"); exists && asin != "" {
			c.TopASINs = append(c.TopASINs, asin)
		}
	})
	if len(c.TopASINs) == 0 {
		doc.Find(`#zg-ordered-list a[href*="/dp/"], div[id^="gridItemRoot"] a[href*="/dp/"]`).Each(func(_ int, s *goquery.Selection) {
			if len(c.TopASINs) >= 12 {
				return
			}
			href, _ := s.Attr("href")
			if asin := ExtractASIN(href); asin != "" {
				c.TopASINs = append(c.TopASINs, asin)
			}
		})
	}
	c.Breadcrumb = dedup(c.Breadcrumb)
	c.ChildNodeIDs = dedup(c.ChildNodeIDs)
	c.TopASINs = dedup(c.TopASINs)

	return c, nil
}

// extractNodeID extracts the "node" query parameter from a URL string.
func extractNodeID(rawURL string) string {
	return extractQueryParam(rawURL, "node")
}
