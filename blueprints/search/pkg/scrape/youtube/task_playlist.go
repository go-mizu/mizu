package youtube

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type PlaylistState struct {
	URL         string
	Status      string
	Error       string
	VideosFound int
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
	url := NormalizePlaylistURL(t.URL)
	emit(&PlaylistState{URL: url, Status: "fetching"})
	data, code, err := t.Client.FetchPageData(ctx, url)
	if err != nil {
		m.Failed++
		emit(&PlaylistState{URL: url, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(url, err.Error())
		}
		return m, nil
	}
	if code == 404 || data == nil {
		m.Skipped++
		emit(&PlaylistState{URL: url, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(url, code, EntityPlaylist)
		}
		return m, nil
	}
	playlist, edges, videos, err := ParsePlaylistPage(data, url)
	if err != nil {
		m.Failed++
		emit(&PlaylistState{URL: url, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(url, err.Error())
		}
		return m, nil
	}
	if err := t.DB.UpsertPlaylist(*playlist); err != nil {
		return m, err
	}
	if err := t.DB.InsertPlaylistVideos(edges); err != nil {
		return m, err
	}
	for _, video := range videos {
		_ = t.DB.UpsertVideo(video)
	}
	if t.StateDB != nil {
		for _, video := range videos {
			_ = t.StateDB.Enqueue(NormalizeVideoURL(video.VideoID), EntityVideo, 3)
		}
		t.StateDB.Done(url, code, EntityPlaylist)
	}
	m.Fetched++
	emit(&PlaylistState{URL: url, Status: "done", VideosFound: len(videos)})
	return m, nil
}
