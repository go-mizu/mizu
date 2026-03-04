package vector

import (
	"fmt"
	"sort"
	"sync"
)

// Config configures a vector store driver.
type Config struct {
	Addr    string            // optional endpoint or DSN
	DataDir string            // optional local directory for drivers that need it
	Options map[string]string // optional driver-specific options
}

// Closer is implemented by stores that need lifecycle management.
type Closer interface {
	Close() error
}

// AddrSetter is implemented by external stores that can override endpoint address.
type AddrSetter interface {
	SetAddr(addr string)
}

// BaseExternal provides common address handling for external drivers.
type BaseExternal struct {
	Addr string
}

// SetAddr stores the connection address.
func (b *BaseExternal) SetAddr(addr string) { b.Addr = addr }

// EffectiveAddr returns Addr if set, otherwise returns def.
func (b *BaseExternal) EffectiveAddr(def string) string {
	if b.Addr != "" {
		return b.Addr
	}
	return def
}

// StoreFactory creates a store using a driver config.
type StoreFactory func(cfg Config) (Store, error)

var (
	storeRegistry   = make(map[string]StoreFactory)
	storeRegistryMu sync.RWMutex
)

// Register registers a vector store driver by name.
func Register(name string, factory StoreFactory) {
	storeRegistryMu.Lock()
	defer storeRegistryMu.Unlock()
	if _, exists := storeRegistry[name]; exists {
		panic(fmt.Sprintf("vector: driver %q already registered", name))
	}
	storeRegistry[name] = factory
}

// Open creates a store by registered driver name.
func Open(name string, cfg Config) (Store, error) {
	storeRegistryMu.RLock()
	factory, ok := storeRegistry[name]
	storeRegistryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("vector: unknown driver %q (available: %v)", name, List())
	}
	return factory(cfg)
}

// List returns all registered driver names in sorted order.
func List() []string {
	storeRegistryMu.RLock()
	defer storeRegistryMu.RUnlock()
	names := make([]string, 0, len(storeRegistry))
	for name := range storeRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
