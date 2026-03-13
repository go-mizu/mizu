package youtube

import (
	"context"
	"net/url"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type SearchState struct {
	Query        string
	Status       string
	Error        string
	ResultsFound int
}

type SearchMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type SearchTask struct {
	Query      string
	MaxResults int
	Enqueue    bool
	Client     *Client
	DB         *DB
	StateDB    *State
}

var _ core.Task[SearchState, SearchMetric] = (*SearchTask)(nil)

func (t *SearchTask) Run(ctx context.Context, emit func(*SearchState)) (SearchMetric, error) {
	var m SearchMetric
	searchURL := BaseURL + "/results?search_query=" + url.QueryEscape(t.Query)
	emit(&SearchState{Query: t.Query, Status: "fetching"})
	data, code, err := t.Client.FetchPageData(ctx, searchURL)
	if err != nil {
		m.Failed++
		emit(&SearchState{Query: t.Query, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(searchURL, err.Error())
		}
		return m, nil
	}
	results, videos, channels, playlists, err := ParseSearchPage(data, t.Query)
	if err != nil {
		m.Failed++
		emit(&SearchState{Query: t.Query, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(searchURL, err.Error())
		}
		return m, nil
	}
	limit := t.MaxResults
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	videoSet := map[string]struct{}{}
	channelSet := map[string]struct{}{}
	playlistSet := map[string]struct{}{}
	for i, result := range results {
		if i >= limit {
			break
		}
		switch result.EntityType {
		case EntityVideo:
			videoSet[result.ID] = struct{}{}
		case EntityChannel:
			channelSet[result.ID] = struct{}{}
		case EntityPlaylist:
			playlistSet[result.ID] = struct{}{}
		}
	}
	for _, item := range videos {
		if _, ok := videoSet[item.VideoID]; ok {
			_ = t.DB.UpsertVideo(item)
			if t.Enqueue && t.StateDB != nil {
				_ = t.StateDB.Enqueue(NormalizeVideoURL(item.VideoID), EntityVideo, 2)
			}
		}
	}
	for _, item := range channels {
		if _, ok := channelSet[item.ChannelID]; ok {
			_ = t.DB.UpsertChannel(item)
			if t.Enqueue && t.StateDB != nil {
				_ = t.StateDB.Enqueue(NormalizeChannelURL(item.ChannelID), EntityChannel, 2)
			}
		}
	}
	for _, item := range playlists {
		if _, ok := playlistSet[item.PlaylistID]; ok {
			_ = t.DB.UpsertPlaylist(item)
			if t.Enqueue && t.StateDB != nil {
				_ = t.StateDB.Enqueue(NormalizePlaylistURL(item.PlaylistID), EntityPlaylist, 2)
			}
		}
	}
	if t.StateDB != nil {
		t.StateDB.Done(searchURL, code, EntitySearch)
	}
	m.Fetched = limit
	emit(&SearchState{Query: t.Query, Status: "done", ResultsFound: limit})
	return m, nil
}
