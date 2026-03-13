package pinterest

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// UserState is the observable state for a UserTask.
type UserState struct {
	URL         string
	Username    string
	Status      string
	Error       string
	BoardsFound int
}

// UserMetric is the final result of a UserTask.
type UserMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// UserTask fetches a Pinterest user profile and their boards, then enqueues each board.
type UserTask struct {
	URL           string // full user URL or bare username
	IncludeBoards bool   // enqueue each board into the crawl queue
	Client        *Client
	DB            *DB
	StateDB       *State // optional; marks visited + enqueues boards
}

var _ core.Task[UserState, UserMetric] = (*UserTask)(nil)

func (t *UserTask) Run(ctx context.Context, emit func(*UserState)) (UserMetric, error) {
	var m UserMetric

	username := ExtractUsername(t.URL)
	userURL := NormalizeUserURL(username)

	if t.StateDB != nil && t.StateDB.IsVisited(userURL) {
		m.Skipped++
		emit(&UserState{URL: userURL, Username: username, Status: "skipped"})
		return m, nil
	}

	emit(&UserState{URL: userURL, Username: username, Status: "fetching_profile"})

	user, boards, err := t.Client.FetchUserBootstrap(ctx, username)
	if err != nil {
		m.Failed++
		emit(&UserState{URL: userURL, Username: username, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(userURL, err.Error())
		}
		return m, nil
	}

	if err := t.DB.UpsertUser(*user); err != nil {
		m.Failed++
		emit(&UserState{URL: userURL, Username: username, Status: "failed", Error: err.Error()})
		return m, nil
	}
	m.Fetched++

	emit(&UserState{URL: userURL, Username: username, Status: "fetching_boards"})

	var totalBoards int
	for _, board := range boards {
		if ctx.Err() != nil {
			break
		}
		if board.IsSecret {
			continue
		}
		if err := t.DB.UpsertBoard(board); err != nil {
			continue
		}
		totalBoards++

		if t.StateDB != nil && t.IncludeBoards && board.URL != "" {
			t.StateDB.Enqueue(board.URL, EntityBoard, 10)
		}
	}

	if t.StateDB != nil {
		t.StateDB.Done(userURL, 200, EntityUser)
	}

	emit(&UserState{URL: userURL, Username: username, Status: "done", BoardsFound: totalBoards})
	return m, nil
}
