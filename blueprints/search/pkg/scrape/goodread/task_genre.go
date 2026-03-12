package goodread

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// GenreState is the observable state for a GenreTask.
type GenreState struct {
	URL        string
	Status     string
	Error      string
	BooksFound int
}

// GenreMetric is the final result of a GenreTask.
type GenreMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// GenreTask fetches and stores a Goodreads genre page.
type GenreTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[GenreState, GenreMetric] = (*GenreTask)(nil)

func (t *GenreTask) Run(ctx context.Context, emit func(*GenreState)) (GenreMetric, error) {
	var m GenreMetric

	emit(&GenreState{URL: t.URL, Status: "fetching"})

	slug := extractIDFromPath(t.URL, "/genres/")
	if slug == "" {
		m.Failed++
		emit(&GenreState{URL: t.URL, Status: "failed", Error: "cannot extract genre slug"})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, "cannot extract genre slug")
		}
		return m, nil
	}

	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&GenreState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, code, "genre")
		}
		return m, nil
	}

	genre, bookIDs, err := ParseGenre(doc, slug, t.URL)
	if err != nil {
		m.Failed++
		emit(&GenreState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	t.DB.UpsertGenre(*genre)

	// Enqueue top books from genre page
	if t.StateDB != nil {
		for _, id := range bookIDs {
			t.StateDB.Enqueue(BaseURL+"/book/show/"+id, "book", 2)
		}
		t.StateDB.Done(t.URL, code, "genre")
	}

	m.Fetched++
	emit(&GenreState{URL: t.URL, Status: "done", BooksFound: len(bookIDs)})
	return m, nil
}
