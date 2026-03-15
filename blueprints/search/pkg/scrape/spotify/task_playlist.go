package spotify

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type PlaylistState struct {
	URL        string
	Status     string
	Error      string
	TracksSeen int
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
	ref, err := ParseRef(t.URL, EntityPlaylist)
	if err != nil {
		return m, err
	}
	if t.StateDB != nil && t.StateDB.IsVisited(ref.URL) {
		_ = t.StateDB.Done(ref.URL, 200, EntityPlaylist)
		m.Skipped++
		emit(&PlaylistState{URL: ref.URL, Status: "skipped"})
		return m, nil
	}

	emit(&PlaylistState{URL: ref.URL, Status: "fetching"})
	page, code, err := t.Client.FetchPage(ctx, ref)
	if err != nil {
		m.Failed++
		if t.StateDB != nil {
			t.StateDB.Fail(ref.URL, err.Error())
		}
		emit(&PlaylistState{URL: ref.URL, Status: "failed", Error: err.Error()})
		return m, nil
	}
	if code == 404 || page == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(ref.URL, 404, EntityPlaylist)
		}
		emit(&PlaylistState{URL: ref.URL, Status: "not_found"})
		return m, nil
	}

	item := page.Item
	playlist := Playlist{
		PlaylistID:        ref.ID,
		Name:              getString(item, "name"),
		Description:       getString(item, "description"),
		Followers:         getInt64(item, "followers"),
		OwnerName:         getString(item, "ownerV2", "data", "name"),
		OwnerUsername:     getString(item, "ownerV2", "data", "username"),
		OwnerURI:          getString(item, "ownerV2", "data", "uri"),
		ImageURL:          firstImageURL(item, "images", "items", "0", "sources"),
		TotalItems:        getInt(item, "content", "totalCount"),
		NextOffset:        getInt(item, "content", "pagingInfo", "nextOffset"),
		URL:               ref.URL,
		SpotifyURI:        ref.URI,
		SourceTitle:       page.Meta.Title,
		SourceDescription: page.Meta.Description,
		FetchedAt:         time.Now(),
	}
	if playlist.ImageURL == "" {
		playlist.ImageURL = page.Meta.ImageURL
	}
	if err := t.DB.UpsertPlaylist(playlist); err != nil {
		m.Failed++
		return m, err
	}

	tracksSeen := 0
	for i, rawItem := range getSlice(item, "content", "items") {
		trackData := getMap(rawItem, "itemV2", "data")
		if len(trackData) == 0 {
			continue
		}
		track, rels := parseTrackRef(trackData)
		if track.TrackID == "" {
			continue
		}
		track.URL = NormalizeEntityURL(EntityTrack, track.TrackID)
		track.SpotifyURI = SpotifyURI(EntityTrack, track.TrackID)
		track.FetchedAt = time.Now()
		_ = t.DB.UpsertTrack(track)
		_ = t.DB.UpsertPlaylistTrack(PlaylistTrack{PlaylistID: playlist.PlaylistID, TrackID: track.TrackID, Ord: i + 1})
		for _, rel := range rels {
			_ = t.DB.UpsertTrackArtist(rel)
			_ = t.DB.UpsertArtist(Artist{
				ArtistID:   rel.ArtistID,
				Name:       rel.ArtistName,
				URL:        NormalizeEntityURL(EntityArtist, rel.ArtistID),
				SpotifyURI: SpotifyURI(EntityArtist, rel.ArtistID),
				FetchedAt:  time.Now(),
			})
			if t.StateDB != nil {
				t.StateDB.Enqueue(NormalizeEntityURL(EntityArtist, rel.ArtistID), EntityArtist, 9)
			}
		}
		if track.AlbumID != "" {
			_ = t.DB.UpsertAlbum(Album{
				AlbumID:     track.AlbumID,
				Name:        track.AlbumName,
				CoverURL:    track.CoverURL,
				ReleaseDate: track.ReleaseDate,
				URL:         NormalizeEntityURL(EntityAlbum, track.AlbumID),
				SpotifyURI:  SpotifyURI(EntityAlbum, track.AlbumID),
				FetchedAt:   time.Now(),
			})
			if t.StateDB != nil {
				t.StateDB.Enqueue(NormalizeEntityURL(EntityAlbum, track.AlbumID), EntityAlbum, 8)
			}
		}
		if t.StateDB != nil {
			t.StateDB.Enqueue(track.URL, EntityTrack, 6)
		}
		tracksSeen = i + 1
	}

	if t.StateDB != nil {
		t.StateDB.Done(ref.URL, 200, EntityPlaylist)
	}
	m.Fetched++
	emit(&PlaylistState{URL: ref.URL, Status: "done", TracksSeen: tracksSeen})
	return m, nil
}
