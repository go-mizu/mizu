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
  --text-muted: #9b9a97;
  --bg-primary: #ffffff;
  --bg-secondary: #f7f6f3;
  --bg-tertiary: #f1f1ef;
  --border-color: #e3e2df;
  --accent-color: #2eaadc;
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.07);
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;
}

@media (prefers-color-scheme: dark) {
  :root {
    --text-primary: #e6e6e6;
    --text-secondary: #a0a0a0;
    --text-muted: #6b6b6b;
    --bg-primary: #191919;
    --bg-secondary: #252525;
    --bg-tertiary: #2d2d2d;
    --border-color: #3a3a3a;
  }

  .callout-default { background: #252525 !important; }
  .callout-gray { background: #2d2d2d !important; }
  .callout-brown { background: #2c2522 !important; }
  .callout-orange { background: #2d2518 !important; }
  .callout-yellow { background: #2c2a1e !important; }
  .callout-green { background: #1e2c26 !important; }
  .callout-blue { background: #1e262d !important; }
  .callout-purple { background: #26222d !important; }
  .callout-pink { background: #2c222a !important; }
  .callout-red { background: #2c2222 !important; }

  .color-gray { color: #9b9a97 !important; }
  .color-brown { color: #b4a18f !important; }
  .color-orange { color: #e09f6b !important; }
  .color-yellow { color: #c6b344 !important; }
  .color-green { color: #5eb39e !important; }
  .color-blue { color: #6eb0d8 !important; }
  .color-purple { color: #a889d8 !important; }
  .color-pink { color: #d888b4 !important; }
  .color-red { color: #e87878 !important; }

  .color-gray_background { background: #383838 !important; }
  .color-brown_background { background: #3c352f !important; }
  .color-orange_background { background: #3c3426 !important; }
  .color-yellow_background { background: #3c3a28 !important; }
  .color-green_background { background: #2a3c33 !important; }
  .color-blue_background { background: #293744 !important; }
  .color-purple_background { background: #352f44 !important; }
  .color-pink_background { background: #3c2f3a !important; }
  .color-red_background { background: #442f2f !important; }
}

*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html {
  font-size: 16px;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-rendering: optimizeLegibility;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, 'Apple Color Emoji', Arial, sans-serif, 'Segoe UI Emoji', 'Segoe UI Symbol';
  font-size: 16px;
  line-height: 1.6;
  color: var(--text-primary);
  background: var(--bg-primary);
  max-width: 900px;
  margin: 0 auto;
  padding: 48px 64px;
  min-height: 100vh;
}

.page {
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.page-cover {
  margin: -48px -64px 32px;
  height: 240px;
  overflow: hidden;
  position: relative;
}

.page-cover::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 60px;
  background: linear-gradient(transparent, var(--bg-primary));
}

.page-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.page-header {
  margin-bottom: 32px;
}

.page-icon {
  font-size: 78px;
  line-height: 1.1;
  display: block;
  margin-bottom: 12px;
}

.page-title {
  font-size: 42px;
  font-weight: 700;
  line-height: 1.2;
  letter-spacing: -0.03em;
  margin: 0;
  border: none;
}

.page-content {
  font-size: 16px;
  line-height: 1.7;
}

h1 { font-size: 32px; font-weight: 700; margin: 40px 0 12px; letter-spacing: -0.02em; line-height: 1.3; }
h2 { font-size: 26px; font-weight: 600; margin: 36px 0 10px; letter-spacing: -0.015em; line-height: 1.35; }
h3 { font-size: 22px; font-weight: 600; margin: 28px 0 8px; letter-spacing: -0.01em; line-height: 1.4; }

h1:first-child, h2:first-child, h3:first-child { margin-top: 0; }

p { margin: 8px 0; }
p:empty { min-height: 24px; }

a { color: var(--accent-color); text-decoration: none; }
a:hover { text-decoration: underline; }

ul, ol { padding-left: 28px; margin: 8px 0; }
li { margin: 4px 0; padding-left: 4px; }
li::marker { color: var(--text-secondary); }

blockquote {
  padding: 4px 0 4px 18px;
  border-left: 3px solid var(--text-primary);
  margin: 12px 0;
  color: var(--text-secondary);
  font-style: italic;
}

code {
  font-family: 'SFMono-Regular', Menlo, Consolas, 'Liberation Mono', monospace;
  font-size: 0.9em;
  background: var(--bg-secondary);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}

pre {
  background: var(--bg-secondary);
  padding: 20px 24px;
  border-radius: var(--radius-md);
  overflow-x: auto;
  margin: 16px 0;
  border: 1px solid var(--border-color);
}

pre code {
  background: none;
  padding: 0;
  font-size: 14px;
  line-height: 1.6;
  font-weight: normal;
  border-radius: 0;
}

hr {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 24px 0;
}

.callout {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 18px 20px;
  border-radius: var(--radius-md);
  margin: 16px 0;
  background: var(--bg-secondary);
}

.callout-icon { font-size: 22px; flex-shrink: 0; line-height: 1.4; }
.callout-content { flex: 1; min-width: 0; }
.callout-content p:first-child { margin-top: 0; }
.callout-content p:last-child { margin-bottom: 0; }

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
  gap: 10px;
  margin: 6px 0;
  padding: 4px 0;
}

.todo input[type="checkbox"] {
  width: 18px;
  height: 18px;
  margin-top: 3px;
  accent-color: var(--accent-color);
  cursor: default;
}

.todo.checked span {
  text-decoration: line-through;
  color: var(--text-secondary);
}

.toggle { margin: 8px 0; }
.toggle summary {
  cursor: pointer;
  padding: 6px 0;
  font-weight: 500;
  list-style: none;
  display: flex;
  align-items: center;
  gap: 6px;
}
.toggle summary::before {
  content: '▸';
  font-size: 12px;
  transition: transform 0.2s;
}
.toggle[open] summary::before {
  transform: rotate(90deg);
}
.toggle-content {
  padding: 8px 0 8px 24px;
  border-left: 2px solid var(--border-color);
  margin-left: 6px;
}

.image, .video {
  margin: 20px 0;
}

.image img, .video video {
  max-width: 100%;
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
}

figcaption {
  font-size: 14px;
  color: var(--text-secondary);
  text-align: center;
  margin-top: 10px;
  font-style: italic;
}

.bookmark {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  margin: 16px 0;
  overflow: hidden;
  transition: box-shadow 0.2s, border-color 0.2s;
}

.bookmark:hover {
  border-color: var(--text-secondary);
  box-shadow: var(--shadow-sm);
}

.bookmark a {
  display: block;
  padding: 18px 20px;
  text-decoration: none;
  color: inherit;
}

.bookmark-title {
  font-weight: 600;
  margin-bottom: 6px;
  font-size: 15px;
}

.bookmark-description {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.5;
}

.bookmark-url {
  font-size: 12px;
  color: var(--text-muted);
}

.embed {
  width: 100%;
  height: 450px;
  border-radius: var(--radius-md);
  margin: 20px 0;
  border: 1px solid var(--border-color);
}

.equation {
  font-family: 'KaTeX_Main', 'Times New Roman', serif;
  text-align: center;
  padding: 24px;
  font-size: 20px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  margin: 16px 0;
}

table {
  width: 100%;
  border-collapse: collapse;
  margin: 20px 0;
  font-size: 15px;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--border-color);
}

th, td {
  border: 1px solid var(--border-color);
  padding: 12px 16px;
  text-align: left;
}

th {
  background: var(--bg-secondary);
  font-weight: 600;
  font-size: 14px;
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

tr:hover td {
  background: var(--bg-secondary);
}

.columns {
  display: flex;
  gap: 28px;
  margin: 16px 0;
}

.column { flex: 1; min-width: 0; }

.child-page, .child-database {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  text-decoration: none;
  color: inherit;
  border-radius: var(--radius-sm);
  margin: 4px 0;
  transition: background 0.15s;
}

.child-page:hover, .child-database:hover {
  background: var(--bg-secondary);
  text-decoration: none;
}

.file-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
  text-decoration: none;
  color: inherit;
  margin: 6px 0;
  font-size: 14px;
  transition: background 0.15s;
}

.file-link:hover {
  background: var(--bg-tertiary);
  text-decoration: none;
}

.template-button {
  padding: 10px 18px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-primary);
  cursor: not-allowed;
  opacity: 0.6;
  font-size: 14px;
}

.mention {
  background: var(--bg-secondary);
  padding: 3px 6px;
  border-radius: 4px;
  font-size: 0.95em;
}

.color-gray { color: #9b9a97; }
.color-brown { color: #64473a; }
.color-orange { color: #d9730d; }
.color-yellow { color: #dfab01; }
.color-green { color: #0f7b6c; }
.color-blue { color: #0b6e99; }
.color-purple { color: #6940a5; }
.color-pink { color: #ad1a72; }
.color-red { color: #e03e3e; }

.color-gray_background { background: #ebeced; padding: 2px 5px; border-radius: 3px; }
.color-brown_background { background: #e9e5e3; padding: 2px 5px; border-radius: 3px; }
.color-orange_background { background: #faebdd; padding: 2px 5px; border-radius: 3px; }
.color-yellow_background { background: #fbf3db; padding: 2px 5px; border-radius: 3px; }
.color-green_background { background: #ddedea; padding: 2px 5px; border-radius: 3px; }
.color-blue_background { background: #ddebf1; padding: 2px 5px; border-radius: 3px; }
.color-purple_background { background: #eae4f2; padding: 2px 5px; border-radius: 3px; }
.color-pink_background { background: #f4dfeb; padding: 2px 5px; border-radius: 3px; }
.color-red_background { background: #fbe4e4; padding: 2px 5px; border-radius: 3px; }

/* Print styles for clean PDF-like output */
@media print {
  html, body {
    max-width: none;
    padding: 0;
    margin: 0;
    font-size: 14px;
    background: white !important;
    color: black !important;
  }

  body { padding: 40px; }

  .page-cover {
    margin: -40px -40px 24px;
    print-color-adjust: exact;
    -webkit-print-color-adjust: exact;
  }

  pre, .callout, .toggle, .bookmark, table {
    page-break-inside: avoid;
  }

  h1, h2, h3 {
    page-break-after: avoid;
  }

  a { color: inherit !important; text-decoration: underline; }

  .callout, pre, .bookmark, table {
    print-color-adjust: exact;
    -webkit-print-color-adjust: exact;
  }
}

/* Responsive adjustments */
@media (max-width: 768px) {
  body { padding: 24px; }
  .page-cover { margin: -24px -24px 24px; height: 180px; }
  .page-title { font-size: 32px; }
  .page-icon { font-size: 60px; }
  h1 { font-size: 26px; }
  h2 { font-size: 22px; }
  h3 { font-size: 18px; }
  .columns { flex-direction: column; gap: 16px; }
  pre { padding: 14px; }
}
`
