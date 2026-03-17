package soundcloud

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type SearchState struct {
	Query        string
	Status       string
	Error        string
	ResultsFound int
}

type SearchMetric struct {
	Fetched int
	Failed  int
}

type SearchTask struct {
	Query   string
	Kind    string
	Limit   int
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[SearchState, SearchMetric] = (*SearchTask)(nil)

func (t *SearchTask) Run(ctx context.Context, emit func(*SearchState)) (SearchMetric, error) {
	var m SearchMetric
	emit(&SearchState{Query: t.Query, Status: "searching"})

	results, err := t.Client.Search(ctx, t.Query, t.Kind, t.Limit)
	if err != nil {
		m.Failed++
		emit(&SearchState{Query: t.Query, Status: "failed", Error: err.Error()})
		return m, nil
	}
	if err := t.DB.UpsertSearchResults(results); err != nil {
		return m, err
	}
	if t.StateDB != nil {
		items := make([]QueueItem, 0, len(results))
		for _, r := range results {
			items = append(items, QueueItem{URL: r.URL, EntityType: r.Kind, Priority: 5})
		}
		_ = t.StateDB.EnqueueBatch(items)
	}

	m.Fetched = len(results)
	emit(&SearchState{Query: t.Query, Status: "done", ResultsFound: len(results)})
	return m, nil
}
