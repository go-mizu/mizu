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

	// Store book
	if err := t.DB.UpsertBook(*book); err != nil {
		m.Failed++
		emit(&BookState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// Extract and store reviews
	reviews := ParseReviews(doc, bookID)
	for _, r := range reviews {
		t.DB.UpsertReview(r)
	}

	// Enqueue discovered links
	if t.StateDB != nil {
		t.enqueueLinks(book)
		t.StateDB.Done(t.URL, code, "book")
	}

	m.Fetched++
	emit(&BookState{URL: t.URL, Status: "done"})
	return m, nil
}

func (t *BookTask) enqueueLinks(b *Book) {
	// Author page
	if b.AuthorID != "" {
		url := BaseURL + "/author/show/" + b.AuthorID
		t.StateDB.Enqueue(url, "author", 5)
	}
	// Series page
	if b.SeriesID != "" {
		url := BaseURL + "/series/" + b.SeriesID
		t.StateDB.Enqueue(url, "series", 3)
	}
	// Similar books
	for _, id := range b.SimilarBookIDs {
		url := BaseURL + "/book/show/" + id
		t.StateDB.Enqueue(url, "book", 2)
	}
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
