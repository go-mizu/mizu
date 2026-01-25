// Package summarize provides URL/text summarization functionality.
package summarize

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// Service handles text and URL summarization.
type Service struct {
	store    store.SummaryStore
	provider llm.Provider
	client   *http.Client
}

// NewService creates a new summarization service.
func NewService(st store.SummaryStore, provider llm.Provider) *Service {
	return &Service{
		store:    st,
		provider: provider,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Summarize summarizes a URL or text.
func (s *Service) Summarize(ctx context.Context, req *types.SummarizeRequest) (*types.SummarizeResponse, error) {
	start := time.Now()

	// Default values
	engine := req.Engine
	if engine == "" {
		engine = types.EngineCecil
	}
	summaryType := req.SummaryType
	if summaryType == "" {
		summaryType = types.SummaryTypeSummary
	}
	useCache := req.Cache == nil || *req.Cache

	// Get content to summarize
	var content string
	var urlHash string

	if req.URL != "" {
		// Check cache first
		urlHash = hashURL(req.URL)
		if useCache {
			cached, err := s.store.GetSummary(ctx, urlHash, string(engine), string(summaryType), req.TargetLanguage)
			if err == nil && cached != nil {
				return &types.SummarizeResponse{
					Meta: types.SummaryMeta{
						ID:   generateID(),
						Node: "local",
						Ms:   time.Since(start).Milliseconds(),
					},
					Data: types.SummaryData{
						Output: cached.Output,
						Tokens: cached.Tokens,
					},
				}, nil
			}
		}

		// Fetch URL content
		fetched, err := s.fetchURL(ctx, req.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch URL: %w", err)
		}
		content = fetched
	} else if req.Text != "" {
		content = req.Text
		urlHash = hashText(content)
	} else {
		return nil, fmt.Errorf("either url or text must be provided")
	}

	// If no LLM provider, return a simple extraction
	if s.provider == nil {
		output := extractSummary(content, summaryType)
		return &types.SummarizeResponse{
			Meta: types.SummaryMeta{
				ID:   generateID(),
				Node: "local",
				Ms:   time.Since(start).Milliseconds(),
			},
			Data: types.SummaryData{
				Output: output,
				Tokens: len(strings.Fields(output)),
			},
		}, nil
	}

	// Generate summary using LLM
	prompt := buildPrompt(content, engine, summaryType, req.TargetLanguage)
	result, err := s.provider.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1000,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	var output string
	if len(result.Choices) > 0 {
		output = result.Choices[0].Message.Content
	}
	tokens := result.Usage.TotalTokens

	// Cache the result
	if useCache && req.URL != "" {
		cache := &types.SummaryCache{
			URLHash:        urlHash,
			URL:            req.URL,
			Engine:         engine,
			SummaryType:    summaryType,
			TargetLanguage: req.TargetLanguage,
			Output:         output,
			Tokens:         tokens,
		}
		s.store.SaveSummary(ctx, cache)
	}

	return &types.SummarizeResponse{
		Meta: types.SummaryMeta{
			ID:   generateID(),
			Node: "local",
			Ms:   time.Since(start).Milliseconds(),
		},
		Data: types.SummaryData{
			Output: output,
			Tokens: tokens,
		},
	}, nil
}

// fetchURL fetches and extracts text content from a URL.
func (s *Service) fetchURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "MizuSearch/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Read body (limit to 1MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", err
	}

	// Simple HTML to text extraction
	content := string(body)
	content = stripHTMLTags(content)
	content = normalizeWhitespace(content)

	// Limit to reasonable length
	if len(content) > 50000 {
		content = content[:50000]
	}

	return content, nil
}

// buildPrompt builds the LLM prompt based on engine and type.
func buildPrompt(content string, engine types.SummaryEngine, summaryType types.SummaryType, targetLang string) string {
	var style string
	switch engine {
	case types.EngineAgnes:
		style = "formal, technical, and analytical"
	case types.EngineMuriel:
		style = "detailed, comprehensive, and thorough"
	default: // Cecil
		style = "friendly, descriptive, and concise"
	}

	var format string
	switch summaryType {
	case types.SummaryTypeTakeaway:
		format = "as a bulleted list of key points"
	default: // Summary
		format = "as a well-structured paragraph"
	}

	langInstruction := ""
	if targetLang != "" {
		langInstruction = fmt.Sprintf(" Write the summary in %s.", targetLang)
	}

	prompt := fmt.Sprintf(`Summarize the following content in a %s style, %s.%s

Content:
%s

Summary:`, style, format, langInstruction, content)

	return prompt
}

// extractSummary extracts a simple summary without LLM.
func extractSummary(content string, summaryType types.SummaryType) string {
	// Simple extraction: first few sentences
	sentences := strings.Split(content, ".")
	var result []string
	wordCount := 0
	maxWords := 150

	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		words := len(strings.Fields(s))
		if wordCount+words > maxWords {
			break
		}
		result = append(result, s)
		wordCount += words
	}

	output := strings.Join(result, ". ")
	if output != "" && !strings.HasSuffix(output, ".") {
		output += "."
	}

	if summaryType == types.SummaryTypeTakeaway {
		// Convert to bullet points
		points := strings.Split(output, ". ")
		var bullets []string
		for _, p := range points {
			p = strings.TrimSpace(p)
			if p != "" {
				bullets = append(bullets, "• "+p)
			}
		}
		return strings.Join(bullets, "\n")
	}

	return output
}

// stripHTMLTags removes HTML tags from content.
func stripHTMLTags(content string) string {
	// Simple HTML tag removal
	result := content
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + " " + result[start+end+1:]
	}
	return result
}

// normalizeWhitespace normalizes whitespace in content.
func normalizeWhitespace(content string) string {
	// Replace multiple whitespace with single space
	result := strings.Join(strings.Fields(content), " ")
	return result
}

// hashURL generates a hash for a URL.
func hashURL(url string) string {
	h := sha256.New()
	h.Write([]byte(url))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// hashText generates a hash for text content.
func hashText(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// generateID generates a unique ID.
func generateID() string {
	h := sha256.New()
	h.Write([]byte(time.Now().String()))
	return hex.EncodeToString(h.Sum(nil))[:8]
}
