package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// IndexStore handles document indexing.
type IndexStore struct {
	db *sql.DB
}

// IndexDocument indexes a new document.
func (s *IndexStore) IndexDocument(ctx context.Context, doc *store.Document) error {
	if doc.ID == "" {
		doc.ID = generateID()
	}
	doc.CrawledAt = time.Now()
	doc.UpdatedAt = time.Now()
	doc.WordCount = countWords(doc.Content)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO documents (id, url, title, description, content, domain, language, content_type, favicon, word_count, crawled_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			content = excluded.content,
			favicon = excluded.favicon,
			word_count = excluded.word_count,
			updated_at = excluded.updated_at
	`, doc.ID, doc.URL, doc.Title, doc.Description, doc.Content, doc.Domain, doc.Language, doc.ContentType, doc.Favicon, doc.WordCount, doc.CrawledAt, doc.UpdatedAt)

	return err
}

// UpdateDocument updates an existing document.
func (s *IndexStore) UpdateDocument(ctx context.Context, doc *store.Document) error {
	doc.UpdatedAt = time.Now()
	doc.WordCount = countWords(doc.Content)

	result, err := s.db.ExecContext(ctx, `
		UPDATE documents SET
			title = ?,
			description = ?,
			content = ?,
			favicon = ?,
			word_count = ?,
			updated_at = ?
		WHERE id = ?
	`, doc.Title, doc.Description, doc.Content, doc.Favicon, doc.WordCount, doc.UpdatedAt, doc.ID)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// DeleteDocument removes a document.
func (s *IndexStore) DeleteDocument(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM documents WHERE id = ?", id)
	if err != nil {
		return err
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
	var desc, content, favicon, metadata sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, url, title, description, content, domain, language, content_type, favicon, word_count, crawled_at, updated_at, metadata
		FROM documents WHERE id = ?
	`, id).Scan(
		&doc.ID, &doc.URL, &doc.Title, &desc, &content, &doc.Domain,
		&doc.Language, &doc.ContentType, &favicon, &doc.WordCount,
		&doc.CrawledAt, &doc.UpdatedAt, &metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, err
	}

	if desc.Valid {
		doc.Description = desc.String
	}
	if content.Valid {
		doc.Content = content.String
	}
	if favicon.Valid {
		doc.Favicon = favicon.String
	}

	return &doc, nil
}

// GetDocumentByURL retrieves a document by URL.
func (s *IndexStore) GetDocumentByURL(ctx context.Context, url string) (*store.Document, error) {
	var doc store.Document
	var desc, content, favicon, metadata sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, url, title, description, content, domain, language, content_type, favicon, word_count, crawled_at, updated_at, metadata
		FROM documents WHERE url = ?
	`, url).Scan(
		&doc.ID, &doc.URL, &doc.Title, &desc, &content, &doc.Domain,
		&doc.Language, &doc.ContentType, &favicon, &doc.WordCount,
		&doc.CrawledAt, &doc.UpdatedAt, &metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, err
	}

	if desc.Valid {
		doc.Description = desc.String
	}
	if content.Valid {
		doc.Content = content.String
	}
	if favicon.Valid {
		doc.Favicon = favicon.String
	}

	return &doc, nil
}

// ListDocuments lists documents with pagination.
func (s *IndexStore) ListDocuments(ctx context.Context, limit, offset int) ([]*store.Document, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, url, title, description, domain, language, content_type, favicon, word_count, crawled_at, updated_at
		FROM documents
		ORDER BY crawled_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*store.Document
	for rows.Next() {
		var doc store.Document
		var desc, favicon sql.NullString
		if err := rows.Scan(
			&doc.ID, &doc.URL, &doc.Title, &desc, &doc.Domain,
			&doc.Language, &doc.ContentType, &favicon, &doc.WordCount,
			&doc.CrawledAt, &doc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if desc.Valid {
			doc.Description = desc.String
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
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO documents (id, url, title, description, content, domain, language, content_type, favicon, word_count, crawled_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			content = excluded.content,
			favicon = excluded.favicon,
			word_count = excluded.word_count,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, doc := range docs {
		if doc.ID == "" {
			doc.ID = generateID()
		}
		doc.CrawledAt = now
		doc.UpdatedAt = now
		doc.WordCount = countWords(doc.Content)

		_, err := stmt.ExecContext(ctx,
			doc.ID, doc.URL, doc.Title, doc.Description, doc.Content,
			doc.Domain, doc.Language, doc.ContentType, doc.Favicon,
			doc.WordCount, doc.CrawledAt, doc.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetIndexStats returns index statistics.
func (s *IndexStore) GetIndexStats(ctx context.Context) (*store.IndexStats, error) {
	stats := &store.IndexStats{
		Languages:    make(map[string]int),
		ContentTypes: make(map[string]int),
	}

	// Total documents
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents").Scan(&stats.TotalDocuments)

	// Total size (approximate)
	s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(LENGTH(content)), 0) FROM documents").Scan(&stats.TotalSize)

	// Last updated
	s.db.QueryRowContext(ctx, "SELECT MAX(updated_at) FROM documents").Scan(&stats.LastUpdated)

	// Languages
	rows, _ := s.db.QueryContext(ctx, "SELECT language, COUNT(*) FROM documents GROUP BY language")
	if rows != nil {
		for rows.Next() {
			var lang string
			var count int
			rows.Scan(&lang, &count)
			stats.Languages[lang] = count
		}
		rows.Close()
	}

	// Content types
	rows, _ = s.db.QueryContext(ctx, "SELECT content_type, COUNT(*) FROM documents GROUP BY content_type")
	if rows != nil {
		for rows.Next() {
			var ct string
			var count int
			rows.Scan(&ct, &count)
			stats.ContentTypes[ct] = count
		}
		rows.Close()
	}

	// Top domains
	rows, _ = s.db.QueryContext(ctx, "SELECT domain, COUNT(*) as cnt FROM documents GROUP BY domain ORDER BY cnt DESC LIMIT 10")
	if rows != nil {
		for rows.Next() {
			var ds store.DomainStat
			rows.Scan(&ds.Domain, &ds.Documents)
			stats.TopDomains = append(stats.TopDomains, ds)
		}
		rows.Close()
	}

	return stats, nil
}

// RebuildIndex rebuilds the FTS index.
func (s *IndexStore) RebuildIndex(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO documents_fts(documents_fts) VALUES('rebuild')")
	return err
}

// OptimizeIndex optimizes the FTS index.
func (s *IndexStore) OptimizeIndex(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO documents_fts(documents_fts) VALUES('optimize')")
	return err
}

// Helper functions

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func countWords(s string) int {
	return len(strings.Fields(s))
}
