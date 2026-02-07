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
func ExtractLinksAndMeta(body []byte, baseURL *url.URL, domain string) PageMeta {
	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	var meta PageMeta
	var inTitle bool
	var titleBuf strings.Builder

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			meta.Title = strings.TrimSpace(titleBuf.String())
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
				href, anchor, rel := extractAnchorAttrs(tokenizer)
				if href == "" {
					continue
				}
				resolved := resolveURL(href, baseURL)
				if resolved == "" {
					continue
				}
				link := Link{
					TargetURL:  resolved,
					AnchorText: truncate(anchor, 200),
					Rel:        rel,
					IsInternal: isInternalURL(resolved, domain),
				}
				meta.Links = append(meta.Links, link)

			case len(tn) == 4 && string(tn) == "link":
				rel, href := extractLinkAttrs(tokenizer)
				if rel == "canonical" && href != "" {
					meta.Canonical = resolveURL(href, baseURL)
				}

			case len(tn) == 4 && string(tn) == "meta":
				name, content := extractMetaAttrs(tokenizer)
				switch strings.ToLower(name) {
				case "description":
					meta.Description = truncate(content, 500)
				case "language":
					meta.Language = content
				}

			case len(tn) == 4 && string(tn) == "html":
				meta.Language = extractAttr(tokenizer, "lang")

			case len(tn) == 5 && string(tn) == "title":
				inTitle = true
			}

		case html.TextToken:
			if inTitle {
				titleBuf.Write(tokenizer.Text())
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			if len(tn) == 5 && string(tn) == "title" {
				inTitle = false
			}
		}
	}
}

func extractAnchorAttrs(z *html.Tokenizer) (href, anchor, rel string) {
	for {
		key, val, more := z.TagAttr()
		k := string(key)
		switch k {
		case "href":
			href = string(val)
		case "rel":
			rel = string(val)
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

func extractMetaAttrs(z *html.Tokenizer) (name, content string) {
	for {
		key, val, more := z.TagAttr()
		k := string(key)
		switch k {
		case "name":
			name = string(val)
		case "content":
			content = string(val)
		}
		if !more {
			break
		}
	}
	return
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
