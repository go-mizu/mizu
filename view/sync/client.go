package sync

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Options configures the sync client.
type Options struct {
	// BaseURL is the sync server endpoint (e.g., "https://api.example.com/_sync")
	BaseURL string

	// Scope is the data partition identifier
	Scope string

	// HTTP is the HTTP client to use. Default: http.DefaultClient with 30s timeout
	HTTP *http.Client

	// Persistence is the persistence implementation. Default: MemoryPersistence
	Persistence Persistence

	// OnError is called when background operations fail
	OnError func(error)

	// OnSync is called after successful sync
	OnSync func(cursor uint64)

	// OnOnline is called when client goes online
	OnOnline func()

	// OnOffline is called when client goes offline
	OnOffline func()

	// PushInterval is the interval for pushing mutations. Default: 1s
	PushInterval time.Duration

	// PullInterval is the interval for polling changes. Default: 30s
	PullInterval time.Duration

	// RetryBackoff is the initial retry backoff. Default: 1s
	RetryBackoff time.Duration

	// MaxRetryBackoff is the maximum retry backoff. Default: 30s
	MaxRetryBackoff time.Duration
}

// Client is the main sync runtime.
type Client struct {
	opts        Options
	store       *Store
	queue       *Queue
	transport   *Transport
	cursor      uint64
	cursorMu    sync.RWMutex
	collections map[string]collectionRef
	collMu      sync.RWMutex
	started     atomic.Bool
	online      atomic.Bool
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	pushCh      chan struct{}
	liveCh      chan uint64 // Live push notifications
}

// New creates a new sync client with the given options.
func New(opts Options) *Client {
	if opts.HTTP == nil {
		opts.HTTP = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.Persistence == nil {
		opts.Persistence = NewMemoryPersistence()
	}
	if opts.PushInterval == 0 {
		opts.PushInterval = time.Second
	}
	if opts.PullInterval == 0 {
		opts.PullInterval = 30 * time.Second
	}
	if opts.RetryBackoff == 0 {
		opts.RetryBackoff = time.Second
	}
	if opts.MaxRetryBackoff == 0 {
		opts.MaxRetryBackoff = 30 * time.Second
	}

	c := &Client{
		opts:        opts,
		store:       NewStore(),
		queue:       NewQueue(),
		transport:   NewTransport(opts.BaseURL, opts.HTTP),
		collections: make(map[string]collectionRef),
		pushCh:      make(chan struct{}, 1),
		liveCh:      make(chan uint64, 16),
	}

	// Load persisted state
	c.loadState()

	return c
}

// Start begins background sync operations.
func (c *Client) Start(ctx context.Context) error {
	if c.started.Swap(true) {
		return ErrAlreadyStarted
	}

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Initial sync
	if err := c.initialSync(); err != nil {
		// Continue anyway, we can work offline
		if c.opts.OnError != nil {
			c.opts.OnError(err)
		}
	}

	// Start background goroutines
	c.wg.Add(2)
	go c.pushLoop()
	go c.pullLoop()

	return nil
}

// Stop halts all background operations.
func (c *Client) Stop() {
	if !c.started.Load() {
		return
	}

	c.cancel()
	c.wg.Wait()
	c.started.Store(false)

	// Persist state
	c.saveState()
}

// Sync triggers an immediate sync cycle.
func (c *Client) Sync() error {
	if !c.started.Load() {
		return ErrNotStarted
	}

	// Push pending mutations first
	if err := c.push(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	// Then pull changes
	return c.pull()
}

// Mutate queues a new mutation.
func (c *Client) Mutate(name string, args map[string]any) string {
	m := Mutation{
		Name:  name,
		Scope: c.opts.Scope,
		Args:  args,
	}
	id := c.queue.Push(m)

	// Trigger push
	select {
	case c.pushCh <- struct{}{}:
	default:
	}

	return id
}

// Store returns the local store.
func (c *Client) Store() *Store {
	return c.store
}

// Queue returns the mutation queue.
func (c *Client) Queue() *Queue {
	return c.queue
}

// Cursor returns the current sync cursor.
func (c *Client) Cursor() uint64 {
	c.cursorMu.RLock()
	defer c.cursorMu.RUnlock()
	return c.cursor
}

// IsOnline returns whether the client is online.
func (c *Client) IsOnline() bool {
	return c.online.Load()
}

// NotifyLive notifies the client of a live push (from live package).
func (c *Client) NotifyLive(cursor uint64) {
	select {
	case c.liveCh <- cursor:
	default:
	}
}

func (c *Client) registerCollection(name string, col collectionRef) {
	c.collMu.Lock()
	c.collections[name] = col
	c.collMu.Unlock()
}

func (c *Client) loadState() {
	// Load cursor
	cursor, err := c.opts.Persistence.LoadCursor()
	if err == nil {
		c.cursor = cursor
	}

	// Load queue
	mutations, seq, err := c.opts.Persistence.LoadQueue()
	if err == nil && len(mutations) > 0 {
		c.queue.Load(mutations)
		c.queue.SetSeq(seq)
	}

	// Load client ID
	clientID, err := c.opts.Persistence.LoadClientID()
	if err == nil && clientID != "" {
		c.queue.SetClientID(clientID)
	} else {
		// Save generated client ID
		_ = c.opts.Persistence.SaveClientID(c.queue.ClientID())
	}

	// Load store
	data, err := c.opts.Persistence.LoadStore()
	if err == nil && data != nil {
		c.store.Load(data)
	}
}

func (c *Client) saveState() {
	_ = c.opts.Persistence.SaveCursor(c.cursor)
	_ = c.opts.Persistence.SaveQueue(c.queue.Pending(), c.queue.CurrentSeq())
	_ = c.opts.Persistence.SaveStore(c.store.Snapshot())
}

func (c *Client) initialSync() error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	// If we have no cursor, do a full snapshot
	if c.cursor == 0 {
		resp, err := c.transport.Snapshot(ctx, c.opts.Scope)
		if err != nil {
			c.setOnline(false)
			return err
		}

		c.store.Load(resp.Data)
		c.setCursor(resp.Cursor)
		c.setOnline(true)
		return nil
	}

	// Otherwise, pull changes since cursor
	return c.pull()
}

func (c *Client) pushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.opts.PushInterval)
	defer ticker.Stop()

	backoff := c.opts.RetryBackoff

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.pushCh:
			if err := c.push(); err != nil {
				if c.opts.OnError != nil {
					c.opts.OnError(err)
				}
			} else {
				backoff = c.opts.RetryBackoff
			}
		case <-ticker.C:
			if c.queue.Len() > 0 {
				if err := c.push(); err != nil {
					if c.opts.OnError != nil {
						c.opts.OnError(err)
					}
					// Exponential backoff
					time.Sleep(backoff)
					backoff *= 2
					if backoff > c.opts.MaxRetryBackoff {
						backoff = c.opts.MaxRetryBackoff
					}
				} else {
					backoff = c.opts.RetryBackoff
				}
			}
		}
	}
}

func (c *Client) pullLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.opts.PullInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case cursor := <-c.liveCh:
			// Live notification - pull if cursor advanced
			if cursor > c.Cursor() {
				if err := c.pull(); err != nil {
					if c.opts.OnError != nil {
						c.opts.OnError(err)
					}
				}
			}
		case <-ticker.C:
			if err := c.pull(); err != nil {
				if c.opts.OnError != nil {
					c.opts.OnError(err)
				}
			}
		}
	}
}

func (c *Client) push() error {
	mutations := c.queue.Pending()
	if len(mutations) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	resp, err := c.transport.Push(ctx, mutations)
	if err != nil {
		c.setOnline(false)
		return err
	}

	c.setOnline(true)

	// Process results
	var maxCursor uint64
	for i, result := range resp.Results {
		if i >= len(mutations) {
			break
		}
		if result.OK {
			c.queue.Remove(mutations[i].ID)
			if result.Cursor > maxCursor {
				maxCursor = result.Cursor
			}
		} else if result.Code == "conflict" {
			// Need to re-sync
			return c.handleConflict()
		}
	}

	if maxCursor > 0 {
		c.setCursor(maxCursor)
		if c.opts.OnSync != nil {
			c.opts.OnSync(maxCursor)
		}
	}

	return nil
}

func (c *Client) pull() error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	cursor := c.Cursor()

	for {
		resp, err := c.transport.Pull(ctx, c.opts.Scope, cursor, 100)
		if err != nil {
			c.setOnline(false)
			if errors.Is(err, ErrCursorTooOld) {
				return c.handleCursorTooOld()
			}
			return err
		}

		c.setOnline(true)

		// Apply changes
		for _, change := range resp.Changes {
			c.applyChange(change)
			if change.Cursor > cursor {
				cursor = change.Cursor
			}
		}

		if len(resp.Changes) > 0 {
			c.setCursor(cursor)
			if c.opts.OnSync != nil {
				c.opts.OnSync(cursor)
			}
		}

		if !resp.HasMore {
			break
		}
	}

	return nil
}

func (c *Client) applyChange(change Change) {
	switch change.Op {
	case OpCreate, OpUpdate:
		c.store.Set(change.Entity, change.ID, change.Data)
	case OpDelete:
		c.store.Delete(change.Entity, change.ID)
	}
}

func (c *Client) handleConflict() error {
	// Clear queue and re-sync
	c.queue.Clear()
	return c.handleCursorTooOld()
}

func (c *Client) handleCursorTooOld() error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	resp, err := c.transport.Snapshot(ctx, c.opts.Scope)
	if err != nil {
		return err
	}

	c.store.Load(resp.Data)
	c.setCursor(resp.Cursor)

	if c.opts.OnSync != nil {
		c.opts.OnSync(resp.Cursor)
	}

	return nil
}

func (c *Client) setCursor(cursor uint64) {
	c.cursorMu.Lock()
	c.cursor = cursor
	c.cursorMu.Unlock()
	_ = c.opts.Persistence.SaveCursor(cursor)
}

func (c *Client) setOnline(online bool) {
	was := c.online.Swap(online)
	if was != online {
		if online && c.opts.OnOnline != nil {
			c.opts.OnOnline()
		} else if !online && c.opts.OnOffline != nil {
			c.opts.OnOffline()
		}
	}
}
