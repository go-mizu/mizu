package memory

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Chunk represents a fragment of a document with position and content metadata.
type Chunk struct {
	StartLine int    // 1-based start line in the original document.
	EndLine   int    // 1-based end line (inclusive).
	Text      string // The chunk content.
	Hash      string // SHA-256 hex digest of Text.
}

// ChunkConfig controls the chunking behaviour.
type ChunkConfig struct {
	MaxTokens     int // Approximate token budget per chunk (default 400).
	OverlapTokens int // Overlap budget carried from the previous chunk (default 80).
}

// DefaultChunkConfig returns the default chunking parameters matching OpenClaw.
// 400 tokens ~ 1600 characters, 80 token overlap ~ 320 characters.
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		MaxTokens:     400,
		OverlapTokens: 80,
	}
}

// EstimateTokens provides a rough token count from character length.
// The heuristic of ~4 characters per token is widely used for English text.
func EstimateTokens(text string) int {
	n := len(text)
	if n == 0 {
		return 0
	}
	return (n + 3) / 4 // ceiling division
}

// ChunkMarkdown splits content into overlapping chunks respecting line
// boundaries. It accumulates lines until the estimated token count exceeds
// MaxTokens, then starts a new chunk retaining the last OverlapTokens worth
// of lines from the previous chunk.
func ChunkMarkdown(content string, cfg ChunkConfig) []Chunk {
	if content == "" {
		return nil
	}

	maxChars := cfg.MaxTokens * 4
	overlapChars := cfg.OverlapTokens * 4

	lines := strings.Split(content, "\n")

	var chunks []Chunk
	var buf []string   // accumulated lines for the current chunk
	var bufChars int   // character count in buf (including newlines between lines)
	startLine := 1     // 1-based start line of current chunk

	flush := func(endLine int) {
		if len(buf) == 0 {
			return
		}
		text := strings.Join(buf, "\n")
		hash := sha256Hex(text)
		chunks = append(chunks, Chunk{
			StartLine: startLine,
			EndLine:   endLine,
			Text:      text,
			Hash:      hash,
		})
	}

	for i, line := range lines {
		lineNum := i + 1 // 1-based
		lineChars := len(line)
		if len(buf) > 0 {
			lineChars++ // account for the joining newline
		}

		// If adding this line would exceed the budget and we already have
		// content, flush the current chunk.
		if bufChars+lineChars > maxChars && len(buf) > 0 {
			flush(lineNum - 1)

			// Build the overlap: walk backwards through buf to collect the
			// last overlapChars worth of lines.
			overlapLines := overlapSuffix(buf, overlapChars)
			buf = make([]string, len(overlapLines))
			copy(buf, overlapLines)
			bufChars = joinedLen(buf)
			startLine = lineNum - len(overlapLines)
			if startLine < 1 {
				startLine = 1
			}
		}

		buf = append(buf, line)
		if len(buf) == 1 {
			bufChars = len(line)
		} else {
			bufChars += 1 + len(line) // newline + line
		}

		// On the very first line of a fresh chunk (after overlap or start),
		// record startLine.
		if len(chunks) == 0 && i == 0 {
			startLine = 1
		}
	}

	// Flush remaining.
	if len(buf) > 0 {
		flush(len(lines))
	}

	return chunks
}

// overlapSuffix returns the last lines from buf whose total character count
// (joined by newlines) does not exceed maxChars. Lines are returned in
// original order.
func overlapSuffix(buf []string, maxChars int) []string {
	if maxChars <= 0 || len(buf) == 0 {
		return nil
	}

	total := 0
	startIdx := len(buf)

	for i := len(buf) - 1; i >= 0; i-- {
		add := len(buf[i])
		if i < len(buf)-1 {
			add++ // newline separator
		}
		if total+add > maxChars {
			break
		}
		total += add
		startIdx = i
	}

	if startIdx >= len(buf) {
		return nil
	}
	result := make([]string, len(buf)-startIdx)
	copy(result, buf[startIdx:])
	return result
}

// joinedLen returns the character count of lines joined by newlines.
func joinedLen(lines []string) int {
	if len(lines) == 0 {
		return 0
	}
	n := 0
	for _, l := range lines {
		n += len(l)
	}
	n += len(lines) - 1 // newline separators
	return n
}

// sha256Hex returns the lowercase hex SHA-256 digest of s.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}
