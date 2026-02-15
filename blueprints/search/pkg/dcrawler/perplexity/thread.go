package perplexity

import (
	"context"
	"fmt"
	"time"
)

// ThreadRunner manages multi-turn conversations with Perplexity.
// It tracks conversation context via backend_uuid and stores all messages in DB.
type ThreadRunner struct {
	client *Client
	db     *DB
	thread *Thread
	opts   SearchOptions
}

// NewThreadRunner creates a runner for a new conversation thread.
func NewThreadRunner(client *Client, db *DB, opts SearchOptions) *ThreadRunner {
	return &ThreadRunner{
		client: client,
		db:     db,
		opts:   opts,
	}
}

// ResumeThread creates a runner that continues an existing thread.
func ResumeThread(client *Client, db *DB, threadID int, opts SearchOptions) (*ThreadRunner, error) {
	thread, err := db.GetThread(threadID)
	if err != nil {
		return nil, fmt.Errorf("get thread %d: %w", threadID, err)
	}

	// Inherit thread's mode/model if not overridden
	if opts.Mode == "" || opts.Mode == ModeAuto {
		opts.Mode = thread.Mode
	}
	if opts.Model == "" {
		opts.Model = thread.Model
	}

	return &ThreadRunner{
		client: client,
		db:     db,
		thread: thread,
		opts:   opts,
	}, nil
}

// Thread returns the current thread (nil if not started yet).
func (tr *ThreadRunner) Thread() *Thread {
	return tr.thread
}

// Start executes the first query and creates a new thread.
func (tr *ThreadRunner) Start(ctx context.Context, query string) (*SearchResult, error) {
	// Create thread
	title := query
	if len(title) > 100 {
		title = title[:100]
	}

	source := "sse"
	if tr.opts.Mode == "" {
		tr.opts.Mode = ModeAuto
	}

	tr.thread = &Thread{
		Title:  title,
		Mode:   tr.opts.Mode,
		Model:  tr.opts.Model,
		Source: source,
	}
	if err := tr.db.CreateThread(tr.thread); err != nil {
		return nil, fmt.Errorf("create thread: %w", err)
	}

	// Save user message
	userMsg := &ThreadMessage{
		ThreadID: tr.thread.ID,
		Role:     "user",
		Content:  query,
	}
	if err := tr.db.AddThreadMessage(userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	// Execute search
	start := time.Now()
	result, err := tr.client.Search(ctx, query, tr.opts)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	result.DurationMs = time.Since(start).Milliseconds()

	// Save assistant response
	assistantMsg := &ThreadMessage{
		ThreadID:    tr.thread.ID,
		Role:        "assistant",
		Content:     result.Answer,
		BackendUUID: result.BackendUUID,
		Citations:   result.Citations,
		WebResults:  result.WebResults,
		RelatedQ:    result.RelatedQ,
		DurationMs:  result.DurationMs,
	}
	if err := tr.db.AddThreadMessage(assistantMsg); err != nil {
		return nil, fmt.Errorf("save assistant message: %w", err)
	}

	// Also save to searches table
	tr.db.SaveSearch(result)

	return result, nil
}

// FollowUp executes a follow-up query using the thread's backend_uuid.
func (tr *ThreadRunner) FollowUp(ctx context.Context, query string) (*SearchResult, error) {
	if tr.thread == nil {
		return nil, fmt.Errorf("thread not started; call Start() first")
	}

	// Get last backend_uuid
	uuid, err := tr.db.GetLastBackendUUID(tr.thread.ID)
	if err != nil || uuid == "" {
		return nil, fmt.Errorf("no backend_uuid for follow-up (thread %d)", tr.thread.ID)
	}

	// Save user message
	userMsg := &ThreadMessage{
		ThreadID: tr.thread.ID,
		Role:     "user",
		Content:  query,
	}
	if err := tr.db.AddThreadMessage(userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	// Execute follow-up search
	opts := tr.opts
	opts.FollowUpUUID = uuid

	start := time.Now()
	result, err := tr.client.Search(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("follow-up search: %w", err)
	}
	result.DurationMs = time.Since(start).Milliseconds()

	// Save assistant response
	assistantMsg := &ThreadMessage{
		ThreadID:    tr.thread.ID,
		Role:        "assistant",
		Content:     result.Answer,
		BackendUUID: result.BackendUUID,
		Citations:   result.Citations,
		WebResults:  result.WebResults,
		RelatedQ:    result.RelatedQ,
		DurationMs:  result.DurationMs,
	}
	if err := tr.db.AddThreadMessage(assistantMsg); err != nil {
		return nil, fmt.Errorf("save assistant message: %w", err)
	}

	tr.db.SaveSearch(result)

	return result, nil
}

// RunWithFollowUps executes an initial query and N automatic follow-up questions.
// Follow-ups are picked from the response's related_queries.
// Returns all results (initial + follow-ups).
func (tr *ThreadRunner) RunWithFollowUps(ctx context.Context, query string, followUpCount int) ([]*SearchResult, error) {
	// Start with initial query
	result, err := tr.Start(ctx, query)
	if err != nil {
		return nil, err
	}

	results := []*SearchResult{result}

	// Execute follow-ups from related queries
	for i := 0; i < followUpCount && i < len(result.RelatedQ); i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		followUpQuery := result.RelatedQ[i]
		followUpResult, err := tr.FollowUp(ctx, followUpQuery)
		if err != nil {
			fmt.Printf("Follow-up %d failed: %v\n", i+1, err)
			break
		}

		results = append(results, followUpResult)
		result = followUpResult // Use latest result's related queries for next follow-up
	}

	return results, nil
}

// FormatThread formats a thread's messages for terminal display.
func FormatThread(thread *Thread, messages []ThreadMessage) string {
	out := fmt.Sprintf("Thread #%d: %s\n", thread.ID, thread.Title)
	out += fmt.Sprintf("Mode: %s  Model: %s  Source: %s  Messages: %d\n",
		thread.Mode, thread.Model, thread.Source, thread.MessageCount)
	out += fmt.Sprintf("Created: %s  Updated: %s\n\n",
		thread.CreatedAt.Format("2006-01-02 15:04"),
		thread.UpdatedAt.Format("2006-01-02 15:04"))

	for _, m := range messages {
		switch m.Role {
		case "user":
			out += fmt.Sprintf(">> %s\n\n", m.Content)
		case "assistant":
			out += fmt.Sprintf("%s\n", m.Content)
			if len(m.Citations) > 0 {
				out += "\nCitations:\n"
				for i, c := range m.Citations {
					if c.Title != "" {
						out += fmt.Sprintf("  [%d] %s — %s\n", i+1, c.URL, c.Title)
					} else {
						out += fmt.Sprintf("  [%d] %s\n", i+1, c.URL)
					}
				}
			}
			if m.DurationMs > 0 {
				out += fmt.Sprintf("\n(%dms)\n", m.DurationMs)
			}
			out += "\n"
		}
	}

	return out
}
