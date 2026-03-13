package spotify

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type AlbumState struct {
	URL        string
	Status     string
	Error      string
	TracksSeen int
}

type AlbumMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type AlbumTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[AlbumState, AlbumMetric] = (*AlbumTask)(nil)

func (t *AlbumTask) Run(ctx context.Context, emit func(*AlbumState)) (AlbumMetric, error) {
	var m AlbumMetric
	ref, err := ParseRef(t.URL, EntityAlbum)
	if err != nil {
		return m, err
	}
	if t.StateDB != nil && t.StateDB.IsVisited(ref.URL) {
		_ = t.StateDB.Done(ref.URL, 200, EntityAlbum)
		m.Skipped++
		emit(&AlbumState{URL: ref.URL, Status: "skipped"})
		return m, nil
	}

	emit(&AlbumState{URL: ref.URL, Status: "fetching"})
	page, code, err := t.Client.FetchPage(ctx, ref)
	if err != nil {
		m.Failed++
		if t.StateDB != nil {
			t.StateDB.Fail(ref.URL, err.Error())
		}
		emit(&AlbumState{URL: ref.URL, Status: "failed", Error: err.Error()})
		return m, nil
	}
	if code == 404 || page == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(ref.URL, 404, EntityAlbum)
		}
		emit(&AlbumState{URL: ref.URL, Status: "not_found"})
		return m, nil
	}

	item := page.Item
	album := Album{
		AlbumID:           ref.ID,
		Name:              getString(item, "name"),
		AlbumType:         getString(item, "type"),
		ReleaseDate:       releaseDateFrom(item, "date"),
		TotalTracks:       getInt(item, "tracksV2", "totalCount"),
		CoverURL:          firstImageURL(item, "coverArt", "sources"),
		CopyrightText:     getString(item, "copyright", "items", "0", "text"),
		URL:               ref.URL,
		SpotifyURI:        ref.URI,
		SourceTitle:       page.Meta.Title,
		SourceDescription: page.Meta.Description,
		FetchedAt:         time.Now(),
	}
	if album.CoverURL == "" {
		album.CoverURL = page.Meta.ImageURL
	}
	if err := t.DB.UpsertAlbum(album); err != nil {
		m.Failed++
		return m, err
	}

	for i, rawArtist := range getSlice(item, "artists", "items") {
		artistID, artistName, artistURI := parseArtistRef(rawArtist)
		if artistID == "" {
			continue
		}
		_ = t.DB.UpsertArtist(Artist{
			ArtistID:   artistID,
			Name:       artistName,
			URL:        NormalizeEntityURL(EntityArtist, artistID),
			SpotifyURI: artistURI,
			FetchedAt:  time.Now(),
		})
		_ = t.DB.UpsertAlbumArtist(AlbumArtist{
			AlbumID:    album.AlbumID,
			ArtistID:   artistID,
			ArtistName: artistName,
			Ord:        i + 1,
		})
		if t.StateDB != nil {
			t.StateDB.Enqueue(NormalizeEntityURL(EntityArtist, artistID), EntityArtist, 10)
		}
	}

	tracksSeen := 0
	for i, rawTrack := range getSlice(item, "tracksV2", "items") {
		trackData := getMap(rawTrack, "track")
		if len(trackData) == 0 {
			continue
		}
		track, trackArtists := parseTrackRef(trackData)
		if track.TrackID == "" {
			continue
		}
		track.AlbumID = album.AlbumID
		track.AlbumName = album.Name
		track.ReleaseDate = album.ReleaseDate
		track.CoverURL = album.CoverURL
		track.URL = NormalizeEntityURL(EntityTrack, track.TrackID)
		track.SpotifyURI = SpotifyURI(EntityTrack, track.TrackID)
		track.FetchedAt = time.Now()
		_ = t.DB.UpsertTrack(track)
		_ = t.DB.UpsertAlbumTrack(AlbumTrack{AlbumID: album.AlbumID, TrackID: track.TrackID, Ord: i + 1})
		for _, rel := range trackArtists {
			_ = t.DB.UpsertTrackArtist(rel)
			_ = t.DB.UpsertArtist(Artist{
				ArtistID:   rel.ArtistID,
				Name:       rel.ArtistName,
				URL:        NormalizeEntityURL(EntityArtist, rel.ArtistID),
				SpotifyURI: SpotifyURI(EntityArtist, rel.ArtistID),
				FetchedAt:  time.Now(),
			})
			if t.StateDB != nil {
				t.StateDB.Enqueue(NormalizeEntityURL(EntityArtist, rel.ArtistID), EntityArtist, 10)
			}
		}
		if t.StateDB != nil {
			t.StateDB.Enqueue(track.URL, EntityTrack, 6)
		}
		tracksSeen++
	}

	if t.StateDB != nil {
		t.StateDB.Done(ref.URL, 200, EntityAlbum)
	}
	m.Fetched++
	emit(&AlbumState{URL: ref.URL, Status: "done", TracksSeen: tracksSeen})
	return m, nil
}
