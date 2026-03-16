package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/arctic"
	"github.com/spf13/cobra"
)

func newArcticBench() *cobra.Command {
	var (
		zstPath    string
		typ        string
		engines    string
		workerList string
		chunkLines int
	)

	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Benchmark processing pipeline on a single .zst file",
		Long: `Runs ProcessZst on a single .zst file with different engine/worker
combinations and outputs a timing comparison table.

Copy a .zst file to a bench directory first:
  cp /root/data/arctic/raw/reddit/comments/RC_2011-01.zst /root/data/arctic/bench/

Then benchmark:
  search arctic bench --zst /root/data/arctic/bench/RC_2011-01.zst --type comments`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zstPath == "" {
				return fmt.Errorf("--zst is required")
			}
			if typ != "comments" && typ != "submissions" {
				return fmt.Errorf("--type must be 'comments' or 'submissions'")
			}

			fi, err := os.Stat(zstPath)
			if err != nil {
				return fmt.Errorf("stat %s: %w", zstPath, err)
			}
			fmt.Printf("Benchmarking %s (%s, %.1f MB)\n\n", filepath.Base(zstPath), typ,
				float64(fi.Size())/(1024*1024))

			// Parse engine and worker lists.
			engineList := strings.Split(engines, ",")
			var workers []int
			for _, w := range strings.Split(workerList, ",") {
				n, err := strconv.Atoi(strings.TrimSpace(w))
				if err != nil {
					return fmt.Errorf("invalid worker count %q: %w", w, err)
				}
				workers = append(workers, n)
			}

			type result struct {
				engine  string
				workers int
				chunks  int
				dur     time.Duration
				rows    int64
				size    int64
			}
			var results []result

			for _, eng := range engineList {
				for _, w := range workers {
					// Create isolated work dir per run.
					workDir, err := os.MkdirTemp("", fmt.Sprintf("arctic-bench-%s-%d-", eng, w))
					if err != nil {
						return err
					}
					defer os.RemoveAll(workDir)

					cfg := arctic.Config{
						WorkDir:           workDir,
						ChunkLines:        chunkLines,
						Engine:            strings.TrimSpace(eng),
						MaxConvertWorkers: w,
						DuckDBMemoryMB:    512,
					}

					// Extract year/mm from filename: RC_YYYY-MM.zst or RS_YYYY-MM.zst
					base := filepath.Base(zstPath)
					stem := strings.TrimSuffix(base, ".zst")
					parts := strings.SplitN(stem, "_", 2)
					if len(parts) != 2 || len(parts[1]) != 7 {
						return fmt.Errorf("unexpected filename format: %s (expected RC_YYYY-MM.zst)", base)
					}
					ym := parts[1]
					year := ym[:4]
					mm := ym[5:]

					fmt.Printf("%-8s  workers=%-2d  ", strings.TrimSpace(eng), w)

					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
					var lastShard int
					start := time.Now()
					pr, err := arctic.ProcessZst(ctx, cfg, zstPath, typ, year, mm,
						func(sr arctic.ShardResult) {
							if !sr.Starting {
								lastShard = sr.Index + 1
							}
						})
					cancel()

					if err != nil {
						fmt.Printf("ERROR: %v\n", err)
						continue
					}

					dur := time.Since(start)
					fmt.Printf("chunks=%-3d  %8s  %d rows  %.1f MB  %d rows/s\n",
						lastShard, dur.Round(100*time.Millisecond),
						pr.TotalRows, float64(pr.TotalSize)/(1024*1024),
						int64(float64(pr.TotalRows)/dur.Seconds()))

					results = append(results, result{
						engine:  strings.TrimSpace(eng),
						workers: w,
						chunks:  lastShard,
						dur:     dur,
						rows:    pr.TotalRows,
						size:    pr.TotalSize,
					})

					// Clean up parquet shards between runs.
					os.RemoveAll(workDir)
				}
			}

			// Summary table.
			fmt.Printf("\n%-8s  %7s  %6s  %10s  %10s  %10s  %10s\n",
				"Engine", "Workers", "Chunks", "Time", "Rows", "Size", "Rows/s")
			fmt.Println(strings.Repeat("-", 72))
			for _, r := range results {
				fmt.Printf("%-8s  %7d  %6d  %10s  %10d  %8.1f MB  %10d\n",
					r.engine, r.workers, r.chunks,
					r.dur.Round(100*time.Millisecond),
					r.rows,
					float64(r.size)/(1024*1024),
					int64(float64(r.rows)/r.dur.Seconds()))
			}

			// Validate row counts match.
			if len(results) > 1 {
				ref := results[0].rows
				allMatch := true
				for _, r := range results[1:] {
					if r.rows != ref {
						fmt.Printf("\nWARNING: row count mismatch: %s/%d=%d vs %s/%d=%d\n",
							results[0].engine, results[0].workers, ref,
							r.engine, r.workers, r.rows)
						allMatch = false
					}
				}
				if allMatch {
					fmt.Printf("\nAll row counts match: %d\n", ref)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&zstPath, "zst", "", "Path to .zst file to benchmark")
	cmd.Flags().StringVar(&typ, "type", "comments", "Type: 'comments' or 'submissions'")
	cmd.Flags().StringVar(&engines, "engines", "duckdb", "Comma-separated engines to test (duckdb,go)")
	cmd.Flags().StringVar(&workerList, "workers", "1,3,6", "Comma-separated worker counts to test")
	cmd.Flags().IntVar(&chunkLines, "chunk-lines", 500_000, "Lines per chunk")

	return cmd
}
