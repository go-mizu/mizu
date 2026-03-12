package export

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// MarkdownExporter writes crawled pages as navigable markdown files. Thread-safe.
type MarkdownExporter struct {
	domain  string
	siteDir string
	mu      sync.Mutex
	written map[string]bool // URL path → written

	// ConvertFn converts raw HTML to markdown. Caller injects this to avoid
	// a direct dependency on pkg/markdown from pkg/export.
	ConvertFn func(html []byte, pageURL string) (title, markdown string)
}

// NewMarkdownExporter creates a markdown exporter.
// convertFn must convert raw HTML bytes + page URL to (title, markdown body).
func NewMarkdownExporter(cfg Config, convertFn func(html []byte, pageURL string) (string, string)) (*MarkdownExporter, error) {
	if cfg.Domain == "" {
		return nil, fmt.Errorf("export: domain is required")
	}
	siteDir := filepath.Join(cfg.OutDir, "markdown", cfg.Domain)
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		return nil, fmt.Errorf("export: create output dir: %w", err)
	}
	return &MarkdownExporter{
		domain:    cfg.Domain,
		siteDir:   siteDir,
		written:   make(map[string]bool),
		ConvertFn: convertFn,
	}, nil
}

// WritePage converts HTML to markdown and writes it with rewritten links.
func (e *MarkdownExporter) WritePage(p Page) (string, error) {
	u, err := url.Parse(p.URL)
	if err != nil {
		return "", fmt.Errorf("parse url %q: %w", p.URL, err)
	}

	localPath := urlToMdPath(u.Path)
	fullPath := filepath.Join(e.siteDir, localPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	title, md := e.ConvertFn(p.HTML, p.URL)
	if md == "" {
		e.mu.Lock()
		e.written[u.Path] = true
		e.mu.Unlock()
		return localPath, nil
	}

	// Rewrite internal markdown links
	md = e.rewriteMarkdownLinks(md, u)

	// Build frontmatter
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("url: %s\n", p.URL))
	if title != "" {
		sb.WriteString(fmt.Sprintf("title: %s\n", escapeMdFrontmatter(title)))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(md)
	sb.WriteString("\n")

	if err := os.WriteFile(fullPath, []byte(sb.String()), 0o644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	e.mu.Lock()
	e.written[u.Path] = true
	e.mu.Unlock()
	return localPath, nil
}

// WriteIndex generates a markdown site index listing all exported pages.
func (e *MarkdownExporter) WriteIndex() error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", e.domain))
	sb.WriteString(fmt.Sprintf("%d pages exported\n\n", len(e.written)))

	paths := make([]string, 0, len(e.written))
	for urlPath := range e.written {
		paths = append(paths, urlPath)
	}
	sort.Strings(paths)

	for _, urlPath := range paths {
		localPath := urlToMdPath(urlPath)
		sb.WriteString(fmt.Sprintf("- [%s](%s)\n", urlPath, localPath))
	}

	indexPath := filepath.Join(e.siteDir, "_index.md")
	return os.WriteFile(indexPath, []byte(sb.String()), 0o644)
}

// Pages returns the number of pages written.
func (e *MarkdownExporter) Pages() int {
	return len(e.written)
}

// mdLinkRe matches markdown links: [text](url)
var mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

// rewriteMarkdownLinks rewrites internal [text](url) links to relative .md paths.
func (e *MarkdownExporter) rewriteMarkdownLinks(md string, pageURL *url.URL) string {
	pageLocalPath := urlToMdPath(pageURL.Path)
	pageDir := path.Dir(pageLocalPath)

	return mdLinkRe.ReplaceAllStringFunc(md, func(match string) string {
		sub := mdLinkRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		text := sub[1]
		href := sub[2]

		if isSpecialURL(href) {
			return match
		}

		resolved := resolveURL(href, pageURL)
		if resolved == nil {
			return match
		}

		if !e.isSameDomain(resolved) {
			return match
		}

		targetLocal := urlToMdPath(resolved.Path)
		rel, err := filepath.Rel(pageDir, targetLocal)
		if err != nil {
			return match
		}
		rel = filepath.ToSlash(rel)
		if resolved.Fragment != "" {
			rel += "#" + resolved.Fragment
		}
		return fmt.Sprintf("[%s](%s)", text, rel)
	})
}

func (e *MarkdownExporter) isSameDomain(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	return host == e.domain || host == "www."+e.domain
}

// urlToMdPath converts a URL path to a local markdown file path.
// /           → index.md
// /about      → about/index.md
// /page.html  → page.md
func urlToMdPath(urlPath string) string {
	urlPath = strings.TrimSpace(urlPath)
	if urlPath == "" || urlPath == "/" {
		return "index.md"
	}
	urlPath = strings.TrimPrefix(urlPath, "/")
	urlPath = strings.TrimSuffix(urlPath, "/")

	ext := path.Ext(urlPath)
	if ext != "" && len(ext) <= 6 {
		// Replace extension with .md
		return urlPath[:len(urlPath)-len(ext)] + ".md"
	}

	return urlPath + "/index.md"
}

func escapeMdFrontmatter(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if strings.ContainsAny(s, ":\"'{}[]|>&*!%#`@,") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}
