package soundcloud

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type TrackState struct {
	URL      string
	Status   string
	Error    string
	Comments int
}

type TrackMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type TrackTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[TrackState, TrackMetric] = (*TrackTask)(nil)

func (t *TrackTask) Run(ctx context.Context, emit func(*TrackState)) (TrackMetric, error) {
	var m TrackMetric
	emit(&TrackState{URL: t.URL, Status: "fetching"})

	doc, body, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&TrackState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&TrackState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			_ = t.StateDB.Done(t.URL, code, EntityTrack)
		}
		return m, nil
	}

	track, user, comments, err := ParseTrackPage(doc, body, t.URL)
	if err != nil {
		m.Failed++
		emit(&TrackState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	if user != nil {
		if err := t.DB.UpsertUser(*user); err != nil {
			return m, err
		}
	}
	if err := t.DB.UpsertTrack(*track); err != nil {
		return m, err
	}
	if err := t.DB.InsertComments(comments); err != nil {
		return m, err
	}

	if t.StateDB != nil {
		items := DiscoverQueueItems(doc)
		if user != nil && user.URL != "" {
			items = append(items, QueueItem{URL: user.URL, EntityType: EntityUser, Priority: 5})
		}
		_ = t.StateDB.EnqueueBatch(items)
		_ = t.StateDB.Done(t.URL, code, EntityTrack)
	}

	m.Fetched++
	emit(&TrackState{URL: t.URL, Status: "done", Comments: len(comments)})
	return m, nil
}

func FetchTrack(ctx context.Context, client *Client, db *DB, stateDB *State, rawURL string) (*Track, error) {
	task := &TrackTask{URL: rawURL, Client: client, DB: db, StateDB: stateDB}
	m, err := task.Run(ctx, func(*TrackState) {})
	if err != nil {
		return nil, err
	}
	if m.Failed > 0 {
		return nil, fmt.Errorf("failed to fetch track")
	}
	return &Track{URL: rawURL}, nil
}
