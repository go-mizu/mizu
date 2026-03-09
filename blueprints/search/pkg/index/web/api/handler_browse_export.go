package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	mizu "github.com/go-mizu/mizu"
)

func handleBrowseExportParquet(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Docs == nil {
			return c.JSON(503, errResp{"doc store not available"})
		}

		var req BrowseExportParquetRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(400, errResp{"invalid JSON: " + err.Error()})
		}
		shard := filepath.Base(req.Shard)
		if shard == "" || shard == "." || shard == "/" {
			return c.JSON(400, errResp{"shard required"})
		}

		includeMarkdownBody := true
		if req.IncludeMarkdownBody != nil {
			includeMarkdownBody = *req.IncludeMarkdownBody
		}

		metaPath := filepath.Join(d.WARCMdBase, shard+".meta.duckdb")
		if _, err := os.Stat(metaPath); err != nil {
			return c.JSON(404, errResp{"shard metadata not found"})
		}
		warcMdPath := filepath.Join(d.WARCMdBase, shard+".md.warc.gz")
		if includeMarkdownBody {
			if _, err := os.Stat(warcMdPath); err != nil {
				return c.JSON(404, errResp{"shard markdown WARC not found"})
			}
		}

		outBase := d.CrawlDir
		if outBase == "" {
			outBase = filepath.Dir(d.WARCMdBase)
		}
		outDir := filepath.Join(outBase, "pack", "parquet_export")
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return c.JSON(500, errResp{"mkdir output: " + err.Error()})
		}

		outName := shard + ".meta.parquet"
		if includeMarkdownBody {
			outName = shard + ".meta.with_body.parquet"
		}
		outPath := filepath.Join(outDir, outName)
		if !req.Overwrite {
			if _, err := os.Stat(outPath); err == nil {
				return c.JSON(409, errResp{"output already exists; pass overwrite=true"})
			}
		}

		if d.ExportShardMeta == nil {
			return c.JSON(503, errResp{"export not available"})
		}

		start := time.Now()
		rows, err := d.ExportShardMeta(c.Context(), metaPath, warcMdPath, outPath, includeMarkdownBody)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		info, _ := os.Stat(outPath)
		var sz int64
		if info != nil {
			sz = info.Size()
		}
		return c.JSON(200, BrowseExportParquetResponse{
			Shard:               shard,
			IncludeMarkdownBody: includeMarkdownBody,
			OutputPath:          outPath,
			Rows:                rows,
			SizeBytes:           sz,
			ElapsedMs:           time.Since(start).Milliseconds(),
		})
	}
}
