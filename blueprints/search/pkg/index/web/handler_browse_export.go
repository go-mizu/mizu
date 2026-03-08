package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/parquet-go/parquet-go"
)

type browseExportParquetRequest struct {
	Shard               string `json:"shard"`
	IncludeMarkdownBody bool   `json:"include_markdown_body"`
	Overwrite           bool   `json:"overwrite,omitempty"`
}

type browseExportParquetResponse struct {
	Shard               string `json:"shard"`
	IncludeMarkdownBody bool   `json:"include_markdown_body"`
	OutputPath          string `json:"output_path"`
	Rows                int64  `json:"rows"`
	SizeBytes           int64  `json:"size_bytes"`
	ElapsedMs           int64  `json:"elapsed_ms"`
}

type browseExportRow struct {
	DocID        string `parquet:"doc_id"`
	URL          string `parquet:"url"`
	Host         string `parquet:"host"`
	Title        string `parquet:"title"`
	CrawlDate    string `parquet:"crawl_date"`
	SizeBytes    int64  `parquet:"size_bytes"`
	WordCount    int32  `parquet:"word_count"`
	WARCRecordID string `parquet:"warc_record_id"`
	RefersTo     string `parquet:"refers_to"`
	GzipOffset   int64  `parquet:"gzip_offset"`
	GzipSize     int64  `parquet:"gzip_size"`
}

type browseExportRowWithBody struct {
	DocID        string `parquet:"doc_id"`
	URL          string `parquet:"url"`
	Host         string `parquet:"host"`
	Title        string `parquet:"title"`
	CrawlDate    string `parquet:"crawl_date"`
	SizeBytes    int64  `parquet:"size_bytes"`
	WordCount    int32  `parquet:"word_count"`
	WARCRecordID string `parquet:"warc_record_id"`
	RefersTo     string `parquet:"refers_to"`
	GzipOffset   int64  `parquet:"gzip_offset"`
	GzipSize     int64  `parquet:"gzip_size"`
	MarkdownBody string `parquet:"markdown_body"`
}

func (s *Server) handleBrowseExportParquet(c *mizu.Ctx) error {
	if s.Docs == nil {
		return c.JSON(503, errResp{"doc store not available"})
	}

	var req browseExportParquetRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errResp{"invalid JSON: " + err.Error()})
	}
	shard := filepath.Base(req.Shard)
	if shard == "" || shard == "." || shard == "/" {
		return c.JSON(400, errResp{"shard required"})
	}

	metaPath := filepath.Join(s.WARCMdBase, shard+".meta.duckdb")
	if _, err := os.Stat(metaPath); err != nil {
		return c.JSON(404, errResp{"shard metadata not found"})
	}
	warcMdPath := filepath.Join(s.WARCMdBase, shard+".md.warc.gz")
	if req.IncludeMarkdownBody {
		if _, err := os.Stat(warcMdPath); err != nil {
			return c.JSON(404, errResp{"shard markdown WARC not found"})
		}
	}

	outBase := s.CrawlDir
	if outBase == "" {
		outBase = filepath.Dir(s.WARCMdBase)
	}
	outDir := filepath.Join(outBase, "pack", "parquet_export")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return c.JSON(500, errResp{"mkdir output: " + err.Error()})
	}

	outName := shard + ".meta.parquet"
	if req.IncludeMarkdownBody {
		outName = shard + ".meta.with_body.parquet"
	}
	outPath := filepath.Join(outDir, outName)
	if !req.Overwrite {
		if _, err := os.Stat(outPath); err == nil {
			return c.JSON(409, errResp{"output already exists; pass overwrite=true"})
		}
	}

	start := time.Now()
	rows, err := exportShardMetaParquet(c.Context(), metaPath, warcMdPath, outPath, req.IncludeMarkdownBody)
	if err != nil {
		return c.JSON(500, errResp{err.Error()})
	}
	info, _ := os.Stat(outPath)
	var sz int64
	if info != nil {
		sz = info.Size()
	}
	return c.JSON(200, browseExportParquetResponse{
		Shard:               shard,
		IncludeMarkdownBody: req.IncludeMarkdownBody,
		OutputPath:          outPath,
		Rows:                rows,
		SizeBytes:           sz,
		ElapsedMs:           time.Since(start).Milliseconds(),
	})
}

func exportShardMetaParquet(ctx context.Context, metaPath, warcMdPath, outPath string, includeMarkdownBody bool) (int64, error) {
	db, err := openShardDB(metaPath + "?access_mode=read_only")
	if err != nil {
		return 0, fmt.Errorf("open meta duckdb: %w", err)
	}
	defer db.Close()

	q := `
		SELECT doc_id, url, host, title, crawl_date, size_bytes, word_count,
		       warc_record_id, refers_to, gzip_offset, gzip_size
		FROM doc_records
		ORDER BY crawl_date DESC
	`
	rs, err := db.QueryContext(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("query doc_records: %w", err)
	}
	defer rs.Close()

	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)
	f, err := os.Create(tmpPath)
	if err != nil {
		return 0, fmt.Errorf("create parquet: %w", err)
	}

	var rowsWritten int64
	if includeMarkdownBody {
		pw := parquet.NewGenericWriter[browseExportRowWithBody](f, parquet.Compression(&parquet.Zstd))
		batch := make([]browseExportRowWithBody, 0, 1000)
		flush := func() error {
			if len(batch) == 0 {
				return nil
			}
			if _, err := pw.Write(batch); err != nil {
				return err
			}
			batch = batch[:0]
			return nil
		}
		for rs.Next() {
			r, err := scanExportRow(rs)
			if err != nil {
				pw.Close()
				f.Close()
				_ = os.Remove(tmpPath)
				return 0, err
			}
			body, readErr := ReadDocByOffset(warcMdPath, r.GzipOffset, r.GzipSize)
			if readErr != nil || len(body) == 0 {
				body, _, readErr = readDocFromWARCMd(warcMdPath, r.DocID)
			}
			if readErr != nil {
				pw.Close()
				f.Close()
				_ = os.Remove(tmpPath)
				return 0, fmt.Errorf("read markdown body doc=%s: %w", r.DocID, readErr)
			}
			batch = append(batch, browseExportRowWithBody{
				DocID:        r.DocID,
				URL:          r.URL,
				Host:         r.Host,
				Title:        r.Title,
				CrawlDate:    r.CrawlDate,
				SizeBytes:    r.SizeBytes,
				WordCount:    r.WordCount,
				WARCRecordID: r.WARCRecordID,
				RefersTo:     r.RefersTo,
				GzipOffset:   r.GzipOffset,
				GzipSize:     r.GzipSize,
				MarkdownBody: sanitizeUTF8(string(body)),
			})
			rowsWritten++
			if len(batch) >= 1000 {
				if err := flush(); err != nil {
					pw.Close()
					f.Close()
					_ = os.Remove(tmpPath)
					return 0, fmt.Errorf("write parquet: %w", err)
				}
			}
		}
		if err := rs.Err(); err != nil {
			pw.Close()
			f.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("scan rows: %w", err)
		}
		if err := flush(); err != nil {
			pw.Close()
			f.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("write parquet: %w", err)
		}
		if err := pw.Close(); err != nil {
			f.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("close parquet: %w", err)
		}
	} else {
		pw := parquet.NewGenericWriter[browseExportRow](f, parquet.Compression(&parquet.Zstd))
		batch := make([]browseExportRow, 0, 1000)
		flush := func() error {
			if len(batch) == 0 {
				return nil
			}
			if _, err := pw.Write(batch); err != nil {
				return err
			}
			batch = batch[:0]
			return nil
		}
		for rs.Next() {
			r, err := scanExportRow(rs)
			if err != nil {
				pw.Close()
				f.Close()
				_ = os.Remove(tmpPath)
				return 0, err
			}
			batch = append(batch, r)
			rowsWritten++
			if len(batch) >= 1000 {
				if err := flush(); err != nil {
					pw.Close()
					f.Close()
					_ = os.Remove(tmpPath)
					return 0, fmt.Errorf("write parquet: %w", err)
				}
			}
		}
		if err := rs.Err(); err != nil {
			pw.Close()
			f.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("scan rows: %w", err)
		}
		if err := flush(); err != nil {
			pw.Close()
			f.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("write parquet: %w", err)
		}
		if err := pw.Close(); err != nil {
			f.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("close parquet: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("close file: %w", err)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("finalize parquet: %w", err)
	}
	return rowsWritten, nil
}

func scanExportRow(rs *sql.Rows) (browseExportRow, error) {
	var r browseExportRow
	if err := rs.Scan(
		&r.DocID,
		&r.URL,
		&r.Host,
		&r.Title,
		&r.CrawlDate,
		&r.SizeBytes,
		&r.WordCount,
		&r.WARCRecordID,
		&r.RefersTo,
		&r.GzipOffset,
		&r.GzipSize,
	); err != nil {
		return browseExportRow{}, fmt.Errorf("scan row: %w", err)
	}
	return r, nil
}
