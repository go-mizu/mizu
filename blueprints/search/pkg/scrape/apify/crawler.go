package apify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// Crawler orchestrates full index + detail collection.
type Crawler struct {
	cfg    Config
	client *Client
	db     *DB
	stats  CrawlStats
}

func New(cfg Config) (*Crawler, error) {
	db, err := OpenDB(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	return &Crawler{
		cfg:    cfg,
		client: NewClient(cfg),
		db:     db,
		stats:  CrawlStats{},
	}, nil
}

func (c *Crawler) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *Crawler) DBPath() string    { return c.db.Path() }
func (c *Crawler) Stats() CrawlStats { return c.stats }

func (c *Crawler) Run(ctx context.Context) error {
	c.stats = CrawlStats{StartedAt: time.Now(), CurrentRunUUID: uuid.NewString()}
	if err := c.db.SaveRunStart(c.stats.CurrentRunUUID, c.cfg); err != nil {
		return fmt.Errorf("save run start: %w", err)
	}

	if !c.cfg.DetailOnly {
		if err := c.runIndex(ctx); err != nil {
			_ = c.db.SaveRunFinish(c.stats.CurrentRunUUID, c.stats, "failed", err.Error())
			return err
		}
	}

	if !c.cfg.IndexOnly {
		if err := c.runDetails(ctx); err != nil {
			_ = c.db.SaveRunFinish(c.stats.CurrentRunUUID, c.stats, "failed", err.Error())
			return err
		}
	}

	c.stats.FinishedAt = time.Now()
	if err := c.db.SaveRunFinish(c.stats.CurrentRunUUID, c.stats, "completed", ""); err != nil {
		return fmt.Errorf("save run finish: %w", err)
	}
	return nil
}

func (c *Crawler) runIndex(ctx context.Context) error {
	first, err := c.client.SearchStorePage(ctx, SearchRequest{Page: c.cfg.InitialPage})
	if err != nil {
		return fmt.Errorf("search page %d: %w", c.cfg.InitialPage, err)
	}
	c.stats.ExpectedTotal = first.NbHits
	c.stats.IndexPages = first.NbPages

	if err := c.persistIndexPage(first); err != nil {
		return err
	}

	if first.NbPages <= 1 {
		return nil
	}

	type pageJob struct {
		page     int
		category string
	}
	jobs := make(chan pageJob, first.NbPages)
	var wg sync.WaitGroup
	var idxErr atomic.Value

	workers := c.cfg.Workers
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					return
				}
				resp, err := c.fetchPageWithRetry(ctx, SearchRequest{Page: job.page, Category: job.category})
				if err != nil {
					idxErr.Store(err)
					continue
				}
				if err := c.persistIndexPage(resp); err != nil {
					idxErr.Store(err)
				}
			}
		}()
	}

	for page := 0; page < first.NbPages; page++ {
		if page == c.cfg.InitialPage {
			continue
		}
		jobs <- pageJob{page: page}
	}

	// Algolia's pagination limit caps broad queries; fetch per-category to expand coverage.
	categories, err := c.client.ListCategories(ctx)
	if err == nil {
		for _, cat := range categories {
			resp, catErr := c.client.SearchStorePage(ctx, SearchRequest{Page: 0, Category: cat})
			if catErr != nil {
				idxErr.Store(fmt.Errorf("category %s page 0: %w", cat, catErr))
				continue
			}
			if persistErr := c.persistIndexPage(resp); persistErr != nil {
				idxErr.Store(persistErr)
			}
			for page := 1; page < resp.NbPages; page++ {
				jobs <- pageJob{page: page, category: cat}
			}
		}
	} else {
		idxErr.Store(fmt.Errorf("list categories: %w", err))
	}

	close(jobs)
	wg.Wait()

	if v := idxErr.Load(); v != nil {
		return fmt.Errorf("index phase had errors: %v", v)
	}
	n, err := c.db.IndexCount()
	if err == nil {
		c.stats.IndexedTotal = n
	}
	return nil
}

func (c *Crawler) fetchPageWithRetry(ctx context.Context, req SearchRequest) (*StoreSearchResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			sleepBackoff(attempt)
		}
		resp, err := c.client.SearchStorePage(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	if req.Category != "" {
		return nil, fmt.Errorf("category %s page %d: %w", req.Category, req.Page, lastErr)
	}
	return nil, fmt.Errorf("page %d: %w", req.Page, lastErr)
}

func (c *Crawler) persistIndexPage(resp *StoreSearchResponse) error {
	for _, hit := range resp.Hits {
		raw, _ := json.Marshal(hit)
		if err := c.db.UpsertIndex(hit, string(raw)); err != nil {
			return fmt.Errorf("upsert index %s: %w", hit.ObjectID, err)
		}
		c.stats.IndexedTotal++
	}
	return nil
}

func (c *Crawler) runDetails(ctx context.Context) error {
	ids, err := c.db.PendingDetailIDs(c.cfg.MaxDetails, c.cfg.RefreshDetails)
	if err != nil {
		return fmt.Errorf("load pending details: %w", err)
	}
	c.stats.DetailQueued = int64(len(ids))
	if len(ids) == 0 {
		return nil
	}

	jobs := make(chan string, len(ids))
	var wg sync.WaitGroup
	var firstErr atomic.Value

	workers := c.cfg.Workers
	if workers < 1 {
		workers = 1
	}

	var limiter *rate.Limiter
	if c.cfg.QPS > 0 {
		limiter = rate.NewLimiter(rate.Limit(c.cfg.QPS), max(1, int(c.cfg.QPS)))
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for objectID := range jobs {
				if ctx.Err() != nil {
					return
				}
				if limiter != nil {
					if err := limiter.Wait(ctx); err != nil {
						firstErr.Store(err)
						return
					}
				}
				if err := c.fetchDetailWithRetry(ctx, objectID); err != nil {
					firstErr.CompareAndSwap(nil, err)
				}
			}
		}()
	}

	for _, id := range ids {
		jobs <- id
	}
	close(jobs)
	wg.Wait()

	if v := firstErr.Load(); v != nil {
		return fmt.Errorf("detail phase completed with partial failures: %v", v)
	}
	return nil
}

func (c *Crawler) fetchDetailWithRetry(ctx context.Context, objectID string) error {
	var lastErr error
	var lastStatus int
	var lastBody []byte

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			sleepBackoff(attempt)
		}
		resp, statusCode, body, err := c.client.FetchActorDetail(ctx, objectID)
		if err == nil && resp != nil && resp.Data != nil {
			enrichErrs := make([]string, 0, 2)
			if c.cfg.EnrichVersions {
				versions, e := c.fetchAllVersionsWithRetry(ctx, objectID)
				if e != nil {
					enrichErrs = append(enrichErrs, "versions: "+e.Error())
				} else if len(versions) > 0 {
					resp.Data.VersionsAll = versions
					resp.Data.Versions = versions
				}
			}
			if c.cfg.EnrichLatestBuild {
				buildID := latestBuildIDFromTagged(resp.Data.TaggedBuilds)
				if buildID != "" {
					buildData, e := c.fetchActorBuildWithRetry(ctx, buildID)
					if e != nil {
						enrichErrs = append(enrichErrs, "latest-build: "+e.Error())
					} else if buildData != nil {
						resp.Data.LatestBuild = buildData
					}
				}
			}
			if len(enrichErrs) > 0 {
				resp.Data.EnrichmentError = strings.Join(enrichErrs, "; ")
			}
			if dbErr := c.db.UpsertDetail(resp.Data, statusCode, string(body)); dbErr != nil {
				lastErr = dbErr
				lastStatus = statusCode
				lastBody = body
				continue
			}
			c.stats.DetailDone++
			c.stats.DetailSuccess++
			return nil
		}
		lastErr = err
		lastStatus = statusCode
		lastBody = body
	}

	c.stats.DetailDone++
	c.stats.DetailFailed++
	errText := "unknown error"
	if lastErr != nil {
		errText = lastErr.Error()
	}
	if dbErr := c.db.UpsertDetailFailure(objectID, lastStatus, errText, string(lastBody)); dbErr != nil {
		return fmt.Errorf("detail %s failed (%s), also failed to persist error: %w", objectID, errText, dbErr)
	}
	return fmt.Errorf("detail %s failed: %s", objectID, errText)
}

func (c *Crawler) fetchAllVersionsWithRetry(ctx context.Context, objectID string) ([]map[string]any, error) {
	var (
		offset int
		out    []map[string]any
	)
	for {
		var (
			resp *ActorVersionsResponse
			err  error
		)
		for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
			if attempt > 0 {
				sleepBackoff(attempt)
			}
			resp, _, _, err = c.client.FetchActorVersions(ctx, objectID, 1000, offset)
			if err == nil {
				break
			}
		}
		if err != nil {
			return out, err
		}
		if resp == nil || len(resp.Data.Items) == 0 {
			break
		}
		out = append(out, resp.Data.Items...)
		offset += len(resp.Data.Items)
		if resp.Data.Total > 0 && offset >= resp.Data.Total {
			break
		}
		if len(resp.Data.Items) < 1000 {
			break
		}
	}
	return out, nil
}

func (c *Crawler) fetchActorBuildWithRetry(ctx context.Context, buildID string) (map[string]any, error) {
	var (
		resp *ActorBuildResponse
		err  error
	)
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			sleepBackoff(attempt)
		}
		resp, _, _, err = c.client.FetchActorBuild(ctx, buildID)
		if err == nil && resp != nil {
			return resp.Data, nil
		}
	}
	return nil, err
}
