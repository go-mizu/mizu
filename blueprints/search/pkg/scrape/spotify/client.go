package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Client struct {
	http       *http.Client
	userAgents []string
	delay      time.Duration
	mu         sync.Mutex
	lastReq    time.Time
}

var initialStateRE = regexp.MustCompile(`(?s)<script id="initialState" type="text/plain">([^<]+)</script>`)

func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     cfg.Workers + 2,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}
	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		userAgents: userAgents,
		delay:      cfg.Delay,
	}
}

func (c *Client) FetchPage(ctx context.Context, ref ParsedRef) (*PageData, int, error) {
	body, code, err := c.Fetch(ctx, ref.URL)
	if err != nil {
		return nil, code, err
	}
	if code == 404 {
		return nil, code, nil
	}
	if code != 200 {
		return nil, code, fmt.Errorf("unexpected HTTP %d for %s", code, ref.URL)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, code, fmt.Errorf("parse HTML: %w", err)
	}

	meta := extractMetadata(doc)
	item, err := extractBootstrapItem(string(body), ref)
	if err != nil {
		item = extractFallbackItem(doc, meta, ref)
		if len(item) == 0 {
			return nil, code, err
		}
	}

	return &PageData{
		Ref:    ref,
		Meta:   meta,
		Item:   item,
		RawURL: ref.URL,
	}, code, nil
}

func (c *Client) Fetch(ctx context.Context, rawURL string) ([]byte, int, error) {
	c.rateLimit()

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	ua := c.userAgents[rand.Intn(len(c.userAgents))]
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", BaseURL+"/")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func (c *Client) rateLimit() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.delay <= 0 {
		return
	}
	since := time.Since(c.lastReq)
	if since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}

func extractMetadata(doc *goquery.Document) PageMetadata {
	meta := PageMetadata{
		Title: strings.TrimSpace(doc.Find("title").First().Text()),
	}
	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		if prop, ok := s.Attr("property"); ok {
			switch prop {
			case "og:title":
				if content := metaAttr(s); content != "" {
					meta.Title = content
				}
			case "og:description":
				if content := metaAttr(s); content != "" {
					meta.Description = content
				}
			case "og:image":
				if content := metaAttr(s); content != "" {
					meta.ImageURL = content
				}
			}
		}
		if name, ok := s.Attr("name"); ok && name == "description" {
			if meta.Description == "" {
				meta.Description = metaAttr(s)
			}
		}
	})
	if href, ok := doc.Find(`link[rel="canonical"]`).Attr("href"); ok {
		meta.Canonical = strings.TrimSpace(href)
	}
	if href, ok := doc.Find(`link[type="application/json+oembed"]`).Attr("href"); ok {
		meta.OEmbedURL = strings.TrimSpace(href)
	}
	return meta
}

func metaAttr(s *goquery.Selection) string {
	v, _ := s.Attr("content")
	return strings.TrimSpace(v)
}

func extractBootstrapItem(html string, ref ParsedRef) (map[string]any, error) {
	m := initialStateRE.FindStringSubmatch(html)
	raw := ""
	if len(m) > 1 {
		raw = strings.TrimSpace(m[1])
	}
	if raw == "" {
		return nil, fmt.Errorf("missing initialState bootstrap payload")
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode initialState: %w", err)
	}

	var state map[string]any
	if err := json.Unmarshal(decoded, &state); err != nil {
		return nil, fmt.Errorf("unmarshal initialState: %w", err)
	}

	items := asMap(getAny(state, "entities", "items"))
	if len(items) == 0 {
		return nil, fmt.Errorf("initialState missing entities.items")
	}

	if item := asMap(items[ref.URI]); len(item) > 0 {
		return item, nil
	}

	if len(items) == 1 {
		for _, v := range items {
			if item := asMap(v); len(item) > 0 {
				return item, nil
			}
		}
	}

	return nil, fmt.Errorf("bootstrap entity %s not found", ref.URI)
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func getAny(v any, path ...string) any {
	cur := v
	for _, p := range path {
		switch x := cur.(type) {
		case map[string]any:
			cur = x[p]
		case []any:
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 || idx >= len(x) {
				return nil
			}
			cur = x[idx]
		default:
			return nil
		}
	}
	return cur
}

func getMap(v any, path ...string) map[string]any {
	return asMap(getAny(v, path...))
}

func getSlice(v any, path ...string) []any {
	return asSlice(getAny(v, path...))
}

func getString(v any, path ...string) string {
	cur := getAny(v, path...)
	switch x := cur.(type) {
	case string:
		return strings.TrimSpace(x)
	case fmt.Stringer:
		return strings.TrimSpace(x.String())
	default:
		return ""
	}
}

func getBool(v any, path ...string) bool {
	cur := getAny(v, path...)
	b, _ := cur.(bool)
	return b
}

func getInt(v any, path ...string) int {
	return int(getInt64(v, path...))
}

func getInt64(v any, path ...string) int64 {
	cur := getAny(v, path...)
	switch x := cur.(type) {
	case float64:
		return int64(x)
	case int:
		return int64(x)
	case int64:
		return x
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	default:
		return 0
	}
}

func parseDescriptionNumber(desc string, re *regexp.Regexp) int64 {
	if desc == "" || re == nil {
		return 0
	}
	m := re.FindStringSubmatch(desc)
	if len(m) < 2 {
		return 0
	}
	return parseCompactNumber(m[1])
}

func parseCompactNumber(raw string) int64 {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return 0
	}
	mult := 1.0
	switch {
	case strings.HasSuffix(raw, "k"):
		mult = 1_000
		raw = strings.TrimSuffix(raw, "k")
	case strings.HasSuffix(raw, "m"):
		mult = 1_000_000
		raw = strings.TrimSuffix(raw, "m")
	case strings.HasSuffix(raw, "b"):
		mult = 1_000_000_000
		raw = strings.TrimSuffix(raw, "b")
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", ""), 64)
	if err != nil {
		return 0
	}
	return int64(f * mult)
}

func firstImageURL(v any, path ...string) string {
	sources := getSlice(v, path...)
	best := ""
	bestW := int64(-1)
	for _, src := range sources {
		u := getString(src, "url")
		if u == "" {
			continue
		}
		w := getInt64(src, "width")
		if best == "" || w > bestW {
			best = u
			bestW = w
		}
	}
	return best
}

func releaseDateFrom(v any, path ...string) string {
	m := getMap(v, path...)
	year := getInt(m, "year")
	month := getInt(m, "month")
	day := getInt(m, "day")
	switch {
	case year == 0:
		return ""
	case month == 0:
		return fmt.Sprintf("%04d", year)
	case day == 0:
		return fmt.Sprintf("%04d-%02d", year, month)
	default:
		return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	}
}

func parseArtistRef(v any) (id, name, uri string) {
	uri = getString(v, "uri")
	if uri != "" {
		_, id = parseSpotifyURI(uri)
	}
	if id == "" {
		id = getString(v, "id")
	}
	name = getString(v, "profile", "name")
	if name == "" {
		name = getString(v, "name")
	}
	return id, name, uri
}

func parseAlbumRef(v any) (id, name, uri, coverURL, releaseDate, albumType string) {
	uri = getString(v, "uri")
	if uri != "" {
		_, id = parseSpotifyURI(uri)
	}
	if id == "" {
		id = getString(v, "id")
	}
	name = getString(v, "name")
	coverURL = firstImageURL(v, "coverArt", "sources")
	releaseDate = releaseDateFrom(v, "date")
	albumType = getString(v, "type")
	return id, name, uri, coverURL, releaseDate, albumType
}

func parseTrackRef(v any) (Track, []TrackArtist) {
	track := Track{
		TrackID:     getString(v, "id"),
		Name:        getString(v, "name"),
		DurationMS:  getInt64(v, "duration", "totalMilliseconds"),
		DiscNumber:  getInt(v, "discNumber"),
		TrackNumber: getInt(v, "trackNumber"),
		Playable:    getBool(v, "playability", "playable"),
		PreviewURL:  getString(v, "previews", "audioPreviews", "items", "0", "url"),
		Playcount:   getInt64(v, "playcount"),
		SpotifyURI:  getString(v, "uri"),
	}
	if track.PreviewURL == "" {
		items := getSlice(v, "previews", "audioPreviews", "items")
		if len(items) > 0 {
			track.PreviewURL = getString(items[0], "url")
		}
	}

	album := getMap(v, "albumOfTrack")
	if len(album) > 0 {
		track.AlbumID, track.AlbumName, _, track.CoverURL, track.ReleaseDate, _ = parseAlbumRef(album)
	}
	if track.TrackID == "" && track.SpotifyURI != "" {
		_, track.TrackID = parseSpotifyURI(track.SpotifyURI)
	}

	artistItems := getSlice(v, "artists", "items")
	if len(artistItems) == 0 {
		artistItems = getSlice(v, "firstArtist", "items")
		artistItems = append(artistItems, getSlice(v, "otherArtists", "items")...)
	}
	var rels []TrackArtist
	for i, item := range artistItems {
		artistID, artistName, _ := parseArtistRef(item)
		if artistID == "" {
			continue
		}
		rels = append(rels, TrackArtist{
			TrackID:    track.TrackID,
			ArtistID:   artistID,
			ArtistName: artistName,
			Ord:        i + 1,
		})
	}
	return track, rels
}

func extractFallbackItem(doc *goquery.Document, meta PageMetadata, ref ParsedRef) map[string]any {
	switch ref.EntityType {
	case EntityTrack:
		return fallbackTrackItem(doc, meta, ref)
	case EntityAlbum:
		return fallbackAlbumItem(doc, meta, ref)
	case EntityArtist:
		return fallbackArtistItem(doc, meta, ref)
	case EntityPlaylist:
		return fallbackPlaylistItem(doc, meta, ref)
	default:
		return nil
	}
}

func fallbackTrackItem(doc *goquery.Document, meta PageMetadata, ref ParsedRef) map[string]any {
	item := map[string]any{
		"id":   ref.ID,
		"name": bestEntityTitle(doc, meta),
		"uri":  ref.URI,
		"playability": map[string]any{
			"playable": true,
		},
	}
	if secs := metaAttrByProp(doc, "music:duration"); secs != "" {
		if n, err := strconv.ParseInt(secs, 10, 64); err == nil {
			item["duration"] = map[string]any{"totalMilliseconds": n * 1000}
		}
	}
	if preview := metaAttrByProp(doc, "og:audio"); preview != "" {
		item["previews"] = map[string]any{"audioPreviews": map[string]any{"items": []any{map[string]any{"url": preview}}}}
	}
	if albumURL := metaAttrByProp(doc, "music:album"); albumURL != "" {
		if _, albumID := parseSpotifyURL(albumURL); albumID != "" {
			item["albumOfTrack"] = map[string]any{
				"id":       albumID,
				"uri":      SpotifyURI(EntityAlbum, albumID),
				"coverArt": map[string]any{"sources": []any{map[string]any{"url": meta.ImageURL, "width": 640}}},
				"date":     releaseDateMap(metaAttrByProp(doc, "music:release_date")),
			}
		}
	}
	artists := metaArtists(doc)
	if len(artists) > 0 {
		item["artists"] = map[string]any{"items": artists}
	}
	return item
}

func fallbackAlbumItem(doc *goquery.Document, meta PageMetadata, ref ParsedRef) map[string]any {
	artists := metaArtists(doc)
	tracks := entityLinkItems(doc, EntityTrack, func(id string, label string) any {
		return map[string]any{
			"track": map[string]any{
				"id":   id,
				"name": label,
				"uri":  SpotifyURI(EntityTrack, id),
				"albumOfTrack": map[string]any{
					"id":   ref.ID,
					"name": meta.Title,
					"uri":  ref.URI,
				},
				"artists": map[string]any{"items": artists},
			},
		}
	})
	return map[string]any{
		"id":   ref.ID,
		"name": bestEntityTitle(doc, meta),
		"type": "ALBUM",
		"uri":  ref.URI,
		"coverArt": map[string]any{
			"sources": []any{map[string]any{"url": meta.ImageURL, "width": 640}},
		},
		"artists": map[string]any{"items": artists},
		"tracksV2": map[string]any{
			"items":      tracks,
			"totalCount": len(tracks),
		},
	}
}

func fallbackArtistItem(doc *goquery.Document, meta PageMetadata, ref ParsedRef) map[string]any {
	name := bestEntityTitle(doc, meta)
	albumItems := entityLinkItems(doc, EntityAlbum, func(id string, label string) any {
		return map[string]any{
			"id":       id,
			"name":     label,
			"uri":      SpotifyURI(EntityAlbum, id),
			"coverArt": map[string]any{"sources": []any{}},
		}
	})
	trackItems := entityLinkItems(doc, EntityTrack, func(id string, label string) any {
		return map[string]any{
			"track": map[string]any{
				"id":   id,
				"name": label,
				"uri":  SpotifyURI(EntityTrack, id),
				"artists": map[string]any{
					"items": []any{map[string]any{"id": ref.ID, "uri": ref.URI, "profile": map[string]any{"name": name}}},
				},
			},
		}
	})
	related := entityLinkItemsExcluding(doc, EntityArtist, ref.ID, func(id string, label string) any {
		return map[string]any{
			"id":      id,
			"uri":     SpotifyURI(EntityArtist, id),
			"profile": map[string]any{"name": label},
		}
	})
	return map[string]any{
		"id": ref.ID,
		"profile": map[string]any{
			"name":      name,
			"biography": map[string]any{"text": ""},
		},
		"stats": map[string]any{
			"monthlyListeners": parseDescriptionNumber(meta.Description, monthlyListenersRE),
		},
		"uri": ref.URI,
		"visuals": map[string]any{
			"avatarImage": map[string]any{
				"sources": []any{map[string]any{"url": meta.ImageURL, "width": 640}},
			},
		},
		"discography": map[string]any{
			"albums":                map[string]any{"items": wrapReleases(albumItems)},
			"singles":               map[string]any{"items": []any{}},
			"popularReleasesAlbums": map[string]any{"items": albumItems},
			"topTracks":             map[string]any{"items": trackItems},
		},
		"relatedContent": map[string]any{
			"relatedArtists": map[string]any{"items": related},
		},
	}
}

func fallbackPlaylistItem(doc *goquery.Document, meta PageMetadata, ref ParsedRef) map[string]any {
	tracks := entityLinkItems(doc, EntityTrack, func(id string, label string) any {
		return map[string]any{
			"itemV2": map[string]any{
				"data": map[string]any{
					"id":   id,
					"name": label,
					"uri":  SpotifyURI(EntityTrack, id),
				},
			},
		}
	})
	return map[string]any{
		"id":          ref.ID,
		"name":        bestEntityTitle(doc, meta),
		"description": meta.Description,
		"followers":   int64(0),
		"uri":         ref.URI,
		"images": map[string]any{
			"items": []any{map[string]any{"sources": []any{map[string]any{"url": meta.ImageURL, "width": 640}}}},
		},
		"content": map[string]any{
			"items":      tracks,
			"totalCount": len(tracks),
			"pagingInfo": map[string]any{},
		},
	}
}

func metaAttrByProp(doc *goquery.Document, prop string) string {
	if sel := doc.Find(`meta[property="` + prop + `"]`).First(); sel.Length() > 0 {
		return metaAttr(sel)
	}
	return ""
}

func metaArtists(doc *goquery.Document) []any {
	var items []any
	seen := make(map[string]struct{})
	doc.Find(`meta[property="music:musician"]`).Each(func(_ int, s *goquery.Selection) {
		u, _ := s.Attr("content")
		_, id := parseSpotifyURL(u)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		items = append(items, map[string]any{
			"id":      id,
			"uri":     SpotifyURI(EntityArtist, id),
			"profile": map[string]any{"name": ""},
		})
	})
	if len(items) > 0 {
		return items
	}
	return entityLinkItems(doc, EntityArtist, func(id string, label string) any {
		return map[string]any{
			"id":      id,
			"uri":     SpotifyURI(EntityArtist, id),
			"profile": map[string]any{"name": label},
		}
	})
}

func entityLinkItems(doc *goquery.Document, entity string, fn func(id string, label string) any) []any {
	return entityLinkItemsExcluding(doc, entity, "", fn)
}

func entityLinkItemsExcluding(doc *goquery.Document, entity, excludeID string, fn func(id string, label string) any) []any {
	var out []any
	seen := make(map[string]struct{})
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		ref, err := ParseRef(resolveSpotifyHref(href), "")
		if err != nil || ref.EntityType != entity || ref.ID == excludeID {
			return
		}
		if _, ok := seen[ref.ID]; ok {
			return
		}
		seen[ref.ID] = struct{}{}
		out = append(out, fn(ref.ID, strings.TrimSpace(s.Text())))
	})
	return out
}

func wrapReleases(items []any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{"releases": map[string]any{"items": []any{item}}})
	}
	return out
}

func resolveSpotifyHref(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return BaseURL + href
	}
	return href
}

func releaseDateMap(raw string) map[string]any {
	out := map[string]any{}
	parts := strings.Split(raw, "-")
	if len(parts) >= 1 {
		if year, err := strconv.Atoi(parts[0]); err == nil {
			out["year"] = year
		}
	}
	if len(parts) >= 2 {
		if month, err := strconv.Atoi(parts[1]); err == nil {
			out["month"] = month
		}
	}
	if len(parts) >= 3 {
		if day, err := strconv.Atoi(parts[2]); err == nil {
			out["day"] = day
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func bestEntityTitle(doc *goquery.Document, meta PageMetadata) string {
	h1 := strings.TrimSpace(doc.Find("h1").First().Text())
	if h1 != "" && !isGenericSpotifyTitle(h1) {
		return h1
	}
	title := normalizeSpotifyTitle(meta.Title)
	if title != "" {
		return title
	}
	return h1
}

func normalizeSpotifyTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " | Spotify")
	if strings.Contains(s, " - song and lyrics by ") {
		s = strings.SplitN(s, " - song and lyrics by ", 2)[0]
	}
	if isGenericSpotifyTitle(s) {
		return ""
	}
	return s
}

func isGenericSpotifyTitle(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "" || s == "spotify" || s == "spotify – web player" || s == "spotify - web player"
}
