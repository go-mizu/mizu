package soundcloud

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type PlaylistState struct {
	URL         string
	Status      string
	Error       string
	TracksFound int
}

type PlaylistMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type PlaylistTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[PlaylistState, PlaylistMetric] = (*PlaylistTask)(nil)

func (t *PlaylistTask) Run(ctx context.Context, emit func(*PlaylistState)) (PlaylistMetric, error) {
	var m PlaylistMetric
	emit(&PlaylistState{URL: t.URL, Status: "fetching"})

	doc, body, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&PlaylistState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&PlaylistState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			_ = t.StateDB.Done(t.URL, code, EntityPlaylist)
		}
		return m, nil
	}

	playlist, user, rels, err := ParsePlaylistPage(doc, body, t.URL)
	if err != nil {
		m.Failed++
		emit(&PlaylistState{URL: t.URL, Status: "failed", Error: err.Error()})
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
	if err := t.DB.UpsertPlaylist(*playlist); err != nil {
		return m, err
	}
	if err := t.DB.InsertPlaylistTracks(rels); err != nil {
		return m, err
	}

	if t.StateDB != nil {
		var items []QueueItem
		if user != nil && user.URL != "" {
			items = append(items, QueueItem{URL: user.URL, EntityType: EntityUser, Priority: 5})
		}
		for _, rel := range rels {
			if rel.TrackURL == "" {
				continue
			}
			items = append(items, QueueItem{URL: rel.TrackURL, EntityType: EntityTrack, Priority: 4})
		}
		items = append(items, DiscoverQueueItems(doc)...)
		_ = t.StateDB.EnqueueBatch(items)
		_ = t.StateDB.Done(t.URL, code, EntityPlaylist)
	}

	m.Fetched++
	emit(&PlaylistState{URL: t.URL, Status: "done", TracksFound: len(rels)})
	return m, nil
}
