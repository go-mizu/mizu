package export

import (
	"fmt"
	"html"
	"strings"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
)

// HTMLConverter converts pages to HTML format.
type HTMLConverter struct{}

// NewHTMLConverter creates a new HTML converter.
func NewHTMLConverter() *HTMLConverter {
	return &HTMLConverter{}
}

// Convert converts an exported page to HTML.
func (c *HTMLConverter) Convert(page *ExportedPage, opts *Request) ([]byte, error) {
	var sb strings.Builder

	// Write HTML document
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")

	// Title
	title := page.Title
	if title == "" {
		title = "Untitled"
	}
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(title)))

	// Embedded styles
	sb.WriteString("<style>\n")
	sb.WriteString(exportCSS)
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")

	// Page container
	sb.WriteString("<article class=\"page\">\n")

	// Cover image
	if page.Cover != "" && opts.IncludeImages {
		sb.WriteString(fmt.Sprintf("<div class=\"page-cover\"><img src=\"%s\" alt=\"Cover\"></div>\n", html.EscapeString(page.Cover)))
	}

	// Page header
	sb.WriteString("<header class=\"page-header\">\n")
	if page.Icon != "" {
		sb.WriteString(fmt.Sprintf("<span class=\"page-icon\">%s</span>\n", page.Icon))
	}
	if page.Title != "" {
		sb.WriteString(fmt.Sprintf("<h1 class=\"page-title\">%s</h1>\n", html.EscapeString(page.Title)))
	}
	sb.WriteString("</header>\n")

	// Page content
	sb.WriteString("<div class=\"page-content\">\n")
	c.convertBlocks(&sb, page.Blocks, opts)
	sb.WriteString("</div>\n")

	sb.WriteString("</article>\n")
	sb.WriteString("</body>\n</html>")

	return []byte(sb.String()), nil
}

// ContentType returns the MIME type.
func (c *HTMLConverter) ContentType() string {
	return "text/html; charset=utf-8"
}

// Extension returns the file extension.
func (c *HTMLConverter) Extension() string {
	return ".html"
}

// convertBlocks converts a list of blocks to HTML.
func (c *HTMLConverter) convertBlocks(sb *strings.Builder, blockList []*blocks.Block, opts *Request) {
	var listType blocks.BlockType
	var listOpen bool

	for _, block := range blockList {
		// Handle list grouping
		isListItem := block.Type == blocks.BlockBulletList || block.Type == blocks.BlockNumberList

		if isListItem {
			if !listOpen || listType != block.Type {
				// Close previous list if different type
				if listOpen {
					c.closeList(sb, listType)
				}
				// Open new list
				c.openList(sb, block.Type)
				listType = block.Type
				listOpen = true
			}
		} else {
			if listOpen {
				c.closeList(sb, listType)
				listOpen = false
			}
		}

		c.convertBlock(sb, block, opts)
	}

	// Close any remaining list
	if listOpen {
		c.closeList(sb, listType)
	}
}

func (c *HTMLConverter) openList(sb *strings.Builder, blockType blocks.BlockType) {
	if blockType == blocks.BlockBulletList {
		sb.WriteString("<ul>\n")
	} else {
		sb.WriteString("<ol>\n")
	}
}

func (c *HTMLConverter) closeList(sb *strings.Builder, blockType blocks.BlockType) {
	if blockType == blocks.BlockBulletList {
		sb.WriteString("</ul>\n")
	} else {
		sb.WriteString("</ol>\n")
	}
}

// convertBlock converts a single block to HTML.
func (c *HTMLConverter) convertBlock(sb *strings.Builder, block *blocks.Block, opts *Request) {
	switch block.Type {
	case blocks.BlockParagraph:
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<p>%s</p>\n", text))

	case blocks.BlockHeading1:
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", text))

	case blocks.BlockHeading2:
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<h2>%s</h2>\n", text))

	case blocks.BlockHeading3:
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<h3>%s</h3>\n", text))

	case blocks.BlockBulletList:
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<li>%s", text))
		if len(block.Children) > 0 {
			sb.WriteString("\n<ul>\n")
			for _, child := range block.Children {
				c.convertBlock(sb, child, opts)
			}
			sb.WriteString("</ul>\n")
		}
		sb.WriteString("</li>\n")

	case blocks.BlockNumberList:
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<li>%s", text))
		if len(block.Children) > 0 {
			sb.WriteString("\n<ol>\n")
			for _, child := range block.Children {
				c.convertBlock(sb, child, opts)
			}
			sb.WriteString("</ol>\n")
		}
		sb.WriteString("</li>\n")

	case blocks.BlockTodo:
		text := c.richTextToHTML(block.Content.RichText, opts)
		checked := ""
		checkedClass := ""
		if block.Content.Checked != nil && *block.Content.Checked {
			checked = " checked"
			checkedClass = " checked"
		}
		sb.WriteString(fmt.Sprintf("<div class=\"todo%s\"><input type=\"checkbox\" disabled%s><span>%s</span></div>\n", checkedClass, checked, text))

	case blocks.BlockToggle:
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString("<details class=\"toggle\">\n")
		sb.WriteString(fmt.Sprintf("<summary>%s</summary>\n", text))
		sb.WriteString("<div class=\"toggle-content\">\n")
		if len(block.Children) > 0 {
			c.convertBlocks(sb, block.Children, opts)
		}
		sb.WriteString("</div>\n</details>\n")

	case blocks.BlockQuote:
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<blockquote>%s</blockquote>\n", text))

	case blocks.BlockCallout:
		icon := block.Content.Icon
		if icon == "" {
			icon = "💡"
		}
		color := block.Content.Color
		if color == "" {
			color = "default"
		}
		text := c.richTextToHTML(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<div class=\"callout callout-%s\">\n", color))
		sb.WriteString(fmt.Sprintf("<span class=\"callout-icon\">%s</span>\n", icon))
		sb.WriteString(fmt.Sprintf("<div class=\"callout-content\">%s", text))
		if len(block.Children) > 0 {
			sb.WriteString("\n")
			c.convertBlocks(sb, block.Children, opts)
		}
		sb.WriteString("</div>\n</div>\n")

	case blocks.BlockCode:
		lang := block.Content.Language
		text := c.richTextToPlainText(block.Content.RichText)
		langClass := ""
		if lang != "" {
			langClass = fmt.Sprintf(" class=\"language-%s\"", lang)
		}
		sb.WriteString(fmt.Sprintf("<pre><code%s>%s</code></pre>\n", langClass, html.EscapeString(text)))

	case blocks.BlockDivider:
		sb.WriteString("<hr>\n")

	case blocks.BlockImage:
		if opts.IncludeImages && block.Content.URL != "" {
			caption := c.richTextToPlainText(block.Content.Caption)
			sb.WriteString("<figure class=\"image\">\n")
			sb.WriteString(fmt.Sprintf("<img src=\"%s\" alt=\"%s\">\n", html.EscapeString(block.Content.URL), html.EscapeString(caption)))
			if caption != "" {
				sb.WriteString(fmt.Sprintf("<figcaption>%s</figcaption>\n", html.EscapeString(caption)))
			}
			sb.WriteString("</figure>\n")
		}

	case blocks.BlockVideo:
		if opts.IncludeFiles && block.Content.URL != "" {
			caption := c.richTextToPlainText(block.Content.Caption)
			sb.WriteString("<figure class=\"video\">\n")
			sb.WriteString(fmt.Sprintf("<video controls src=\"%s\"></video>\n", html.EscapeString(block.Content.URL)))
			if caption != "" {
				sb.WriteString(fmt.Sprintf("<figcaption>%s</figcaption>\n", html.EscapeString(caption)))
			}
			sb.WriteString("</figure>\n")
		}

	case blocks.BlockFile:
		if opts.IncludeFiles && block.Content.URL != "" {
			caption := c.richTextToPlainText(block.Content.Caption)
			if caption == "" {
				caption = "Download file"
			}
			sb.WriteString(fmt.Sprintf("<a href=\"%s\" class=\"file-link\" download>%s</a>\n", html.EscapeString(block.Content.URL), html.EscapeString(caption)))
		}

	case blocks.BlockBookmark:
		if block.Content.URL != "" {
			title := block.Content.Title
			if title == "" {
				title = block.Content.URL
			}
			desc := block.Content.Description
			sb.WriteString("<div class=\"bookmark\">\n")
			sb.WriteString(fmt.Sprintf("<a href=\"%s\" target=\"_blank\">\n", html.EscapeString(block.Content.URL)))
			sb.WriteString(fmt.Sprintf("<div class=\"bookmark-title\">%s</div>\n", html.EscapeString(title)))
			if desc != "" {
				sb.WriteString(fmt.Sprintf("<div class=\"bookmark-description\">%s</div>\n", html.EscapeString(desc)))
			}
			sb.WriteString(fmt.Sprintf("<div class=\"bookmark-url\">%s</div>\n", html.EscapeString(block.Content.URL)))
			sb.WriteString("</a>\n</div>\n")
		}

	case blocks.BlockEmbed:
		if block.Content.URL != "" {
			sb.WriteString(fmt.Sprintf("<iframe src=\"%s\" class=\"embed\" frameborder=\"0\" allowfullscreen></iframe>\n", html.EscapeString(block.Content.URL)))
		}

	case blocks.BlockEquation:
		text := c.richTextToPlainText(block.Content.RichText)
		sb.WriteString(fmt.Sprintf("<div class=\"equation\">%s</div>\n", html.EscapeString(text)))

	case blocks.BlockTable:
		c.convertTable(sb, block, opts)

	case blocks.BlockColumnList:
		colCount := len(block.Children)
		if colCount == 0 {
			colCount = 2
		}
		sb.WriteString(fmt.Sprintf("<div class=\"columns columns-%d\">\n", colCount))
		for _, col := range block.Children {
			if col.Type == blocks.BlockColumn {
				sb.WriteString("<div class=\"column\">\n")
				if len(col.Children) > 0 {
					c.convertBlocks(sb, col.Children, opts)
				}
				sb.WriteString("</div>\n")
			}
		}
		sb.WriteString("</div>\n")

	case blocks.BlockChildPage:
		text := c.richTextToPlainText(block.Content.RichText)
		if text == "" {
			text = "Subpage"
		}
		filename := sanitizeFilename(text) + ".html"
		sb.WriteString(fmt.Sprintf("<a href=\"%s\" class=\"child-page\">📄 %s</a>\n", filename, html.EscapeString(text)))

	case blocks.BlockChildDB, blocks.BlockLinkedDB:
		text := c.richTextToPlainText(block.Content.RichText)
		if text == "" {
			text = "Database"
		}
		sb.WriteString(fmt.Sprintf("<div class=\"child-database\">📊 %s</div>\n", html.EscapeString(text)))

	case blocks.BlockBreadcrumb:
		// Skip breadcrumbs in export

	case blocks.BlockTemplateButton:
		text := block.Content.ButtonText
		if text == "" {
			text = "Template"
		}
		sb.WriteString(fmt.Sprintf("<button class=\"template-button\" disabled>%s</button>\n", html.EscapeString(text)))

	case blocks.BlockSyncedBlock:
		// Export synced block content as regular content
		if len(block.Children) > 0 {
			c.convertBlocks(sb, block.Children, opts)
		}

	default:
		// Unknown block type - try to extract text
		text := c.richTextToHTML(block.Content.RichText, opts)
		if text != "" {
			sb.WriteString(fmt.Sprintf("<p>%s</p>\n", text))
		}
	}
}

// convertTable converts a table block to HTML table.
func (c *HTMLConverter) convertTable(sb *strings.Builder, block *blocks.Block, opts *Request) {
	if len(block.Children) == 0 {
		return
	}

	sb.WriteString("<table>\n")

	for i, row := range block.Children {
		if row.Type != blocks.BlockTableRow {
			continue
		}

		// Use thead for first row if table has header
		if i == 0 && block.Content.HasHeader {
			sb.WriteString("<thead>\n<tr>\n")
		} else if i == 1 && block.Content.HasHeader {
			sb.WriteString("</thead>\n<tbody>\n<tr>\n")
		} else if i == 0 {
			sb.WriteString("<tbody>\n<tr>\n")
		} else {
			sb.WriteString("<tr>\n")
		}

		tag := "td"
		if i == 0 && block.Content.HasHeader {
			tag = "th"
		}

		for _, cell := range row.Content.RichText {
			text := c.richTextToHTML([]blocks.RichText{cell}, opts)
			sb.WriteString(fmt.Sprintf("<%s>%s</%s>\n", tag, text, tag))
		}

		sb.WriteString("</tr>\n")
	}

	if block.Content.HasHeader && len(block.Children) > 1 {
		sb.WriteString("</tbody>\n")
	} else if len(block.Children) > 0 {
		sb.WriteString("</tbody>\n")
	}
	sb.WriteString("</table>\n")
}

// richTextToHTML converts rich text to HTML with formatting.
func (c *HTMLConverter) richTextToHTML(richText []blocks.RichText, opts *Request) string {
	var parts []string

	for _, rt := range richText {
		text := html.EscapeString(rt.Text)

		// Apply annotations
		if rt.Annotations.Code {
			text = "<code>" + text + "</code>"
		}
		if rt.Annotations.Bold {
			text = "<strong>" + text + "</strong>"
		}
		if rt.Annotations.Italic {
			text = "<em>" + text + "</em>"
		}
		if rt.Annotations.Strikethrough {
			text = "<del>" + text + "</del>"
		}
		if rt.Annotations.Underline {
			text = "<u>" + text + "</u>"
		}
		if rt.Annotations.Color != "" && rt.Annotations.Color != "default" {
			text = fmt.Sprintf("<span class=\"color-%s\">%s</span>", rt.Annotations.Color, text)
		}

		// Handle links
		if rt.Link != "" {
			text = fmt.Sprintf("<a href=\"%s\">%s</a>", html.EscapeString(rt.Link), text)
		}

		// Handle mentions
		if rt.Mention != nil {
			switch rt.Mention.Type {
			case "user":
				text = fmt.Sprintf("<span class=\"mention mention-user\">@%s</span>", text)
			case "page":
				text = fmt.Sprintf("<a href=\"%s.html\" class=\"mention mention-page\">%s</a>", rt.Mention.PageID, text)
			case "date":
				text = fmt.Sprintf("<span class=\"mention mention-date\">%s</span>", rt.Mention.Date)
			}
		}

		parts = append(parts, text)
	}

	return strings.Join(parts, "")
}

// richTextToPlainText converts rich text to plain text (no formatting).
func (c *HTMLConverter) richTextToPlainText(richText []blocks.RichText) string {
	var parts []string
	for _, rt := range richText {
		parts = append(parts, rt.Text)
	}
	return strings.Join(parts, "")
}

// exportCSS contains the embedded CSS for HTML exports.
const exportCSS = `
:root {
  --text-primary: #37352f;
  --text-secondary: #6b6b6b;
  --bg-primary: #ffffff;
  --bg-secondary: #f7f6f3;
  --border-color: #e3e2df;
  --accent-color: #2eaadc;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
  font-size: 16px;
  line-height: 1.5;
  color: var(--text-primary);
  background: var(--bg-primary);
  max-width: 900px;
  margin: 0 auto;
  padding: 40px 60px;
}

.page-cover {
  margin: -40px -60px 24px;
  height: 200px;
  overflow: hidden;
}

.page-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.page-header {
  margin-bottom: 24px;
}

.page-icon {
  font-size: 78px;
  line-height: 1;
  display: block;
  margin-bottom: 8px;
}

.page-title {
  font-size: 40px;
  font-weight: 700;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

h1 { font-size: 30px; font-weight: 700; margin: 32px 0 8px; letter-spacing: -0.02em; }
h2 { font-size: 24px; font-weight: 600; margin: 28px 0 8px; letter-spacing: -0.015em; }
h3 { font-size: 20px; font-weight: 600; margin: 24px 0 8px; letter-spacing: -0.01em; }

p { margin: 4px 0; }

ul, ol { padding-left: 24px; margin: 4px 0; }
li { margin: 2px 0; }

blockquote {
  padding-left: 14px;
  border-left: 3px solid var(--text-primary);
  margin: 8px 0;
  color: var(--text-secondary);
}

code {
  font-family: 'SFMono-Regular', Menlo, Consolas, monospace;
  font-size: 85%;
  background: var(--bg-secondary);
  padding: 2px 6px;
  border-radius: 4px;
}

pre {
  background: var(--bg-secondary);
  padding: 16px;
  border-radius: 4px;
  overflow-x: auto;
  margin: 8px 0;
}

pre code {
  background: none;
  padding: 0;
  font-size: 14px;
  line-height: 1.6;
}

hr {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 16px 0;
}

.callout {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border-radius: 4px;
  margin: 8px 0;
  background: var(--bg-secondary);
}

.callout-icon { font-size: 20px; flex-shrink: 0; }
.callout-content { flex: 1; min-width: 0; }

.callout-default { background: #f7f6f3; }
.callout-gray { background: #f1f1ef; }
.callout-brown { background: #f4eeee; }
.callout-orange { background: #fbecdd; }
.callout-yellow { background: #fbf3db; }
.callout-green { background: #edf3ec; }
.callout-blue { background: #e7f3f8; }
.callout-purple { background: #f6f3f9; }
.callout-pink { background: #faf1f5; }
.callout-red { background: #fdebec; }

.todo {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 4px 0;
}

.todo input { margin-top: 4px; }
.todo.checked span { text-decoration: line-through; color: var(--text-secondary); }

.toggle { margin: 4px 0; }
.toggle summary { cursor: pointer; padding: 4px 0; }
.toggle-content { padding-left: 24px; }

.image, .video {
  margin: 16px 0;
}

.image img, .video video {
  max-width: 100%;
  border-radius: 4px;
}

figcaption {
  font-size: 14px;
  color: var(--text-secondary);
  text-align: center;
  margin-top: 8px;
}

.bookmark {
  border: 1px solid var(--border-color);
  border-radius: 4px;
  margin: 8px 0;
  overflow: hidden;
}

.bookmark a {
  display: block;
  padding: 16px;
  text-decoration: none;
  color: inherit;
}

.bookmark-title {
  font-weight: 500;
  margin-bottom: 4px;
}

.bookmark-description {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 4px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.bookmark-url {
  font-size: 12px;
  color: var(--text-secondary);
}

.embed {
  width: 100%;
  height: 400px;
  border-radius: 4px;
  margin: 16px 0;
}

.equation {
  font-family: 'KaTeX_Main', serif;
  text-align: center;
  padding: 16px;
  font-size: 18px;
}

table {
  width: 100%;
  border-collapse: collapse;
  margin: 16px 0;
}

th, td {
  border: 1px solid var(--border-color);
  padding: 8px 12px;
  text-align: left;
}

th {
  background: var(--bg-secondary);
  font-weight: 600;
}

.columns {
  display: flex;
  gap: 24px;
  margin: 8px 0;
}

.column { flex: 1; min-width: 0; }

.child-page, .child-database {
  display: block;
  padding: 8px 0;
  text-decoration: none;
  color: inherit;
}

.child-page:hover { background: var(--bg-secondary); }

.file-link {
  display: inline-block;
  padding: 8px 16px;
  background: var(--bg-secondary);
  border-radius: 4px;
  text-decoration: none;
  color: inherit;
  margin: 4px 0;
}

.template-button {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--bg-primary);
  cursor: not-allowed;
  opacity: 0.7;
}

.mention { background: var(--bg-secondary); padding: 2px 4px; border-radius: 3px; }

.color-gray { color: #9b9a97; }
.color-brown { color: #64473a; }
.color-orange { color: #d9730d; }
.color-yellow { color: #dfab01; }
.color-green { color: #0f7b6c; }
.color-blue { color: #0b6e99; }
.color-purple { color: #6940a5; }
.color-pink { color: #ad1a72; }
.color-red { color: #e03e3e; }

.color-gray_background { background: #ebeced; }
.color-brown_background { background: #e9e5e3; }
.color-orange_background { background: #faebdd; }
.color-yellow_background { background: #fbf3db; }
.color-green_background { background: #ddedea; }
.color-blue_background { background: #ddebf1; }
.color-purple_background { background: #eae4f2; }
.color-pink_background { background: #f4dfeb; }
.color-red_background { background: #fbe4e4; }

@media print {
  body { max-width: none; padding: 0; }
  .page-cover { margin: 0 0 24px; }
  pre, .callout, .toggle { page-break-inside: avoid; }
  h1, h2, h3 { page-break-after: avoid; }
}
`
