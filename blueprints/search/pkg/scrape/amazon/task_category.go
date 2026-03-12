package amazon

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// CategoryState is the observable state for a CategoryTask.
type CategoryState struct {
	URL           string
	Status        string
	Error         string
	ProductsFound int
}

// CategoryMetric is the final result of a CategoryTask.
type CategoryMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// CategoryTask fetches and stores an Amazon category/browse-node page.
type CategoryTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[CategoryState, CategoryMetric] = (*CategoryTask)(nil)

func (t *CategoryTask) Run(ctx context.Context, emit func(*CategoryState)) (CategoryMetric, error) {
	var m CategoryMetric

	emit(&CategoryState{URL: t.URL, Status: "fetching"})

	// 1. IsVisited check
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		emit(&CategoryState{URL: t.URL, Status: "skipped"})
		return m, nil
	}

	// 2. Fetch
	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&CategoryState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&CategoryState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, EntityCategory, code)
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		msg := fmt.Sprintf("HTTP %d", code)
		emit(&CategoryState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	emit(&CategoryState{URL: t.URL, Status: "parsing"})

	// 3. Parse
	cat, err := ParseCategory(doc, t.URL)
	if err != nil {
		m.Failed++
		emit(&CategoryState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 4. DB upsert
	if err := t.DB.UpsertCategory(*cat); err != nil {
		m.Failed++
		emit(&CategoryState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 5. Mark done and enqueue discovered links
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntityCategory, code)

		var items []QueueItem

		// Child nodes as categories
		for _, nodeID := range cat.ChildNodeIDs {
			items = append(items, QueueItem{
				URL:        BaseURL + "/s?rh=n%3A" + nodeID,
				EntityType: EntityCategory,
				Priority:   5,
			})
		}

		// Top ASINs as products
		for _, asin := range cat.TopASINs {
			items = append(items, QueueItem{
				URL:        BaseURL + "/dp/" + asin,
				EntityType: EntityProduct,
				Priority:   10,
			})
		}

		t.StateDB.EnqueueBatch(items)
	}

	m.Fetched++
	emit(&CategoryState{URL: t.URL, Status: "done", ProductsFound: len(cat.TopASINs)})
	return m, nil
}
