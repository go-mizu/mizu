package hn

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TypeCount is a grouped item count by HN item type.
type TypeCount struct {
	Type  string
	Count int64
}

// LocalStatus summarizes local HN dataset files and database state.
type LocalStatus struct {
	DataDir          string
	ParquetPath      string
	ParquetExists    bool
	ParquetSize      int64
	CHParquetDir     string
	CHParquetCount   int
	CHParquetBytes   int64
	CHParquetMinID   int64
	CHParquetMaxID   int64
	CHParquetSpan    int64
	CHParquetRows    int64
	CHParquetMaxTime string
	CHDeltaDir       string
	CHDeltaCount     int
	CHDeltaBytes     int64
	CHDeltaMinID     int64
	CHDeltaMaxID     int64
	CHDeltaSpan      int64
	CHDeltaRows      int64
	CHDeltaMaxTime   string
	APIChunksDir     string
	APIChunkCount    int
	APIChunkBytes    int64
	APIChunkSample   []string
	DBPath           string
	DBExists         bool
	DBSize           int64
	DBRows           int64
	DBMinID          int64
	DBMaxID          int64
	DBMaxTime        string
	DBTypes          []TypeCount
}

func (c Config) LocalStatus(ctx context.Context) (*LocalStatus, error) {
	cfg := c.WithDefaults()
	st := &LocalStatus{
		DataDir:      cfg.BaseDir(),
		ParquetPath:  cfg.RawParquetPath(),
		CHParquetDir: cfg.ClickHouseParquetDir(),
		CHDeltaDir:   cfg.ClickHouseDeltaParquetDir(),
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
		if chunks, err := listLocalCHChunks(st.CHParquetDir); err == nil && len(chunks) > 0 {
			st.CHParquetMinID = chunks[0].StartID
			st.CHParquetMaxID = chunks[0].EndID
			for _, cf := range chunks[1:] {
				if cf.StartID < st.CHParquetMinID {
					st.CHParquetMinID = cf.StartID
				}
				if cf.EndID > st.CHParquetMaxID {
					st.CHParquetMaxID = cf.EndID
				}
			}
			st.CHParquetSpan = detectCHChunkSpan(chunks)
		}
		_ = fillParquetSetStatus(ctx, filepath.Join(st.CHParquetDir, "*.parquet"), &st.CHParquetRows, &st.CHParquetMaxID, &st.CHParquetMaxTime)
	}

	chDeltaFiles, err := sortedGlob(filepath.Join(st.CHDeltaDir, "*.parquet"))
	if err == nil {
		st.CHDeltaCount = len(chDeltaFiles)
		for _, p := range chDeltaFiles {
			if sz, ok := fileSize(p); ok {
				st.CHDeltaBytes += sz
			}
		}
		if chunks, err := listLocalCHChunks(st.CHDeltaDir); err == nil && len(chunks) > 0 {
			st.CHDeltaMinID = chunks[0].StartID
			st.CHDeltaMaxID = chunks[0].EndID
			for _, cf := range chunks[1:] {
				if cf.StartID < st.CHDeltaMinID {
					st.CHDeltaMinID = cf.StartID
				}
				if cf.EndID > st.CHDeltaMaxID {
					st.CHDeltaMaxID = cf.EndID
				}
			}
			st.CHDeltaSpan = detectCHChunkSpan(chunks)
		}
		_ = fillParquetSetStatus(ctx, filepath.Join(st.CHDeltaDir, "*.parquet"), &st.CHDeltaRows, &st.CHDeltaMaxID, &st.CHDeltaMaxTime)
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

	var minID, maxID sql.NullInt64
	var maxTime sql.NullString
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*)::BIGINT, MIN(id), MAX(id), CAST(MAX(time_ts) AS VARCHAR) FROM items`).Scan(&st.DBRows, &minID, &maxID, &maxTime)
	if minID.Valid {
		st.DBMinID = minID.Int64
	}
	if maxID.Valid {
		st.DBMaxID = maxID.Int64
	}
	if maxTime.Valid {
		st.DBMaxTime = maxTime.String
	}
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

func fillParquetSetStatus(ctx context.Context, globPattern string, rowsOut, maxIDOut *int64, maxTimeOut *string) error {
	paths, err := sortedGlob(globPattern)
	if err != nil || len(paths) == 0 {
		return err
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("open duckdb for parquet status: %w", err)
	}
	defer db.Close()
	var rows sql.NullInt64
	var maxID sql.NullInt64
	q := fmt.Sprintf(`SELECT COUNT(*)::BIGINT, MAX(id)
FROM read_parquet('%s', union_by_name=true)`, escapeSQLString(globPattern))
	if err := db.QueryRowContext(ctx, q).Scan(&rows, &maxID); err != nil {
		return err
	}
	if rows.Valid && rowsOut != nil {
		*rowsOut = rows.Int64
	}
	if maxID.Valid && maxIDOut != nil {
		*maxIDOut = maxID.Int64
	}
	if maxTimeOut != nil {
		for i := len(paths) - 1; i >= 0; i-- {
			if s, ok := parquetFileMaxTime(ctx, paths[i]); ok {
				*maxTimeOut = s
				break
			}
		}
	}
	return nil
}

func parquetFileMaxTime(ctx context.Context, path string) (string, bool) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return "", false
	}
	defer db.Close()
	escaped := escapeSQLString(path)
	for _, q := range []string{
		fmt.Sprintf(`SELECT CAST(MAX(epoch_ms(try_cast(time AS BIGINT) * 1000)) AS VARCHAR) FROM read_parquet('%s')`, escaped),
		fmt.Sprintf(`SELECT CAST(MAX(time) AS VARCHAR) FROM read_parquet('%s')`, escaped),
	} {
		var s sql.NullString
		if err := db.QueryRowContext(ctx, q).Scan(&s); err == nil && s.Valid && s.String != "" {
			// Normalize numeric unix seconds if returned by some parquet variants.
			if n, perr := parseInt64Any(s.String); perr == nil && n > 0 {
				return time.Unix(n, 0).UTC().Format("2006-01-02 15:04:05"), true
			}
			return s.String, true
		}
	}
	return "", false
}

func (c Config) RemoveParquet() error {
	return os.Remove(c.RawParquetPath())
}
