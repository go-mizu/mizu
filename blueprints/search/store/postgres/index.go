package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// IndexStore implements store.IndexStore using PostgreSQL.
type IndexStore struct {
	db *sql.DB
}

// IndexDocument indexes a new document.
func (s *IndexStore) IndexDocument(ctx context.Context, doc *store.Document) error {
	// Extract domain from URL
	if doc.Domain == "" {
		u, err := url.Parse(doc.URL)
		if err == nil {
			doc.Domain = u.Host
		}
	}

	// Count words
	if doc.WordCount == 0 {
		doc.WordCount = len(strings.Fields(doc.Content))
	}

	// Set timestamps
	now := time.Now()
	if doc.CrawledAt.IsZero() {
		doc.CrawledAt = now
	}
	doc.UpdatedAt = now

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO search.documents (
			url, title, description, content, domain, language, content_type,
			favicon, word_count, crawled_at, updated_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (url) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			favicon = EXCLUDED.favicon,
			word_count = EXCLUDED.word_count,
			updated_at = EXCLUDED.updated_at,
			metadata = EXCLUDED.metadata
	`,
		doc.URL, doc.Title, doc.Description, doc.Content, doc.Domain,
		doc.Language, doc.ContentType, doc.Favicon, doc.WordCount,
		doc.CrawledAt, doc.UpdatedAt, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	return nil
}

// UpdateDocument updates an existing document.
func (s *IndexStore) UpdateDocument(ctx context.Context, doc *store.Document) error {
	doc.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE search.documents SET
			title = $2,
			description = $3,
			content = $4,
			favicon = $5,
			word_count = $6,
			updated_at = $7,
			metadata = $8
		WHERE id = $1
	`,
		doc.ID, doc.Title, doc.Description, doc.Content,
		doc.Favicon, doc.WordCount, doc.UpdatedAt, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// DeleteDocument deletes a document.
func (s *IndexStore) DeleteDocument(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM search.documents WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// GetDocument retrieves a document by ID.
func (s *IndexStore) GetDocument(ctx context.Context, id string) (*store.Document, error) {
	var doc store.Document
	var metadataJSON []byte
	var description, favicon sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, url, title, description, content, domain, language,
			content_type, favicon, word_count, crawled_at, updated_at, metadata
		FROM search.documents WHERE id = $1
	`, id).Scan(
		&doc.ID, &doc.URL, &doc.Title, &description, &doc.Content,
		&doc.Domain, &doc.Language, &doc.ContentType, &favicon,
		&doc.WordCount, &doc.CrawledAt, &doc.UpdatedAt, &metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if description.Valid {
		doc.Description = description.String
	}
	if favicon.Valid {
		doc.Favicon = favicon.String
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &doc.Metadata)
	}

	return &doc, nil
}

// GetDocumentByURL retrieves a document by URL.
func (s *IndexStore) GetDocumentByURL(ctx context.Context, url string) (*store.Document, error) {
	var doc store.Document
	var metadataJSON []byte
	var description, favicon sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, url, title, description, content, domain, language,
			content_type, favicon, word_count, crawled_at, updated_at, metadata
		FROM search.documents WHERE url = $1
	`, url).Scan(
		&doc.ID, &doc.URL, &doc.Title, &description, &doc.Content,
		&doc.Domain, &doc.Language, &doc.ContentType, &favicon,
		&doc.WordCount, &doc.CrawledAt, &doc.UpdatedAt, &metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if description.Valid {
		doc.Description = description.String
	}
	if favicon.Valid {
		doc.Favicon = favicon.String
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &doc.Metadata)
	}

	return &doc, nil
}

// ListDocuments lists documents with pagination.
func (s *IndexStore) ListDocuments(ctx context.Context, limit, offset int) ([]*store.Document, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, url, title, description, domain, language, content_type,
			favicon, word_count, crawled_at, updated_at
		FROM search.documents
		ORDER BY crawled_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	var docs []*store.Document
	for rows.Next() {
		var doc store.Document
		var description, favicon sql.NullString
		if err := rows.Scan(
			&doc.ID, &doc.URL, &doc.Title, &description, &doc.Domain,
			&doc.Language, &doc.ContentType, &favicon, &doc.WordCount,
			&doc.CrawledAt, &doc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		if description.Valid {
			doc.Description = description.String
		}
		if favicon.Valid {
			doc.Favicon = favicon.String
		}
		docs = append(docs, &doc)
	}

	return docs, nil
}

// BulkIndex indexes multiple documents.
func (s *IndexStore) BulkIndex(ctx context.Context, docs []*store.Document) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO search.documents (
			url, title, description, content, domain, language, content_type,
			favicon, word_count, crawled_at, updated_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (url) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			favicon = EXCLUDED.favicon,
			word_count = EXCLUDED.word_count,
			updated_at = EXCLUDED.updated_at,
			metadata = EXCLUDED.metadata
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, doc := range docs {
		if doc.Domain == "" {
			u, err := url.Parse(doc.URL)
			if err == nil {
				doc.Domain = u.Host
			}
		}
		if doc.WordCount == 0 {
			doc.WordCount = len(strings.Fields(doc.Content))
		}
		if doc.CrawledAt.IsZero() {
			doc.CrawledAt = now
		}
		doc.UpdatedAt = now

		metadataJSON, _ := json.Marshal(doc.Metadata)

		_, err := stmt.ExecContext(ctx,
			doc.URL, doc.Title, doc.Description, doc.Content, doc.Domain,
			doc.Language, doc.ContentType, doc.Favicon, doc.WordCount,
			doc.CrawledAt, doc.UpdatedAt, metadataJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to index document %s: %w", doc.URL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetIndexStats returns index statistics.
func (s *IndexStore) GetIndexStats(ctx context.Context) (*store.IndexStats, error) {
	stats := &store.IndexStats{
		Languages:    make(map[string]int),
		ContentTypes: make(map[string]int),
	}

	// Total documents
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM search.documents").Scan(&stats.TotalDocuments)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	// Total size (estimate)
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(pg_column_size(content)), 0) FROM search.documents
	`).Scan(&stats.TotalSize)
	if err != nil {
		stats.TotalSize = 0
	}

	// Last updated
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(updated_at), NOW()) FROM search.documents
	`).Scan(&stats.LastUpdated)
	if err != nil {
		stats.LastUpdated = time.Now()
	}

	// Languages
	rows, err := s.db.QueryContext(ctx, `
		SELECT language, COUNT(*) FROM search.documents GROUP BY language
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var lang string
			var count int
			if err := rows.Scan(&lang, &count); err == nil {
				stats.Languages[lang] = count
			}
		}
	}

	// Content types
	rows, err = s.db.QueryContext(ctx, `
		SELECT content_type, COUNT(*) FROM search.documents GROUP BY content_type
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ct string
			var count int
			if err := rows.Scan(&ct, &count); err == nil {
				stats.ContentTypes[ct] = count
			}
		}
	}

	// Top domains
	rows, err = s.db.QueryContext(ctx, `
		SELECT domain, COUNT(*) as doc_count
		FROM search.documents
		GROUP BY domain
		ORDER BY doc_count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ds store.DomainStat
			if err := rows.Scan(&ds.Domain, &ds.Documents); err == nil {
				stats.TopDomains = append(stats.TopDomains, ds)
			}
		}
	}

	return stats, nil
}

// RebuildIndex rebuilds the search index.
func (s *IndexStore) RebuildIndex(ctx context.Context) error {
	// Re-analyze the table to update statistics
	_, err := s.db.ExecContext(ctx, "ANALYZE search.documents")
	if err != nil {
		return fmt.Errorf("failed to analyze table: %w", err)
	}

	// Reindex the GIN index
	_, err = s.db.ExecContext(ctx, "REINDEX INDEX CONCURRENTLY search.idx_documents_search_vector")
	if err != nil {
		// Try without CONCURRENTLY if it fails
		_, err = s.db.ExecContext(ctx, "REINDEX INDEX search.idx_documents_search_vector")
		if err != nil {
			return fmt.Errorf("failed to reindex: %w", err)
		}
	}

	return nil
}

// OptimizeIndex optimizes the search index.
func (s *IndexStore) OptimizeIndex(ctx context.Context) error {
	// Vacuum the table
	_, err := s.db.ExecContext(ctx, "VACUUM ANALYZE search.documents")
	if err != nil {
		return fmt.Errorf("failed to vacuum table: %w", err)
	}

	return nil
}
