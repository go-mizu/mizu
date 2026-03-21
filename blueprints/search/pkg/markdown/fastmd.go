package markdown

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// fastMarkdown converts a trafilatura-extracted *html.Node directly to
// Markdown without render→reparse→html-to-markdown. This eliminates two
// full DOM traversals and the html-to-markdown plugin overhead.
func fastMarkdown(node *html.Node) string {
	var b strings.Builder
	// Pre-allocate based on subtree text size (markdown is ~20-30% of HTML).
	// Falls back to 4 KB if the node has no measurable text.
	textLen, _ := estimateTextLen(node)
	grow := textLen / 3
	if grow < 4096 {
		grow = 4096
	}
	b.Grow(grow)
	walkNode(&b, node, false)
	out := b.String()
	return collapseBlankLines(strings.TrimSpace(out))
}

func walkNode(b *strings.Builder, n *html.Node, pre bool) {
	if n == nil {
		return
	}
	switch n.Type {
	case html.TextNode:
		if pre {
			b.WriteString(n.Data)
		} else {
			writeCollapsedText(b, n.Data)
		}
		return
	case html.ElementNode:
		// handled below
	default:
		// recurse into children for DocumentNode, etc.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkNode(b, c, pre)
		}
		return
	}

	tag := n.DataAtom
	switch tag {
	case atom.Script, atom.Style, atom.Noscript, atom.Iframe:
		return // skip entirely

	case atom.P, atom.Div, atom.Section, atom.Article, atom.Main, atom.Aside:
		b.WriteString("\n\n")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")

	case atom.H1:
		b.WriteString("\n\n# ")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")
	case atom.H2:
		b.WriteString("\n\n## ")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")
	case atom.H3:
		b.WriteString("\n\n### ")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")
	case atom.H4:
		b.WriteString("\n\n#### ")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")
	case atom.H5:
		b.WriteString("\n\n##### ")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")
	case atom.H6:
		b.WriteString("\n\n###### ")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")

	case atom.Blockquote:
		b.WriteString("\n\n> ")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")

	case atom.Pre:
		b.WriteString("\n\n```\n")
		walkChildren(b, n, true)
		b.WriteString("\n```\n\n")

	case atom.Code:
		if !pre {
			b.WriteByte('`')
			walkChildren(b, n, pre)
			b.WriteByte('`')
		} else {
			walkChildren(b, n, true)
		}

	case atom.Strong, atom.B:
		b.WriteString("**")
		walkChildren(b, n, pre)
		b.WriteString("**")

	case atom.Em, atom.I:
		b.WriteByte('*')
		walkChildren(b, n, pre)
		b.WriteByte('*')

	case atom.Del, atom.S:
		b.WriteString("~~")
		walkChildren(b, n, pre)
		b.WriteString("~~")

	case atom.A:
		text := textContent(n)
		href := getAttr(n, "href")
		if href != "" && text != "" {
			b.WriteByte('[')
			b.WriteString(strings.TrimSpace(text))
			b.WriteString("](")
			b.WriteString(href)
			b.WriteByte(')')
		} else {
			walkChildren(b, n, pre)
		}

	case atom.Br:
		b.WriteString("  \n")

	case atom.Hr:
		b.WriteString("\n\n---\n\n")

	case atom.Ul:
		b.WriteString("\n\n")
		walkListItems(b, n, false, pre)
		b.WriteByte('\n')

	case atom.Ol:
		b.WriteString("\n\n")
		walkListItems(b, n, true, pre)
		b.WriteByte('\n')

	case atom.Li:
		// handled by walkListItems
		walkChildren(b, n, pre)

	case atom.Table:
		b.WriteString("\n\n")
		renderTable(b, n)
		b.WriteString("\n\n")

	case atom.Img:
		alt := getAttr(n, "alt")
		src := getAttr(n, "src")
		if src != "" {
			b.WriteString("![")
			b.WriteString(alt)
			b.WriteString("](")
			b.WriteString(src)
			b.WriteByte(')')
		}

	case atom.Span, atom.Label, atom.Small, atom.Sub, atom.Sup, atom.Abbr, atom.Mark, atom.U:
		walkChildren(b, n, pre)

	case atom.Dd, atom.Dt, atom.Figcaption, atom.Figure, atom.Details, atom.Summary:
		b.WriteString("\n\n")
		walkChildren(b, n, pre)
		b.WriteString("\n\n")

	default:
		walkChildren(b, n, pre)
	}
}

func walkChildren(b *strings.Builder, n *html.Node, pre bool) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkNode(b, c, pre)
	}
}

func walkListItems(b *strings.Builder, list *html.Node, ordered bool, pre bool) {
	idx := 1
	for c := list.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.DataAtom != atom.Li {
			continue
		}
		if ordered {
			b.WriteString(strconv.Itoa(idx))
			b.WriteString(". ")
			idx++
		} else {
			b.WriteString("- ")
		}
		walkChildren(b, c, pre)
		b.WriteByte('\n')
	}
}

func renderTable(b *strings.Builder, table *html.Node) {
	var rows [][]string
	forEachElement(table, atom.Tr, func(tr *html.Node) {
		var cells []string
		forEachElement(tr, 0, func(cell *html.Node) {
			if cell.DataAtom == atom.Td || cell.DataAtom == atom.Th {
				cells = append(cells, strings.TrimSpace(textContent(cell)))
			}
		})
		if len(cells) > 0 {
			rows = append(rows, cells)
		}
	})
	if len(rows) == 0 {
		return
	}
	// Header
	b.WriteString("| ")
	b.WriteString(strings.Join(rows[0], " | "))
	b.WriteString(" |\n|")
	for range rows[0] {
		b.WriteString(" --- |")
	}
	b.WriteByte('\n')
	// Body
	for _, row := range rows[1:] {
		b.WriteString("| ")
		b.WriteString(strings.Join(row, " | "))
		b.WriteString(" |\n")
	}
}

func forEachElement(n *html.Node, tag atom.Atom, fn func(*html.Node)) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if tag == 0 || c.DataAtom == tag {
				fn(c)
			} else {
				forEachElement(c, tag, fn)
			}
		}
	}
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(textContent(c))
	}
	return b.String()
}

func writeCollapsedText(b *strings.Builder, s string) {
	// Collapse runs of whitespace to single spaces
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
}

func collapseBlankLines(s string) string {
	// Replace 3+ consecutive newlines with 2
	var b strings.Builder
	b.Grow(len(s))
	nlCount := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			nlCount++
			if nlCount <= 2 {
				b.WriteByte('\n')
			}
		} else {
			nlCount = 0
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// estimateTextLen quickly counts text bytes in a node subtree for Builder pre-allocation.
func estimateTextLen(n *html.Node) (int, int) {
	total := 0
	nodes := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			total += len(n.Data)
		}
		nodes++
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return total, nodes
}
