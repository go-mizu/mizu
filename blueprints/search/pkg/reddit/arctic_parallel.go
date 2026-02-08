package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// ParallelDownload downloads from the Arctic Shift API using parallel month-based
// partitions for maximum throughput. Each month gets its own JSONL file,
// downloaded concurrently.
func (c *ArcticClient) ParallelDownload(ctx context.Context, target ArcticTarget, kind FileKind,
	afterEpoch int64, beforeEpoch int64, workers int, cb ArcticProgressCallback) error {

	if afterEpoch < 1104537600 {
		afterEpoch = 1104537600 // 2005-01-01
	}
	if beforeEpoch <= 0 {
		beforeEpoch = time.Now().Unix()
	}

	// Create partition directory
	partDir := target.PartitionDir(kind)
	os.MkdirAll(partDir, 0o755)

	// Split into monthly chunks
	startTime := time.Unix(afterEpoch, 0).UTC()
	endTime := time.Unix(beforeEpoch, 0).UTC()

	type monthChunk struct {
		label  string // e.g. "2024-01"
		after  int64
		before int64
		file   string
	}

	var chunks []monthChunk
	for y := startTime.Year(); y <= endTime.Year(); y++ {
		startMonth := time.January
		endMonth := time.December
		if y == startTime.Year() {
			startMonth = startTime.Month()
		}
		if y == endTime.Year() {
			endMonth = endTime.Month()
		}
		for m := startMonth; m <= endMonth; m++ {
			chunkAfter := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).Unix()
			chunkBefore := time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC).Unix()

			if chunkAfter < afterEpoch {
				chunkAfter = afterEpoch
			}
			if chunkBefore > beforeEpoch {
				chunkBefore = beforeEpoch
			}
			if chunkAfter >= chunkBefore {
				continue
			}

			label := fmt.Sprintf("%d-%02d", y, m)
			outFile := filepath.Join(partDir, label+".jsonl")
			chunks = append(chunks, monthChunk{label: label, after: chunkAfter, before: chunkBefore, file: outFile})
		}
	}

	if len(chunks) == 0 {
		return nil
	}

	if workers <= 0 {
		workers = 8
	}
	if workers > len(chunks) {
		workers = len(chunks)
	}

	// Build API params
	endpoint := "posts"
	if kind == Comments {
		endpoint = "comments"
	}
	param := "subreddit"
	if target.Kind == "user" {
		param = "author"
	}
	fields := commentFields
	if kind == Submissions {
		fields = submissionFields
	}

	// Per-worker HTTP client with shorter timeout to avoid hanging
	makeClient := func() *http.Client {
		return &http.Client{Timeout: 30 * time.Second}
	}

	start := time.Now()
	var totalItems atomic.Int64
	var totalBytes atomic.Int64
	var chunksDone atomic.Int32

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

	for _, chunk := range chunks {
		chunk := chunk
		g.Go(func() error {
			hc := makeClient()

			// Skip if partition already downloaded
			if st, err := os.Stat(chunk.file); err == nil && st.Size() > 0 {
				chunksDone.Add(1)
				return nil
			}

			f, err := os.Create(chunk.file)
			if err != nil {
				return fmt.Errorf("%s: create file: %w", chunk.label, err)
			}
			defer f.Close()

			currentAfter := chunk.after
			retries := 0

			for {
				select {
				case <-gctx.Done():
					return gctx.Err()
				default:
				}

				url := fmt.Sprintf("%s/api/%s/search?%s=%s&limit=auto&sort=asc&fields=%s&after=%d&before=%d",
					c.baseURL, endpoint, param, target.Name, fields, currentAfter, chunk.before)

				req, err := http.NewRequestWithContext(gctx, "GET", url, nil)
				if err != nil {
					return fmt.Errorf("%s: %w", chunk.label, err)
				}

				resp, err := hc.Do(req)
				if err != nil {
					retries++
					if retries > 10 {
						return fmt.Errorf("%s: max retries: %w", chunk.label, err)
					}
					backoff := min(time.Duration(1<<uint(retries-1))*time.Second, 30*time.Second)
					select {
					case <-time.After(backoff):
					case <-gctx.Done():
						return gctx.Err()
					}
					continue
				}

				if resp.StatusCode == 429 {
					resp.Body.Close()
					select {
					case <-time.After(30 * time.Second):
					case <-gctx.Done():
						return gctx.Err()
					}
					continue
				}

				if resp.StatusCode != 200 {
					resp.Body.Close()
					retries++
					if retries > 10 {
						return fmt.Errorf("%s: HTTP %d after retries", chunk.label, resp.StatusCode)
					}
					backoff := min(time.Duration(1<<uint(retries-1))*time.Second, 30*time.Second)
					select {
					case <-time.After(backoff):
					case <-gctx.Done():
						return gctx.Err()
					}
					continue
				}

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					retries++
					if retries > 10 {
						return fmt.Errorf("%s: read body: %w", chunk.label, err)
					}
					continue
				}

				retries = 0

				var result struct {
					Data []json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal(body, &result); err != nil {
					return fmt.Errorf("%s: decode: %w", chunk.label, err)
				}

				if len(result.Data) == 0 {
					break // Chunk done
				}

				// Write JSONL
				var batchBytes int64
				for _, item := range result.Data {
					line := append(item, '\n')
					n, err := f.Write(line)
					if err != nil {
						return fmt.Errorf("%s: write: %w", chunk.label, err)
					}
					batchBytes += int64(n)
				}

				totalItems.Add(int64(len(result.Data)))
				totalBytes.Add(batchBytes)

				// Pagination: advance after cursor
				var lastItem struct {
					CreatedUTC json.Number `json:"created_utc"`
				}
				if err := json.Unmarshal(result.Data[len(result.Data)-1], &lastItem); err != nil {
					return fmt.Errorf("%s: parse cursor: %w", chunk.label, err)
				}

				lastEpoch, err := lastItem.CreatedUTC.Int64()
				if err != nil {
					f64, err2 := lastItem.CreatedUTC.Float64()
					if err2 != nil {
						return fmt.Errorf("%s: parse epoch: %w", chunk.label, err)
					}
					lastEpoch = int64(f64)
				}

				if lastEpoch == currentAfter {
					lastEpoch++
				}
				currentAfter = lastEpoch

				// Progress callback
				if cb != nil {
					cb(ArcticProgress{
						Kind:      kind,
						Items:     totalItems.Load(),
						Bytes:     totalBytes.Load(),
						BatchSize: len(result.Data),
						Elapsed:   time.Since(start),
					})
				}
			}

			chunksDone.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Concatenate partition files into single JSONL
	jsonlPath := target.JSONLPath(kind)
	outFile, err := os.Create(jsonlPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	for _, chunk := range chunks {
		pf, err := os.Open(chunk.file)
		if err != nil {
			continue
		}
		io.Copy(outFile, pf)
		pf.Close()
	}

	// Final progress
	if cb != nil {
		cb(ArcticProgress{
			Kind:    kind,
			Items:   totalItems.Load(),
			Bytes:   totalBytes.Load(),
			Done:    true,
			Elapsed: time.Since(start),
		})
	}

	return nil
}
