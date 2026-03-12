package markdown

import (
	"bytes"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	readability "github.com/go-shiori/go-readability"
	trafilatura "github.com/go-mizu/mizu/blueprints/search/pkg/trafilatura"
	"golang.org/x/net/html"
	htmlcharset "golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// mdConverterPool reuses html-to-markdown converters to cut per-call allocation.
var mdConverterPool = sync.Pool{
	New: func() any {
		return converter.NewConverter(
			converter.WithPlugins(
				base.NewBasePlugin(),
				commonmark.NewCommonmarkPlugin(),
			),
		)
	},
}

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
	// Skip htmldate extraction entirely — we already have WARC-Date from
	// Common Crawl headers. htmldate (with go-dateparser regex) accounts for
	// ~40% of trafilatura time; disabling it is the single biggest speedup.
	opts.HtmlDateMode = trafilatura.Disabled

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

	// Step 2: convert extracted DOM directly to markdown using fastMarkdown.
	// This replaces the previous render → reparse → html-to-markdown pipeline,
	// eliminating two full DOM traversals and the html-to-markdown plugin overhead.
	md := fastMarkdown(extracted.ContentNode)

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

	md, err := convertStringToMarkdown(article.Content, pageURL)
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

// convertNodeToMarkdown converts an *html.Node to trimmed markdown string
// using a pooled converter to reduce allocations.
func convertNodeToMarkdown(node *html.Node, pageURL string) (string, error) {
	conv := mdConverterPool.Get().(*converter.Converter)
	defer mdConverterPool.Put(conv)

	var opts []converter.ConvertOptionFunc
	if pageURL != "" {
		opts = append(opts, converter.WithDomain(pageURL))
	}
	mdBytes, err := conv.ConvertNode(node, opts...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(mdBytes)), nil
}

// convertStringToMarkdown converts an HTML string to trimmed markdown string
// using a pooled converter to reduce allocations.
func convertStringToMarkdown(htmlStr string, pageURL string) (string, error) {
	conv := mdConverterPool.Get().(*converter.Converter)
	defer mdConverterPool.Put(conv)

	var opts []converter.ConvertOptionFunc
	if pageURL != "" {
		opts = append(opts, converter.WithDomain(pageURL))
	}
	mdStr, err := conv.ConvertString(htmlStr, opts...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(mdStr), nil
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

