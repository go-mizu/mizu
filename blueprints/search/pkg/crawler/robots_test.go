package crawler

import (
	"strings"
	"testing"
	"time"
)

func TestParseRobots(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /private/
Disallow: /admin/
Allow: /admin/public/
Crawl-delay: 2

User-agent: MizuCrawler
Disallow: /secret/
Allow: /secret/ok/
Crawl-delay: 5

Sitemap: https://example.com/sitemap.xml
Sitemap: https://example.com/sitemap2.xml
`

	data := ParseRobots(strings.NewReader(robotsTxt), "MizuCrawler/1.0")

	// Should use MizuCrawler-specific rules
	if len(data.Disallowed) != 1 || data.Disallowed[0] != "/secret/" {
		t.Errorf("Disallowed = %v, want [/secret/]", data.Disallowed)
	}
	if len(data.Allowed) != 1 || data.Allowed[0] != "/secret/ok/" {
		t.Errorf("Allowed = %v, want [/secret/ok/]", data.Allowed)
	}
	if data.CrawlDelay != 5*time.Second {
		t.Errorf("CrawlDelay = %v, want 5s", data.CrawlDelay)
	}
	if len(data.Sitemaps) != 2 {
		t.Errorf("Sitemaps = %v, want 2 entries", data.Sitemaps)
	}
}

func TestParseRobotsWildcard(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /private/
Crawl-delay: 3
`

	data := ParseRobots(strings.NewReader(robotsTxt), "SomeOtherBot/1.0")

	if len(data.Disallowed) != 1 || data.Disallowed[0] != "/private/" {
		t.Errorf("Disallowed = %v, want [/private/]", data.Disallowed)
	}
	if data.CrawlDelay != 3*time.Second {
		t.Errorf("CrawlDelay = %v, want 3s", data.CrawlDelay)
	}
}

func TestIsPathAllowed(t *testing.T) {
	data := &RobotsData{
		Disallowed: []string{"/private/", "/admin/"},
		Allowed:    []string{"/admin/public/"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/page", true},
		{"/private/secret", false},
		{"/admin/dashboard", false},
		{"/admin/public/page", true}, // allowed overrides disallowed
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isPathAllowed(tt.path, data)
			if got != tt.want {
				t.Errorf("isPathAllowed(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchRobotsPattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"/page", "/page", true},
		{"/page/sub", "/page", true},
		{"/other", "/page", false},
		{"/page.html", "/page$", false},
		{"/page", "/page$", true},
		{"/search?q=test", "/search", true},
		{"/files/document.pdf", "/*.pdf$", true},
		{"/files/document.pdf", "/*.pdf", true},
		{"/files/document.txt", "/*.pdf$", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchRobotsPattern(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchRobotsPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}
