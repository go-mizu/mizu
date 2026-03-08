package markdown

import (
	"bytes"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ConvertLight is a lightweight HTML→Markdown converter that skips trafilatura's
// full extraction pipeline. It parses HTML, finds the main content region
// (article/main/role=main), strips boilerplate using text-density scoring,
// and converts directly to markdown. ~5-10x faster than Convert with
// near-identical quality on well-structured pages.
func ConvertLight(rawHTML []byte, pageURL string) Result {
	start := time.Now()
	htmlSize := len(rawHTML)

	doc, err := parseHTMLFast(rawHTML)
	if err != nil {
		ms := int(time.Since(start).Milliseconds())
		return Result{HTMLSize: htmlSize, ConvertMs: ms, Error: "html parse: " + err.Error()}
	}

	// Find <body> or use doc root
	body := findBody(doc)
	if body == nil {
		body = doc
	}

	// Strip boilerplate elements first (before content region detection)
	stripBoilerplate(body)

	// Try to find <article>, <main>, or role="main" (like trafilatura).
	content := findContentRegion(body)
	if content == nil {
		// Fallback: find the div/section with the most <p> tags.
		// Only use if it has at least 3 paragraphs (strong content signal).
		candidate := findBestContentBlock(body)
		if candidate != nil && countElements(candidate, atom.P) >= 3 {
			content = candidate
		}
	}
	if content == nil {
		content = body
	}

	// Strip link-heavy blocks (navigation remnants inside content)
	stripLinkHeavyBlocks(content)

	// Extract title: og:title > <title> tag
	title := extractOGTitle(doc)
	if title == "" {
		title = extractTitle(doc)
	}

	// Convert directly to markdown
	md := fastMarkdown(content)

	// If content region produced too little, fall back to full body
	if len(md) < 50 && content != body {
		md = fastMarkdown(body)
	}

	if len(md) < 10 {
		ms := int(time.Since(start).Milliseconds())
		return Result{HTMLSize: htmlSize, ConvertMs: ms, Error: "no content extracted"}
	}

	mdSize := len(md)
	ms := int(time.Since(start).Milliseconds())

	return Result{
		Markdown:       md,
		Title:          title,
		HasContent:     true,
		HTMLSize:       htmlSize,
		MarkdownSize:   mdSize,
		HTMLTokens:     EstimateTokens(htmlSize),
		MarkdownTokens: EstimateTokens(mdSize),
		ConvertMs:      ms,
	}
}

// findContentRegion looks for <article>, <main>, or an element with role="main".
// Returns nil if no content region is found.
func findContentRegion(body *html.Node) *html.Node {
	// Priority: <main> > <article> > role="main"
	var mainEl, articleEl, roleMainEl *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			return
		}
		switch n.DataAtom {
		case atom.Main:
			if mainEl == nil {
				mainEl = n
			}
			return
		case atom.Article:
			if articleEl == nil {
				articleEl = n
			}
			return
		}
		for _, a := range n.Attr {
			if a.Key == "role" && a.Val == "main" {
				if roleMainEl == nil {
					roleMainEl = n
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(body)

	if mainEl != nil {
		return mainEl
	}
	if articleEl != nil {
		return articleEl
	}
	return roleMainEl
}

// findBestContentBlock finds the div/section with the highest text content.
// This mimics trafilatura's scoring: the block with the most text is likely
// the main content area.
func findBestContentBlock(body *html.Node) *html.Node {
	var best *html.Node
	var bestScore int

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		// Only consider div, section, td as potential content blocks
		switch n.DataAtom {
		case atom.Div, atom.Section, atom.Td:
		default:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			return
		}

		textLen, _ := subtreeSize(n)
		// Count <p> tags as a strong signal of content
		pCount := countElements(n, atom.P)
		score := textLen + pCount*200

		if score > bestScore && textLen > 200 {
			best = n
			bestScore = score
		}

		// Also check children (content block might be nested)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(body)
	return best
}

func countElements(n *html.Node, tag atom.Atom) int {
	count := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == tag {
			count++
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return count
}

// extractOGTitle extracts title from <meta property="og:title"> tag.
func extractOGTitle(doc *html.Node) string {
	var title string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if title != "" {
			return
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Meta {
			var prop, content string
			for _, a := range n.Attr {
				switch a.Key {
				case "property":
					prop = a.Val
				case "name":
					if prop == "" {
						prop = a.Val
					}
				case "content":
					content = a.Val
				}
			}
			if (prop == "og:title" || prop == "twitter:title") && content != "" {
				title = strings.TrimSpace(content)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return title
}

func findBody(doc *html.Node) *html.Node {
	var body *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if body != nil {
			return
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Body {
			body = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return body
}

func extractTitle(doc *html.Node) string {
	var title string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if title != "" {
			return
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Title {
			title = textContent(n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return title
}

// stripBoilerplate removes boilerplate elements using tag-based and
// text-density-aware heuristics.
func stripBoilerplate(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if c.Type != html.ElementNode {
			continue
		}
		// Always strip these regardless of content
		if isAlwaysBoilerplate(c) {
			n.RemoveChild(c)
			continue
		}
		// For structural tags (nav, header, footer, aside), check text density
		// before removing — some sites put real content in these elements.
		if isStructuralBoilerplate(c) {
			if textDensity(c) < 0.3 {
				n.RemoveChild(c)
				continue
			}
			// High text density — keep it but strip its children recursively
		}
		// Class/id pattern matching with density guard
		if hasBoilerplateAttr(c) && textDensity(c) < 0.3 {
			n.RemoveChild(c)
			continue
		}
		stripBoilerplate(c)
	}
}

// isAlwaysBoilerplate returns true for elements that never contain useful content.
func isAlwaysBoilerplate(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Noscript, atom.Iframe, atom.Svg,
		atom.Form, atom.Button, atom.Input, atom.Select, atom.Textarea,
		atom.Aside:
		return true
	}
	return false
}

// isStructuralBoilerplate returns true for elements that are usually boilerplate
// but may contain content on some pages.
func isStructuralBoilerplate(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Nav, atom.Header, atom.Footer:
		return true
	}
	return false
}

// textDensity computes the ratio of text bytes to total bytes (including tags)
// for a subtree. Returns 0.0–1.0.
func textDensity(n *html.Node) float64 {
	textLen, totalLen := subtreeSize(n)
	if totalLen == 0 {
		return 0
	}
	return float64(textLen) / float64(totalLen)
}

func subtreeSize(n *html.Node) (textBytes, totalBytes int) {
	if n.Type == html.TextNode {
		trimmed := strings.TrimSpace(n.Data)
		return len(trimmed), len(n.Data)
	}
	if n.Type == html.ElementNode {
		// Approximate tag overhead: <tag> + </tag>
		totalBytes += len(n.Data)*2 + 5
		for _, a := range n.Attr {
			totalBytes += len(a.Key) + len(a.Val) + 4
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		t, tot := subtreeSize(c)
		textBytes += t
		totalBytes += tot
	}
	return
}

// stripLinkHeavyBlocks removes child elements where links dominate the text
// content (>50% of text is inside <a> tags). These are typically breadcrumbs,
// tag clouds, "related articles" lists, etc. that trafilatura would score low.
func stripLinkHeavyBlocks(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if c.Type != html.ElementNode {
			continue
		}
		// Only check block-level elements
		switch c.DataAtom {
		case atom.Div, atom.Section, atom.Ul, atom.Ol, atom.Aside:
		default:
			stripLinkHeavyBlocks(c)
			continue
		}

		textLen, _ := subtreeSize(c)
		linkTextLen := linkTextSize(c)
		linkCount := countElements(c, atom.A)
		// For lists: strip if most items are link-only (3+ links, >40% link text)
		// For divs: stricter threshold (5+ links, >60% link text)
		isList := c.DataAtom == atom.Ul || c.DataAtom == atom.Ol
		minLinks := 5
		minPct := 60
		if isList {
			minLinks = 3
			minPct = 40
		}
		if textLen > 50 && linkCount >= minLinks && linkTextLen*100/textLen > minPct {
			n.RemoveChild(c)
			continue
		}
		stripLinkHeavyBlocks(c)
	}
}

// linkTextSize returns the total text bytes inside <a> tags in a subtree.
func linkTextSize(n *html.Node) int {
	total := 0
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inLink bool) {
		if n.Type == html.TextNode && inLink {
			total += len(strings.TrimSpace(n.Data))
			return
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			inLink = true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inLink)
		}
	}
	walk(n, false)
	return total
}

var boilerplatePatterns = [][]byte{
	[]byte("sidebar"), []byte("menu"), []byte("navbar"),
	[]byte("footer"), []byte("header"), []byte("cookie"),
	[]byte("banner"), []byte("popup"), []byte("modal"),
	[]byte("advertisement"), []byte("social"), []byte("share"),
	[]byte("comment"), []byte("related"),
}

func hasBoilerplateAttr(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key != "class" && a.Key != "id" && a.Key != "role" {
			continue
		}
		val := []byte(a.Val)
		for _, pat := range boilerplatePatterns {
			if bytes.Contains(val, pat) {
				return true
			}
		}
		if a.Key == "role" && (a.Val == "navigation" || a.Val == "banner" || a.Val == "contentinfo") {
			return true
		}
	}
	return false
}
