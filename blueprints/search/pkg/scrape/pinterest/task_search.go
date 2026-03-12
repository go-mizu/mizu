package pinterest

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// SearchState is the observable state for a SearchTask.
type SearchState struct {
	Query     string
	Status    string
	Error     string
	Page      int
	PinsFound int
}

// SearchMetric is the final result of a SearchTask.
type SearchMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Pages   int
}

// SearchTask fetches pins from a Pinterest search query and stores them in the DB.
type SearchTask struct {
	Query   string
	MaxPins int    // 0 = use DefaultMaxPins
	Client  *Client
	DB      *DB
	StateDB *State // optional; marks search URL as visited
	Config  Config
}

var _ core.Task[SearchState, SearchMetric] = (*SearchTask)(nil)

func (t *SearchTask) Run(ctx context.Context, emit func(*SearchState)) (SearchMetric, error) {
	var m SearchMetric

	maxPins := t.MaxPins
	if maxPins <= 0 {
		maxPins = t.Config.MaxPins
		if maxPins <= 0 {
			maxPins = DefaultMaxPins
		}
	}

	searchURL := fmt.Sprintf("%s/search/pins/?q=%s", BaseURL, t.Query)

	emit(&SearchState{Query: t.Query, Status: "searching"})

	pins, err := t.Client.SearchPins(ctx, t.Query, maxPins)
	if err != nil {
		m.Failed++
		emit(&SearchState{Query: t.Query, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(searchURL, err.Error())
		}
		return m, nil
	}

	emit(&SearchState{Query: t.Query, Status: "storing", PinsFound: len(pins)})

	for i, pin := range pins {
		if ctx.Err() != nil {
			break
		}
		if err := t.DB.UpsertPin(pin); err != nil {
			m.Failed++
			continue
		}
		m.Fetched++

		if (i+1)%50 == 0 {
			emit(&SearchState{
				Query:     t.Query,
				Status:    "storing",
				PinsFound: len(pins),
				Page:      (i + 1) / 25,
			})
		}
	}

	if t.StateDB != nil {
		t.StateDB.Done(searchURL, 200, EntitySearch)
	}

	emit(&SearchState{
		Query:     t.Query,
		Status:    "done",
		PinsFound: len(pins),
		Page:      len(pins) / 25,
	})
	return m, nil
}
