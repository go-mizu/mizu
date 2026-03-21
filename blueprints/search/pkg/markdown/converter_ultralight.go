package markdown

import (
	"bytes"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ConvertUltraLight is a zero-DOM HTML→Markdown converter. Instead of building
// a full DOM tree via html.Parse (which dominates CPU — 46% per pprof), it uses
// html.Tokenizer for single-pass streaming extraction. This eliminates:
//   - html.Parse DOM tree allocation (thousands of html.Node objects per page)
//   - Multiple tree walks (stripBoilerplate, findContentRegion, subtreeSize)
//   - GC pressure from short-lived DOM trees
//
// Tradeoffs vs ConvertLight:
//   - No text-density scoring (can't walk subtrees without a tree)
//   - No findBestContentBlock fallback (no tree to score)
//   - Slightly less precise boilerplate removal
//   - ~5-10× faster per document
func ConvertUltraLight(rawHTML []byte, pageURL string) Result {
	start := time.Now()
	htmlSize := len(rawHTML)

	tkn := html.NewTokenizer(bytes.NewReader(rawHTML))
	var b strings.Builder
	b.Grow(htmlSize / 5)

	var title string
	var inBody bool
	var skipDepth int    // > 0 means we're inside a tag to skip
	var inPre bool       // inside <pre>
	var inTitle bool     // inside <title>
	var inAnchor bool    // inside <a>
	var anchorHref string
	var anchorText strings.Builder

	// Content region tracking: prefer <article>/<main> if found.
	// We buffer all output; if we ever enter an <article>/<main>, we
	// reset and only keep content from within that region.
	var regionDepth int // > 0 means we're inside a content region
	var foundRegion bool
	var preRegionLen int // length of b before entering first region

	for {
		tt := tkn.Next()
		if tt == html.ErrorToken {
			break
		}

		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			tn, _ := tkn.TagName()
			tag := atom.Lookup(tn)

			// Extract <title> content
			if tag == atom.Title && !inBody {
				inTitle = true
				continue
			}
			if tag == atom.Body {
				inBody = true
				continue
			}
			if !inBody {
				// Check for og:title in <meta>
				if tag == atom.Meta && title == "" {
					title = extractMetaTitle(tkn)
				}
				continue
			}

			// Inside a skip region — just track depth
			if skipDepth > 0 {
				if tt != html.SelfClosingTagToken {
					skipDepth++
				}
				continue
			}

			// Tags to always skip (boilerplate + non-content)
			switch tag {
			case atom.Script, atom.Style, atom.Noscript, atom.Iframe,
				atom.Svg, atom.Form, atom.Button, atom.Input,
				atom.Select, atom.Textarea, atom.Template:
				skipDepth = 1
				continue
			case atom.Nav, atom.Footer:
				// Skip nav/footer (usually boilerplate)
				skipDepth = 1
				continue
			}

			// Content region detection
			if tag == atom.Article || tag == atom.Main {
				if !foundRegion {
					foundRegion = true
					preRegionLen = b.Len()
					b.Reset()
					b.Grow(htmlSize / 5)
				}
				regionDepth++
			}

			// If we found a content region but we're outside it, skip
			if foundRegion && regionDepth == 0 {
				continue
			}

			// Emit markdown formatting
			switch tag {
			case atom.H1:
				b.WriteString("\n\n# ")
			case atom.H2:
				b.WriteString("\n\n## ")
			case atom.H3:
				b.WriteString("\n\n### ")
			case atom.H4:
				b.WriteString("\n\n#### ")
			case atom.H5:
				b.WriteString("\n\n##### ")
			case atom.H6:
				b.WriteString("\n\n###### ")
			case atom.P, atom.Div, atom.Section, atom.Blockquote,
				atom.Dd, atom.Dt, atom.Figure, atom.Figcaption,
				atom.Details, atom.Summary:
				b.WriteString("\n\n")
			case atom.Br:
				b.WriteString("  \n")
			case atom.Hr:
				b.WriteString("\n\n---\n\n")
			case atom.Strong, atom.B:
				b.WriteString("**")
			case atom.Em, atom.I:
				b.WriteByte('*')
			case atom.Del, atom.S:
				b.WriteString("~~")
			case atom.Pre:
				b.WriteString("\n\n```\n")
				inPre = true
			case atom.Code:
				if !inPre {
					b.WriteByte('`')
				}
			case atom.Ul, atom.Ol:
				b.WriteByte('\n')
			case atom.Li:
				b.WriteString("\n- ")
			case atom.A:
				inAnchor = true
				anchorHref = extractHref(tkn)
				anchorText.Reset()
			case atom.Img:
				alt, src := extractImgAttrs(tkn)
				if src != "" {
					b.WriteString("![")
					b.WriteString(alt)
					b.WriteString("](")
					b.WriteString(src)
					b.WriteByte(')')
				}
			}

		case html.EndTagToken:
			tn, _ := tkn.TagName()
			tag := atom.Lookup(tn)

			if tag == atom.Title {
				inTitle = false
				continue
			}
			if !inBody {
				continue
			}

			if skipDepth > 0 {
				skipDepth--
				continue
			}

			// Content region end
			if tag == atom.Article || tag == atom.Main {
				if regionDepth > 0 {
					regionDepth--
				}
			}

			if foundRegion && regionDepth == 0 && tag != atom.Article && tag != atom.Main {
				continue
			}

			switch tag {
			case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
				b.WriteString("\n\n")
			case atom.P, atom.Div, atom.Section, atom.Blockquote,
				atom.Dd, atom.Dt, atom.Figure, atom.Figcaption,
				atom.Details, atom.Summary:
				b.WriteString("\n\n")
			case atom.Strong, atom.B:
				b.WriteString("**")
			case atom.Em, atom.I:
				b.WriteByte('*')
			case atom.Del, atom.S:
				b.WriteString("~~")
			case atom.Pre:
				b.WriteString("\n```\n\n")
				inPre = false
			case atom.Code:
				if !inPre {
					b.WriteByte('`')
				}
			case atom.A:
				if inAnchor {
					text := strings.TrimSpace(anchorText.String())
					if anchorHref != "" && text != "" {
						b.WriteByte('[')
						b.WriteString(text)
						b.WriteString("](")
						b.WriteString(anchorHref)
						b.WriteByte(')')
					} else if text != "" {
						b.WriteString(text)
					}
					inAnchor = false
				}
			}

		case html.TextToken:
			if inTitle {
				if title == "" {
					title = strings.TrimSpace(string(tkn.Text()))
				}
				continue
			}
			if !inBody || skipDepth > 0 {
				continue
			}
			if foundRegion && regionDepth == 0 {
				continue
			}

			text := tkn.Text()
			if inAnchor {
				anchorText.Write(text)
				continue
			}
			if inPre {
				b.Write(text)
			} else {
				writeCollapsedBytes(&b, text)
			}
		}
	}

	_ = preRegionLen // used to track reset point

	md := collapseBlankLines(strings.TrimSpace(b.String()))
	if len(md) < 10 {
		ms := int(time.Since(start).Milliseconds())
		return Result{HTMLSize: htmlSize, ConvertMs: ms, Error: "no content extracted"}
	}

	mdSize := len(md)
	ms := int(time.Since(start).Milliseconds())

	return Result{
		Markdown:       md,
		Title:          title,
		HasContent:     true,
		HTMLSize:       htmlSize,
		MarkdownSize:   mdSize,
		HTMLTokens:     EstimateTokens(htmlSize),
		MarkdownTokens: EstimateTokens(mdSize),
		ConvertMs:      ms,
	}
}

// extractMetaTitle extracts og:title or twitter:title from current meta tag attributes.
func extractMetaTitle(tkn *html.Tokenizer) string {
	var prop, content string
	for {
		key, val, more := tkn.TagAttr()
		k := string(key)
		v := string(val)
		switch k {
		case "property", "name":
			prop = v
		case "content":
			content = v
		}
		if !more {
			break
		}
	}
	if (prop == "og:title" || prop == "twitter:title") && content != "" {
		return strings.TrimSpace(content)
	}
	return ""
}

// extractHref returns the href attribute value from the current tag.
func extractHref(tkn *html.Tokenizer) string {
	for {
		key, val, more := tkn.TagAttr()
		if string(key) == "href" {
			return string(val)
		}
		if !more {
			break
		}
	}
	return ""
}

// extractImgAttrs returns alt and src from the current img tag.
func extractImgAttrs(tkn *html.Tokenizer) (alt, src string) {
	for {
		key, val, more := tkn.TagAttr()
		switch string(key) {
		case "alt":
			alt = string(val)
		case "src":
			src = string(val)
		}
		if !more {
			break
		}
	}
	return
}

// writeCollapsedBytes writes text bytes to b, collapsing whitespace runs to single spaces.
func writeCollapsedBytes(b *strings.Builder, text []byte) {
	inSpace := false
	for _, c := range text {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		} else {
			b.WriteByte(c)
			inSpace = false
		}
	}
}
