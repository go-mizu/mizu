// Package indexer provides document indexing functionality.
package indexer

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// API defines the indexer service contract.
type API interface {
	// IndexDocument indexes a single document.
	IndexDocument(ctx context.Context, doc *store.Document) error

	// UpdateDocument updates an existing document.
	UpdateDocument(ctx context.Context, doc *store.Document) error

	// DeleteDocument removes a document from the index.
	DeleteDocument(ctx context.Context, id string) error

	// GetDocument retrieves a document by ID.
	GetDocument(ctx context.Context, id string) (*store.Document, error)

	// BulkIndex indexes multiple documents at once.
	BulkIndex(ctx context.Context, docs []*store.Document) error

	// GetStats returns index statistics.
	GetStats(ctx context.Context) (*store.IndexStats, error)

	// RebuildIndex rebuilds the entire search index.
	RebuildIndex(ctx context.Context) error
}

// Store defines the data access contract for indexing.
type Store interface {
	Index() store.IndexStore
}
