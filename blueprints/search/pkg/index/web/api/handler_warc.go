package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
	webstore "github.com/go-mizu/mizu/blueprints/search/pkg/index/web/store"
)

func handleWARCList(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		crawlID := strings.TrimSpace(c.Query("crawl"))
		if crawlID == "" {
			crawlID = d.CrawlID
		}
		crawlDir := d.CrawlDir
		if d.ResolveCrawlDir != nil {
			crawlDir = d.ResolveCrawlDir(crawlID)
		}
		offset := queryIntAPI(c, "offset", 0)
		limit := queryIntAPI(c, "limit", 200)
		if limit <= 0 {
			limit = 200
		}
		if limit > 1000 {
			limit = 1000
		}
		q := strings.ToLower(strings.TrimSpace(c.Query("q")))

		var recs []webstore.WARCRecord
		var summaryMeta DataSummaryWithMeta

		if d.Meta != nil {
			var err error
			recs, summaryMeta, err = d.Meta.ListWARCs(c.Context(), crawlID, crawlDir)
			if err != nil {
				return c.JSON(500, errResp{err.Error()})
			}
		} else if d.ListWARCsFallback != nil {
			recs, summaryMeta = d.ListWARCsFallback(c.Context(), crawlID, crawlDir)
		} else {
			summaryMeta = DataSummaryWithMeta{
				MetaBackend:     "scan-fallback",
				MetaGeneratedAt: time.Now().UTC().Format(time.RFC3339),
			}
		}

		// Text filter.
		filtered := recs
		if q != "" {
			filtered = make([]webstore.WARCRecord, 0, len(recs))
			for _, rec := range recs {
				if strings.Contains(strings.ToLower(rec.WARCIndex), q) ||
					strings.Contains(strings.ToLower(rec.Filename), q) ||
					strings.Contains(strings.ToLower(rec.RemotePath), q) {
					filtered = append(filtered, rec)
				}
			}
		}

		// Phase filter.
		phase := strings.ToLower(strings.TrimSpace(c.Query("phase")))
		if phase != "" {
			sumMap := d.SumInt64Map
			if sumMap == nil {
				sumMap = defaultSumInt64Map
			}
			phased := make([]webstore.WARCRecord, 0, len(filtered))
			for _, rec := range filtered {
				hasFTS := sumMap(rec.FTSBytes) > 0
				hasMD := rec.MarkdownBytes > 0
				hasParquet := rec.PackBytes["parquet"] > 0
				switch phase {
				case "downloaded":
					if rec.WARCBytes > 0 {
						phased = append(phased, rec)
					}
				case "markdown":
					if hasMD {
						phased = append(phased, rec)
					}
				case "indexed":
					if hasFTS {
						phased = append(phased, rec)
					}
				case "exported":
					if hasParquet {
						phased = append(phased, rec)
					}
				}
			}
			filtered = phased
		}

		var summary WARCSummary
		if d.SummarizeWARCs != nil {
			summary = d.SummarizeWARCs(recs)
		}

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

		rows := make([]WARCAPIRecord, 0, len(page))
		for _, rec := range page {
			var row WARCAPIRecord
			if d.BuildWARCRow != nil {
				row = d.BuildWARCRow(c.Context(), rec, crawlDir)
			} else {
				row = warcRecordToAPIRecord(rec)
			}
			rows = append(rows, row)
		}

		var sys WARCSystemStats
		if d.CollectSystemStats != nil {
			sys = d.CollectSystemStats(crawlDir)
		}

		return c.JSON(200, WARCListResponse{
			CrawlID:         crawlID,
			Offset:          offset,
			Limit:           limit,
			Total:           total,
			Summary:         summary,
			WARCs:           rows,
			System:          sys,
			MetaBackend:     summaryMeta.MetaBackend,
			MetaGeneratedAt: summaryMeta.MetaGeneratedAt,
			MetaStale:       summaryMeta.MetaStale,
			MetaRefreshing:  summaryMeta.MetaRefreshing,
			MetaLastError:   summaryMeta.MetaLastError,
		})
	}
}

func handleWARCDetail(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		crawlID := strings.TrimSpace(c.Query("crawl"))
		if crawlID == "" {
			crawlID = d.CrawlID
		}
		warcIndex, _, err := normalizeWARCIndex(d, c.Param("index"))
		if err != nil {
			return c.JSON(400, errResp{err.Error()})
		}
		crawlDir := d.CrawlDir
		if d.ResolveCrawlDir != nil {
			crawlDir = d.ResolveCrawlDir(crawlID)
		}

		var rec webstore.WARCRecord
		var ok bool
		var summaryMeta DataSummaryWithMeta

		if d.Meta != nil {
			rec, ok, summaryMeta, err = d.Meta.GetWARC(c.Context(), crawlID, crawlDir, warcIndex)
			if err != nil {
				return c.JSON(500, errResp{err.Error()})
			}
		} else if d.ListWARCsFallback != nil {
			var rows []webstore.WARCRecord
			rows, summaryMeta = d.ListWARCsFallback(c.Context(), crawlID, crawlDir)
			for _, r := range rows {
				if r.WARCIndex == warcIndex {
					rec = r
					ok = true
					break
				}
			}
		} else {
			summaryMeta = DataSummaryWithMeta{
				MetaBackend:     "scan-fallback",
				MetaGeneratedAt: time.Now().UTC().Format(time.RFC3339),
			}
		}
		if !ok {
			return c.JSON(404, errResp{"warc not found"})
		}

		filesToken := strconv.Itoa(parseWARCInt(warcIndex))
		var allJobs []*pipeline.Job
		if d.Jobs != nil {
			allJobs = d.Jobs.List()
		}
		related := relatedWARCJobs(allJobs, filesToken, crawlID)

		var warcRow WARCAPIRecord
		if d.BuildWARCRow != nil {
			warcRow = d.BuildWARCRow(c.Context(), rec, crawlDir)
		} else {
			warcRow = warcRecordToAPIRecord(rec)
		}

		var sys WARCSystemStats
		if d.CollectSystemStats != nil {
			sys = d.CollectSystemStats(crawlDir)
		}

		return c.JSON(200, WARCDetailResponse{
			CrawlID:         crawlID,
			WARC:            warcRow,
			Jobs:            related,
			System:          sys,
			MetaBackend:     summaryMeta.MetaBackend,
			MetaGeneratedAt: summaryMeta.MetaGeneratedAt,
			MetaStale:       summaryMeta.MetaStale,
			MetaRefreshing:  summaryMeta.MetaRefreshing,
			MetaLastError:   summaryMeta.MetaLastError,
		})
	}
}

type warcActionRequest struct {
	Action string `json:"action"`
	Format string `json:"format"`
	Engine string `json:"engine"`
	Source string `json:"source"`
	Target string `json:"target"`
	Crawl  string `json:"crawl"`
}

func handleWARCAction(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		warcIndex, n, err := normalizeWARCIndex(d, c.Param("index"))
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
			crawlID = d.CrawlID
		}
		crawlDir := d.CrawlDir
		if d.ResolveCrawlDir != nil {
			crawlDir = d.ResolveCrawlDir(crawlID)
		}
		fileToken := strconv.Itoa(n)

		// Resolve local index from WARC record filename.
		localIdx := warcIndex
		if d.Meta != nil && d.WARCIndexFromPath != nil {
			if rec, ok, _, _ := d.Meta.GetWARC(c.Context(), crawlID, crawlDir, warcIndex); ok && rec.Filename != "" {
				if s, ok2 := d.WARCIndexFromPath(rec.Filename); ok2 {
					localIdx = s
				}
			}
		}

		var job *pipeline.Job
		var deletedPaths []string

		switch action {
		case "download":
			job = warcCreateAndRunJob(d, pipeline.JobConfig{Type: "download", CrawlID: crawlID, Files: fileToken})
		case "markdown":
			job = warcCreateAndRunJob(d, pipeline.JobConfig{Type: "markdown", CrawlID: crawlID, Files: fileToken})
		case "pack":
			format := strings.TrimSpace(req.Format)
			if format == "" {
				format = "parquet"
			}
			job = warcCreateAndRunJob(d, pipeline.JobConfig{Type: "pack", CrawlID: crawlID, Files: fileToken, Format: format})
		case "index":
			eng := strings.TrimSpace(req.Engine)
			if eng == "" {
				eng = d.EngineName
			}
			src := strings.TrimSpace(req.Source)
			if src == "" {
				src = "files"
			}
			job = warcCreateAndRunJob(d, pipeline.JobConfig{Type: "index", CrawlID: crawlID, Files: fileToken, Engine: eng, Source: src})
		case "reindex":
			eng := strings.TrimSpace(req.Engine)
			if eng == "" {
				eng = d.EngineName
			}
			if d.DeleteWARCArtifacts != nil {
				if deletedPaths, err = d.DeleteWARCArtifacts(crawlDir, localIdx, "index", "", eng); err != nil {
					return c.JSON(500, errResp{err.Error()})
				}
			}
			src := strings.TrimSpace(req.Source)
			if src == "" {
				src = "files"
			}
			job = warcCreateAndRunJob(d, pipeline.JobConfig{Type: "index", CrawlID: crawlID, Files: fileToken, Engine: eng, Source: src})
		case "delete":
			target := strings.TrimSpace(req.Target)
			if target == "" {
				target = "all"
			}
			if d.DeleteWARCArtifacts != nil {
				if deletedPaths, err = d.DeleteWARCArtifacts(crawlDir, localIdx, target, req.Format, req.Engine); err != nil {
					return c.JSON(500, errResp{err.Error()})
				}
			}
		default:
			return c.JSON(400, errResp{fmt.Sprintf("unknown action %q", action)})
		}

		refreshAccepted := false
		if d.Meta != nil {
			refreshAccepted = d.Meta.TriggerRefresh(crawlID, crawlDir, true)
		}

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
}

// warcCreateAndRunJob creates a job and starts it, returning a snapshot.
func warcCreateAndRunJob(d *Deps, cfg pipeline.JobConfig) *pipeline.Job {
	if d.Jobs == nil {
		return nil
	}
	job := d.Jobs.Create(cfg)
	snap := *job
	d.Jobs.RunJob(job)
	return &snap
}

// relatedWARCJobs returns jobs whose Files token matches and crawlID is compatible.
func relatedWARCJobs(allJobs []*pipeline.Job, filesToken, crawlID string) []*pipeline.Job {
	var out []*pipeline.Job
	for _, j := range allJobs {
		if j.Config.CrawlID != "" && j.Config.CrawlID != crawlID {
			continue
		}
		for _, tok := range strings.Split(j.Config.Files, ",") {
			if strings.TrimSpace(tok) == filesToken {
				out = append(out, j)
				break
			}
		}
	}
	return out
}

// normalizeWARCIndex parses and zero-pads a WARC index URL param.
func normalizeWARCIndex(d *Deps, raw string) (string, int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, fmt.Errorf("missing warc index")
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return "", 0, fmt.Errorf("invalid warc index %q", raw)
	}
	if d != nil && d.FormatWARCIndex != nil {
		return d.FormatWARCIndex(n), n, nil
	}
	return fmt.Sprintf("%05d", n), n, nil
}

// parseWARCInt parses a 5-digit WARC index string to int.
func parseWARCInt(idx string) int {
	n, err := strconv.Atoi(strings.TrimSpace(idx))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// warcRecordToAPIRecord converts a webstore.WARCRecord to a WARCAPIRecord
// without enrichment (used as fallback when BuildWARCRow is nil).
func warcRecordToAPIRecord(rec webstore.WARCRecord) WARCAPIRecord {
	sumMap := defaultSumInt64Map
	return WARCAPIRecord{
		Index:         rec.WARCIndex,
		ManifestIndex: rec.ManifestIndex,
		Filename:      rec.Filename,
		RemotePath:    rec.RemotePath,
		WARCBytes:     rec.WARCBytes,
		MarkdownDocs:  rec.MarkdownDocs,
		MarkdownBytes: rec.MarkdownBytes,
		PackBytes:     rec.PackBytes,
		FTSBytes:      rec.FTSBytes,
		TotalBytes:    rec.WARCBytes + rec.MarkdownBytes + sumMap(rec.PackBytes) + sumMap(rec.FTSBytes),
		HasWARC:       rec.WARCBytes > 0,
		HasMarkdown:   rec.MarkdownBytes > 0,
		HasPack:       sumMap(rec.PackBytes) > 0,
		HasFTS:        sumMap(rec.FTSBytes) > 0,
	}
}
