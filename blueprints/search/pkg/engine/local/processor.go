package local

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// Processor handles search requests for engines.
type Processor struct {
	engine     engines.Engine
	httpClient *http.Client
	suspended  *SuspendedStatus
	mu         sync.Mutex
}

// SuspendedStatus tracks engine suspension.
type SuspendedStatus struct {
	mu               sync.Mutex
	ContinuousErrors int
	SuspendEndTime   time.Time
	SuspendReason    string
}

// NewSuspendedStatus creates a new SuspendedStatus.
func NewSuspendedStatus() *SuspendedStatus {
	return &SuspendedStatus{}
}

// IsSuspended returns true if the engine is currently suspended.
func (s *SuspendedStatus) IsSuspended() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().Before(s.SuspendEndTime)
}

// Suspend suspends the engine.
func (s *SuspendedStatus) Suspend(duration time.Duration, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ContinuousErrors++
	s.SuspendEndTime = time.Now().Add(duration)
	s.SuspendReason = reason
}

// Resume resumes the engine.
func (s *SuspendedStatus) Resume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ContinuousErrors = 0
	s.SuspendEndTime = time.Time{}
	s.SuspendReason = ""
}

// NewProcessor creates a new processor for an engine.
func NewProcessor(engine engines.Engine, httpClient *http.Client) *Processor {
	return &Processor{
		engine:     engine,
		httpClient: httpClient,
		suspended:  NewSuspendedStatus(),
	}
}

// Engine returns the processor's engine.
func (p *Processor) Engine() engines.Engine {
	return p.engine
}

// IsSuspended returns true if the engine is suspended.
func (p *Processor) IsSuspended() bool {
	return p.suspended.IsSuspended()
}

// SuspendReason returns the suspension reason.
func (p *Processor) SuspendReason() string {
	p.suspended.mu.Lock()
	defer p.suspended.mu.Unlock()
	return p.suspended.SuspendReason
}

// GetParams builds request parameters for a search.
func (p *Processor) GetParams(query string, opts *SearchOptions, category engines.Category) *engines.RequestParams {
	eng := p.engine

	// Check paging support
	if opts.Page > 1 && !eng.SupportsPaging() {
		return nil
	}

	// Check max page
	if eng.MaxPage() > 0 && opts.Page > eng.MaxPage() {
		return nil
	}

	// Check time range support
	if opts.TimeRange != "" && !eng.SupportsTimeRange() {
		return nil
	}

	params := engines.NewRequestParams()
	params.Query = query
	params.Category = category
	params.PageNo = opts.Page
	params.SafeSearch = engines.SafeSearchLevel(opts.SafeSearch)
	params.TimeRange = engines.TimeRange(opts.TimeRange)
	params.Language = opts.Language
	params.Locale = opts.Locale

	// Set timeout
	timeout := eng.Timeout()
	if opts.Timeout > 0 && opts.Timeout < timeout {
		timeout = opts.Timeout
	}
	params.Timeout = timeout

	// Copy engine data
	if opts.EngineData != nil {
		if data, ok := opts.EngineData[eng.Name()]; ok {
			params.EngineData = data
		}
	}

	return params
}

// Search executes a search with the engine.
func (p *Processor) Search(ctx context.Context, query string, params *engines.RequestParams) (*engines.EngineResults, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch eng := p.engine.(type) {
	case engines.OnlineEngine:
		return p.searchOnline(ctx, eng, query, params)
	case engines.OfflineEngine:
		return p.searchOffline(ctx, eng, query, params)
	default:
		return nil, fmt.Errorf("unsupported engine type for %s", p.engine.Name())
	}
}

func (p *Processor) searchOnline(ctx context.Context, eng engines.OnlineEngine, query string, params *engines.RequestParams) (*engines.EngineResults, error) {
	engineName := eng.Name()

	// Build request
	if err := eng.Request(ctx, query, params); err != nil {
		slog.Warn("engine request build failed",
			"engine", engineName,
			"query", query,
			"error", err,
		)
		return nil, fmt.Errorf("request build failed: %w", err)
	}

	// Skip if URL is empty
	if params.URL == "" {
		slog.Debug("engine skipped - no URL generated",
			"engine", engineName,
			"query", query,
		)
		return nil, nil
	}

	// Create HTTP request
	req, err := p.buildHTTPRequest(ctx, params)
	if err != nil {
		slog.Warn("engine http request build failed",
			"engine", engineName,
			"error", err,
		)
		return nil, fmt.Errorf("http request build failed: %w", err)
	}

	// Set timeout
	timeout := params.Timeout
	if timeout == 0 {
		timeout = eng.Timeout()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.suspended.Suspend(time.Minute, err.Error())
		slog.Warn("engine http request failed",
			"engine", engineName,
			"query", query,
			"url", params.URL,
			"error", err,
		)
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if params.RaiseForHTTPError && resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		p.suspended.Suspend(time.Minute, fmt.Sprintf("HTTP %d", resp.StatusCode))
		slog.Warn("engine http error",
			"engine", engineName,
			"query", query,
			"status", resp.StatusCode,
			"body", string(body),
		)
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	results, err := eng.Response(ctx, resp, params)
	if err != nil {
		slog.Warn("engine response parse failed",
			"engine", engineName,
			"query", query,
			"error", err,
		)
		return nil, fmt.Errorf("response parse failed: %w", err)
	}

	// Log empty results
	if results == nil || len(results.Results) == 0 {
		slog.Debug("engine returned no results",
			"engine", engineName,
			"query", query,
		)
	} else {
		slog.Debug("engine search completed",
			"engine", engineName,
			"query", query,
			"results", len(results.Results),
		)
	}

	// Resume on success
	p.suspended.Resume()
	return results, nil
}

func (p *Processor) searchOffline(ctx context.Context, eng engines.OfflineEngine, query string, params *engines.RequestParams) (*engines.EngineResults, error) {
	results, err := eng.Search(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("offline search failed: %w", err)
	}
	return results, nil
}

func (p *Processor) buildHTTPRequest(ctx context.Context, params *engines.RequestParams) (*http.Request, error) {
	var body io.Reader

	method := strings.ToUpper(params.Method)
	if method == "" {
		method = "GET"
	}

	// Build body for POST requests
	if method == "POST" {
		if len(params.JSON) > 0 {
			// JSON body handling would go here
			// For now, use form data
		}
		if len(params.Data) > 0 {
			body = strings.NewReader(params.Data.Encode())
		}
		if len(params.Content) > 0 {
			body = bytes.NewReader(params.Content)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, params.URL, body)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, values := range params.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Set default headers if not set
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	}

	// Set Content-Type for POST
	if method == "POST" && req.Header.Get("Content-Type") == "" {
		if len(params.JSON) > 0 {
			req.Header.Set("Content-Type", "application/json")
		} else {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	// Set cookies
	for _, cookie := range params.Cookies {
		req.AddCookie(cookie)
	}

	return req, nil
}

// ProcessorMap manages processors for all engines.
type ProcessorMap struct {
	mu         sync.RWMutex
	processors map[string]*Processor
}

// NewProcessorMap creates a new ProcessorMap.
func NewProcessorMap() *ProcessorMap {
	return &ProcessorMap{
		processors: make(map[string]*Processor),
	}
}

// Get returns a processor by engine name.
func (pm *ProcessorMap) Get(name string) (*Processor, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	p, ok := pm.processors[name]
	return p, ok
}

// Set sets a processor for an engine.
func (pm *ProcessorMap) Set(name string, processor *Processor) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.processors[name] = processor
}

// Delete removes a processor.
func (pm *ProcessorMap) Delete(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.processors, name)
}

// All returns all processors.
func (pm *ProcessorMap) All() []*Processor {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make([]*Processor, 0, len(pm.processors))
	for _, p := range pm.processors {
		result = append(result, p)
	}
	return result
}

// BuildURL builds a URL with query parameters.
func BuildURL(baseURL string, params map[string]string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u.String()
}
