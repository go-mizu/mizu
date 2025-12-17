// Package sync provides a client-side synchronization runtime for offline-first
// interactive applications.
//
// It integrates:
//   - sync package as the authoritative correctness layer
//   - Reactive state management (Signal, Computed, Effect) for UI binding
//   - Optional live package as a latency accelerator
//
// # Core Concepts
//
// Client is the main runtime coordinating local state, mutation queue, and sync:
//
//	client := sync.New(sync.Options{
//	    BaseURL: "https://api.example.com/_sync",
//	    Scope:   "user:123",
//	})
//	client.Start(ctx)
//
// Signal provides reactive state that notifies dependents on change:
//
//	count := sync.NewSignal(0)
//	count.Set(1) // Triggers dependents
//
// Computed derives values that recompute when dependencies change:
//
//	doubled := sync.NewComputed(func() int {
//	    return count.Get() * 2
//	})
//
// Effect runs side effects when dependencies change:
//
//	sync.NewEffect(func() {
//	    fmt.Println("Count:", count.Get())
//	})
//
// Collection manages synchronized entities:
//
//	todos := sync.NewCollection[Todo](client, "todo")
//	todo := todos.Create("abc", Todo{Title: "Buy milk"})
//
// # Offline Support
//
// The client operates fully offline. Mutations are queued locally and pushed
// when connectivity is restored. The sync protocol ensures idempotent,
// conflict-free convergence.
//
// # Live Integration
//
// When a live connection is provided, the client receives push notifications
// for immediate sync triggers, reducing latency compared to polling.
package sync

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	gosync "sync"
	"sync/atomic"
	"time"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var (
	// ErrNotStarted is returned when client operations are called before Start.
	ErrNotStarted = errors.New("sync: client not started")

	// ErrAlreadyStarted is returned when Start is called on a running client.
	ErrAlreadyStarted = errors.New("sync: client already started")
)

// Internal error for cursor too old (triggers re-sync).
var errCursorTooOld = errors.New("sync: cursor too old")

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

// Op defines the type of change operation.
type Op string

const (
	OpCreate Op = "create"
	OpUpdate Op = "update"
	OpDelete Op = "delete"
)

// mutation represents a client request to change state.
type mutation struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Scope     string         `json:"scope,omitempty"`
	Client    string         `json:"client,omitempty"`
	Seq       uint64         `json:"seq,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
}

// change is a single durable state change from the server.
type change struct {
	Cursor uint64    `json:"cursor"`
	Scope  string    `json:"scope"`
	Entity string    `json:"entity"`
	ID     string    `json:"id"`
	Op     Op        `json:"op"`
	Data   []byte    `json:"data,omitempty"`
	Time   time.Time `json:"time"`
}

// result describes the outcome of applying a mutation.
type result struct {
	OK      bool     `json:"ok"`
	Cursor  uint64   `json:"cursor,omitempty"`
	Code    string   `json:"code,omitempty"`
	Error   string   `json:"error,omitempty"`
	Changes []change `json:"changes,omitempty"`
}

// -----------------------------------------------------------------------------
// Reactive: Signal
// -----------------------------------------------------------------------------

// tracker is the global dependency tracker for reactive computations.
var tracker = newDependencyTracker()

// dependencyTracker tracks signal access during computation.
type dependencyTracker struct {
	mu    gosync.Mutex
	stack []*computedBase // Stack of currently computing computeds
}

func newDependencyTracker() *dependencyTracker {
	return &dependencyTracker{}
}

func (t *dependencyTracker) push(c *computedBase) {
	t.mu.Lock()
	t.stack = append(t.stack, c)
	t.mu.Unlock()
}

func (t *dependencyTracker) pop() {
	t.mu.Lock()
	if len(t.stack) > 0 {
		t.stack = t.stack[:len(t.stack)-1]
	}
	t.mu.Unlock()
}

func (t *dependencyTracker) current() *computedBase {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.stack) == 0 {
		return nil
	}
	return t.stack[len(t.stack)-1]
}

// subscriber is something that can be notified when a signal changes.
type subscriber interface {
	notify()
}

// Signal is a reactive value container that notifies dependents on change.
type Signal[T any] struct {
	mu          gosync.RWMutex
	value       T
	subscribers map[subscriber]struct{}
	version     uint64
}

// NewSignal creates a new signal with the given initial value.
func NewSignal[T any](initial T) *Signal[T] {
	return &Signal[T]{
		value:       initial,
		subscribers: make(map[subscriber]struct{}),
	}
}

// Get returns the current value and registers as a dependency if within a computation.
func (s *Signal[T]) Get() T {
	s.mu.RLock()
	value := s.value
	s.mu.RUnlock()

	// Register dependency if we're inside a computed/effect
	if c := tracker.current(); c != nil {
		s.subscribe(c)
	}

	return value
}

// Set updates the value and notifies all subscribers.
func (s *Signal[T]) Set(value T) {
	s.mu.Lock()
	s.value = value
	s.version++
	subs := make([]subscriber, 0, len(s.subscribers))
	for sub := range s.subscribers {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	// Notify outside lock to avoid deadlock
	for _, sub := range subs {
		sub.notify()
	}
}

// Update applies a function to the current value.
func (s *Signal[T]) Update(fn func(T) T) {
	s.mu.Lock()
	s.value = fn(s.value)
	s.version++
	subs := make([]subscriber, 0, len(s.subscribers))
	for sub := range s.subscribers {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	for _, sub := range subs {
		sub.notify()
	}
}

func (s *Signal[T]) subscribe(sub subscriber) {
	s.mu.Lock()
	s.subscribers[sub] = struct{}{}
	s.mu.Unlock()
}

// -----------------------------------------------------------------------------
// Reactive: Computed
// -----------------------------------------------------------------------------

// computedBase is the non-generic base for dependency tracking.
type computedBase struct {
	dirty       atomic.Bool
	running     atomic.Bool
	mu          gosync.RWMutex
	subscribers map[subscriber]struct{}
}

func (c *computedBase) notify() {
	c.dirty.Store(true)
	// Propagate to our subscribers
	c.mu.RLock()
	subs := make([]subscriber, 0, len(c.subscribers))
	for sub := range c.subscribers {
		subs = append(subs, sub)
	}
	c.mu.RUnlock()

	for _, sub := range subs {
		sub.notify()
	}
}

func (c *computedBase) subscribe(sub subscriber) {
	c.mu.Lock()
	if c.subscribers == nil {
		c.subscribers = make(map[subscriber]struct{})
	}
	c.subscribers[sub] = struct{}{}
	c.mu.Unlock()
}

// Computed is a derived value that recomputes when dependencies change.
type Computed[T any] struct {
	base  computedBase
	fn    func() T
	value T
	mu    gosync.RWMutex
}

// NewComputed creates a new computed value with the given computation function.
func NewComputed[T any](fn func() T) *Computed[T] {
	c := &Computed[T]{
		fn: fn,
	}
	c.base.dirty.Store(true)
	return c
}

// Get returns the computed value, recomputing if necessary.
func (c *Computed[T]) Get() T {
	// Register as dependency if we're inside another computed
	if outer := tracker.current(); outer != nil {
		c.base.subscribe(outer)
	}

	if !c.base.dirty.Load() {
		c.mu.RLock()
		v := c.value
		c.mu.RUnlock()
		return v
	}

	// Prevent recursive recomputation
	if c.base.running.Swap(true) {
		c.mu.RLock()
		v := c.value
		c.mu.RUnlock()
		return v
	}
	defer c.base.running.Store(false)

	// Track dependencies during computation
	tracker.push(&c.base)
	value := c.fn()
	tracker.pop()

	c.mu.Lock()
	c.value = value
	c.mu.Unlock()
	c.base.dirty.Store(false)

	return value
}

// -----------------------------------------------------------------------------
// Reactive: Effect
// -----------------------------------------------------------------------------

// effectTracker wraps an Effect for dependency tracking.
type effectTracker struct {
	effect *Effect
}

func (t *effectTracker) notify() {
	t.effect.triggerRun()
}

// Effect runs a side effect when dependencies change.
type Effect struct {
	tracker effectTracker
	fn      func()
	stopped atomic.Bool
	dirty   atomic.Bool
	running atomic.Bool
	mu      gosync.Mutex
}

// NewEffect creates and immediately runs a new effect.
func NewEffect(fn func()) *Effect {
	e := &Effect{fn: fn}
	e.tracker.effect = e
	e.run()
	return e
}

// Stop prevents future runs of the effect.
func (e *Effect) Stop() {
	e.stopped.Store(true)
}

func (e *Effect) run() {
	if e.stopped.Load() {
		return
	}

	// Prevent recursive runs
	if e.running.Swap(true) {
		return
	}
	defer e.running.Store(false)

	e.mu.Lock()
	defer e.mu.Unlock()

	// Track dependencies using a temporary computedBase
	base := &computedBase{}
	tracker.push(base)
	e.fn()
	tracker.pop()

	// Subscribe this effect to all accessed signals
	base.mu.Lock()
	base.subscribers = map[subscriber]struct{}{&e.tracker: {}}
	base.mu.Unlock()

	e.dirty.Store(false)
}

func (e *Effect) triggerRun() {
	e.dirty.Store(true)
	// Run asynchronously to avoid blocking signal updates
	go e.run()
}

// -----------------------------------------------------------------------------
// Store (internal)
// -----------------------------------------------------------------------------

// store is the local state container for synchronized data.
type store struct {
	mu       gosync.RWMutex
	data     map[string]map[string][]byte // entity -> id -> data
	version  *Signal[uint64]              // Global version for reactivity
	onChange func(entity, id string, op Op)
}

// newStore creates a new empty store.
func newStore() *store {
	return &store{
		data:    make(map[string]map[string][]byte),
		version: NewSignal[uint64](0),
	}
}

func (s *store) get(entity, id string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[entity]
	if !ok {
		return nil, false
	}
	data, ok := items[id]
	if !ok {
		return nil, false
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, true
}

func (s *store) set(entity, id string, data []byte) {
	s.mu.Lock()

	if s.data[entity] == nil {
		s.data[entity] = make(map[string][]byte)
	}

	_, exists := s.data[entity][id]
	op := OpCreate
	if exists {
		op = OpUpdate
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	s.data[entity][id] = cp

	s.mu.Unlock()

	s.version.Update(func(v uint64) uint64 { return v + 1 })

	if s.onChange != nil {
		s.onChange(entity, id, op)
	}
}

func (s *store) delete(entity, id string) {
	s.mu.Lock()

	deleted := false
	if items, ok := s.data[entity]; ok {
		if _, exists := items[id]; exists {
			delete(items, id)
			deleted = true
			if len(items) == 0 {
				delete(s.data, entity)
			}
		}
	}

	s.mu.Unlock()

	if deleted {
		s.version.Update(func(v uint64) uint64 { return v + 1 })
		if s.onChange != nil {
			s.onChange(entity, id, OpDelete)
		}
	}
}

func (s *store) has(entity, id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[entity]
	if !ok {
		return false
	}
	_, ok = items[id]
	return ok
}

func (s *store) list(entity string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[entity]
	if !ok {
		return nil
	}

	ids := make([]string, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	return ids
}

func (s *store) all(entity string) map[string][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.data[entity]
	if !ok {
		return nil
	}

	result := make(map[string][]byte, len(items))
	for id, data := range items {
		cp := make([]byte, len(data))
		copy(cp, data)
		result[id] = cp
	}
	return result
}

func (s *store) snapshot() map[string]map[string][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]map[string][]byte, len(s.data))
	for entity, items := range s.data {
		result[entity] = make(map[string][]byte, len(items))
		for id, data := range items {
			cp := make([]byte, len(data))
			copy(cp, data)
			result[entity][id] = cp
		}
	}
	return result
}

func (s *store) load(data map[string]map[string][]byte) {
	s.mu.Lock()

	s.data = make(map[string]map[string][]byte, len(data))

	for entity, items := range data {
		s.data[entity] = make(map[string][]byte, len(items))
		for id, bytes := range items {
			cp := make([]byte, len(bytes))
			copy(cp, bytes)
			s.data[entity][id] = cp
		}
	}

	s.mu.Unlock()

	s.version.Update(func(v uint64) uint64 { return v + 1 })
}

func (s *store) count(entity string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data[entity])
}

// -----------------------------------------------------------------------------
// Queue (internal)
// -----------------------------------------------------------------------------

// queue manages pending mutations.
type queue struct {
	mu        gosync.RWMutex
	mutations []mutation
	byID      map[string]int
	seq       uint64
	clientID  string
}

func newQueue() *queue {
	return &queue{
		byID:     make(map[string]int),
		clientID: generateClientID(),
	}
}

func (q *queue) push(m mutation) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	if m.ID == "" {
		m.ID = generateMutationID()
	}

	if _, exists := q.byID[m.ID]; exists {
		return m.ID
	}

	q.seq++
	m.Seq = q.seq
	m.Client = q.clientID

	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}

	q.byID[m.ID] = len(q.mutations)
	q.mutations = append(q.mutations, m)

	return m.ID
}

func (q *queue) pending() []mutation {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]mutation, len(q.mutations))
	copy(result, q.mutations)
	return result
}

func (q *queue) len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.mutations)
}

func (q *queue) clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.mutations = nil
	q.byID = make(map[string]int)
}

func (q *queue) remove(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	idx, ok := q.byID[id]
	if !ok {
		return
	}

	q.mutations = append(q.mutations[:idx], q.mutations[idx+1:]...)

	delete(q.byID, id)
	for i := idx; i < len(q.mutations); i++ {
		q.byID[q.mutations[i].ID] = i
	}
}

func (q *queue) getClientID() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.clientID
}

func (q *queue) setClientID(id string) {
	q.mu.Lock()
	q.clientID = id
	q.mu.Unlock()
}

func (q *queue) currentSeq() uint64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.seq
}

func (q *queue) setSeq(seq uint64) {
	q.mu.Lock()
	q.seq = seq
	q.mu.Unlock()
}

func (q *queue) loadMutations(mutations []mutation) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.mutations = make([]mutation, len(mutations))
	copy(q.mutations, mutations)
	q.byID = make(map[string]int, len(mutations))

	for i, m := range q.mutations {
		q.byID[m.ID] = i
		if m.Seq > q.seq {
			q.seq = m.Seq
		}
	}
}

func generateMutationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func generateClientID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// -----------------------------------------------------------------------------
// Transport (internal)
// -----------------------------------------------------------------------------

type transport struct {
	baseURL string
	http    *http.Client
}

func newTransport(baseURL string, httpClient *http.Client) *transport {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &transport{
		baseURL: baseURL,
		http:    httpClient,
	}
}

func (t *transport) pushMutations(ctx context.Context, mutations []mutation) ([]result, error) {
	req := struct {
		Mutations []mutation `json:"mutations"`
	}{Mutations: mutations}

	var resp struct {
		Results []result `json:"results"`
	}
	if err := t.post(ctx, "/push", req, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (t *transport) pull(ctx context.Context, scope string, cursor uint64, limit int) ([]change, bool, error) {
	req := struct {
		Scope  string `json:"scope,omitempty"`
		Cursor uint64 `json:"cursor"`
		Limit  int    `json:"limit,omitempty"`
	}{Scope: scope, Cursor: cursor, Limit: limit}

	var resp struct {
		Changes []change `json:"changes"`
		HasMore bool     `json:"has_more"`
	}
	if err := t.post(ctx, "/pull", req, &resp); err != nil {
		return nil, false, err
	}
	return resp.Changes, resp.HasMore, nil
}

func (t *transport) snapshot(ctx context.Context, scope string) (map[string]map[string][]byte, uint64, error) {
	req := struct {
		Scope string `json:"scope,omitempty"`
	}{Scope: scope}

	var resp struct {
		Data   map[string]map[string][]byte `json:"data"`
		Cursor uint64                       `json:"cursor"`
	}
	if err := t.post(ctx, "/snapshot", req, &resp); err != nil {
		return nil, 0, err
	}
	return resp.Data, resp.Cursor, nil
}

func (t *transport) post(ctx context.Context, path string, body, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusGone {
		return errCursorTooOld
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Code  string `json:"code"`
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Persistence
// -----------------------------------------------------------------------------

// Persistence defines the interface for persisting client state.
type Persistence interface {
	// Save persists all client state atomically.
	Save(cursor uint64, clientID string, queue []mutation, store map[string]map[string][]byte) error

	// Load restores client state.
	Load() (cursor uint64, clientID string, queue []mutation, store map[string]map[string][]byte, err error)
}

// -----------------------------------------------------------------------------
// Client
// -----------------------------------------------------------------------------

// Options configures the sync client.
type Options struct {
	// BaseURL is the sync server endpoint (e.g., "https://api.example.com/_sync")
	BaseURL string

	// Scope is the data partition identifier
	Scope string

	// HTTP is the HTTP client to use. Default: http.DefaultClient with 30s timeout
	HTTP *http.Client

	// Persistence is the persistence implementation. Default: no persistence
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
}

// Client is the main sync runtime.
type Client struct {
	opts        Options
	store       *store
	queue       *queue
	transport   *transport
	cursor      uint64
	cursorMu    gosync.RWMutex
	collections map[string]collectionRef
	collMu      gosync.RWMutex
	started     atomic.Bool
	online      atomic.Bool
	ctx         context.Context
	cancel      context.CancelFunc
	wg          gosync.WaitGroup
	pushCh      chan struct{}
	liveCh      chan uint64
}

// New creates a new sync client with the given options.
func New(opts Options) *Client {
	if opts.HTTP == nil {
		opts.HTTP = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.PushInterval == 0 {
		opts.PushInterval = time.Second
	}
	if opts.PullInterval == 0 {
		opts.PullInterval = 30 * time.Second
	}

	c := &Client{
		opts:        opts,
		store:       newStore(),
		queue:       newQueue(),
		transport:   newTransport(opts.BaseURL, opts.HTTP),
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

	c.saveState()
}

// Sync triggers an immediate sync cycle.
func (c *Client) Sync() error {
	if !c.started.Load() {
		return ErrNotStarted
	}

	if err := c.push(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return c.pull()
}

// Mutate queues a new mutation.
func (c *Client) Mutate(name string, args map[string]any) string {
	m := mutation{
		Name:  name,
		Scope: c.opts.Scope,
		Args:  args,
	}
	id := c.queue.push(m)

	select {
	case c.pushCh <- struct{}{}:
	default:
	}

	return id
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
	if c.opts.Persistence == nil {
		return
	}

	cursor, clientID, mutations, storeData, err := c.opts.Persistence.Load()
	if err != nil {
		return
	}

	c.cursor = cursor
	if clientID != "" {
		c.queue.setClientID(clientID)
	}
	if len(mutations) > 0 {
		c.queue.loadMutations(mutations)
	}
	if storeData != nil {
		c.store.load(storeData)
	}
}

func (c *Client) saveState() {
	if c.opts.Persistence == nil {
		return
	}
	_ = c.opts.Persistence.Save(
		c.cursor,
		c.queue.getClientID(),
		c.queue.pending(),
		c.store.snapshot(),
	)
}

func (c *Client) initialSync() error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	if c.cursor == 0 {
		data, cursor, err := c.transport.snapshot(ctx, c.opts.Scope)
		if err != nil {
			c.setOnline(false)
			return err
		}

		c.store.load(data)
		c.setCursor(cursor)
		c.setOnline(true)
		return nil
	}

	return c.pull()
}

func (c *Client) pushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.opts.PushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.pushCh:
			if err := c.push(); err != nil && c.opts.OnError != nil {
				c.opts.OnError(err)
			}
		case <-ticker.C:
			if c.queue.len() > 0 {
				if err := c.push(); err != nil && c.opts.OnError != nil {
					c.opts.OnError(err)
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
			if cursor > c.Cursor() {
				if err := c.pull(); err != nil && c.opts.OnError != nil {
					c.opts.OnError(err)
				}
			}
		case <-ticker.C:
			if err := c.pull(); err != nil && c.opts.OnError != nil {
				c.opts.OnError(err)
			}
		}
	}
}

func (c *Client) push() error {
	mutations := c.queue.pending()
	if len(mutations) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	results, err := c.transport.pushMutations(ctx, mutations)
	if err != nil {
		c.setOnline(false)
		return err
	}

	c.setOnline(true)

	var maxCursor uint64
	for i, res := range results {
		if i >= len(mutations) {
			break
		}
		if res.OK {
			c.queue.remove(mutations[i].ID)
			if res.Cursor > maxCursor {
				maxCursor = res.Cursor
			}
		} else if res.Code == "conflict" {
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
		changes, hasMore, err := c.transport.pull(ctx, c.opts.Scope, cursor, 100)
		if err != nil {
			c.setOnline(false)
			if errors.Is(err, errCursorTooOld) {
				return c.handleCursorTooOld()
			}
			return err
		}

		c.setOnline(true)

		for _, chg := range changes {
			c.applyChange(chg)
			if chg.Cursor > cursor {
				cursor = chg.Cursor
			}
		}

		if len(changes) > 0 {
			c.setCursor(cursor)
			if c.opts.OnSync != nil {
				c.opts.OnSync(cursor)
			}
		}

		if !hasMore {
			break
		}
	}

	return nil
}

func (c *Client) applyChange(chg change) {
	switch chg.Op {
	case OpCreate, OpUpdate:
		c.store.set(chg.Entity, chg.ID, chg.Data)
	case OpDelete:
		c.store.delete(chg.Entity, chg.ID)
	}
}

func (c *Client) handleConflict() error {
	c.queue.clear()
	return c.handleCursorTooOld()
}

func (c *Client) handleCursorTooOld() error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	data, cursor, err := c.transport.snapshot(ctx, c.opts.Scope)
	if err != nil {
		return err
	}

	c.store.load(data)
	c.setCursor(cursor)

	if c.opts.OnSync != nil {
		c.opts.OnSync(cursor)
	}

	return nil
}

func (c *Client) setCursor(cursor uint64) {
	c.cursorMu.Lock()
	c.cursor = cursor
	c.cursorMu.Unlock()
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

// -----------------------------------------------------------------------------
// Entity and Collection
// -----------------------------------------------------------------------------

// Entity represents a single synchronized record with reactive access.
type Entity[T any] struct {
	id         string
	entityType string
	collection *Collection[T]
	pending    bool
}

// ID returns the entity's unique identifier.
func (e *Entity[T]) ID() string {
	return e.id
}

// Get returns the current value (reactive).
func (e *Entity[T]) Get() T {
	_ = e.collection.client.store.version.Get()

	data, ok := e.collection.client.store.get(e.entityType, e.id)
	if !ok {
		var zero T
		return zero
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		var zero T
		return zero
	}
	return value
}

// Set updates the entity value (queues mutation).
func (e *Entity[T]) Set(value T) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	e.collection.client.store.set(e.entityType, e.id, data)
	e.pending = true

	e.collection.client.Mutate(e.entityType+".update", map[string]any{
		"id":   e.id,
		"data": value,
	})
}

// Delete removes the entity (queues mutation).
func (e *Entity[T]) Delete() {
	e.collection.client.store.delete(e.entityType, e.id)
	e.pending = true

	e.collection.client.Mutate(e.entityType+".delete", map[string]any{
		"id": e.id,
	})
}

// Exists returns whether the entity exists in the store.
func (e *Entity[T]) Exists() bool {
	_ = e.collection.client.store.version.Get()
	return e.collection.client.store.has(e.entityType, e.id)
}

// Collection manages a set of entities of the same type.
type Collection[T any] struct {
	name     string
	client   *Client
	entities map[string]*Entity[T]
	mu       gosync.RWMutex
}

// collectionRef is a type-erased reference to a collection for internal use.
type collectionRef interface {
	collectionName() string
}

func (c *Collection[T]) collectionName() string {
	return c.name
}

// NewCollection creates a new collection bound to the client.
func NewCollection[T any](client *Client, name string) *Collection[T] {
	col := &Collection[T]{
		name:     name,
		client:   client,
		entities: make(map[string]*Entity[T]),
	}

	client.registerCollection(name, col)

	return col
}

// Get returns an entity by ID, creating a lazy reference if needed.
func (c *Collection[T]) Get(id string) *Entity[T] {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.entities[id]; ok {
		return e
	}

	e := &Entity[T]{
		id:         id,
		entityType: c.name,
		collection: c,
	}
	c.entities[id] = e
	return e
}

// Create creates a new entity with the given value.
func (c *Collection[T]) Create(id string, value T) *Entity[T] {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	c.client.store.set(c.name, id, data)

	c.client.Mutate(c.name+".create", map[string]any{
		"id":   id,
		"data": value,
	})

	e := c.Get(id)
	e.pending = true
	return e
}

// All returns all entities in the collection (reactive).
func (c *Collection[T]) All() []*Entity[T] {
	_ = c.client.store.version.Get()

	ids := c.client.store.list(c.name)
	entities := make([]*Entity[T], 0, len(ids))
	for _, id := range ids {
		entities = append(entities, c.Get(id))
	}
	return entities
}

// Count returns the number of entities in the collection (reactive).
func (c *Collection[T]) Count() int {
	_ = c.client.store.version.Get()
	return c.client.store.count(c.name)
}

// Find returns entities matching the predicate.
func (c *Collection[T]) Find(predicate func(T) bool) []*Entity[T] {
	_ = c.client.store.version.Get()

	all := c.client.store.all(c.name)
	var result []*Entity[T]

	for id, data := range all {
		var value T
		if err := json.Unmarshal(data, &value); err != nil {
			continue
		}
		if predicate(value) {
			result = append(result, c.Get(id))
		}
	}
	return result
}

// Has checks if an entity with the given ID exists.
func (c *Collection[T]) Has(id string) bool {
	_ = c.client.store.version.Get()
	return c.client.store.has(c.name, id)
}
