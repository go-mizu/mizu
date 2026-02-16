package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestSSEChunkTiming instruments SearchStream to measure when each SSE event arrives.
// Run with: LIVE_TEST=1 go test -run TestSSEChunkTiming -v -count=1
func TestSSEChunkTiming(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}

	cfg := Config{DataDir: t.TempDir(), Timeout: defaultTimeout}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	query := "what is kuzudb"
	t0 := time.Now()

	t.Logf("[%6dms] Starting SSE search: %q", 0, query)

	chunkN := 0
	prevMediaCount := 0
	prevImageCount := 0
	prevBlockCount := 0
	prevKCCount := 0

	result, err := client.SearchStream(ctx, query, DefaultSearchOptions(), func(data map[string]any) {
		chunkN++
		elapsed := time.Since(t0).Milliseconds()

		// Extract text field status
		textStatus := "nil"
		textLen := 0
		if textRaw, ok := data["text"]; ok {
			switch v := textRaw.(type) {
			case bool:
				textStatus = fmt.Sprintf("bool(%v)", v)
			case string:
				textStatus = fmt.Sprintf("string(len=%d)", len(v))
				textLen = len(v)
			case map[string]any:
				textStatus = "map"
			}
		}

		// Check for useful intermediate data
		mediaCount := 0
		if mi, ok := data["media_items"]; ok {
			if arr, ok := mi.([]any); ok {
				mediaCount = len(arr)
			}
		}

		imageCount := 0
		if ic, ok := data["image_completions"]; ok {
			if arr, ok := ic.([]any); ok {
				imageCount = len(arr)
			}
		}

		blockCount := 0
		if b, ok := data["blocks"]; ok {
			if arr, ok := b.([]any); ok {
				blockCount = len(arr)
			}
		}

		kcCount := 0
		if kc, ok := data["knowledge_cards"]; ok {
			if arr, ok := kc.([]any); ok {
				kcCount = len(arr)
			}
		}

		// Only log if something interesting changed
		changed := mediaCount != prevMediaCount || imageCount != prevImageCount ||
			blockCount != prevBlockCount || kcCount != prevKCCount || textLen > 0

		if chunkN <= 3 || changed || chunkN%20 == 0 {
			t.Logf("[%6dms] Chunk #%d: text=%s media=%d images=%d blocks=%d kc=%d",
				elapsed, chunkN, textStatus, mediaCount, imageCount, blockCount, kcCount)

			// Dump media_items if new
			if mediaCount > prevMediaCount {
				if mi, ok := data["media_items"].([]any); ok {
					for i, item := range mi {
						if m, ok := item.(map[string]any); ok {
							t.Logf("  media[%d]: url=%v type=%v", i, m["url"], m["type"])
						}
					}
				}
			}

			// Dump image_completions if new
			if imageCount > prevImageCount {
				if ic, ok := data["image_completions"].([]any); ok {
					for i, item := range ic {
						if i >= 3 {
							t.Logf("  ... and %d more images", imageCount-3)
							break
						}
						if m, ok := item.(map[string]any); ok {
							t.Logf("  image[%d]: url=%v", i, m["image_url"])
						}
					}
				}
			}

			// Dump blocks if new
			if blockCount > prevBlockCount {
				if b, ok := data["blocks"].([]any); ok {
					for i, item := range b {
						if i >= 3 {
							t.Logf("  ... and %d more blocks", blockCount-3)
							break
						}
						if m, ok := item.(map[string]any); ok {
							t.Logf("  block[%d]: type=%v", i, m["block_type"])
						}
					}
				}
			}

			// Dump knowledge_cards if new
			if kcCount > prevKCCount {
				b, _ := json.Marshal(data["knowledge_cards"])
				preview := string(b)
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				t.Logf("  knowledge_cards: %s", preview)
			}

			// Check for search_results or web_results at top level
			if wr, ok := data["web_results"]; ok {
				t.Logf("  HAS web_results! type=%T", wr)
			}
			if sr, ok := data["search_results"]; ok {
				t.Logf("  HAS search_results! type=%T", sr)
			}
			if src, ok := data["sources"]; ok {
				if arr, ok := src.([]any); ok && len(arr) > 0 {
					t.Logf("  sources: %d items", len(arr))
					for i, s := range arr {
						if i >= 3 {
							break
						}
						t.Logf("    source[%d]: %v", i, s)
					}
				}
			}
		}

		prevMediaCount = mediaCount
		prevImageCount = imageCount
		prevBlockCount = blockCount
		prevKCCount = kcCount
	})

	elapsed := time.Since(t0).Milliseconds()

	if err != nil {
		t.Fatalf("[%6dms] SSE search error: %v", elapsed, err)
	}

	t.Logf("[%6dms] DONE — Total chunks: %d", elapsed, chunkN)
	t.Logf("  Answer length: %d chars", len(result.Answer))
	t.Logf("  Citations: %d", len(result.Citations))
	t.Logf("  WebResults: %d", len(result.WebResults))
	t.Logf("  Related: %d", len(result.RelatedQ))
}

// TestAPIStreamTiming instruments the official API ChatStream to measure token delivery.
// Run with: LIVE_TEST=1 PERPLEXITY_API_KEY=pplx-xxx go test -run TestAPIStreamTiming -v -count=1
func TestAPIStreamTiming(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}
	apiKey := os.Getenv("PERPLEXITY_API_KEY")
	if apiKey == "" {
		t.Skip("set PERPLEXITY_API_KEY to run API tests")
	}

	client := NewAPIClient(apiKey, 0)
	query := "what is kuzudb"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t0 := time.Now()
	t.Logf("[%6dms] Starting API stream: %q", 0, query)

	chunkN := 0
	prevLen := 0
	firstTokenMs := int64(0)

	req := &ChatRequest{
		Model: APISonar,
		Messages: []ChatMessage{
			{Role: "user", Content: query},
		},
		ReturnRelated:    true,
		ReturnImages:     true,
		WebSearchOptions: &WebSearchOptions{SearchContextSize: "medium"},
	}

	resp, err := client.ChatStream(ctx, req, func(content string) {
		chunkN++
		elapsed := time.Since(t0).Milliseconds()
		if firstTokenMs == 0 {
			firstTokenMs = elapsed
		}

		delta := len(content) - prevLen
		prevLen = len(content)

		preview := content
		if len(preview) > 80 {
			preview = "..." + preview[len(preview)-80:]
		}

		t.Logf("[%6dms] Chunk #%d: +%d chars (total=%d) %q",
			elapsed, chunkN, delta, len(content), preview)
	})

	elapsed := time.Since(t0).Milliseconds()

	if err != nil {
		t.Fatalf("[%6dms] API stream error: %v", elapsed, err)
	}

	t.Logf("[%6dms] DONE — Total chunks: %d, first token: %dms", elapsed, chunkN, firstTokenMs)
	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		t.Logf("  Answer length: %d chars", len(resp.Choices[0].Message.Content))
	}
	t.Logf("  Citations: %d", len(resp.Citations))
	t.Logf("  SearchResults: %d", len(resp.SearchResults))
}
