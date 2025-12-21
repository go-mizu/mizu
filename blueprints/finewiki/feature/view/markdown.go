package view

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// markdown is the configured goldmark instance
var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.Linkify,
		extension.TaskList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
		html.WithUnsafe(), // Allow raw HTML in markdown
	),
)

// wikiLinkRegex matches [[Page Name]] or [[Page Name|Display Text]]
var wikiLinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// RenderMarkdown converts markdown text to HTML with wiki link support.
// wikiname is used to generate internal links like /page?wiki=enwiki&title=PageName
func RenderMarkdown(text, wikiname string) (string, error) {
	// First, convert wiki-style links [[Page]] or [[Page|Text]] to markdown links
	text = convertWikiLinks(text, wikiname)

	var buf bytes.Buffer
	if err := markdown.Convert([]byte(text), &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// convertWikiLinks converts [[Page Name]] to [Page Name](/page?wiki=xxx&title=Page+Name)
// and [[Page Name|Display]] to [Display](/page?wiki=xxx&title=Page+Name)
func convertWikiLinks(text, wikiname string) string {
	return wikiLinkRegex.ReplaceAllStringFunc(text, func(match string) string {
		submatches := wikiLinkRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		pageName := strings.TrimSpace(submatches[1])
		displayText := pageName

		if len(submatches) > 2 && submatches[2] != "" {
			displayText = strings.TrimSpace(submatches[2])
		}

		// Build the URL
		pageURL := "/page?wiki=" + url.QueryEscape(wikiname) + "&title=" + url.QueryEscape(pageName)

		return "[" + displayText + "](" + pageURL + ")"
	})
}

// RenderPage renders a Page to HTML.
// Uses WikiText when available (to preserve internal links), falling back to Text.
func RenderPage(p *Page) (string, error) {
	// Prefer WikiText for content with links preserved
	if p.WikiText != "" {
		// Convert WikiText to markdown, then render to HTML
		markdown := ConvertWikiTextToMarkdown(p.WikiText, p.WikiName)
		return RenderMarkdown(markdown, p.WikiName)
	}

	// Fallback to pre-processed Text field
	if p.Text != "" {
		return RenderMarkdown(p.Text, p.WikiName)
	}

	return "", nil
}

// RenderText converts plain text to basic HTML (preserving paragraphs).
// Also converts wiki links. Use this as a fallback when markdown parsing is not desired.
func RenderText(text, wikiname string) string {
	// Convert wiki links first
	text = convertWikiLinks(text, wikiname)

	// Escape HTML
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")

	// Convert double newlines to paragraphs
	paragraphs := strings.Split(text, "\n\n")
	var buf bytes.Buffer
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Convert single newlines to <br>
		p = strings.ReplaceAll(p, "\n", "<br>\n")
		buf.WriteString("<p>")
		buf.WriteString(p)
		buf.WriteString("</p>\n")
	}

	return buf.String()
}
