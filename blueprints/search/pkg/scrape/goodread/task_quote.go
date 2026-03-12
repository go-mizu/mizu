package goodread

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// QuoteState is the observable state for a QuoteTask.
type QuoteState struct {
	URL         string
	Status      string
	Error       string
	QuotesFound int
}

// QuoteMetric is the final result of a QuoteTask.
type QuoteMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// QuoteTask fetches and stores Goodreads quotes from a page.
type QuoteTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[QuoteState, QuoteMetric] = (*QuoteTask)(nil)

func (t *QuoteTask) Run(ctx context.Context, emit func(*QuoteState)) (QuoteMetric, error) {
	var m QuoteMetric

	emit(&QuoteState{URL: t.URL, Status: "fetching"})

	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&QuoteState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		emit(&QuoteState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, code, "quote")
		}
		return m, nil
	}

	emit(&QuoteState{URL: t.URL, Status: "parsing"})

	quotes, err := ParseQuotes(doc, t.URL)
	if err != nil {
		m.Failed++
		emit(&QuoteState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	for _, q := range quotes {
		t.DB.UpsertQuote(q)
	}

	if t.StateDB != nil {
		t.StateDB.Done(t.URL, code, "quote")
	}

	m.Fetched++
	emit(&QuoteState{URL: t.URL, Status: "done", QuotesFound: len(quotes)})
	return m, nil
}
