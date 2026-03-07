package embed

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Config holds driver configuration.
type Config struct {
	Dir       string // local directory for model files / cache
	Addr      string // server address (remote drivers like llamacpp)
	Model     string // model name or path override
	BatchSize int    // max inputs per Embed call (0 = driver default)
}

// Driver extends Model with lifecycle management.
//
// A Driver wraps a Model with initialization and cleanup.
// Drivers are created via the registry and must be Open'd before use.
type Driver interface {
	Model
	Open(ctx context.Context, cfg Config) error
	Close() error
}

// --- registry ---

// Factory creates a new uninitialized Driver.
type Factory func() Driver

var (
	registry   = make(map[string]Factory)
	registryMu sync.RWMutex
)

// Register adds a named driver factory to the global registry.
// Panics if a driver with the same name is already registered.
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("embed: driver %q already registered", name))
	}
	registry[name] = factory
}

// New creates a new uninitialized driver by name.
func New(name string) (Driver, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("embed: unknown driver %q (available: %v)", name, List())
	}
	return factory(), nil
}

// List returns sorted names of all registered drivers.
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
