package web

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
)

// buildWARCRecords scans local crawl artifacts and merges manifest identity into
// per-WARC records for dashboard list/detail pages.
func buildWARCRecords(crawlID, crawlDir string, manifestPaths []string, updatedAt time.Time) []metastore.WARCRecord {
	records := make(map[string]*metastore.WARCRecord, len(manifestPaths))
	ensure := func(idx string) *metastore.WARCRecord {
		if rec, ok := records[idx]; ok {
			return rec
		}
		rec := &metastore.WARCRecord{
			CrawlID:       crawlID,
			WARCIndex:     idx,
			PackBytes:     make(map[string]int64),
			FTSBytes:      make(map[string]int64),
			UpdatedAt:     updatedAt,
			ManifestIndex: -1,
		}
		records[idx] = rec
		return rec
	}

	for i, p := range manifestPaths {
		idx, ok := warcIndexFromPathStrict(p)
		if !ok {
			idx = formatWARCIndex(i)
		}
		rec := ensure(idx)
		rec.ManifestIndex = int64(i)
		rec.RemotePath = p
		if rec.Filename == "" {
			rec.Filename = filepath.Base(p)
		}
	}

	warcDir := filepath.Join(crawlDir, "warc")
	if entries, err := os.ReadDir(warcDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			idx, ok := warcIndexFromPathStrict(e.Name())
			if !ok {
				continue
			}
			rec := ensure(idx)
			if rec.Filename == "" {
				rec.Filename = e.Name()
			}
			if info, err := e.Info(); err == nil {
				rec.WARCBytes = info.Size()
			}
		}
	}

	mdRoot := filepath.Join(crawlDir, "markdown")
	if entries, err := os.ReadDir(mdRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() || !isNumericName(e.Name()) {
				continue
			}
			idx := normalizeWARCIndex(e.Name())
			rec := ensure(idx)
			docs, bytes := scanMarkdownShard(filepath.Join(mdRoot, e.Name()))
			rec.MarkdownDocs = docs
			rec.MarkdownBytes = bytes
		}
	}

	packRoot := filepath.Join(crawlDir, "pack")
	if formats, err := os.ReadDir(packRoot); err == nil {
		for _, formatEntry := range formats {
			if !formatEntry.IsDir() {
				continue
			}
			format := formatEntry.Name()
			formatDir := filepath.Join(packRoot, format)
			filepath.WalkDir(formatDir, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil || d.IsDir() {
					return nil
				}
				idx, ok := warcIndexFromPackFile(d.Name())
				if !ok {
					return nil
				}
				rec := ensure(idx)
				if info, err := d.Info(); err == nil {
					rec.PackBytes[format] += info.Size()
				}
				return nil
			})
		}
	}

	ftsRoot := filepath.Join(crawlDir, "fts")
	if engines, err := os.ReadDir(ftsRoot); err == nil {
		for _, engineEntry := range engines {
			if !engineEntry.IsDir() {
				continue
			}
			engine := engineEntry.Name()
			engineDir := filepath.Join(ftsRoot, engine)
			shards, err := os.ReadDir(engineDir)
			if err != nil {
				continue
			}
			for _, shard := range shards {
				if !shard.IsDir() || !isNumericName(shard.Name()) {
					continue
				}
				idx := normalizeWARCIndex(shard.Name())
				rec := ensure(idx)
				rec.FTSBytes[engine] += dirSize(filepath.Join(engineDir, shard.Name()))
			}
		}
	}

	out := make([]metastore.WARCRecord, 0, len(records))
	for _, rec := range records {
		rec.TotalBytes = rec.WARCBytes + rec.MarkdownBytes + sumInt64Map(rec.PackBytes) + sumInt64Map(rec.FTSBytes)
		out = append(out, *rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WARCIndex < out[j].WARCIndex })
	return out
}

func scanMarkdownShard(dir string) (docs int64, bytes int64) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		docs++
		if info, err := d.Info(); err == nil {
			bytes += info.Size()
		}
		return nil
	})
	return docs, bytes
}

func dirSize(dir string) int64 {
	var total int64
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func sumInt64Map(m map[string]int64) int64 {
	var total int64
	for _, v := range m {
		total += v
	}
	return total
}

func formatWARCIndex(i int) string {
	return fmt.Sprintf("%05d", i)
}

func normalizeWARCIndex(s string) string {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return s
	}
	return formatWARCIndex(n)
}

func warcIndexFromPathStrict(path string) (string, bool) {
	base := filepath.Base(path)
	trimmed := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".warc")
	parts := strings.Split(trimmed, "-")
	if len(parts) == 0 {
		return "", false
	}
	last := parts[len(parts)-1]
	if len(last) != 5 || !isNumericName(last) {
		return "", false
	}
	return last, true
}

func warcIndexFromPackFile(name string) (string, bool) {
	base := filepath.Base(name)
	if len(base) < 5 {
		return "", false
	}
	candidate := base[:5]
	if !isNumericName(candidate) {
		return "", false
	}
	return candidate, true
}
