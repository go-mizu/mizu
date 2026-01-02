package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // Allow raw HTML (will be sanitized separately)
		),
	)
}

// Render converts Markdown to HTML.
func Render(source string) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderSafe converts Markdown to HTML and sanitizes the output.
func RenderSafe(source string) (string, error) {
	html, err := Render(source)
	if err != nil {
		return "", err
	}
	return Sanitize(html), nil
}

// Sanitize removes dangerous HTML elements and attributes.
func Sanitize(html string) string {
	// Remove script tags
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")

	// Remove style tags
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")

	// Remove on* event handlers
	onRe := regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["'][^"']*["']`)
	html = onRe.ReplaceAllString(html, "")

	// Remove javascript: URLs
	jsRe := regexp.MustCompile(`(?i)href\s*=\s*["']javascript:[^"']*["']`)
	html = jsRe.ReplaceAllString(html, "")

	// Remove data: URLs in src
	dataRe := regexp.MustCompile(`(?i)src\s*=\s*["']data:[^"']*["']`)
	html = dataRe.ReplaceAllString(html, "")

	return html
}

// Summary extracts a plain text summary from Markdown.
func Summary(source string, maxLen int) string {
	// Remove headers
	headerRe := regexp.MustCompile(`(?m)^#+\s+`)
	source = headerRe.ReplaceAllString(source, "")

	// Remove images
	imgRe := regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	source = imgRe.ReplaceAllString(source, "")

	// Convert links to just text
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	source = linkRe.ReplaceAllString(source, "$1")

	// Remove code blocks
	codeBlockRe := regexp.MustCompile("(?s)```.*?```")
	source = codeBlockRe.ReplaceAllString(source, "")

	// Remove inline code
	inlineCodeRe := regexp.MustCompile("`[^`]+`")
	source = inlineCodeRe.ReplaceAllString(source, "")

	// Remove emphasis markers
	emphRe := regexp.MustCompile(`[*_~]+`)
	source = emphRe.ReplaceAllString(source, "")

	// Collapse whitespace
	wsRe := regexp.MustCompile(`\s+`)
	source = wsRe.ReplaceAllString(source, " ")
	source = strings.TrimSpace(source)

	// Truncate
	runes := []rune(source)
	if len(runes) > maxLen {
		source = string(runes[:maxLen-3]) + "..."
	}

	return source
}
