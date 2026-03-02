package hn

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

type chunkFileKind string

const (
	chunkFileKindClickHouse chunkFileKind = "clickhouse"
	chunkFileKindAPI        chunkFileKind = "api"
)

type localChunkFile struct {
	Kind    chunkFileKind
	Path    string
	StartID int64
	EndID   int64
	Size    int64
}

var (
	reCHChunkFile  = regexp.MustCompile(`^id_(\d{1,20})_(\d{1,20})\.parquet$`)
	reAPIChunkFile = regexp.MustCompile(`^items_(\d{1,20})_(\d{1,20})\.jsonl$`)
)

func parseCHChunkFilePath(path string) (localChunkFile, bool) {
	base := filepath.Base(path)
	m := reCHChunkFile.FindStringSubmatch(base)
	if len(m) != 3 {
		return localChunkFile{}, false
	}
	startID, err1 := strconv.ParseInt(m[1], 10, 64)
	endID, err2 := strconv.ParseInt(m[2], 10, 64)
	if err1 != nil || err2 != nil || startID <= 0 || endID < startID {
		return localChunkFile{}, false
	}
	sz, _ := fileSize(path)
	return localChunkFile{Kind: chunkFileKindClickHouse, Path: path, StartID: startID, EndID: endID, Size: sz}, true
}

func parseAPIChunkFilePath(path string) (localChunkFile, bool) {
	base := filepath.Base(path)
	m := reAPIChunkFile.FindStringSubmatch(base)
	if len(m) != 3 {
		return localChunkFile{}, false
	}
	startID, err1 := strconv.ParseInt(m[1], 10, 64)
	endID, err2 := strconv.ParseInt(m[2], 10, 64)
	if err1 != nil || err2 != nil || startID <= 0 || endID < startID {
		return localChunkFile{}, false
	}
	sz, _ := fileSize(path)
	return localChunkFile{Kind: chunkFileKindAPI, Path: path, StartID: startID, EndID: endID, Size: sz}, true
}

func listLocalCHChunks(dir string) ([]localChunkFile, error) {
	paths, err := sortedGlob(filepath.Join(dir, "*.parquet"))
	if err != nil {
		return nil, err
	}
	out := make([]localChunkFile, 0, len(paths))
	for _, p := range paths {
		if cf, ok := parseCHChunkFilePath(p); ok {
			out = append(out, cf)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartID == out[j].StartID {
			if out[i].EndID == out[j].EndID {
				return out[i].Path < out[j].Path
			}
			return out[i].EndID < out[j].EndID
		}
		return out[i].StartID < out[j].StartID
	})
	return out, nil
}

func listLocalAPIChunks(dir string) ([]localChunkFile, error) {
	paths, err := sortedGlob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	out := make([]localChunkFile, 0, len(paths))
	for _, p := range paths {
		if cf, ok := parseAPIChunkFilePath(p); ok {
			out = append(out, cf)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartID == out[j].StartID {
			if out[i].EndID == out[j].EndID {
				return out[i].Path < out[j].Path
			}
			return out[i].EndID < out[j].EndID
		}
		return out[i].StartID < out[j].StartID
	})
	return out, nil
}

func tailRefreshStartID(startID, endID, span int64, refreshTailChunks int) int64 {
	if refreshTailChunks <= 0 || span <= 0 {
		return 0
	}
	chunkStarts := make([]int64, 0, 16)
	for s := startID; s <= endID; s += span {
		chunkStarts = append(chunkStarts, s)
	}
	if len(chunkStarts) == 0 {
		return 0
	}
	idx := len(chunkStarts) - refreshTailChunks
	if idx < 0 {
		idx = 0
	}
	return chunkStarts[idx]
}

func expectedCHChunkEnd(chunkStart, startID, endID, span int64) (int64, bool) {
	if span <= 0 || chunkStart < startID || chunkStart > endID {
		return 0, false
	}
	// Only starts aligned to the target range are considered expected.
	if (chunkStart-startID)%span != 0 {
		return 0, false
	}
	chunkEnd := chunkStart + span - 1
	if chunkEnd > endID {
		chunkEnd = endID
	}
	return chunkEnd, true
}

func compactPathList(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		m[p] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for p := range m {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func minPositiveInt64(vals ...int64) int64 {
	var out int64
	for _, v := range vals {
		if v <= 0 {
			continue
		}
		if out == 0 || v < out {
			out = v
		}
	}
	return out
}

func newestChunkStarts(chunks []localChunkFile, n int) []int64 {
	if n <= 0 || len(chunks) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(chunks))
	starts := make([]int64, 0, len(chunks))
	for _, cf := range chunks {
		if _, ok := seen[cf.StartID]; ok {
			continue
		}
		seen[cf.StartID] = struct{}{}
		starts = append(starts, cf.StartID)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	if len(starts) > n {
		starts = starts[len(starts)-n:]
	}
	return starts
}

func chunkRangeString(startID, endID int64) string {
	if startID <= 0 && endID <= 0 {
		return ""
	}
	if endID <= 0 {
		return fmt.Sprintf("%d-", startID)
	}
	return fmt.Sprintf("%d-%d", startID, endID)
}

func detectCHChunkSpan(chunks []localChunkFile) int64 {
	if len(chunks) == 0 {
		return 0
	}
	type agg struct {
		count int
	}
	bySpan := map[int64]agg{}
	for _, cf := range chunks {
		span := cf.EndID - cf.StartID + 1
		if span <= 0 {
			continue
		}
		a := bySpan[span]
		a.count++
		bySpan[span] = a
	}
	var bestSpan int64
	var bestCount int
	for span, a := range bySpan {
		if a.count > bestCount || (a.count == bestCount && span > bestSpan) {
			bestSpan = span
			bestCount = a.count
		}
	}
	return bestSpan
}

func (c Config) DetectLocalClickHouseChunkSpan() (int64, bool) {
	chunks, err := listLocalCHChunks(c.WithDefaults().ClickHouseParquetDir())
	if err == nil {
		if span := detectCHChunkSpan(chunks); span > 0 {
			return span, true
		}
	}
	if st, err := c.ReadDownloadState(); err == nil && st != nil && st.ClickHouse != nil && st.ClickHouse.ChunkIDSpan > 0 {
		return st.ClickHouse.ChunkIDSpan, true
	}
	return 0, false
}
