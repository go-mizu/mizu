// Package sanitize provides input sanitization utilities to prevent XSS and injection attacks.
package sanitize

import (
	"errors"
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Validation errors.
var (
	ErrEmptyInput        = errors.New("input cannot be empty")
	ErrInputTooLong      = errors.New("input exceeds maximum length")
	ErrInvalidCharacters = errors.New("input contains invalid characters")
)

// Text sanitizes user text input for safe storage and display.
// It escapes HTML entities and removes dangerous content.
func Text(input string) string {
	if input == "" {
		return ""
	}

	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove other control characters except newlines and tabs
	var sb strings.Builder
	sb.Grow(len(input))
	for _, r := range input {
		if r == '\n' || r == '\r' || r == '\t' || !unicode.IsControl(r) {
			sb.WriteRune(r)
		}
	}
	input = sb.String()

	// HTML escape to prevent XSS
	input = html.EscapeString(input)

	return input
}

// TextWithMaxLength sanitizes text and enforces a maximum length.
func TextWithMaxLength(input string, maxLen int) (string, error) {
	if len(input) > maxLen {
		return "", ErrInputTooLong
	}
	return Text(input), nil
}

// Username validates and sanitizes usernames.
// Allowed: alphanumeric, underscore, hyphen
// Length: 3-32 characters
func Username(input string) (string, error) {
	input = strings.TrimSpace(input)

	if input == "" {
		return "", ErrEmptyInput
	}
	if len(input) < 3 {
		return "", errors.New("username must be at least 3 characters")
	}
	if len(input) > 32 {
		return "", errors.New("username must not exceed 32 characters")
	}

	// Allow only alphanumeric, underscore, hyphen
	for _, r := range input {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return "", ErrInvalidCharacters
		}
	}

	// Usernames should not start with a number or special char
	if unicode.IsDigit(rune(input[0])) || input[0] == '_' || input[0] == '-' {
		return "", errors.New("username must start with a letter")
	}

	return strings.ToLower(input), nil
}

// Email performs basic email validation and sanitization.
func Email(input string) (string, error) {
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)

	if input == "" {
		return "", nil // Empty email is allowed
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(input) {
		return "", errors.New("invalid email format")
	}

	if len(input) > 254 {
		return "", errors.New("email address too long")
	}

	return input, nil
}

// Phone sanitizes phone numbers.
// Removes non-digit characters except leading +.
func Phone(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(input))

	for i, r := range input {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		} else if r == '+' && i == 0 {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// DisplayName sanitizes display names.
// Allows letters, numbers, spaces, and some punctuation.
func DisplayName(input string) (string, error) {
	input = strings.TrimSpace(input)

	if input == "" {
		return "", nil // Empty display name uses username
	}
	if len(input) > 64 {
		return "", errors.New("display name must not exceed 64 characters")
	}

	// Remove control characters
	var sb strings.Builder
	sb.Grow(len(input))
	for _, r := range input {
		if !unicode.IsControl(r) {
			sb.WriteRune(r)
		}
	}
	input = sb.String()

	// HTML escape
	return html.EscapeString(input), nil
}

// MessageContent sanitizes message content.
// Preserves newlines and basic formatting but escapes HTML.
func MessageContent(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", errors.New("message cannot be empty")
	}

	const maxMessageLength = 4096
	if len(input) > maxMessageLength {
		return "", errors.New("message exceeds maximum length")
	}

	return Text(input), nil
}

// URL validates and sanitizes URLs.
// Only allows http and https schemes.
func URL(input string) (string, error) {
	input = strings.TrimSpace(input)

	if input == "" {
		return "", nil
	}

	// Must start with http:// or https://
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return "", errors.New("URL must start with http:// or https://")
	}

	// Basic URL validation
	urlRegex := regexp.MustCompile(`^https?://[^\s<>"{}|\\^` + "`" + `\[\]]+$`)
	if !urlRegex.MatchString(input) {
		return "", errors.New("invalid URL format")
	}

	// Prevent javascript: and data: schemes that might be URL-encoded
	lower := strings.ToLower(input)
	if strings.Contains(lower, "javascript:") || strings.Contains(lower, "data:") {
		return "", errors.New("URL contains forbidden scheme")
	}

	return input, nil
}

// SearchQuery sanitizes search queries.
func SearchQuery(input string) string {
	input = strings.TrimSpace(input)

	// Remove dangerous SQL characters
	replacer := strings.NewReplacer(
		"'", "",
		"\"", "",
		";", "",
		"--", "",
		"/*", "",
		"*/", "",
	)
	input = replacer.Replace(input)

	// Limit length
	if len(input) > 256 {
		input = input[:256]
	}

	return input
}

// StripTags removes all HTML tags from input.
func StripTags(input string) string {
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	return tagRegex.ReplaceAllString(input, "")
}

// NormalizeNewlines converts all newline variants to \n.
func NormalizeNewlines(input string) string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	return input
}

// LimitNewlines limits consecutive newlines to a maximum count.
func LimitNewlines(input string, max int) string {
	if max < 1 {
		max = 1
	}

	pattern := regexp.MustCompile(`\n{` + string(rune('0'+max+1)) + `,}`)
	replacement := strings.Repeat("\n", max)
	return pattern.ReplaceAllString(input, replacement)
}
