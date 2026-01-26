// Package ai provides AI-powered search with RAG, query decomposition, and agentic reasoning.
package ai

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/feature/chunker"
	"github.com/go-mizu/mizu/blueprints/search/feature/search"
	"github.com/go-mizu/mizu/blueprints/search/feature/session"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// CacheEntry represents a cached LLM response.
type CacheEntry struct {
	QueryHash        string
	Query            string
	Mode             string
	Model            string
	ResponseText     string
	Citations        string
	FollowUps        string
	RelatedQuestions string
	InputTokens      int
	OutputTokens     int
	ExpiresAt        *time.Time
}

// CacheStore interface for LLM response caching.
type CacheStore interface {
	Get(ctx context.Context, queryHash, mode, model string) (*CacheEntry, error)
	Set(ctx context.Context, entry *CacheEntry) error
	Delete(ctx context.Context, queryHash, mode, model string) error
	GetStats(ctx context.Context) (map[string]any, error)
}

// LogEntry represents an LLM API request/response log.
type LogEntry struct {
	RequestID    string
	Provider     string
	Model        string
	Mode         string
	Query        string
	RequestJSON  string
	ResponseJSON string
	Status       string
	ErrorMessage string
	InputTokens  int
	OutputTokens int
	DurationMs   int
	CostUSD      float64
}

// LogStore interface for LLM API logging.
type LogStore interface {
	Log(ctx context.Context, entry *LogEntry) error
	GetStats(ctx context.Context) (map[string]any, error)
}

// Mode determines the inference strategy.
type Mode string

const (
	ModeQuick    Mode = "quick"    // Single-pass RAG (fast, 270M model)
	ModeDeep     Mode = "deep"     // Query decomposition (balanced, 1B model)
	ModeResearch Mode = "research" // Agentic loop (comprehensive, 4B model)
)

// Config holds AI service configuration.
type Config struct {
	QuickProvider    llm.Provider
	DeepProvider     llm.Provider
	ResearchProvider llm.Provider
	MaxIterations    int // For research mode (default 10)
	MaxSources       int // Maximum sources to fetch (default 10)
	CacheStore       CacheStore
	LogStore         LogStore
	CacheTTL         time.Duration // Default: 24 hours
}

// Service orchestrates AI-powered search.
type Service struct {
	providers     map[Mode]llm.Provider
	search        *search.Service
	chunker       *chunker.Service
	sessions      *session.Service
	cache         CacheStore
	logger        LogStore
	maxIterations int
	maxSources    int
	cacheTTL      time.Duration
}

// getProviderForMode returns the provider for a specific mode, with fallback.
// This respects the user's mode selection instead of always using the most expensive model.
func (s *Service) getProviderForMode(mode Mode) llm.Provider {
	// First, try to get the provider for the requested mode
	if p, ok := s.providers[mode]; ok && p != nil {
		return p
	}
	// Fallback: try any available provider (prefer cheaper for cost savings)
	if p, ok := s.providers[ModeQuick]; ok && p != nil {
		return p
	}
	if p, ok := s.providers[ModeDeep]; ok && p != nil {
		return p
	}
	if p, ok := s.providers[ModeResearch]; ok && p != nil {
		return p
	}
	return nil
}


// Query represents an AI search query.
type Query struct {
	Text      string `json:"text"`
	Mode      Mode   `json:"mode"`
	SessionID string `json:"session_id,omitempty"`
}

// Response represents an AI search response.
type Response struct {
	Answer           string             `json:"answer"`
	Citations        []session.Citation `json:"citations"`
	FollowUps        []string           `json:"follow_ups"`         // Backward compat
	RelatedQuestions []RelatedQuestion  `json:"related_questions"`  // Enhanced follow-ups
	Images           []ImageResult      `json:"images"`             // Related images
	Sources          []Source           `json:"sources"`
	Reasoning        []ReasoningStep    `json:"reasoning,omitempty"`
	SessionID        string             `json:"session_id"`
	Mode             Mode               `json:"mode"`
	Usage            *TokenUsage        `json:"usage,omitempty"`    // Token usage stats
}

// RelatedQuestion represents a categorized follow-up question.
type RelatedQuestion struct {
	Text     string `json:"text"`
	Category string `json:"category,omitempty"` // deeper, related, practical, comparison, current
}

// ImageResult represents an image for the carousel.
type ImageResult struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Title        string `json:"title"`
	SourceURL    string `json:"source_url"`
	SourceDomain string `json:"source_domain"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

// Source represents a fetched source.
type Source struct {
	URL       string          `json:"url"`
	Title     string          `json:"title"`
	Chunks    []chunker.Chunk `json:"chunks,omitempty"`
	FetchedAt time.Time       `json:"fetched_at"`
}

// ReasoningStep represents a step in the reasoning process.
type ReasoningStep struct {
	Type   string `json:"type"` // subquery, search, fetch, analyze, synthesize
	Input  string `json:"input"`
	Output string `json:"output"`
}

// StreamEvent represents a streaming response event.
type StreamEvent struct {
	Type      string             `json:"type"` // start, token, citation, thinking, search, done, error
	Content   string             `json:"content,omitempty"`
	Citation  *session.Citation  `json:"citation,omitempty"`
	Thinking  string             `json:"thinking,omitempty"`
	Query     string             `json:"query,omitempty"`
	Response  *StreamResponse    `json:"response,omitempty"`
	Error     string             `json:"error,omitempty"`
}

// TokenUsage tracks token consumption and cost.
type TokenUsage struct {
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CacheReadTokens  int     `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int     `json:"cache_write_tokens,omitempty"`
	CostUSD          float64 `json:"cost_usd,omitempty"`
	TokensPerSecond  float64 `json:"tokens_per_second,omitempty"`
}

// StreamResponse is the response object sent with the done event.
type StreamResponse struct {
	Text             string             `json:"text"`
	Mode             Mode               `json:"mode"`
	Citations        []session.Citation `json:"citations"`
	FollowUps        []string           `json:"follow_ups"`         // Backward compat
	RelatedQuestions []RelatedQuestion  `json:"related_questions"`  // Enhanced follow-ups
	Images           []ImageResult      `json:"images"`             // Related images
	SessionID        string             `json:"session_id"`
	SourcesUsed      int                `json:"sources_used"`
	Usage            *TokenUsage        `json:"usage,omitempty"`    // Token usage stats
	Provider         string             `json:"provider,omitempty"` // Provider name (e.g., "llamacpp", "claude")
	Model            string             `json:"model,omitempty"`    // Model ID (e.g., "gemma-3-270m", "claude-haiku-4.5")
	FromCache        bool               `json:"from_cache"`         // Whether response came from cache
}

// New creates a new AI service.
func New(cfg Config, searchSvc *search.Service, chunkerSvc *chunker.Service, sessionSvc *session.Service) *Service {
	providers := make(map[Mode]llm.Provider)
	if cfg.QuickProvider != nil {
		providers[ModeQuick] = cfg.QuickProvider
	}
	if cfg.DeepProvider != nil {
		providers[ModeDeep] = cfg.DeepProvider
	}
	if cfg.ResearchProvider != nil {
		providers[ModeResearch] = cfg.ResearchProvider
	}

	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = 10
	}
	maxSrc := cfg.MaxSources
	if maxSrc <= 0 {
		maxSrc = 10
	}
	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 24 * time.Hour
	}

	return &Service{
		providers:     providers,
		search:        searchSvc,
		chunker:       chunkerSvc,
		sessions:      sessionSvc,
		cache:         cfg.CacheStore,
		logger:        cfg.LogStore,
		maxIterations: maxIter,
		maxSources:    maxSrc,
		cacheTTL:      cacheTTL,
	}
}

// hashQuery generates a hash for caching based on query text.
func hashQuery(query string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(query))))
	return hex.EncodeToString(h[:])
}

// logRequest logs an LLM API request/response.
func (s *Service) logRequest(ctx context.Context, entry *LogEntry) {
	if s.logger == nil {
		return
	}
	// Log asynchronously to avoid blocking
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.logger.Log(ctx, entry)
	}()
}

// getCached attempts to retrieve a cached response.
func (s *Service) getCached(ctx context.Context, query string, mode Mode, model string) (*CacheEntry, bool) {
	if s.cache == nil {
		return nil, false
	}
	hash := hashQuery(query)
	entry, err := s.cache.Get(ctx, hash, string(mode), model)
	if err != nil || entry == nil {
		return nil, false
	}
	return entry, true
}

// setCache stores a response in the cache.
func (s *Service) setCache(ctx context.Context, query string, mode Mode, model string, response *Response) {
	if s.cache == nil {
		return
	}

	citations, _ := json.Marshal(response.Citations)
	followUps, _ := json.Marshal(response.FollowUps)
	related, _ := json.Marshal(response.RelatedQuestions)

	expiresAt := time.Now().Add(s.cacheTTL)
	entry := &CacheEntry{
		QueryHash:        hashQuery(query),
		Query:            query,
		Mode:             string(mode),
		Model:            model,
		ResponseText:     response.Answer,
		Citations:        string(citations),
		FollowUps:        string(followUps),
		RelatedQuestions: string(related),
		ExpiresAt:        &expiresAt,
	}
	if response.Usage != nil {
		entry.InputTokens = response.Usage.InputTokens
		entry.OutputTokens = response.Usage.OutputTokens
	}

	// Cache asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.cache.Set(ctx, entry)
	}()
}

// Process handles an AI query and returns a complete response.
func (s *Service) Process(ctx context.Context, query Query) (*Response, error) {
	provider, ok := s.providers[query.Mode]
	if !ok {
		// Fall back to any available provider
		for _, p := range s.providers {
			provider = p
			break
		}
	}
	if provider == nil {
		return nil, fmt.Errorf("ai: no provider available for mode %s", query.Mode)
	}

	switch query.Mode {
	case ModeQuick:
		return s.processQuick(ctx, provider, query)
	case ModeDeep:
		return s.processDeep(ctx, provider, query)
	case ModeResearch:
		return s.processResearch(ctx, provider, query)
	default:
		return s.processQuick(ctx, provider, query)
	}
}

// ProcessStream handles an AI query with streaming response.
func (s *Service) ProcessStream(ctx context.Context, query Query) (<-chan StreamEvent, error) {
	provider, ok := s.providers[query.Mode]
	if !ok {
		for _, p := range s.providers {
			provider = p
			break
		}
	}
	if provider == nil {
		return nil, fmt.Errorf("ai: no provider available for mode %s", query.Mode)
	}

	// Get the model name for caching
	modelName := ""
	if provider != nil {
		models, _ := provider.Models(ctx)
		if len(models) > 0 {
			modelName = models[0].ID
		}
	}

	// Get provider name
	providerName := ""
	if provider != nil {
		providerName = provider.Name()
	}

	// Check cache first for non-streaming response
	if cached, found := s.getCached(ctx, query.Text, query.Mode, modelName); found {
		ch := make(chan StreamEvent, 10)
		go func() {
			defer close(ch)
			// Parse cached data
			var citations []session.Citation
			var followUps []string
			var related []RelatedQuestion
			_ = json.Unmarshal([]byte(cached.Citations), &citations)
			_ = json.Unmarshal([]byte(cached.FollowUps), &followUps)
			_ = json.Unmarshal([]byte(cached.RelatedQuestions), &related)

			// Send cached content as tokens for streaming effect
			ch <- StreamEvent{Type: "thinking", Thinking: "Using cached response..."}
			ch <- StreamEvent{Type: "token", Content: cached.ResponseText}
			ch <- StreamEvent{
				Type: "done",
				Response: &StreamResponse{
					Text:             cached.ResponseText,
					Mode:             Mode(cached.Mode),
					Citations:        citations,
					FollowUps:        followUps,
					RelatedQuestions: related,
					SessionID:        query.SessionID,
					SourcesUsed:      len(citations),
					Usage: &TokenUsage{
						InputTokens:  cached.InputTokens,
						OutputTokens: cached.OutputTokens,
					},
					Provider:  providerName,
					Model:     modelName,
					FromCache: true,
				},
			}
		}()
		return ch, nil
	}

	// For Deep and Research modes, prefer quick provider for interactive feedback
	// The larger models are too slow for interactive use
	if query.Mode == ModeResearch || query.Mode == ModeDeep {
		if quickProvider, ok := s.providers[ModeQuick]; ok {
			provider = quickProvider
			// Update model name and provider name for the fallback provider
			providerName = provider.Name()
			if models, err := provider.Models(ctx); err == nil && len(models) > 0 {
				modelName = models[0].ID
			}
		}
	}

	ch := make(chan StreamEvent, 100)

	go func() {
		defer close(ch)

		startTime := time.Now()
		var resp *Response
		var err error

		switch query.Mode {
		case ModeQuick:
			resp, err = s.processQuickStream(ctx, provider, query, ch)
		case ModeDeep:
			resp, err = s.processDeepStream(ctx, provider, query, ch)
		case ModeResearch:
			resp, err = s.processResearchStream(ctx, provider, query, ch)
		default:
			resp, err = s.processQuickStream(ctx, provider, query, ch)
		}

		durationMs := int(time.Since(startTime).Milliseconds())

		if err != nil {
			// Log error
			s.logRequest(ctx, &LogEntry{
				RequestID:    generateRequestID(),
				Provider:     "llm",
				Model:        modelName,
				Mode:         string(query.Mode),
				Query:        query.Text,
				RequestJSON:  "{}",
				Status:       "error",
				ErrorMessage: err.Error(),
				DurationMs:   durationMs,
			})
			ch <- StreamEvent{Type: "error", Error: err.Error()}
			return
		}

		// Log successful request
		var inputTokens, outputTokens int
		if resp.Usage != nil {
			inputTokens = resp.Usage.InputTokens
			outputTokens = resp.Usage.OutputTokens
		}
		s.logRequest(ctx, &LogEntry{
			RequestID:    generateRequestID(),
			Provider:     "llm",
			Model:        modelName,
			Mode:         string(query.Mode),
			Query:        query.Text,
			RequestJSON:  "{}",
			Status:       "success",
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			DurationMs:   durationMs,
			CostUSD:      estimateCost(modelName, inputTokens, outputTokens),
		})

		// Cache the response
		s.setCache(ctx, query.Text, query.Mode, modelName, resp)

		// Send done event with full response
		ch <- StreamEvent{
			Type: "done",
			Response: &StreamResponse{
				Text:             resp.Answer,
				Mode:             resp.Mode,
				Citations:        resp.Citations,
				FollowUps:        resp.FollowUps,
				RelatedQuestions: resp.RelatedQuestions,
				Images:           resp.Images,
				SessionID:        resp.SessionID,
				SourcesUsed:      len(resp.Sources),
				Usage:            resp.Usage,
				Provider:         providerName,
				Model:            modelName,
				FromCache:        false,
			},
		}
	}()

	return ch, nil
}

// generateRequestID generates a unique request ID for logging.
func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// estimateCost estimates the cost of an LLM request based on model and tokens.
func estimateCost(model string, inputTokens, outputTokens int) float64 {
	// Pricing per million tokens
	var inputPrice, outputPrice float64
	switch {
	case strings.Contains(model, "haiku"):
		inputPrice, outputPrice = 1.0, 5.0
	case strings.Contains(model, "sonnet"):
		inputPrice, outputPrice = 3.0, 15.0
	case strings.Contains(model, "opus"):
		inputPrice, outputPrice = 15.0, 75.0
	default:
		inputPrice, outputPrice = 1.0, 5.0 // Default to haiku pricing
	}
	return (float64(inputTokens) * inputPrice / 1_000_000) + (float64(outputTokens) * outputPrice / 1_000_000)
}

// processQuick implements single-pass RAG.
func (s *Service) processQuick(ctx context.Context, provider llm.Provider, query Query) (*Response, error) {
	// Fetch images in parallel with search
	imagesCh := make(chan []ImageResult, 1)
	go func() {
		imagesCh <- s.fetchImagesForQuery(ctx, query.Text)
	}()

	// Search for relevant results
	searchResp, err := s.search.Search(ctx, query.Text, store.SearchOptions{})
	if err != nil {
		return nil, fmt.Errorf("ai: search failed: %w", err)
	}

	// Build context from search results
	var contextParts []string
	var sources []Source
	var citations []session.Citation

	for i, result := range searchResp.Results {
		if i >= 5 {
			break
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s\n%s\nURL: %s", i+1, result.Title, result.Snippet, result.URL))
		sources = append(sources, Source{
			URL:       result.URL,
			Title:     result.Title,
			FetchedAt: time.Now(),
		})
		citations = append(citations, session.Citation{
			Index:   i + 1,
			URL:     result.URL,
			Title:   result.Title,
			Snippet: result.Snippet,
		})
	}

	// Enhance citations with domain and favicon
	citations = enhanceCitations(citations)

	// Build conversation context if in session
	var messages []llm.Message
	if query.SessionID != "" && s.sessions != nil {
		history, _ := s.sessions.GetConversationContext(ctx, query.SessionID)
		for _, h := range history {
			messages = append(messages, llm.Message{Role: h["role"], Content: h["content"]})
		}
	}

	// Add system prompt - enhanced for comprehensive answers
	systemPrompt := `You are a knowledgeable AI search assistant. Provide comprehensive, well-structured answers based on the search results.

Guidelines:
- Start with a clear introductory paragraph summarizing the topic
- Use markdown headers (##, ###) to organize into logical sections
- Include specific details, facts, and context from the sources
- Use inline citations like [1], [2] to reference sources
- For topics about media (anime, movies, games), include: overview, plot/story, characters, themes, reception
- For topics about people, include: background, achievements, notable works
- For topics about concepts, include: definition, explanation, examples, applications
- Write 3-5 substantial paragraphs minimum
- End with any relevant recent developments or additional context

If the search results don't contain relevant information, say so.`

	messages = append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("Search results:\n%s\n\nQuestion: %s\n\nProvide a comprehensive, detailed answer with sections:", strings.Join(contextParts, "\n\n"), query.Text),
	})

	// Use the mode-specific provider (respects user's mode selection)
	answerProvider := provider

	// Generate response with increased token limit
	chatResp, err := answerProvider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    messages,
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: completion failed: %w", err)
	}

	answer := ""
	if len(chatResp.Choices) > 0 {
		answer = chatResp.Choices[0].Message.Content
	}

	// Generate related questions (enhanced follow-ups)
	relatedQuestions := s.generateRelatedQuestions(ctx, provider, query.Text, answer, citations)
	followUps := make([]string, 0, len(relatedQuestions))
	for _, q := range relatedQuestions {
		followUps = append(followUps, q.Text)
	}

	// Get images from parallel fetch
	images := <-imagesCh

	// Create or update session
	sessionID := query.SessionID
	if sessionID == "" && s.sessions != nil {
		sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
		if sess != nil {
			sessionID = sess.ID
		}
	}
	if sessionID != "" && s.sessions != nil {
		s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeQuick), nil)
		s.sessions.AddMessage(ctx, sessionID, "assistant", answer, string(ModeQuick), citations)
	}

	return &Response{
		Answer:           answer,
		Citations:        citations,
		FollowUps:        followUps,
		RelatedQuestions: relatedQuestions,
		Images:           images,
		Sources:          sources,
		SessionID:        sessionID,
		Mode:             ModeQuick,
	}, nil
}

// processQuickStream implements streaming single-pass RAG.
func (s *Service) processQuickStream(ctx context.Context, provider llm.Provider, query Query, ch chan<- StreamEvent) (*Response, error) {
	// Send start event
	ch <- StreamEvent{Type: "start"}

	// Fetch images in parallel
	imagesCh := make(chan []ImageResult, 1)
	go func() {
		imagesCh <- s.fetchImagesForQuery(ctx, query.Text)
	}()

	// Send search event
	ch <- StreamEvent{Type: "search", Query: query.Text}

	// Search for relevant results
	searchResp, err := s.search.Search(ctx, query.Text, store.SearchOptions{})
	if err != nil {
		return nil, fmt.Errorf("ai: search failed: %w", err)
	}

	// Build context and send citations
	var contextParts []string
	var sources []Source
	var citations []session.Citation

	for i, result := range searchResp.Results {
		if i >= 5 {
			break
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s\n%s\nURL: %s", i+1, result.Title, result.Snippet, result.URL))
		source := Source{
			URL:       result.URL,
			Title:     result.Title,
			FetchedAt: time.Now(),
		}
		sources = append(sources, source)
		cit := session.Citation{
			Index:   i + 1,
			URL:     result.URL,
			Title:   result.Title,
			Snippet: result.Snippet,
		}
		citations = append(citations, cit)
	}

	// Enhance citations with domain and favicon
	citations = enhanceCitations(citations)

	// Send enhanced citation events
	for i := range citations {
		ch <- StreamEvent{Type: "citation", Citation: &citations[i]}
	}

	// Build messages
	var messages []llm.Message
	if query.SessionID != "" && s.sessions != nil {
		history, _ := s.sessions.GetConversationContext(ctx, query.SessionID)
		for _, h := range history {
			messages = append(messages, llm.Message{Role: h["role"], Content: h["content"]})
		}
	}

	systemPrompt := `You are a knowledgeable AI search assistant. Provide comprehensive, well-structured answers based on the search results.

Guidelines:
- Start with a clear introductory paragraph summarizing the topic
- Use markdown headers (##, ###) to organize into logical sections
- Include specific details, facts, and context from the sources
- Use inline citations like [1], [2] to reference sources
- For topics about media (anime, movies, games), include: overview, plot/story, characters, themes, reception
- For topics about people, include: background, achievements, notable works
- For topics about concepts, include: definition, explanation, examples, applications
- Write 3-5 substantial paragraphs minimum
- End with any relevant recent developments or additional context`

	messages = append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("Search results:\n%s\n\nQuestion: %s\n\nProvide a comprehensive, detailed answer with sections:", strings.Join(contextParts, "\n\n"), query.Text),
	})

	// Use the mode-specific provider (respects user's mode selection)
	streamProvider := provider

	// Stream response with increased token limit
	stream, err := streamProvider.ChatCompletionStream(ctx, llm.ChatRequest{
		Messages:    messages,
		MaxTokens:   2048,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: stream failed: %w", err)
	}

	var answer strings.Builder
	var usage TokenUsage
	startTime := time.Now()
	for event := range stream {
		if event.Error != nil {
			return nil, event.Error
		}
		if event.Delta != "" {
			answer.WriteString(event.Delta)
			ch <- StreamEvent{Type: "token", Content: event.Delta}
		}
		// Track token usage from stream events
		if event.Usage != nil {
			usage.InputTokens = event.Usage.PromptTokens
			usage.OutputTokens = event.Usage.CompletionTokens
			usage.TotalTokens = event.Usage.TotalTokens
			usage.CacheReadTokens = event.Usage.CacheReadTokens
			usage.CacheWriteTokens = event.Usage.CacheWriteTokens
		}
		if event.InputTokens > 0 {
			usage.InputTokens = event.InputTokens
		}
		if event.OutputTokens > 0 {
			usage.OutputTokens = event.OutputTokens
		}
	}
	// Calculate tokens per second
	elapsed := time.Since(startTime).Seconds()
	if elapsed > 0 && usage.OutputTokens > 0 {
		usage.TokensPerSecond = float64(usage.OutputTokens) / elapsed
	}
	usage.TotalTokens = usage.InputTokens + usage.OutputTokens

	// Generate related questions (enhanced follow-ups)
	relatedQuestions := s.generateRelatedQuestions(ctx, provider, query.Text, answer.String(), citations)
	followUps := make([]string, 0, len(relatedQuestions))
	for _, q := range relatedQuestions {
		followUps = append(followUps, q.Text)
	}

	// Get images from parallel fetch
	images := <-imagesCh

	// Save to session
	sessionID := query.SessionID
	if sessionID == "" && s.sessions != nil {
		sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
		if sess != nil {
			sessionID = sess.ID
		}
	}
	if sessionID != "" && s.sessions != nil {
		s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeQuick), nil)
		s.sessions.AddMessage(ctx, sessionID, "assistant", answer.String(), string(ModeQuick), citations)
	}

	return &Response{
		Answer:           answer.String(),
		Citations:        citations,
		FollowUps:        followUps,
		RelatedQuestions: relatedQuestions,
		Images:           images,
		Sources:          sources,
		SessionID:        sessionID,
		Mode:             ModeQuick,
		Usage:            &usage,
	}, nil
}

// processDeep implements query decomposition with parallel search.
func (s *Service) processDeep(ctx context.Context, provider llm.Provider, query Query) (*Response, error) {
	reasoning := []ReasoningStep{}

	// Step 1: Decompose query into sub-queries
	subQueries := s.decomposeQuery(ctx, provider, query.Text)
	reasoning = append(reasoning, ReasoningStep{
		Type:   "decompose",
		Input:  query.Text,
		Output: strings.Join(subQueries, "; "),
	})

	// Step 2: Search for each sub-query
	var allResults []types.SearchResult
	var sources []Source
	var citations []session.Citation

	for _, sq := range subQueries {
		searchResp, err := s.search.Search(ctx, sq, store.SearchOptions{})
		if err != nil {
			continue
		}
		reasoning = append(reasoning, ReasoningStep{
			Type:   "search",
			Input:  sq,
			Output: fmt.Sprintf("Found %d results", len(searchResp.Results)),
		})
		allResults = append(allResults, searchResp.Results...)
	}

	// Step 3: Fetch and chunk top pages
	seen := make(map[string]bool)
	var docs []*chunker.Document
	for _, result := range allResults {
		if seen[result.URL] || len(docs) >= s.maxSources {
			continue
		}
		seen[result.URL] = true

		if s.chunker != nil {
			doc, err := s.chunker.Fetch(ctx, result.URL)
			if err == nil {
				docs = append(docs, doc)
				reasoning = append(reasoning, ReasoningStep{
					Type:   "fetch",
					Input:  result.URL,
					Output: fmt.Sprintf("Fetched %d chunks", len(doc.Chunks)),
				})
			}
		}

		sources = append(sources, Source{
			URL:       result.URL,
			Title:     result.Title,
			FetchedAt: time.Now(),
		})
	}

	// Step 4: Get relevant chunks
	var chunks []chunker.Chunk
	if s.chunker != nil && len(docs) > 0 {
		chunks, _ = s.chunker.GetRelevantChunks(ctx, docs, query.Text, 10)
	}

	// Build context
	var contextParts []string
	for i, src := range sources {
		if i >= 10 {
			break
		}
		text := ""
		for _, chunk := range chunks {
			if chunk.URL == src.URL {
				text = chunk.Text
				break
			}
		}
		if text == "" {
			for _, result := range allResults {
				if result.URL == src.URL {
					text = result.Snippet
					break
				}
			}
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s\n%s\nURL: %s", i+1, src.Title, truncate(text, 500), src.URL))
		citations = append(citations, session.Citation{
			Index:   i + 1,
			URL:     src.URL,
			Title:   src.Title,
			Snippet: truncate(text, 200),
		})
	}

	// Step 5: Synthesize answer
	var messages []llm.Message
	if query.SessionID != "" && s.sessions != nil {
		history, _ := s.sessions.GetConversationContext(ctx, query.SessionID)
		for _, h := range history {
			messages = append(messages, llm.Message{Role: h["role"], Content: h["content"]})
		}
	}

	systemPrompt := `You are an expert AI research assistant. Synthesize information from multiple sources to provide a thorough, well-organized answer.

Guidelines:
- Begin with a comprehensive overview paragraph (2-3 sentences)
- Use markdown headers (## and ###) to organize content into clear sections
- For each major point, provide context, details, and citations
- Include relevant background information, history, or context
- For topics about media (anime, manga, movies):
  - Story/Plot Overview: Summarize the narrative arc
  - Main Characters: Describe key characters and their roles
  - Themes: Discuss major themes and what makes it notable
  - Reception/Adaptations: Cover critical acclaim, awards, adaptations
- For people: Background, Career/Achievements, Notable Works, Legacy
- For concepts: Definition, Explanation, Examples, Applications, History
- Write substantial content with 4-6 well-developed sections
- Use inline citations [1], [2], [3] throughout
- Conclude with recent developments, future outlook, or key takeaways

Be thorough and informative. Aim for the depth of an encyclopedia article.`

	messages = append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("Research context:\n%s\n\nQuestion: %s\n\nProvide a comprehensive, detailed answer with multiple sections. Write at least 4-5 paragraphs covering all relevant aspects:", strings.Join(contextParts, "\n\n"), query.Text),
	})

	// Use the mode-specific provider
	answerProvider := provider

	chatResp, err := answerProvider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: completion failed: %w", err)
	}

	answer := ""
	if len(chatResp.Choices) > 0 {
		answer = chatResp.Choices[0].Message.Content
	}

	reasoning = append(reasoning, ReasoningStep{
		Type:   "synthesize",
		Input:  fmt.Sprintf("%d sources", len(sources)),
		Output: truncate(answer, 100),
	})

	// Generate related questions with full context
	relatedQuestions := s.generateRelatedQuestions(ctx, provider, query.Text, answer, citations)
	followUps := make([]string, 0, len(relatedQuestions))
	for _, q := range relatedQuestions {
		followUps = append(followUps, q.Text)
	}

	// Save to session
	sessionID := query.SessionID
	if sessionID == "" && s.sessions != nil {
		sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
		if sess != nil {
			sessionID = sess.ID
		}
	}
	if sessionID != "" && s.sessions != nil {
		s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeDeep), nil)
		s.sessions.AddMessage(ctx, sessionID, "assistant", answer, string(ModeDeep), citations)
	}

	return &Response{
		Answer:           answer,
		Citations:        citations,
		FollowUps:        followUps,
		RelatedQuestions: relatedQuestions,
		Sources:          sources,
		Reasoning:        reasoning,
		SessionID:        sessionID,
		Mode:             ModeDeep,
	}, nil
}

// processDeepStream is the streaming version of processDeep.
func (s *Service) processDeepStream(ctx context.Context, provider llm.Provider, query Query, ch chan<- StreamEvent) (*Response, error) {
	reasoning := []ReasoningStep{}

	// Send start event
	ch <- StreamEvent{Type: "start"}

	// Decompose
	step := ReasoningStep{Type: "decompose", Input: query.Text}
	ch <- StreamEvent{Type: "thinking", Thinking: fmt.Sprintf("Decomposing query: %s", query.Text)}

	subQueries := s.decomposeQuery(ctx, provider, query.Text)
	step.Output = strings.Join(subQueries, "; ")
	reasoning = append(reasoning, step)
	ch <- StreamEvent{Type: "thinking", Thinking: fmt.Sprintf("Sub-queries: %s", step.Output)}

	// Search each sub-query
	var allResults []types.SearchResult
	var sources []Source
	var citations []session.Citation

	for _, sq := range subQueries {
		searchStep := ReasoningStep{Type: "search", Input: sq}
		ch <- StreamEvent{Type: "search", Query: sq}

		searchResp, err := s.search.Search(ctx, sq, store.SearchOptions{})
		if err != nil {
			continue
		}
		searchStep.Output = fmt.Sprintf("Found %d results", len(searchResp.Results))
		reasoning = append(reasoning, searchStep)
		ch <- StreamEvent{Type: "thinking", Thinking: fmt.Sprintf("Search '%s': Found %d results", sq, len(searchResp.Results))}
		allResults = append(allResults, searchResp.Results...)
	}

	// Fetch pages
	seen := make(map[string]bool)
	var docs []*chunker.Document
	for _, result := range allResults {
		if seen[result.URL] || len(docs) >= s.maxSources {
			continue
		}
		seen[result.URL] = true

		source := Source{URL: result.URL, Title: result.Title, FetchedAt: time.Now()}
		sources = append(sources, source)

		if s.chunker != nil {
			doc, err := s.chunker.Fetch(ctx, result.URL)
			if err == nil {
				docs = append(docs, doc)
			}
		}
	}

	// Get relevant chunks and build context
	var chunks []chunker.Chunk
	if s.chunker != nil && len(docs) > 0 {
		chunks, _ = s.chunker.GetRelevantChunks(ctx, docs, query.Text, 10)
	}

	var contextParts []string
	for i, src := range sources {
		if i >= 10 {
			break
		}
		text := ""
		for _, chunk := range chunks {
			if chunk.URL == src.URL {
				text = chunk.Text
				break
			}
		}
		if text == "" {
			for _, result := range allResults {
				if result.URL == src.URL {
					text = result.Snippet
					break
				}
			}
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s\n%s\nURL: %s", i+1, src.Title, truncate(text, 500), src.URL))
		cit := session.Citation{Index: i + 1, URL: src.URL, Title: src.Title, Snippet: truncate(text, 200)}
		citations = append(citations, cit)
		ch <- StreamEvent{Type: "citation", Citation: &cit}
	}

	// Synthesize with streaming
	synthStep := ReasoningStep{Type: "synthesize", Input: fmt.Sprintf("%d sources", len(sources))}
	ch <- StreamEvent{Type: "thinking", Thinking: fmt.Sprintf("Synthesizing from %d sources", len(sources))}

	var messages []llm.Message
	if query.SessionID != "" && s.sessions != nil {
		history, _ := s.sessions.GetConversationContext(ctx, query.SessionID)
		for _, h := range history {
			messages = append(messages, llm.Message{Role: h["role"], Content: h["content"]})
		}
	}

	systemPrompt := `You are an expert AI research assistant. Synthesize information from multiple sources to provide a thorough, well-organized answer.

Guidelines:
- Begin with a comprehensive overview paragraph
- Use markdown headers (## and ###) to organize content into clear sections
- For each major point, provide context, details, and citations
- Include relevant background information, history, or context
- For topics about media: Story/Plot, Characters, Themes, Reception
- For people: Background, Career, Notable Works, Legacy
- For concepts: Definition, Explanation, Examples, Applications
- Write substantial content with 4-6 well-developed sections
- Use inline citations [1], [2], [3] throughout
- Conclude with recent developments or key takeaways`

	messages = append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("Research context:\n%s\n\nQuestion: %s\n\nProvide a comprehensive, detailed answer with multiple sections:", strings.Join(contextParts, "\n\n"), query.Text),
	})

	// Use the deep mode provider for synthesis
	synthProvider := s.getProviderForMode(ModeDeep)
	if synthProvider == nil {
		synthProvider = provider
	}

	stream, err := synthProvider.ChatCompletionStream(ctx, llm.ChatRequest{
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		return nil, err
	}

	var answer strings.Builder
	for event := range stream {
		if event.Error != nil {
			return nil, event.Error
		}
		if event.Delta != "" {
			answer.WriteString(event.Delta)
			ch <- StreamEvent{Type: "token", Content: event.Delta}
		}
	}

	synthStep.Output = truncate(answer.String(), 100)
	reasoning = append(reasoning, synthStep)

	// Generate related questions with full context
	relatedQuestions := s.generateRelatedQuestions(ctx, provider, query.Text, answer.String(), citations)
	followUps := make([]string, 0, len(relatedQuestions))
	for _, q := range relatedQuestions {
		followUps = append(followUps, q.Text)
	}

	sessionID := query.SessionID
	if sessionID == "" && s.sessions != nil {
		sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
		if sess != nil {
			sessionID = sess.ID
		}
	}
	if sessionID != "" && s.sessions != nil {
		s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeDeep), nil)
		s.sessions.AddMessage(ctx, sessionID, "assistant", answer.String(), string(ModeDeep), citations)
	}

	return &Response{
		Answer:           answer.String(),
		Citations:        citations,
		FollowUps:        followUps,
		RelatedQuestions: relatedQuestions,
		Sources:          sources,
		Reasoning:        reasoning,
		SessionID:        sessionID,
		Mode:             ModeDeep,
	}, nil
}

// processResearch implements the agentic research loop.
func (s *Service) processResearch(ctx context.Context, provider llm.Provider, query Query) (*Response, error) {
	reasoning := []ReasoningStep{}
	var sources []Source
	var citations []session.Citation
	var notes []string

	// Agent tools
	tools := []string{"search", "fetch", "note", "answer"}

	// Initial planning
	planPrompt := fmt.Sprintf(`You are a research agent. Plan how to answer this question thoroughly.
Question: %s

Available tools: %s
Output a numbered list of steps.`, query.Text, strings.Join(tools, ", "))

	planResp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: planPrompt},
		},
		MaxTokens:   512,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}

	plan := ""
	if len(planResp.Choices) > 0 {
		plan = planResp.Choices[0].Message.Content
	}
	reasoning = append(reasoning, ReasoningStep{Type: "plan", Input: query.Text, Output: plan})

	// Agent loop
	agentContext := fmt.Sprintf("Question: %s\n\nPlan:\n%s\n\nNotes:", query.Text, plan)
	seen := make(map[string]bool)

	for i := 0; i < s.maxIterations; i++ {
		// Decide next action
		actionPrompt := fmt.Sprintf(`%s
%s

What's your next action? Choose ONE:
- search: <query> - search for information
- fetch: <url> - fetch a specific page
- note: <observation> - record an observation
- answer: <final answer> - provide final answer (with citations)

Respond with just the action and argument.`, agentContext, strings.Join(notes, "\n"))

		actionResp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
			Messages: []llm.Message{
				{Role: "user", Content: actionPrompt},
			},
			MaxTokens:   256,
			Temperature: 0.5,
		})
		if err != nil {
			break
		}

		action := ""
		if len(actionResp.Choices) > 0 {
			action = strings.TrimSpace(actionResp.Choices[0].Message.Content)
		}

		reasoning = append(reasoning, ReasoningStep{Type: "action", Input: truncate(agentContext, 200), Output: action})

		// Parse and execute action
		if strings.HasPrefix(action, "search:") {
			sq := strings.TrimSpace(strings.TrimPrefix(action, "search:"))
			searchResp, err := s.search.Search(ctx, sq, store.SearchOptions{})
			if err == nil {
				limit := 5
				if len(searchResp.Results) < limit {
					limit = len(searchResp.Results)
				}
				for _, r := range searchResp.Results[:limit] {
					if !seen[r.URL] {
						seen[r.URL] = true
						sources = append(sources, Source{URL: r.URL, Title: r.Title, FetchedAt: time.Now()})
						notes = append(notes, fmt.Sprintf("[%d] %s: %s", len(sources), r.Title, r.Snippet))
					}
				}
			}
		} else if strings.HasPrefix(action, "fetch:") {
			url := strings.TrimSpace(strings.TrimPrefix(action, "fetch:"))
			if s.chunker != nil && !seen[url] {
				doc, err := s.chunker.Fetch(ctx, url)
				if err == nil {
					seen[url] = true
					sources = append(sources, Source{URL: doc.URL, Title: doc.Title, FetchedAt: doc.FetchedAt})
					if len(doc.Chunks) > 0 {
						notes = append(notes, fmt.Sprintf("[%d] %s: %s", len(sources), doc.Title, truncate(doc.Chunks[0].Text, 300)))
					}
				}
			}
		} else if strings.HasPrefix(action, "note:") {
			note := strings.TrimSpace(strings.TrimPrefix(action, "note:"))
			notes = append(notes, "Observation: "+note)
		} else if strings.HasPrefix(action, "answer:") {
			// Final answer
			answer := strings.TrimSpace(strings.TrimPrefix(action, "answer:"))

			// Build citations from sources
			for i, src := range sources {
				citations = append(citations, session.Citation{
					Index: i + 1,
					URL:   src.URL,
					Title: src.Title,
				})
			}

			// Generate related questions with context
			relatedQuestions := s.generateRelatedQuestions(ctx, provider, query.Text, answer, citations)
			followUps := make([]string, 0, len(relatedQuestions))
			for _, q := range relatedQuestions {
				followUps = append(followUps, q.Text)
			}

			sessionID := query.SessionID
			if sessionID == "" && s.sessions != nil {
				sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
				if sess != nil {
					sessionID = sess.ID
				}
			}
			if sessionID != "" && s.sessions != nil {
				s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeResearch), nil)
				s.sessions.AddMessage(ctx, sessionID, "assistant", answer, string(ModeResearch), citations)
			}

			return &Response{
				Answer:           answer,
				Citations:        citations,
				FollowUps:        followUps,
				RelatedQuestions: relatedQuestions,
				Sources:          sources,
				Reasoning:        reasoning,
				SessionID:        sessionID,
				Mode:             ModeResearch,
			}, nil
		}
	}

	// If loop exhausted, synthesize from notes with comprehensive prompt
	synthPrompt := fmt.Sprintf(`Based on your research notes, provide a thorough, well-structured answer.

Question: %s

Research Notes:
%s

Instructions:
- Start with a clear introductory overview (2-3 sentences)
- Use markdown headers (## and ###) to organize into logical sections
- For topics about media: Story/Plot, Characters, Themes, Reception
- For topics about people: Background, Career, Notable Works
- For topics about concepts: Definition, Explanation, Examples, Applications
- Include specific details and facts from the notes
- Use inline citations [1], [2], [3] throughout
- Write at least 4-5 substantial paragraphs

Provide a comprehensive answer:`, query.Text, strings.Join(notes, "\n"))

	// Use the research mode provider for synthesis
	researchProvider := s.getProviderForMode(ModeResearch)
	if researchProvider == nil {
		researchProvider = provider
	}

	synthResp, err := researchProvider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: synthPrompt}},
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}

	answer := ""
	if len(synthResp.Choices) > 0 {
		answer = synthResp.Choices[0].Message.Content
	}

	for i, src := range sources {
		citations = append(citations, session.Citation{Index: i + 1, URL: src.URL, Title: src.Title})
	}

	// Generate related questions with context
	relatedQuestions := s.generateRelatedQuestions(ctx, provider, query.Text, answer, citations)
	followUps := make([]string, 0, len(relatedQuestions))
	for _, q := range relatedQuestions {
		followUps = append(followUps, q.Text)
	}

	sessionID := query.SessionID
	if sessionID == "" && s.sessions != nil {
		sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
		if sess != nil {
			sessionID = sess.ID
		}
	}
	if sessionID != "" && s.sessions != nil {
		s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeResearch), nil)
		s.sessions.AddMessage(ctx, sessionID, "assistant", answer, string(ModeResearch), citations)
	}

	return &Response{
		Answer:           answer,
		Citations:        citations,
		FollowUps:        followUps,
		RelatedQuestions: relatedQuestions,
		Sources:          sources,
		Reasoning:        reasoning,
		SessionID:        sessionID,
		Mode:             ModeResearch,
	}, nil
}

// processResearchStream is the streaming version of processResearch with real-time feedback.
// It performs multiple searches automatically and synthesizes results without relying on LLM to choose actions.
func (s *Service) processResearchStream(ctx context.Context, provider llm.Provider, query Query, ch chan<- StreamEvent) (*Response, error) {
	// Send start event
	ch <- StreamEvent{Type: "start"}

	reasoning := []ReasoningStep{}
	var sources []Source
	var citations []session.Citation
	seen := make(map[string]bool)

	// Phase 1: Multi-search with variations
	ch <- StreamEvent{Type: "thinking", Thinking: "Starting comprehensive research..."}

	searchQueries := []string{
		query.Text,
		query.Text + " explained",
		query.Text + " overview",
	}

	for i, sq := range searchQueries {
		ch <- StreamEvent{Type: "search", Query: sq}
		ch <- StreamEvent{Type: "thinking", Thinking: fmt.Sprintf("Search %d/3: %s", i+1, sq)}

		searchResp, err := s.search.Search(ctx, sq, store.SearchOptions{})
		if err != nil {
			continue
		}

		reasoning = append(reasoning, ReasoningStep{
			Type:   "search",
			Input:  sq,
			Output: fmt.Sprintf("Found %d results", len(searchResp.Results)),
		})

		// Collect top results
		limit := 3
		if len(searchResp.Results) < limit {
			limit = len(searchResp.Results)
		}
		for _, r := range searchResp.Results[:limit] {
			if !seen[r.URL] {
				seen[r.URL] = true
				sources = append(sources, Source{URL: r.URL, Title: r.Title, FetchedAt: time.Now()})
				cit := session.Citation{
					Index:   len(citations) + 1,
					URL:     r.URL,
					Title:   r.Title,
					Snippet: r.Snippet,
				}
				citations = append(citations, cit)
				ch <- StreamEvent{Type: "citation", Citation: &cit}
			}
		}
	}

	ch <- StreamEvent{Type: "thinking", Thinking: fmt.Sprintf("Gathered %d sources, synthesizing answer...", len(sources))}

	// Phase 2: Build context from citations
	var contextParts []string
	for _, cit := range citations {
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s: %s", cit.Index, cit.Title, cit.Snippet))
	}

	// Phase 3: Stream synthesized answer with comprehensive prompt
	synthPrompt := fmt.Sprintf(`You are an expert research assistant. Based on the following search results, provide a thorough, well-structured answer about "%s".

Search Results:
%s

Instructions:
- Start with a clear introductory overview (2-3 sentences)
- Use markdown headers (## and ###) to organize into logical sections
- For topics about media (anime, manga, movies, games): cover Story Overview, Main Characters, Themes, Reception/Adaptations
- For topics about people: cover Background, Career/Achievements, Notable Works
- For topics about concepts: cover Definition, Explanation, Examples, Applications
- Include specific details, facts, and context from the sources
- Use inline citations [1], [2], [3] throughout the answer
- Write at least 4-5 substantial paragraphs with multiple sections
- Conclude with recent developments or additional context

Provide a comprehensive answer:`, query.Text, strings.Join(contextParts, "\n\n"))

	// Use the research mode provider for synthesis
	researchProvider := s.getProviderForMode(ModeResearch)
	if researchProvider == nil {
		researchProvider = provider
	}

	stream, err := researchProvider.ChatCompletionStream(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: synthPrompt}},
		MaxTokens:   2048,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		return nil, err
	}

	var answer strings.Builder
	for event := range stream {
		if event.Error != nil {
			return nil, event.Error
		}
		if event.Delta != "" {
			answer.WriteString(event.Delta)
			ch <- StreamEvent{Type: "token", Content: event.Delta}
		}
	}

	reasoning = append(reasoning, ReasoningStep{
		Type:   "synthesize",
		Input:  fmt.Sprintf("%d sources", len(sources)),
		Output: truncate(answer.String(), 100),
	})

	// Generate related questions with context
	relatedQuestions := s.generateRelatedQuestions(ctx, provider, query.Text, answer.String(), citations)
	followUps := make([]string, 0, len(relatedQuestions))
	for _, q := range relatedQuestions {
		followUps = append(followUps, q.Text)
	}

	// Save session
	sessionID := query.SessionID
	if sessionID == "" && s.sessions != nil {
		sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
		if sess != nil {
			sessionID = sess.ID
		}
	}
	if sessionID != "" && s.sessions != nil {
		s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeResearch), nil)
		s.sessions.AddMessage(ctx, sessionID, "assistant", answer.String(), string(ModeResearch), citations)
	}

	return &Response{
		Answer:           answer.String(),
		Citations:        citations,
		FollowUps:        followUps,
		RelatedQuestions: relatedQuestions,
		Sources:          sources,
		Reasoning:        reasoning,
		SessionID:        sessionID,
		Mode:             ModeResearch,
	}, nil
}

// decomposeQuery breaks a complex query into sub-queries.
func (s *Service) decomposeQuery(ctx context.Context, provider llm.Provider, query string) []string {
	prompt := fmt.Sprintf(`Break this question into 3-5 specific sub-questions for thorough research.
Question: %s

Output each sub-question on a new line, nothing else.`, query)

	resp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   256,
		Temperature: 0.5,
	})
	if err != nil {
		return []string{query}
	}

	if len(resp.Choices) == 0 {
		return []string{query}
	}

	lines := strings.Split(resp.Choices[0].Message.Content, "\n")
	var subQueries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove numbering like "1.", "2.", "-", etc.
		line = strings.TrimLeft(line, "0123456789.-) ")
		if len(line) > 10 {
			subQueries = append(subQueries, line)
		}
	}

	if len(subQueries) == 0 {
		return []string{query}
	}

	return subQueries
}

// generateFollowUps generates follow-up questions (backward compat).
func (s *Service) generateFollowUps(ctx context.Context, provider llm.Provider, query, answer string) []string {
	related := s.generateRelatedQuestions(ctx, provider, query, answer, nil)
	followUps := make([]string, 0, len(related))
	for _, q := range related {
		followUps = append(followUps, q.Text)
	}
	return followUps
}

// generateRelatedQuestions generates categorized related questions based on the topic and answer.
func (s *Service) generateRelatedQuestions(ctx context.Context, provider llm.Provider, query, answer string, citations []session.Citation) []RelatedQuestion {
	// Build source context from citations for topic awareness
	sourceContext := ""
	for i, c := range citations {
		if i < 5 {
			sourceContext += fmt.Sprintf("- %s\n", c.Title)
		}
	}

	// Extract key info from answer for better context (first 500 chars)
	answerContext := answer
	if len(answerContext) > 500 {
		answerContext = answerContext[:500] + "..."
	}

	// Use the quick provider for generating follow-up questions (cheaper, fast)
	questionProvider := s.getProviderForMode(ModeQuick)
	if questionProvider == nil {
		questionProvider = provider
	}

	prompt := fmt.Sprintf(`You are generating follow-up search queries for a user who just searched about: "%s"

Here's a summary of what they learned:
%s

Sources consulted:
%s

Generate 5 highly relevant follow-up questions that a curious user would naturally want to explore next. Each question should:
- Be directly related to the topic (not generic)
- Be specific and searchable
- Build on what the user just learned

Categories:
1. DEEPER: A question exploring more detail about a specific aspect mentioned
2. RELATED: A question about a connected topic, person, or concept mentioned
3. BACKGROUND: A question about history, origins, or context
4. COMPARISON: A question comparing to similar things or alternatives
5. CURRENT: A question about recent news, updates, or future developments

Format each question on its own line with its category prefix (e.g., "DEEPER: What is...?")

Questions:`, query, answerContext, sourceContext)

	resp, err := questionProvider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   512,
		Temperature: 0.6, // Slightly higher for variety but still consistent
	})
	if err != nil {
		return nil
	}

	if len(resp.Choices) == 0 {
		return nil
	}

	lines := strings.Split(resp.Choices[0].Message.Content, "\n")
	var questions []RelatedQuestion

	// Category order matches prompt order (updated to match new prompt)
	categoryOrder := []string{"deeper", "related", "background", "comparison", "current"}
	categoryIdx := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip preamble/intro lines
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "here") || strings.HasPrefix(lower, "okay") ||
			strings.HasPrefix(lower, "sure") || strings.HasPrefix(lower, "questions") {
			continue
		}

		// Strip numbering, bullets, markdown
		line = strings.TrimLeft(line, "0123456789.-) *•")
		line = strings.ReplaceAll(line, "**", "")
		line = strings.TrimSpace(line)

		// Check for category prefix (DEEPER:, RELATED:, etc.)
		for _, cat := range categoryOrder {
			prefix := strings.ToUpper(cat) + ":"
			if strings.HasPrefix(strings.ToUpper(line), prefix) {
				line = strings.TrimSpace(line[len(prefix):])
				if len(line) > 10 {
					questions = append(questions, RelatedQuestion{Text: line, Category: cat})
				}
				break
			}
		}

		// If no prefix found but looks like a question, add with sequential category
		if len(line) > 15 && strings.HasSuffix(line, "?") && len(questions) < 5 {
			cat := ""
			if categoryIdx < len(categoryOrder) {
				cat = categoryOrder[categoryIdx]
				categoryIdx++
			}
			questions = append(questions, RelatedQuestion{Text: line, Category: cat})
		}
	}

	return questions
}

// fetchImagesForQuery fetches related images for a query.
func (s *Service) fetchImagesForQuery(ctx context.Context, query string) []ImageResult {
	if s.search == nil {
		return nil
	}

	images, err := s.search.SearchImages(ctx, query, store.SearchOptions{
		PerPage: 6,
	})
	if err != nil {
		return nil
	}

	results := make([]ImageResult, 0, len(images))
	for _, img := range images {
		results = append(results, ImageResult{
			URL:          img.URL,
			ThumbnailURL: img.ThumbnailURL,
			Title:        img.Title,
			SourceURL:    img.SourceURL,
			SourceDomain: img.SourceDomain,
			Width:        img.Width,
			Height:       img.Height,
		})
	}
	return results
}

// enhanceCitations adds domain and favicon to citations.
func enhanceCitations(citations []session.Citation) []session.Citation {
	enhanced := make([]session.Citation, len(citations))
	domainCount := make(map[string]int)

	// Count domains for grouping
	for _, c := range citations {
		domain := extractDomain(c.URL)
		domainCount[domain]++
	}

	for i, c := range citations {
		domain := extractDomain(c.URL)
		enhanced[i] = session.Citation{
			Index:        c.Index,
			URL:          c.URL,
			Title:        c.Title,
			Snippet:      c.Snippet,
			Domain:       domain,
			Favicon:      fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=32", domain),
			OtherSources: domainCount[domain] - 1,
		}
	}
	return enhanced
}

// extractDomain extracts domain from URL.
func extractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	// Simple extraction without full URL parsing for performance
	url := strings.TrimPrefix(rawURL, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	if idx := strings.Index(url, "/"); idx > 0 {
		url = url[:idx]
	}
	return url
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}


// GetModes returns available AI modes.
func GetModes() []ModeInfo {
	return []ModeInfo{
		{ID: string(ModeQuick), Name: "Quick", Description: "Fast single-pass answer", Model: "gemma-3-270m", Available: true},
		{ID: string(ModeDeep), Name: "Deep", Description: "Multi-source research", Model: "gemma-3-1b", Available: true},
		{ID: string(ModeResearch), Name: "Research", Description: "Comprehensive investigation", Model: "gemma-3-4b", Available: true},
		{ID: string(ModeDeepSearch), Name: "Deep Search", Description: "Google-style comprehensive report", Model: "gemma-3-4b", Available: true},
	}
}

// ModeInfo describes an AI mode.
type ModeInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model"`
	Available   bool   `json:"available"`
}
