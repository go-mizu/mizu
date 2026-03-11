package perplexity

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIClient uses the official Perplexity REST API with API key auth.
type APIClient struct {
	httpClient *http.Client
	apiKey     string
	keyID      int // DB ID for tracking usage
}

// NewAPIClient creates a new API client with the given API key.
func NewAPIClient(apiKey string, keyID int) *APIClient {
	return &APIClient{
		httpClient: &http.Client{Timeout: apiTimeout},
		apiKey:     apiKey,
		keyID:      keyID,
	}
}

// Chat sends a chat completion request and returns the response.
func (c *APIClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiChatCompletions, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
			Duration:   time.Since(start),
		}
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w (body: %s)", err, truncate(string(respBody), 200))
	}

	return &chatResp, nil
}

// ChatStream sends a streaming chat completion request.
// The onChunk callback is called for each streamed chunk.
// Returns the final assembled response.
func (c *APIClient) ChatStream(ctx context.Context, req *ChatRequest, onChunk func(content string)) (*ChatResponse, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiChatCompletions, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)

	var finalResp *ChatResponse
	var fullContent string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := line[6:]
		if data == "[DONE]" {
			break
		}

		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				fullContent = content // Perplexity sends full content, not deltas
				if onChunk != nil {
					onChunk(content)
				}
			}
		}

		// Check for finish
		if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != "" {
			// Try to parse as full ChatResponse (last chunk may have citations)
			var full ChatResponse
			if err := json.Unmarshal([]byte(data), &full); err == nil {
				finalResp = &full
			}
		}
	}

	if finalResp == nil {
		// Build response from collected data
		finalResp = &ChatResponse{
			Choices: []ChatChoice{{
				Message: &ChatMessage{
					Role:    "assistant",
					Content: fullContent,
				},
				FinishReason: "stop",
			}},
		}
	}

	// Ensure content is populated
	if len(finalResp.Choices) > 0 && finalResp.Choices[0].Message == nil {
		finalResp.Choices[0].Message = &ChatMessage{
			Role:    "assistant",
			Content: fullContent,
		}
	}

	return finalResp, nil
}

// Search sends a search-only request (raw results, no LLM generation).
func (c *APIClient) Search(ctx context.Context, req *SearchAPIRequest) (*SearchAPIResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiSearch, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api search request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var searchResp SearchAPIResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	return &searchResp, nil
}

// ToSearchResult converts an API ChatResponse to a SearchResult for unified storage.
func (c *APIClient) ToSearchResult(resp *ChatResponse, query string, durationMs int64) *SearchResult {
	result := &SearchResult{
		Query:      query,
		Source:     "api",
		Model:      resp.Model,
		Mode:       "api",
		SearchedAt: time.Now(),
		APIKeyID:   c.keyID,
		DurationMs: durationMs,
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		result.Answer = resp.Choices[0].Message.Content
	}

	// Convert citations
	for _, url := range resp.Citations {
		result.Citations = append(result.Citations, Citation{
			URL:    url,
			Domain: extractDomain(url),
		})
	}

	// Convert search results
	for _, sr := range resp.SearchResults {
		result.WebResults = append(result.WebResults, WebResult{
			Name:    sr.Title,
			URL:     sr.URL,
			Snippet: sr.Snippet,
			Date:    sr.Date,
		})
		// Also add to citations if not already there
		found := false
		for _, c := range result.Citations {
			if c.URL == sr.URL {
				found = true
				break
			}
		}
		if !found {
			result.Citations = append(result.Citations, Citation{
				URL:     sr.URL,
				Title:   sr.Title,
				Snippet: sr.Snippet,
				Date:    sr.Date,
				Domain:  extractDomain(sr.URL),
			})
		}
	}

	if resp.Usage != nil {
		result.TokensUsed = resp.Usage.TotalTokens
	}

	return result
}

// setHeaders sets API authentication headers.
func (c *APIClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// APIError represents an API error response.
type APIError struct {
	StatusCode int
	Body       string
	Duration   time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error HTTP %d: %s", e.StatusCode, truncate(e.Body, 200))
}

// IsRateLimit returns true if this is a rate limit error.
func (e *APIError) IsRateLimit() bool {
	return e.StatusCode == 429
}

// IsAuth returns true if this is an authentication error.
func (e *APIError) IsAuth() bool {
	return e.StatusCode == 401 || e.StatusCode == 403
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
