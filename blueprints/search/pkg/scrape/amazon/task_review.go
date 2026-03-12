package amazon

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// ReviewState is the observable state for a ReviewTask.
type ReviewState struct {
	URL          string
	ASIN         string
	Status       string
	Error        string
	Pages        int
	ReviewsFound int
}

// ReviewMetric is the final result of a ReviewTask.
type ReviewMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Pages   int
}

// ReviewTask fetches and stores all review pages for an Amazon product.
type ReviewTask struct {
	URL      string
	ASIN     string
	Client   *Client
	DB       *DB
	StateDB  *State
	MaxPages int // 0 = unlimited
}

var _ core.Task[ReviewState, ReviewMetric] = (*ReviewTask)(nil)

func (t *ReviewTask) Run(ctx context.Context, emit func(*ReviewState)) (ReviewMetric, error) {
	var m ReviewMetric

	state := &ReviewState{URL: t.URL, ASIN: t.ASIN, Status: "fetching"}
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

		reviews, nextURL, err := ParseReviews(doc, t.ASIN, currentURL)
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

		for _, r := range reviews {
			t.DB.UpsertReview(r)
		}

		state.Pages = page
		state.ReviewsFound += len(reviews)
		state.Status = "fetching"
		emit(state)

		m.Pages++
		page++

		if nextURL == "" {
			break
		}
		currentURL = nextURL
	}

	// Mark first URL done
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntityReview, 200)
	}

	m.Fetched++
	state.Status = "done"
	emit(state)
	return m, nil
}
