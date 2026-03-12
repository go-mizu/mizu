package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func searchEngineFromCtx(c *mizu.Ctx, d *Deps) string {
	if engine := strings.TrimSpace(c.Query("engine")); engine != "" {
		return engine
	}
	return d.EngineName
}

func handleSearch(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		q := c.Query("q")
		if q == "" {
			return c.JSON(400, errResp{"missing q parameter"})
		}
		engineName := searchEngineFromCtx(c, d)
		limit := queryIntAPI(c, "limit", 20)
		offset := queryIntAPI(c, "offset", 0)
		if limit > 100 {
			limit = 100
		}

		shardDirs, err := discoverShards(d.ResolveFTSBase(engineName), d.IsNumericName)
		if err != nil {
			return c.JSON(500, errResp{"no FTS index: " + err.Error()})
		}

		type shardResult struct {
			shard string
			hits  []index.Hit
			total int
			err   error
		}

		t0 := time.Now()
		results := make([]shardResult, len(shardDirs))
		var wg sync.WaitGroup
		for i, sd := range shardDirs {
			wg.Add(1)
			go func(idx int, dir, shardName string) {
				defer wg.Done()
				eng, err := index.NewEngine(engineName)
				if err != nil {
					results[idx].err = err
					return
				}
				if d.Addr != "" {
					if setter, ok := eng.(index.AddrSetter); ok {
						setter.SetAddr(d.Addr)
					}
				}
				if err := eng.Open(c.Context(), dir); err != nil {
					results[idx].err = err
					return
				}
				defer eng.Close()

				res, err := eng.Search(c.Context(), index.Query{Text: q, Limit: limit + offset})
				if err != nil {
					results[idx].err = err
					return
				}
				results[idx] = shardResult{shard: shardName, hits: res.Hits, total: res.Total}
			}(i, sd.path, sd.name)
		}
		wg.Wait()

		var allHits []SearchHit
		totalCount := 0
		for _, sr := range results {
			if sr.err != nil {
				continue
			}
			totalCount += sr.total
			for _, h := range sr.hits {
				allHits = append(allHits, SearchHit{
					DocID:   h.DocID,
					Shard:   sr.shard,
					Score:   h.Score,
					Snippet: h.Snippet,
				})
			}
		}
		sort.Slice(allHits, func(i, j int) bool { return allHits[i].Score > allHits[j].Score })
		if offset < len(allHits) {
			allHits = allHits[offset:]
		} else {
			allHits = nil
		}
		if len(allHits) > limit {
			allHits = allHits[:limit]
		}

		// Enrich hits with DocStore metadata.
		if d.Docs != nil && len(allHits) > 0 {
			var mu sync.Mutex
			var wg2 sync.WaitGroup
			for i := range allHits {
				wg2.Add(1)
				go func(idx int) {
					defer wg2.Done()
					rec, ok, _ := d.Docs.GetDoc(c.Context(), d.CrawlID, allHits[idx].Shard, allHits[idx].DocID)
					if !ok {
						return
					}
					mu.Lock()
					allHits[idx].URL = rec.URL
					allHits[idx].Host = rec.Host
					allHits[idx].Title = rec.Title
					if !rec.CrawlDate.IsZero() {
						t := rec.CrawlDate
						allHits[idx].CrawlDate = &t
					}
					allHits[idx].SizeBytes = rec.SizeBytes
					allHits[idx].WordCount = rec.WordCount
					mu.Unlock()
				}(i)
			}
			wg2.Wait()
		}

		elapsed := time.Since(t0).Milliseconds()
		return c.JSON(200, SearchResponse{
			Hits:      allHits,
			Total:     totalCount,
			ElapsedMs: elapsed,
			Query:     q,
			Engine:    engineName,
			Shards:    len(shardDirs),
		})
	}
}

func handleStats(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		engineName := searchEngineFromCtx(c, d)
		shardDirs, err := discoverShards(d.ResolveFTSBase(engineName), d.IsNumericName)
		if err != nil {
			return c.JSON(500, errResp{"no FTS index: " + err.Error()})
		}

		type shardStats struct {
			docs int64
			disk int64
		}
		results := make([]shardStats, len(shardDirs))
		var wg sync.WaitGroup
		for i, sd := range shardDirs {
			wg.Add(1)
			go func(idx int, dir string) {
				defer wg.Done()
				eng, err := index.NewEngine(engineName)
				if err != nil {
					return
				}
				if d.Addr != "" {
					if setter, ok := eng.(index.AddrSetter); ok {
						setter.SetAddr(d.Addr)
					}
				}
				if err := eng.Open(c.Context(), dir); err != nil {
					return
				}
				st, err := eng.Stats(c.Context())
				eng.Close()
				if err != nil {
					return
				}
				results[idx] = shardStats{docs: st.DocCount, disk: st.DiskBytes}
			}(i, sd.path)
		}
		wg.Wait()

		var totalDocs int64
		var totalDisk int64
		for _, r := range results {
			totalDocs += r.docs
			totalDisk += r.disk
		}

		fb := d.FormatBytes
		if fb == nil {
			fb = defaultFormatBytes
		}

		return c.JSON(200, StatsResponse{
			Engine:    engineName,
			Crawl:     d.CrawlID,
			Shards:    len(shardDirs),
			TotalDocs: totalDocs,
			TotalDisk: fb(totalDisk),
		})
	}
}

func handleDoc(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		shard := c.Param("shard")
		docid := c.Param("docid")
		if shard == "" || docid == "" {
			return c.JSON(400, errResp{"missing shard or docid"})
		}

		var raw []byte
		var meta DocRecord

		if d.Docs != nil {
			if rec, ok, _ := d.Docs.GetDoc(c.Context(), d.CrawlID, shard, docid); ok {
				meta = rec
			}
		}

		warcMdPath := filepath.Join(d.WARCMdBase, shard+".md.warc.gz")

		if meta.GzipOffset > 0 && meta.GzipSize > 0 && d.ReadDocByOffset != nil {
			body, err := d.ReadDocByOffset(warcMdPath, meta.GzipOffset, meta.GzipSize)
			if err == nil {
				raw = body
			}
		}

		if raw == nil && d.ReadDocFromWARCMd != nil {
			body, found, err := d.ReadDocFromWARCMd(warcMdPath, docid)
			if err == nil && found {
				raw = body
			}
		}

		if raw == nil {
			return c.JSON(404, errResp{"document not found"})
		}

		if meta.Title == "" && len(raw) > 0 && d.ExtractDocTitle != nil {
			head := raw
			if len(head) > 2048 {
				head = head[:2048]
			}
			meta.Title = d.ExtractDocTitle(head, meta.URL)
		}

		html := ""
		if d.RenderMarkdown != nil {
			rendered, err := d.RenderMarkdown(raw)
			if err != nil {
				return c.JSON(500, errResp{"markdown render failed"})
			}
			html = rendered
		}

		wordCount := len(strings.Fields(string(raw)))

		crawlDateStr := ""
		if !meta.CrawlDate.IsZero() {
			crawlDateStr = meta.CrawlDate.UTC().Format(time.RFC3339)
		}

		return c.JSON(200, DocResponse{
			DocID:        docid,
			Shard:        shard,
			URL:          meta.URL,
			Title:        meta.Title,
			CrawlDate:    crawlDateStr,
			SizeBytes:    int64(len(raw)),
			WordCount:    wordCount,
			WARCRecordID: meta.WARCRecordID,
			RefersTo:     meta.RefersTo,
			Markdown:     string(raw),
			HTML:         html,
		})
	}
}

func handleBrowse(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		shard := c.Query("shard")
		if shard == "" {
			return handleBrowseShards(d, c)
		}
		return handleBrowseDocs(d, c, shard)
	}
}

func handleBrowseShards(d *Deps, c *mizu.Ctx) error {
	crawlID := d.CrawlID
	crawlDir := d.CrawlDir

	var recs interface{ /* metastore.WARCRecord list */ } = nil
	_ = recs

	// Get shard metas from DocStore.
	var metas []DocShardMeta
	if d.Docs != nil {
		metas, _ = d.Docs.ListShardMetas(c.Context(), crawlID)
	}
	metaByName := make(map[string]DocShardMeta, len(metas))
	for _, m := range metas {
		metaByName[m.Shard] = m
	}

	// We need the WARC records to build shard list; use Meta if available.
	sumMap := d.SumInt64Map
	if sumMap == nil {
		sumMap = defaultSumInt64Map
	}
	warcIndexFrom := d.WARCIndexFromPath
	isNumeric := d.IsNumericName
	if isNumeric == nil {
		isNumeric = defaultIsNumericName
	}

	var entries []ShardEntry

	if d.Meta != nil {
		metaRecs, _, err := d.Meta.ListWARCs(c.Context(), crawlID, crawlDir)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		for _, rec := range metaRecs {
			if rec.WARCBytes <= 0 {
				continue
			}
			localIdx := rec.WARCIndex
			if rec.Filename != "" && warcIndexFrom != nil {
				if s, ok := warcIndexFrom(rec.Filename); ok {
					localIdx = s
				}
			}

			hasMarkdown := rec.MarkdownBytes > 0
			if !hasMarkdown {
				warcMdPath := filepath.Join(crawlDir, "warc_md", localIdx+".md.warc.gz")
				if _, err := os.Stat(warcMdPath); err == nil {
					hasMarkdown = true
				}
			}

			hasFTS := sumMap(rec.FTSBytes) > 0
			e := ShardEntry{
				Name:    localIdx,
				HasPack: hasMarkdown || hasFTS,
				HasFTS:  hasFTS,
			}

			if m, ok := metaByName[localIdx]; ok {
				e.HasScan = true
				e.FileCount = int(m.TotalDocs)
				e.TotalSize = m.TotalSizeBytes
				if !m.LastDocDate.IsZero() {
					e.LastDocDate = m.LastDocDate.UTC().Format(time.RFC3339)
				}
				e.MetaStale = time.Since(m.LastScannedAt) > time.Hour
				if !m.LastScannedAt.IsZero() {
					e.LastScannedAt = m.LastScannedAt.UTC().Format(time.RFC3339)
				}
				if e.HasPack && e.MetaStale && d.Docs != nil {
					e.Scanning = d.Docs.IsScanning(crawlID, localIdx)
				}
			} else if e.HasPack && d.Docs != nil {
				e.Scanning = d.Docs.IsScanning(crawlID, localIdx)
			}
			entries = append(entries, e)
		}
	}

	return c.JSON(200, BrowseShardsResponse{
		Shards:     entries,
		HasDocMeta: d.Docs != nil,
	})
}

func handleBrowseDocs(d *Deps, c *mizu.Ctx, shard string) error {
	page := queryIntAPI(c, "page", 1)
	pageSize := queryIntAPI(c, "page_size", 100)
	if pageSize > 500 {
		pageSize = 500
	}
	q := c.Query("q")
	sortBy := c.Query("sort")

	if d.Docs == nil {
		return c.JSON(503, errResp{"doc store not available"})
	}

	meta, hasMeta, _ := d.Docs.GetShardMeta(c.Context(), d.CrawlID, shard)
	scanning := d.Docs.IsScanning(d.CrawlID, shard)

	if !hasMeta {
		warcMdPath := filepath.Join(d.WARCMdBase, shard+".md.warc.gz")
		if _, err := os.Stat(warcMdPath); err != nil {
			return c.JSON(404, errResp{"shard not packed yet"})
		}
		if !scanning {
			scanning = true
			go func() {
				if _, err := d.Docs.ScanShard(context.Background(), d.CrawlID, shard, warcMdPath); err != nil {
					// log but continue
					_ = err
				}
				if d.Hub != nil {
					d.Hub.BroadcastAll(map[string]string{"type": "shard_scan", "shard": shard})
				}
			}()
		}
		return c.JSON(200, BrowseDocsResponse{
			Shard:      shard,
			Docs:       []DocJSON{},
			Total:      0,
			Page:       1,
			Scanning:   scanning,
			NotScanned: !scanning,
		})
	}

	docs, total, err := d.Docs.ListDocs(c.Context(), d.CrawlID, shard, page, pageSize, q, sortBy)
	if err != nil {
		return c.JSON(500, errResp{err.Error()})
	}

	metaStale := time.Since(meta.LastScannedAt) > time.Hour
	if metaStale && !scanning {
		go func() {
			warcMdPath := filepath.Join(d.WARCMdBase, shard+".md.warc.gz")
			if _, err := os.Stat(warcMdPath); err != nil {
				return
			}
			if _, err := d.Docs.ScanShard(context.Background(), d.CrawlID, shard, warcMdPath); err != nil {
				_ = err
			}
			if d.Hub != nil {
				d.Hub.BroadcastAll(map[string]string{"type": "shard_scan", "shard": shard})
			}
		}()
	}

	docsOut := make([]DocJSON, len(docs))
	for i, doc := range docs {
		docsOut[i] = DocJSON{
			DocID:     doc.DocID,
			Shard:     doc.Shard,
			URL:       doc.URL,
			Host:      doc.Host,
			Title:     doc.Title,
			SizeBytes: doc.SizeBytes,
			WordCount: doc.WordCount,
		}
		if !doc.CrawlDate.IsZero() {
			docsOut[i].CrawlDate = doc.CrawlDate.UTC().Format(time.RFC3339)
		}
	}

	lastScannedAt := ""
	if !meta.LastScannedAt.IsZero() {
		lastScannedAt = meta.LastScannedAt.UTC().Format(time.RFC3339)
	}
	return c.JSON(200, BrowseDocsResponse{
		Shard:         shard,
		Docs:          docsOut,
		Total:         int(total),
		Page:          page,
		PageSize:      pageSize,
		MetaStale:     metaStale,
		Scanning:      scanning,
		LastScannedAt: lastScannedAt,
	})
}

// ── Utility helpers ───────────────────────────────────────────────────────────

type shardDir struct {
	name string
	path string
}

func discoverShards(ftsBase string, isNumeric func(string) bool) ([]shardDir, error) {
	if isNumeric == nil {
		isNumeric = defaultIsNumericName
	}
	entries, err := os.ReadDir(ftsBase)
	if err != nil {
		return nil, err
	}
	var dirs []shardDir
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !isNumeric(e.Name()) {
			continue
		}
		dirs = append(dirs, shardDir{name: e.Name(), path: filepath.Join(ftsBase, e.Name())})
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no shard directories in %s", ftsBase)
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	return dirs, nil
}

func defaultIsNumericName(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func defaultSumInt64Map(m map[string]int64) int64 {
	var total int64
	for _, v := range m {
		total += v
	}
	return total
}

func defaultFormatBytes(b int64) string {
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

func queryIntAPI(c *mizu.Ctx, key string, def int) int {
	s := c.Query(key)
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
}
