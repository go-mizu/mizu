// Package bulkhead provides bulkhead isolation pattern middleware for Mizu.
package bulkhead

import (
	"net/http"
	"sync"

	"github.com/go-mizu/mizu"
)

// Options configures the bulkhead middleware.
type Options struct {
	// Name is the bulkhead name.
	Name string

	// MaxConcurrent is the maximum concurrent requests.
	// Default: 10.
	MaxConcurrent int

	// MaxWait is the maximum requests waiting.
	// Default: 10.
	MaxWait int

	// ErrorHandler handles rejected requests.
	ErrorHandler func(c *mizu.Ctx) error
}

// Bulkhead manages a single isolation compartment.
type Bulkhead struct {
	name    string
	sem     chan struct{}
	waiting int
	maxWait int
	mu      sync.Mutex
	opts    Options
}

// NewBulkhead creates a new bulkhead.
func NewBulkhead(opts Options) *Bulkhead {
	// Use defaults if not specified (0 or negative)
	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = 10
	}
	if opts.MaxWait <= 0 {
		opts.MaxWait = 10
	}

	return &Bulkhead{
		name:    opts.Name,
		sem:     make(chan struct{}, opts.MaxConcurrent),
		maxWait: opts.MaxWait,
		opts:    opts,
	}
}

// Middleware returns the bulkhead middleware.
func (b *Bulkhead) Middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Try non-blocking acquire
			select {
			case b.sem <- struct{}{}:
				defer func() { <-b.sem }()
				return next(c)
			default:
				// Check if we can wait
				b.mu.Lock()
				if b.waiting >= b.maxWait {
					b.mu.Unlock()
					return b.handleRejection(c)
				}
				b.waiting++
				b.mu.Unlock()

				// Block waiting for slot
				select {
				case b.sem <- struct{}{}:
					b.mu.Lock()
					b.waiting--
					b.mu.Unlock()

					defer func() { <-b.sem }()
					return next(c)
				case <-c.Context().Done():
					b.mu.Lock()
					b.waiting--
					b.mu.Unlock()
					return c.Context().Err()
				}
			}
		}
	}
}

func (b *Bulkhead) handleRejection(c *mizu.Ctx) error {
	if b.opts.ErrorHandler != nil {
		return b.opts.ErrorHandler(c)
	}
	return c.Text(http.StatusServiceUnavailable, "Bulkhead full")
}

// Stats returns current bulkhead statistics.
func (b *Bulkhead) Stats() Stats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return Stats{
		Name:       b.name,
		Active:     len(b.sem),
		MaxActive:  cap(b.sem),
		Waiting:    b.waiting,
		MaxWaiting: b.maxWait,
		Available:  cap(b.sem) - len(b.sem),
	}
}

// Stats contains bulkhead statistics.
type Stats struct {
	Name       string
	Active     int
	MaxActive  int
	Waiting    int
	MaxWaiting int
	Available  int
}

// New creates bulkhead middleware with options.
func New(opts Options) mizu.Middleware {
	b := NewBulkhead(opts)
	return b.Middleware()
}

// Manager manages multiple bulkheads.
type Manager struct {
	bulkheads map[string]*Bulkhead
	mu        sync.RWMutex
}

// NewManager creates a new bulkhead manager.
func NewManager() *Manager {
	return &Manager{
		bulkheads: make(map[string]*Bulkhead),
	}
}

// Get gets or creates a bulkhead.
func (m *Manager) Get(name string, maxConcurrent, maxWait int) *Bulkhead {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b, ok := m.bulkheads[name]; ok {
		return b
	}

	b := NewBulkhead(Options{
		Name:          name,
		MaxConcurrent: maxConcurrent,
		MaxWait:       maxWait,
	})
	m.bulkheads[name] = b
	return b
}

// Stats returns stats for all bulkheads.
func (m *Manager) Stats() map[string]Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]Stats)
	for name, b := range m.bulkheads {
		stats[name] = b.Stats()
	}
	return stats
}

// ForPath creates middleware that uses path-based bulkheads.
func ForPath(manager *Manager, maxConcurrent, maxWait int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path
			b := manager.Get(path, maxConcurrent, maxWait)
			return b.Middleware()(next)(c)
		}
	}
}
