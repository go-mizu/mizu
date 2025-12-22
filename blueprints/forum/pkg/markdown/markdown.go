// Package markdown provides Markdown parsing and sanitization for forum content.
package markdown

import (
	"regexp"
	"strings"
)

var (
	// Username mention pattern: @username
	mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]{3,20})`)

	// Forum mention pattern: /f/forumname
	forumRegex = regexp.MustCompile(`/f/([a-zA-Z0-9_-]{3,30})`)

	// Unsafe HTML patterns
	unsafeHTML = regexp.MustCompile(`<script|<iframe|javascript:|onerror=|onclick=`)
)

// ParsedContent represents processed markdown content.
type ParsedContent struct {
	Content  string   // Processed content
	Mentions []string // Extracted @usernames
	Forums   []string // Extracted /f/forums
}

// Parse processes markdown content, extracting mentions and sanitizing.
func Parse(content string) *ParsedContent {
	result := &ParsedContent{
		Content:  content,
		Mentions: []string{},
		Forums:   []string{},
	}

	// Extract mentions
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			result.Mentions = append(result.Mentions, strings.ToLower(match[1]))
		}
	}

	// Extract forum references
	matches = forumRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			result.Forums = append(result.Forums, strings.ToLower(match[1]))
		}
	}

	// Sanitize content (basic XSS prevention)
	result.Content = Sanitize(result.Content)

	return result
}

// Sanitize removes potentially dangerous HTML/JS from content.
func Sanitize(content string) string {
	// Remove dangerous patterns
	content = unsafeHTML.ReplaceAllString(content, "")

	// TODO: Add proper markdown-to-HTML conversion with whitelist
	// For now, just return sanitized plain text/markdown

	return content
}

// RenderToHTML converts markdown to safe HTML.
// This is a placeholder - in production, use a proper markdown library
// like goldmark or blackfriday with HTML sanitization.
func RenderToHTML(content string) string {
	// Parse and extract mentions/forums
	parsed := Parse(content)

	// Convert mentions to links
	html := mentionRegex.ReplaceAllString(parsed.Content, `<a href="/u/$1" class="mention">@$1</a>`)

	// Convert forum references to links
	html = forumRegex.ReplaceAllString(html, `<a href="/f/$1" class="forum-link">/f/$1</a>`)

	// Convert basic markdown (very simplified - use a real library in production)
	html = convertBasicMarkdown(html)

	return html
}

// convertBasicMarkdown applies basic markdown transformations.
// In production, use a proper markdown library.
func convertBasicMarkdown(content string) string {
	// Bold: **text**
	bold := regexp.MustCompile(`\*\*(.+?)\*\*`)
	content = bold.ReplaceAllString(content, `<strong>$1</strong>`)

	// Italic: *text*
	italic := regexp.MustCompile(`\*(.+?)\*`)
	content = italic.ReplaceAllString(content, `<em>$1</em>`)

	// Code: `code`
	code := regexp.MustCompile("`(.+?)`")
	content = code.ReplaceAllString(content, `<code>$1</code>`)

	// Links: [text](url)
	links := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
	content = links.ReplaceAllString(content, `<a href="$2" rel="nofollow noopener">$1</a>`)

	// Line breaks
	content = strings.ReplaceAll(content, "\n", "<br>")

	return content
}

// ExtractMentions extracts @username mentions from content.
func ExtractMentions(content string) []string {
	var mentions []string
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, strings.ToLower(match[1]))
		}
	}
	return mentions
}

// ExtractForums extracts /f/forum references from content.
func ExtractForums(content string) []string {
	var forums []string
	matches := forumRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			forums = append(forums, strings.ToLower(match[1]))
		}
	}
	return forums
}
