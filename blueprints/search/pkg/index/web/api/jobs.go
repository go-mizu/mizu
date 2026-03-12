package api

import (
	"encoding/json"
	"log"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
)

func listJobs(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		jobs := d.Jobs.List()
		snapshots := make([]pipeline.Job, len(jobs))
		for i, j := range jobs {
			snapshots[i] = *j
		}
		return c.JSON(200, struct {
			Jobs []pipeline.Job `json:"jobs"`
		}{Jobs: snapshots})
	}
}

func getJob(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		id := c.Param("id")
		job := d.Jobs.Get(id)
		if job == nil {
			return c.JSON(404, errResp{"job not found"})
		}
		snapshot := *job
		return c.JSON(200, &snapshot)
	}
}

func createJob(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var cfg pipeline.JobConfig
		if err := json.NewDecoder(c.Request().Body).Decode(&cfg); err != nil {
			return c.JSON(400, errResp{"invalid JSON: " + err.Error()})
		}
		if cfg.Type == "" {
			return c.JSON(400, errResp{"missing type field"})
		}

		job := d.Jobs.Create(cfg)
		log.Printf("[api] INFO  job create id=%s type=%s crawl=%s files=%s engine=%s source=%s format=%s",
			job.ID, cfg.Type, cfg.CrawlID, cfg.Files, cfg.Engine, cfg.Source, cfg.Format)
		snapshot := *job
		d.Jobs.RunJob(job)
		return c.JSON(201, &snapshot)
	}
}

func cancelJob(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		id := c.Param("id")
		if ok := d.Jobs.Cancel(id); !ok {
			return c.JSON(404, errResp{"job not found"})
		}
		log.Printf("[api] INFO  job cancel id=%s", id)
		return c.JSON(200, struct {
			Status string `json:"status"`
		}{"cancelled"})
	}
}

func clearJobs(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		cleared := d.Jobs.Clear()
		return c.JSON(200, struct {
			Cleared int `json:"cleared"`
		}{cleared})
	}
}
