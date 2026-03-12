package amazon

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// BrandState is the observable state for a BrandTask.
type BrandState struct {
	URL    string
	Status string
	Error  string
}

// BrandMetric is the final result of a BrandTask.
type BrandMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// BrandTask fetches and stores an Amazon brand/store page.
type BrandTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[BrandState, BrandMetric] = (*BrandTask)(nil)

func (t *BrandTask) Run(ctx context.Context, emit func(*BrandState)) (BrandMetric, error) {
	var m BrandMetric

	emit(&BrandState{URL: t.URL, Status: "fetching"})

	// 1. IsVisited check
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		emit(&BrandState{URL: t.URL, Status: "skipped"})
		return m, nil
	}

	// 2. Fetch
	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&BrandState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&BrandState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, EntityBrand, code)
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		msg := fmt.Sprintf("HTTP %d", code)
		emit(&BrandState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	emit(&BrandState{URL: t.URL, Status: "parsing"})

	// 3. Parse
	brand, err := ParseBrand(doc, t.URL)
	if err != nil {
		m.Failed++
		emit(&BrandState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 4. DB upsert
	if err := t.DB.UpsertBrand(*brand); err != nil {
		m.Failed++
		emit(&BrandState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 5. Mark done and enqueue featured ASINs as products
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntityBrand, code)

		var items []QueueItem
		for _, asin := range brand.FeaturedASINs {
			items = append(items, QueueItem{
				URL:        BaseURL + "/dp/" + asin,
				EntityType: EntityProduct,
				Priority:   10,
			})
		}
		t.StateDB.EnqueueBatch(items)
	}

	m.Fetched++
	emit(&BrandState{URL: t.URL, Status: "done"})
	return m, nil
}
