package text

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	mentionRegex  = regexp.MustCompile(`@([a-zA-Z0-9_]{3,20})`)
	urlRegex      = regexp.MustCompile(`https?://[^\s<>\[\]]+`)
	domainRegex   = regexp.MustCompile(`^(?:https?://)?([^/]+)`)
	whitespaceRe  = regexp.MustCompile(`\s+`)
)

// ExtractMentions extracts @username mentions from text.
func ExtractMentions(text string) []string {
	matches := mentionRegex.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		username := strings.ToLower(m[1])
		if !seen[username] {
			seen[username] = true
			result = append(result, username)
		}
	}
	return result
}

// ExtractURLs extracts URLs from text.
func ExtractURLs(text string) []string {
	return urlRegex.FindAllString(text, -1)
}

// ExtractDomain extracts the domain from a URL.
func ExtractDomain(url string) string {
	matches := domainRegex.FindStringSubmatch(url)
	if len(matches) > 1 {
		domain := matches[1]
		// Remove www. prefix
		domain = strings.TrimPrefix(domain, "www.")
		return domain
	}
	return ""
}

// CharCount returns the number of Unicode characters in a string.
func CharCount(s string) int {
	return utf8.RuneCountInString(s)
}

// Truncate truncates a string to n characters with ellipsis.
func Truncate(s string, n int) string {
	if CharCount(s) <= n {
		return s
	}
	runes := []rune(s)
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

// Slugify converts text to a URL-safe slug.
func Slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}
	s = result.String()

	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	// Limit length
	if len(s) > 80 {
		s = s[:80]
		// Don't end with a hyphen
		s = strings.TrimRight(s, "-")
	}

	return s
}

// NormalizeWhitespace collapses multiple whitespace into single spaces.
func NormalizeWhitespace(s string) string {
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(s, " "))
}

// StripHTML removes HTML tags from text.
func StripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

// WordCount returns the number of words in a string.
func WordCount(s string) int {
	s = NormalizeWhitespace(s)
	if s == "" {
		return 0
	}
	return len(strings.Fields(s))
}

// ReadTimeMinutes estimates reading time in minutes.
func ReadTimeMinutes(s string) int {
	words := WordCount(s)
	// Average reading speed: 200 words per minute
	minutes := words / 200
	if minutes < 1 {
		return 1
	}
	return minutes
}
