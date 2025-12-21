package view

import (
	"net/url"
	"regexp"
	"strings"
)

// Regex patterns for WikiText conversion
var (
	// Code blocks: <syntaxhighlight lang="python">...</syntaxhighlight>
	syntaxHighlightRe = regexp.MustCompile(`(?s)<syntaxhighlight([^>]*)>(.*?)</syntaxhighlight>`)

	// Source blocks: <source lang="python">...</source> (deprecated but still used)
	sourceRe = regexp.MustCompile(`(?s)<source([^>]*)>(.*?)</source>`)

	// Inline code: <code>...</code>
	inlineCodeRe = regexp.MustCompile(`<code>([^<]*)</code>`)

	// Preformatted text: <pre>...</pre>
	preRe = regexp.MustCompile(`(?s)<pre>([^<]*)</pre>`)

	// Language attribute in syntaxhighlight/source tags
	langAttrRe = regexp.MustCompile(`lang\s*=\s*["']?([^"'\s>]+)["']?`)

	// Internal links: [[Page]] or [[Page|Display]]
	wikiLinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

	// Bold: '''text'''
	boldRe = regexp.MustCompile(`'''([^']+)'''`)

	// Italic: ''text''
	italicRe = regexp.MustCompile(`''([^']+)''`)

	// Headings: == Heading ==
	heading6Re = regexp.MustCompile(`(?m)^======\s*([^=]+)\s*======\s*$`)
	heading5Re = regexp.MustCompile(`(?m)^=====\s*([^=]+)\s*=====\s*$`)
	heading4Re = regexp.MustCompile(`(?m)^====\s*([^=]+)\s*====\s*$`)
	heading3Re = regexp.MustCompile(`(?m)^===\s*([^=]+)\s*===\s*$`)
	heading2Re = regexp.MustCompile(`(?m)^==\s*([^=]+)\s*==\s*$`)

	// Templates: {{...}} - needs to handle nested braces
	templateRe = regexp.MustCompile(`\{\{[^{}]*\}\}`)

	// References: <ref>...</ref> and <ref ... />
	refBlockRe = regexp.MustCompile(`(?s)<ref[^>]*>.*?</ref>`)
	refSelfRe  = regexp.MustCompile(`<ref[^/]*/\s*>`)

	// File/Image links: [[File:...]] or [[Image:...]] (also Vietnamese variants)
	fileLinkRe = regexp.MustCompile(`(?i)\[\[(File|Image|Tập tin|Hình):[^\]]+\]\]`)

	// Category links: [[Category:...]] (also Vietnamese variant)
	categoryRe = regexp.MustCompile(`(?i)\[\[(Category|Thể loại):[^\]]+\]\]`)

	// Interwiki links: [[en:Page]], [[fr:Page]], etc.
	interwikiRe = regexp.MustCompile(`\[\[[a-z]{2,3}:[^\]]+\]\]`)

	// Infobox templates (multiline)
	infoboxRe = regexp.MustCompile(`(?s)\{\{Infobox[^{}]*(?:\{[^{}]*\}[^{}]*)*\}\}`)

	// Wiki tables: {| ... |}
	tableRe = regexp.MustCompile(`(?s)\{\|.*?\|\}`)

	// HTML comments
	commentRe = regexp.MustCompile(`(?s)<!--.*?-->`)

	// Nowiki tags
	nowikiRe = regexp.MustCompile(`(?s)<nowiki>.*?</nowiki>`)

	// External links: [http://... text]
	extLinkRe = regexp.MustCompile(`\[(https?://[^\s\]]+)\s+([^\]]+)\]`)

	// Bare external links in brackets: [http://...]
	extLinkBareRe = regexp.MustCompile(`\[(https?://[^\s\]]+)\]`)
)

// ConvertWikiTextToMarkdown converts MediaWiki markup to markdown.
// Handles common WikiText patterns:
// - [[Page]] → [Page](/page?wiki=xxx&title=Page)
// - [[Page|Display]] → [Display](/page?wiki=xxx&title=Page)
// - '''bold''' → **bold**
// - ''italic'' → *italic*
// - == Heading == → ## Heading
// - {{templates}} → stripped
// - <ref>...</ref> → stripped
func ConvertWikiTextToMarkdown(wikitext, wikiname string) string {
	if wikitext == "" {
		return ""
	}

	text := wikitext

	// Convert code blocks FIRST to preserve code content before other processing
	text = convertCodeBlocks(text)

	// Remove HTML comments first
	text = commentRe.ReplaceAllString(text, "")

	// Remove nowiki blocks
	text = nowikiRe.ReplaceAllString(text, "")

	// Remove infobox templates (before general templates)
	text = infoboxRe.ReplaceAllString(text, "")

	// Remove templates (may need multiple passes for nested)
	for i := 0; i < 5; i++ {
		newText := templateRe.ReplaceAllString(text, "")
		if newText == text {
			break
		}
		text = newText
	}

	// Remove wiki tables
	text = tableRe.ReplaceAllString(text, "")

	// Remove references
	text = refBlockRe.ReplaceAllString(text, "")
	text = refSelfRe.ReplaceAllString(text, "")

	// Remove file/image links
	text = fileLinkRe.ReplaceAllString(text, "")

	// Remove category links
	text = categoryRe.ReplaceAllString(text, "")

	// Remove interwiki links
	text = interwikiRe.ReplaceAllString(text, "")

	// Convert internal wiki links
	text = convertWikiLinksFromWikiText(text, wikiname)

	// Convert external links [url text] to [text](url)
	text = extLinkRe.ReplaceAllString(text, "[$2]($1)")
	text = extLinkBareRe.ReplaceAllString(text, "$1")

	// Convert headings (from deepest to shallowest to avoid conflicts)
	text = heading6Re.ReplaceAllString(text, "###### $1")
	text = heading5Re.ReplaceAllString(text, "##### $1")
	text = heading4Re.ReplaceAllString(text, "#### $1")
	text = heading3Re.ReplaceAllString(text, "### $1")
	text = heading2Re.ReplaceAllString(text, "## $1")

	// Convert bold (must be before italic)
	text = boldRe.ReplaceAllString(text, "**$1**")

	// Convert italic
	text = italicRe.ReplaceAllString(text, "*$1*")

	// Clean up excessive newlines
	text = cleanupNewlines(text)

	return text
}

// convertWikiLinksFromWikiText converts [[Page]] and [[Page|Text]] to markdown links.
// This is similar to convertWikiLinks in markdown.go but handles more edge cases.
func convertWikiLinksFromWikiText(text, wikiname string) string {
	return wikiLinkRe.ReplaceAllStringFunc(text, func(match string) string {
		submatches := wikiLinkRe.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		pageName := strings.TrimSpace(submatches[1])
		displayText := pageName

		// Handle piped links: [[Page|Display Text]]
		if len(submatches) > 2 && submatches[2] != "" {
			displayText = strings.TrimSpace(submatches[2])
		}

		// Skip special namespace links (already handled by other regexes but double-check)
		if strings.Contains(pageName, ":") {
			lowerPage := strings.ToLower(pageName)
			if strings.HasPrefix(lowerPage, "file:") ||
				strings.HasPrefix(lowerPage, "image:") ||
				strings.HasPrefix(lowerPage, "category:") ||
				strings.HasPrefix(lowerPage, "tập tin:") ||
				strings.HasPrefix(lowerPage, "hình:") ||
				strings.HasPrefix(lowerPage, "thể loại:") {
				return ""
			}
		}

		// Handle section links: [[Page#Section]] → [[Page]]
		if idx := strings.Index(pageName, "#"); idx != -1 {
			pageName = pageName[:idx]
		}

		// Skip empty page names
		if pageName == "" {
			return displayText
		}

		// Build the internal link URL
		pageURL := "/page?wiki=" + url.QueryEscape(wikiname) + "&title=" + url.QueryEscape(pageName)

		return "[" + displayText + "](" + pageURL + ")"
	})
}

// cleanupNewlines removes excessive blank lines.
func cleanupNewlines(text string) string {
	// Replace 3+ newlines with 2
	multiNewlineRe := regexp.MustCompile(`\n{3,}`)
	text = multiNewlineRe.ReplaceAllString(text, "\n\n")

	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// convertCodeBlocks converts Wikipedia code block tags to markdown fenced code blocks.
// Handles:
// - <syntaxhighlight lang="python">...</syntaxhighlight> → ```python\n...\n```
// - <source lang="python">...</source> → ```python\n...\n```
// - <code>...</code> → `...`
// - <pre>...</pre> → ```\n...\n```
func convertCodeBlocks(text string) string {
	// Helper function to convert a code block match to fenced markdown
	convertBlock := func(re *regexp.Regexp, match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		attrs := submatches[1] // Attributes like lang="python"
		code := submatches[2]  // The code content

		// Extract language from attributes
		lang := ""
		if langMatch := langAttrRe.FindStringSubmatch(attrs); len(langMatch) > 1 {
			lang = strings.ToLower(langMatch[1])
		}

		// Trim leading/trailing whitespace from code but preserve internal formatting
		code = strings.TrimSpace(code)

		// Return fenced code block
		return "\n```" + lang + "\n" + code + "\n```\n"
	}

	// Convert <syntaxhighlight> blocks
	text = syntaxHighlightRe.ReplaceAllStringFunc(text, func(match string) string {
		return convertBlock(syntaxHighlightRe, match)
	})

	// Convert <source> blocks
	text = sourceRe.ReplaceAllStringFunc(text, func(match string) string {
		return convertBlock(sourceRe, match)
	})

	// Convert <pre> blocks to fenced code blocks (no language)
	text = preRe.ReplaceAllStringFunc(text, func(match string) string {
		submatches := preRe.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		code := strings.TrimSpace(submatches[1])
		return "\n```\n" + code + "\n```\n"
	})

	// Convert inline <code> to backticks
	text = inlineCodeRe.ReplaceAllString(text, "`$1`")

	return text
}
