package soundcloud

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type UserState struct {
	URL    string
	Status string
	Error  string
}

type UserMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type UserTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[UserState, UserMetric] = (*UserTask)(nil)

func (t *UserTask) Run(ctx context.Context, emit func(*UserState)) (UserMetric, error) {
	var m UserMetric
	emit(&UserState{URL: t.URL, Status: "fetching"})

	doc, body, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&UserState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&UserState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			_ = t.StateDB.Done(t.URL, code, EntityUser)
		}
		return m, nil
	}

	user, err := ParseUserPage(doc, body, t.URL)
	if err != nil {
		m.Failed++
		emit(&UserState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if err := t.DB.UpsertUser(*user); err != nil {
		return m, err
	}

	if t.StateDB != nil {
		_ = t.StateDB.EnqueueBatch(DiscoverQueueItems(doc))
		_ = t.StateDB.Done(t.URL, code, EntityUser)
	}

	m.Fetched++
	emit(&UserState{URL: t.URL, Status: "done"})
	return m, nil
}
