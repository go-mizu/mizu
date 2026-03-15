package spotify

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

var monthlyListenersRE = regexp.MustCompile(`(?i)([0-9][0-9.,]*\s*[kmb]?)\s+monthly listeners`)

type TrackState struct {
	URL    string
	Status string
	Error  string
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
	ref, err := ParseRef(t.URL, EntityTrack)
	if err != nil {
		return m, err
	}
	if t.StateDB != nil && t.StateDB.IsVisited(ref.URL) {
		_ = t.StateDB.Done(ref.URL, 200, EntityTrack)
		m.Skipped++
		emit(&TrackState{URL: ref.URL, Status: "skipped"})
		return m, nil
	}

	emit(&TrackState{URL: ref.URL, Status: "fetching"})
	page, code, err := t.Client.FetchPage(ctx, ref)
	if err != nil {
		m.Failed++
		emit(&TrackState{URL: ref.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(ref.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || page == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(ref.URL, 404, EntityTrack)
		}
		emit(&TrackState{URL: ref.URL, Status: "not_found"})
		return m, nil
	}

	track, artistRels := parseTrackRef(page.Item)
	if track.TrackID == "" {
		track.TrackID = ref.ID
	}
	if track.Name == "" {
		track.Name = strings.TrimSpace(strings.TrimSuffix(page.Meta.Title, " | Spotify"))
	}
	track.URL = ref.URL
	track.SpotifyURI = ref.URI
	track.SourceTitle = page.Meta.Title
	track.SourceDescription = page.Meta.Description
	track.FetchedAt = time.Now()

	if track.CoverURL == "" {
		track.CoverURL = page.Meta.ImageURL
	}
	if err := t.DB.UpsertTrack(track); err != nil {
		m.Failed++
		return m, err
	}

	if track.AlbumID != "" {
		_ = t.DB.UpsertAlbum(Album{
			AlbumID:           track.AlbumID,
			Name:              track.AlbumName,
			ReleaseDate:       track.ReleaseDate,
			CoverURL:          track.CoverURL,
			URL:               NormalizeEntityURL(EntityAlbum, track.AlbumID),
			SpotifyURI:        SpotifyURI(EntityAlbum, track.AlbumID),
			SourceDescription: fmt.Sprintf("Discovered from track %s", track.Name),
			FetchedAt:         time.Now(),
		})
		if t.StateDB != nil {
			t.StateDB.Enqueue(NormalizeEntityURL(EntityAlbum, track.AlbumID), EntityAlbum, 8)
		}
	}

	for _, rel := range artistRels {
		if rel.TrackID == "" {
			rel.TrackID = track.TrackID
		}
		_ = t.DB.UpsertTrackArtist(rel)
		artist := Artist{
			ArtistID:          rel.ArtistID,
			Name:              rel.ArtistName,
			URL:               NormalizeEntityURL(EntityArtist, rel.ArtistID),
			SpotifyURI:        SpotifyURI(EntityArtist, rel.ArtistID),
			MonthlyListeners:  parseDescriptionNumber(page.Meta.Description, monthlyListenersRE),
			SourceDescription: "Discovered from track artist relation",
			FetchedAt:         time.Now(),
		}
		_ = t.DB.UpsertArtist(artist)
		if t.StateDB != nil {
			t.StateDB.Enqueue(artist.URL, EntityArtist, 10)
		}
	}

	if t.StateDB != nil {
		t.StateDB.Done(ref.URL, 200, EntityTrack)
	}
	m.Fetched++
	emit(&TrackState{URL: ref.URL, Status: "done"})
	return m, nil
}
