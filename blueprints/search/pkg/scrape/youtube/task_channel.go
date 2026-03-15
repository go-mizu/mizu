package youtube

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type ChannelState struct {
	URL         string
	Status      string
	Error       string
	VideosFound int
}

type ChannelMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type ChannelTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[ChannelState, ChannelMetric] = (*ChannelTask)(nil)

func (t *ChannelTask) Run(ctx context.Context, emit func(*ChannelState)) (ChannelMetric, error) {
	var m ChannelMetric
	url := NormalizeChannelURL(t.URL)
	emit(&ChannelState{URL: url, Status: "fetching"})
	data, code, err := t.Client.FetchPageData(ctx, url)
	if err != nil {
		m.Failed++
		emit(&ChannelState{URL: url, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(url, err.Error())
		}
		return m, nil
	}
	if code == 404 || data == nil {
		m.Skipped++
		emit(&ChannelState{URL: url, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(url, code, EntityChannel)
		}
		return m, nil
	}
	channel, videos, err := ParseChannelPage(data, url)
	if err != nil {
		m.Failed++
		emit(&ChannelState{URL: url, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(url, err.Error())
		}
		return m, nil
	}
	if err := t.DB.UpsertChannel(*channel); err != nil {
		return m, err
	}
	for _, video := range videos {
		_ = t.DB.UpsertVideo(video)
	}
	if t.StateDB != nil {
		if channel.UploadsPlaylistID != "" {
			_ = t.StateDB.Enqueue(NormalizePlaylistURL(channel.UploadsPlaylistID), EntityPlaylist, 4)
		}
		for _, video := range videos {
			_ = t.StateDB.Enqueue(NormalizeVideoURL(video.VideoID), EntityVideo, 3)
		}
		t.StateDB.Done(url, code, EntityChannel)
	}
	m.Fetched++
	emit(&ChannelState{URL: url, Status: "done", VideosFound: len(videos)})
	return m, nil
}
