package youtube

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type VideoState struct {
	URL        string
	Status     string
	Error      string
	Related    int
	Transcript bool
}

type VideoMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type VideoTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[VideoState, VideoMetric] = (*VideoTask)(nil)

func (t *VideoTask) Run(ctx context.Context, emit func(*VideoState)) (VideoMetric, error) {
	var m VideoMetric
	url := NormalizeVideoURL(t.URL)
	emit(&VideoState{URL: url, Status: "fetching"})
	data, code, err := t.Client.FetchPageData(ctx, url)
	if err != nil {
		m.Failed++
		emit(&VideoState{URL: url, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(url, err.Error())
		}
		return m, nil
	}
	if code == 404 || data == nil {
		m.Skipped++
		emit(&VideoState{URL: url, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(url, code, EntityVideo)
		}
		return m, nil
	}
	emit(&VideoState{URL: url, Status: "parsing"})
	video, tracks, related, err := ParseVideoPage(data, url)
	if err != nil {
		m.Failed++
		emit(&VideoState{URL: url, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(url, err.Error())
		}
		return m, nil
	}
	if len(tracks) > 0 {
		if transcript, err := t.Client.FetchTranscript(ctx, tracks[0].BaseURL+"&fmt=srv3"); err == nil && transcript != "" {
			video.Transcript = transcript
			video.TranscriptLanguage = tracks[0].LanguageCode
		}
	}
	if err := t.DB.UpsertVideo(*video); err != nil {
		m.Failed++
		emit(&VideoState{URL: url, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(url, err.Error())
		}
		return m, nil
	}
	if err := t.DB.InsertCaptionTracks(tracks); err != nil {
		return m, fmt.Errorf("insert caption tracks: %w", err)
	}
	if err := t.DB.InsertRelatedVideos(related); err != nil {
		return m, fmt.Errorf("insert related videos: %w", err)
	}
	if t.StateDB != nil {
		t.enqueueDiscovered(video, related)
		t.StateDB.Done(url, code, EntityVideo)
	}
	m.Fetched++
	emit(&VideoState{URL: url, Status: "done", Related: len(related), Transcript: video.Transcript != ""})
	return m, nil
}

func (t *VideoTask) enqueueDiscovered(video *Video, related []RelatedVideo) {
	if video.ChannelID != "" {
		_ = t.StateDB.Enqueue(NormalizeChannelURL(video.ChannelID), EntityChannel, 5)
	}
	for _, item := range related {
		_ = t.StateDB.Enqueue(NormalizeVideoURL(item.RelatedVideoID), EntityVideo, 2)
	}
}
