// Package vector defines minimal interfaces for vector indexing and search.
//
// The package is intentionally small. It mirrors the pkg/index style while
// leaving storage, similarity metrics, and commit/visibility semantics to
// concrete implementations.
package vector

import "context"

// Store represents a logical vector storage backend.
//
// A Store may manage multiple independent collections (namespaces).
// Examples:
//   - A single SQLite or DuckDB database containing multiple vector tables.
//   - A remote vector search cluster with multiple indexes.
//   - An embedded ANN engine with multiple logical indexes.
//
// Store implementations are responsible for connection management
// and lifecycle outside of this interface.
type Store interface {
	// Collection returns a handle to a named collection.
	//
	// The collection is logical and may be lazily created.
	// Repeated calls with the same name should return handles
	// operating on the same underlying collection.
	Collection(name string) Collection
}

// Collection represents a single logical vector index.
//
// A Collection combines vector indexing and searching capabilities.
// It corresponds to one table, one index, or one logical namespace
// depending on the backend.
type Collection interface {
	// Init prepares underlying structures if required.
	//
	// For SQL backends, this may create tables and vector indexes.
	// For embedded engines, this may open or initialize index files.
	// Init should be safe to call multiple times.
	Init(ctx context.Context) error

	Indexer
	Searcher
}

// Indexer indexes batches of vector items.
//
// Semantics:
//   - Index should be idempotent for the same ID where the backend supports upsert.
//   - If the backend is append only, it should replace older versions logically
//     (for example via delete then insert) or document the behavior.
//   - Dimension mismatches should return an error.
//   - Visibility guarantees (immediate vs delayed commit) are driver defined.
//   - Implementations should respect ctx for cancellation and timeouts.
type Indexer interface {
	Index(ctx context.Context, items []Item) error
}

// Searcher executes vector similarity queries.
//
// Semantics:
//   - Search performs nearest neighbor retrieval over Item.Vector.
//   - Similarity and scoring are driver defined (for example cosine, dot, L2).
//   - K is the maximum number of hits. If zero, driver defaults apply.
//   - Filters are optional. Backends may ignore unsupported filters.
//   - Total should represent the total matched items after filtering if the
//     backend can provide it. If not available, drivers may set Total equal
//     to len(Hits).
//   - Implementations should respect ctx for cancellation and timeouts.
type Searcher interface {
	Search(ctx context.Context, q Query) (Results, error)
}

// Item is a single vector item.
//
// Contract:
//   - ID must be stable and unique within a collection.
//   - Vector is the canonical embedding representation.
//   - Metadata is optional opaque data for filtering or retrieval.
//     Drivers may store it as JSON, a blob, or ignore it.
type Item struct {
	ID       string             `json:"id"`                 // required: unique identifier
	Vector   []float32          `json:"vector"`             // required: embedding vector
	Metadata map[string]string  `json:"metadata,omitempty"` // optional: driver defined
}

// Query is a vector search query.
//
// Contract:
//   - Vector is the query embedding.
//   - K is the requested number of nearest neighbors. If zero, driver defaults apply.
//   - Filter is an optional set of equality constraints over Metadata keys.
//     Drivers may support a richer filter language via driver specific extensions.
type Query struct {
	Vector []float32          `json:"vector"`          // required: query embedding
	K      int                `json:"k,omitempty"`     // max number of hits
	Filter map[string]string  `json:"filter,omitempty"` // optional: equality constraints
}

// Results is the search output.
//
// Contract:
//   - Hits contains the nearest neighbors in descending relevance.
//   - Total is the total number of matched items if available.
type Results struct {
	Hits  []Hit `json:"hits,omitempty"`
	Total int   `json:"total,omitempty"`
}

// Hit represents a single nearest neighbor.
//
// Contract:
//   - ID must match the original Item.ID.
//   - Score is backend defined. Higher means more similar.
//   - Distance is optional. Drivers may leave Distance at zero if unsupported
//     or if Score already represents the distance.
type Hit struct {
	ID       string  `json:"id"`                  // required: matched item id
	Score    float64 `json:"score,omitempty"`     // similarity score
	Distance float64 `json:"distance,omitempty"`  // optional distance
}