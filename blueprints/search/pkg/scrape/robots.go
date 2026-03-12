package scrape

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/temoto/robotstxt"
)

// RobotsChecker checks paths against robots.txt rules.
type RobotsChecker struct {
	group      *robotstxt.Group
	crawlDelay time.Duration
	sitemaps   []string
}

// FetchRobots downloads and parses robots.txt for the given domain.
// Returns an allow-all checker if robots.txt is unavailable.
func FetchRobots(ctx context.Context, client *http.Client, domain string) (*RobotsChecker, error) {
	body := FetchRobotsRaw(ctx, client, domain)
	if body == nil {
		return &RobotsChecker{}, nil
	}
	return ParseRobotsBody(body), nil
}

// FetchRobotsRaw downloads robots.txt and returns the raw body.
// Returns nil if robots.txt is unavailable.
func FetchRobotsRaw(ctx context.Context, client *http.Client, domain string) []byte {
	robotsURL := fmt.Sprintf("https://%s/robots.txt", domain)
	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil
	}
	return body
}

// ParseRobotsBody parses a robots.txt body into a RobotsChecker.
func ParseRobotsBody(body []byte) *RobotsChecker {
	robots, err := robotstxt.FromBytes(body)
	if err != nil {
		return &RobotsChecker{}
	}

	group := robots.FindGroup("MizuCrawler")
	if group == nil {
		group = robots.FindGroup("*")
	}

	sitemaps := extractSitemapDirectives(string(body))

	rc := &RobotsChecker{
		group:    group,
		sitemaps: sitemaps,
	}
	if group != nil {
		rc.crawlDelay = group.CrawlDelay
	}
	return rc
}

// IsAllowed checks if a path is allowed by robots.txt.
func (r *RobotsChecker) IsAllowed(path string) bool {
	if r.group == nil {
		return true
	}
	return r.group.Test(path)
}

// CrawlDelay returns the crawl-delay directive value.
func (r *RobotsChecker) CrawlDelay() time.Duration {
	return r.crawlDelay
}

// Sitemaps returns sitemap URLs found in robots.txt.
func (r *RobotsChecker) Sitemaps() []string {
	return r.sitemaps
}

// extractSitemapDirectives parses "Sitemap:" lines from robots.txt body.
func extractSitemapDirectives(body string) []string {
	var sitemaps []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "sitemap:") {
			url := strings.TrimSpace(line[len("sitemap:"):])
			if url != "" {
				sitemaps = append(sitemaps, url)
			}
		}
	}
	return sitemaps
}
