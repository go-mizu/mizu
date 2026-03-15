package ebay

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// ItemState is the observable state for an ItemTask.
type ItemState struct {
	URL    string
	Status string
	Error  string
}

// ItemMetric is the final result of an ItemTask.
type ItemMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// ItemTask fetches and stores a single eBay item page.
type ItemTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[ItemState, ItemMetric] = (*ItemTask)(nil)

func (t *ItemTask) Run(ctx context.Context, emit func(*ItemState)) (ItemMetric, error) {
	var m ItemMetric

	t.URL = NormalizeItemURL(t.URL)
	emit(&ItemState{URL: t.URL, Status: "fetching"})

	if ExtractItemID(t.URL) == "" {
		m.Failed++
		msg := "cannot extract item ID"
		emit(&ItemState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		emit(&ItemState{URL: t.URL, Status: "skipped"})
		return m, nil
	}

	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&ItemState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&ItemState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, EntityItem, code)
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		msg := fmt.Sprintf("HTTP %d", code)
		emit(&ItemState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	emit(&ItemState{URL: t.URL, Status: "parsing"})

	item, related, err := ParseItem(doc, t.URL)
	if err != nil {
		m.Failed++
		emit(&ItemState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	if err := t.DB.UpsertItem(*item); err != nil {
		m.Failed++
		emit(&ItemState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	if t.StateDB != nil {
		_ = t.StateDB.Done(t.URL, EntityItem, code)
		if len(related) > 0 {
			items := make([]QueueItem, 0, len(related))
			for _, rawURL := range related {
				items = append(items, QueueItem{
					URL:        rawURL,
					EntityType: EntityItem,
					Priority:   2,
				})
			}
			_ = t.StateDB.EnqueueBatch(items)
		}
	}

	m.Fetched++
	emit(&ItemState{URL: t.URL, Status: "done"})
	return m, nil
}
