package amazon

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// AuthorState is the observable state for an AuthorTask.
type AuthorState struct {
	URL    string
	Status string
	Error  string
}

// AuthorMetric is the final result of an AuthorTask.
type AuthorMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// AuthorTask fetches and stores an Amazon author profile page.
type AuthorTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[AuthorState, AuthorMetric] = (*AuthorTask)(nil)

func (t *AuthorTask) Run(ctx context.Context, emit func(*AuthorState)) (AuthorMetric, error) {
	var m AuthorMetric

	emit(&AuthorState{URL: t.URL, Status: "fetching"})

	// 1. IsVisited check
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		emit(&AuthorState{URL: t.URL, Status: "skipped"})
		return m, nil
	}

	// 2. Fetch
	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&AuthorState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&AuthorState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, EntityAuthor, code)
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		msg := fmt.Sprintf("HTTP %d", code)
		emit(&AuthorState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	emit(&AuthorState{URL: t.URL, Status: "parsing"})

	// 3. Parse
	author, err := ParseAuthor(doc, t.URL)
	if err != nil {
		m.Failed++
		emit(&AuthorState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 4. DB upsert
	if err := t.DB.UpsertAuthor(*author); err != nil {
		m.Failed++
		emit(&AuthorState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 5. Mark done and enqueue book ASINs as products
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntityAuthor, code)

		var items []QueueItem
		for _, asin := range author.BookASINs {
			items = append(items, QueueItem{
				URL:        BaseURL + "/dp/" + asin,
				EntityType: EntityProduct,
				Priority:   10,
			})
		}
		t.StateDB.EnqueueBatch(items)
	}

	m.Fetched++
	emit(&AuthorState{URL: t.URL, Status: "done"})
	return m, nil
}
