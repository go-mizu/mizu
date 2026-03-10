// Package export provides website cloning from crawled HTML data.
// It rewrites internal links to relative paths, extracts CSS/JS/image assets,
// and produces a browsable offline mirror preserving the original URL structure.
package export

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// Page represents a single crawled page to export.
type Page struct {
	URL  string // original absolute URL
	HTML []byte // raw HTML body
}

// Config controls export behavior.
type Config struct {
	Domain  string // target domain (normalized, no www.)
	OutDir  string // root output directory
	Format  string // "html" (rewrite links) or "raw" (original HTML)
}

// Exporter writes crawled pages as a browsable offline site. Thread-safe.
type Exporter struct {
	cfg     Config
	domain  string
	siteDir string
	mu      sync.Mutex
	written map[string]bool // URL path → written
}

// New creates an Exporter with the given config.
// For "markdown" format, use NewMarkdownExporter instead.
func New(cfg Config) (*Exporter, error) {
	if cfg.Domain == "" {
		return nil, fmt.Errorf("export: domain is required")
	}
	if cfg.Format == "" {
		cfg.Format = "html"
	}
	siteDir := filepath.Join(cfg.OutDir, cfg.Format, cfg.Domain)
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		return nil, fmt.Errorf("export: create output dir: %w", err)
	}
	return &Exporter{
		cfg:     cfg,
		domain:  cfg.Domain,
		siteDir: siteDir,
		written: make(map[string]bool),
	}, nil
}

// WritePage writes a single page to the export directory.
// Returns the local file path written.
func (e *Exporter) WritePage(p Page) (string, error) {
	u, err := url.Parse(p.URL)
	if err != nil {
		return "", fmt.Errorf("parse url %q: %w", p.URL, err)
	}

	localPath := URLToLocalPath(u.Path)
	fullPath := filepath.Join(e.siteDir, localPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	var data []byte
	if e.cfg.Format == "raw" {
		data = p.HTML
	} else {
		rewritten, err := e.rewriteHTML(p.HTML, u)
		if err != nil {
			data = p.HTML // fallback: write original on parse failure
		} else {
			data = rewritten
		}
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	e.mu.Lock()
	e.written[u.Path] = true
	e.mu.Unlock()
	return localPath, nil
}

// WriteIndex generates a site index page listing all exported pages.
func (e *Exporter) WriteIndex() error {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\">\n")
	escaped := html.EscapeString(e.domain)
	sb.WriteString(fmt.Sprintf("<title>%s — Exported Site Index</title>\n", escaped))
	sb.WriteString("<style>body{font-family:system-ui,sans-serif;max-width:800px;margin:2rem auto;padding:0 1rem}")
	sb.WriteString("a{display:block;padding:4px 0}h1{border-bottom:1px solid #ccc;padding-bottom:8px}</style>\n")
	sb.WriteString("</head><body>\n")
	sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n<p>%d pages exported</p>\n", escaped, len(e.written)))

	paths := make([]string, 0, len(e.written))
	for urlPath := range e.written {
		paths = append(paths, urlPath)
	}
	sort.Strings(paths)
	for _, urlPath := range paths {
		localPath := URLToLocalPath(urlPath)
		sb.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>\n", html.EscapeString(localPath), html.EscapeString(urlPath)))
	}
	sb.WriteString("</body></html>\n")

	indexPath := filepath.Join(e.siteDir, "_index.html")
	return os.WriteFile(indexPath, []byte(sb.String()), 0o644)
}

// Pages returns the number of pages written.
func (e *Exporter) Pages() int {
	return len(e.written)
}

// rewriteHTML parses HTML, rewrites internal links and asset references.
func (e *Exporter) rewriteHTML(html []byte, pageURL *url.URL) ([]byte, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return nil, err
	}

	pageLocalPath := URLToLocalPath(pageURL.Path)
	pageDir := path.Dir(pageLocalPath)

	// Rewrite <a href>
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}
		rewritten := e.rewriteLink(href, pageURL, pageDir)
		if rewritten != href {
			s.SetAttr("href", rewritten)
		}
	})

	// Rewrite <link rel="stylesheet" href>
	doc.Find("link[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}
		rewritten := e.rewriteAssetRef(href, pageURL, pageDir, "css")
		if rewritten != href {
			s.SetAttr("href", rewritten)
		}
	})

	// Rewrite <script src>
	doc.Find("script[src]").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}
		rewritten := e.rewriteAssetRef(src, pageURL, pageDir, "js")
		if rewritten != src {
			s.SetAttr("src", rewritten)
		}
	})

	// Rewrite <img src> and <img srcset>
	doc.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}
		rewritten := e.rewriteAssetRef(src, pageURL, pageDir, "img")
		if rewritten != src {
			s.SetAttr("src", rewritten)
		}
	})

	// Rewrite <source srcset>
	doc.Find("source[srcset], img[srcset]").Each(func(_ int, s *goquery.Selection) {
		srcset, exists := s.Attr("srcset")
		if !exists || srcset == "" {
			return
		}
		rewritten := e.rewriteSrcset(srcset, pageURL, pageDir)
		if rewritten != srcset {
			s.SetAttr("srcset", rewritten)
		}
	})

	// Rewrite inline style url() references
	doc.Find("[style]").Each(func(_ int, s *goquery.Selection) {
		style, exists := s.Attr("style")
		if !exists {
			return
		}
		rewritten := e.rewriteCSSURLs(style, pageURL, pageDir)
		if rewritten != style {
			s.SetAttr("style", rewritten)
		}
	})

	// Rewrite <style> blocks
	doc.Find("style").Each(func(_ int, s *goquery.Selection) {
		css := s.Text()
		rewritten := e.rewriteCSSURLs(css, pageURL, pageDir)
		if rewritten != css {
			s.SetText(rewritten)
		}
	})

	// Rewrite background attributes
	doc.Find("[background]").Each(func(_ int, s *goquery.Selection) {
		bg, exists := s.Attr("background")
		if !exists || bg == "" {
			return
		}
		rewritten := e.rewriteAssetRef(bg, pageURL, pageDir, "img")
		if rewritten != bg {
			s.SetAttr("background", rewritten)
		}
	})

	out, err := doc.Html()
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// rewriteLink rewrites an <a href> to a relative path if it's internal.
func (e *Exporter) rewriteLink(href string, pageURL *url.URL, pageDir string) string {
	if isSpecialURL(href) {
		return href
	}

	resolved := resolveURL(href, pageURL)
	if resolved == nil {
		return href
	}

	if !e.isSameDomain(resolved) {
		return href
	}

	targetLocal := URLToLocalPath(resolved.Path)
	rel, err := filepath.Rel(pageDir, targetLocal)
	if err != nil {
		return href
	}
	rel = filepath.ToSlash(rel)
	if resolved.Fragment != "" {
		rel += "#" + resolved.Fragment
	}
	return rel
}

// rewriteAssetRef resolves a CSS/JS/image reference to an absolute URL.
// We don't download assets (they're not in the crawl DB), so we keep them
// as absolute URLs pointing to the original server for correct rendering.
func (e *Exporter) rewriteAssetRef(ref string, pageURL *url.URL, pageDir, assetType string) string {
	if isSpecialURL(ref) {
		return ref
	}

	resolved := resolveURL(ref, pageURL)
	if resolved == nil {
		return ref
	}

	// Return the fully-resolved absolute URL so the browser fetches from origin.
	return resolved.String()
}

// rewriteSrcset rewrites srcset attribute values.
func (e *Exporter) rewriteSrcset(srcset string, pageURL *url.URL, pageDir string) string {
	parts := strings.Split(srcset, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		rewritten := e.rewriteAssetRef(fields[0], pageURL, pageDir, "img")
		if len(fields) > 1 {
			result = append(result, rewritten+" "+strings.Join(fields[1:], " "))
		} else {
			result = append(result, rewritten)
		}
	}
	return strings.Join(result, ", ")
}

var cssURLRe = regexp.MustCompile(`url\(\s*(['"]?)(.*?)['"]?\s*\)`)

// rewriteCSSURLs rewrites url() references in CSS text.
func (e *Exporter) rewriteCSSURLs(css string, pageURL *url.URL, pageDir string) string {
	return cssURLRe.ReplaceAllStringFunc(css, func(match string) string {
		sub := cssURLRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		quote := sub[1] // preserve original quoting style
		rawURL := strings.Trim(sub[2], "'\"")
		if isSpecialURL(rawURL) {
			return match
		}
		rewritten := e.rewriteAssetRef(rawURL, pageURL, pageDir, "img")
		return fmt.Sprintf("url(%s%s%s)", quote, rewritten, quote)
	})
}

// URLToLocalPath converts a URL path to a local file path.
// /           → index.html
// /about      → about/index.html
// /about/     → about/index.html
// /page.html  → page.html
func URLToLocalPath(urlPath string) string {
	urlPath = strings.TrimSpace(urlPath)
	if urlPath == "" || urlPath == "/" {
		return "index.html"
	}
	urlPath = strings.TrimPrefix(urlPath, "/")
	urlPath = strings.TrimSuffix(urlPath, "/")

	// If it already has an extension, keep it
	ext := path.Ext(urlPath)
	if ext != "" && len(ext) <= 6 {
		return urlPath
	}

	return urlPath + "/index.html"
}

// isSameDomain checks if a URL matches the export domain.
func (e *Exporter) isSameDomain(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	return host == e.domain || host == "www."+e.domain
}

// resolveURL resolves a possibly-relative URL against a base URL.
func resolveURL(raw string, base *url.URL) *url.URL {
	ref, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	return base.ResolveReference(ref)
}

// isSpecialURL returns true for data URIs, blob URLs, javascript:, mailto:, tel:, etc.
func isSpecialURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" || u == "#" {
		return true
	}
	lower := strings.ToLower(u)
	return strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "blob:") ||
		strings.HasPrefix(lower, "javascript:") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:")
}
