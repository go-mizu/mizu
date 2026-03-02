package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ClickHouseRemoteInfo describes the current state of the ClickHouse HN table.
type ClickHouseRemoteInfo struct {
	BaseURL   string    `json:"base_url"`
	Database  string    `json:"database"`
	Table     string    `json:"table"`
	Count     int64     `json:"count"`
	MaxID     int64     `json:"max_id"`
	MaxTime   string    `json:"max_time"`
	CheckedAt time.Time `json:"checked_at"`
	User      string    `json:"user,omitempty"`
}

// ClickHouseDownloadOptions controls chunked parquet export from sql.clickhouse.com.
type ClickHouseDownloadOptions struct {
	FromID            int64
	ToID              int64 // 0 means remote max id
	ChunkIDSpan       int64
	Parallelism       int
	RefreshTailChunks int
	OutputDir         string
	AlignCheckpoints  bool
	Force             bool
}

// ClickHouseDownloadProgress reports chunk-level export progress.
type ClickHouseDownloadProgress struct {
	ChunkStart      int64
	ChunkEnd        int64
	ChunkPath       string
	ChunksTotal     int
	ChunksDone      int
	ChunksSkipped   int
	ChunksActive    int
	ChunkBytes      int64
	ChunkElapsed    time.Duration
	ChunkSpeedBPS   float64
	BytesDone       int64
	BytesSkipped    int64
	OverallSpeedBPS float64
	Elapsed         time.Duration
	Complete        bool
	Detail          string
}

// ClickHouseDownloadResult summarizes a clickhouse parquet export.
type ClickHouseDownloadResult struct {
	Dir               string
	StartID           int64
	EndID             int64
	RemoteMaxID       int64
	RemoteCount       int64
	ChunkIDSpan       int64
	ChunksTotal       int
	ChunksDone        int
	ChunksSkipped     int
	BytesDone         int64
	BytesSkipped      int64
	FilesPruned       int
	TailRefreshed     int
	IncrementalFromID int64
	RemoteInfo        *ClickHouseRemoteInfo
}

func (c Config) ClickHouseInfo(ctx context.Context) (*ClickHouseRemoteInfo, error) {
	cfg := c.WithDefaults()
	fq := cfg.clickHouseFQTable()
	query := fmt.Sprintf(`SELECT toInt64(count()) AS c, toInt64(max(id)) AS max_id, toString(max(time)) AS max_time FROM %s FORMAT JSONEachRow`, fq)
	body, err := cfg.clickHouseQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	var row struct {
		Count   any    `json:"c"`
		MaxID   any    `json:"max_id"`
		MaxTime string `json:"max_time"`
	}
	if err := json.Unmarshal(body, &row); err != nil {
		return nil, fmt.Errorf("decode clickhouse metadata: %w", err)
	}
	count, err := parseInt64Any(row.Count)
	if err != nil {
		return nil, fmt.Errorf("parse clickhouse count: %w", err)
	}
	maxID, err := parseInt64Any(row.MaxID)
	if err != nil {
		return nil, fmt.Errorf("parse clickhouse max_id: %w", err)
	}
	return &ClickHouseRemoteInfo{
		BaseURL:   cfg.ClickHouseBaseURL,
		Database:  cfg.ClickHouseDatabase,
		Table:     cfg.ClickHouseTable,
		Count:     count,
		MaxID:     maxID,
		MaxTime:   row.MaxTime,
		CheckedAt: time.Now().UTC(),
		User:      cfg.ClickHouseUser,
	}, nil
}

func (c Config) DownloadClickHouseParquet(ctx context.Context, opts ClickHouseDownloadOptions, cb func(ClickHouseDownloadProgress)) (*ClickHouseDownloadResult, error) {
	cfg := c.WithDefaults()
	if err := cfg.EnsureRawDirs(); err != nil {
		return nil, fmt.Errorf("prepare directories: %w", err)
	}
	remote, err := cfg.ClickHouseInfo(ctx)
	if err != nil {
		return nil, err
	}
	span := opts.ChunkIDSpan
	if span <= 0 {
		span = 500_000
	}
	parallelism := opts.Parallelism
	if parallelism <= 0 {
		parallelism = 4
	}
	refreshTailChunks := opts.RefreshTailChunks
	if refreshTailChunks <= 0 {
		refreshTailChunks = 2
	}
	startID := opts.FromID
	if startID <= 0 {
		startID = 1
	}
	endID := opts.ToID
	if endID <= 0 || endID > remote.MaxID {
		endID = remote.MaxID
	}
	if endID < startID {
		return nil, fmt.Errorf("invalid id range: from=%d to=%d", startID, endID)
	}
	ranges := buildClickHouseChunkRanges(startID, endID, span, opts.AlignCheckpoints)
	chunksTotal := len(ranges)
	outDir := strings.TrimSpace(opts.OutputDir)
	if outDir == "" {
		outDir = cfg.ClickHouseParquetDir()
	}
	res := &ClickHouseDownloadResult{
		Dir:         outDir,
		StartID:     startID,
		EndID:       endID,
		RemoteMaxID: remote.MaxID,
		RemoteCount: remote.Count,
		ChunkIDSpan: span,
		ChunksTotal: chunksTotal,
		RemoteInfo:  remote,
	}
	res.IncrementalFromID = startID
	if !opts.Force {
		if tailFrom := tailRefreshStartID(startID, endID, span, refreshTailChunks); tailFrom > 0 {
			res.IncrementalFromID = tailFrom
		}
	}
	started := time.Now()
	type chunkTask struct {
		startID int64
		endID   int64
		path    string
	}
	var tasks []chunkTask
	var mu sync.Mutex
	active := 0

	existingChunks, _ := listLocalCHChunks(outDir)
	byStart := make(map[int64][]localChunkFile)
	for _, cf := range existingChunks {
		byStart[cf.StartID] = append(byStart[cf.StartID], cf)
	}
	refreshTailStart := int64(0)
	if !opts.Force {
		refreshTailStart = tailRefreshStartID(startID, endID, span, refreshTailChunks)
	}
	var prunePaths []string
	for start, files := range byStart {
		inRange := start >= startID && start <= endID
		if !inRange {
			continue
		}
		expectedEnd, alignedTarget := expectedCHChunkEnd(start, startID, endID, span)
		if !alignedTarget {
			for _, cf := range files {
				prunePaths = append(prunePaths, cf.Path)
			}
			continue
		}
		needRefresh := !opts.Force && refreshTailStart > 0 && start >= refreshTailStart
		var exact []localChunkFile
		for _, cf := range files {
			if needRefresh || cf.EndID != expectedEnd || cf.Size <= 0 {
				prunePaths = append(prunePaths, cf.Path)
				continue
			}
			exact = append(exact, cf)
		}
		if len(exact) > 1 {
			// Keep the largest exact file and prune duplicates for this start range.
			bestIdx := 0
			for i := 1; i < len(exact); i++ {
				if exact[i].Size > exact[bestIdx].Size || (exact[i].Size == exact[bestIdx].Size && exact[i].Path > exact[bestIdx].Path) {
					bestIdx = i
				}
			}
			for i, cf := range exact {
				if i == bestIdx {
					continue
				}
				prunePaths = append(prunePaths, cf.Path)
			}
		}
	}
	prunePaths = compactPathList(prunePaths)
	for _, p := range prunePaths {
		if err := os.Remove(p); err == nil || os.IsNotExist(err) {
			res.FilesPruned++
		}
	}
	if refreshTailStart > 0 {
		for s := refreshTailStart; s <= endID; s += span {
			res.TailRefreshed++
		}
	}

	emit := func(p ClickHouseDownloadProgress) {
		if cb != nil {
			cb(p)
		}
	}
	updateProgress := func(base ClickHouseDownloadProgress) {
		elapsed := time.Since(started)
		base.ChunksTotal = chunksTotal
		base.ChunksDone = res.ChunksDone
		base.ChunksSkipped = res.ChunksSkipped
		base.ChunksActive = active
		base.BytesDone = res.BytesDone
		base.BytesSkipped = res.BytesSkipped
		base.Elapsed = elapsed
		if elapsed > 0 {
			base.OverallSpeedBPS = float64(res.BytesDone) / elapsed.Seconds()
		}
		emit(base)
	}

	for _, r := range ranges {
		chunkStart, chunkEnd := r[0], r[1]
		path := filepath.Join(outDir, fmt.Sprintf("id_%09d_%09d.parquet", chunkStart, chunkEnd))
		if opts.Force {
			_ = os.Remove(path)
		}
		if !opts.Force && fileExistsNonEmpty(path) {
			sz, _ := fileSize(path)
			mu.Lock()
			res.ChunksDone++
			res.ChunksSkipped++
			res.BytesSkipped += sz
			updateProgress(ClickHouseDownloadProgress{
				ChunkStart: chunkStart,
				ChunkEnd:   chunkEnd,
				ChunkPath:  path,
				ChunkBytes: sz,
				Detail:     "skipped existing chunk",
			})
			mu.Unlock()
			continue
		}
		tasks = append(tasks, chunkTask{startID: chunkStart, endID: chunkEnd, path: path})
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(parallelism)
	for _, t := range tasks {
		t := t
		g.Go(func() error {
			startedChunk := time.Now()
			mu.Lock()
			active++
			mu.Unlock()

			bytesWritten, err := cfg.downloadClickHouseChunk(gctx, t.startID, t.endID, t.path)
			elapsedChunk := time.Since(startedChunk)

			mu.Lock()
			defer mu.Unlock()
			active--
			if err != nil {
				return err
			}
			res.ChunksDone++
			res.BytesDone += bytesWritten
			chSpeed := 0.0
			if elapsedChunk > 0 {
				chSpeed = float64(bytesWritten) / elapsedChunk.Seconds()
			}
			updateProgress(ClickHouseDownloadProgress{
				ChunkStart:    t.startID,
				ChunkEnd:      t.endID,
				ChunkPath:     t.path,
				ChunkBytes:    bytesWritten,
				ChunkElapsed:  elapsedChunk,
				ChunkSpeedBPS: chSpeed,
			})
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	if cb != nil {
		mu.Lock()
		updateProgress(ClickHouseDownloadProgress{Complete: true})
		mu.Unlock()
	}
	return res, nil
}

func (c Config) downloadClickHouseChunk(ctx context.Context, startID, endID int64, outPath string) (int64, error) {
	cfg := c.WithDefaults()
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, fmt.Errorf("create clickhouse dir: %w", err)
	}
	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)

	query := fmt.Sprintf(`SELECT * FROM %s WHERE id >= %d AND id <= %d ORDER BY id FORMAT Parquet`, cfg.clickHouseFQTable(), startID, endID)

	var lastErr error
	for attempt := 1; attempt <= 4; attempt++ {
		req, err := cfg.newClickHouseRequest(ctx, query)
		if err != nil {
			return 0, err
		}
		resp, err := cfg.clickHouseDownloadHTTPClient().Do(req)
		if err != nil {
			lastErr = fmt.Errorf("clickhouse chunk %d-%d request: %w", startID, endID, err)
		} else {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				f, ferr := os.Create(tmpPath)
				if ferr != nil {
					resp.Body.Close()
					return 0, fmt.Errorf("create chunk file: %w", ferr)
				}
				n, cerr := io.Copy(f, resp.Body)
				bodyCloseErr := resp.Body.Close()
				fileCloseErr := f.Close()
				if cerr == nil && bodyCloseErr == nil && fileCloseErr == nil {
					if err := os.Rename(tmpPath, outPath); err != nil {
						_ = os.Remove(tmpPath)
						return 0, fmt.Errorf("rename chunk file: %w", err)
					}
					return n, nil
				}
				_ = os.Remove(tmpPath)
				if cerr != nil {
					lastErr = fmt.Errorf("write clickhouse chunk %d-%d: %w", startID, endID, cerr)
				} else if bodyCloseErr != nil {
					lastErr = fmt.Errorf("close clickhouse response body %d-%d: %w", startID, endID, bodyCloseErr)
				} else {
					lastErr = fmt.Errorf("close clickhouse chunk file %d-%d: %w", startID, endID, fileCloseErr)
				}
			} else {
				b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
				resp.Body.Close()
				lastErr = fmt.Errorf("clickhouse chunk %d-%d returned %d: %s", startID, endID, resp.StatusCode, strings.TrimSpace(string(b)))
				if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
					return 0, lastErr
				}
			}
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("clickhouse chunk %d-%d failed", startID, endID)
	}
	return 0, lastErr
}

func buildClickHouseChunkRanges(startID, endID, span int64, alignCheckpoints bool) [][2]int64 {
	if startID <= 0 || endID <= 0 || endID < startID || span <= 0 {
		return nil
	}
	out := make([][2]int64, 0, int((endID-startID)/span)+2)
	if !alignCheckpoints {
		for s := startID; s <= endID; s += span {
			e := s + span - 1
			if e > endID {
				e = endID
			}
			out = append(out, [2]int64{s, e})
		}
		return out
	}
	for s := startID; s <= endID; {
		e := ((s-1)/span+1)*span
		if e > endID {
			e = endID
		}
		out = append(out, [2]int64{s, e})
		s = e + 1
	}
	return out
}

func (c Config) clickHouseQuery(ctx context.Context, query string) ([]byte, error) {
	cfg := c.WithDefaults()
	req, err := cfg.newClickHouseRequest(ctx, query)
	if err != nil {
		return nil, err
	}
	resp, err := cfg.clickHouseHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("clickhouse query request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read clickhouse query response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("clickhouse query returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (c Config) newClickHouseRequest(ctx context.Context, query string) (*http.Request, error) {
	cfg := c.WithDefaults()
	u, err := url.Parse(cfg.ClickHouseBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse clickhouse base URL: %w", err)
	}
	q := u.Query()
	if cfg.ClickHouseUser != "" {
		q.Set("user", cfg.ClickHouseUser)
	}
	if cfg.ClickHouseDatabase != "" {
		q.Set("database", cfg.ClickHouseDatabase)
	}
	q.Set("max_result_rows", "0")
	q.Set("max_result_bytes", "0")
	q.Set("result_overflow_mode", "throw")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("create clickhouse request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	return req, nil
}

func (c Config) clickHouseHTTPClient() *http.Client {
	cfg := c.WithDefaults()
	base := cfg.httpClient()
	if strings.TrimSpace(cfg.ClickHouseDNSServer) == "" {
		return base
	}

	clone := *base
	var tr *http.Transport
	if bt, ok := base.Transport.(*http.Transport); ok && bt != nil {
		tr = bt.Clone()
	} else {
		tr = http.DefaultTransport.(*http.Transport).Clone()
	}
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := &net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", cfg.ClickHouseDNSServer)
		},
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  resolver,
	}
	tr.DialContext = dialer.DialContext
	clone.Transport = tr
	return &clone
}

func (c Config) clickHouseDownloadHTTPClient() *http.Client {
	base := c.clickHouseHTTPClient()
	clone := *base
	// Chunked parquet exports can stream for minutes; avoid aborting mid-body.
	clone.Timeout = 0
	return &clone
}

func (c Config) clickHouseFQTable() string {
	cfg := c.WithDefaults()
	return quoteCHIdent(cfg.ClickHouseDatabase) + "." + quoteCHIdent(cfg.ClickHouseTable)
}

func quoteCHIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func parseInt64Any(v any) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	case json.Number:
		return x.Int64()
	case string:
		return strconv.ParseInt(strings.TrimSpace(x), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}
