// Package slug provides URL slug generation utilities.
package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	multiDash       = regexp.MustCompile(`-+`)
)

// Generate creates a URL-friendly slug from a string.
func Generate(s string) string {
	// Normalize Unicode characters
	s = norm.NFKD.String(s)

	// Convert to lowercase
	s = strings.ToLower(s)

	// Remove diacritics
	var result strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue // Skip combining marks
		}
		result.WriteRune(r)
	}
	s = result.String()

	// Replace non-alphanumeric with dashes
	s = nonAlphanumeric.ReplaceAllString(s, "-")

	// Collapse multiple dashes
	s = multiDash.ReplaceAllString(s, "-")

	// Trim dashes from ends
	s = strings.Trim(s, "-")

	return s
}

// Unique generates a unique slug by appending a suffix if needed.
func Unique(base string, suffix string) string {
	if suffix == "" {
		return base
	}
	return base + "-" + suffix
}
