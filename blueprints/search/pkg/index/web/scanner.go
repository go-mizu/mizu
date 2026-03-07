package web

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DataSummary holds statistics about a crawl's local data directory.
// All map fields are always initialized (never nil) so that JSON
// marshaling produces {} rather than null for empty maps.
type DataSummary struct {
	CrawlID       string           `json:"crawl_id"`
	WARCCount     int              `json:"warc_count"`
	WARCTotalSize int64            `json:"warc_total_size"`
	MDShards      int              `json:"md_shards"`
	MDTotalSize   int64            `json:"md_total_size"`
	MDDocEstimate int              `json:"md_doc_estimate"`
	PackFormats   map[string]int64 `json:"pack_formats"`
	FTSEngines    map[string]int64 `json:"fts_engines"`
	FTSShardCount map[string]int   `json:"fts_shard_count"`
}

// ScanDataDir walks a crawl data directory and computes summary statistics
// without opening DuckDB files. The expected layout is:
//
//	{crawlDir}/warc/                           WARC files
//	{crawlDir}/warc_md/{shardIdx}.md.warc.gz   Extracted markdown (WARC format)
//	{crawlDir}/pack/{format}/*                 Packed bundles per format
//	{crawlDir}/fts/{engine}/{shardIdx}/        FTS index shards per engine
func ScanDataDir(crawlDir string) DataSummary {
	ds := DataSummary{
		PackFormats:   make(map[string]int64),
		FTSEngines:    make(map[string]int64),
		FTSShardCount: make(map[string]int),
	}

	scanWARC(crawlDir, &ds)
	scanWARCMdDir(crawlDir, &ds)
	scanPack(crawlDir, &ds)
	scanFTS(crawlDir, &ds)

	return ds
}

// scanWARC counts files and sums sizes in {crawlDir}/warc/.
func scanWARC(crawlDir string, ds *DataSummary) {
	warcDir := filepath.Join(crawlDir, "warc")
	entries, err := os.ReadDir(warcDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ds.WARCCount++
		if info, err := e.Info(); err == nil {
			ds.WARCTotalSize += info.Size()
		}
	}
}

// scanWARCMdDir counts shards and sums sizes in {crawlDir}/warc_md/*.md.warc.gz.
func scanWARCMdDir(crawlDir string, ds *DataSummary) {
	mdDir := filepath.Join(crawlDir, "warc_md")
	entries, err := os.ReadDir(mdDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
			continue
		}
		ds.MDShards++
		if info, err := e.Info(); err == nil {
			ds.MDTotalSize += info.Size()
		}
	}
}

// scanPack sums sizes per format in {crawlDir}/pack/{format}/.
func scanPack(crawlDir string, ds *DataSummary) {
	packDir := filepath.Join(crawlDir, "pack")
	formats, err := os.ReadDir(packDir)
	if err != nil {
		return
	}
	for _, fmtEntry := range formats {
		if !fmtEntry.IsDir() {
			continue
		}
		fmtName := fmtEntry.Name()
		fmtPath := filepath.Join(packDir, fmtName)
		filepath.WalkDir(fmtPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if info, err := d.Info(); err == nil {
				ds.PackFormats[fmtName] += info.Size()
			}
			return nil
		})
	}
}

// scanFTS sums sizes and counts shards per engine in
// {crawlDir}/fts/{engine}/{shardIdx}/.
func scanFTS(crawlDir string, ds *DataSummary) {
	ftsDir := filepath.Join(crawlDir, "fts")
	engines, err := os.ReadDir(ftsDir)
	if err != nil {
		return
	}
	for _, engEntry := range engines {
		if !engEntry.IsDir() {
			continue
		}
		engName := engEntry.Name()
		engPath := filepath.Join(ftsDir, engName)

		shards, err := os.ReadDir(engPath)
		if err != nil {
			continue
		}
		for _, shard := range shards {
			if !shard.IsDir() {
				continue
			}
			ds.FTSShardCount[engName]++
			shardPath := filepath.Join(engPath, shard.Name())
			filepath.WalkDir(shardPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if info, err := d.Info(); err == nil {
					ds.FTSEngines[engName] += info.Size()
				}
				return nil
			})
		}
	}
}

// FormatBytes returns a human-readable byte string (e.g. "1.5 KB", "2.3 MB").
func FormatBytes(b int64) string {
	if b == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	v := float64(b)
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", b)
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}
