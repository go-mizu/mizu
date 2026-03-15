package goodread

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// UserState is the observable state for a UserTask.
type UserState struct {
	URL    string
	Status string
	Error  string
}

// UserMetric is the final result of a UserTask.
type UserMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// UserTask fetches and stores a Goodreads user profile.
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

	userID := extractIDFromPath(t.URL, "/user/show/")
	if userID == "" {
		// Try profile URL: goodreads.com/<username>
		userID = extractUsernameFromURL(t.URL)
	}
	if userID == "" {
		m.Failed++
		emit(&UserState{URL: t.URL, Status: "failed", Error: "cannot extract user ID"})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, "cannot extract user ID")
		}
		return m, nil
	}

	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&UserState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || doc == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, code, "user")
		}
		return m, nil
	}

	user, err := ParseUser(doc, userID, t.URL)
	if err != nil {
		m.Failed++
		emit(&UserState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	t.DB.UpsertUser(*user)

	if t.StateDB != nil {
		t.StateDB.Done(t.URL, code, "user")
	}

	m.Fetched++
	emit(&UserState{URL: t.URL, Status: "done"})
	return m, nil
}

func extractUsernameFromURL(url string) string {
	// goodreads.com/<username> pattern
	if idx := lastIndex(url, "/"); idx >= 0 {
		part := url[idx+1:]
		if len(part) > 0 && !contains([]string{"show", "list", "author", "book", "series", "quotes", "genres", "user"}, part) {
			return part
		}
	}
	return ""
}

func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
