package index

import "context"

// Store represents a logical index storage backend.
//
// A Store may manage multiple independent collections (namespaces).
// Examples:
//   - A single SQLite or DuckDB database containing multiple FTS tables.
//   - A remote search cluster with multiple indexes.
//   - An embedded engine with multiple logical indexes.
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

// Collection represents a single logical full text index.
//
// A Collection combines indexing and searching capabilities.
// It corresponds to one FTS table, one search index, or one
// logical namespace depending on the backend.
type Collection interface {
	// Init prepares underlying structures if required.
	//
	// For SQL backends, this may create tables and FTS indexes.
	// For embedded engines, this may open or initialize index files.
	// Init should be safe to call multiple times.
	Init(ctx context.Context) error

	Indexer
	Searcher
}

// Indexer indexes batches of full text documents.
//
// Semantics:
//   - Index should be idempotent for the same DocID where the backend supports upsert.
//   - If the backend is append only, it should replace older versions logically
//     (for example via delete then insert) or document the behavior.
//   - Visibility guarantees (immediate vs delayed commit) are driver defined.
//   - Implementations should respect ctx for cancellation and timeouts.
type Indexer interface {
	Index(ctx context.Context, docs []Document) error
}

// Searcher executes full text queries.
//
// Semantics:
//   - Search performs a full text match over Document.Text.
//   - Ranking and scoring strategy are driver defined.
//   - Limit and Offset must be honored when supported by the backend.
//   - Total should represent the total matched documents ignoring pagination
//     when the backend can provide it. If not available, drivers may set Total
//     equal to len(Hits).
//   - Implementations should respect ctx for cancellation and timeouts.
type Searcher interface {
	Search(ctx context.Context, q Query) (Results, error)
}

// Document is a single full text document.
//
// Contract:
//   - DocID must be stable and unique within an index.
//   - Text is the canonical searchable content.
//   - No structured fields are assumed; drivers should map this to their
//     native schema (for example: a single TEXT column with an FTS index,
//     or a document body field in an embedded engine).
type Document struct {
	DocID string `json:"doc_id"` // required: unique identifier
	Text  string `json:"text"`   // required: full text content
}

// Query is a full text query.
//
// Contract:
//   - Text is interpreted using the backend's query syntax
//     (for example: MATCH syntax in SQL FTS, query parser in embedded engines).
//   - Limit and Offset implement pagination. If zero, driver defaults apply.
type Query struct {
	Text   string `json:"text"`             // required: query expression
	Limit  int    `json:"limit,omitempty"`  // max number of hits
	Offset int    `json:"offset,omitempty"` // pagination offset
}

// Results is the search output.
//
// Contract:
//   - Hits contains the current page of results.
//   - Total is the total number of matched documents if available.
type Results struct {
	Hits  []Hit `json:"hits,omitempty"`
	Total int   `json:"total,omitempty"`
}

// Hit represents a single matched document.
//
// Contract:
//   - DocID must match the original Document.DocID.
//   - Score is backend defined. Higher means more relevant.
//   - Snippet is an optional highlighted fragment derived from Text.
//     Drivers may leave Snippet empty if unsupported.
type Hit struct {
	DocID   string  `json:"doc_id"`            // required: matched document id
	Score   float64 `json:"score,omitempty"`   // relevance score
	Snippet string  `json:"snippet,omitempty"` // optional highlight fragment
}
