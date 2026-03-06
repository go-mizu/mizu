package web

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
)

var knownPackFormats = []string{"parquet", "bin", "duckdb", "markdown"}

// WARCListResponse is returned by GET /api/warc.
type WARCListResponse struct {
	CrawlID         string           `json:"crawl_id"`
	Offset          int              `json:"offset"`
	Limit           int              `json:"limit"`
	Total           int              `json:"total"`
	Summary         warcSummaryStats `json:"summary"`
	WARCs           []warcAPIRecord  `json:"warcs"`
	System          warcSystemStats  `json:"system"`
	MetaBackend     string           `json:"meta_backend"`
	MetaGeneratedAt string           `json:"meta_generated_at"`
	MetaStale       bool             `json:"meta_stale"`
	MetaRefreshing  bool             `json:"meta_refreshing"`
	MetaLastError   string           `json:"meta_last_error"`
}

// WARCDetailResponse is returned by GET /api/warc/{index}.
type WARCDetailResponse struct {
	CrawlID         string          `json:"crawl_id"`
	WARC            warcAPIRecord   `json:"warc"`
	Jobs            []*Job          `json:"jobs"`
	System          warcSystemStats `json:"system"`
	MetaBackend     string          `json:"meta_backend"`
	MetaGeneratedAt string          `json:"meta_generated_at"`
	MetaStale       bool            `json:"meta_stale"`
	MetaRefreshing  bool            `json:"meta_refreshing"`
	MetaLastError   string          `json:"meta_last_error"`
}

// WARCActionResponse is returned by POST /api/warc/{index}/action.
type WARCActionResponse struct {
	OK              bool     `json:"ok"`
	Action          string   `json:"action"`
	CrawlID         string   `json:"crawl_id"`
	WARCIndex       string   `json:"warc_index"`
	Job             *Job     `json:"job"`
	DeletedPaths    []string `json:"deleted_paths"`
	RefreshAccepted bool     `json:"refresh_accepted"`
}

type warcSummaryStats struct {
	Total         int   `json:"total"`
	Downloaded    int   `json:"downloaded"`
	MarkdownReady int   `json:"markdown_ready"`
	Packed        int   `json:"packed"`
	Indexed       int   `json:"indexed"`
	WARCBytes     int64 `json:"warc_bytes"`
	MarkdownBytes int64 `json:"markdown_bytes"`
	PackBytes     int64 `json:"pack_bytes"`
	FTSBytes      int64 `json:"fts_bytes"`
	TotalBytes    int64 `json:"total_bytes"`
}

type warcSystemStats struct {
	MemAlloc      int64 `json:"mem_alloc"`
	MemHeapSys    int64 `json:"mem_heap_sys"`
	MemStackInuse int64 `json:"mem_stack_inuse"`
	Goroutines    int   `json:"goroutines"`
	DiskTotal     int64 `json:"disk_total"`
	DiskUsed      int64 `json:"disk_used"`
	DiskFree      int64 `json:"disk_free"`
}

type warcAPIRecord struct {
	Index         string           `json:"index"`
	ManifestIndex int64            `json:"manifest_index"`
	Filename      string           `json:"filename"`
	RemotePath    string           `json:"remote_path"`
	WARCBytes     int64            `json:"warc_bytes"`     // warc/*.warc.gz size
	WARCMdBytes   int64            `json:"warc_md_bytes"`  // warc_md/*.md.warc.gz size
	WARCMdDocs    int64            `json:"warc_md_docs"`   // doc count from DocStore or scan
	MarkdownDocs  int64            `json:"markdown_docs"`  // deprecated: old markdown/ dir count
	MarkdownBytes int64            `json:"markdown_bytes"` // deprecated: old markdown/ dir size
	PackBytes     map[string]int64 `json:"pack_bytes"`
	FTSBytes      map[string]int64 `json:"fts_bytes"`
	TotalBytes    int64            `json:"total_bytes"`
	HasWARC       bool             `json:"has_warc"`
	HasMarkdown   bool             `json:"has_markdown"` // true when warc_md_bytes > 0
	HasPack       bool             `json:"has_pack"`
	HasFTS        bool             `json:"has_fts"`
	UpdatedAt     string           `json:"updated_at,omitempty"`
}

func (s *Server) handleWARCList(c *mizu.Ctx) error {
	crawlID := strings.TrimSpace(c.Query("crawl"))
	if crawlID == "" {
		crawlID = s.CrawlID
	}
	crawlDir := s.resolveCrawlDir(crawlID)
	offset := queryIntCtx(c, "offset", 0)
	limit := queryIntCtx(c, "limit", 200)
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))

	var (
		recs        []metastore.WARCRecord
		summaryMeta DataSummaryWithMeta
		err         error
	)
	if s.Meta != nil {
		recs, summaryMeta, err = s.Meta.ListWARCs(c.Context(), crawlID, crawlDir)
		if err != nil {
			logErrorf("warc list meta lookup failed crawl=%s err=%v", crawlID, err)
			return c.JSON(500, errResp{err.Error()})
		}
	} else {
		recs = buildWARCRecords(crawlID, crawlDir, nil, time.Now().UTC())
		summaryMeta = DataSummaryWithMeta{
			MetaBackend:     "scan-fallback",
			MetaGeneratedAt: time.Now().UTC().Format(time.RFC3339),
		}
	}

	filtered := recs
	if q != "" {
		filtered = make([]metastore.WARCRecord, 0, len(recs))
		for _, rec := range recs {
			if strings.Contains(strings.ToLower(rec.WARCIndex), q) ||
				strings.Contains(strings.ToLower(rec.Filename), q) ||
				strings.Contains(strings.ToLower(rec.RemotePath), q) {
				filtered = append(filtered, rec)
			}
		}
	}

	// Phase filter: show only WARCs at a specific pipeline stage.
	// Three user-facing phases: download → markdown → index.
	// Pack is an internal detail exposed only in advanced actions.
	phase := strings.ToLower(strings.TrimSpace(c.Query("phase")))
	if phase != "" {
		phased := make([]metastore.WARCRecord, 0, len(filtered))
		for _, rec := range filtered {
			hasFTS := sumInt64Map(rec.FTSBytes) > 0
			hasMD := rec.MarkdownDocs > 0 || rec.MarkdownBytes > 0
			switch phase {
			case "download":
				if rec.WARCBytes <= 0 {
					phased = append(phased, rec)
				}
			case "markdown":
				if rec.WARCBytes > 0 && !hasMD {
					phased = append(phased, rec)
				}
			case "index":
				if hasMD && !hasFTS {
					phased = append(phased, rec)
				}
			case "complete":
				if rec.WARCBytes > 0 && hasMD && hasFTS {
					phased = append(phased, rec)
				}
			}
		}
		filtered = phased
	}

	stats := summarizeWARCRecords(recs)
	total := len(filtered)
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := filtered[offset:end]

	rows := make([]warcAPIRecord, 0, len(page))
	for _, rec := range page {
		row := toWARCAPIRecord(rec)
		enrichWARCAPIRecord(c.Context(), &row, filepath.Join(crawlDir, "warc_md"), s.Docs)
		rows = append(rows, row)
	}
	sys := collectWARCSystemStats(crawlDir)
	logInfof("warc list crawl=%s total=%d offset=%d limit=%d query=%q", crawlID, total, offset, limit, q)

	return c.JSON(200, WARCListResponse{
		CrawlID:         crawlID,
		Offset:          offset,
		Limit:           limit,
		Total:           total,
		Summary:         stats,
		WARCs:           rows,
		System:          sys,
		MetaBackend:     summaryMeta.MetaBackend,
		MetaGeneratedAt: summaryMeta.MetaGeneratedAt,
		MetaStale:       summaryMeta.MetaStale,
		MetaRefreshing:  summaryMeta.MetaRefreshing,
		MetaLastError:   summaryMeta.MetaLastError,
	})
}

func (s *Server) handleWARCDetail(c *mizu.Ctx) error {
	crawlID := strings.TrimSpace(c.Query("crawl"))
	if crawlID == "" {
		crawlID = s.CrawlID
	}
	warcIndex, _, err := normalizeWARCIndexParam(c.Param("index"))
	if err != nil {
		return c.JSON(400, errResp{err.Error()})
	}
	crawlDir := s.resolveCrawlDir(crawlID)

	var (
		rec         metastore.WARCRecord
		ok          bool
		summaryMeta DataSummaryWithMeta
	)
	if s.Meta != nil {
		rec, ok, summaryMeta, err = s.Meta.GetWARC(c.Context(), crawlID, crawlDir, warcIndex)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
	} else {
		rows := buildWARCRecords(crawlID, crawlDir, nil, time.Now().UTC())
		for _, row := range rows {
			if row.WARCIndex == warcIndex {
				rec = row
				ok = true
				break
			}
		}
		summaryMeta = DataSummaryWithMeta{
			MetaBackend:     "scan-fallback",
			MetaGeneratedAt: time.Now().UTC().Format(time.RFC3339),
		}
	}
	if !ok {
		return c.JSON(404, errResp{"warc not found"})
	}

	filesToken := strconv.Itoa(parseWARCInt(warcIndex))
	related := relatedWARCJobs(s.Jobs.List(), filesToken, crawlID)
	warcRow := toWARCAPIRecord(rec)
	enrichWARCAPIRecord(c.Context(), &warcRow, filepath.Join(crawlDir, "warc_md"), s.Docs)
	return c.JSON(200, WARCDetailResponse{
		CrawlID:         crawlID,
		WARC:            warcRow,
		Jobs:            related,
		System:          collectWARCSystemStats(crawlDir),
		MetaBackend:     summaryMeta.MetaBackend,
		MetaGeneratedAt: summaryMeta.MetaGeneratedAt,
		MetaStale:       summaryMeta.MetaStale,
		MetaRefreshing:  summaryMeta.MetaRefreshing,
		MetaLastError:   summaryMeta.MetaLastError,
	})
}

type warcActionRequest struct {
	Action string `json:"action"`
	Fast   bool   `json:"fast"`
	Format string `json:"format"`
	Engine string `json:"engine"`
	Source string `json:"source"`
	Target string `json:"target"`
	Crawl  string `json:"crawl"`
}

func (s *Server) handleWARCAction(c *mizu.Ctx) error {
	warcIndex, n, err := normalizeWARCIndexParam(c.Param("index"))
	if err != nil {
		return c.JSON(400, errResp{err.Error()})
	}
	var req warcActionRequest
	if c.Request().Body != nil {
		_ = json.NewDecoder(c.Request().Body).Decode(&req)
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		return c.JSON(400, errResp{"missing action"})
	}

	crawlID := strings.TrimSpace(req.Crawl)
	if crawlID == "" {
		crawlID = s.CrawlID
	}
	crawlDir := s.resolveCrawlDir(crawlID)
	fileToken := strconv.Itoa(n)

	var (
		job          *Job
		deletedPaths []string
	)
	switch action {
	case "download":
		job = s.createAndRunJob(JobConfig{Type: "download", CrawlID: crawlID, Files: fileToken})
	case "markdown":
		job = s.createAndRunJob(JobConfig{Type: "markdown", CrawlID: crawlID, Files: fileToken, Fast: req.Fast})
	case "pack":
		format := strings.TrimSpace(req.Format)
		if format == "" {
			format = "parquet"
		}
		job = s.createAndRunJob(JobConfig{Type: "pack", CrawlID: crawlID, Files: fileToken, Format: format})
	case "index":
		engine := strings.TrimSpace(req.Engine)
		if engine == "" {
			engine = s.EngineName
		}
		source := strings.TrimSpace(req.Source)
		if source == "" {
			source = "files"
		}
		job = s.createAndRunJob(JobConfig{Type: "index", CrawlID: crawlID, Files: fileToken, Engine: engine, Source: source})
	case "reindex":
		engine := strings.TrimSpace(req.Engine)
		if engine == "" {
			engine = s.EngineName
		}
		if deletedPaths, err = deleteWARCArtifacts(crawlDir, warcIndex, "index", "", engine); err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		source := strings.TrimSpace(req.Source)
		if source == "" {
			source = "files"
		}
		job = s.createAndRunJob(JobConfig{Type: "index", CrawlID: crawlID, Files: fileToken, Engine: engine, Source: source})
	case "delete":
		target := strings.TrimSpace(req.Target)
		if target == "" {
			target = "all"
		}
		if deletedPaths, err = deleteWARCArtifacts(crawlDir, warcIndex, target, req.Format, req.Engine); err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
	default:
		return c.JSON(400, errResp{fmt.Sprintf("unknown action %q", action)})
	}

	refreshAccepted := false
	if s.Meta != nil {
		refreshAccepted = s.Meta.TriggerRefresh(crawlID, crawlDir, true)
	}
	logInfof("warc action crawl=%s warc=%s action=%s deleted=%d job=%s", crawlID, warcIndex, action, len(deletedPaths), jobID(job))
	return c.JSON(200, WARCActionResponse{
		OK:              true,
		Action:          action,
		CrawlID:         crawlID,
		WARCIndex:       warcIndex,
		Job:             job,
		DeletedPaths:    deletedPaths,
		RefreshAccepted: refreshAccepted,
	})
}

func (s *Server) createAndRunJob(cfg JobConfig) *Job {
	job := s.Jobs.Create(cfg)
	logInfof("warc action created job id=%s type=%s crawl=%s files=%s engine=%s source=%s format=%s fast=%t",
		job.ID, cfg.Type, cfg.CrawlID, cfg.Files, cfg.Engine, cfg.Source, cfg.Format, cfg.Fast)
	s.Jobs.RunJob(job)
	return job
}

func jobID(j *Job) string {
	if j == nil {
		return ""
	}
	return j.ID
}

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
		if rec.MarkdownDocs > 0 || rec.MarkdownBytes > 0 {
			out.MarkdownReady++
		}
		if packBytes > 0 {
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
// and DocStore for a single warcAPIRecord. warcMdBase is the warc_md/ directory.
func enrichWARCAPIRecord(ctx context.Context, r *warcAPIRecord, warcMdBase string, docs *DocStore) {
	mdPath := filepath.Join(warcMdBase, r.Index+".md.warc.gz")
	if info, err := os.Stat(mdPath); err == nil {
		r.WARCMdBytes = info.Size()
	}
	if docs != nil {
		if meta, ok, _ := docs.GetShardMeta(ctx, "", r.Index); ok {
			r.WARCMdDocs = meta.TotalDocs
		}
	}
	r.HasMarkdown = r.WARCMdBytes > 0
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

func relatedWARCJobs(jobs []*Job, filesToken, crawlID string) []*Job {
	if len(jobs) == 0 {
		return nil
	}
	out := make([]*Job, 0, 8)
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
		path := filepath.Join(crawlDir, "markdown", warcIndex)
		if err := deleteDirIfExists(path); err != nil {
			return nil, fmt.Errorf("delete markdown shard %s: %w", warcIndex, err)
		}
	}

	if target == "pack" || target == "all" {
		formats := knownPackFormats
		if format != "" {
			formats = []string{format}
		}
		for _, fmtName := range formats {
			path, err := packFilePath(filepath.Join(crawlDir, "pack"), fmtName, warcIndex)
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
