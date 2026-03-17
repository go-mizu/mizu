package amazon

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// QAState is the observable state for a QATask.
type QAState struct {
	URL      string
	ASIN     string
	Status   string
	Error    string
	Pages    int
	QAsFound int
}

// QAMetric is the final result of a QATask.
type QAMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Pages   int
}

// QATask fetches and stores all Q&A pages for an Amazon product.
type QATask struct {
	URL      string
	ASIN     string
	Client   *Client
	DB       *DB
	StateDB  *State
	MaxPages int // 0 = unlimited
}

var _ core.Task[QAState, QAMetric] = (*QATask)(nil)

func (t *QATask) Run(ctx context.Context, emit func(*QAState)) (QAMetric, error) {
	var m QAMetric

	state := &QAState{URL: t.URL, ASIN: t.ASIN, Status: "fetching"}
	emit(state)

	page := 1
	currentURL := t.URL

	for {
		if ctx.Err() != nil {
			break
		}
		if t.MaxPages > 0 && page > t.MaxPages {
			break
		}

		doc, code, err := t.Client.FetchHTML(ctx, currentURL)
		if err != nil {
			m.Failed++
			state.Status = "failed"
			state.Error = err.Error()
			emit(state)
			if t.StateDB != nil {
				t.StateDB.Fail(t.URL, err.Error())
			}
			return m, nil
		}
		if code == 404 {
			m.Skipped++
			state.Status = "not_found"
			emit(state)
			if t.StateDB != nil {
				t.StateDB.Done(t.URL, EntityQA, code)
			}
			break
		}
		if doc == nil {
			m.Failed++
			msg := fmt.Sprintf("HTTP %d", code)
			state.Status = "failed"
			state.Error = msg
			emit(state)
			if t.StateDB != nil {
				t.StateDB.Fail(t.URL, msg)
			}
			return m, nil
		}

		qas, nextURL, err := ParseQA(doc, t.ASIN, currentURL)
		if err != nil {
			m.Failed++
			state.Status = "failed"
			state.Error = err.Error()
			emit(state)
			if t.StateDB != nil {
				t.StateDB.Fail(t.URL, err.Error())
			}
			return m, nil
		}

		for _, q := range qas {
			t.DB.UpsertQA(q)
		}

		state.Pages = page
		state.QAsFound += len(qas)
		state.Status = "fetching"
		emit(state)

		m.Pages++
		page++

		if nextURL == "" {
			break
		}
		currentURL = nextURL
	}

	if m.Skipped > 0 && m.Pages == 0 {
		return m, nil
	}

	// Mark first URL done
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntityQA, 200)
	}

	m.Fetched++
	state.Status = "done"
	emit(state)
	return m, nil
}
