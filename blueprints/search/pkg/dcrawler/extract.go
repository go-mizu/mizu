package dcrawler

import (
	"bytes"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// PageMeta holds metadata extracted from an HTML page.
type PageMeta struct {
	Title       string
	Description string
	Language    string
	Canonical   string
	Links       []Link
}

// ExtractLinksAndMeta parses HTML using a streaming tokenizer (no DOM)
// and extracts links, title, description, language, and canonical URL.
//
// Link discovery covers:
//   - <a href> with anchor text capture (text between open/close tags)
//   - <a title> as fallback anchor text when text content is empty
//   - <area href> (image map links) with alt text
//   - <base href> (overrides URL resolution base)
//   - <link rel="canonical|next|prev|alternate"> (pagination, alternates)
//   - <meta http-equiv="refresh"> (redirect URLs)
//   - <iframe src> (internal embedded pages only)
func ExtractLinksAndMeta(body []byte, baseURL *url.URL, domain string, extractImages bool) PageMeta {
	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	var meta PageMeta
	var inTitle bool
	var titleBuf strings.Builder

	// Anchor text tracking: capture text between <a> and </a>
	var inAnchor bool
	var anchorBuf strings.Builder
	var currentLink Link
	var currentTitle string // <a title="..."> fallback

	// <base href> can override URL resolution base
	effectiveBase := baseURL

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			meta.Title = strings.TrimSpace(titleBuf.String())
			// Finalize any unclosed anchor
			if inAnchor && currentLink.TargetURL != "" {
				anchorText := normalizeText(anchorBuf.String())
				if anchorText == "" {
					anchorText = currentTitle
				}
				currentLink.AnchorText = truncate(anchorText, 200)
				meta.Links = append(meta.Links, currentLink)
			}
			return meta

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := tokenizer.TagName()

			if !hasAttr {
				if len(tn) == 5 && string(tn) == "title" {
					inTitle = true
				}
				continue
			}

			switch {
			case len(tn) == 1 && tn[0] == 'a':
				// Finalize any previous unclosed anchor
				if inAnchor && currentLink.TargetURL != "" {
					anchorText := normalizeText(anchorBuf.String())
					if anchorText == "" {
						anchorText = currentTitle
					}
					currentLink.AnchorText = truncate(anchorText, 200)
					meta.Links = append(meta.Links, currentLink)
				}
				href, rel, title := extractAnchorInfo(tokenizer)
				anchorBuf.Reset()
				currentTitle = title
				if href == "" {
					inAnchor = false
					continue
				}
				resolved := resolveURL(href, effectiveBase)
				if resolved == "" {
					inAnchor = false
					continue
				}
				currentLink = Link{
					TargetURL:  resolved,
					Rel:        rel,
					IsInternal: isInternalURL(resolved, domain),
				}
				inAnchor = true

				if tt == html.SelfClosingTagToken {
					currentLink.AnchorText = truncate(title, 200)
					meta.Links = append(meta.Links, currentLink)
					inAnchor = false
					currentLink = Link{}
				}

			case len(tn) == 4 && string(tn) == "area":
				href, rel, alt := extractAreaAttrs(tokenizer)
				if href == "" {
					continue
				}
				resolved := resolveURL(href, effectiveBase)
				if resolved == "" {
					continue
				}
				meta.Links = append(meta.Links, Link{
					TargetURL:  resolved,
					AnchorText: truncate(alt, 200),
					Rel:        rel,
					IsInternal: isInternalURL(resolved, domain),
				})

			case len(tn) == 4 && string(tn) == "base":
				href := extractAttr(tokenizer, "href")
				if href != "" {
					if u, err := url.Parse(href); err == nil {
						effectiveBase = baseURL.ResolveReference(u)
					}
				}

			case len(tn) == 4 && string(tn) == "link":
				rel, href := extractLinkAttrs(tokenizer)
				if href == "" {
					continue
				}
				switch rel {
				case "canonical":
					meta.Canonical = resolveURL(href, effectiveBase)
				case "next", "prev", "alternate":
					resolved := resolveURL(href, effectiveBase)
					if resolved != "" {
						meta.Links = append(meta.Links, Link{
							TargetURL:  resolved,
							Rel:        rel,
							IsInternal: isInternalURL(resolved, domain),
						})
					}
				}

			case len(tn) == 4 && string(tn) == "meta":
				name, content, httpEquiv, property := extractMetaAttrs(tokenizer)
				switch {
				case strings.EqualFold(name, "description"):
					meta.Description = truncate(content, 500)
				case strings.EqualFold(name, "language"):
					meta.Language = content
				case strings.EqualFold(httpEquiv, "refresh") && content != "":
					if u := parseMetaRefreshURL(content); u != "" {
						resolved := resolveURL(u, effectiveBase)
						if resolved != "" {
							meta.Links = append(meta.Links, Link{
								TargetURL:  resolved,
								Rel:        "meta-refresh",
								IsInternal: isInternalURL(resolved, domain),
							})
						}
					}
				case extractImages && content != "" && strings.EqualFold(property, "og:image"):
					resolved := resolveURL(content, effectiveBase)
					if resolved != "" {
						meta.Links = append(meta.Links, Link{
							TargetURL:  resolved,
							Rel:        "og:image",
							IsInternal: isInternalURL(resolved, domain),
						})
					}
				}

			case len(tn) == 4 && string(tn) == "html":
				meta.Language = extractAttr(tokenizer, "lang")

			case len(tn) == 5 && string(tn) == "title":
				inTitle = true

			case len(tn) == 3 && string(tn) == "img":
				if extractImages {
					src, srcset, alt := extractImgAttrs(tokenizer)
					if src != "" {
						resolved := resolveURL(src, effectiveBase)
						if resolved != "" {
							meta.Links = append(meta.Links, Link{
								TargetURL:  resolved,
								AnchorText: truncate(alt, 200),
								Rel:        "image",
								IsInternal: isInternalURL(resolved, domain),
							})
						}
					}
					for _, u := range parseSrcset(srcset) {
						resolved := resolveURL(u, effectiveBase)
						if resolved != "" {
							meta.Links = append(meta.Links, Link{
								TargetURL:  resolved,
								Rel:        "image-srcset",
								IsInternal: isInternalURL(resolved, domain),
							})
						}
					}
				}

			case len(tn) == 6 && string(tn) == "iframe":
				src := extractAttr(tokenizer, "src")
				if src != "" {
					resolved := resolveURL(src, effectiveBase)
					if resolved != "" && isInternalURL(resolved, domain) {
						meta.Links = append(meta.Links, Link{
							TargetURL:  resolved,
							Rel:        "iframe",
							IsInternal: true,
						})
					}
				}
			}

		case html.TextToken:
			text := tokenizer.Text()
			if inTitle {
				titleBuf.Write(text)
			}
			if inAnchor {
				anchorBuf.Write(text)
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			switch {
			case len(tn) == 5 && string(tn) == "title":
				inTitle = false
			case len(tn) == 1 && tn[0] == 'a':
				if inAnchor {
					if currentLink.TargetURL != "" {
						anchorText := normalizeText(anchorBuf.String())
						if anchorText == "" {
							anchorText = currentTitle
						}
						currentLink.AnchorText = truncate(anchorText, 200)
						meta.Links = append(meta.Links, currentLink)
					}
					inAnchor = false
					currentLink = Link{}
					currentTitle = ""
					anchorBuf.Reset()
				}
			}
		}
	}
}

// extractAnchorInfo extracts href, rel, and title from an <a> tag.
func extractAnchorInfo(z *html.Tokenizer) (href, rel, title string) {
	for {
		key, val, more := z.TagAttr()
		switch string(key) {
		case "href":
			href = string(val)
		case "rel":
			rel = string(val)
		case "title":
			title = string(val)
		}
		if !more {
			break
		}
	}
	return
}

// extractAreaAttrs extracts href, rel, and alt from an <area> tag.
func extractAreaAttrs(z *html.Tokenizer) (href, rel, alt string) {
	for {
		key, val, more := z.TagAttr()
		switch string(key) {
		case "href":
			href = string(val)
		case "rel":
			rel = string(val)
		case "alt":
			alt = string(val)
		}
		if !more {
			break
		}
	}
	return
}

func extractLinkAttrs(z *html.Tokenizer) (rel, href string) {
	for {
		key, val, more := z.TagAttr()
		k := string(key)
		switch k {
		case "rel":
			rel = string(val)
		case "href":
			href = string(val)
		}
		if !more {
			break
		}
	}
	return
}

// extractMetaAttrs extracts name, content, http-equiv, and property from a <meta> tag.
func extractMetaAttrs(z *html.Tokenizer) (name, content, httpEquiv, property string) {
	for {
		key, val, more := z.TagAttr()
		switch string(key) {
		case "name":
			name = string(val)
		case "content":
			content = string(val)
		case "http-equiv":
			httpEquiv = string(val)
		case "property":
			property = string(val)
		}
		if !more {
			break
		}
	}
	return
}

// extractImgAttrs extracts src, srcset, and alt from an <img> tag.
func extractImgAttrs(z *html.Tokenizer) (src, srcset, alt string) {
	for {
		key, val, more := z.TagAttr()
		switch string(key) {
		case "src":
			src = string(val)
		case "srcset":
			srcset = string(val)
		case "alt":
			alt = string(val)
		}
		if !more {
			break
		}
	}
	return
}

// parseSrcset extracts URLs from an HTML srcset attribute value.
// Format: "url1 descriptor1, url2 descriptor2, ..."
func parseSrcset(srcset string) []string {
	if srcset == "" {
		return nil
	}
	var urls []string
	for _, part := range strings.Split(srcset, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) >= 1 {
			u := fields[0]
			if u != "" && !strings.HasPrefix(u, "data:") {
				urls = append(urls, u)
			}
		}
	}
	return urls
}

func extractAttr(z *html.Tokenizer, attrName string) string {
	for {
		key, val, more := z.TagAttr()
		if string(key) == attrName {
			return string(val)
		}
		if !more {
			break
		}
	}
	return ""
}

func resolveURL(href string, base *url.URL) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") || strings.HasPrefix(href, "data:") {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(u)
	return resolved.String()
}

func isInternalURL(rawURL, domain string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// normalizeText collapses whitespace runs (tabs, newlines, spaces) into single spaces
// and trims leading/trailing whitespace, matching browser rendering behavior.
func normalizeText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func isHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
}

// parseMetaRefreshURL extracts the URL from a meta refresh content attribute.
// Format: "seconds; url=URL" or "seconds;URL='quoted'" or just "seconds; url=URL"
func parseMetaRefreshURL(content string) string {
	lower := strings.ToLower(content)
	idx := strings.Index(lower, "url=")
	if idx < 0 {
		return ""
	}
	u := strings.TrimSpace(content[idx+4:])
	if u == "" {
		return ""
	}
	// Remove optional surrounding quotes
	if len(u) >= 2 && (u[0] == '\'' || u[0] == '"') {
		quote := u[0]
		if end := strings.IndexByte(u[1:], quote); end >= 0 {
			u = u[1 : end+1]
		} else {
			u = u[1:]
		}
	}
	return u
}
