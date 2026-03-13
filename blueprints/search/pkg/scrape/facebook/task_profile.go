package facebook

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type ProfileState struct {
	URL        string
	Status     string
	PostsFound int
	Error      string
}

type ProfileMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Posts   int
}

type ProfileTask struct {
	URL         string
	Client      *Client
	DB          *DB
	StateDB     *State
	MaxPages    int
	MaxComments int
}

var _ core.Task[ProfileState, ProfileMetric] = (*ProfileTask)(nil)

func (t *ProfileTask) Run(ctx context.Context, emit func(*ProfileState)) (ProfileMetric, error) {
	var m ProfileMetric
	target := NormalizeProfileURL(t.URL)
	emit(&ProfileState{URL: target, Status: "fetching"})

	doc, code, err := t.Client.FetchHTML(ctx, target)
	if err != nil {
		m.Failed++
		emit(&ProfileState{URL: target, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(target, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		emit(&ProfileState{URL: target, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(target, EntityProfile, code)
		}
		return m, nil
	}

	profile := ParseProfile(doc, target)
	if profile.ProfileID == "" {
		m.Failed++
		err := fmt.Errorf("could not infer profile identity")
		emit(&ProfileState{URL: target, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(target, err.Error())
		}
		return m, nil
	}
	if err := t.DB.UpsertProfile(*profile); err != nil {
		m.Failed++
		emit(&ProfileState{URL: target, Status: "failed", Error: err.Error()})
		return m, nil
	}

	posts, comments := collectFeed(ctx, t.Client, target, profile.ProfileID, profile.Name, EntityProfile, maxOrDefault(t.MaxPages, DefaultMaxPages), t.MaxComments)
	for _, post := range posts {
		if err := t.DB.UpsertPost(post); err == nil {
			m.Posts++
		}
	}
	_ = t.DB.InsertComments(comments)
	if t.StateDB != nil {
		enqueueDiscoveredLinks(t.StateDB, doc, target)
		t.StateDB.Done(target, EntityProfile, code)
	}

	m.Fetched++
	emit(&ProfileState{URL: target, Status: "done", PostsFound: m.Posts})
	return m, nil
}
