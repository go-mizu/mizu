package export

import (
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
)

// MarkdownConverter converts pages to Markdown format.
type MarkdownConverter struct{}

// NewMarkdownConverter creates a new Markdown converter.
func NewMarkdownConverter() *MarkdownConverter {
	return &MarkdownConverter{}
}

// Convert converts an exported page to Markdown.
func (c *MarkdownConverter) Convert(page *ExportedPage, opts *Request) ([]byte, error) {
	var sb strings.Builder

	// Write page title
	if page.Title != "" {
		if page.Icon != "" {
			sb.WriteString(fmt.Sprintf("# %s %s\n\n", page.Icon, page.Title))
		} else {
			sb.WriteString(fmt.Sprintf("# %s\n\n", page.Title))
		}
	}

	// Convert blocks
	c.convertBlocks(&sb, page.Blocks, 0, opts)

	return []byte(sb.String()), nil
}

// ContentType returns the MIME type.
func (c *MarkdownConverter) ContentType() string {
	return "text/markdown; charset=utf-8"
}

// Extension returns the file extension.
func (c *MarkdownConverter) Extension() string {
	return ".md"
}

// convertBlocks converts a list of blocks to Markdown.
func (c *MarkdownConverter) convertBlocks(sb *strings.Builder, blockList []*blocks.Block, indent int, opts *Request) {
	var listCounter int
	var prevType blocks.BlockType

	for _, block := range blockList {
		// Reset list counter if not consecutive numbered list
		if block.Type != blocks.BlockNumberList && prevType == blocks.BlockNumberList {
			listCounter = 0
		}
		if block.Type == blocks.BlockNumberList {
			listCounter++
		}

		c.convertBlock(sb, block, indent, listCounter, opts)
		prevType = block.Type
	}
}

// convertBlock converts a single block to Markdown.
func (c *MarkdownConverter) convertBlock(sb *strings.Builder, block *blocks.Block, indent, listNum int, opts *Request) {
	indentStr := strings.Repeat("  ", indent)

	switch block.Type {
	case blocks.BlockParagraph:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		if text != "" {
			sb.WriteString(indentStr + text + "\n\n")
		}

	case blocks.BlockHeading1:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		sb.WriteString("# " + text + "\n\n")

	case blocks.BlockHeading2:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		sb.WriteString("## " + text + "\n\n")

	case blocks.BlockHeading3:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		sb.WriteString("### " + text + "\n\n")

	case blocks.BlockBulletList:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		sb.WriteString(indentStr + "- " + text + "\n")
		if len(block.Children) > 0 {
			c.convertBlocks(sb, block.Children, indent+1, opts)
		} else {
			sb.WriteString("\n")
		}

	case blocks.BlockNumberList:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("%s%d. %s\n", indentStr, listNum, text))
		if len(block.Children) > 0 {
			c.convertBlocks(sb, block.Children, indent+1, opts)
		} else {
			sb.WriteString("\n")
		}

	case blocks.BlockTodo:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		checkbox := "[ ]"
		if block.Content.Checked != nil && *block.Content.Checked {
			checkbox = "[x]"
		}
		sb.WriteString(fmt.Sprintf("%s- %s %s\n", indentStr, checkbox, text))
		if len(block.Children) > 0 {
			c.convertBlocks(sb, block.Children, indent+1, opts)
		} else {
			sb.WriteString("\n")
		}

	case blocks.BlockToggle:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("<details>\n<summary>%s</summary>\n\n", text))
		if len(block.Children) > 0 {
			c.convertBlocks(sb, block.Children, 0, opts)
		}
		sb.WriteString("</details>\n\n")

	case blocks.BlockQuote:
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			sb.WriteString("> " + line + "\n")
		}
		sb.WriteString("\n")

	case blocks.BlockCallout:
		// Callouts don't have Markdown equivalent, use HTML
		icon := block.Content.Icon
		if icon == "" {
			icon = "💡"
		}
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		sb.WriteString(fmt.Sprintf("> %s %s\n\n", icon, text))

	case blocks.BlockCode:
		lang := block.Content.Language
		if lang == "" {
			lang = ""
		}
		text := c.richTextToPlainText(block.Content.RichText)
		sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", lang, text))

	case blocks.BlockDivider:
		sb.WriteString("---\n\n")

	case blocks.BlockImage:
		if opts.IncludeImages && block.Content.URL != "" {
			caption := c.richTextToPlainText(block.Content.Caption)
			if caption != "" {
				sb.WriteString(fmt.Sprintf("![%s](%s)\n\n", caption, block.Content.URL))
			} else {
				sb.WriteString(fmt.Sprintf("![](%s)\n\n", block.Content.URL))
			}
		}

	case blocks.BlockVideo:
		if opts.IncludeFiles && block.Content.URL != "" {
			caption := c.richTextToPlainText(block.Content.Caption)
			if caption != "" {
				sb.WriteString(fmt.Sprintf("[Video: %s](%s)\n\n", caption, block.Content.URL))
			} else {
				sb.WriteString(fmt.Sprintf("[Video](%s)\n\n", block.Content.URL))
			}
		}

	case blocks.BlockFile:
		if opts.IncludeFiles && block.Content.URL != "" {
			caption := c.richTextToPlainText(block.Content.Caption)
			if caption != "" {
				sb.WriteString(fmt.Sprintf("[%s](%s)\n\n", caption, block.Content.URL))
			} else {
				sb.WriteString(fmt.Sprintf("[File](%s)\n\n", block.Content.URL))
			}
		}

	case blocks.BlockBookmark:
		title := block.Content.Title
		if title == "" {
			title = block.Content.URL
		}
		if block.Content.URL != "" {
			sb.WriteString(fmt.Sprintf("[%s](%s)\n\n", title, block.Content.URL))
		}

	case blocks.BlockEmbed:
		if block.Content.URL != "" {
			sb.WriteString(fmt.Sprintf("[Embed](%s)\n\n", block.Content.URL))
		}

	case blocks.BlockEquation:
		text := c.richTextToPlainText(block.Content.RichText)
		sb.WriteString(fmt.Sprintf("$$\n%s\n$$\n\n", text))

	case blocks.BlockTable:
		c.convertTable(sb, block, opts)

	case blocks.BlockColumnList:
		// Convert columns as sequential content
		for _, col := range block.Children {
			if col.Type == blocks.BlockColumn && len(col.Children) > 0 {
				c.convertBlocks(sb, col.Children, indent, opts)
			}
		}

	case blocks.BlockChildPage:
		// Link to child page
		text := c.richTextToPlainText(block.Content.RichText)
		if text == "" {
			text = "Subpage"
		}
		sb.WriteString(fmt.Sprintf("[%s](./%s.md)\n\n", text, sanitizeFilename(text)))

	case blocks.BlockChildDB, blocks.BlockLinkedDB:
		// Link to database
		text := c.richTextToPlainText(block.Content.RichText)
		if text == "" {
			text = "Database"
		}
		sb.WriteString(fmt.Sprintf("[%s (Database)](./%s.csv)\n\n", text, sanitizeFilename(text)))

	case blocks.BlockBreadcrumb:
		// Skip breadcrumbs in export

	case blocks.BlockTemplateButton:
		text := block.Content.ButtonText
		if text == "" {
			text = "Template"
		}
		sb.WriteString(fmt.Sprintf("**[%s]**\n\n", text))

	case blocks.BlockSyncedBlock:
		// Export synced block content as regular content
		if len(block.Children) > 0 {
			c.convertBlocks(sb, block.Children, indent, opts)
		}

	default:
		// Unknown block type - try to extract text
		text := c.richTextToMarkdown(block.Content.RichText, opts)
		if text != "" {
			sb.WriteString(text + "\n\n")
		}
	}
}

// convertTable converts a table block to Markdown table.
func (c *MarkdownConverter) convertTable(sb *strings.Builder, block *blocks.Block, opts *Request) {
	if len(block.Children) == 0 {
		return
	}

	width := block.Content.TableWidth
	if width == 0 {
		width = 3 // Default width
	}

	for i, row := range block.Children {
		if row.Type != blocks.BlockTableRow {
			continue
		}

		// Build row cells
		cells := make([]string, width)
		for j := 0; j < width && j < len(row.Content.RichText); j++ {
			cells[j] = c.richTextToMarkdown([]blocks.RichText{row.Content.RichText[j]}, opts)
		}

		sb.WriteString("| " + strings.Join(cells, " | ") + " |\n")

		// Add header separator after first row if table has header
		if i == 0 && block.Content.HasHeader {
			sep := make([]string, width)
			for j := range sep {
				sep[j] = "---"
			}
			sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")
		}
	}
	sb.WriteString("\n")
}

// richTextToMarkdown converts rich text to Markdown with formatting.
func (c *MarkdownConverter) richTextToMarkdown(richText []blocks.RichText, opts *Request) string {
	var parts []string

	for _, rt := range richText {
		text := rt.Text

		// Apply annotations
		if rt.Annotations.Code {
			text = "`" + text + "`"
		}
		if rt.Annotations.Bold {
			text = "**" + text + "**"
		}
		if rt.Annotations.Italic {
			text = "*" + text + "*"
		}
		if rt.Annotations.Strikethrough {
			text = "~~" + text + "~~"
		}
		if rt.Annotations.Underline {
			text = "<u>" + text + "</u>"
		}

		// Handle links
		if rt.Link != "" && opts.IncludeImages {
			text = fmt.Sprintf("[%s](%s)", text, rt.Link)
		}

		// Handle mentions
		if rt.Mention != nil {
			switch rt.Mention.Type {
			case "user":
				text = "@" + text
			case "page":
				text = fmt.Sprintf("[%s](./%s.md)", text, rt.Mention.PageID)
			case "date":
				text = rt.Mention.Date
			}
		}

		parts = append(parts, text)
	}

	return strings.Join(parts, "")
}

// richTextToPlainText converts rich text to plain text (no formatting).
func (c *MarkdownConverter) richTextToPlainText(richText []blocks.RichText) string {
	var parts []string
	for _, rt := range richText {
		parts = append(parts, rt.Text)
	}
	return strings.Join(parts, "")
}

// sanitizeFilename makes a string safe for use as a filename.
func sanitizeFilename(name string) string {
	// Replace unsafe characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	name = replacer.Replace(name)

	// Trim spaces and dots
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")

	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}

	if name == "" {
		name = "untitled"
	}

	return name
}
