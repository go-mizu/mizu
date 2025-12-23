package markdown

import (
	"bytes"
	htmlPkg "html"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.Linkify,
	),
	goldmark.WithRendererOptions(
		gmhtml.WithHardWraps(),
		gmhtml.WithXHTML(),
	),
)

// Render converts markdown text to HTML.
func Render(text string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(text), &buf); err != nil {
		// Fallback to escaped text if rendering fails
		return "<p>" + htmlPkg.EscapeString(text) + "</p>"
	}
	return buf.String()
}

// RenderPlain converts text to HTML paragraphs without markdown processing.
func RenderPlain(text string) string {
	if text == "" {
		return ""
	}

	// Split into paragraphs
	paragraphs := strings.Split(text, "\n\n")
	var buf strings.Builder

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Convert single newlines to <br>
		p = strings.ReplaceAll(htmlPkg.EscapeString(p), "\n", "<br>\n")
		buf.WriteString("<p>")
		buf.WriteString(p)
		buf.WriteString("</p>\n")
	}

	return buf.String()
}

// StripHTML removes HTML tags from text.
func StripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Truncate truncates text to maxLen characters, adding ellipsis if needed.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
