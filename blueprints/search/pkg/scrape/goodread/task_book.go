package goodread

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// BookState is the observable state for a BookTask.
type BookState struct {
	URL    string
	Status string
	Error  string
}

// BookMetric is the final result of a BookTask.
type BookMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// BookTask fetches and stores a single Goodreads book page.
type BookTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State // optional; if set, marks visited + enqueues discovered links
}

var _ core.Task[BookState, BookMetric] = (*BookTask)(nil)

func (t *BookTask) Run(ctx context.Context, emit func(*BookState)) (BookMetric, error) {
	var m BookMetric

	emit(&BookState{URL: t.URL, Status: "fetching"})

	bookID := extractIDFromPath(t.URL, "/book/show/")
	if bookID == "" {
		// Strip query params and try again
		u := t.URL
		if idx := strings.Index(u, "?"); idx > 0 {
			u = u[:idx]
		}
		bookID = extractIDFromPath(u, "/book/show/")
	}
	if bookID == "" {
		m.Failed++
		emit(&BookState{URL: t.URL, Status: "failed", Error: "cannot extract book ID"})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, "cannot extract book ID")
		}
		return m, nil
	}

	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&BookState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&BookState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, 404, "book")
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		emit(&BookState{URL: t.URL, Status: "failed", Error: fmt.Sprintf("HTTP %d", code)})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, fmt.Sprintf("HTTP %d", code))
		}
		return m, nil
	}

	emit(&BookState{URL: t.URL, Status: "parsing"})

	book, err := ParseBook(doc, bookID, t.URL)
	if err != nil {
		m.Failed++
		emit(&BookState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}


	// Extract and store reviews — batched in one transaction with the book
	reviews := ParseReviews(doc, bookID)
	if err := t.DB.UpsertBookWithReviews(*book, reviews); err != nil {
		m.Failed++
		emit(&BookState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// Enqueue discovered links and mark done in one transaction
	if t.StateDB != nil {
		links := t.collectLinks(book)
		t.StateDB.DoneAndEnqueue(t.URL, code, "book", links)
	}

	m.Fetched++
	emit(&BookState{URL: t.URL, Status: "done"})
	return m, nil
}

func (t *BookTask) collectLinks(b *Book) []QueueItem {
	var links []QueueItem
	// Author page
	if b.AuthorID != "" {
		links = append(links, QueueItem{
			URL:        BaseURL + "/author/show/" + b.AuthorID,
			EntityType: "author",
			Priority:   5,
		})
	}
	// Series page
	if b.SeriesID != "" {
		links = append(links, QueueItem{
			URL:        BaseURL + "/series/" + b.SeriesID,
			EntityType: "series",
			Priority:   3,
		})
	}
	// Similar books removed — they cause unbounded queue growth.
	// Books are already seeded from sitemaps.
	return links
}

// FetchBook is a convenience wrapper for fetching a single book from the CLI.
func FetchBook(ctx context.Context, client *Client, db *DB, stateDB *State, url string) (*Book, error) {
	task := &BookTask{
		URL:     url,
		Client:  client,
		DB:      db,
		StateDB: stateDB,
	}
	m, err := task.Run(ctx, func(*BookState) {})
	if err != nil {
		return nil, err
	}
	if m.Failed > 0 {
		return nil, fmt.Errorf("failed to fetch book")
	}

	// Retrieve from DB
	bookID := extractIDFromPath(url, "/book/show/")
	rows, err := db.db.Query(`SELECT book_id, title, author_name, avg_rating FROM books WHERE book_id = ?`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var b Book
		b.FetchedAt = time.Now()
		rows.Scan(&b.BookID, &b.Title, &b.AuthorName, &b.AvgRating)
		return &b, nil
	}
	return nil, fmt.Errorf("book not found in DB after fetch")
}
