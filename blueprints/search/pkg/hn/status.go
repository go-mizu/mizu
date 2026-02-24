package hn

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TypeCount is a grouped item count by HN item type.
type TypeCount struct {
	Type  string
	Count int64
}

// LocalStatus summarizes local HN dataset files and database state.
type LocalStatus struct {
	DataDir        string
	ParquetPath    string
	ParquetExists  bool
	ParquetSize    int64
	CHParquetDir   string
	CHParquetCount int
	CHParquetBytes int64
	APIChunksDir   string
	APIChunkCount  int
	APIChunkBytes  int64
	APIChunkSample []string
	DBPath         string
	DBExists       bool
	DBSize         int64
	DBRows         int64
	DBTypes        []TypeCount
}

func (c Config) LocalStatus(ctx context.Context) (*LocalStatus, error) {
	cfg := c.WithDefaults()
	st := &LocalStatus{
		DataDir:      cfg.BaseDir(),
		ParquetPath:  cfg.RawParquetPath(),
		CHParquetDir: cfg.ClickHouseParquetDir(),
		APIChunksDir: cfg.APIChunksDir(),
		DBPath:       cfg.DefaultDBPath(),
	}
	if size, ok := fileSize(st.ParquetPath); ok {
		st.ParquetExists = true
		st.ParquetSize = size
	}

	chParquetFiles, err := sortedGlob(filepath.Join(st.CHParquetDir, "*.parquet"))
	if err == nil {
		st.CHParquetCount = len(chParquetFiles)
		for _, p := range chParquetFiles {
			if sz, ok := fileSize(p); ok {
				st.CHParquetBytes += sz
			}
		}
	}

	chunkFiles, err := sortedGlob(filepath.Join(st.APIChunksDir, "*.jsonl"))
	if err == nil {
		st.APIChunkCount = len(chunkFiles)
		for _, p := range chunkFiles {
			if sz, ok := fileSize(p); ok {
				st.APIChunkBytes += sz
			}
		}
		for i := 0; i < len(chunkFiles) && i < 3; i++ {
			st.APIChunkSample = append(st.APIChunkSample, filepath.Base(chunkFiles[i]))
		}
	}

	if size, ok := fileSize(st.DBPath); ok {
		st.DBExists = true
		st.DBSize = size
		_ = fillDBStatus(ctx, st)
	}
	return st, nil
}

func fillDBStatus(ctx context.Context, st *LocalStatus) error {
	db, err := sql.Open("duckdb", st.DBPath+"?access_mode=read_only")
	if err != nil {
		return fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items`).Scan(&st.DBRows)
	rows, err := db.QueryContext(ctx, `SELECT COALESCE(type, ''), COUNT(*) FROM items GROUP BY 1 ORDER BY 2 DESC, 1 ASC LIMIT 10`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var tc TypeCount
		if err := rows.Scan(&tc.Type, &tc.Count); err == nil {
			st.DBTypes = append(st.DBTypes, tc)
		}
	}
	sort.Slice(st.DBTypes, func(i, j int) bool {
		if st.DBTypes[i].Count == st.DBTypes[j].Count {
			return st.DBTypes[i].Type < st.DBTypes[j].Type
		}
		return st.DBTypes[i].Count > st.DBTypes[j].Count
	})
	return nil
}

func (c Config) RemoveParquet() error {
	return os.Remove(c.RawParquetPath())
}
