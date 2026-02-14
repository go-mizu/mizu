package dcrawler

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// sitemapIndex represents a <sitemapindex> root element.
type sitemapIndex struct {
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc string `xml:"loc"`
}

// urlSet represents a <urlset> root element.
type urlSet struct {
	URLs []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc string `xml:"loc"`
}

// DiscoverSitemapURLs fetches and parses sitemap.xml (and sitemap indexes)
// to discover seed URLs. Returns up to maxURLs URLs.
func DiscoverSitemapURLs(ctx context.Context, client *http.Client, domain string, robotsSitemaps []string, maxURLs int) ([]string, error) {
	if maxURLs <= 0 {
		maxURLs = 1_000_000
	}

	candidates := make([]string, 0, len(robotsSitemaps)+2)
	seen := make(map[string]bool)

	// Add robots.txt sitemaps first (highest priority)
	for _, s := range robotsSitemaps {
		if !seen[s] {
			seen[s] = true
			candidates = append(candidates, s)
		}
	}
	// Add well-known locations (including gzipped variants)
	for _, suffix := range []string{"/sitemap.xml", "/sitemap.xml.gz", "/sitemap_index.xml", "/sitemap_index.xml.gz"} {
		u := fmt.Sprintf("https://%s%s", domain, suffix)
		if !seen[u] {
			seen[u] = true
			candidates = append(candidates, u)
		}
	}

	var urls []string
	seenURLs := make(map[string]bool)

	var discover func(sitemapURL string, depth int)
	discover = func(sitemapURL string, depth int) {
		if depth > 3 || len(urls) >= maxURLs {
			return
		}
		body, err := fetchSitemap(ctx, client, sitemapURL)
		if err != nil || len(body) == 0 {
			return
		}

		if isSitemapIndexXML(body) {
			var idx sitemapIndex
			if err := xml.Unmarshal(body, &idx); err != nil {
				return
			}
			for _, sm := range idx.Sitemaps {
				loc := strings.TrimSpace(sm.Loc)
				if loc != "" && !seen[loc] {
					seen[loc] = true
					discover(loc, depth+1)
				}
			}
		} else {
			var us urlSet
			if err := xml.Unmarshal(body, &us); err != nil {
				return
			}
			for _, u := range us.URLs {
				loc := strings.TrimSpace(u.Loc)
				if loc != "" && !seenURLs[loc] && len(urls) < maxURLs {
					seenURLs[loc] = true
					urls = append(urls, loc)
				}
			}
		}
	}

	for _, c := range candidates {
		discover(c, 0)
		if len(urls) >= maxURLs {
			break
		}
	}
	return urls, nil
}

func fetchSitemap(ctx context.Context, client *http.Client, sitemapURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", sitemapURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/xml, text/xml, */*")
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	if strings.HasSuffix(sitemapURL, ".gz") || resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip: %w", err)
		}
		defer gr.Close()
		reader = gr
	}
	return io.ReadAll(io.LimitReader(reader, 50*1024*1024)) // 50MB max
}

func isSitemapIndexXML(body []byte) bool {
	// Quick check: look for <sitemapindex in the first 500 bytes
	prefix := body
	if len(prefix) > 500 {
		prefix = prefix[:500]
	}
	return strings.Contains(strings.ToLower(string(prefix)), "<sitemapindex")
}
