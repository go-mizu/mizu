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
//
// Key design: WARCIndex is the manifest position (formatWARCIndex(i)), ensuring
// all 100K manifest entries are unique even though CC segment filenames reuse
// 00000–00999 across segments. Local disk data is linked via filename lookup.
func buildWARCRecords(crawlID, crawlDir string, manifestPaths []string, updatedAt time.Time) []metastore.WARCRecord {
	records := make(map[string]*metastore.WARCRecord, max(len(manifestPaths), 64))
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

	// filenameToKey maps WARC filename → manifest-position key ("99000" etc.)
	filenameToKey := make(map[string]string, len(manifestPaths))
	for i, p := range manifestPaths {
		idx := formatWARCIndex(i) // manifest position is the unique key
		rec := ensure(idx)
		rec.ManifestIndex = int64(i)
		rec.RemotePath = p
		if rec.Filename == "" {
			rec.Filename = filepath.Base(p)
		}
		filenameToKey[filepath.Base(p)] = idx
	}

	// localSuffixToKey maps the 5-digit filename suffix ("00000") to the
	// manifest-position key for the local WARC file. Built from the warc/ scan.
	localSuffixToKey := make(map[string]string)

	warcDir := filepath.Join(crawlDir, "warc")
	if entries, err := os.ReadDir(warcDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			// Find the manifest key for this filename, or fall back to local suffix.
			key, ok := filenameToKey[e.Name()]
			if !ok {
				key, ok = warcIndexFromPathStrict(e.Name())
				if !ok {
					continue
				}
			}
			rec := ensure(key)
			if rec.Filename == "" {
				rec.Filename = e.Name()
			}
			if info, err := e.Info(); err == nil {
				rec.WARCBytes = info.Size()
			}
			// Record local suffix → manifest key for subdirectory linkage.
			if localSuffix, ok2 := warcIndexFromPathStrict(e.Name()); ok2 {
				localSuffixToKey[localSuffix] = key
			}
		}
	}

	// resolveLocalKey returns the record key for a local-suffix-keyed artifact
	// (markdown, fts, pack directories). Prefers manifest-position key when known.
	resolveLocalKey := func(localSuffix string) string {
		if key, ok := localSuffixToKey[localSuffix]; ok {
			return key
		}
		return localSuffix
	}

	// Scan warc_md/ for .md.warc.gz packed files (new format).
	warcMdRoot := filepath.Join(crawlDir, "warc_md")
	if entries, err := os.ReadDir(warcMdRoot); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
				continue
			}
			shard := strings.TrimSuffix(e.Name(), ".md.warc.gz")
			if !isNumericName(shard) {
				continue
			}
			rec := ensure(resolveLocalKey(normalizeWARCIndex(shard)))
			if info, err := e.Info(); err == nil {
				rec.MarkdownBytes = info.Size()
			}
			// MarkdownDocs left as 0 here; enriched from DocStore in handleWARCList.
		}
	}

	// Scan markdown/ for individual .md files (old format).
	markdownRoot := filepath.Join(crawlDir, "markdown")
	if shards, err := os.ReadDir(markdownRoot); err == nil {
		for _, shard := range shards {
			if !shard.IsDir() || !isNumericName(shard.Name()) {
				continue
			}
			rec := ensure(resolveLocalKey(normalizeWARCIndex(shard.Name())))
			if rec.MarkdownBytes > 0 {
				continue // already detected via warc_md/
			}
			docs, bytes := scanMarkdownShard(filepath.Join(markdownRoot, shard.Name()))
			if docs > 0 {
				rec.MarkdownDocs = docs
				rec.MarkdownBytes = bytes
			}
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
				localSuffix, ok := warcIndexFromPackFile(d.Name())
				if !ok {
					return nil
				}
				rec := ensure(resolveLocalKey(localSuffix))
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
				rec := ensure(resolveLocalKey(normalizeWARCIndex(shard.Name())))
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
