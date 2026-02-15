package perplexity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Search executes a search query via SSE and returns structured results.
func (c *Client) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	if c.csrfToken == "" {
		if err := c.InitSession(ctx); err != nil {
			return nil, fmt.Errorf("init session: %w", err)
		}
	}

	// Validate mode/model
	modeMap, ok := modelPreference[opts.Mode]
	if !ok {
		return nil, fmt.Errorf("invalid mode: %s", opts.Mode)
	}
	modelPref, ok := modeMap[opts.Model]
	if !ok {
		return nil, fmt.Errorf("invalid model %q for mode %q", opts.Model, opts.Mode)
	}

	// Check pro quota
	if opts.Mode != ModeAuto && !c.authenticated {
		return nil, fmt.Errorf("mode %q requires authentication; use Register() first", opts.Mode)
	}

	// Build payload
	var lastUUID *string
	if opts.FollowUpUUID != "" {
		lastUUID = &opts.FollowUpUUID
	}

	lang := opts.Language
	if lang == "" {
		lang = c.cfg.Language
	}

	sources := opts.Sources
	if len(sources) == 0 {
		sources = []string{SourceWeb}
	}

	payload := ssePayload{
		QueryStr: query,
		Params: sseParams{
			Attachments:         []string{},
			FrontendContextUUID: uuid.New().String(),
			FrontendUUID:        uuid.New().String(),
			IsIncognito:         opts.Incognito,
			Language:            lang,
			LastBackendUUID:     lastUUID,
			Mode:                modePayload[opts.Mode],
			ModelPreference:     modelPref,
			Source:              "default",
			Sources:             sources,
			Version:             apiVersion,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Use a longer timeout for SSE streaming
	sseCtx, cancel := context.WithTimeout(ctx, sseReadTimeout)
	defer cancel()

	resp, err := c.doRequest(sseCtx, "POST", endpointSSEAsk, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("SSE request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("SSE request failed: HTTP %d", resp.StatusCode)
	}

	// Parse SSE stream — keep last chunk
	var lastChunk map[string]any
	err = parseSSEStream(resp.Body, func(data map[string]any) error {
		lastChunk = data

		// If streaming, we could call opts.OnChunk here
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse SSE: %w", err)
	}

	if lastChunk == nil {
		return nil, fmt.Errorf("empty SSE response")
	}

	result, err := extractSearchResult(lastChunk, query, opts)
	if err != nil {
		return nil, fmt.Errorf("extract result: %w", err)
	}

	result.SearchedAt = time.Now()
	return result, nil
}

// SearchStream executes a search query and streams each SSE event via the callback.
// The callback receives partial results as they arrive.
// Returns the final complete SearchResult.
func (c *Client) SearchStream(ctx context.Context, query string, opts SearchOptions, onEvent func(map[string]any)) (*SearchResult, error) {
	if c.csrfToken == "" {
		if err := c.InitSession(ctx); err != nil {
			return nil, fmt.Errorf("init session: %w", err)
		}
	}

	modeMap, ok := modelPreference[opts.Mode]
	if !ok {
		return nil, fmt.Errorf("invalid mode: %s", opts.Mode)
	}
	modelPref, ok := modeMap[opts.Model]
	if !ok {
		return nil, fmt.Errorf("invalid model %q for mode %q", opts.Model, opts.Mode)
	}

	if opts.Mode != ModeAuto && !c.authenticated {
		return nil, fmt.Errorf("mode %q requires authentication", opts.Mode)
	}

	var lastUUID *string
	if opts.FollowUpUUID != "" {
		lastUUID = &opts.FollowUpUUID
	}

	lang := opts.Language
	if lang == "" {
		lang = c.cfg.Language
	}

	sources := opts.Sources
	if len(sources) == 0 {
		sources = []string{SourceWeb}
	}

	payload := ssePayload{
		QueryStr: query,
		Params: sseParams{
			Attachments:         []string{},
			FrontendContextUUID: uuid.New().String(),
			FrontendUUID:        uuid.New().String(),
			IsIncognito:         opts.Incognito,
			Language:            lang,
			LastBackendUUID:     lastUUID,
			Mode:                modePayload[opts.Mode],
			ModelPreference:     modelPref,
			Source:              "default",
			Sources:             sources,
			Version:             apiVersion,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	sseCtx, cancel := context.WithTimeout(ctx, sseReadTimeout)
	defer cancel()

	resp, err := c.doRequest(sseCtx, "POST", endpointSSEAsk, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("SSE request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("SSE request failed: HTTP %d", resp.StatusCode)
	}

	var lastChunk map[string]any
	err = parseSSEStream(resp.Body, func(data map[string]any) error {
		lastChunk = data
		if onEvent != nil {
			onEvent(data)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse SSE: %w", err)
	}

	if lastChunk == nil {
		return nil, fmt.Errorf("empty SSE response")
	}

	result, err := extractSearchResult(lastChunk, query, opts)
	if err != nil {
		return nil, err
	}

	result.SearchedAt = time.Now()
	return result, nil
}
