package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/api"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
)

// Type aliases - these types are now defined in api/
type (
	warcSummaryStats = api.WARCSummary
	warcSystemStats  = api.WARCSystemStats
	warcAPIRecord    = api.WARCAPIRecord
)

var knownPackFormats = []string{"parquet", "bin", "duckdb", "markdown"}

func normalizeWARCIndexParam(raw string) (string, int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, fmt.Errorf("missing warc index")
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return "", 0, fmt.Errorf("invalid warc index %q", raw)
	}
	return formatWARCIndex(n), n, nil
}

func parseWARCInt(idx string) int {
	n, err := strconv.Atoi(strings.TrimSpace(idx))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func summarizeWARCRecords(recs []metastore.WARCRecord) warcSummaryStats {
	var out warcSummaryStats
	out.Total = len(recs)
	for _, rec := range recs {
		out.WARCBytes += rec.WARCBytes
		out.MarkdownBytes += rec.MarkdownBytes
		packBytes := sumInt64Map(rec.PackBytes)
		ftsBytes := sumInt64Map(rec.FTSBytes)
		out.PackBytes += packBytes
		out.FTSBytes += ftsBytes
		out.TotalBytes += rec.WARCBytes + rec.MarkdownBytes + packBytes + ftsBytes

		if rec.WARCBytes > 0 {
			out.Downloaded++
		}
		if rec.MarkdownBytes > 0 {
			out.MarkdownReady++
		}
		if rec.PackBytes["parquet"] > 0 {
			out.Packed++
		}
		if ftsBytes > 0 {
			out.Indexed++
		}
	}
	return out
}

func toWARCAPIRecord(rec metastore.WARCRecord) warcAPIRecord {
	pack := cloneMap(rec.PackBytes)
	fts := cloneMap(rec.FTSBytes)
	packTotal := sumInt64Map(pack)
	ftsTotal := sumInt64Map(fts)
	total := rec.TotalBytes
	if total <= 0 {
		total = rec.WARCBytes + rec.MarkdownBytes + packTotal + ftsTotal
	}
	out := warcAPIRecord{
		Index:         rec.WARCIndex,
		ManifestIndex: rec.ManifestIndex,
		Filename:      rec.Filename,
		RemotePath:    rec.RemotePath,
		WARCBytes:     rec.WARCBytes,
		MarkdownDocs:  rec.MarkdownDocs,
		MarkdownBytes: rec.MarkdownBytes,
		PackBytes:     pack,
		FTSBytes:      fts,
		TotalBytes:    total,
		HasWARC:       rec.WARCBytes > 0,
		HasPack:       packTotal > 0,
		HasFTS:        ftsTotal > 0,
	}
	if !rec.UpdatedAt.IsZero() {
		out.UpdatedAt = rec.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

// enrichWARCAPIRecord fills WARCMdBytes, WARCMdDocs, and HasMarkdown from live disk
// and DocStore for a single warcAPIRecord. crawlDir is the crawl root directory.
func enrichWARCAPIRecord(ctx context.Context, r *warcAPIRecord, crawlDir string, docs *DocStore) {
	localIdx := r.Index
	if r.Filename != "" {
		if s, ok := warcIndexFromPathStrict(r.Filename); ok {
			localIdx = s
		}
	}

	// Check warc_md/{localIdx}.md.warc.gz
	mdPath := filepath.Join(crawlDir, "warc_md", localIdx+".md.warc.gz")
	if info, err := os.Stat(mdPath); err == nil {
		r.WARCMdBytes = info.Size()
	}
	if docs != nil {
		if meta, ok, _ := docs.GetShardMeta(ctx, "", localIdx); ok {
			r.WARCMdDocs = meta.TotalDocs
		}
	}
	r.HasMarkdown = r.WARCMdBytes > 0 || r.MarkdownBytes > 0
}

func cloneMap(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func relatedWARCJobs(jobs []*pipeline.Job, filesToken, crawlID string) []*pipeline.Job {
	if len(jobs) == 0 {
		return nil
	}
	out := make([]*pipeline.Job, 0, 8)
	for _, job := range jobs {
		if job == nil {
			continue
		}
		if crawlID != "" && job.Config.CrawlID != "" && job.Config.CrawlID != crawlID {
			continue
		}
		if strings.TrimSpace(job.Config.Files) == filesToken {
			out = append(out, job)
		}
		if len(out) >= 20 {
			break
		}
	}
	return out
}

func collectWARCSystemStats(crawlDir string) warcSystemStats {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	stats := warcSystemStats{
		MemAlloc:      int64(ms.Alloc),
		MemHeapSys:    int64(ms.HeapSys),
		MemStackInuse: int64(ms.StackInuse),
		Goroutines:    runtime.NumGoroutine(),
	}
	var fsinfo syscall.Statfs_t
	if err := syscall.Statfs(crawlDir, &fsinfo); err == nil {
		total := int64(fsinfo.Blocks) * int64(fsinfo.Bsize)
		free := int64(fsinfo.Bavail) * int64(fsinfo.Bsize)
		stats.DiskTotal = total
		stats.DiskFree = free
		stats.DiskUsed = total - free
	}
	return stats
}

func deleteWARCArtifacts(crawlDir, warcIndex, target, format, engine string) ([]string, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		target = "all"
	}
	format = strings.ToLower(strings.TrimSpace(format))
	engine = strings.TrimSpace(engine)
	removed := make([]string, 0, 8)
	addRemoved := func(path string) {
		if path != "" {
			removed = append(removed, path)
		}
	}

	deleteDirIfExists := func(path string) error {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		addRemoved(path)
		return nil
	}
	deleteFileIfExists := func(path string) error {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		addRemoved(path)
		return nil
	}

	if target == "warc" || target == "all" {
		warcDir := filepath.Join(crawlDir, "warc")
		if entries, err := os.ReadDir(warcDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				idx, ok := warcIndexFromPathStrict(e.Name())
				if ok && idx == warcIndex {
					if err := deleteFileIfExists(filepath.Join(warcDir, e.Name())); err != nil {
						return nil, fmt.Errorf("delete warc file %s: %w", e.Name(), err)
					}
				}
			}
		}
	}

	if target == "markdown" || target == "all" {
		path := filepath.Join(crawlDir, "warc_md", warcIndex+".md.warc.gz")
		if err := deleteFileIfExists(path); err != nil {
			return nil, fmt.Errorf("delete warc_md %s: %w", warcIndex, err)
		}
		// Also delete per-shard DocStore metadata.
		metaPath := filepath.Join(crawlDir, "warc_md", warcIndex+".meta.duckdb")
		_ = deleteFileIfExists(metaPath)
		_ = deleteFileIfExists(metaPath + ".wal")
	}

	if target == "pack" || target == "all" {
		formats := knownPackFormats
		if format != "" {
			formats = []string{format}
		}
		for _, fmtName := range formats {
			path, err := pipeline.PackPath(filepath.Join(crawlDir, "pack"), fmtName, warcIndex)
			if err != nil {
				if format != "" {
					return nil, err
				}
				continue
			}
			if err := deleteFileIfExists(path); err != nil {
				return nil, fmt.Errorf("delete pack file %s: %w", path, err)
			}
		}
	}

	if target == "index" || target == "all" {
		ftsRoot := filepath.Join(crawlDir, "fts")
		if engine != "" {
			path := filepath.Join(ftsRoot, engine, warcIndex)
			if err := deleteDirIfExists(path); err != nil {
				return nil, fmt.Errorf("delete fts shard %s engine %s: %w", warcIndex, engine, err)
			}
		} else if engines, err := os.ReadDir(ftsRoot); err == nil {
			for _, e := range engines {
				if !e.IsDir() {
					continue
				}
				path := filepath.Join(ftsRoot, e.Name(), warcIndex)
				if err := deleteDirIfExists(path); err != nil {
					return nil, fmt.Errorf("delete fts shard %s engine %s: %w", warcIndex, e.Name(), err)
				}
			}
		}
	}

	sort.Strings(removed)
	return removed, nil
}
