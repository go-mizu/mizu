// Package ai provides AI-powered search with RAG, query decomposition, and agentic reasoning.
package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
}

// Service orchestrates AI-powered search.
type Service struct {
	providers     map[Mode]llm.Provider
	search        *search.Service
	chunker       *chunker.Service
	sessions      *session.Service
	maxIterations int
	maxSources    int
}

// Query represents an AI search query.
type Query struct {
	Text      string `json:"text"`
	Mode      Mode   `json:"mode"`
	SessionID string `json:"session_id,omitempty"`
}

// Response represents an AI search response.
type Response struct {
	Answer    string            `json:"answer"`
	Citations []session.Citation `json:"citations"`
	FollowUps []string          `json:"follow_ups"`
	Sources   []Source          `json:"sources"`
	Reasoning []ReasoningStep   `json:"reasoning,omitempty"`
	SessionID string            `json:"session_id"`
	Mode      Mode              `json:"mode"`
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

// StreamResponse is the response object sent with the done event.
type StreamResponse struct {
	Text        string             `json:"text"`
	Mode        Mode               `json:"mode"`
	Citations   []session.Citation `json:"citations"`
	FollowUps   []string           `json:"follow_ups"`
	SessionID   string             `json:"session_id"`
	SourcesUsed int                `json:"sources_used"`
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

	return &Service{
		providers:     providers,
		search:        searchSvc,
		chunker:       chunkerSvc,
		sessions:      sessionSvc,
		maxIterations: maxIter,
		maxSources:    maxSrc,
	}
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

	ch := make(chan StreamEvent, 100)

	go func() {
		defer close(ch)

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

		if err != nil {
			ch <- StreamEvent{Type: "error", Error: err.Error()}
			return
		}

		// Send done event with full response
		ch <- StreamEvent{
			Type: "done",
			Response: &StreamResponse{
				Text:        resp.Answer,
				Mode:        resp.Mode,
				Citations:   resp.Citations,
				FollowUps:   resp.FollowUps,
				SessionID:   resp.SessionID,
				SourcesUsed: len(resp.Sources),
			},
		}
	}()

	return ch, nil
}

// processQuick implements single-pass RAG.
func (s *Service) processQuick(ctx context.Context, provider llm.Provider, query Query) (*Response, error) {
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

	// Build conversation context if in session
	var messages []llm.Message
	if query.SessionID != "" && s.sessions != nil {
		history, _ := s.sessions.GetConversationContext(ctx, query.SessionID)
		for _, h := range history {
			messages = append(messages, llm.Message{Role: h["role"], Content: h["content"]})
		}
	}

	// Add system prompt
	systemPrompt := `You are a helpful AI search assistant. Answer the user's question based on the provided search results.
Use inline citations like [1], [2] to reference sources. Be concise and accurate.
If the search results don't contain relevant information, say so.`

	messages = append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("Search results:\n%s\n\nQuestion: %s", strings.Join(contextParts, "\n\n"), query.Text),
	})

	// Generate response
	chatResp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: completion failed: %w", err)
	}

	answer := ""
	if len(chatResp.Choices) > 0 {
		answer = chatResp.Choices[0].Message.Content
	}

	// Generate follow-up questions
	followUps := s.generateFollowUps(ctx, provider, query.Text, answer)

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
		Answer:    answer,
		Citations: citations,
		FollowUps: followUps,
		Sources:   sources,
		SessionID: sessionID,
		Mode:      ModeQuick,
	}, nil
}

// processQuickStream implements streaming single-pass RAG.
func (s *Service) processQuickStream(ctx context.Context, provider llm.Provider, query Query, ch chan<- StreamEvent) (*Response, error) {
	// Send start event
	ch <- StreamEvent{Type: "start"}

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
		// Send citation event
		ch <- StreamEvent{Type: "citation", Citation: &cit}
	}

	// Build messages
	var messages []llm.Message
	if query.SessionID != "" && s.sessions != nil {
		history, _ := s.sessions.GetConversationContext(ctx, query.SessionID)
		for _, h := range history {
			messages = append(messages, llm.Message{Role: h["role"], Content: h["content"]})
		}
	}

	systemPrompt := `You are a helpful AI search assistant. Answer the user's question based on the provided search results.
Use inline citations like [1], [2] to reference sources. Be concise and accurate.`

	messages = append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("Search results:\n%s\n\nQuestion: %s", strings.Join(contextParts, "\n\n"), query.Text),
	})

	// Stream response
	stream, err := provider.ChatCompletionStream(ctx, llm.ChatRequest{
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: stream failed: %w", err)
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

	// Generate follow-ups
	followUps := s.generateFollowUps(ctx, provider, query.Text, answer.String())

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
		Answer:    answer.String(),
		Citations: citations,
		FollowUps: followUps,
		Sources:   sources,
		SessionID: sessionID,
		Mode:      ModeQuick,
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

	systemPrompt := `You are a thorough AI research assistant. Synthesize information from multiple sources to provide a comprehensive answer.
Use inline citations like [1], [2] to reference sources. Be detailed but organized.`

	messages = append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("Research context:\n%s\n\nQuestion: %s\n\nProvide a comprehensive answer:", strings.Join(contextParts, "\n\n"), query.Text),
	})

	chatResp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
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

	reasoning = append(reasoning, ReasoningStep{
		Type:   "synthesize",
		Input:  fmt.Sprintf("%d sources", len(sources)),
		Output: truncate(answer, 100),
	})

	followUps := s.generateFollowUps(ctx, provider, query.Text, answer)

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
		Answer:    answer,
		Citations: citations,
		FollowUps: followUps,
		Sources:   sources,
		Reasoning: reasoning,
		SessionID: sessionID,
		Mode:      ModeDeep,
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

	systemPrompt := `You are a thorough AI research assistant. Synthesize information from multiple sources.
Use inline citations like [1], [2]. Be detailed but organized.`

	messages = append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("Research context:\n%s\n\nQuestion: %s", strings.Join(contextParts, "\n\n"), query.Text),
	})

	stream, err := provider.ChatCompletionStream(ctx, llm.ChatRequest{
		Messages:    messages,
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

	synthStep.Output = truncate(answer.String(), 100)
	reasoning = append(reasoning, synthStep)

	followUps := s.generateFollowUps(ctx, provider, query.Text, answer.String())

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
		Answer:    answer.String(),
		Citations: citations,
		FollowUps: followUps,
		Sources:   sources,
		Reasoning: reasoning,
		SessionID: sessionID,
		Mode:      ModeDeep,
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

			followUps := s.generateFollowUps(ctx, provider, query.Text, answer)

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
				Answer:    answer,
				Citations: citations,
				FollowUps: followUps,
				Sources:   sources,
				Reasoning: reasoning,
				SessionID: sessionID,
				Mode:      ModeResearch,
			}, nil
		}
	}

	// If loop exhausted, synthesize from notes
	synthPrompt := fmt.Sprintf(`Based on your research notes, provide a comprehensive answer.
Question: %s
Notes:
%s

Provide a detailed answer with citations [1], [2], etc.`, query.Text, strings.Join(notes, "\n"))

	synthResp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: synthPrompt}},
		MaxTokens:   2048,
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

	followUps := s.generateFollowUps(ctx, provider, query.Text, answer)

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
		Answer:    answer,
		Citations: citations,
		FollowUps: followUps,
		Sources:   sources,
		Reasoning: reasoning,
		SessionID: sessionID,
		Mode:      ModeResearch,
	}, nil
}

// processResearchStream is the streaming version of processResearch.
func (s *Service) processResearchStream(ctx context.Context, provider llm.Provider, query Query, ch chan<- StreamEvent) (*Response, error) {
	// Send start event
	ch <- StreamEvent{Type: "start"}

	// For simplicity, process research and stream the final answer
	resp, err := s.processResearch(ctx, provider, query)
	if err != nil {
		return nil, err
	}

	// Stream reasoning steps as thinking
	for _, step := range resp.Reasoning {
		ch <- StreamEvent{Type: "thinking", Thinking: fmt.Sprintf("%s: %s -> %s", step.Type, step.Input, step.Output)}
	}

	// Stream citations
	for _, cit := range resp.Citations {
		c := cit
		ch <- StreamEvent{Type: "citation", Citation: &c}
	}

	// Stream answer tokens
	for _, c := range resp.Answer {
		ch <- StreamEvent{Type: "token", Content: string(c)}
	}

	return resp, nil
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

// generateFollowUps generates follow-up questions.
func (s *Service) generateFollowUps(ctx context.Context, provider llm.Provider, query, answer string) []string {
	prompt := fmt.Sprintf(`Based on this Q&A, suggest 3 follow-up questions the user might ask.
Question: %s
Answer: %s

Output each follow-up on a new line, nothing else.`, query, truncate(answer, 500))

	resp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   128,
		Temperature: 0.7,
	})
	if err != nil {
		return nil
	}

	if len(resp.Choices) == 0 {
		return nil
	}

	lines := strings.Split(resp.Choices[0].Message.Content, "\n")
	var followUps []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "0123456789.-) ")
		if len(line) > 10 && len(followUps) < 3 {
			followUps = append(followUps, line)
		}
	}

	return followUps
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
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
