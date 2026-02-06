package crawler

import (
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ExtractResult holds data extracted from an HTML page.
type ExtractResult struct {
	Title       string
	Description string
	Content     string
	Language    string
	Links       []string
	Metadata    map[string]string
}

// Extract parses HTML from a reader using a streaming tokenizer and extracts
// page metadata, text content, and links. It resolves relative links against baseURL.
func Extract(r io.Reader, baseURL string) ExtractResult {
	z := html.NewTokenizer(r)
	res := ExtractResult{
		Metadata: make(map[string]string),
	}

	var (
		text      strings.Builder
		inTitle   bool
		inScript  bool
		inStyle   bool
		skipDepth int // depth inside nav/footer/header/aside elements
	)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			res.Content = collapseWhitespace(text.String())
			return res

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := z.TagName()
			tag := atom.Lookup(tn)

			switch tag {
			case atom.Title:
				if tt == html.StartTagToken {
					inTitle = true
				}
			case atom.Script:
				inScript = true
			case atom.Style:
				inStyle = true
			case atom.Nav, atom.Footer, atom.Header, atom.Aside:
				skipDepth++
			case atom.Html:
				if hasAttr {
					attrs := readAttrs(z)
					if lang, ok := attrs["lang"]; ok {
						res.Language = lang
					}
				}
			case atom.Meta:
				if hasAttr {
					attrs := readAttrs(z)
					handleMeta(&res, attrs)
				}
			case atom.A:
				if hasAttr {
					attrs := readAttrs(z)
					if href, ok := attrs["href"]; ok {
						if resolved, err := ResolveURL(baseURL, href); err == nil {
							if IsValidCrawlURL(resolved) {
								res.Links = append(res.Links, resolved)
							}
						}
					}
				}
			case atom.Link:
				if hasAttr {
					attrs := readAttrs(z)
					if attrs["rel"] == "canonical" {
						if href, ok := attrs["href"]; ok {
							res.Metadata["canonical"] = href
						}
					}
				}
			}

		case html.EndTagToken:
			tn, _ := z.TagName()
			tag := atom.Lookup(tn)
			switch tag {
			case atom.Title:
				inTitle = false
			case atom.Script:
				inScript = false
			case atom.Style:
				inStyle = false
			case atom.Nav, atom.Footer, atom.Header, atom.Aside:
				if skipDepth > 0 {
					skipDepth--
				}
			}

		case html.TextToken:
			data := string(z.Text())
			if inTitle && res.Title == "" {
				res.Title = strings.TrimSpace(data)
			}
			if !inScript && !inStyle && skipDepth == 0 {
				text.WriteString(data)
				text.WriteByte(' ')
			}
		}
	}
}

// readAttrs reads all attributes from the current token.
func readAttrs(z *html.Tokenizer) map[string]string {
	attrs := make(map[string]string)
	for {
		key, val, more := z.TagAttr()
		if len(key) > 0 {
			attrs[strings.ToLower(string(key))] = string(val)
		}
		if !more {
			break
		}
	}
	return attrs
}

// handleMeta processes a <meta> tag's attributes.
func handleMeta(res *ExtractResult, attrs map[string]string) {
	name := strings.ToLower(attrs["name"])
	property := strings.ToLower(attrs["property"])
	content := attrs["content"]

	switch {
	case name == "description":
		res.Description = content
	case name == "language" || name == "content-language":
		if res.Language == "" {
			res.Language = content
		}
	case property == "og:title":
		res.Metadata["og:title"] = content
	case property == "og:description":
		res.Metadata["og:description"] = content
	case property == "og:image":
		res.Metadata["og:image"] = content
	case property == "og:url":
		res.Metadata["og:url"] = content
	case property == "og:type":
		res.Metadata["og:type"] = content
	case property == "og:locale":
		if res.Language == "" {
			res.Language = content
		}
	case name == "robots":
		res.Metadata["robots"] = content
	case name == "author":
		res.Metadata["author"] = content
	}
}

// collapseWhitespace trims and collapses consecutive whitespace.
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}
