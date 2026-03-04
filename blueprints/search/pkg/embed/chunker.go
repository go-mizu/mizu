package embed

import "strings"

// ChunkText splits text into chunks of at most maxChars characters,
// splitting on paragraph boundaries ("\n\n"). Adjacent chunks overlap
// by up to overlap characters to preserve context across boundaries.
//
// If a single paragraph exceeds maxChars it is hard-split at maxChars.
// Returns nil for empty input.
func ChunkText(text string, maxChars, overlap int) []string {
	if maxChars <= 0 {
		maxChars = 2000
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxChars {
		overlap = maxChars / 4
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	paragraphs := splitParagraphs(text)
	var chunks []string
	var buf strings.Builder

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			chunks = append(chunks, s)
		}
		buf.Reset()

		// Overlap: carry tail of previous chunk into next.
		if overlap > 0 && len(s) > overlap {
			buf.WriteString(s[len(s)-overlap:])
		}
	}

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// If adding this paragraph exceeds maxChars, flush first.
		if buf.Len() > 0 && buf.Len()+1+len(p) > maxChars {
			flush()
		}

		// Hard-split oversized paragraphs.
		for len(p) > maxChars {
			if buf.Len() > 0 {
				flush()
			}
			buf.WriteString(p[:maxChars])
			flush()
			p = p[maxChars:]
		}

		if p == "" {
			continue
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(p)
	}

	flush()
	return chunks
}

func splitParagraphs(text string) []string {
	return strings.Split(text, "\n\n")
}
