package gateway

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
)

// memoryRegistry manages per-agent MemoryManager instances.
// Each agent with a workspace directory gets its own memory index.
type memoryRegistry struct {
	mu       sync.RWMutex
	managers map[string]*memory.MemoryManager // keyed by workspace dir
}

func newMemoryRegistry() *memoryRegistry {
	return &memoryRegistry{
		managers: make(map[string]*memory.MemoryManager),
	}
}

// get returns the MemoryManager for the given workspace directory.
// On first access it creates the manager, ensures the .memory directory,
// and indexes the workspace. Returns nil if workspaceDir is empty.
func (r *memoryRegistry) get(workspaceDir string) (*memory.MemoryManager, error) {
	if workspaceDir == "" {
		return nil, nil
	}

	r.mu.RLock()
	if m, ok := r.managers[workspaceDir]; ok {
		r.mu.RUnlock()
		return m, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if m, ok := r.managers[workspaceDir]; ok {
		return m, nil
	}

	// Ensure the workspace directory exists.
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Create .memory directory inside workspace.
	memDir := filepath.Join(workspaceDir, ".memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		return nil, fmt.Errorf("create memory dir: %w", err)
	}

	dbPath := filepath.Join(memDir, "index.db")
	cfg := memory.DefaultMemoryConfig()
	cfg.WorkspaceDir = workspaceDir

	m, err := memory.NewMemoryManager(dbPath, workspaceDir, cfg)
	if err != nil {
		return nil, fmt.Errorf("create memory manager for %s: %w", workspaceDir, err)
	}

	// Ensure today's daily memory log exists.
	if err := m.EnsureDailyLog(); err != nil {
		fmt.Fprintf(os.Stderr, "memory: daily log error for %s: %v\n", workspaceDir, err)
	}

	// Index workspace files in background-safe manner.
	// Errors are logged but don't prevent the manager from being usable.
	if err := m.IndexAll(); err != nil {
		// Log but continue — FTS search will still work for already-indexed files.
		fmt.Fprintf(os.Stderr, "memory: index error for %s: %v\n", workspaceDir, err)
	}

	r.managers[workspaceDir] = m
	return m, nil
}

// closeAll releases all memory managers. Called on gateway shutdown.
func (r *memoryRegistry) closeAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for dir, m := range r.managers {
		m.Close()
		delete(r.managers, dir)
	}
}
