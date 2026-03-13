package amazon

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Crawler struct {
	cfg    Config
	client *Client
	db     *DB
}

func New(cfg Config) (*Crawler, error) {
	db, err := OpenDB(cfg.DBPath())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return &Crawler{cfg: cfg, client: NewClient(cfg), db: db}, nil
}

func (c *Crawler) Close() error { return c.db.Close() }

func (c *Crawler) DiscoverPages(ctx context.Context, query string) ([]string, error) {
	start := 1
	if c.cfg.Resume {
		last, err := c.db.LastPageForQuery(query)
		if err == nil && last > 0 {
			start = last + 1
		}
	}
	pages := make([]string, 0, c.cfg.MaxPages)
	for page := start; page <= c.cfg.MaxPages; page++ {
		select {
		case <-ctx.Done():
			return pages, ctx.Err()
		default:
		}
		pages = append(pages, c.client.SearchURL(query, page))
	}
	return pages, nil
}

func (c *Crawler) Crawl(ctx context.Context, query string) (CrawlStats, error) {
	defer c.Close()
	pages, err := c.DiscoverPages(ctx, query)
	if err != nil {
		return CrawlStats{}, err
	}

	type result struct {
		products []Product
		hasNext  bool
		page     int
		err      error
	}

	jobs := make(chan int)
	results := make(chan result)
	workers := c.cfg.Workers
	if workers < 1 {
		workers = 1
	}

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range jobs {
				body, err := c.client.FetchSearchPage(ctx, query, page)
				if err != nil {
					results <- result{page: page, err: err}
					continue
				}
				if err := ValidateSearchHTML(body); err != nil {
					results <- result{page: page, err: err}
					continue
				}

				products, hasNext, err := ParseSearchResults(
					fmt.Sprintf("https://%s", c.cfg.Market),
					query,
					page,
					body,
				)
				if err != nil {
					results <- result{page: page, err: err}
					continue
				}
				now := time.Now()
				for i := range products {
					products[i].ScrapedAt = now
				}
				results <- result{page: page, products: products, hasNext: hasNext}
				sleepRate(c.cfg.RateLimit)
			}
		}()
	}

	go func() {
		for idx := range len(pages) {
			jobs <- idx + 1
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	all := make([]Product, 0, 128)
	continueTo := c.cfg.MaxPages
	for r := range results {
		if r.err != nil {
			return CrawlStats{}, fmt.Errorf("page %d: %w", r.page, r.err)
		}
		if len(r.products) > 0 {
			all = append(all, r.products...)
		}
		if !r.hasNext && r.page < continueTo {
			continueTo = r.page
		}
	}

	filtered := all[:0]
	for _, p := range all {
		if p.ResultPage <= continueTo {
			filtered = append(filtered, p)
		}
	}

	if err := c.db.InsertProducts(filtered); err != nil {
		return CrawlStats{}, fmt.Errorf("save products: %w", err)
	}

	uniq := map[string]struct{}{}
	for _, p := range filtered {
		uniq[p.ASIN] = struct{}{}
	}
	return CrawlStats{Query: query, Pages: continueTo, Products: len(filtered), UniqueASIN: len(uniq)}, nil
}
