package goodread

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// ListState is the observable state for a ListTask.
type ListState struct {
	URL        string
	Status     string
	Error      string
	BooksFound int
}

// ListMetric is the final result of a ListTask.
type ListMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// ListTask fetches and stores a single Goodreads list page.
type ListTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[ListState, ListMetric] = (*ListTask)(nil)

func (t *ListTask) Run(ctx context.Context, emit func(*ListState)) (ListMetric, error) {
	var m ListMetric

	emit(&ListState{URL: t.URL, Status: "fetching"})

	listID := extractIDFromPath(t.URL, "/list/show/")
	if listID == "" {
		m.Failed++
		emit(&ListState{URL: t.URL, Status: "failed", Error: "cannot extract list ID"})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, "cannot extract list ID")
		}
		return m, nil
	}

	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&ListState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		emit(&ListState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, code, "list")
		}
		return m, nil
	}

	emit(&ListState{URL: t.URL, Status: "parsing"})

	list, books, err := ParseList(doc, listID, t.URL)
	if err != nil {
		m.Failed++
		emit(&ListState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	t.DB.UpsertList(*list)
	t.DB.InsertListBooks(books)

	emit(&ListState{URL: t.URL, Status: "storing", BooksFound: len(books)})

	// Enqueue books
	if t.StateDB != nil {
		for _, lb := range books {
			t.StateDB.Enqueue(BaseURL+"/book/show/"+lb.BookID, "book", 3)
		}
		t.StateDB.Done(t.URL, code, "list")
	}

	m.Fetched++
	emit(&ListState{URL: t.URL, Status: "done", BooksFound: len(books)})
	return m, nil
}
