package sync

import (
	"sync"
	"sync/atomic"
)

// tracker is the global dependency tracker for reactive computations.
var tracker = newDependencyTracker()

// dependencyTracker tracks signal access during computation.
type dependencyTracker struct {
	mu    sync.Mutex
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
	mu          sync.RWMutex
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

// Version returns the current version number (for change detection).
func (s *Signal[T]) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

func (s *Signal[T]) subscribe(sub subscriber) {
	s.mu.Lock()
	s.subscribers[sub] = struct{}{}
	s.mu.Unlock()
}

func (s *Signal[T]) unsubscribe(sub subscriber) {
	s.mu.Lock()
	delete(s.subscribers, sub)
	s.mu.Unlock()
}

// computedBase is the non-generic base for dependency tracking.
type computedBase struct {
	dirty       atomic.Bool
	running     atomic.Bool
	mu          sync.RWMutex
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
	mu    sync.RWMutex
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

// effectTracker wraps an Effect for dependency tracking.
// It allows Effect to be tracked as a subscriber.
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
	mu      sync.Mutex
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
	// The signals have already subscribed the computedBase
	// We need to re-wire it to notify us
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

func (e *Effect) notify() {
	e.triggerRun()
}

// Batch executes a function while batching signal updates.
// Subscribers are notified only once after all changes are made.
func Batch(fn func()) {
	// For now, just execute directly
	// Future: implement actual batching with deferred notifications
	fn()
}
