package perplexity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// parseSSEStream reads SSE chunks from a response body and returns parsed events.
// Each chunk is delimited by \r\n\r\n. Events start with "event: message\r\ndata: ".
func parseSSEStream(body io.Reader, onChunk func(map[string]any) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024) // 1MB max

	// Custom split function for \r\n\r\n delimiter
	scanner.Split(splitSSEChunks)

	for scanner.Scan() {
		content := scanner.Text()

		if strings.HasPrefix(content, sseEndOfStream) {
			return nil // stream complete
		}

		if !strings.HasPrefix(content, sseEventPrefix) {
			continue
		}

		// Extract JSON data after "event: message\r\ndata: "
		dataStr := content[len(sseDataPrefix):]
		// Trim any trailing \r\n
		dataStr = strings.TrimRight(dataStr, "\r\n")

		var data map[string]any
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			continue // skip malformed chunks
		}

		if err := onChunk(data); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// splitSSEChunks is a bufio.SplitFunc that splits on \r\n\r\n.
func splitSSEChunks(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for \r\n\r\n delimiter
	delim := []byte(sseChunkDelim)
	if i := indexOf(data, delim); i >= 0 {
		return i + len(delim), data[:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}

// indexOf finds the first occurrence of sep in data.
func indexOf(data, sep []byte) int {
	for i := 0; i <= len(data)-len(sep); i++ {
		match := true
		for j := range sep {
			if data[i+j] != sep[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// extractSearchResult parses the final SSE chunk into a SearchResult.
func extractSearchResult(data map[string]any, query string, opts SearchOptions) (*SearchResult, error) {
	result := &SearchResult{
		Query:  query,
		Mode:   opts.Mode,
		Model:  opts.Model,
		Source: "sse",
	}

	// Extract backend_uuid
	if uuid, ok := data["backend_uuid"].(string); ok {
		result.BackendUUID = uuid
	}

	// Extract query_str echo
	if qs, ok := data["query_str"].(string); ok && qs != "" {
		result.Query = qs
	}

	// The "text" field contains the full answer data as JSON.
	// It may be a JSON string (needs parse) or already a parsed object.
	// The answer JSON has: answer, web_results, chunks, structured_answer, related_queries, etc.
	if textRaw, ok := data["text"]; ok && textRaw != nil {
		switch v := textRaw.(type) {
		case string:
			if v != "" {
				// Try to parse as JSON object first (most common format)
				var textMap map[string]any
				if err := json.Unmarshal([]byte(v), &textMap); err == nil {
					extractAnswer(textMap, result)
				} else {
					// Try as step array (legacy format)
					parseTextSteps(v, result)
				}
			}
		case map[string]any:
			extractAnswer(v, result)
		case bool:
			// text can be true/false as a completion flag — skip
		}
	}

	// Fallback: extract from top-level data if text parsing didn't find answer
	if result.Answer == "" {
		extractAnswer(data, result)
	}

	// Build citations from web results
	if len(result.Citations) == 0 && len(result.WebResults) > 0 {
		for _, w := range result.WebResults {
			result.Citations = append(result.Citations, Citation{
				URL:     w.URL,
				Title:   w.Name,
				Snippet: w.Snippet,
				Date:    w.Date,
				Domain:  extractDomain(w.URL),
			})
		}
	}

	// Extract related_queries from top-level data if not found inside text
	if len(result.RelatedQ) == 0 {
		if rq, ok := data["related_queries"]; ok {
			result.RelatedQ = parseStringSlice(rq)
		}
	}

	return result, nil
}

// extractAnswer extracts the clean answer text from the SSE response.
// The answer can be in several places:
//   - data["structured_answer"][0]["text"] — clean markdown (most reliable)
//   - data["answer"] as string — JSON string needing parse: {"answer":"...", ...}
//   - data["answer"] as map — pre-parsed JSON: {answer: "...", ...}
//   - data["text"] — parsed by parseTextSteps earlier
func extractAnswer(data map[string]any, result *SearchResult) {
	// Priority 1: structured_answer — contains clean markdown
	if sa, ok := data["structured_answer"]; ok {
		if arr, ok := sa.([]any); ok && len(arr) > 0 {
			if obj, ok := arr[0].(map[string]any); ok {
				if text, ok := obj["text"].(string); ok && text != "" {
					result.Answer = text
					return
				}
			}
		}
	}

	// Priority 2: answer as map (pre-parsed JSON object)
	if answerMap, ok := data["answer"].(map[string]any); ok {
		if innerAnswer, ok := answerMap["answer"].(string); ok && innerAnswer != "" {
			result.Answer = innerAnswer
		}
		if wr, ok := answerMap["web_results"]; ok && len(result.WebResults) == 0 {
			result.WebResults = parseWebResults(wr)
		}
		if rq, ok := answerMap["related_queries"]; ok && len(result.RelatedQ) == 0 {
			result.RelatedQ = parseStringSlice(rq)
		}
		if sa, ok := answerMap["structured_answer"]; ok {
			if arr, ok := sa.([]any); ok && len(arr) > 0 {
				if obj, ok := arr[0].(map[string]any); ok {
					if text, ok := obj["text"].(string); ok && text != "" {
						result.Answer = text
					}
				}
			}
		}
		return
	}

	// Priority 3: answer as string (JSON-encoded)
	if answerStr, ok := data["answer"].(string); ok && answerStr != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(answerStr), &parsed); err == nil {
			// Recurse with the parsed map directly (it contains answer, structured_answer, etc.)
			extractAnswer(parsed, result)
			// Also extract web_results and related from inside
			if wr, ok := parsed["web_results"]; ok && len(result.WebResults) == 0 {
				result.WebResults = parseWebResults(wr)
			}
			return
		}
		// Not JSON — use as plain text
		result.Answer = answerStr
	}
}

// parseTextSteps handles the step array JSON in the text field.
// Steps: INITIAL_QUERY, SEARCH, READING, FINAL
// The FINAL step has a "content" object with "answer" (JSON string with the actual answer).
func parseTextSteps(textStr string, result *SearchResult) {
	// Parse as array of generic objects (steps may have varying content shapes)
	var steps []map[string]any
	if err := json.Unmarshal([]byte(textStr), &steps); err != nil {
		// Not a JSON array — might be plain text
		result.Answer = textStr
		return
	}

	for _, step := range steps {
		stepType, _ := step["step_type"].(string)
		if stepType != "FINAL" {
			continue
		}

		content, ok := step["content"].(map[string]any)
		if !ok {
			continue
		}

		// The answer is inside content as a JSON string or direct string
		answerRaw, ok := content["answer"]
		if !ok {
			continue
		}

		switch v := answerRaw.(type) {
		case string:
			// Try to parse as JSON {"answer": "...", "web_results": [...], ...}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(v), &parsed); err == nil {
				extractAnswer(parsed, result)
				// Also extract web_results from inside
				if wr, ok := parsed["web_results"]; ok && len(result.WebResults) == 0 {
					result.WebResults = parseWebResults(wr)
				}
				if rq, ok := parsed["related_queries"]; ok && len(result.RelatedQ) == 0 {
					result.RelatedQ = parseStringSlice(rq)
				}
			} else {
				result.Answer = v
			}
		case map[string]any:
			extractAnswer(v, result)
		}
		break
	}
}

// parseWebResults extracts web results from the SSE response.
func parseWebResults(raw any) []WebResult {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var results []WebResult
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		wr := WebResult{
			Name:    getStr(m, "name"),
			URL:     getStr(m, "url"),
			Snippet: getStr(m, "snippet"),
			Date:    getStr(m, "timestamp"),
		}
		if wr.Date == "" {
			wr.Date = getStr(m, "date")
		}
		results = append(results, wr)
	}
	return results
}

// parseMediaItems extracts media items from the SSE response.
func parseMediaItems(raw any) []MediaItem {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var items []MediaItem
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, MediaItem{
			URL:  getStr(m, "url"),
			Type: getStr(m, "type"),
			Alt:  getStr(m, "alt"),
		})
	}
	return items
}

// parseChunks extracts answer chunks from the SSE response.
func parseChunks(raw any) []Chunk {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var chunks []Chunk
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ch := Chunk{Text: getStr(m, "text")}
		if indices, ok := m["source_indices"].([]any); ok {
			for _, idx := range indices {
				if n, ok := idx.(float64); ok {
					ch.SourceIndices = append(ch.SourceIndices, int(n))
				}
			}
		}
		chunks = append(chunks, ch)
	}
	return chunks
}

// parseStringSlice extracts a string slice from a JSON array.
func parseStringSlice(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// getStr extracts a string from a map.
func getStr(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// extractDomain extracts the domain from a URL.
func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// FormatAnswer formats a SearchResult for terminal display.
func FormatAnswer(r *SearchResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Query: %s\n\n", r.Query)

	if r.Answer != "" {
		fmt.Fprintf(&b, "Answer:\n%s\n", r.Answer)
	}

	if len(r.Citations) > 0 {
		b.WriteString("\nCitations:\n")
		for i, c := range r.Citations {
			if c.Title != "" {
				fmt.Fprintf(&b, "  [%d] %s — %s\n", i+1, c.URL, c.Title)
			} else {
				fmt.Fprintf(&b, "  [%d] %s\n", i+1, c.URL)
			}
		}
	}

	if len(r.RelatedQ) > 0 {
		b.WriteString("\nRelated:\n")
		for _, q := range r.RelatedQ {
			fmt.Fprintf(&b, "  - %s\n", q)
		}
	}

	if r.BackendUUID != "" {
		fmt.Fprintf(&b, "\nFollow-up UUID: %s\n", r.BackendUUID)
	}

	return b.String()
}
