package telegram

import "strings"

// markdownToHTML converts markdown text to Telegram-compatible HTML.
//
// Supported conversions:
//   - **bold** to <b>bold</b>
//   - *italic* to <i>italic</i>
//   - `code` to <code>code</code>
//   - ```block``` to <pre>block</pre>
//   - [text](url) to <a href="url">text</a>
//
// HTML entities (&, <, >) are escaped in plain text but preserved inside
// code blocks and already-processed tags.
func markdownToHTML(md string) string {
	// Phase 1: Extract and protect code blocks (``` ... ```).
	// Replace them with placeholders so they are not affected by later passes.
	var codeBlocks []string
	result := extractCodeBlocks(md, &codeBlocks)

	// Phase 2: Extract and protect inline code (` ... `).
	var inlineCodes []string
	result = extractInlineCode(result, &inlineCodes)

	// Phase 3: Escape HTML entities in the remaining plain text.
	result = escapeHTML(result)

	// Phase 4: Convert markdown links [text](url) to <a> tags.
	result = convertLinks(result)

	// Phase 5: Convert **bold** to <b>.
	result = convertBold(result)

	// Phase 6: Convert *italic* to <i> (single asterisks only).
	result = convertItalic(result)

	// Phase 7: Restore inline code placeholders with <code> tags.
	result = restoreInlineCode(result, inlineCodes)

	// Phase 8: Restore code block placeholders with <pre> tags.
	result = restoreCodeBlocks(result, codeBlocks)

	return result
}

// escapeHTML escapes &, <, > for Telegram HTML mode.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

const (
	codeBlockPlaceholder = "\x00CB"
	inlineCodePlaceholder = "\x00IC"
)

// extractCodeBlocks finds ``` delimited code blocks, stores their content,
// and replaces them with indexed placeholders. Handles optional language
// identifiers after the opening ```.
func extractCodeBlocks(s string, blocks *[]string) string {
	var b strings.Builder
	b.Grow(len(s))

	for {
		start := strings.Index(s, "```")
		if start == -1 {
			b.WriteString(s)
			break
		}

		b.WriteString(s[:start])

		rest := s[start+3:]

		// Skip optional language identifier on the same line as opening ```.
		if nl := strings.IndexByte(rest, '\n'); nl != -1 {
			langLine := rest[:nl]
			// If the closing ``` is on the same line as the opening, treat
			// everything between them as the code content.
			if closeInLine := strings.Index(langLine, "```"); closeInLine != -1 {
				content := langLine[:closeInLine]
				idx := len(*blocks)
				*blocks = append(*blocks, content)
				b.WriteString(codeBlockPlaceholder)
				b.WriteString(intToStr(idx))
				b.WriteString(codeBlockPlaceholder)
				s = rest[closeInLine+3:]
				continue
			}
			rest = rest[nl+1:]
		} else {
			// No newline after opening ```, look for closing ``` in remainder.
			end := strings.Index(rest, "```")
			if end == -1 {
				// Unclosed code block: output remainder as-is.
				b.WriteString("```")
				b.WriteString(rest)
				break
			}
			content := rest[:end]
			idx := len(*blocks)
			*blocks = append(*blocks, content)
			b.WriteString(codeBlockPlaceholder)
			b.WriteString(intToStr(idx))
			b.WriteString(codeBlockPlaceholder)
			s = rest[end+3:]
			continue
		}

		end := strings.Index(rest, "```")
		if end == -1 {
			// Unclosed code block: output remainder as-is.
			b.WriteString("```")
			b.WriteString(s[start+3:])
			break
		}

		content := rest[:end]
		// Trim a single trailing newline from the code content for cleaner output.
		content = strings.TrimSuffix(content, "\n")

		idx := len(*blocks)
		*blocks = append(*blocks, content)
		b.WriteString(codeBlockPlaceholder)
		b.WriteString(intToStr(idx))
		b.WriteString(codeBlockPlaceholder)

		s = rest[end+3:]
	}

	return b.String()
}

// extractInlineCode finds ` delimited inline code spans, stores their content,
// and replaces them with indexed placeholders.
func extractInlineCode(s string, codes *[]string) string {
	var b strings.Builder
	b.Grow(len(s))

	for {
		start := strings.IndexByte(s, '`')
		if start == -1 {
			b.WriteString(s)
			break
		}

		b.WriteString(s[:start])

		rest := s[start+1:]
		end := strings.IndexByte(rest, '`')
		if end == -1 {
			// Unclosed backtick: output remainder as-is.
			b.WriteByte('`')
			b.WriteString(rest)
			break
		}

		content := rest[:end]
		idx := len(*codes)
		*codes = append(*codes, content)
		b.WriteString(inlineCodePlaceholder)
		b.WriteString(intToStr(idx))
		b.WriteString(inlineCodePlaceholder)

		s = rest[end+1:]
	}

	return b.String()
}

// convertLinks converts markdown links [text](url) to HTML anchor tags.
func convertLinks(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for {
		// Find opening bracket.
		openBracket := strings.IndexByte(s, '[')
		if openBracket == -1 {
			b.WriteString(s)
			break
		}

		b.WriteString(s[:openBracket])

		rest := s[openBracket+1:]

		// Find closing bracket.
		closeBracket := strings.IndexByte(rest, ']')
		if closeBracket == -1 {
			b.WriteByte('[')
			s = rest
			continue
		}

		text := rest[:closeBracket]
		afterBracket := rest[closeBracket+1:]

		// Expect ( immediately after ].
		if len(afterBracket) == 0 || afterBracket[0] != '(' {
			b.WriteByte('[')
			s = rest
			continue
		}

		// Find closing parenthesis.
		closeParen := strings.IndexByte(afterBracket[1:], ')')
		if closeParen == -1 {
			b.WriteByte('[')
			s = rest
			continue
		}

		url := afterBracket[1 : closeParen+1]

		b.WriteString(`<a href="`)
		b.WriteString(url)
		b.WriteString(`">`)
		b.WriteString(text)
		b.WriteString(`</a>`)

		s = afterBracket[closeParen+2:]
	}

	return b.String()
}

// convertBold converts **text** to <b>text</b>.
func convertBold(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for {
		start := strings.Index(s, "**")
		if start == -1 {
			b.WriteString(s)
			break
		}

		b.WriteString(s[:start])

		rest := s[start+2:]
		end := strings.Index(rest, "**")
		if end == -1 {
			// Unclosed bold marker: output as-is.
			b.WriteString("**")
			s = rest
			continue
		}

		b.WriteString("<b>")
		b.WriteString(rest[:end])
		b.WriteString("</b>")

		s = rest[end+2:]
	}

	return b.String()
}

// convertItalic converts *text* to <i>text</i>.
// Runs after convertBold, so ** sequences have already been consumed.
func convertItalic(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for {
		start := strings.IndexByte(s, '*')
		if start == -1 {
			b.WriteString(s)
			break
		}

		b.WriteString(s[:start])

		rest := s[start+1:]
		end := strings.IndexByte(rest, '*')
		if end == -1 {
			// Unclosed italic marker: output as-is.
			b.WriteByte('*')
			s = rest
			continue
		}

		b.WriteString("<i>")
		b.WriteString(rest[:end])
		b.WriteString("</i>")

		s = rest[end+1:]
	}

	return b.String()
}

// restoreInlineCode replaces inline code placeholders with <code> tags.
// The content is HTML-escaped since it was extracted before the escape pass.
func restoreInlineCode(s string, codes []string) string {
	for i, code := range codes {
		placeholder := inlineCodePlaceholder + intToStr(i) + inlineCodePlaceholder
		s = strings.ReplaceAll(s, placeholder, "<code>"+escapeHTML(code)+"</code>")
	}
	return s
}

// restoreCodeBlocks replaces code block placeholders with <pre> tags.
// The content is HTML-escaped since it was extracted before the escape pass.
func restoreCodeBlocks(s string, blocks []string) string {
	for i, block := range blocks {
		placeholder := codeBlockPlaceholder + intToStr(i) + codeBlockPlaceholder
		s = strings.ReplaceAll(s, placeholder, "<pre>"+escapeHTML(block)+"</pre>")
	}
	return s
}

// intToStr converts a small non-negative integer to its string representation
// without importing strconv.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse.
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
