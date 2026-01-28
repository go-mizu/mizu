package fineweb

import (
	"fmt"
	"sort"
	"sync"
)

// DriverFactory creates a driver instance with the given config.
type DriverFactory func(cfg DriverConfig) (Driver, error)

var (
	registry   = make(map[string]DriverFactory)
	registryMu sync.RWMutex
)

// Register adds a driver factory to the registry.
// It panics if a driver with the same name is already registered.
func Register(name string, factory DriverFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("fineweb: driver %q already registered", name))
	}
	registry[name] = factory
}

// Open creates a driver by name using the registered factory.
func Open(name string, cfg DriverConfig) (Driver, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("fineweb: unknown driver %q (available: %v)", name, List())
	}

	return factory(cfg)
}

// List returns all registered driver names in sorted order.
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

// IsRegistered checks if a driver is registered.
func IsRegistered(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	_, ok := registry[name]
	return ok
}

// MustOpen creates a driver by name, panicking on error.
// Useful for initialization where failure should be fatal.
func MustOpen(name string, cfg DriverConfig) Driver {
	d, err := Open(name, cfg)
	if err != nil {
		panic(err)
	}
	return d
}

// OpenAll opens all registered drivers with the given config.
// Returns a map of driver name to driver (or error).
func OpenAll(cfg DriverConfig) map[string]DriverOrError {
	names := List()
	results := make(map[string]DriverOrError, len(names))

	for _, name := range names {
		d, err := Open(name, cfg)
		results[name] = DriverOrError{Driver: d, Error: err}
	}

	return results
}

// DriverOrError holds either a driver or an error.
type DriverOrError struct {
	Driver Driver
	Error  error
}

// CloseAll closes multiple drivers, collecting any errors.
func CloseAll(drivers ...Driver) error {
	var errs []error
	for _, d := range drivers {
		if d != nil {
			if err := d.Close(); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", d.Name(), err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing drivers: %v", errs)
	}
	return nil
}
