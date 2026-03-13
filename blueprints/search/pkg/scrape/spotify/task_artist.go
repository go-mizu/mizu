package spotify

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type ArtistState struct {
	URL        string
	Status     string
	Error      string
	AlbumsSeen int
	TracksSeen int
}

type ArtistMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type ArtistTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[ArtistState, ArtistMetric] = (*ArtistTask)(nil)

func (t *ArtistTask) Run(ctx context.Context, emit func(*ArtistState)) (ArtistMetric, error) {
	var m ArtistMetric
	ref, err := ParseRef(t.URL, EntityArtist)
	if err != nil {
		return m, err
	}
	if t.StateDB != nil && t.StateDB.IsVisited(ref.URL) {
		_ = t.StateDB.Done(ref.URL, 200, EntityArtist)
		m.Skipped++
		emit(&ArtistState{URL: ref.URL, Status: "skipped"})
		return m, nil
	}

	emit(&ArtistState{URL: ref.URL, Status: "fetching"})
	page, code, err := t.Client.FetchPage(ctx, ref)
	if err != nil {
		m.Failed++
		if t.StateDB != nil {
			t.StateDB.Fail(ref.URL, err.Error())
		}
		emit(&ArtistState{URL: ref.URL, Status: "failed", Error: err.Error()})
		return m, nil
	}
	if code == 404 || page == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(ref.URL, 404, EntityArtist)
		}
		emit(&ArtistState{URL: ref.URL, Status: "not_found"})
		return m, nil
	}

	item := page.Item
	artist := Artist{
		ArtistID:          ref.ID,
		Name:              getString(item, "profile", "name"),
		Biography:         getString(item, "profile", "biography", "text"),
		Followers:         getInt64(item, "stats", "followers"),
		MonthlyListeners:  getInt64(item, "stats", "monthlyListeners"),
		AvatarURL:         firstImageURL(item, "visuals", "avatarImage", "sources"),
		ExternalLinks:     extractExternalLinks(getSlice(item, "profile", "externalLinks", "items")),
		URL:               ref.URL,
		SpotifyURI:        ref.URI,
		SourceTitle:       page.Meta.Title,
		SourceDescription: page.Meta.Description,
		FetchedAt:         time.Now(),
	}
	if artist.Name == "" {
		artist.Name = page.Meta.Title
	}
	if artist.MonthlyListeners == 0 {
		artist.MonthlyListeners = parseDescriptionNumber(page.Meta.Description, monthlyListenersRE)
	}
	if err := t.DB.UpsertArtist(artist); err != nil {
		m.Failed++
		return m, err
	}

	albumCount := 0
	for _, release := range collectArtistReleases(item) {
		if release.AlbumID == "" {
			continue
		}
		_ = t.DB.UpsertAlbum(release)
		if t.StateDB != nil {
			t.StateDB.Enqueue(release.URL, EntityAlbum, 8)
		}
		albumCount++
	}

	trackCount := 0
	for i, rawTop := range getSlice(item, "discography", "topTracks", "items") {
		trackData := getMap(rawTop, "track")
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
		for _, rel := range rels {
			_ = t.DB.UpsertTrackArtist(rel)
		}
		if t.StateDB != nil {
			t.StateDB.Enqueue(track.URL, EntityTrack, 6)
		}
		trackCount = i + 1
	}

	for i, rawRelated := range getSlice(item, "relatedContent", "relatedArtists", "items") {
		relatedID, relatedName, relatedURI := parseArtistRef(rawRelated)
		if relatedID == "" {
			continue
		}
		_ = t.DB.UpsertArtist(Artist{
			ArtistID:   relatedID,
			Name:       relatedName,
			AvatarURL:  firstImageURL(rawRelated, "visuals", "avatarImage", "sources"),
			URL:        NormalizeEntityURL(EntityArtist, relatedID),
			SpotifyURI: relatedURI,
			FetchedAt:  time.Now(),
		})
		_ = t.DB.UpsertArtistRelated(ArtistRelated{
			ArtistID:        artist.ArtistID,
			RelatedArtistID: relatedID,
			RelatedName:     relatedName,
			Ord:             i + 1,
		})
		if t.StateDB != nil {
			t.StateDB.Enqueue(NormalizeEntityURL(EntityArtist, relatedID), EntityArtist, 9)
		}
	}

	if t.StateDB != nil {
		t.StateDB.Done(ref.URL, 200, EntityArtist)
	}
	m.Fetched++
	emit(&ArtistState{URL: ref.URL, Status: "done", AlbumsSeen: albumCount, TracksSeen: trackCount})
	return m, nil
}

func extractExternalLinks(items []any) []ExternalLink {
	out := make([]ExternalLink, 0, len(items))
	for _, item := range items {
		name := getString(item, "name")
		url := getString(item, "url")
		if url == "" {
			continue
		}
		out = append(out, ExternalLink{Name: name, URL: url})
	}
	return out
}

func collectArtistReleases(item map[string]any) []Album {
	var out []Album
	seen := make(map[string]struct{})
	add := func(raw any) {
		id, name, uri, coverURL, releaseDate, albumType := parseAlbumRef(raw)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, Album{
			AlbumID:     id,
			Name:        name,
			AlbumType:   albumType,
			ReleaseDate: releaseDate,
			CoverURL:    coverURL,
			URL:         NormalizeEntityURL(EntityAlbum, id),
			SpotifyURI:  uri,
			FetchedAt:   time.Now(),
		})
	}

	add(getMap(item, "discography", "latest"))
	for _, raw := range getSlice(item, "discography", "popularReleasesAlbums", "items") {
		add(raw)
	}
	for _, group := range getSlice(item, "discography", "albums", "items") {
		for _, rel := range getSlice(group, "releases", "items") {
			add(rel)
		}
	}
	for _, group := range getSlice(item, "discography", "singles", "items") {
		for _, rel := range getSlice(group, "releases", "items") {
			add(rel)
		}
	}
	return out
}
