package api

import (
	"encoding/json"
	"fmt"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/scrape"
)

type startRequest struct {
	Domain           string `json:"domain"`
	Mode             string `json:"mode"`
	MaxPages         int    `json:"max_pages"`
	MaxDepth         int    `json:"max_depth"`
	Workers          int    `json:"workers"`
	TimeoutS         int    `json:"timeout_s"`
	StoreBody        bool   `json:"store_body"`
	Resume           bool   `json:"resume"`
	NoRobots         bool   `json:"no_robots"`
	NoSitemap        bool   `json:"no_sitemap"`
	IncludeSubdomain bool   `json:"include_subdomain"`
	ScrollCount      int    `json:"scroll_count"`
	Continuous       bool   `json:"continuous"`
	StaleHours       int    `json:"stale_hours"`
	SeedURL          string `json:"seed_url"`
	WorkerToken      string `json:"worker_token"`
	WorkerURL        string `json:"worker_url"`
	WorkerBrowser    bool   `json:"worker_browser"`
}

func startScrape(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req startRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(400, errResp{"invalid JSON: " + err.Error()})
		}
		if req.Domain == "" {
			return c.JSON(400, errResp{"domain is required"})
		}
		domain := dcrawler.NormalizeDomain(req.Domain)
		if domain == "" {
			return c.JSON(400, errResp{"invalid domain"})
		}
		if job := findActiveScrapeJob(d, domain); job != nil {
			return c.JSON(409, errResp{fmt.Sprintf("scrape already running: job %s", job.ID)})
		}

		sourceBytes, _ := json.Marshal(scrape.StartParams{
			Mode:             req.Mode,
			MaxPages:         req.MaxPages,
			MaxDepth:         req.MaxDepth,
			Workers:          req.Workers,
			TimeoutS:         req.TimeoutS,
			StoreBody:        req.StoreBody,
			Resume:           req.Resume,
			NoRobots:         req.NoRobots,
			NoSitemap:        req.NoSitemap,
			IncludeSubdomain: req.IncludeSubdomain,
			ScrollCount:      req.ScrollCount,
			Continuous:       req.Continuous,
			StaleHours:       req.StaleHours,
			SeedURL:          req.SeedURL,
			WorkerToken:      req.WorkerToken,
			WorkerURL:        req.WorkerURL,
			WorkerBrowser:    req.WorkerBrowser,
		})
		cfg := pipeline.JobConfig{
			Type:   "scrape",
			Domain: domain,
			Source: string(sourceBytes),
		}
		job := d.Jobs.Create(cfg)
		snap := *job
		d.Jobs.RunJob(job)
		return c.JSON(201, struct {
			JobID  string `json:"job_id"`
			Domain string `json:"domain"`
		}{snap.ID, domain})
	}
}

func resumeScrape(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		domain := dcrawler.NormalizeDomain(c.Param("domain"))
		if domain == "" {
			return c.JSON(400, errResp{"invalid domain"})
		}
		if job := findActiveScrapeJob(d, domain); job != nil {
			return c.JSON(409, errResp{fmt.Sprintf("scrape already running: job %s", job.ID)})
		}

		cfg := pipeline.JobConfig{
			Type:   "scrape",
			Domain: domain,
			Source: `{"resume":true}`,
		}
		job := d.Jobs.Create(cfg)
		snap := *job
		d.Jobs.RunJob(job)
		return c.JSON(201, struct {
			JobID  string `json:"job_id"`
			Domain string `json:"domain"`
		}{snap.ID, domain})
	}
}

func stopScrape(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		domain := dcrawler.NormalizeDomain(c.Param("domain"))
		job := findActiveScrapeJob(d, domain)
		if job == nil {
			return c.JSON(404, errResp{"no active scrape for domain"})
		}
		d.Jobs.Cancel(job.ID)
		return c.JSON(200, struct {
			Status string `json:"status"`
			JobID  string `json:"job_id"`
		}{"cancelled", job.ID})
	}
}

func listScrape(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Scrape == nil {
			return c.JSON(503, errResp{"scrape store not available"})
		}
		resp, err := d.Scrape.ListDomains()
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		return c.JSON(200, resp)
	}
}

func scrapeStatus(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Scrape == nil {
			return c.JSON(503, errResp{"scrape store not available"})
		}
		domain := dcrawler.NormalizeDomain(c.Param("domain"))
		stats, _ := d.Scrape.GetDomainStats(domain)

		var activeJob *scrape.JobInfo
		if job := findActiveScrapeJob(d, domain); job != nil {
			activeJob = &scrape.JobInfo{
				ID:       job.ID,
				Status:   job.Status,
				Progress: job.Progress,
				Message:  job.Message,
				Rate:     job.Rate,
			}
		}

		// Also check recently completed scrape jobs for stats.
		if activeJob == nil {
			for _, job := range d.Jobs.List() {
				if job.Config.Domain == domain && job.Config.Type == "scrape" &&
					(job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled") {
					activeJob = &scrape.JobInfo{
						ID:       job.ID,
						Status:   job.Status,
						Progress: job.Progress,
						Message:  job.Message,
					}
					break
				}
			}
		}

		return c.JSON(200, scrape.DomainStatus{
			Domain:    domain,
			Stats:     stats,
			ActiveJob: activeJob,
			HasData:   stats != nil || activeJob != nil,
		})
	}
}

func scrapePages(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Scrape == nil {
			return c.JSON(503, errResp{"scrape store not available"})
		}
		domain := dcrawler.NormalizeDomain(c.Param("domain"))
		page := queryInt(c, "page", 1)
		pageSize := queryInt(c, "page_size", 50)
		if pageSize > 200 {
			pageSize = 200
		}
		q := c.Query("q")
		sort := c.Query("sort")
		statusFilter := c.Query("status")

		resp, err := d.Scrape.GetPages(domain, page, pageSize, q, sort, statusFilter)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		return c.JSON(200, resp)
	}
}

func scrapePipeline(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		domain := dcrawler.NormalizeDomain(c.Param("domain"))
		if domain == "" {
			return c.JSON(400, errResp{"invalid domain"})
		}
		cfg := pipeline.JobConfig{
			Type:   "scrape_markdown",
			Domain: domain,
		}
		job := d.Jobs.Create(cfg)
		snap := *job
		d.Jobs.RunJob(job)
		return c.JSON(201, struct {
			JobID string `json:"job_id"`
			Type  string `json:"type"`
		}{snap.ID, "scrape_markdown"})
	}
}

func scrapeIndex(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		domain := dcrawler.NormalizeDomain(c.Param("domain"))
		if domain == "" {
			return c.JSON(400, errResp{"invalid domain"})
		}
		cfg := pipeline.JobConfig{
			Type:   "scrape_index",
			Domain: domain,
			Engine: "dahlia",
		}
		job := d.Jobs.Create(cfg)
		snap := *job
		d.Jobs.RunJob(job)
		return c.JSON(201, struct {
			JobID string `json:"job_id"`
			Type  string `json:"type"`
		}{snap.ID, "scrape_index"})
	}
}

// findActiveScrapeJob returns the running/queued scrape job for a domain, or nil.
func findActiveScrapeJob(d *Deps, domain string) *pipeline.Job {
	for _, job := range d.Jobs.List() {
		if job.Config.Domain == domain && job.Config.Type == "scrape" &&
			(job.Status == "running" || job.Status == "queued") {
			return job
		}
	}
	return nil
}

// queryInt reads a query parameter as int with a default fallback.
func queryInt(c *mizu.Ctx, name string, def int) int {
	v := c.Query(name)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}
