package memory

import (
	"strings"
	"testing"
)

func TestChunkMarkdown_EmptyContent(t *testing.T) {
	chunks := ChunkMarkdown("", DefaultChunkConfig())
	if chunks != nil {
		t.Fatalf("expected nil for empty content, got %d chunks", len(chunks))
	}
}

func TestChunkMarkdown_ShortContent(t *testing.T) {
	// A short string well under 400 tokens (~1600 chars) should produce exactly one chunk.
	content := "Hello, world!\nThis is a short document.\nNothing fancy here."
	chunks := ChunkMarkdown(content, DefaultChunkConfig())

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for short content, got %d", len(chunks))
	}

	c := chunks[0]
	if c.StartLine != 1 {
		t.Errorf("StartLine = %d, want 1", c.StartLine)
	}
	if c.EndLine != 3 {
		t.Errorf("EndLine = %d, want 3", c.EndLine)
	}
	if c.Text != content {
		t.Errorf("Text mismatch:\ngot:  %q\nwant: %q", c.Text, content)
	}
	if c.Hash == "" {
		t.Error("Hash should not be empty")
	}
	if len(c.Hash) != 64 {
		t.Errorf("Hash length = %d, want 64 hex chars (SHA-256)", len(c.Hash))
	}
}

func TestChunkMarkdown_MultipleChunks(t *testing.T) {
	// Build content that exceeds 400 tokens (1600 chars).
	// Each line is ~80 chars so 30 lines ~ 2400 chars ~ 600 tokens.
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, strings.Repeat("abcdefgh ", 9)) // ~81 chars per line
	}
	content := strings.Join(lines, "\n")

	chunks := ChunkMarkdown(content, DefaultChunkConfig())
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks for long content, got %d", len(chunks))
	}

	// First chunk should start at line 1.
	if chunks[0].StartLine != 1 {
		t.Errorf("first chunk StartLine = %d, want 1", chunks[0].StartLine)
	}

	// Each chunk must have a non-empty hash.
	for i, c := range chunks {
		if c.Hash == "" {
			t.Errorf("chunk %d has empty Hash", i)
		}
	}

	// Chunks should cover all lines: last chunk EndLine == total line count.
	if chunks[len(chunks)-1].EndLine != len(lines) {
		t.Errorf("last chunk EndLine = %d, want %d", chunks[len(chunks)-1].EndLine, len(lines))
	}
}

func TestChunkMarkdown_LineNumbers(t *testing.T) {
	// 5 lines, each short, should result in a single chunk with correct bounds.
	content := "line1\nline2\nline3\nline4\nline5"
	chunks := ChunkMarkdown(content, DefaultChunkConfig())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].StartLine != 1 {
		t.Errorf("StartLine = %d, want 1", chunks[0].StartLine)
	}
	if chunks[0].EndLine != 5 {
		t.Errorf("EndLine = %d, want 5", chunks[0].EndLine)
	}
}

func TestChunkMarkdown_OverlapSharedText(t *testing.T) {
	// Build content long enough to force multiple chunks, then verify overlap.
	// 400 tokens ~ 1600 chars. Use short identifiable lines.
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, strings.Repeat("x", 60)+"-line-"+string(rune('A'+i%26)))
	}
	content := strings.Join(lines, "\n")

	chunks := ChunkMarkdown(content, DefaultChunkConfig())
	if len(chunks) < 2 {
		t.Fatalf("expected >=2 chunks, got %d", len(chunks))
	}

	// Overlap means the second chunk's start line should be <= the first chunk's end line.
	// (The overlap carries lines from the end of the previous chunk.)
	if chunks[1].StartLine > chunks[0].EndLine {
		t.Errorf("expected overlap: chunk[1].StartLine (%d) should be <= chunk[0].EndLine (%d)",
			chunks[1].StartLine, chunks[0].EndLine)
	}

	// The overlapping region text should appear in both chunks.
	overlapStart := chunks[1].StartLine
	overlapEnd := chunks[0].EndLine
	if overlapStart <= overlapEnd {
		// Extract the overlapping lines from chunk 0 and chunk 1.
		chunk0Lines := strings.Split(chunks[0].Text, "\n")
		chunk1Lines := strings.Split(chunks[1].Text, "\n")

		// The last few lines of chunk 0 should match the first few lines of chunk 1.
		overlapCount := overlapEnd - overlapStart + 1
		if overlapCount > 0 && overlapCount <= len(chunk0Lines) && overlapCount <= len(chunk1Lines) {
			tail := chunk0Lines[len(chunk0Lines)-overlapCount:]
			head := chunk1Lines[:overlapCount]
			for i := 0; i < overlapCount; i++ {
				if tail[i] != head[i] {
					t.Errorf("overlap mismatch at relative line %d: %q vs %q", i, tail[i], head[i])
				}
			}
		}
	}
}

func TestChunkMarkdown_HashIsSHA256(t *testing.T) {
	content := "some unique content\nfor hashing"
	chunks := ChunkMarkdown(content, DefaultChunkConfig())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	hash := chunks[0].Hash
	if len(hash) != 64 {
		t.Errorf("Hash length = %d, want 64 (SHA-256 hex)", len(hash))
	}

	// All chars should be valid lowercase hex.
	for _, r := range hash {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Errorf("Hash contains non-hex character: %c", r)
			break
		}
	}
}

func TestEstimateTokens_Empty(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Errorf("EstimateTokens(\"\") = %d, want 0", got)
	}
}

func TestEstimateTokens_Short(t *testing.T) {
	// "hello" is 5 chars. (5+3)/4 = 2.
	if got := EstimateTokens("hello"); got != 2 {
		t.Errorf("EstimateTokens(\"hello\") = %d, want 2", got)
	}
}

func TestEstimateTokens_400Chars(t *testing.T) {
	s := strings.Repeat("a", 400)
	// (400+3)/4 = 100 (ceiling division with remainder 3).
	got := EstimateTokens(s)
	if got != 100 {
		t.Errorf("EstimateTokens(400 chars) = %d, want 100", got)
	}
}

func TestDefaultChunkConfig(t *testing.T) {
	cfg := DefaultChunkConfig()
	if cfg.MaxTokens != 400 {
		t.Errorf("MaxTokens = %d, want 400", cfg.MaxTokens)
	}
	if cfg.OverlapTokens != 80 {
		t.Errorf("OverlapTokens = %d, want 80", cfg.OverlapTokens)
	}
}
