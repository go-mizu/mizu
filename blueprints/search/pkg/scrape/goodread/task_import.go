package goodread

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// ImportState is the observable state for an ImportTask.
type ImportState struct {
	Imported int64
	Failed   int64
	RPS      float64
}

// ImportMetric is the final result of an ImportTask.
type ImportMetric struct {
	Imported int64
	Failed   int64
	Duration time.Duration
}

// ImportTask runs Phase 2 of the two-phase pipeline: reads cached HTML files,
// parses them in parallel, batch-writes to DuckDB in a single transaction per
// batch, marks items done, enqueues discovered links, and deletes the HTML files.
type ImportTask struct {
	Config    Config
	DB        *DB
	StateDB   *State
	BatchSize int // items per transaction (default 100)

	// FetchDone is an optional channel that is closed when the fetch phase is
	// complete. When set, ImportTask will poll for new fetched items instead of
	// exiting immediately when the queue is momentarily empty.
	FetchDone <-chan struct{}
}

var _ core.Task[ImportState, ImportMetric] = (*ImportTask)(nil)

// importResult holds parsed output for one queue item.
type importResult struct {
	item    QueueItem
	links   []QueueItem
	writeFn func(tx *sql.Tx) error // nil if err != nil
	err     error
}

func (t *ImportTask) Run(ctx context.Context, emit func(*ImportState)) (ImportMetric, error) {
	start := time.Now()

	var (
		imported atomic.Int64
		failed   atomic.Int64
	)

	batchSize := t.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	// Parse workers — CPU-bound, cap at NumCPU to avoid over-subscribing cores
	// when FetchTask is running concurrently with many goroutines.
	parseWorkers := runtime.NumCPU()
	if parseWorkers < 4 {
		parseWorkers = 4
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(start).Seconds()
				rps := 0.0
				if elapsed > 0 {
					rps = float64(imported.Load()) / elapsed
				}
				emit(&ImportState{
					Imported: imported.Load(),
					Failed:   failed.Load(),
					RPS:      rps,
				})
			}
		}
	}()

	for {
		if ctx.Err() != nil {
			break
		}

		items, err := t.StateDB.PopFetched(batchSize)
		if err != nil {
			return ImportMetric{}, fmt.Errorf("pop fetched: %w", err)
		}
		if len(items) == 0 {
			if t.FetchDone != nil {
				// Fetch still running — wait for more items or fetch completion.
				select {
				case <-t.FetchDone:
					// Fetch phase finished; do one final drain then exit.
					items, _ = t.StateDB.PopFetched(batchSize)
					if len(items) == 0 {
						goto done
					}
				case <-ctx.Done():
					goto done
				case <-time.After(500 * time.Millisecond):
					continue
				}
			} else {
				break
			}
		}

		// ── Phase A: parse concurrently ──────────────────────────────
		results := make([]importResult, len(items))
		var wg sync.WaitGroup
		sem := make(chan struct{}, parseWorkers)
		for i, item := range items {
			i, item := i, item
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				results[i].item = item
				links, writeFn, parseErr := t.parseItem(item)
				results[i].links = links
				results[i].writeFn = writeFn
				results[i].err = parseErr
			}()
		}
		wg.Wait()

		// ── Phase B: batch write in one transaction ──────────────────
		tx, txErr := t.DB.db.Begin()
		if txErr != nil {
			// Can't open tx — mark everything as failed and retry later.
			for i := range results {
				t.StateDB.Fail(results[i].item.URL, txErr.Error())
				failed.Add(1)
			}
			continue
		}

		commitFailed := false
		for i := range results {
			if results[i].err != nil || results[i].writeFn == nil {
				continue // will be handled below
			}
			if err := results[i].writeFn(tx); err != nil {
				results[i].err = fmt.Errorf("write: %w", err)
			}
		}
		if err := tx.Commit(); err != nil {
			commitFailed = true
			tx.Rollback() //nolint:errcheck
			for i := range results {
				if results[i].err == nil {
					results[i].err = fmt.Errorf("batch commit: %w", err)
				}
			}
		}
		_ = commitFailed

		// ── Phase C: update state, delete HTML ───────────────────────
		for i := range results {
			r := &results[i]
			if r.err != nil {
				failed.Add(1)
				t.StateDB.Fail(r.item.URL, r.err.Error())
				continue
			}
			if err := t.StateDB.DoneAndEnqueue(r.item.URL, 200, r.item.EntityType, r.links); err != nil {
				failed.Add(1)
			} else {
				imported.Add(1)
				if r.item.HtmlPath != "" {
					DeleteHTML(r.item.HtmlPath) //nolint:errcheck
				}
			}
		}
	}
done:
	return ImportMetric{
		Imported: imported.Load(),
		Failed:   failed.Load(),
		Duration: time.Since(start),
	}, nil
}

// parseItem loads HTML from disk and parses it into an entity, returning
// the discovered links and a closure that writes the entity to a given tx.
func (t *ImportTask) parseItem(item QueueItem) ([]QueueItem, func(*sql.Tx) error, error) {
	if item.HtmlPath == "" {
		return nil, nil, fmt.Errorf("no html_path for %s", item.URL)
	}
	html, err := LoadHTML(item.HtmlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load html: %w", err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, nil, fmt.Errorf("parse html: %w", err)
	}
	return t.dispatchParse(item, doc)
}

func (t *ImportTask) dispatchParse(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	switch item.EntityType {
	case "book":
		return parseBook2(item, doc)
	case "author":
		return parseAuthor2(item, doc)
	case "series":
		return parseSeries2(item, doc)
	case "list":
		return parseList2(item, doc)
	case "quote":
		return parseQuote2(item, doc)
	case "user":
		return parseUser2(item, doc)
	case "genre":
		return parseGenre2(item, doc)
	case "shelf":
		return parseShelf2(item, doc)
	default:
		// Unknown entity — no-op write
		return nil, func(tx *sql.Tx) error { return nil }, nil
	}
}

func parseBook2(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	bookID := extractIDFromPath(item.URL, "/book/show/")
	if bookID == "" {
		return nil, nil, fmt.Errorf("cannot extract book ID from %s", item.URL)
	}
	book, err := ParseBook(doc, bookID, item.URL)
	if err != nil {
		return nil, nil, err
	}
	reviews := ParseReviews(doc, bookID)

	var links []QueueItem
	if book.AuthorID != "" {
		links = append(links, QueueItem{URL: BaseURL + "/author/show/" + book.AuthorID, EntityType: "author", Priority: 5})
	}
	if book.SeriesID != "" {
		links = append(links, QueueItem{URL: BaseURL + "/series/" + book.SeriesID, EntityType: "series", Priority: 3})
	}

	b := *book
	writeFn := func(tx *sql.Tx) error {
		return insertBookWithReviewsTx(tx, b, reviews)
	}
	return links, writeFn, nil
}

func parseAuthor2(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	authorID := extractIDFromPath(item.URL, "/author/show/")
	if authorID == "" {
		return nil, nil, fmt.Errorf("cannot extract author ID from %s", item.URL)
	}
	author, err := ParseAuthor(doc, authorID, item.URL)
	if err != nil {
		return nil, nil, err
	}
	a := *author

	// Enqueue books found on the author page
	var links []QueueItem
	seen := map[string]bool{}
	doc.Find("a[href*='/book/show/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if id := extractIDFromPath(href, "/book/show/"); id != "" && !seen[id] {
			seen[id] = true
			links = append(links, QueueItem{URL: BaseURL + "/book/show/" + id, EntityType: "book", Priority: 4})
		}
	})

	writeFn := func(tx *sql.Tx) error {
		return insertAuthorTx(tx, a)
	}
	return links, writeFn, nil
}

func parseSeries2(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	seriesID := extractIDFromPath(item.URL, "/series/")
	if seriesID == "" {
		return nil, nil, fmt.Errorf("cannot extract series ID from %s", item.URL)
	}
	series, books, err := ParseSeries(doc, seriesID, item.URL)
	if err != nil {
		return nil, nil, err
	}
	s := *series
	writeFn := func(tx *sql.Tx) error {
		return insertSeriesTx(tx, s, books)
	}
	return nil, writeFn, nil
}

func parseList2(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	listID := extractIDFromPath(item.URL, "/list/show/")
	if listID == "" {
		return nil, nil, fmt.Errorf("cannot extract list ID from %s", item.URL)
	}
	list, listBooks, err := ParseList(doc, listID, item.URL)
	if err != nil {
		return nil, nil, err
	}
	l := *list
	writeFn := func(tx *sql.Tx) error {
		return insertListTx(tx, l, listBooks)
	}
	return nil, writeFn, nil
}

func parseQuote2(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	quotes, err := ParseQuotes(doc, item.URL)
	if err != nil {
		return nil, nil, err
	}
	writeFn := func(tx *sql.Tx) error {
		return insertQuotesTx(tx, quotes)
	}
	return nil, writeFn, nil
}

func parseUser2(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	userID := extractIDFromPath(item.URL, "/user/show/")
	if userID == "" {
		return nil, nil, fmt.Errorf("cannot extract user ID from %s", item.URL)
	}
	user, err := ParseUser(doc, userID, item.URL)
	if err != nil {
		return nil, nil, err
	}
	u := *user
	writeFn := func(tx *sql.Tx) error {
		return insertUserTx(tx, u)
	}
	return nil, writeFn, nil
}

func parseGenre2(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	slug := entityIDFromURL(item.URL)
	if slug == "" {
		return nil, nil, fmt.Errorf("cannot extract genre slug from %s", item.URL)
	}
	genre, _, err := ParseGenre(doc, slug, item.URL)
	if err != nil {
		return nil, nil, err
	}
	g := *genre
	writeFn := func(tx *sql.Tx) error {
		return insertGenreTx(tx, g)
	}
	return nil, writeFn, nil
}

func parseShelf2(item QueueItem, doc *goquery.Document) ([]QueueItem, func(*sql.Tx) error, error) {
	userID := extractIDFromPath(item.URL, "/review/list/")
	shelfName := extractQueryParam(item.URL, "shelf")
	if shelfName == "" {
		shelfName = "read"
	}
	shelf, shelfBooks, err := ParseShelf(doc, userID, shelfName, item.URL)
	if err != nil {
		return nil, nil, err
	}
	s := *shelf
	writeFn := func(tx *sql.Tx) error {
		return insertShelfTx(tx, s, shelfBooks)
	}
	return nil, writeFn, nil
}
