// Package driver provides the vector database driver registry.
// Import specific driver packages to register them.
package driver

import (
	"fmt"
	"sort"
	"sync"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]vectorize.Driver)
)

// Register makes a driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver vectorize.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()

	if driver == nil {
		panic("vectorize: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("vectorize: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open opens a database specified by its driver name and data source name (DSN).
// The DSN format is driver-specific.
func Open(driverName, dsn string) (vectorize.DB, error) {
	driversMu.RLock()
	driver, ok := drivers[driverName]
	driversMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("vectorize: unknown driver %q (forgotten import?)", driverName)
	}
	return driver.Open(dsn)
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()

	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Unregister removes a driver from the registry.
// This is primarily useful for testing.
func Unregister(name string) {
	driversMu.Lock()
	defer driversMu.Unlock()
	delete(drivers, name)
}
