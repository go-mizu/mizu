package local

import (
	"fmt"
	"sync"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// Registry manages registered search engines.
type Registry struct {
	mu         sync.RWMutex
	engines    map[string]engines.Engine
	shortcuts  map[string]string
	byCategory map[engines.Category][]engines.Engine
}

// NewRegistry creates a new engine registry.
func NewRegistry() *Registry {
	return &Registry{
		engines:    make(map[string]engines.Engine),
		shortcuts:  make(map[string]string),
		byCategory: make(map[engines.Category][]engines.Engine),
	}
}

// Register registers an engine.
func (r *Registry) Register(engine engines.Engine) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := engine.Name()
	if _, exists := r.engines[name]; exists {
		return fmt.Errorf("engine %q already registered", name)
	}

	shortcut := engine.Shortcut()
	if shortcut != "" && shortcut != "-" {
		if existing, exists := r.shortcuts[shortcut]; exists {
			return fmt.Errorf("shortcut %q already used by engine %q", shortcut, existing)
		}
		r.shortcuts[shortcut] = name
	}

	r.engines[name] = engine

	// Index by category
	for _, cat := range engine.Categories() {
		r.byCategory[cat] = append(r.byCategory[cat], engine)
	}

	return nil
}

// Get returns an engine by name.
func (r *Registry) Get(name string) (engines.Engine, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	eng, ok := r.engines[name]
	return eng, ok
}

// GetByShortcut returns an engine by shortcut.
func (r *Registry) GetByShortcut(shortcut string) (engines.Engine, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok := r.shortcuts[shortcut]
	if !ok {
		return nil, false
	}
	eng, ok := r.engines[name]
	return eng, ok
}

// GetByCategory returns all engines for a category.
func (r *Registry) GetByCategory(category engines.Category) []engines.Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byCategory[category]
}

// All returns all registered engines.
func (r *Registry) All() []engines.Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]engines.Engine, 0, len(r.engines))
	for _, eng := range r.engines {
		result = append(result, eng)
	}
	return result
}

// Names returns all engine names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]string, 0, len(r.engines))
	for name := range r.engines {
		result = append(result, name)
	}
	return result
}

// Categories returns all categories with registered engines.
func (r *Registry) Categories() []engines.Category {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]engines.Category, 0, len(r.byCategory))
	for cat := range r.byCategory {
		result = append(result, cat)
	}
	return result
}

// EngineCount returns the number of registered engines.
func (r *Registry) EngineCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.engines)
}

// Unregister removes an engine from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	eng, ok := r.engines[name]
	if !ok {
		return fmt.Errorf("engine %q not found", name)
	}

	// Remove shortcut
	shortcut := eng.Shortcut()
	if shortcut != "" && shortcut != "-" {
		delete(r.shortcuts, shortcut)
	}

	// Remove from categories
	for _, cat := range eng.Categories() {
		engs := r.byCategory[cat]
		for i, e := range engs {
			if e.Name() == name {
				r.byCategory[cat] = append(engs[:i], engs[i+1:]...)
				break
			}
		}
	}

	delete(r.engines, name)
	return nil
}
