package crawler

import (
	"net/url"
	"path"
	"strings"
)

// NormalizeURL normalizes a URL for deduplication.
// It lowercases scheme and host, removes default ports, removes fragments,
// removes trailing slashes, and sorts query parameters.
func NormalizeURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Only handle http/https
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", &url.Error{Op: "normalize", URL: rawURL, Err: nil}
	}

	// Lowercase host
	u.Host = strings.ToLower(u.Host)

	// Remove default ports
	host := u.Hostname()
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		u.Host = host
	}

	// Clean path
	u.Path = path.Clean(u.Path)
	if u.Path == "." {
		u.Path = "/"
	}
	// Ensure leading slash
	if !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}

	// Remove fragment
	u.Fragment = ""

	// Sort query parameters
	if u.RawQuery != "" {
		params := u.Query()
		u.RawQuery = params.Encode()
	}

	return u.String(), nil
}

// ResolveURL resolves a relative URL against a base URL.
func ResolveURL(base, ref string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(refURL).String(), nil
}

// IsSameScope checks if a URL is within scope of the start URL.
func IsSameScope(startURL, targetURL string, scope ScopePolicy) bool {
	start, err := url.Parse(startURL)
	if err != nil {
		return false
	}
	target, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	switch scope {
	case ScopeSameDomain:
		return strings.EqualFold(start.Hostname(), target.Hostname())
	case ScopeSameHost:
		startHost := strings.ToLower(start.Hostname())
		targetHost := strings.ToLower(target.Hostname())
		return targetHost == startHost || strings.HasSuffix(targetHost, "."+startHost)
	case ScopeSubpath:
		if !strings.EqualFold(start.Hostname(), target.Hostname()) {
			return false
		}
		startPath := start.Path
		if !strings.HasSuffix(startPath, "/") {
			startPath = path.Dir(startPath) + "/"
		}
		return strings.HasPrefix(target.Path, startPath)
	default:
		return false
	}
}

// MatchesGlobs checks if a URL matches any of the given glob patterns.
// Patterns are matched against the URL path.
func MatchesGlobs(rawURL string, patterns []string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, u.Path); matched {
			return true
		}
		// Also try matching the full URL
		if matched, _ := path.Match(pattern, rawURL); matched {
			return true
		}
	}
	return false
}

// IsValidCrawlURL checks if a URL is suitable for crawling.
func IsValidCrawlURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}

	// Skip common non-HTML extensions
	ext := strings.ToLower(path.Ext(u.Path))
	switch ext {
	case ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp",
		".mp3", ".mp4", ".avi", ".mov", ".wmv",
		".zip", ".tar", ".gz", ".rar", ".7z",
		".exe", ".dmg", ".iso",
		".css", ".js", ".woff", ".woff2", ".ttf", ".eot",
		".xml", ".json", ".rss", ".atom":
		return false
	}

	return true
}

// DomainOf extracts the hostname from a URL.
func DomainOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
