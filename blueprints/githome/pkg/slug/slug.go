package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]+`)
	multipleDashes  = regexp.MustCompile(`-+`)
)

// Make creates a URL-safe slug from a string
func Make(s string) string {
	// Normalize unicode
	s = norm.NFKD.String(s)

	// Convert to lowercase
	s = strings.ToLower(s)

	// Remove non-alphanumeric characters (except dashes)
	s = nonAlphanumeric.ReplaceAllString(s, "-")

	// Replace multiple dashes with single dash
	s = multipleDashes.ReplaceAllString(s, "-")

	// Trim dashes from ends
	s = strings.Trim(s, "-")

	return s
}

// IsValid checks if a string is a valid slug (only lowercase letters, numbers, and dashes)
func IsValid(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLower(r) && !unicode.IsDigit(r) && r != '-' {
			return false
		}
	}
	// Can't start or end with dash
	if s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	// Can't have consecutive dashes
	if strings.Contains(s, "--") {
		return false
	}
	return true
}
