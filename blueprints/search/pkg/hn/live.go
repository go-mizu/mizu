package hn

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

// LocalHighWatermark summarizes the highest known local HN id across raw/downloaded/imported assets.
type LocalHighWatermark struct {
	FromDownloadState int64
	FromCHChunks      int64
	FromCHDelta       int64
	FromDB            int64
	MaxKnownID        int64
}

func (c Config) LocalHighWatermark(ctx context.Context, dbPath string) (*LocalHighWatermark, error) {
	cfg := c.WithDefaults()
	out := &LocalHighWatermark{}

	if st, err := cfg.ReadDownloadState(); err == nil && st != nil {
		if st.ClickHouse != nil && st.ClickHouse.EndID > out.FromDownloadState {
			out.FromDownloadState = st.ClickHouse.EndID
		}
		if st.Delta != nil && st.Delta.EndID > out.FromDownloadState {
			out.FromDownloadState = st.Delta.EndID
		}
	}
	if chDelta, err := listLocalCHChunks(cfg.ClickHouseDeltaParquetDir()); err == nil {
		for _, cf := range chDelta {
			if cf.EndID > out.FromCHDelta {
				out.FromCHDelta = cf.EndID
			}
		}
	}
	if chChunks, err := listLocalCHChunks(cfg.ClickHouseParquetDir()); err == nil {
		for _, cf := range chChunks {
			if cf.EndID > out.FromCHChunks {
				out.FromCHChunks = cf.EndID
			}
		}
	}
	if dbPath == "" {
		dbPath = cfg.DefaultDBPath()
	}
	if mx, err := cfg.dbMaxID(ctx, dbPath); err == nil {
		out.FromDB = mx
	}

	out.MaxKnownID = out.FromDownloadState
	if out.FromCHDelta > out.MaxKnownID {
		out.MaxKnownID = out.FromCHDelta
	}
	if out.FromCHChunks > out.MaxKnownID {
		out.MaxKnownID = out.FromCHChunks
	}
	if out.FromDB > out.MaxKnownID {
		out.MaxKnownID = out.FromDB
	}
	return out, nil
}

func (c Config) SuggestClickHouseDeltaStartID(ctx context.Context, explicitFromID int64, dbPath string) (int64, *LocalHighWatermark, error) {
	if explicitFromID > 0 {
		hw, _ := c.LocalHighWatermark(ctx, dbPath)
		return explicitFromID, hw, nil
	}
	hw, err := c.LocalHighWatermark(ctx, dbPath)
	if err != nil {
		return 1, nil, err
	}
	fromID := hw.MaxKnownID + 1
	if fromID <= 0 {
		fromID = 1
	}
	return fromID, hw, nil
}

func (c Config) dbMaxID(ctx context.Context, dbPath string) (int64, error) {
	if dbPath == "" {
		dbPath = c.WithDefaults().DefaultDBPath()
	}
	if !fileExistsNonEmpty(dbPath) {
		return 0, nil
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return 0, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()
	var mx sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT MAX(id) FROM items`).Scan(&mx); err != nil {
		return 0, err
	}
	if !mx.Valid {
		return 0, nil
	}
	return mx.Int64, nil
}
