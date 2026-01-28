package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/store/sqlite"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

// NewCrawl creates the crawl command
func NewCrawl() *cobra.Command {
	var (
		depth   int
		limit   int
		delay   int
		sitemap string
	)

	cmd := &cobra.Command{
		Use:   "crawl [url]",
		Short: "Crawl and index web pages",
		Long: `Crawl and index web pages starting from a URL.

Examples:
  search crawl https://golang.org
  search crawl https://example.com --depth 3 --limit 100
  search crawl --sitemap https://example.com/sitemap.xml`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var startURL string
			if len(args) > 0 {
				startURL = args[0]
			}
			return runCrawl(cmd.Context(), startURL, sitemap, depth, limit, delay)
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 2, "Maximum crawl depth")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum pages to crawl")
	cmd.Flags().IntVar(&delay, "delay", 1000, "Delay between requests in milliseconds")
	cmd.Flags().StringVar(&sitemap, "sitemap", "", "Sitemap URL to crawl")

	return cmd
}

func runCrawl(ctx context.Context, startURL, sitemapURL string, depth, limit, delay int) error {
	if startURL == "" && sitemapURL == "" {
		return fmt.Errorf("either a URL or --sitemap is required")
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Crawling web pages..."))
	fmt.Println()

	// Connect to database
	fmt.Println(infoStyle.Render("Opening SQLite database..."))
	s, err := sqlite.New(GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()
	fmt.Println(successStyle.Render("  Database opened"))

	crawler := &Crawler{
		store:    s,
		maxDepth: depth,
		maxPages: limit,
		delay:    time.Duration(delay) * time.Millisecond,
		visited:  make(map[string]bool),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	var crawled int
	if sitemapURL != "" {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Crawling sitemap: %s", sitemapURL)))
		crawled, err = crawler.CrawlSitemap(ctx, sitemapURL)
	} else {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Crawling URL: %s (depth: %d)", startURL, depth)))
		crawled, err = crawler.Crawl(ctx, startURL, 0)
	}

	if err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("Crawled and indexed %d pages", crawled)))
	fmt.Println()

	return nil
}

// Crawler handles web crawling
type Crawler struct {
	store    store.Store
	maxDepth int
	maxPages int
	delay    time.Duration
	visited  map[string]bool
	client   *http.Client
	crawled  int
}

// Crawl crawls a URL and its links
func (c *Crawler) Crawl(ctx context.Context, pageURL string, depth int) (int, error) {
	if depth > c.maxDepth || c.crawled >= c.maxPages {
		return c.crawled, nil
	}

	// Normalize URL
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return c.crawled, nil
	}
	normalizedURL := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path

	// Check if already visited
	if c.visited[normalizedURL] {
		return c.crawled, nil
	}
	c.visited[normalizedURL] = true

	// Fetch the page
	resp, err := c.client.Get(pageURL)
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Failed to fetch %s: %v", pageURL, err)))
		return c.crawled, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.crawled, nil
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return c.crawled, nil
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.crawled, nil
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return c.crawled, nil
	}

	// Extract data
	title := extractTitle(doc)
	description := extractMetaDescription(doc)
	content := extractTextContent(doc)
	links := extractLinks(doc, parsedURL)

	// Index the document
	document := &store.Document{
		URL:         normalizedURL,
		Title:       title,
		Description: description,
		Content:     content,
		Domain:      parsedURL.Host,
		Language:    "en",
		ContentType: "text/html",
		CrawledAt:   time.Now(),
	}

	if err := c.store.Index().IndexDocument(ctx, document); err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Failed to index %s: %v", pageURL, err)))
	} else {
		c.crawled++
		fmt.Println(successStyle.Render(fmt.Sprintf("  [%d] %s", c.crawled, title)))
	}

	// Delay before next request
	time.Sleep(c.delay)

	// Crawl links
	for _, link := range links {
		if c.crawled >= c.maxPages {
			break
		}
		select {
		case <-ctx.Done():
			return c.crawled, ctx.Err()
		default:
			c.Crawl(ctx, link, depth+1)
		}
	}

	return c.crawled, nil
}

// CrawlSitemap crawls URLs from a sitemap
func (c *Crawler) CrawlSitemap(ctx context.Context, sitemapURL string) (int, error) {
	resp, err := c.client.Get(sitemapURL)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read sitemap: %w", err)
	}

	// Simple XML parsing for sitemap URLs
	content := string(body)
	urls := extractSitemapURLs(content)

	for _, u := range urls {
		if c.crawled >= c.maxPages {
			break
		}
		select {
		case <-ctx.Done():
			return c.crawled, ctx.Err()
		default:
			c.Crawl(ctx, u, 0)
		}
	}

	return c.crawled, nil
}

// Helper functions

func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return strings.TrimSpace(n.FirstChild.Data)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := extractTitle(c); title != "" {
			return title
		}
	}
	return ""
}

func extractMetaDescription(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var name, content string
		for _, attr := range n.Attr {
			if attr.Key == "name" && attr.Val == "description" {
				name = attr.Val
			}
			if attr.Key == "content" {
				content = attr.Val
			}
		}
		if name == "description" {
			return content
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if desc := extractMetaDescription(c); desc != "" {
			return desc
		}
	}
	return ""
}

func extractTextContent(n *html.Node) string {
	var text strings.Builder
	extractText(n, &text)
	return strings.TrimSpace(text.String())
}

func extractText(n *html.Node, text *strings.Builder) {
	if n.Type == html.TextNode {
		text.WriteString(n.Data)
		text.WriteString(" ")
	}
	// Skip script, style, nav, footer, header
	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "nav", "footer", "header", "aside":
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, text)
	}
}

func extractLinks(n *html.Node, base *url.URL) []string {
	var links []string
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				link, err := base.Parse(attr.Val)
				if err == nil && link.Host == base.Host {
					links = append(links, link.String())
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links = append(links, extractLinks(c, base)...)
	}
	return links
}

func extractSitemapURLs(content string) []string {
	var urls []string
	// Simple extraction of URLs from sitemap XML
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<loc>") && strings.HasSuffix(line, "</loc>") {
			u := strings.TrimPrefix(line, "<loc>")
			u = strings.TrimSuffix(u, "</loc>")
			urls = append(urls, u)
		}
	}
	return urls
}
