package amazon

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// SearchState is the observable state for a SearchTask.
type SearchState struct {
	URL          string
	Query        string
	Status       string
	Error        string
	ResultsFound int
}

// SearchMetric is the final result of a SearchTask.
type SearchMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// SearchTask fetches and stores an Amazon search results page.
type SearchTask struct {
	URL      string
	Query    string
	Page     int
	Client   *Client
	DB       *DB
	StateDB  *State
	MaxPages int // 0 = unlimited
}

var _ core.Task[SearchState, SearchMetric] = (*SearchTask)(nil)

func (t *SearchTask) Run(ctx context.Context, emit func(*SearchState)) (SearchMetric, error) {
	var m SearchMetric

	page := t.Page
	if page <= 0 {
		page = 1
	}

	emit(&SearchState{URL: t.URL, Query: t.Query, Status: "fetching"})

	// 1. Fetch
	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&SearchState{URL: t.URL, Query: t.Query, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&SearchState{URL: t.URL, Query: t.Query, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, EntitySearch, code)
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		msg := fmt.Sprintf("HTTP %d", code)
		emit(&SearchState{URL: t.URL, Query: t.Query, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	emit(&SearchState{URL: t.URL, Query: t.Query, Status: "parsing"})

	// 2. Parse
	sr, productURLs, nextPageURL, err := ParseSearch(doc, t.Query, page, t.URL)
	if err != nil {
		m.Failed++
		emit(&SearchState{URL: t.URL, Query: t.Query, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 3. DB upsert
	if err := t.DB.UpsertSearchResult(sr); err != nil {
		m.Failed++
		emit(&SearchState{URL: t.URL, Query: t.Query, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 4. Mark done and enqueue discovered links
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntitySearch, code)

		var items []QueueItem

		// Product URLs
		for _, pURL := range productURLs {
			items = append(items, QueueItem{
				URL:        pURL,
				EntityType: EntityProduct,
				Priority:   10,
			})
		}

		// Next page
		if nextPageURL != "" && (t.MaxPages == 0 || page < t.MaxPages) {
			items = append(items, QueueItem{
				URL:        nextPageURL,
				EntityType: EntitySearch,
				Priority:   5,
			})
		}

		t.StateDB.EnqueueBatch(items)
	}

	m.Fetched++
	emit(&SearchState{URL: t.URL, Query: t.Query, Status: "done", ResultsFound: len(productURLs)})
	return m, nil
}
