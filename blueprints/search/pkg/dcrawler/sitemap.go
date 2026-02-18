package dcrawler

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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
// Uses concurrent fetching for sitemap indexes with many child sitemaps.
func DiscoverSitemapURLs(ctx context.Context, client *http.Client, domain string, robotsSitemaps []string, maxURLs int) ([]string, error) {
	if maxURLs <= 0 {
		maxURLs = 1_000_000
	}

	candidates := make([]string, 0, len(robotsSitemaps)+2)
	seen := make(map[string]bool)

	// Add robots.txt sitemaps first (highest priority — may contain non-standard paths)
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

	var mu sync.Mutex
	var urls []string
	seenURLs := make(map[string]bool)
	var urlCount atomic.Int64

	addURLs := func(newURLs []string) {
		mu.Lock()
		defer mu.Unlock()
		for _, loc := range newURLs {
			if loc != "" && !seenURLs[loc] && len(urls) < maxURLs {
				seenURLs[loc] = true
				urls = append(urls, loc)
				urlCount.Store(int64(len(urls)))
			}
		}
	}

	// fetchAndParse fetches a single sitemap and returns its URLs.
	fetchAndParse := func(sitemapURL string) []string {
		body, err := fetchSitemap(ctx, client, sitemapURL)
		if err != nil || len(body) == 0 {
			return nil
		}
		var us urlSet
		if err := xml.Unmarshal(body, &us); err != nil {
			return nil
		}
		result := make([]string, 0, len(us.URLs))
		for _, u := range us.URLs {
			loc := strings.TrimSpace(u.Loc)
			if loc != "" {
				result = append(result, loc)
			}
		}
		return result
	}

	for _, candidate := range candidates {
		if ctx.Err() != nil || urlCount.Load() >= int64(maxURLs) {
			break
		}

		body, err := fetchSitemap(ctx, client, candidate)
		if err != nil || len(body) == 0 {
			continue
		}

		if isSitemapIndexXML(body) {
			// Parse the index to get child sitemap URLs
			var idx sitemapIndex
			if err := xml.Unmarshal(body, &idx); err != nil {
				continue
			}

			// Collect child sitemaps not yet seen
			var childSitemaps []string
			for _, sm := range idx.Sitemaps {
				loc := strings.TrimSpace(sm.Loc)
				if loc != "" && !seen[loc] {
					seen[loc] = true
					childSitemaps = append(childSitemaps, loc)
				}
			}

			if len(childSitemaps) == 0 {
				continue
			}

			// Fetch child sitemaps concurrently (up to 20 at a time)
			concurrency := 20
			if len(childSitemaps) < concurrency {
				concurrency = len(childSitemaps)
			}
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup

			for _, childURL := range childSitemaps {
				if ctx.Err() != nil || urlCount.Load() >= int64(maxURLs) {
					break
				}

				wg.Add(1)
				sem <- struct{}{}
				go func(u string) {
					defer wg.Done()
					defer func() { <-sem }()

					childURLs := fetchAndParse(u)
					if len(childURLs) > 0 {
						addURLs(childURLs)
					}
				}(childURL)
			}
			wg.Wait()
		} else {
			// Direct URL set
			var us urlSet
			if err := xml.Unmarshal(body, &us); err != nil {
				continue
			}
			result := make([]string, 0, len(us.URLs))
			for _, u := range us.URLs {
				loc := strings.TrimSpace(u.Loc)
				if loc != "" {
					result = append(result, loc)
				}
			}
			addURLs(result)
		}
	}

	return urls, nil
}

func fetchSitemap(ctx context.Context, client *http.Client, sitemapURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", sitemapURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
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
