package amazon

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// BestsellerState is the observable state for a BestsellerTask.
type BestsellerState struct {
	URL          string
	Status       string
	Error        string
	EntriesFound int
}

// BestsellerMetric is the final result of a BestsellerTask.
type BestsellerMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// BestsellerTask fetches and stores an Amazon bestseller list page.
type BestsellerTask struct {
	URL      string
	ListType string
	Category string
	NodeID   string
	Client   *Client
	DB       *DB
	StateDB  *State
}

var _ core.Task[BestsellerState, BestsellerMetric] = (*BestsellerTask)(nil)

func (t *BestsellerTask) Run(ctx context.Context, emit func(*BestsellerState)) (BestsellerMetric, error) {
	var m BestsellerMetric

	emit(&BestsellerState{URL: t.URL, Status: "fetching"})

	// 1. IsVisited check
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		emit(&BestsellerState{URL: t.URL, Status: "skipped"})
		return m, nil
	}

	// 2. Fetch
	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&BestsellerState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&BestsellerState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, EntityBestseller, code)
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		msg := fmt.Sprintf("HTTP %d", code)
		emit(&BestsellerState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	emit(&BestsellerState{URL: t.URL, Status: "parsing"})

	// 3. Parse
	list, entries, err := ParseBestseller(doc, t.ListType, t.Category, t.NodeID, t.URL)
	if err != nil {
		m.Failed++
		emit(&BestsellerState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 4. DB upsert
	if err := t.DB.UpsertBestsellerList(*list); err != nil {
		m.Failed++
		emit(&BestsellerState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if err := t.DB.InsertBestsellerEntries(entries); err != nil {
		m.Failed++
		emit(&BestsellerState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 5. Mark done and enqueue entry ASINs as products
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntityBestseller, code)

		var items []QueueItem
		for _, e := range entries {
			if e.ASIN != "" {
				items = append(items, QueueItem{
					URL:        BaseURL + "/dp/" + e.ASIN,
					EntityType: EntityProduct,
					Priority:   10,
				})
			}
		}
		t.StateDB.EnqueueBatch(items)
	}

	m.Fetched++
	emit(&BestsellerState{URL: t.URL, Status: "done", EntriesFound: len(entries)})
	return m, nil
}
