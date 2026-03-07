//go:build !chdb

package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
)

func init() {
	metastore.Register("duckdb", driver{})
}

type driver struct{}

func (driver) Open(dsn string, _ metastore.Options) (metastore.Store, error) {
	if dsn == "" {
		return nil, errors.New("duckdb: empty dsn")
	}
	if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
		return nil, fmt.Errorf("duckdb: mkdir parent: %w", err)
	}

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("duckdb: open: %w", err)
	}
	return &store{db: db}, nil
}

type store struct {
	db *sql.DB
}

func (s *store) Name() string { return "duckdb" }

func (s *store) Init(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS crawl_summary (
			crawl_id VARCHAR PRIMARY KEY,
			warc_count BIGINT NOT NULL,
			warc_total_size BIGINT NOT NULL,
			md_shards BIGINT NOT NULL,
			md_total_size BIGINT NOT NULL,
			md_doc_estimate BIGINT NOT NULL,
			generated_at VARCHAR NOT NULL,
			scan_duration_ms BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS pack_summary (
			crawl_id VARCHAR NOT NULL,
			format VARCHAR NOT NULL,
			bytes BIGINT NOT NULL,
			PRIMARY KEY (crawl_id, format)
		)`,
		`CREATE TABLE IF NOT EXISTS fts_summary (
			crawl_id VARCHAR NOT NULL,
			engine VARCHAR NOT NULL,
			bytes BIGINT NOT NULL,
			shard_count BIGINT NOT NULL,
			PRIMARY KEY (crawl_id, engine)
		)`,
		`CREATE TABLE IF NOT EXISTS warc_summary (
			crawl_id VARCHAR NOT NULL,
			warc_index VARCHAR NOT NULL,
			manifest_index BIGINT NOT NULL DEFAULT -1,
			filename VARCHAR,
			remote_path VARCHAR,
			warc_bytes BIGINT NOT NULL DEFAULT 0,
			markdown_docs BIGINT NOT NULL DEFAULT 0,
			markdown_bytes BIGINT NOT NULL DEFAULT 0,
			total_bytes BIGINT NOT NULL DEFAULT 0,
			updated_at VARCHAR NOT NULL,
			PRIMARY KEY (crawl_id, warc_index)
		)`,
		`CREATE TABLE IF NOT EXISTS warc_pack_summary (
			crawl_id VARCHAR NOT NULL,
			warc_index VARCHAR NOT NULL,
			format VARCHAR NOT NULL,
			bytes BIGINT NOT NULL,
			PRIMARY KEY (crawl_id, warc_index, format)
		)`,
		`CREATE TABLE IF NOT EXISTS warc_fts_summary (
			crawl_id VARCHAR NOT NULL,
			warc_index VARCHAR NOT NULL,
			engine VARCHAR NOT NULL,
			bytes BIGINT NOT NULL,
			PRIMARY KEY (crawl_id, warc_index, engine)
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_state (
			crawl_id VARCHAR PRIMARY KEY,
			status VARCHAR NOT NULL,
			started_at VARCHAR,
			finished_at VARCHAR,
			last_error VARCHAR,
			generation BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id VARCHAR PRIMARY KEY,
			type VARCHAR NOT NULL,
			status VARCHAR NOT NULL,
			config_json VARCHAR NOT NULL DEFAULT '{}',
			progress DOUBLE NOT NULL DEFAULT 0,
			message VARCHAR NOT NULL DEFAULT '',
			rate DOUBLE NOT NULL DEFAULT 0,
			error VARCHAR NOT NULL DEFAULT '',
			started_at VARCHAR NOT NULL,
			ended_at VARCHAR
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("duckdb: init schema: %w", err)
		}
	}
	return nil
}

func (s *store) GetSummary(ctx context.Context, crawlID string) (metastore.SummaryRecord, bool, error) {
	var rec metastore.SummaryRecord
	rec.PackFormats = make(map[string]int64)
	rec.FTSEngines = make(map[string]int64)
	rec.FTSShardCount = make(map[string]int64)

	var generatedAt string
	var scanDurationMS int64
	err := s.db.QueryRowContext(ctx, `
		SELECT crawl_id, warc_count, warc_total_size, md_shards, md_total_size, md_doc_estimate, generated_at, scan_duration_ms
		FROM crawl_summary WHERE crawl_id = ?
	`, crawlID).Scan(
		&rec.CrawlID,
		&rec.WARCCount,
		&rec.WARCTotalSize,
		&rec.MDShards,
		&rec.MDTotalSize,
		&rec.MDDocEstimate,
		&generatedAt,
		&scanDurationMS,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return metastore.SummaryRecord{}, false, nil
	}
	if err != nil {
		return metastore.SummaryRecord{}, false, fmt.Errorf("duckdb: get crawl_summary: %w", err)
	}
	rec.ScanDuration = time.Duration(scanDurationMS) * time.Millisecond
	if generatedAt != "" {
		if t, pErr := time.Parse(time.RFC3339Nano, generatedAt); pErr == nil {
			rec.GeneratedAt = t
		}
	}

	pRows, err := s.db.QueryContext(ctx, `SELECT format, bytes FROM pack_summary WHERE crawl_id = ?`, crawlID)
	if err != nil {
		return metastore.SummaryRecord{}, false, fmt.Errorf("duckdb: get pack_summary: %w", err)
	}
	defer pRows.Close()
	for pRows.Next() {
		var format string
		var bytes int64
		if err := pRows.Scan(&format, &bytes); err != nil {
			return metastore.SummaryRecord{}, false, fmt.Errorf("duckdb: scan pack_summary: %w", err)
		}
		rec.PackFormats[format] = bytes
	}
	if err := pRows.Err(); err != nil {
		return metastore.SummaryRecord{}, false, fmt.Errorf("duckdb: iter pack_summary: %w", err)
	}

	fRows, err := s.db.QueryContext(ctx, `SELECT engine, bytes, shard_count FROM fts_summary WHERE crawl_id = ?`, crawlID)
	if err != nil {
		return metastore.SummaryRecord{}, false, fmt.Errorf("duckdb: get fts_summary: %w", err)
	}
	defer fRows.Close()
	for fRows.Next() {
		var engine string
		var bytes int64
		var shards int64
		if err := fRows.Scan(&engine, &bytes, &shards); err != nil {
			return metastore.SummaryRecord{}, false, fmt.Errorf("duckdb: scan fts_summary: %w", err)
		}
		rec.FTSEngines[engine] = bytes
		rec.FTSShardCount[engine] = shards
	}
	if err := fRows.Err(); err != nil {
		return metastore.SummaryRecord{}, false, fmt.Errorf("duckdb: iter fts_summary: %w", err)
	}

	return rec, true, nil
}

func (s *store) PutSummary(ctx context.Context, rec metastore.SummaryRecord) error {
	if rec.GeneratedAt.IsZero() {
		rec.GeneratedAt = time.Now().UTC()
	}
	if rec.PackFormats == nil {
		rec.PackFormats = make(map[string]int64)
	}
	if rec.FTSEngines == nil {
		rec.FTSEngines = make(map[string]int64)
	}
	if rec.FTSShardCount == nil {
		rec.FTSShardCount = make(map[string]int64)
	}
	for i := range rec.WARCs {
		if rec.WARCs[i].PackBytes == nil {
			rec.WARCs[i].PackBytes = make(map[string]int64)
		}
		if rec.WARCs[i].FTSBytes == nil {
			rec.WARCs[i].FTSBytes = make(map[string]int64)
		}
		if rec.WARCs[i].UpdatedAt.IsZero() {
			rec.WARCs[i].UpdatedAt = rec.GeneratedAt
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("duckdb: begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM crawl_summary WHERE crawl_id = ?`, rec.CrawlID); err != nil {
		return fmt.Errorf("duckdb: delete crawl_summary: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO crawl_summary (
			crawl_id, warc_count, warc_total_size, md_shards, md_total_size, md_doc_estimate, generated_at, scan_duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rec.CrawlID, rec.WARCCount, rec.WARCTotalSize, rec.MDShards, rec.MDTotalSize, rec.MDDocEstimate,
		rec.GeneratedAt.UTC().Format(time.RFC3339Nano), rec.ScanDuration.Milliseconds(),
	)
	if err != nil {
		return fmt.Errorf("duckdb: insert crawl_summary: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM pack_summary WHERE crawl_id = ?`, rec.CrawlID); err != nil {
		return fmt.Errorf("duckdb: delete pack_summary: %w", err)
	}
	for format, bytes := range rec.PackFormats {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO pack_summary (crawl_id, format, bytes) VALUES (?, ?, ?)
		`, rec.CrawlID, format, bytes); err != nil {
			return fmt.Errorf("duckdb: insert pack_summary: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM fts_summary WHERE crawl_id = ?`, rec.CrawlID); err != nil {
		return fmt.Errorf("duckdb: delete fts_summary: %w", err)
	}
	for engine, bytes := range rec.FTSEngines {
		shards := rec.FTSShardCount[engine]
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO fts_summary (crawl_id, engine, bytes, shard_count) VALUES (?, ?, ?, ?)
		`, rec.CrawlID, engine, bytes, shards); err != nil {
			return fmt.Errorf("duckdb: insert fts_summary: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM warc_pack_summary WHERE crawl_id = ?`, rec.CrawlID); err != nil {
		return fmt.Errorf("duckdb: delete warc_pack_summary: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM warc_fts_summary WHERE crawl_id = ?`, rec.CrawlID); err != nil {
		return fmt.Errorf("duckdb: delete warc_fts_summary: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM warc_summary WHERE crawl_id = ?`, rec.CrawlID); err != nil {
		return fmt.Errorf("duckdb: delete warc_summary: %w", err)
	}
	for _, wr := range rec.WARCs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO warc_summary (
				crawl_id, warc_index, manifest_index, filename, remote_path,
				warc_bytes, markdown_docs, markdown_bytes, total_bytes, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, rec.CrawlID, wr.WARCIndex, wr.ManifestIndex, wr.Filename, wr.RemotePath,
			wr.WARCBytes, wr.MarkdownDocs, wr.MarkdownBytes, wr.TotalBytes,
			wr.UpdatedAt.UTC().Format(time.RFC3339Nano),
		); err != nil {
			return fmt.Errorf("duckdb: insert warc_summary: %w", err)
		}
		for format, bytes := range wr.PackBytes {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO warc_pack_summary (crawl_id, warc_index, format, bytes) VALUES (?, ?, ?, ?)
			`, rec.CrawlID, wr.WARCIndex, format, bytes); err != nil {
				return fmt.Errorf("duckdb: insert warc_pack_summary: %w", err)
			}
		}
		for engine, bytes := range wr.FTSBytes {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO warc_fts_summary (crawl_id, warc_index, engine, bytes) VALUES (?, ?, ?, ?)
			`, rec.CrawlID, wr.WARCIndex, engine, bytes); err != nil {
				return fmt.Errorf("duckdb: insert warc_fts_summary: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("duckdb: commit tx: %w", err)
	}
	return nil
}

func (s *store) ListWARCs(ctx context.Context, crawlID string) ([]metastore.WARCRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT warc_index, manifest_index, filename, remote_path, warc_bytes, markdown_docs, markdown_bytes, total_bytes, updated_at
		FROM warc_summary
		WHERE crawl_id = ?
		ORDER BY warc_index ASC
	`, crawlID)
	if err != nil {
		return nil, fmt.Errorf("duckdb: list warc_summary: %w", err)
	}
	defer rows.Close()

	out := make([]metastore.WARCRecord, 0, 256)
	for rows.Next() {
		var wr metastore.WARCRecord
		var updatedAt string
		wr.CrawlID = crawlID
		wr.PackBytes = make(map[string]int64)
		wr.FTSBytes = make(map[string]int64)
		if err := rows.Scan(
			&wr.WARCIndex,
			&wr.ManifestIndex,
			&wr.Filename,
			&wr.RemotePath,
			&wr.WARCBytes,
			&wr.MarkdownDocs,
			&wr.MarkdownBytes,
			&wr.TotalBytes,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("duckdb: scan warc_summary: %w", err)
		}
		if updatedAt != "" {
			if t, pErr := time.Parse(time.RFC3339Nano, updatedAt); pErr == nil {
				wr.UpdatedAt = t
			}
		}
		out = append(out, wr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdb: iter warc_summary: %w", err)
	}
	if len(out) == 0 {
		return out, nil
	}
	if err := s.loadWARCMaps(ctx, crawlID, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *store) GetWARC(ctx context.Context, crawlID, warcIndex string) (metastore.WARCRecord, bool, error) {
	var wr metastore.WARCRecord
	var updatedAt string
	wr.CrawlID = crawlID
	wr.PackBytes = make(map[string]int64)
	wr.FTSBytes = make(map[string]int64)
	err := s.db.QueryRowContext(ctx, `
		SELECT warc_index, manifest_index, filename, remote_path, warc_bytes, markdown_docs, markdown_bytes, total_bytes, updated_at
		FROM warc_summary
		WHERE crawl_id = ? AND warc_index = ?
	`, crawlID, warcIndex).Scan(
		&wr.WARCIndex,
		&wr.ManifestIndex,
		&wr.Filename,
		&wr.RemotePath,
		&wr.WARCBytes,
		&wr.MarkdownDocs,
		&wr.MarkdownBytes,
		&wr.TotalBytes,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return metastore.WARCRecord{}, false, nil
	}
	if err != nil {
		return metastore.WARCRecord{}, false, fmt.Errorf("duckdb: get warc_summary: %w", err)
	}
	if updatedAt != "" {
		if t, pErr := time.Parse(time.RFC3339Nano, updatedAt); pErr == nil {
			wr.UpdatedAt = t
		}
	}
	recs := []metastore.WARCRecord{wr}
	if err := s.loadWARCMaps(ctx, crawlID, recs); err != nil {
		return metastore.WARCRecord{}, false, err
	}
	return recs[0], true, nil
}

func (s *store) loadWARCMaps(ctx context.Context, crawlID string, recs []metastore.WARCRecord) error {
	byIndex := make(map[string]*metastore.WARCRecord, len(recs))
	for i := range recs {
		rec := &recs[i]
		byIndex[rec.WARCIndex] = rec
	}

	pRows, err := s.db.QueryContext(ctx, `
		SELECT warc_index, format, bytes
		FROM warc_pack_summary
		WHERE crawl_id = ?
	`, crawlID)
	if err != nil {
		return fmt.Errorf("duckdb: list warc_pack_summary: %w", err)
	}
	defer pRows.Close()
	for pRows.Next() {
		var warcIndex, format string
		var bytes int64
		if err := pRows.Scan(&warcIndex, &format, &bytes); err != nil {
			return fmt.Errorf("duckdb: scan warc_pack_summary: %w", err)
		}
		if wr, ok := byIndex[warcIndex]; ok {
			wr.PackBytes[format] = bytes
		}
	}
	if err := pRows.Err(); err != nil {
		return fmt.Errorf("duckdb: iter warc_pack_summary: %w", err)
	}

	fRows, err := s.db.QueryContext(ctx, `
		SELECT warc_index, engine, bytes
		FROM warc_fts_summary
		WHERE crawl_id = ?
	`, crawlID)
	if err != nil {
		return fmt.Errorf("duckdb: list warc_fts_summary: %w", err)
	}
	defer fRows.Close()
	for fRows.Next() {
		var warcIndex, engine string
		var bytes int64
		if err := fRows.Scan(&warcIndex, &engine, &bytes); err != nil {
			return fmt.Errorf("duckdb: scan warc_fts_summary: %w", err)
		}
		if wr, ok := byIndex[warcIndex]; ok {
			wr.FTSBytes[engine] = bytes
		}
	}
	if err := fRows.Err(); err != nil {
		return fmt.Errorf("duckdb: iter warc_fts_summary: %w", err)
	}
	return nil
}

func (s *store) GetRefreshState(ctx context.Context, crawlID string) (metastore.RefreshState, bool, error) {
	var st metastore.RefreshState
	st.CrawlID = crawlID

	var startedAt sql.NullString
	var finishedAt sql.NullString
	var lastErr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT status, started_at, finished_at, last_error, generation
		FROM refresh_state WHERE crawl_id = ?
	`, crawlID).Scan(&st.Status, &startedAt, &finishedAt, &lastErr, &st.Generation)
	if errors.Is(err, sql.ErrNoRows) {
		return metastore.RefreshState{}, false, nil
	}
	if err != nil {
		return metastore.RefreshState{}, false, fmt.Errorf("duckdb: get refresh_state: %w", err)
	}

	if startedAt.Valid && startedAt.String != "" {
		if t, pErr := time.Parse(time.RFC3339Nano, startedAt.String); pErr == nil {
			st.StartedAt = &t
		}
	}
	if finishedAt.Valid && finishedAt.String != "" {
		if t, pErr := time.Parse(time.RFC3339Nano, finishedAt.String); pErr == nil {
			st.FinishedAt = &t
		}
	}
	if lastErr.Valid {
		st.LastError = lastErr.String
	}

	return st, true, nil
}

func (s *store) SetRefreshState(ctx context.Context, st metastore.RefreshState) error {
	startedAt := ""
	finishedAt := ""
	if st.StartedAt != nil {
		startedAt = st.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if st.FinishedAt != nil {
		finishedAt = st.FinishedAt.UTC().Format(time.RFC3339Nano)
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM refresh_state WHERE crawl_id = ?`, st.CrawlID); err != nil {
		return fmt.Errorf("duckdb: delete refresh_state: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO refresh_state (crawl_id, status, started_at, finished_at, last_error, generation)
		VALUES (?, ?, ?, ?, ?, ?)
	`, st.CrawlID, st.Status, startedAt, finishedAt, st.LastError, st.Generation)
	if err != nil {
		return fmt.Errorf("duckdb: insert refresh_state: %w", err)
	}
	return nil
}

func (s *store) ListJobs(ctx context.Context) ([]metastore.JobRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, status, config_json, progress, message, rate, error, started_at, ended_at
		FROM jobs ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("duckdb: list jobs: %w", err)
	}
	defer rows.Close()

	var out []metastore.JobRecord
	for rows.Next() {
		var rec metastore.JobRecord
		var startedAt string
		var endedAt sql.NullString
		if err := rows.Scan(
			&rec.ID, &rec.Type, &rec.Status, &rec.Config,
			&rec.Progress, &rec.Message, &rec.Rate, &rec.Error,
			&startedAt, &endedAt,
		); err != nil {
			return nil, fmt.Errorf("duckdb: scan job: %w", err)
		}
		if t, pErr := time.Parse(time.RFC3339Nano, startedAt); pErr == nil {
			rec.StartedAt = t
		}
		if endedAt.Valid && endedAt.String != "" {
			if t, pErr := time.Parse(time.RFC3339Nano, endedAt.String); pErr == nil {
				rec.EndedAt = &t
			}
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *store) PutJob(ctx context.Context, rec metastore.JobRecord) error {
	endedAt := ""
	if rec.EndedAt != nil {
		endedAt = rec.EndedAt.UTC().Format(time.RFC3339Nano)
	}
	// DuckDB doesn't support ON CONFLICT ... DO UPDATE, so delete+insert.
	if _, err := s.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, rec.ID); err != nil {
		return fmt.Errorf("duckdb: delete job: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO jobs (id, type, status, config_json, progress, message, rate, error, started_at, ended_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rec.ID, rec.Type, rec.Status, rec.Config, rec.Progress, rec.Message, rec.Rate, rec.Error,
		rec.StartedAt.UTC().Format(time.RFC3339Nano), endedAt)
	if err != nil {
		return fmt.Errorf("duckdb: insert job: %w", err)
	}
	return nil
}

func (s *store) DeleteAllJobs(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM jobs`)
	if err != nil {
		return fmt.Errorf("duckdb: delete all jobs: %w", err)
	}
	return nil
}

func (s *store) Close() error {
	return s.db.Close()
}
