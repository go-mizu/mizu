package hn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// APIDownloadOptions controls the chunked Hacker News API fallback downloader.
type APIDownloadOptions struct {
	Workers   int
	ChunkSize int
	FromID    int64
	ToID      int64 // 0 means discover via /maxitem.json
	Force     bool
}

// APIDownloadProgress reports chunk-level progress for API fallback downloads.
type APIDownloadProgress struct {
	ChunkStart    int64
	ChunkEnd      int64
	ChunkPath     string
	ChunksTotal   int
	ChunksDone    int
	ChunksSkipped int
	IDsProcessed  int64
	ItemsWritten  int64
	MaxItem       int64
	Elapsed       time.Duration
	Complete      bool
	Detail        string
}

// APIDownloadResult describes the output of an API fallback download.
type APIDownloadResult struct {
	Dir           string
	StartID       int64
	EndID         int64
	MaxItem       int64
	ChunksTotal   int
	ChunksDone    int
	ChunksSkipped int
	IDsProcessed  int64
	ItemsWritten  int64
}

func (c Config) DownloadAPI(ctx context.Context, opts APIDownloadOptions, cb func(APIDownloadProgress)) (*APIDownloadResult, error) {
	cfg := c.WithDefaults()
	if err := cfg.EnsureRawDirs(); err != nil {
		return nil, fmt.Errorf("prepare directories: %w", err)
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = 32
	}
	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	startID := opts.FromID
	if startID <= 0 {
		startID = 1
	}
	endID := opts.ToID
	maxItem := opts.ToID
	if endID <= 0 {
		var err error
		maxItem, err = cfg.GetMaxItem(ctx)
		if err != nil {
			return nil, err
		}
		endID = maxItem
	}
	if endID < startID {
		return nil, fmt.Errorf("invalid id range: from=%d to=%d", startID, endID)
	}

	chunksTotal := int((endID-startID)/int64(chunkSize) + 1)
	res := &APIDownloadResult{
		Dir:         cfg.APIChunksDir(),
		StartID:     startID,
		EndID:       endID,
		MaxItem:     maxItem,
		ChunksTotal: chunksTotal,
	}
	started := time.Now()

	for chunkStart := startID; chunkStart <= endID; chunkStart += int64(chunkSize) {
		chunkEnd := chunkStart + int64(chunkSize) - 1
		if chunkEnd > endID {
			chunkEnd = endID
		}
		path := filepath.Join(cfg.APIChunksDir(), chunkFileName(chunkStart, chunkEnd))
		if opts.Force {
			_ = os.Remove(path)
		}
		if !opts.Force && fileExistsNonEmpty(path) {
			res.ChunksSkipped++
			res.ChunksDone++
			res.IDsProcessed += (chunkEnd - chunkStart + 1)
			if cb != nil {
				cb(APIDownloadProgress{
					ChunkStart:    chunkStart,
					ChunkEnd:      chunkEnd,
					ChunkPath:     path,
					ChunksTotal:   chunksTotal,
					ChunksDone:    res.ChunksDone,
					ChunksSkipped: res.ChunksSkipped,
					IDsProcessed:  res.IDsProcessed,
					ItemsWritten:  res.ItemsWritten,
					MaxItem:       maxItem,
					Elapsed:       time.Since(started),
					Detail:        "skipped existing chunk",
				})
			}
			continue
		}

		itemsWritten, idsProcessed, err := cfg.downloadAPIChunk(ctx, chunkStart, chunkEnd, workers, path)
		if err != nil {
			return nil, err
		}
		res.ChunksDone++
		res.IDsProcessed += idsProcessed
		res.ItemsWritten += itemsWritten
		if cb != nil {
			cb(APIDownloadProgress{
				ChunkStart:    chunkStart,
				ChunkEnd:      chunkEnd,
				ChunkPath:     path,
				ChunksTotal:   chunksTotal,
				ChunksDone:    res.ChunksDone,
				ChunksSkipped: res.ChunksSkipped,
				IDsProcessed:  res.IDsProcessed,
				ItemsWritten:  res.ItemsWritten,
				MaxItem:       maxItem,
				Elapsed:       time.Since(started),
			})
		}
	}

	if cb != nil {
		cb(APIDownloadProgress{
			ChunksTotal:   chunksTotal,
			ChunksDone:    res.ChunksDone,
			ChunksSkipped: res.ChunksSkipped,
			IDsProcessed:  res.IDsProcessed,
			ItemsWritten:  res.ItemsWritten,
			MaxItem:       maxItem,
			Elapsed:       time.Since(started),
			Complete:      true,
		})
	}
	return res, nil
}

func (c Config) downloadAPIChunk(ctx context.Context, startID, endID int64, workers int, outPath string) (itemsWritten int64, idsProcessed int64, err error) {
	type job struct {
		idx int
		id  int64
	}
	type result struct {
		idx  int
		body []byte
		err  error
	}

	n := int(endID-startID) + 1
	jobs := make(chan job)
	results := make(chan result, n)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				body, ferr := c.fetchHNItemRaw(ctx, j.id)
				select {
				case results <- result{idx: j.idx, body: body, err: ferr}:
				case <-ctx.Done():
					return
				}
				if ferr != nil {
					return
				}
			}
		}()
	}

	go func() {
		for i := 0; i < n; i++ {
			select {
			case <-ctx.Done():
				close(jobs)
				wg.Wait()
				close(results)
				return
			case jobs <- job{idx: i, id: startID + int64(i)}:
			}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	bodies := make([][]byte, n)
	var firstErr error
	for r := range results {
		if r.err != nil && firstErr == nil {
			firstErr = r.err
			cancel()
		}
		if r.err == nil {
			bodies[r.idx] = r.body
		}
		idsProcessed++
	}
	if firstErr != nil {
		return 0, idsProcessed, firstErr
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, idsProcessed, fmt.Errorf("create chunk dir: %w", err)
	}
	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)
	f, err := os.Create(tmpPath)
	if err != nil {
		return 0, idsProcessed, fmt.Errorf("create chunk file: %w", err)
	}
	for _, b := range bodies {
		if len(b) == 0 {
			continue // null item or missing
		}
		if _, err := f.Write(b); err != nil {
			f.Close()
			_ = os.Remove(tmpPath)
			return 0, idsProcessed, fmt.Errorf("write chunk: %w", err)
		}
		if _, err := f.Write([]byte{'\n'}); err != nil {
			f.Close()
			_ = os.Remove(tmpPath)
			return 0, idsProcessed, fmt.Errorf("write chunk newline: %w", err)
		}
		itemsWritten++
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return 0, idsProcessed, fmt.Errorf("close chunk file: %w", err)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return 0, idsProcessed, fmt.Errorf("rename chunk file: %w", err)
	}
	return itemsWritten, idsProcessed, nil
}

func (c Config) fetchHNItemRaw(ctx context.Context, id int64) ([]byte, error) {
	cfg := c.WithDefaults()
	url := strings.TrimRight(cfg.APIBaseURL, "/") + fmt.Sprintf("/item/%d.json", id)

	var lastErr error
	for attempt := 1; attempt <= 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create item request %d: %w", id, err)
		}
		resp, err := cfg.httpClient().Do(req)
		if err != nil {
			lastErr = fmt.Errorf("GET item %d: %w", id, err)
		} else {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && readErr == nil {
				trimmed := bytes.TrimSpace(body)
				if bytes.Equal(trimmed, []byte("null")) || len(trimmed) == 0 {
					return nil, nil
				}
				// Validate JSON so broken responses fail fast.
				var js json.RawMessage
				if err := json.Unmarshal(trimmed, &js); err != nil {
					lastErr = fmt.Errorf("decode item %d: %w", id, err)
				} else {
					return trimmed, nil
				}
			} else if readErr != nil {
				lastErr = fmt.Errorf("read item %d: %w", id, readErr)
			} else if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
				lastErr = fmt.Errorf("GET item %d returned %d", id, resp.StatusCode)
			} else {
				return nil, fmt.Errorf("GET item %d returned %d", id, resp.StatusCode)
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("GET item %d failed", id)
	}
	return nil, lastErr
}
