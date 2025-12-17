package sync

import (
	"encoding/json"
	"sync"
)

// Entity represents a single synchronized record with reactive access.
type Entity[T any] struct {
	id         string
	entityType string
	collection *Collection[T]
	pending    bool // True if local changes are pending sync
}

// ID returns the entity's unique identifier.
func (e *Entity[T]) ID() string {
	return e.id
}

// Get returns the current value (reactive).
func (e *Entity[T]) Get() T {
	// Access store version to register dependency
	_ = e.collection.client.store.Version().Get()

	data, ok := e.collection.client.store.Get(e.entityType, e.id)
	if !ok {
		var zero T
		return zero
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		var zero T
		return zero
	}
	return value
}

// Set updates the entity value (queues mutation).
func (e *Entity[T]) Set(value T) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	// Apply optimistically
	e.collection.client.store.Set(e.entityType, e.id, data)
	e.pending = true

	// Queue mutation
	e.collection.client.Mutate(e.entityType+".update", map[string]any{
		"id":   e.id,
		"data": value,
	})
}

// Delete removes the entity (queues mutation).
func (e *Entity[T]) Delete() {
	// Apply optimistically
	e.collection.client.store.Delete(e.entityType, e.id)
	e.pending = true

	// Queue mutation
	e.collection.client.Mutate(e.entityType+".delete", map[string]any{
		"id": e.id,
	})
}

// Exists returns whether the entity exists in the store.
func (e *Entity[T]) Exists() bool {
	// Access store version to register dependency
	_ = e.collection.client.store.Version().Get()
	return e.collection.client.store.Has(e.entityType, e.id)
}

// IsPending returns whether local changes are pending sync.
func (e *Entity[T]) IsPending() bool {
	return e.pending
}

// Collection manages a set of entities of the same type.
type Collection[T any] struct {
	name     string
	client   *Client
	entities map[string]*Entity[T]
	mu       sync.RWMutex
}

// CollectionOption configures a collection.
type CollectionOption func(*collectionConfig)

type collectionConfig struct {
	// Future options
}

// NewCollection creates a new collection bound to the client.
func NewCollection[T any](client *Client, name string, opts ...CollectionOption) *Collection[T] {
	col := &Collection[T]{
		name:     name,
		client:   client,
		entities: make(map[string]*Entity[T]),
	}

	// Apply options
	cfg := &collectionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Register with client
	client.registerCollection(name, col)

	return col
}

// Get returns an entity by ID, creating a lazy reference if needed.
func (c *Collection[T]) Get(id string) *Entity[T] {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.entities[id]; ok {
		return e
	}

	e := &Entity[T]{
		id:         id,
		entityType: c.name,
		collection: c,
	}
	c.entities[id] = e
	return e
}

// Create creates a new entity with the given value.
func (c *Collection[T]) Create(id string, value T) *Entity[T] {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	// Apply optimistically
	c.client.store.Set(c.name, id, data)

	// Queue mutation
	c.client.Mutate(c.name+".create", map[string]any{
		"id":   id,
		"data": value,
	})

	// Get or create entity reference
	e := c.Get(id)
	e.pending = true
	return e
}

// All returns all entities in the collection (reactive).
func (c *Collection[T]) All() []*Entity[T] {
	// Access store version to register dependency
	_ = c.client.store.Version().Get()

	ids := c.client.store.List(c.name)
	entities := make([]*Entity[T], 0, len(ids))
	for _, id := range ids {
		entities = append(entities, c.Get(id))
	}
	return entities
}

// Count returns the number of entities in the collection (reactive).
func (c *Collection[T]) Count() int {
	// Access store version to register dependency
	_ = c.client.store.Version().Get()
	return c.client.store.Count(c.name)
}

// Find returns entities matching the predicate.
func (c *Collection[T]) Find(predicate func(T) bool) []*Entity[T] {
	// Access store version to register dependency
	_ = c.client.store.Version().Get()

	all := c.client.store.All(c.name)
	var result []*Entity[T]

	for id, data := range all {
		var value T
		if err := json.Unmarshal(data, &value); err != nil {
			continue
		}
		if predicate(value) {
			result = append(result, c.Get(id))
		}
	}
	return result
}

// First returns the first entity matching the predicate, or nil.
func (c *Collection[T]) First(predicate func(T) bool) *Entity[T] {
	// Access store version to register dependency
	_ = c.client.store.Version().Get()

	all := c.client.store.All(c.name)

	for id, data := range all {
		var value T
		if err := json.Unmarshal(data, &value); err != nil {
			continue
		}
		if predicate(value) {
			return c.Get(id)
		}
	}
	return nil
}

// IDs returns all entity IDs in the collection.
func (c *Collection[T]) IDs() []string {
	// Access store version to register dependency
	_ = c.client.store.Version().Get()
	return c.client.store.List(c.name)
}

// Has checks if an entity with the given ID exists.
func (c *Collection[T]) Has(id string) bool {
	// Access store version to register dependency
	_ = c.client.store.Version().Get()
	return c.client.store.Has(c.name, id)
}

// Clear removes all entities from the collection.
func (c *Collection[T]) Clear() {
	ids := c.client.store.List(c.name)
	for _, id := range ids {
		c.client.store.Delete(c.name, id)
		c.client.Mutate(c.name+".delete", map[string]any{
			"id": id,
		})
	}
}

// Name returns the collection name.
func (c *Collection[T]) Name() string {
	return c.name
}

// collectionRef is a type-erased reference to a collection for internal use.
type collectionRef interface {
	Name() string
}
