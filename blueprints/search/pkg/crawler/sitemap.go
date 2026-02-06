package crawler

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// sitemapIndex represents a sitemap index file.
type sitemapIndex struct {
	XMLName  xml.Name      `xml:"sitemapindex"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

// sitemapEntry is an entry in a sitemap index.
type sitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// urlSet represents a standard urlset sitemap.
type urlSet struct {
	XMLName xml.Name  `xml:"urlset"`
	URLs    []siteURL `xml:"url"`
}

// siteURL is a single URL in a sitemap.
type siteURL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod"`
	ChangeFreq string  `xml:"changefreq"`
	Priority   float64 `xml:"priority"`
}

// SitemapURL is a parsed URL from a sitemap.
type SitemapURL struct {
	URL        string
	LastMod    time.Time
	ChangeFreq string
	Priority   float64
}

// FetchSitemap fetches and parses a sitemap URL, following sitemap index files.
// Returns all discovered URLs.
func FetchSitemap(client *http.Client, sitemapURL string, maxURLs int) ([]SitemapURL, error) {
	resp, err := client.Get(sitemapURL)
	if err != nil {
		return nil, fmt.Errorf("fetching sitemap %s: %w", sitemapURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("sitemap %s returned status %d", sitemapURL, resp.StatusCode)
	}

	return parseSitemap(client, resp.Body, maxURLs)
}

func parseSitemap(client *http.Client, r io.Reader, maxURLs int) ([]SitemapURL, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading sitemap: %w", err)
	}

	// Try parsing as sitemap index first
	var idx sitemapIndex
	if err := xml.Unmarshal(body, &idx); err == nil && len(idx.Sitemaps) > 0 {
		return parseSitemapIndex(client, idx, maxURLs)
	}

	// Try as regular urlset
	var us urlSet
	if err := xml.Unmarshal(body, &us); err == nil && len(us.URLs) > 0 {
		return parseURLSet(us, maxURLs), nil
	}

	return nil, fmt.Errorf("unrecognized sitemap format")
}

func parseSitemapIndex(client *http.Client, idx sitemapIndex, maxURLs int) ([]SitemapURL, error) {
	var all []SitemapURL
	for _, entry := range idx.Sitemaps {
		if maxURLs > 0 && len(all) >= maxURLs {
			break
		}
		remaining := 0
		if maxURLs > 0 {
			remaining = maxURLs - len(all)
		}
		urls, err := FetchSitemap(client, entry.Loc, remaining)
		if err != nil {
			// Skip broken sub-sitemaps
			continue
		}
		all = append(all, urls...)
	}
	return all, nil
}

func parseURLSet(us urlSet, maxURLs int) []SitemapURL {
	var urls []SitemapURL
	for _, u := range us.URLs {
		if maxURLs > 0 && len(urls) >= maxURLs {
			break
		}
		su := SitemapURL{
			URL:        u.Loc,
			ChangeFreq: u.ChangeFreq,
			Priority:   u.Priority,
		}
		if u.LastMod != "" {
			if t, err := parseLastMod(u.LastMod); err == nil {
				su.LastMod = t
			}
		}
		urls = append(urls, su)
	}
	return urls
}

// parseLastMod tries multiple date formats for sitemap lastmod.
func parseLastMod(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable date: %s", s)
}
