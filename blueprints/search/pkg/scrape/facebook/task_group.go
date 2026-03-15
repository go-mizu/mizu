package facebook

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type GroupState struct {
	URL        string
	Status     string
	PostsFound int
	Error      string
}

type GroupMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Posts   int
}

type GroupTask struct {
	URL         string
	Client      *Client
	DB          *DB
	StateDB     *State
	MaxPages    int
	MaxComments int
}

var _ core.Task[GroupState, GroupMetric] = (*GroupTask)(nil)

func (t *GroupTask) Run(ctx context.Context, emit func(*GroupState)) (GroupMetric, error) {
	var m GroupMetric
	target := NormalizeGroupURL(t.URL)
	emit(&GroupState{URL: target, Status: "fetching"})

	doc, code, err := t.Client.FetchHTML(ctx, target)
	if err != nil {
		m.Failed++
		emit(&GroupState{URL: target, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(target, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		emit(&GroupState{URL: target, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(target, EntityGroup, code)
		}
		return m, nil
	}

	group := ParseGroup(doc, target)
	if group.GroupID == "" {
		m.Failed++
		err := fmt.Errorf("could not infer group identity")
		emit(&GroupState{URL: target, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(target, err.Error())
		}
		return m, nil
	}
	if err := t.DB.UpsertGroup(*group); err != nil {
		m.Failed++
		emit(&GroupState{URL: target, Status: "failed", Error: err.Error()})
		return m, nil
	}

	posts, comments := collectFeed(ctx, t.Client, target, group.GroupID, group.Name, EntityGroup, maxOrDefault(t.MaxPages, DefaultMaxPages), t.MaxComments)
	for _, post := range posts {
		if err := t.DB.UpsertPost(post); err == nil {
			m.Posts++
		}
	}
	_ = t.DB.InsertComments(comments)
	if t.StateDB != nil {
		enqueueDiscoveredLinks(t.StateDB, doc, target)
		t.StateDB.Done(target, EntityGroup, code)
	}

	m.Fetched++
	emit(&GroupState{URL: target, Status: "done", PostsFound: m.Posts})
	return m, nil
}
