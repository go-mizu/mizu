package index

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Engine is a pluggable FTS backend with lifecycle management.
type Engine interface {
	Name() string
	Open(ctx context.Context, dir string) error
	Close() error
	Stats(ctx context.Context) (EngineStats, error)
	Index(ctx context.Context, docs []Document) error
	Search(ctx context.Context, q Query) (Results, error)
}

// EngineStats reports index metadata.
type EngineStats struct {
	DocCount  int64
	DiskBytes int64
}

// --- registry ---

type EngineFactory func() Engine

var (
	registry   = make(map[string]EngineFactory)
	registryMu sync.RWMutex
)

func Register(name string, factory EngineFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("index: driver %q already registered", name))
	}
	registry[name] = factory
}

func NewEngine(name string) (Engine, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("index: unknown driver %q (available: %v)", name, List())
	}
	return factory(), nil
}

func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
