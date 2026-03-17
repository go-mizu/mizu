package discord

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type UserState struct {
	UserID   string
	Username string
	Status   string
	Error    string
}

type UserMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type UserTask struct {
	ID      string // raw user ID or discord://users/{id} URL
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[UserState, UserMetric] = (*UserTask)(nil)

func (t *UserTask) Run(ctx context.Context, emit func(*UserState)) (UserMetric, error) {
	var m UserMetric

	ref, err := ParseRef(t.ID, EntityUser)
	if err != nil {
		return m, err
	}

	if t.StateDB != nil && t.StateDB.IsVisited(ref.URL) {
		_ = t.StateDB.Done(ref.URL, 200, EntityUser)
		m.Skipped++
		emit(&UserState{UserID: ref.ID, Status: "skipped"})
		return m, nil
	}

	emit(&UserState{UserID: ref.ID, Status: "fetching"})

	raw, code, err := t.Client.FetchUser(ctx, ref.ID)
	if err != nil {
		m.Failed++
		if t.StateDB != nil {
			t.StateDB.Fail(ref.URL, err.Error())
		}
		emit(&UserState{UserID: ref.ID, Status: "failed", Error: err.Error()})
		return m, nil
	}
	if code == 404 || raw == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(ref.URL, 404, EntityUser)
		}
		emit(&UserState{UserID: ref.ID, Status: "not_found"})
		return m, nil
	}

	user := ParseUser(raw)
	if user.UserID == "" {
		user.UserID = ref.ID
	}
	user.FetchedAt = time.Now()

	if err := t.DB.UpsertUser(user); err != nil {
		m.Failed++
		return m, err
	}

	if t.StateDB != nil {
		t.StateDB.Done(ref.URL, 200, EntityUser)
	}
	m.Fetched++
	emit(&UserState{UserID: ref.ID, Username: user.Username, Status: "done"})
	return m, nil
}
