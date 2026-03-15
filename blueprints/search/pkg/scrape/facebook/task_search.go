package facebook

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type SearchState struct {
	Query        string
	Status       string
	Page         int
	ResultsFound int
	Error        string
}

type SearchMetric struct {
	Fetched int
	Failed  int
	Results int
}

type SearchTask struct {
	Query      string
	SearchType string
	Client     *Client
	DB         *DB
	StateDB    *State
	MaxPages   int
}

var _ core.Task[SearchState, SearchMetric] = (*SearchTask)(nil)

func (t *SearchTask) Run(ctx context.Context, emit func(*SearchState)) (SearchMetric, error) {
	var m SearchMetric
	if t.Query == "" {
		return m, fmt.Errorf("query is required")
	}
	maxPages := maxOrDefault(t.MaxPages, DefaultMaxPages)
	searchType := defaultString(t.SearchType, "top")
	nextURL := BuildSearchURL(t.Query, searchType, 1)
	page := 0

	for nextURL != "" && page < maxPages {
		page++
		emit(&SearchState{Query: t.Query, Status: "fetching", Page: page, ResultsFound: m.Results})

		doc, code, err := t.Client.FetchHTML(ctx, nextURL)
		if err != nil {
			m.Failed++
			emit(&SearchState{Query: t.Query, Status: "failed", Page: page, Error: err.Error()})
			return m, nil
		}
		if code == 404 || doc == nil {
			break
		}
		results := ParseSearchResults(doc, t.Query, nextURL)
		_ = t.DB.InsertSearchResults(results)
		for _, r := range results {
			m.Results++
			if t.StateDB != nil {
				_ = t.StateDB.Enqueue(r.ResultURL, r.EntityType, 4)
			}
		}

		nextURL = ParseNextPage(doc, nextURL)
	}

	m.Fetched++
	emit(&SearchState{Query: t.Query, Status: "done", Page: page, ResultsFound: m.Results})
	return m, nil
}
