package metastore

import (
	"fmt"
	"sort"
	"sync"
)

// Driver opens a concrete Store for a DSN.
type Driver interface {
	Open(dsn string, opts Options) (Store, error)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Driver)
)

// Register registers a metastore driver by name and panics on duplicates.
func Register(name string, d Driver) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("metastore: driver %q already registered", name))
	}
	registry[name] = d
}

// Open opens a store by driver name and DSN.
func Open(name, dsn string, opts Options) (Store, error) {
	registryMu.RLock()
	driver, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("metastore: unknown driver %q (available: %v)", name, List())
	}
	return driver.Open(dsn, opts)
}

// List returns sorted registered metastore drivers.
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
