package cc

import (
	"net/url"
	"strings"
)

// RawLink is a link extracted from HTML before URL resolution.
type RawLink struct {
	Href   string
	Anchor string
	Rel    string
}

// SiteLink is a resolved link ready for storage.
type SiteLink struct {
	SourceURL  string
	TargetURL  string
	AnchorText string
	Rel        string
	IsInternal bool
}

// ExtractLinks extracts all <a href="..."> links from HTML body.
// Uses fast string scanning (no regex, no HTML parser).
func ExtractLinks(body []byte) []RawLink {
	s := string(body)
	lower := strings.ToLower(s)
	var links []RawLink

	pos := 0
	for {
		idx := strings.Index(lower[pos:], "<a ")
		if idx < 0 {
			idx = strings.Index(lower[pos:], "<a\n")
			if idx < 0 {
				idx = strings.Index(lower[pos:], "<a\t")
				if idx < 0 {
					break
				}
			}
		}
		tagStart := pos + idx
		pos = tagStart + 3

		// Find the end of the opening tag
		tagEnd := strings.Index(s[pos:], ">")
		if tagEnd < 0 {
			break
		}
		tagEnd += pos

		// Extract attributes from the opening tag
		tagContent := s[tagStart : tagEnd+1]
		tagLower := strings.ToLower(tagContent)

		href := extractAttr(tagContent, tagLower, "href")
		if href == "" {
			pos = tagEnd + 1
			continue
		}

		rel := extractAttr(tagContent, tagLower, "rel")

		// Extract anchor text: content between > and </a>
		anchor := ""
		closeTag := strings.Index(lower[tagEnd+1:], "</a>")
		if closeTag >= 0 {
			anchor = s[tagEnd+1 : tagEnd+1+closeTag]
			anchor = stripTags(anchor)
			anchor = strings.TrimSpace(anchor)
			if len(anchor) > 500 {
				anchor = anchor[:500]
			}
			pos = tagEnd + 1 + closeTag + 4
		} else {
			pos = tagEnd + 1
		}

		links = append(links, RawLink{
			Href:   strings.TrimSpace(href),
			Anchor: anchor,
			Rel:    rel,
		})
	}

	return links
}

// ResolveLinks resolves raw links against a base URL and classifies them.
func ResolveLinks(raw []RawLink, pageURL, domain string) []SiteLink {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool, len(raw))
	var links []SiteLink

	for _, r := range raw {
		href := r.Href

		// Skip non-HTTP links
		if strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") ||
			strings.HasPrefix(href, "data:") ||
			strings.HasPrefix(href, "#") ||
			href == "" {
			continue
		}

		// Resolve relative URL
		ref, err := url.Parse(href)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(ref)
		target := resolved.String()

		// Deduplicate
		if seen[target] {
			continue
		}
		seen[target] = true

		// Classify internal/external
		isInternal := false
		if resolved.Host != "" {
			targetHost := strings.ToLower(resolved.Host)
			domainLower := strings.ToLower(domain)
			isInternal = targetHost == domainLower ||
				targetHost == "www."+domainLower ||
				strings.HasSuffix(targetHost, "."+domainLower)
		}

		links = append(links, SiteLink{
			SourceURL:  pageURL,
			TargetURL:  target,
			AnchorText: r.Anchor,
			Rel:        r.Rel,
			IsInternal: isInternal,
		})
	}

	return links
}

// extractAttr extracts an HTML attribute value from a tag.
// tagContent is the original case, tagLower is lowercase.
func extractAttr(tagContent, tagLower, attr string) string {
	attr = strings.ToLower(attr)

	// Try attr="value"
	idx := strings.Index(tagLower, attr+`="`)
	if idx >= 0 {
		start := idx + len(attr) + 2
		end := strings.Index(tagContent[start:], `"`)
		if end >= 0 {
			return tagContent[start : start+end]
		}
	}

	// Try attr='value'
	idx = strings.Index(tagLower, attr+`='`)
	if idx >= 0 {
		start := idx + len(attr) + 2
		end := strings.Index(tagContent[start:], `'`)
		if end >= 0 {
			return tagContent[start : start+end]
		}
	}

	// Try attr=value (no quotes)
	idx = strings.Index(tagLower, attr+`=`)
	if idx >= 0 {
		start := idx + len(attr) + 1
		end := strings.IndexAny(tagContent[start:], " \t\n\r>")
		if end >= 0 {
			return tagContent[start : start+end]
		}
		return tagContent[start:]
	}

	return ""
}

// stripTags removes HTML tags from a string, keeping text content.
func stripTags(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			b.WriteByte(' ')
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	return b.String()
}
