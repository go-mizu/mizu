package goodread

import (
	"context"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// AuthorState is the observable state for an AuthorTask.
type AuthorState struct {
	URL    string
	Status string
	Error  string
}

// AuthorMetric is the final result of an AuthorTask.
type AuthorMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// AuthorTask fetches and stores a single Goodreads author page.
type AuthorTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[AuthorState, AuthorMetric] = (*AuthorTask)(nil)

func (t *AuthorTask) Run(ctx context.Context, emit func(*AuthorState)) (AuthorMetric, error) {
	var m AuthorMetric

	emit(&AuthorState{URL: t.URL, Status: "fetching"})

	authorID := extractIDFromPath(t.URL, "/author/show/")
	if authorID == "" {
		m.Failed++
		emit(&AuthorState{URL: t.URL, Status: "failed", Error: "cannot extract author ID"})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, "cannot extract author ID")
		}
		return m, nil
	}

	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&AuthorState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		emit(&AuthorState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, code, "author")
		}
		return m, nil
	}

	emit(&AuthorState{URL: t.URL, Status: "parsing"})

	author, err := ParseAuthor(doc, authorID, t.URL)
	if err != nil {
		m.Failed++
		emit(&AuthorState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	if err := t.DB.UpsertAuthor(*author); err != nil {
		m.Failed++
		emit(&AuthorState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// Enqueue author's books
	if t.StateDB != nil {
		enqueueAuthorBooks(doc, t.StateDB)
		t.StateDB.Done(t.URL, code, "author")
	}

	m.Fetched++
	emit(&AuthorState{URL: t.URL, Status: "done"})
	return m, nil
}

func enqueueAuthorBooks(doc *goquery.Document, stateDB *State) {
	seen := map[string]bool{}
	doc.Find("a[href*='/book/show/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if id := extractIDFromPath(href, "/book/show/"); id != "" && !seen[id] {
			seen[id] = true
			stateDB.Enqueue(BaseURL+"/book/show/"+id, "book", 4)
		}
	})
	_ = fmt.Sprintf // ensure fmt is used
}
