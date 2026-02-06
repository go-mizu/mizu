package crawler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RobotsCache fetches and caches robots.txt per domain.
type RobotsCache struct {
	mu      sync.RWMutex
	entries map[string]*RobotsData
	client  *http.Client
	ua      string
}

// RobotsData holds parsed robots.txt rules for a domain.
type RobotsData struct {
	Disallowed []string
	Allowed    []string
	CrawlDelay time.Duration
	Sitemaps   []string
}

// NewRobotsCache creates a new robots.txt cache.
func NewRobotsCache(client *http.Client, userAgent string) *RobotsCache {
	return &RobotsCache{
		entries: make(map[string]*RobotsData),
		client:  client,
		ua:      userAgent,
	}
}

// IsAllowed checks if a URL is allowed by robots.txt.
// Fetches and caches robots.txt on first access per domain.
func (rc *RobotsCache) IsAllowed(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	domain := u.Scheme + "://" + u.Host

	rc.mu.RLock()
	data, ok := rc.entries[domain]
	rc.mu.RUnlock()

	if !ok {
		data = rc.fetch(domain)
		rc.mu.Lock()
		rc.entries[domain] = data
		rc.mu.Unlock()
	}

	return isPathAllowed(u.Path, data)
}

// GetCrawlDelay returns the crawl delay for a domain, or 0 if none.
func (rc *RobotsCache) GetCrawlDelay(rawURL string) time.Duration {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	domain := u.Scheme + "://" + u.Host

	rc.mu.RLock()
	data, ok := rc.entries[domain]
	rc.mu.RUnlock()

	if !ok {
		return 0
	}
	return data.CrawlDelay
}

// GetSitemaps returns sitemap URLs from robots.txt for a domain.
func (rc *RobotsCache) GetSitemaps(rawURL string) []string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	domain := u.Scheme + "://" + u.Host

	rc.mu.RLock()
	data, ok := rc.entries[domain]
	rc.mu.RUnlock()

	if !ok {
		return nil
	}
	return data.Sitemaps
}

func (rc *RobotsCache) fetch(domain string) *RobotsData {
	robotsURL := domain + "/robots.txt"
	resp, err := rc.client.Get(robotsURL)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		// If robots.txt is not found or errors, allow everything
		return &RobotsData{}
	}
	defer resp.Body.Close()

	return ParseRobots(resp.Body, rc.ua)
}

// ParseRobots parses robots.txt content for the given user-agent.
func ParseRobots(r io.Reader, userAgent string) *RobotsData {
	data := &RobotsData{}
	scanner := bufio.NewScanner(r)

	// Normalize our user agent for matching
	uaLower := strings.ToLower(userAgent)
	// Extract first word of user-agent for matching
	uaFirst := uaLower
	if i := strings.IndexByte(uaFirst, '/'); i > 0 {
		uaFirst = uaFirst[:i]
	}

	var (
		currentUA     string
		inMatchingUA  bool
		inWildcardUA  bool
		specificRules RobotsData
		wildcardRules RobotsData
		hasSpecific   bool
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle sitemap directives (not user-agent specific)
		if strings.HasPrefix(strings.ToLower(line), "sitemap:") {
			sitemapURL := strings.TrimSpace(line[len("sitemap:"):])
			if sitemapURL != "" {
				data.Sitemaps = append(data.Sitemaps, sitemapURL)
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "user-agent":
			currentUA = strings.ToLower(value)
			if currentUA == "*" {
				inWildcardUA = true
				inMatchingUA = false
			} else if strings.Contains(uaLower, currentUA) || strings.Contains(uaFirst, currentUA) {
				inMatchingUA = true
				inWildcardUA = false
				hasSpecific = true
			} else {
				inMatchingUA = false
				inWildcardUA = false
			}

		case "disallow":
			if value == "" {
				continue
			}
			if inMatchingUA {
				specificRules.Disallowed = append(specificRules.Disallowed, value)
			} else if inWildcardUA {
				wildcardRules.Disallowed = append(wildcardRules.Disallowed, value)
			}

		case "allow":
			if inMatchingUA {
				specificRules.Allowed = append(specificRules.Allowed, value)
			} else if inWildcardUA {
				wildcardRules.Allowed = append(wildcardRules.Allowed, value)
			}

		case "crawl-delay":
			if delay, err := strconv.Atoi(value); err == nil {
				d := time.Duration(delay) * time.Second
				if inMatchingUA {
					specificRules.CrawlDelay = d
				} else if inWildcardUA {
					wildcardRules.CrawlDelay = d
				}
			}
		}
	}

	// Use specific rules if found, otherwise wildcard
	if hasSpecific {
		data.Disallowed = specificRules.Disallowed
		data.Allowed = specificRules.Allowed
		data.CrawlDelay = specificRules.CrawlDelay
	} else {
		data.Disallowed = wildcardRules.Disallowed
		data.Allowed = wildcardRules.Allowed
		data.CrawlDelay = wildcardRules.CrawlDelay
	}

	return data
}

// isPathAllowed checks if a path is allowed by robots.txt rules.
func isPathAllowed(path string, data *RobotsData) bool {
	if data == nil {
		return true
	}

	// Check allowed first (more specific rules win)
	for _, pattern := range data.Allowed {
		if matchRobotsPattern(path, pattern) {
			return true
		}
	}

	// Check disallowed
	for _, pattern := range data.Disallowed {
		if matchRobotsPattern(path, pattern) {
			return false
		}
	}

	return true
}

// matchRobotsPattern matches a path against a robots.txt pattern.
// Supports * wildcard and $ end anchor.
func matchRobotsPattern(path, pattern string) bool {
	if pattern == "" {
		return false
	}

	// Handle $ end anchor
	if strings.HasSuffix(pattern, "$") {
		pattern = strings.TrimSuffix(pattern, "$")
		if !strings.Contains(pattern, "*") {
			return path == pattern
		}
	}

	// Handle * wildcard
	if strings.Contains(pattern, "*") {
		return matchWildcard(path, pattern)
	}

	return strings.HasPrefix(path, pattern)
}

// matchWildcard matches a string against a pattern with * wildcards.
func matchWildcard(s, pattern string) bool {
	parts := strings.Split(pattern, "*")
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(s[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && idx != 0 {
			// First part must match at start
			return false
		}
		pos += idx + len(part)
	}
	return true
}

// String returns a debug representation.
func (d *RobotsData) String() string {
	return fmt.Sprintf("RobotsData{disallow=%v, allow=%v, delay=%v, sitemaps=%v}",
		d.Disallowed, d.Allowed, d.CrawlDelay, d.Sitemaps)
}
