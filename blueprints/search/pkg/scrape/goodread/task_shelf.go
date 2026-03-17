package goodread

import (
	"context"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// ShelfState is the observable state for a ShelfTask.
type ShelfState struct {
	URL        string
	Status     string
	Error      string
	BooksFound int
	Pages      int
}

// ShelfMetric is the final result of a ShelfTask.
type ShelfMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Pages   int
}

// ShelfTask fetches and stores a Goodreads user shelf, following pagination.
type ShelfTask struct {
	URL       string
	UserID    string
	ShelfName string
	Client    *Client
	DB        *DB
	StateDB   *State
	MaxPages  int // 0 = unlimited
}

var _ core.Task[ShelfState, ShelfMetric] = (*ShelfTask)(nil)

func (t *ShelfTask) Run(ctx context.Context, emit func(*ShelfState)) (ShelfMetric, error) {
	var m ShelfMetric

	currentURL := t.URL
	shelfName := t.ShelfName
	userID := t.UserID

	// Parse userID and shelfName from URL if not provided
	if userID == "" {
		userID = extractIDFromPath(currentURL, "/review/list/")
	}
	if shelfName == "" {
		shelfName = extractQueryParam(currentURL, "shelf")
		if shelfName == "" {
			shelfName = "read"
		}
	}

	var allBooks []ShelfBook

	for page := 1; ; page++ {
		if t.MaxPages > 0 && page > t.MaxPages {
			break
		}

		emit(&ShelfState{URL: currentURL, Status: "fetching", Pages: page})

		doc, code, err := t.Client.FetchHTML(ctx, currentURL)
		if err != nil {
			m.Failed++
			emit(&ShelfState{URL: currentURL, Status: "failed", Error: err.Error()})
			if t.StateDB != nil {
				t.StateDB.Fail(t.URL, err.Error())
			}
			return m, nil
		}
		if code == 404 || doc == nil {
			m.Skipped++
			break
		}

		shelf, books, err := ParseShelf(doc, userID, shelfName, currentURL)
		if err != nil {
			m.Failed++
			break
		}

		if page == 1 {
			t.DB.UpsertShelf(*shelf)
		}
		allBooks = append(allBooks, books...)

		emit(&ShelfState{URL: currentURL, Status: "parsing", BooksFound: len(allBooks), Pages: page})

		if len(books) == 0 {
			break
		}

		next := ParseShelfNextPage(doc)
		if next == "" {
			break
		}
		currentURL = next
	}

	t.DB.InsertShelfBooks(allBooks)

	if t.StateDB != nil {
		t.StateDB.Done(t.URL, 200, "shelf")
	}

	m.Fetched++
	m.Pages = m.Fetched
	emit(&ShelfState{URL: t.URL, Status: "done", BooksFound: len(allBooks)})
	return m, nil
}

// extractQueryParam returns the value of a URL query parameter.
func extractQueryParam(rawURL, param string) string {
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		query := rawURL[idx+1:]
		for _, part := range strings.Split(query, "&") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 && kv[0] == param {
				return kv[1]
			}
		}
	}
	return ""
}
