package goodread

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// SeriesState is the observable state for a SeriesTask.
type SeriesState struct {
	URL    string
	Status string
	Error  string
}

// SeriesMetric is the final result of a SeriesTask.
type SeriesMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// SeriesTask fetches and stores a single Goodreads series page.
type SeriesTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[SeriesState, SeriesMetric] = (*SeriesTask)(nil)

func (t *SeriesTask) Run(ctx context.Context, emit func(*SeriesState)) (SeriesMetric, error) {
	var m SeriesMetric

	emit(&SeriesState{URL: t.URL, Status: "fetching"})

	seriesID := extractIDFromPath(t.URL, "/series/")
	if seriesID == "" {
		m.Failed++
		errMsg := "cannot extract series ID"
		emit(&SeriesState{URL: t.URL, Status: "failed", Error: errMsg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, errMsg)
		}
		return m, nil
	}

	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&SeriesState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		emit(&SeriesState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, code, "series")
		}
		return m, nil
	}

	emit(&SeriesState{URL: t.URL, Status: "parsing"})

	series, books, err := ParseSeries(doc, seriesID, t.URL)
	if err != nil {
		m.Failed++
		emit(&SeriesState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	if err := t.DB.UpsertSeries(*series); err != nil {
		m.Failed++
		emit(&SeriesState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	t.DB.InsertSeriesBooks(books)

	// Enqueue series books
	if t.StateDB != nil {
		for _, sb := range books {
			url := BaseURL + "/book/show/" + sb.BookID
			t.StateDB.Enqueue(url, "book", 3)
		}
		t.StateDB.Done(t.URL, code, "series")
	}

	m.Fetched++
	emit(&SeriesState{URL: t.URL, Status: "done"})
	return m, nil
}
