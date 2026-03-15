package ebay

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// SearchState is the observable state for a SearchTask.
type SearchState struct {
	Query      string
	Status     string
	Error      string
	Page       int
	ItemsFound int
}

// SearchMetric is the final result of a SearchTask.
type SearchMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Pages   int
}

// SearchTask fetches and stores eBay search-result pages.
type SearchTask struct {
	Query   string
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
	Config  Config
}

var _ core.Task[SearchState, SearchMetric] = (*SearchTask)(nil)

func (t *SearchTask) Run(ctx context.Context, emit func(*SearchState)) (SearchMetric, error) {
	var m SearchMetric

	query := t.Query
	startPage := 1
	startURL := t.URL

	if startURL != "" {
		var err error
		query, startPage, err = ParseSearchURL(startURL)
		if err != nil {
			m.Failed++
			emit(&SearchState{Status: "failed", Error: err.Error()})
			if t.StateDB != nil {
				t.StateDB.Fail(startURL, err.Error())
			}
			return m, nil
		}
	}
	if query == "" {
		m.Failed++
		msg := "missing search query"
		emit(&SearchState{Status: "failed", Error: msg})
		return m, nil
	}
	if startURL == "" {
		startURL = SearchURL(query, startPage)
	}

	if t.StateDB != nil && t.StateDB.IsVisited(startURL) {
		m.Skipped++
		emit(&SearchState{Query: query, Status: "skipped", Page: startPage})
		return m, nil
	}

	maxPages := t.Config.MaxPages
	if maxPages <= 0 {
		maxPages = DefaultConfig().MaxPages
	}

	currentURL := startURL
	currentPage := startPage
	visitedStart := false

	for currentURL != "" && currentPage < startPage+maxPages {
		emit(&SearchState{Query: query, Status: "fetching", Page: currentPage})

		doc, code, err := t.Client.FetchHTML(ctx, currentURL)
		if err != nil {
			m.Failed++
			emit(&SearchState{Query: query, Status: "failed", Page: currentPage, Error: err.Error()})
			if t.StateDB != nil {
				t.StateDB.Fail(startURL, err.Error())
			}
			return m, nil
		}
		if code == 404 || doc == nil {
			break
		}

		sr, itemURLs, nextPageURL, err := ParseSearch(doc, query, currentPage, currentURL)
		if err != nil {
			m.Failed++
			emit(&SearchState{Query: query, Status: "failed", Page: currentPage, Error: err.Error()})
			if t.StateDB != nil {
				t.StateDB.Fail(startURL, err.Error())
			}
			return m, nil
		}

		if err := t.DB.UpsertSearchResult(sr); err != nil {
			m.Failed++
			emit(&SearchState{Query: query, Status: "failed", Page: currentPage, Error: err.Error()})
			if t.StateDB != nil {
				t.StateDB.Fail(startURL, err.Error())
			}
			return m, nil
		}

		if t.StateDB != nil && len(itemURLs) > 0 {
			items := make([]QueueItem, 0, len(itemURLs))
			for _, itemURL := range itemURLs {
				items = append(items, QueueItem{
					URL:        itemURL,
					EntityType: EntityItem,
					Priority:   10,
				})
			}
			_ = t.StateDB.EnqueueBatch(items)
		}

		m.Fetched += len(itemURLs)
		m.Pages++
		emit(&SearchState{
			Query:      query,
			Status:     "parsed",
			Page:       currentPage,
			ItemsFound: len(itemURLs),
		})

		if !visitedStart && t.StateDB != nil {
			_ = t.StateDB.Done(startURL, EntitySearch, code)
			visitedStart = true
		}

		if nextPageURL == "" || currentPage >= startPage+maxPages-1 {
			break
		}
		currentURL = nextPageURL
		currentPage++
	}

	if t.StateDB != nil && !visitedStart {
		_ = t.StateDB.Done(startURL, EntitySearch, 200)
	}

	emit(&SearchState{Query: query, Status: "done", Page: currentPage, ItemsFound: m.Fetched})
	return m, nil
}
