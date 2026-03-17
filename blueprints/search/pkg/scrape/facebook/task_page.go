package facebook

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type PageState struct {
	URL        string
	Status     string
	PostsFound int
	Error      string
}

type PageMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Posts   int
}

type PageTask struct {
	URL         string
	Client      *Client
	DB          *DB
	StateDB     *State
	MaxPages    int
	MaxComments int
}

var _ core.Task[PageState, PageMetric] = (*PageTask)(nil)

func (t *PageTask) Run(ctx context.Context, emit func(*PageState)) (PageMetric, error) {
	var m PageMetric
	target := NormalizePageURL(t.URL)
	emit(&PageState{URL: target, Status: "fetching"})

	doc, code, err := t.Client.FetchHTML(ctx, target)
	if err != nil {
		m.Failed++
		emit(&PageState{URL: target, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(target, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		emit(&PageState{URL: target, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(target, EntityPage, code)
		}
		return m, nil
	}

	page := ParsePage(doc, target)
	if page.PageID == "" {
		m.Failed++
		err := fmt.Errorf("could not infer page identity")
		emit(&PageState{URL: target, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(target, err.Error())
		}
		return m, nil
	}
	if err := t.DB.UpsertPage(*page); err != nil {
		m.Failed++
		emit(&PageState{URL: target, Status: "failed", Error: err.Error()})
		return m, nil
	}

	posts, comments := collectFeed(ctx, t.Client, target, page.PageID, page.Name, EntityPage, maxOrDefault(t.MaxPages, DefaultMaxPages), t.MaxComments)
	for _, post := range posts {
		if err := t.DB.UpsertPost(post); err == nil {
			m.Posts++
		}
	}
	_ = t.DB.InsertComments(comments)
	if t.StateDB != nil {
		enqueueDiscoveredLinks(t.StateDB, doc, target)
		t.StateDB.Done(target, EntityPage, code)
	}

	m.Fetched++
	emit(&PageState{URL: target, Status: "done", PostsFound: m.Posts})
	return m, nil
}
