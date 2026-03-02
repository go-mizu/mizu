package markdown

import (
	"bytes"
	"net/url"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	readability "github.com/go-shiori/go-readability"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
	htmlcharset "golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// Result holds the output of a single HTML → Markdown conversion.
type Result struct {
	Markdown   string
	Title      string
	Language   string
	HasContent bool // trafilatura found main content

	HTMLSize       int
	MarkdownSize   int
	HTMLTokens     int
	MarkdownTokens int
	ConvertMs      int
	Error          string
}

// Convert extracts readable content from raw HTML and converts it to Markdown.
// The pageURL is used for resolving relative links; it may be empty.
func Convert(rawHTML []byte, pageURL string) Result {
	start := time.Now()
	htmlSize := len(rawHTML)

	var opts trafilatura.Options
	opts.EnableFallback = true
	opts.ExcludeComments = true
	opts.IncludeLinks = true
	opts.IncludeImages = false
	opts.Focus = trafilatura.FavorRecall
	opts.Deduplicate = true

	if pageURL != "" {
		if u, err := url.Parse(pageURL); err == nil {
			opts.OriginalURL = u
		}
	}

	// Step 1: parse HTML and extract main content via trafilatura.
	// trafilatura.Extract() calls dom.Parse() internally, which runs chardet's
	// n-gram charset detection over the ENTIRE document — a major bottleneck.
	// Instead we call parseHTMLFast() which uses charset.DetermineEncoding()
	// (BOM + <meta charset> scan of the first 1024 bytes only) then hands the
	// parsed *html.Node directly to trafilatura.ExtractDocument(). All extraction
	// features (dedup, fallback, language detection, etc.) are preserved.
	doc, parseErr := parseHTMLFast(rawHTML)
	if parseErr != nil {
		ms := int(time.Since(start).Milliseconds())
		return Result{HTMLSize: htmlSize, ConvertMs: ms, Error: "html parse: " + parseErr.Error()}
	}
	extracted, err := trafilatura.ExtractDocument(doc, opts)
	if err != nil || extracted == nil || extracted.ContentNode == nil {
		ms := int(time.Since(start).Milliseconds())
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "no content extracted"
		}
		return Result{
			HTMLSize:  htmlSize,
			ConvertMs: ms,
			Error:     errMsg,
		}
	}

	title := extracted.Metadata.Title
	lang := extracted.Metadata.Language

	// Step 2: render extracted DOM back to HTML string.
	// We render rather than calling ConvertNode directly because html-to-markdown's
	// collapse pass expects a complete normalised document; trafilatura's ContentNode
	// is a partial fragment that html.Render + html.Parse normalises through Go's
	// HTML parser. Benchmarks confirm the render+reparse is faster and uses fewer
	// allocs than passing the raw fragment to ConvertNode.
	var buf strings.Builder
	if err := html.Render(&buf, extracted.ContentNode); err != nil {
		ms := int(time.Since(start).Milliseconds())
		return Result{
			HTMLSize:  htmlSize,
			Title:     title,
			Language:  lang,
			ConvertMs: ms,
			Error:     "html render: " + err.Error(),
		}
	}

	// Step 3: convert rendered HTML to markdown
	var convOpts []converter.ConvertOptionFunc
	if pageURL != "" {
		convOpts = append(convOpts, converter.WithDomain(pageURL))
	}
	mdBytes, err := htmltomarkdown.ConvertString(buf.String(), convOpts...)
	if err != nil {
		ms := int(time.Since(start).Milliseconds())
		return Result{
			HTMLSize:  htmlSize,
			Title:     title,
			Language:  lang,
			ConvertMs: ms,
			Error:     "md convert: " + err.Error(),
		}
	}

	md := strings.TrimSpace(mdBytes)
	mdSize := len(md)
	ms := int(time.Since(start).Milliseconds())

	return Result{
		Markdown:       md,
		Title:          title,
		Language:       lang,
		HasContent:     true,
		HTMLSize:       htmlSize,
		MarkdownSize:   mdSize,
		HTMLTokens:     EstimateTokens(htmlSize),
		MarkdownTokens: EstimateTokens(mdSize),
		ConvertMs:      ms,
	}
}

// ConvertFast extracts content using go-readability (Mozilla Readability.js port)
// and converts to Markdown. It is 3-8x faster than Convert at the cost of slightly
// lower extraction quality on noisy pages. Use --fast mode for bulk processing where
// throughput matters more than edge-case accuracy.
func ConvertFast(rawHTML []byte, pageURL string) Result {
	start := time.Now()
	htmlSize := len(rawHTML)

	var pageU *url.URL
	if pageURL != "" {
		if u, err := url.Parse(pageURL); err == nil {
			pageU = u
		}
	}

	article, err := readability.FromReader(bytes.NewReader(rawHTML), pageU)
	if err != nil || article.Length == 0 {
		ms := int(time.Since(start).Milliseconds())
		errMsg := "no content extracted"
		if err != nil {
			errMsg = err.Error()
		}
		return Result{
			HTMLSize:  htmlSize,
			ConvertMs: ms,
			Error:     errMsg,
		}
	}

	title := article.Title
	lang := article.Language

	// article.Content is already a normalised HTML string produced by go-readability,
	// so we can feed it directly to ConvertString without an extra html.Render pass.
	var convOpts []converter.ConvertOptionFunc
	if pageURL != "" {
		convOpts = append(convOpts, converter.WithDomain(pageURL))
	}
	mdBytes, err := htmltomarkdown.ConvertString(article.Content, convOpts...)
	if err != nil {
		ms := int(time.Since(start).Milliseconds())
		return Result{
			HTMLSize:  htmlSize,
			Title:     title,
			Language:  lang,
			ConvertMs: ms,
			Error:     "md convert: " + err.Error(),
		}
	}

	md := strings.TrimSpace(mdBytes)
	mdSize := len(md)
	ms := int(time.Since(start).Milliseconds())

	return Result{
		Markdown:       md,
		Title:          title,
		Language:       lang,
		HasContent:     true,
		HTMLSize:       htmlSize,
		MarkdownSize:   mdSize,
		HTMLTokens:     EstimateTokens(htmlSize),
		MarkdownTokens: EstimateTokens(mdSize),
		ConvertMs:      ms,
	}
}

// EstimateTokens approximates token count: ~4 bytes per token for English text.
func EstimateTokens(byteLen int) int {
	return (byteLen + 3) / 4
}

// parseHTMLFast parses rawHTML into an *html.Node without running chardet's
// full n-gram character-encoding detection over the whole document.
//
// Strategy:
//   - charset.DetermineEncoding scans only the first 1024 bytes for BOM and
//     <meta charset> / <meta http-equiv="Content-Type"> declarations.
//   - For UTF-8 (the vast majority of Common Crawl content), html.Parse is
//     called directly — no transcoding overhead at all.
//   - For pages that declare a non-UTF-8 charset in their <meta> tag or BOM
//     (e.g. GB2312, Shift-JIS), we transcode to UTF-8 via x/text/transform
//     before parsing, preserving correct extraction on those pages.
//   - Pages with no charset declaration default to UTF-8; the rare undeclared
//     non-UTF-8 page (< 1% of CC) will be garbled regardless of the detector.
func parseHTMLFast(rawHTML []byte) (*html.Node, error) {
	_, name, _ := htmlcharset.DetermineEncoding(rawHTML, "text/html")
	if name == "utf-8" {
		return html.Parse(bytes.NewReader(rawHTML))
	}
	enc, _ := htmlcharset.Lookup(name)
	if enc == nil {
		// Unknown encoding — fall back to UTF-8 assumption.
		return html.Parse(bytes.NewReader(rawHTML))
	}
	return html.Parse(transform.NewReader(bytes.NewReader(rawHTML), enc.NewDecoder()))
}

