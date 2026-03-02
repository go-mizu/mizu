package hn

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

type DownloadState struct {
	Version     int                 `json:"version"`
	CompletedAt time.Time           `json:"completed_at"`
	SourceUsed  string              `json:"source_used"`
	ClickHouse  *ClickHouseRunState `json:"clickhouse,omitempty"`
	Delta       *ClickHouseRunState `json:"delta,omitempty"`
}

type ClickHouseRunState struct {
	StartID           int64 `json:"start_id"`
	EndID             int64 `json:"end_id"`
	RemoteMaxID       int64 `json:"remote_max_id"`
	RemoteCount       int64 `json:"remote_count"`
	ChunkIDSpan       int64 `json:"chunk_id_span"`
	TailRefreshChunks int   `json:"tail_refresh_chunks"`
	IncrementalFromID int64 `json:"incremental_from_id"`
}

type ImportState struct {
	Version      int       `json:"version"`
	CompletedAt  time.Time `json:"completed_at"`
	DBPath       string    `json:"db_path"`
	SourceUsed   string    `json:"source_used"`
	Mode         string    `json:"mode"`
	RowsBefore   int64     `json:"rows_before"`
	RowsAfter    int64     `json:"rows_after"`
	RowsDelta    int64     `json:"rows_delta"`
	ImportFromID int64     `json:"import_from_id,omitempty"`
}

func (c Config) ReadDownloadState() (*DownloadState, error) {
	var st DownloadState
	if err := readJSONFile(c.DownloadStatePath(), &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (c Config) WriteDownloadState(st *DownloadState) error {
	if st == nil {
		return errors.New("nil download state")
	}
	if st.Version == 0 {
		st.Version = 1
	}
	if st.CompletedAt.IsZero() {
		st.CompletedAt = time.Now().UTC()
	}
	return writeJSONFile(c.DownloadStatePath(), st)
}

func (c Config) ReadImportState() (*ImportState, error) {
	var st ImportState
	if err := readJSONFile(c.ImportStatePath(), &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (c Config) WriteImportState(st *ImportState) error {
	if st == nil {
		return errors.New("nil import state")
	}
	if st.Version == 0 {
		st.Version = 1
	}
	if st.CompletedAt.IsZero() {
		st.CompletedAt = time.Now().UTC()
	}
	return writeJSONFile(c.ImportStatePath(), st)
}

func (st *DownloadState) IncrementalFromIDFor(source ImportSource) int64 {
	if st == nil {
		return 0
	}
	switch source {
	case ImportSourceClickHouse:
		if st.Delta != nil {
			if st.Delta.IncrementalFromID > 0 {
				return st.Delta.IncrementalFromID
			}
			if st.Delta.StartID > 0 {
				return st.Delta.StartID
			}
		}
		if st.ClickHouse != nil {
			if st.ClickHouse.IncrementalFromID > 0 {
				return st.ClickHouse.IncrementalFromID
			}
			return st.ClickHouse.StartID
		}
	case ImportSourceAPI:
		return 0
	case ImportSourceHybrid:
		var chFrom, deltaFrom int64
		if st.Delta != nil {
			if st.Delta.IncrementalFromID > 0 {
				deltaFrom = st.Delta.IncrementalFromID
			} else {
				deltaFrom = st.Delta.StartID
			}
		}
		if st.ClickHouse != nil {
			if st.ClickHouse.IncrementalFromID > 0 {
				chFrom = st.ClickHouse.IncrementalFromID
			} else {
				chFrom = st.ClickHouse.StartID
			}
		}
		return minPositiveInt64(deltaFrom, chFrom)
	}
	return 0
}

func readJSONFile(path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(dst)
}
