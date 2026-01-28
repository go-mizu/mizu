package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/feature/chunker"
	"github.com/go-mizu/mizu/blueprints/search/feature/session"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/search/store"
)

// ModeDeepSearch is the comprehensive research mode.
const ModeDeepSearch Mode = "deepsearch"

// DeepSearchConfig holds configuration for deep search.
type DeepSearchConfig struct {
	MaxSources     int           // Maximum sources to fetch (default 50)
	WorkerPool     int           // Number of parallel fetch workers (default 10)
	FetchTimeout   time.Duration // Timeout for fetching each source
	AnalysisDepth  int           // Number of chunks to analyze per source (default 5)
	ReportSections int           // Number of sections in the report (default 4-6)
}

// DeepSearchProgress represents the progress of a deep search operation.
type DeepSearchProgress struct {
	Phase       string `json:"phase"`        // searching, fetching, analyzing, writing
	Current     int    `json:"current"`      // Current item number
	Total       int    `json:"total"`        // Total items
	Message     string `json:"message"`      // Human-readable status
	SectionName string `json:"section,omitempty"` // Current section being written
}

// DeepSearchStreamEvent extends StreamEvent for deep search.
type DeepSearchStreamEvent struct {
	Type      string              `json:"type"` // progress, source, citation, section, token, done, error
	Progress  *DeepSearchProgress `json:"progress,omitempty"`
	Source    *Source             `json:"source,omitempty"`
	Citation  *session.Citation   `json:"citation,omitempty"`
	Section   *ReportSection      `json:"section,omitempty"`
	Token     string              `json:"token,omitempty"`
	FollowUps []string            `json:"follow_ups,omitempty"`
	Error     string              `json:"error,omitempty"`
	SessionID string              `json:"session_id,omitempty"`
}

// ReportSection represents a section in the deep search report.
type ReportSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Order   int    `json:"order"`
}

// DeepSearchResponse represents the full deep search response.
type DeepSearchResponse struct {
	Query      string              `json:"query"`
	Overview   string              `json:"overview"`
	KeyFindings []string           `json:"key_findings"`
	Sections   []ReportSection     `json:"sections"`
	Methodology string              `json:"methodology"`
	Citations  []session.Citation  `json:"citations"`
	Sources    []Source            `json:"sources"`
	FollowUps  []string            `json:"follow_ups"`
	SessionID  string              `json:"session_id"`
	Mode       Mode                `json:"mode"`
	Duration   time.Duration       `json:"duration"`
}

// ProcessDeepSearch performs a comprehensive deep search.
func (s *Service) ProcessDeepSearch(ctx context.Context, query Query) (*DeepSearchResponse, error) {
	startTime := time.Now()

	// Use quick provider for speed (research provider is too slow for interactive use)
	provider := s.providers[ModeQuick]
	if provider == nil {
		provider = s.providers[ModeResearch]
	}
	if provider == nil {
		// Fall back to any available provider
		for _, p := range s.providers {
			provider = p
			break
		}
	}
	if provider == nil {
		return nil, fmt.Errorf("ai: no provider available for deep search")
	}

	cfg := DeepSearchConfig{
		MaxSources:     10, // Reduced for faster response
		WorkerPool:     5,
		FetchTimeout:   15 * time.Second,
		AnalysisDepth:  3,
		ReportSections: 3,
	}

	// Phase 1: Generate search queries
	subQueries := s.generateDeepSearchQueries(ctx, provider, query.Text)
	if len(subQueries) == 0 {
		subQueries = []string{query.Text}
	}

	// Phase 2: Search all queries
	var allURLs []string
	seenURLs := make(map[string]bool)

	for _, sq := range subQueries {
		searchResp, err := s.search.Search(ctx, sq, store.SearchOptions{})
		if err != nil {
			continue
		}
		for _, result := range searchResp.Results {
			if !seenURLs[result.URL] && len(allURLs) < cfg.MaxSources {
				seenURLs[result.URL] = true
				allURLs = append(allURLs, result.URL)
			}
		}
	}

	// Phase 3: Fetch and chunk sources in parallel
	sources, chunks := s.fetchSourcesParallel(ctx, allURLs, cfg)

	// Phase 4: Analyze and rank content
	relevantChunks := s.rankChunks(ctx, provider, query.Text, chunks, cfg.AnalysisDepth*len(sources))

	// Phase 5: Generate report
	report := s.generateDeepReport(ctx, provider, query.Text, relevantChunks, sources, cfg)

	// Build citations
	var citations []session.Citation
	for i, src := range sources {
		citations = append(citations, session.Citation{
			Index:   i + 1,
			URL:     src.URL,
			Title:   src.Title,
			Snippet: s.getSnippetForSource(src, relevantChunks),
		})
	}

	// Generate follow-ups
	followUps := s.generateFollowUps(ctx, provider, query.Text, report.Overview)

	// Save to session
	sessionID := query.SessionID
	if sessionID == "" && s.sessions != nil {
		sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
		if sess != nil {
			sessionID = sess.ID
		}
	}
	if sessionID != "" && s.sessions != nil {
		s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeDeepSearch), nil)
		fullReport := s.formatReportAsMarkdown(report)
		s.sessions.AddMessage(ctx, sessionID, "assistant", fullReport, string(ModeDeepSearch), citations)
	}

	return &DeepSearchResponse{
		Query:       query.Text,
		Overview:    report.Overview,
		KeyFindings: report.KeyFindings,
		Sections:    report.Sections,
		Methodology: report.Methodology,
		Citations:   citations,
		Sources:     sources,
		FollowUps:   followUps,
		SessionID:   sessionID,
		Mode:        ModeDeepSearch,
		Duration:    time.Since(startTime),
	}, nil
}

// ProcessDeepSearchStream performs deep search with streaming progress updates.
func (s *Service) ProcessDeepSearchStream(ctx context.Context, query Query) (<-chan DeepSearchStreamEvent, error) {
	ch := make(chan DeepSearchStreamEvent, 100)

	go func() {
		defer close(ch)

		startTime := time.Now()

		// Use quick provider for interactive feedback (research provider is too slow)
		provider := s.providers[ModeQuick]
		if provider == nil {
			provider = s.providers[ModeResearch]
		}
		if provider == nil {
			for _, p := range s.providers {
				provider = p
				break
			}
		}
		if provider == nil {
			ch <- DeepSearchStreamEvent{Type: "error", Error: "no provider available"}
			return
		}

		cfg := DeepSearchConfig{
			MaxSources:     10, // Reduced for faster response
			WorkerPool:     5,
			FetchTimeout:   15 * time.Second,
			AnalysisDepth:  3,
			ReportSections: 3,
		}

		// Phase 1: Generate search queries
		ch <- DeepSearchStreamEvent{
			Type: "progress",
			Progress: &DeepSearchProgress{
				Phase:   "searching",
				Message: "Generating search queries...",
			},
		}

		subQueries := s.generateDeepSearchQueries(ctx, provider, query.Text)
		if len(subQueries) == 0 {
			subQueries = []string{query.Text}
		}

		// Phase 2: Search
		ch <- DeepSearchStreamEvent{
			Type: "progress",
			Progress: &DeepSearchProgress{
				Phase:   "searching",
				Current: 0,
				Total:   len(subQueries),
				Message: fmt.Sprintf("Searching %d queries...", len(subQueries)),
			},
		}

		var allURLs []string
		seenURLs := make(map[string]bool)
		urlTitles := make(map[string]string)
		urlSnippets := make(map[string]string)

		for i, sq := range subQueries {
			searchResp, err := s.search.Search(ctx, sq, store.SearchOptions{})
			if err != nil {
				continue
			}
			for _, result := range searchResp.Results {
				if !seenURLs[result.URL] && len(allURLs) < cfg.MaxSources {
					seenURLs[result.URL] = true
					urlTitles[result.URL] = result.Title
					urlSnippets[result.URL] = result.Snippet
					allURLs = append(allURLs, result.URL)
				}
			}
			ch <- DeepSearchStreamEvent{
				Type: "progress",
				Progress: &DeepSearchProgress{
					Phase:   "searching",
					Current: i + 1,
					Total:   len(subQueries),
					Message: fmt.Sprintf("Found %d sources...", len(allURLs)),
				},
			}
		}

		// Phase 3: Build sources from search results (skip slow fetching)
		ch <- DeepSearchStreamEvent{
			Type: "progress",
			Progress: &DeepSearchProgress{
				Phase:   "processing",
				Message: "Processing sources...",
			},
		}

		// Build sources and chunks from snippets instead of fetching
		var sources []Source
		var relevantChunks []chunker.Chunk
		seen := make(map[string]bool)
		for i, url := range allURLs {
			if seen[url] || i >= cfg.MaxSources {
				continue
			}
			seen[url] = true
			title := urlTitles[url]
			snippet := urlSnippets[url]
			sources = append(sources, Source{
				URL:       url,
				Title:     title,
				FetchedAt: time.Now(),
			})
			if snippet != "" {
				relevantChunks = append(relevantChunks, chunker.Chunk{
					URL:  url,
					Text: snippet,
				})
			}
			ch <- DeepSearchStreamEvent{
				Type:   "source",
				Source: &sources[len(sources)-1],
			}
		}

		// Phase 4: Generate report with streaming
		ch <- DeepSearchStreamEvent{
			Type: "progress",
			Progress: &DeepSearchProgress{
				Phase:   "writing",
				Message: "Writing comprehensive report...",
			},
		}

		report := s.generateDeepReportStream(ctx, provider, query.Text, relevantChunks, sources, cfg, ch)

		// Build citations
		var citations []session.Citation
		for i, src := range sources {
			cit := session.Citation{
				Index:   i + 1,
				URL:     src.URL,
				Title:   src.Title,
				Snippet: s.getSnippetForSource(src, relevantChunks),
			}
			citations = append(citations, cit)
			ch <- DeepSearchStreamEvent{Type: "citation", Citation: &cit}
		}

		// Generate follow-ups
		followUps := s.generateFollowUps(ctx, provider, query.Text, report.Overview)

		// Save to session
		sessionID := query.SessionID
		if sessionID == "" && s.sessions != nil {
			sess, _ := s.sessions.Create(ctx, truncate(query.Text, 50))
			if sess != nil {
				sessionID = sess.ID
			}
		}
		if sessionID != "" && s.sessions != nil {
			s.sessions.AddMessage(ctx, sessionID, "user", query.Text, string(ModeDeepSearch), nil)
			fullReport := s.formatReportAsMarkdown(report)
			s.sessions.AddMessage(ctx, sessionID, "assistant", fullReport, string(ModeDeepSearch), citations)
		}

		ch <- DeepSearchStreamEvent{
			Type:      "done",
			FollowUps: followUps,
			SessionID: sessionID,
		}

		_ = time.Since(startTime)
	}()

	return ch, nil
}

// generateDeepSearchQueries generates comprehensive search queries.
func (s *Service) generateDeepSearchQueries(ctx context.Context, provider llm.Provider, query string) []string {
	prompt := fmt.Sprintf(`Generate 5-8 comprehensive search queries to thoroughly research this topic.
Include queries for: background, latest developments, expert opinions, comparisons, and practical applications.

Topic: %s

Output each query on a new line, nothing else.`, query)

	resp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   512,
		Temperature: 0.7,
	})
	if err != nil {
		return nil
	}

	if len(resp.Choices) == 0 {
		return nil
	}

	lines := strings.Split(resp.Choices[0].Message.Content, "\n")
	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "0123456789.-) ")
		if len(line) > 10 {
			queries = append(queries, line)
		}
	}

	return queries
}

// fetchSourcesParallel fetches sources in parallel using a worker pool.
func (s *Service) fetchSourcesParallel(ctx context.Context, urls []string, cfg DeepSearchConfig) ([]Source, []chunker.Chunk) {
	if s.chunker == nil {
		return nil, nil
	}

	type result struct {
		source Source
		chunks []chunker.Chunk
	}

	resultCh := make(chan result, len(urls))
	urlCh := make(chan string, len(urls))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < cfg.WorkerPool; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range urlCh {
				fetchCtx, cancel := context.WithTimeout(ctx, cfg.FetchTimeout)
				doc, err := s.chunker.Fetch(fetchCtx, url)
				cancel()

				if err != nil {
					continue
				}

				resultCh <- result{
					source: Source{
						URL:       doc.URL,
						Title:     doc.Title,
						Chunks:    doc.Chunks,
						FetchedAt: doc.FetchedAt,
					},
					chunks: doc.Chunks,
				}
			}
		}()
	}

	// Send URLs to workers
	for _, url := range urls {
		urlCh <- url
	}
	close(urlCh)

	// Wait for all workers
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	var sources []Source
	var allChunks []chunker.Chunk
	for r := range resultCh {
		sources = append(sources, r.source)
		allChunks = append(allChunks, r.chunks...)
	}

	return sources, allChunks
}

// fetchSourcesParallelWithProgress fetches sources with progress updates.
func (s *Service) fetchSourcesParallelWithProgress(ctx context.Context, urls []string, titles map[string]string, cfg DeepSearchConfig, ch chan<- DeepSearchStreamEvent) ([]Source, []chunker.Chunk) {
	if s.chunker == nil {
		return nil, nil
	}

	type result struct {
		source Source
		chunks []chunker.Chunk
	}

	resultCh := make(chan result, len(urls))
	urlCh := make(chan string, len(urls))
	var completed int32
	var mu sync.Mutex

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < cfg.WorkerPool; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range urlCh {
				fetchCtx, cancel := context.WithTimeout(ctx, cfg.FetchTimeout)
				doc, err := s.chunker.Fetch(fetchCtx, url)
				cancel()

				mu.Lock()
				completed++
				current := int(completed)
				mu.Unlock()

				if err != nil {
					continue
				}

				src := Source{
					URL:       doc.URL,
					Title:     doc.Title,
					Chunks:    doc.Chunks,
					FetchedAt: doc.FetchedAt,
				}

				ch <- DeepSearchStreamEvent{
					Type:   "source",
					Source: &src,
				}

				ch <- DeepSearchStreamEvent{
					Type: "progress",
					Progress: &DeepSearchProgress{
						Phase:   "fetching",
						Current: current,
						Total:   len(urls),
						Message: fmt.Sprintf("Fetched %d/%d sources", current, len(urls)),
					},
				}

				resultCh <- result{source: src, chunks: doc.Chunks}
			}
		}()
	}

	// Send URLs
	for _, url := range urls {
		urlCh <- url
	}
	close(urlCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var sources []Source
	var allChunks []chunker.Chunk
	for r := range resultCh {
		sources = append(sources, r.source)
		allChunks = append(allChunks, r.chunks...)
	}

	return sources, allChunks
}

// rankChunks ranks chunks by relevance to the query.
func (s *Service) rankChunks(ctx context.Context, provider llm.Provider, query string, chunks []chunker.Chunk, limit int) []chunker.Chunk {
	if len(chunks) <= limit {
		return chunks
	}

	// Use embeddings if available
	if s.chunker != nil {
		relevant, err := s.chunker.GetRelevantChunks(ctx, nil, query, limit)
		if err == nil && len(relevant) > 0 {
			return relevant
		}
	}

	// Simple relevance: prefer chunks containing query terms
	queryTerms := strings.Fields(strings.ToLower(query))
	type scoredChunk struct {
		chunk chunker.Chunk
		score int
	}

	var scored []scoredChunk
	for _, chunk := range chunks {
		text := strings.ToLower(chunk.Text)
		score := 0
		for _, term := range queryTerms {
			if strings.Contains(text, term) {
				score++
			}
		}
		scored = append(scored, scoredChunk{chunk, score})
	}

	// Sort by score (simple bubble sort for small lists)
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	result := make([]chunker.Chunk, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].chunk)
	}

	return result
}

type deepReport struct {
	Overview    string
	KeyFindings []string
	Sections    []ReportSection
	Methodology string
}

// generateDeepReport generates the full research report.
func (s *Service) generateDeepReport(ctx context.Context, provider llm.Provider, query string, chunks []chunker.Chunk, sources []Source, cfg DeepSearchConfig) *deepReport {
	// Build context
	var contextParts []string
	for i, chunk := range chunks {
		if i >= 30 {
			break
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s\n%s", i+1, chunk.URL, truncate(chunk.Text, 800)))
	}

	prompt := fmt.Sprintf(`Based on comprehensive research from %d sources, write a detailed report answering:

%s

Research context:
%s

Write a report with this exact structure (use markdown):

## Overview
[2-3 sentence executive summary]

## Key Findings
- Finding 1 with citation [1]
- Finding 2 with citation [2]
- Finding 3 with citation [3]

## Detailed Analysis

### [Subtopic 1]
[Detailed content with inline citations]

### [Subtopic 2]
[Detailed content with inline citations]

### [Subtopic 3]
[Detailed content with inline citations]

## Methodology
[Brief description of research approach]

Use inline citations [1], [2] etc. Be comprehensive and accurate.`, len(sources), query, strings.Join(contextParts, "\n\n"))

	resp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return &deepReport{Overview: "Failed to generate report: " + err.Error()}
	}

	if len(resp.Choices) == 0 {
		return &deepReport{Overview: "No response generated"}
	}

	return s.parseDeepReport(resp.Choices[0].Message.Content)
}

// generateDeepReportStream generates report with streaming tokens.
func (s *Service) generateDeepReportStream(ctx context.Context, provider llm.Provider, query string, chunks []chunker.Chunk, sources []Source, cfg DeepSearchConfig, ch chan<- DeepSearchStreamEvent) *deepReport {
	// Build context
	var contextParts []string
	for i, chunk := range chunks {
		if i >= 30 {
			break
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] %s\n%s", i+1, chunk.URL, truncate(chunk.Text, 800)))
	}

	prompt := fmt.Sprintf(`Based on comprehensive research from %d sources, write a detailed report answering:

%s

Research context:
%s

Write a report with this exact structure (use markdown):

## Overview
[2-3 sentence executive summary]

## Key Findings
- Finding 1 with citation [1]
- Finding 2 with citation [2]
- Finding 3 with citation [3]

## Detailed Analysis

### [Subtopic 1]
[Detailed content with inline citations]

### [Subtopic 2]
[Detailed content with inline citations]

### [Subtopic 3]
[Detailed content with inline citations]

## Methodology
[Brief description of research approach]

Use inline citations [1], [2] etc. Be comprehensive and accurate.`, len(sources), query, strings.Join(contextParts, "\n\n"))

	stream, err := provider.ChatCompletionStream(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4096,
		Temperature: 0.7,
		Stream:      true,
	})
	if err != nil {
		return &deepReport{Overview: "Failed to generate report: " + err.Error()}
	}

	var fullText strings.Builder
	currentSection := ""

	for event := range stream {
		if event.Error != nil {
			break
		}
		if event.Delta != "" {
			fullText.WriteString(event.Delta)
			ch <- DeepSearchStreamEvent{Type: "token", Token: event.Delta}

			// Track section changes
			if strings.Contains(event.Delta, "## ") {
				lines := strings.Split(fullText.String(), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "## ") {
						newSection := strings.TrimPrefix(line, "## ")
						if newSection != currentSection {
							currentSection = newSection
							ch <- DeepSearchStreamEvent{
								Type: "progress",
								Progress: &DeepSearchProgress{
									Phase:       "writing",
									SectionName: currentSection,
									Message:     fmt.Sprintf("Writing: %s", currentSection),
								},
							}
						}
					}
				}
			}
		}
	}

	return s.parseDeepReport(fullText.String())
}

// parseDeepReport parses markdown report into structured format.
func (s *Service) parseDeepReport(content string) *deepReport {
	report := &deepReport{}
	lines := strings.Split(content, "\n")

	currentSection := ""
	var currentContent strings.Builder
	var sections []ReportSection
	sectionOrder := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Save previous section
			if currentSection != "" && currentSection != "Overview" && currentSection != "Key Findings" && currentSection != "Methodology" {
				sections = append(sections, ReportSection{
					Title:   currentSection,
					Content: strings.TrimSpace(currentContent.String()),
					Order:   sectionOrder,
				})
				sectionOrder++
			} else if currentSection == "Overview" {
				report.Overview = strings.TrimSpace(currentContent.String())
			} else if currentSection == "Key Findings" {
				// Parse key findings
				for _, l := range strings.Split(currentContent.String(), "\n") {
					l = strings.TrimSpace(l)
					if strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "* ") {
						report.KeyFindings = append(report.KeyFindings, strings.TrimLeft(l, "- *"))
					}
				}
			} else if currentSection == "Methodology" {
				report.Methodology = strings.TrimSpace(currentContent.String())
			}

			currentSection = strings.TrimPrefix(line, "## ")
			currentContent.Reset()
		} else if strings.HasPrefix(line, "### ") {
			// Subsection within Detailed Analysis
			if currentContent.Len() > 0 {
				sections = append(sections, ReportSection{
					Title:   currentSection,
					Content: strings.TrimSpace(currentContent.String()),
					Order:   sectionOrder,
				})
				sectionOrder++
			}
			currentSection = strings.TrimPrefix(line, "### ")
			currentContent.Reset()
		} else {
			currentContent.WriteString(line + "\n")
		}
	}

	// Save last section
	if currentSection != "" {
		if currentSection == "Methodology" {
			report.Methodology = strings.TrimSpace(currentContent.String())
		} else if currentSection != "Overview" && currentSection != "Key Findings" {
			sections = append(sections, ReportSection{
				Title:   currentSection,
				Content: strings.TrimSpace(currentContent.String()),
				Order:   sectionOrder,
			})
		}
	}

	report.Sections = sections
	return report
}

// formatReportAsMarkdown formats the report as markdown.
func (s *Service) formatReportAsMarkdown(report *deepReport) string {
	var sb strings.Builder

	sb.WriteString("## Overview\n\n")
	sb.WriteString(report.Overview)
	sb.WriteString("\n\n")

	if len(report.KeyFindings) > 0 {
		sb.WriteString("## Key Findings\n\n")
		for _, finding := range report.KeyFindings {
			sb.WriteString("- " + finding + "\n")
		}
		sb.WriteString("\n")
	}

	if len(report.Sections) > 0 {
		sb.WriteString("## Detailed Analysis\n\n")
		for _, section := range report.Sections {
			sb.WriteString("### " + section.Title + "\n\n")
			sb.WriteString(section.Content)
			sb.WriteString("\n\n")
		}
	}

	if report.Methodology != "" {
		sb.WriteString("## Methodology\n\n")
		sb.WriteString(report.Methodology)
	}

	return sb.String()
}

// getSnippetForSource gets a relevant snippet for a source.
func (s *Service) getSnippetForSource(source Source, chunks []chunker.Chunk) string {
	for _, chunk := range chunks {
		if chunk.URL == source.URL {
			return truncate(chunk.Text, 200)
		}
	}
	if len(source.Chunks) > 0 {
		return truncate(source.Chunks[0].Text, 200)
	}
	return ""
}
