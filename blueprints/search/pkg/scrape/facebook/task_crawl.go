package facebook

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type CrawlState struct {
	Done     int64
	Pending  int64
	Failed   int64
	InFlight []string
	RPS      float64
}

type CrawlMetric struct {
	Done     int64
	Failed   int64
	Duration time.Duration
}

type CrawlTask struct {
	Config  Config
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[CrawlState, CrawlMetric] = (*CrawlTask)(nil)

func (t *CrawlTask) Run(ctx context.Context, emit func(*CrawlState)) (CrawlMetric, error) {
	start := time.Now()
	var done atomic.Int64
	var mu sync.Mutex
	inFlight := map[int]string{}

	workers := maxOrDefault(t.Config.Workers, DefaultWorkers)
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pending, _, _, failed := t.StateDB.QueueStats()
				mu.Lock()
				urls := make([]string, 0, len(inFlight))
				for _, u := range inFlight {
					urls = append(urls, u)
				}
				mu.Unlock()
				rps := 0.0
				if elapsed := time.Since(start).Seconds(); elapsed > 0 {
					rps = float64(done.Load()) / elapsed
				}
				emit(&CrawlState{Done: done.Load(), Pending: pending, Failed: failed, InFlight: urls, RPS: rps})
			}
		}
	}()

	workerID := 0
	for {
		if ctx.Err() != nil {
			break
		}
		pending, _, _, _ := t.StateDB.QueueStats()
		if pending == 0 {
			wg.Wait()
			pending, _, _, _ = t.StateDB.QueueStats()
			if pending == 0 {
				break
			}
		}

		items, err := t.StateDB.Pop(workers)
		if err != nil || len(items) == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, item := range items {
			item := item
			wid := workerID
			workerID++
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				mu.Lock()
				inFlight[wid] = item.URL
				mu.Unlock()

				runEntityTask(ctx, item, t.Config, t.Client, t.DB, t.StateDB)

				mu.Lock()
				delete(inFlight, wid)
				mu.Unlock()
				done.Add(1)
			}()
		}
	}

	wg.Wait()
	_, _, _, failed := t.StateDB.QueueStats()
	return CrawlMetric{Done: done.Load(), Failed: failed, Duration: time.Since(start)}, nil
}

func runEntityTask(ctx context.Context, item QueueItem, cfg Config, client *Client, db *DB, stateDB *State) {
	switch item.EntityType {
	case EntityPage:
		_, _ = (&PageTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB, MaxPages: cfg.MaxPages, MaxComments: cfg.MaxComments}).Run(ctx, func(*PageState) {})
	case EntityProfile:
		_, _ = (&ProfileTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB, MaxPages: cfg.MaxPages, MaxComments: cfg.MaxComments}).Run(ctx, func(*ProfileState) {})
	case EntityGroup:
		_, _ = (&GroupTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB, MaxPages: cfg.MaxPages, MaxComments: cfg.MaxComments}).Run(ctx, func(*GroupState) {})
	case EntityPost:
		_, _ = (&PostTask{URL: item.URL, Client: client, DB: db, StateDB: stateDB, MaxComments: cfg.MaxComments}).Run(ctx, func(*PostState) {})
	case EntitySearch:
		_, _ = (&SearchTask{Query: item.URL, SearchType: "top", Client: client, DB: db, StateDB: stateDB, MaxPages: cfg.MaxPages}).Run(ctx, func(*SearchState) {})
	default:
		_ = stateDB.Done(item.URL, item.EntityType, 0)
	}
}

func collectFeed(ctx context.Context, client *Client, startURL, ownerID, ownerName, ownerType string, maxPages, maxComments int) ([]Post, []Comment) {
	var posts []Post
	var comments []Comment
	seenPost := map[string]bool{}
	nextURL := startURL
	for page := 0; nextURL != "" && page < maxPages; page++ {
		doc, _, err := client.FetchHTML(ctx, nextURL)
		if err != nil || doc == nil {
			break
		}
		ps := ParsePosts(doc, nextURL, ownerID, ownerName, ownerType)
		for _, p := range ps {
			if p.PostID == "" || seenPost[p.PostID] {
				continue
			}
			seenPost[p.PostID] = true
			posts = append(posts, p)
			cs := ParseComments(doc, p.PostID, nextURL, maxComments)
			comments = append(comments, cs...)
		}
		nextURL = ParseNextPage(doc, nextURL)
	}
	return posts, comments
}

func enqueueDiscoveredLinks(stateDB *State, doc *goquery.Document, rawURL string) {
	for _, item := range DiscoverLinks(doc, rawURL) {
		_ = stateDB.Enqueue(item.URL, item.EntityType, item.Priority)
	}
}

func maxOrDefault(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func now() time.Time {
	return time.Now()
}
