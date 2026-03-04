//go:build !chdb

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// PackDuckDBRaw packs all markdown files from markdownDir into a DuckDB database
// with a raw docs table (no FTS index). This is the fastest DuckDB baseline for
// bulk import benchmarking.
func PackDuckDBRaw(ctx context.Context, markdownDir, packPath string, workers, batchSize int, progress index.ProgressFunc) (*index.PipelineStats, error) {
	if err := os.MkdirAll(filepath.Dir(packPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.Remove(packPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	db, err := sql.Open("duckdb", packPath)
	if err != nil {
		return nil, fmt.Errorf("open duckdb raw: %w", err)
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE docs (doc_id VARCHAR PRIMARY KEY, text VARCHAR)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create docs table: %w", err)
	}

	eng := &duckdbRawWriter{db: db}

	stats, pipeErr := index.RunPipeline(ctx, eng, index.PipelineConfig{
		SourceDir: markdownDir,
		BatchSize: batchSize,
		Workers:   workers,
	}, progress)

	closeErr := db.Close()

	if pipeErr != nil {
		os.Remove(packPath)
		return stats, pipeErr
	}
	return stats, closeErr
}

// RunPipelineFromDuckDBRaw reads a DuckDB raw pack and feeds documents into engine.
// Counts total rows upfront so progress can show percentage.
func RunPipelineFromDuckDBRaw(ctx context.Context, engine index.Engine, packPath string, batchSize int, progress index.PackProgressFunc) (*index.PipelineStats, error) {
	db, err := sql.Open("duckdb", packPath)
	if err != nil {
		return nil, fmt.Errorf("open duckdb raw: %w", err)
	}
	defer db.Close()

	var total int64
	_ = db.QueryRowContext(ctx, "SELECT count(*) FROM docs").Scan(&total)

	docCh := make(chan index.Document, max(batchSize*2, 1024))
	go func() {
		defer close(docCh)
		rows, err := db.QueryContext(ctx, "SELECT doc_id, text FROM docs")
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			if ctx.Err() != nil {
				return
			}
			var doc index.Document
			if err := rows.Scan(&doc.DocID, &doc.Text); err != nil {
				return
			}
			select {
			case docCh <- doc:
			case <-ctx.Done():
				return
			}
		}
	}()

	return index.RunPipelineFromChannel(ctx, engine, docCh, total, batchSize, progress)
}

// duckdbRawWriter implements index.Engine for writing into the raw docs table.
type duckdbRawWriter struct {
	db *sql.DB
}

func (e *duckdbRawWriter) Name() string                                         { return "duckdb-raw-writer" }
func (e *duckdbRawWriter) Open(_ context.Context, _ string) error               { return nil }
func (e *duckdbRawWriter) Close() error                                         { return nil }
func (e *duckdbRawWriter) Stats(_ context.Context) (index.EngineStats, error)   { return index.EngineStats{}, nil }
func (e *duckdbRawWriter) Search(_ context.Context, _ index.Query) (index.Results, error) {
	return index.Results{}, nil
}

func (e *duckdbRawWriter) Index(ctx context.Context, docs []index.Document) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO docs (doc_id, text) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, doc := range docs {
		if _, err := stmt.ExecContext(ctx, doc.DocID, string(doc.Text)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
