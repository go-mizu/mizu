package dcrawler

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
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

	// Script content tracking for __NEXT_DATA__, JSON-LD, inline JS
	var inScript bool
	var scriptBuf strings.Builder
	var scriptType string // "next-data", "json-ld", "inline"

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
				switch {
				case len(tn) == 5 && string(tn) == "title":
					inTitle = true
				case len(tn) == 6 && string(tn) == "script":
					scriptType = "inline"
					inScript = true
					scriptBuf.Reset()
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
				case "next", "prev", "alternate", "prefetch", "preload", "prerender":
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

			case len(tn) == 6 && string(tn) == "script":
				sType, sID := extractScriptAttrs(tokenizer)
				if sID == "__NEXT_DATA__" {
					scriptType = "next-data"
					inScript = true
					scriptBuf.Reset()
				} else if strings.EqualFold(sType, "application/ld+json") {
					scriptType = "json-ld"
					inScript = true
					scriptBuf.Reset()
				} else {
					scriptType = "inline"
					inScript = true
					scriptBuf.Reset()
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
			if inScript {
				scriptBuf.Write(text)
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
			case len(tn) == 6 && string(tn) == "script":
				if inScript {
					content := scriptBuf.String()
					switch scriptType {
					case "next-data":
						for _, link := range extractNextDataLinks(content, effectiveBase, domain) {
							meta.Links = append(meta.Links, link)
						}
					case "json-ld":
						for _, link := range extractJSONLDLinks(content, effectiveBase, domain) {
							meta.Links = append(meta.Links, link)
						}
					case "inline":
						for _, link := range extractInlineJSLinks(content, effectiveBase, domain) {
							meta.Links = append(meta.Links, link)
						}
					}
					inScript = false
					scriptBuf.Reset()
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

// extractScriptAttrs extracts type and id from a <script> tag.
func extractScriptAttrs(z *html.Tokenizer) (sType, sID string) {
	for {
		key, val, more := z.TagAttr()
		switch string(key) {
		case "type":
			sType = string(val)
		case "id":
			sID = string(val)
		}
		if !more {
			break
		}
	}
	return
}

// extractNextDataLinks extracts internal URL paths from Next.js __NEXT_DATA__ JSON.
// The JSON contains route info, page props, and pre-fetched data with internal paths.
func extractNextDataLinks(content string, base *url.URL, domain string) []Link {
	var data map[string]any
	if json.Unmarshal([]byte(content), &data) != nil {
		return nil
	}
	seen := make(map[string]bool)
	var links []Link
	extractURLsFromJSON(data, base, domain, seen, &links, 0)
	return links
}

// extractJSONLDLinks extracts URLs from JSON-LD structured data.
func extractJSONLDLinks(content string, base *url.URL, domain string) []Link {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	seen := make(map[string]bool)
	var links []Link

	// JSON-LD can be a single object or array
	if content[0] == '[' {
		var arr []map[string]any
		if json.Unmarshal([]byte(content), &arr) == nil {
			for _, obj := range arr {
				extractJSONLDURLs(obj, base, domain, seen, &links)
			}
		}
	} else {
		var obj map[string]any
		if json.Unmarshal([]byte(content), &obj) == nil {
			extractJSONLDURLs(obj, base, domain, seen, &links)
		}
	}
	return links
}

// jsonLDURLFields are JSON-LD fields that contain URLs.
var jsonLDURLFields = map[string]bool{
	"url": true, "@id": true, "mainentityofpage": true,
	"sameas": true, "image": true, "logo": true,
	"thumbnailurl": true, "contenturl": true,
}

func extractJSONLDURLs(obj map[string]any, base *url.URL, domain string, seen map[string]bool, links *[]Link) {
	for k, v := range obj {
		key := strings.ToLower(k)
		switch val := v.(type) {
		case string:
			if jsonLDURLFields[key] && looksLikeURL(val) {
				resolved := resolveURL(val, base)
				if resolved != "" && !seen[resolved] {
					seen[resolved] = true
					*links = append(*links, Link{
						TargetURL:  resolved,
						Rel:        "json-ld",
						IsInternal: isInternalURL(resolved, domain),
					})
				}
			}
		case map[string]any:
			extractJSONLDURLs(val, base, domain, seen, links)
		case []any:
			for _, item := range val {
				switch inner := item.(type) {
				case string:
					if jsonLDURLFields[key] && looksLikeURL(inner) {
						resolved := resolveURL(inner, base)
						if resolved != "" && !seen[resolved] {
							seen[resolved] = true
							*links = append(*links, Link{
								TargetURL:  resolved,
								Rel:        "json-ld",
								IsInternal: isInternalURL(resolved, domain),
							})
						}
					}
				case map[string]any:
					extractJSONLDURLs(inner, base, domain, seen, links)
				}
			}
		}
	}
}

// extractURLsFromJSON recursively walks a JSON structure and extracts URL-like strings.
func extractURLsFromJSON(v any, base *url.URL, domain string, seen map[string]bool, links *[]Link, depth int) {
	if depth > 10 {
		return
	}
	switch val := v.(type) {
	case string:
		if isInternalPath(val) {
			resolved := resolveURL(val, base)
			if resolved != "" && !seen[resolved] && isInternalURL(resolved, domain) {
				seen[resolved] = true
				*links = append(*links, Link{
					TargetURL:  resolved,
					Rel:        "next-data",
					IsInternal: true,
				})
			}
		}
	case map[string]any:
		for _, child := range val {
			extractURLsFromJSON(child, base, domain, seen, links, depth+1)
		}
	case []any:
		for _, child := range val {
			extractURLsFromJSON(child, base, domain, seen, links, depth+1)
		}
	}
}

// inlineJSPathRe matches quoted internal paths in JavaScript: "/blog/some-post"
var inlineJSPathRe = regexp.MustCompile(`["'](/[a-zA-Z][^"'\\]{1,200})["']`)

// extractInlineJSLinks extracts internal URL paths from inline JavaScript.
func extractInlineJSLinks(content string, base *url.URL, domain string) []Link {
	// Only process scripts that look like they contain paths
	if !strings.Contains(content, `"/`) && !strings.Contains(content, `'/`) {
		return nil
	}
	// Skip very large scripts (minified bundles) — they're unlikely to have useful paths
	if len(content) > 100_000 {
		return nil
	}

	seen := make(map[string]bool)
	var links []Link
	for _, match := range inlineJSPathRe.FindAllStringSubmatch(content, 200) {
		path := match[1]
		if isJunkPath(path) {
			continue
		}
		resolved := resolveURL(path, base)
		if resolved != "" && !seen[resolved] && isInternalURL(resolved, domain) {
			seen[resolved] = true
			links = append(links, Link{
				TargetURL:  resolved,
				Rel:        "inline-js",
				IsInternal: true,
			})
		}
	}
	return links
}

// isInternalPath checks if a string looks like an internal URL path.
func isInternalPath(s string) bool {
	if len(s) < 2 || len(s) > 300 {
		return false
	}
	if s[0] != '/' || s[1] == '/' { // skip "//cdn.example.com"
		return false
	}
	// Must start with a letter after /
	c := s[1]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
		return false
	}
	// Skip asset paths
	if isJunkPath(s) {
		return false
	}
	return true
}

// isJunkPath returns true for paths that are clearly not crawlable pages.
func isJunkPath(s string) bool {
	lower := strings.ToLower(s)
	for _, ext := range []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".map"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	for _, prefix := range []string{"/_next/", "/_nuxt/", "/static/", "/assets/", "/webpack/", "/chunks/"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "/")
}
