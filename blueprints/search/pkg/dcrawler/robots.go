package dcrawler

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
	robotsURL := fmt.Sprintf("https://%s/robots.txt", domain)
	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		return &RobotsChecker{}, nil
	}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return &RobotsChecker{}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return &RobotsChecker{}, nil
	}

	robots, err := robotstxt.FromBytes(body)
	if err != nil {
		return &RobotsChecker{}, nil
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
	return rc, nil
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
