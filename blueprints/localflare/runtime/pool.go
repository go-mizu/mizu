package runtime

import (
	"context"
	"net/http"
	"sync"

	"github.com/go-mizu/blueprints/localflare/store"
)

// Pool manages a pool of Runtime instances for concurrent Worker execution.
// This provides better performance than creating a new runtime for each request.
type Pool struct {
	store    store.Store
	pool     chan *Runtime
	size     int
	mu       sync.Mutex
	scripts  map[string]string // cached compiled scripts
}

// PoolConfig configures the runtime pool.
type PoolConfig struct {
	Store    store.Store
	PoolSize int // Number of runtime instances to keep in pool
}

// NewPool creates a new runtime pool.
func NewPool(cfg PoolConfig) *Pool {
	size := cfg.PoolSize
	if size <= 0 {
		size = 10 // Default pool size
	}

	p := &Pool{
		store:   cfg.Store,
		pool:    make(chan *Runtime, size),
		size:    size,
		scripts: make(map[string]string),
	}

	// Pre-warm the pool
	for i := 0; i < size; i++ {
		rt := New(Config{Store: cfg.Store})
		p.pool <- rt
	}

	return p
}

// Execute runs a Worker script with the given request.
func (p *Pool) Execute(ctx context.Context, scriptID string, script string, req *http.Request, bindings map[string]string) (*WorkerResponse, error) {
	// Get a runtime from the pool
	rt := p.acquire()
	defer p.release(rt)

	// Setup bindings for this execution
	rt.setupBindings(bindings)

	return rt.Execute(ctx, script, req)
}

// ExecuteCached executes a cached script (for repeated executions).
func (p *Pool) ExecuteCached(ctx context.Context, scriptID string, req *http.Request, bindings map[string]string) (*WorkerResponse, error) {
	p.mu.Lock()
	script, ok := p.scripts[scriptID]
	p.mu.Unlock()

	if !ok {
		return nil, &ScriptNotFoundError{ScriptID: scriptID}
	}

	return p.Execute(ctx, scriptID, script, req, bindings)
}

// CacheScript caches a script for later execution.
func (p *Pool) CacheScript(scriptID, script string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scripts[scriptID] = script
}

// InvalidateScript removes a script from the cache.
func (p *Pool) InvalidateScript(scriptID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.scripts, scriptID)
}

// acquire gets a runtime from the pool.
func (p *Pool) acquire() *Runtime {
	select {
	case rt := <-p.pool:
		return rt
	default:
		// Pool exhausted, create a new runtime
		return New(Config{Store: p.store})
	}
}

// release returns a runtime to the pool.
func (p *Pool) release(rt *Runtime) {
	select {
	case p.pool <- rt:
		// Returned to pool
	default:
		// Pool full, close the runtime
		rt.Close()
	}
}

// Close shuts down all runtimes in the pool.
func (p *Pool) Close() {
	close(p.pool)
	for rt := range p.pool {
		rt.Close()
	}
}

// Stats returns pool statistics.
func (p *Pool) Stats() PoolStats {
	return PoolStats{
		PoolSize:     p.size,
		Available:    len(p.pool),
		CachedScripts: len(p.scripts),
	}
}

// PoolStats contains pool statistics.
type PoolStats struct {
	PoolSize      int `json:"pool_size"`
	Available     int `json:"available"`
	CachedScripts int `json:"cached_scripts"`
}

// ScriptNotFoundError indicates a script was not found in cache.
type ScriptNotFoundError struct {
	ScriptID string
}

func (e *ScriptNotFoundError) Error() string {
	return "script not found: " + e.ScriptID
}

// Executor provides a high-level interface for Worker execution.
type Executor struct {
	pool  *Pool
	store store.Store
}

// NewExecutor creates a new Worker executor.
func NewExecutor(store store.Store, poolSize int) *Executor {
	return &Executor{
		pool: NewPool(PoolConfig{
			Store:    store,
			PoolSize: poolSize,
		}),
		store: store,
	}
}

// ExecuteWorker executes a Worker by ID.
func (e *Executor) ExecuteWorker(ctx context.Context, workerID string, req *http.Request) (*WorkerResponse, error) {
	// Get worker from store
	worker, err := e.store.Workers().GetByID(ctx, workerID)
	if err != nil {
		return nil, err
	}

	// Build bindings map
	bindings := worker.Bindings

	return e.pool.Execute(ctx, workerID, worker.Script, req, bindings)
}

// ExecuteWorkerByName executes a Worker by name.
func (e *Executor) ExecuteWorkerByName(ctx context.Context, workerName string, req *http.Request) (*WorkerResponse, error) {
	// Get worker from store
	worker, err := e.store.Workers().GetByName(ctx, workerName)
	if err != nil {
		return nil, err
	}

	// Build bindings map
	bindings := worker.Bindings

	return e.pool.Execute(ctx, worker.ID, worker.Script, req, bindings)
}

// ExecuteScript executes a raw script.
func (e *Executor) ExecuteScript(ctx context.Context, script string, req *http.Request, bindings map[string]string) (*WorkerResponse, error) {
	return e.pool.Execute(ctx, "", script, req, bindings)
}

// Close shuts down the executor.
func (e *Executor) Close() {
	e.pool.Close()
}

// Stats returns executor statistics.
func (e *Executor) Stats() PoolStats {
	return e.pool.Stats()
}
