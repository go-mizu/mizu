package indexer

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// Service implements the indexer API.
type Service struct {
	store Store
}

// NewService creates a new indexer service.
func NewService(s Store) *Service {
	return &Service{store: s}
}

// IndexDocument indexes a single document.
func (s *Service) IndexDocument(ctx context.Context, doc *store.Document) error {
	return s.store.Index().IndexDocument(ctx, doc)
}

// UpdateDocument updates an existing document.
func (s *Service) UpdateDocument(ctx context.Context, doc *store.Document) error {
	return s.store.Index().UpdateDocument(ctx, doc)
}

// DeleteDocument removes a document from the index.
func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	return s.store.Index().DeleteDocument(ctx, id)
}

// GetDocument retrieves a document by ID.
func (s *Service) GetDocument(ctx context.Context, id string) (*store.Document, error) {
	return s.store.Index().GetDocument(ctx, id)
}

// BulkIndex indexes multiple documents at once.
func (s *Service) BulkIndex(ctx context.Context, docs []*store.Document) error {
	return s.store.Index().BulkIndex(ctx, docs)
}

// GetStats returns index statistics.
func (s *Service) GetStats(ctx context.Context) (*store.IndexStats, error) {
	return s.store.Index().GetIndexStats(ctx)
}

// RebuildIndex rebuilds the entire search index.
func (s *Service) RebuildIndex(ctx context.Context) error {
	return s.store.Index().RebuildIndex(ctx)
}
