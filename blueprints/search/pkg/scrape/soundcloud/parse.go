package soundcloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var hydrationRe = regexp.MustCompile(`window\.__sc_hydration\s*=\s*(\[[\s\S]*?\]);`)

type hydrationEntry struct {
	Hydratable string          `json:"hydratable"`
	Data       json.RawMessage `json:"data"`
}

type apiUser struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	FullName           string `json:"full_name"`
	Description        string `json:"description"`
	AvatarURL          string `json:"avatar_url"`
	City               string `json:"city"`
	CountryCode        string `json:"country_code"`
	FollowersCount     int    `json:"followers_count"`
	FollowingsCount    int    `json:"followings_count"`
	TrackCount         int    `json:"track_count"`
	PlaylistCount      int    `json:"playlist_count"`
	LikesCount         int    `json:"likes_count"`
	PlaylistLikesCount int    `json:"playlist_likes_count"`
	Permalink          string `json:"permalink"`
	PermalinkURL       string `json:"permalink_url"`
	Verified           bool   `json:"verified"`
	CreatedAt          string `json:"created_at"`
	Badges             struct {
		Verified bool `json:"verified"`
	} `json:"badges"`
}

type apiTrack struct {
	ID            int64   `json:"id"`
	UserID        int64   `json:"user_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Genre         string  `json:"genre"`
	TagList       string  `json:"tag_list"`
	ArtworkURL    string  `json:"artwork_url"`
	WaveformURL   string  `json:"waveform_url"`
	LabelName     string  `json:"label_name"`
	License       string  `json:"license"`
	Duration      int64   `json:"duration"`
	PlaybackCount int64   `json:"playback_count"`
	LikesCount    int     `json:"likes_count"`
	CommentCount  int     `json:"comment_count"`
	DownloadCount int     `json:"download_count"`
	RepostsCount  int     `json:"reposts_count"`
	Downloadable  bool    `json:"downloadable"`
	Streamable    bool    `json:"streamable"`
	ReleaseDate   string  `json:"release_date"`
	CreatedAt     string  `json:"created_at"`
	PermalinkURL  string  `json:"permalink_url"`
	User          apiUser `json:"user"`
}

type apiPlaylist struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	ArtworkURL   string     `json:"artwork_url"`
	TrackCount   int        `json:"track_count"`
	Duration     int64      `json:"duration"`
	LikesCount   int        `json:"likes_count"`
	RepostsCount int        `json:"reposts_count"`
	SetType      string     `json:"set_type"`
	IsAlbum      bool       `json:"is_album"`
	CreatedAt    string     `json:"created_at"`
	PublishedAt  string     `json:"published_at"`
	PermalinkURL string     `json:"permalink_url"`
	User         apiUser    `json:"user"`
	Tracks       []apiTrack `json:"tracks"`
}

func ParseTrackPage(doc *goquery.Document, body []byte, pageURL string) (*Track, *User, []Comment, error) {
	entries, _ := parseHydration(body)
	var rawTrack apiTrack
	var rawUser apiUser
	for _, e := range entries {
		switch e.Hydratable {
		case "sound":
			_ = json.Unmarshal(e.Data, &rawTrack)
		case "user":
			_ = json.Unmarshal(e.Data, &rawUser)
		}
	}
	if rawTrack.ID == 0 {
		return parseTrackFallback(doc, pageURL)
	}
	user := rawTrack.User
	if user.ID == 0 {
		user = rawUser
	}
	track := toTrack(rawTrack, pageURL)
	return &track, userPtr(user), parseComments(doc, rawTrack.ID), nil
}

func ParsePlaylistPage(doc *goquery.Document, body []byte, pageURL string) (*Playlist, *User, []PlaylistTrack, error) {
	entries, _ := parseHydration(body)
	var raw apiPlaylist
	for _, e := range entries {
		if e.Hydratable == "playlist" {
			_ = json.Unmarshal(e.Data, &raw)
			break
		}
	}
	if raw.ID == 0 {
		return parsePlaylistFallback(doc, pageURL)
	}

	playlist := toPlaylist(raw, pageURL)
	trackURLs := parsePlaylistTrackURLs(doc)
	var rels []PlaylistTrack
	seen := map[int64]struct{}{}
	for i, t := range raw.Tracks {
		if t.ID == 0 {
			continue
		}
		rel := PlaylistTrack{
			PlaylistID: raw.ID,
			TrackID:    t.ID,
			Position:   i + 1,
			TrackURL:   normalizeSCURL(t.PermalinkURL),
		}
		if rel.TrackURL == "" && i < len(trackURLs) {
			rel.TrackURL = trackURLs[i]
		}
		rels = append(rels, rel)
		seen[t.ID] = struct{}{}
	}

	for i, trackURL := range trackURLs {
		trackID := extractNumericSuffix(trackURL)
		if _, ok := seen[trackID]; ok {
			continue
		}
		rels = append(rels, PlaylistTrack{
			PlaylistID: raw.ID,
			TrackID:    trackID,
			Position:   len(rels) + 1 + i,
			TrackURL:   trackURL,
		})
	}

	return &playlist, userPtr(raw.User), rels, nil
}

func ParseUserPage(doc *goquery.Document, body []byte, pageURL string) (*User, error) {
	entries, _ := parseHydration(body)
	for _, e := range entries {
		if e.Hydratable != "user" {
			continue
		}
		var raw apiUser
		if err := json.Unmarshal(e.Data, &raw); err == nil && raw.ID != 0 {
			u := toUser(raw, pageURL)
			return &u, nil
		}
	}

	userID := extractNumericMeta(doc, `meta[property="al:ios:url"]`, "soundcloud://users:")
	u := &User{
		UserID:      userID,
		Username:    strings.TrimPrefix(strings.Trim(parseCanonical(doc), "/"), "/"),
		FullName:    textOrMeta(doc, `meta[property="og:title"]`, "content"),
		Description: textOrMeta(doc, `meta[name="description"]`, "content"),
		AvatarURL:   textOrMeta(doc, `meta[property="og:image"]`, "content"),
		URL:         pageURL,
		FetchedAt:   time.Now(),
	}
	if u.Username == "" {
		u.Username = pathSegments(pageURL)[0]
	}
	return u, nil
}

func parseSearchResult(query string, raw json.RawMessage) (SearchResult, bool) {
	var probe struct {
		Kind         string `json:"kind"`
		Title        string `json:"title"`
		Username     string `json:"username"`
		PermalinkURL string `json:"permalink_url"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return SearchResult{}, false
	}
	title := probe.Title
	if title == "" {
		title = probe.Username
	}
	entityType := ""
	switch probe.Kind {
	case "track":
		entityType = EntityTrack
	case "playlist":
		entityType = EntityPlaylist
	case "user":
		entityType = EntityUser
	default:
		return SearchResult{}, false
	}
	return SearchResult{
		SearchID:  query + "|" + entityType + "|" + probe.PermalinkURL,
		Query:     query,
		Kind:      entityType,
		Title:     title,
		URL:       normalizeSCURL(probe.PermalinkURL),
		FetchedAt: time.Now(),
	}, true
}

func DiscoverQueueItems(doc *goquery.Document) []QueueItem {
	var items []QueueItem
	seen := map[string]struct{}{}
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		href, ok := sel.Attr("href")
		if !ok {
			return
		}
		u := normalizeSCURL(href)
		entityType := InferEntityType(u)
		if entityType == "" || u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		items = append(items, QueueItem{URL: u, EntityType: entityType, Priority: 1})
	})
	return items
}

func InferEntityType(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if u.Host != "" && !strings.Contains(u.Host, "soundcloud.com") {
		return ""
	}
	parts := pathSegments(rawURL)
	if len(parts) == 0 {
		return ""
	}
	if parts[0] == "search" || parts[0] == "popular" || parts[0] == "discover" || parts[0] == "charts" || parts[0] == "stream" {
		return ""
	}
	if len(parts) == 1 {
		return EntityUser
	}
	switch parts[1] {
	case "sets":
		if len(parts) >= 3 {
			return EntityPlaylist
		}
		return ""
	case "likes", "reposts", "comments", "albums", "tracks", "spotlight", "following", "followers":
		return ""
	default:
		if len(parts) != 2 {
			return ""
		}
		return EntityTrack
	}
}

func normalizeSCURL(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return BaseURL + raw
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.Count(raw, "/") >= 1 {
		return BaseURL + "/" + strings.TrimLeft(raw, "/")
	}
	return BaseURL + "/" + strings.TrimLeft(raw, "/")
}

func parseHydration(body []byte) ([]hydrationEntry, error) {
	m := hydrationRe.FindSubmatch(body)
	if len(m) != 2 {
		return nil, fmt.Errorf("soundcloud hydration not found")
	}
	var entries []hydrationEntry
	if err := json.Unmarshal(m[1], &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func parseTrackFallback(doc *goquery.Document, pageURL string) (*Track, *User, []Comment, error) {
	trackID := extractTrackIDFromDoc(doc)
	userURL := textOrMeta(doc, `meta[property="soundcloud:user"]`, "content")
	u := &User{
		URL:       normalizeSCURL(userURL),
		Username:  firstPathSegment(userURL),
		FetchedAt: time.Now(),
	}
	t := &Track{
		TrackID:       trackID,
		Title:         textOrMeta(doc, `meta[property="twitter:title"]`, "content"),
		Description:   htmlUnescape(textOrMeta(doc, `meta[property="og:description"]`, "content")),
		ArtworkURL:    textOrMeta(doc, `meta[property="og:image"]`, "content"),
		DurationMS:    parseDurationISO(doc.Find(`meta[itemprop="duration"]`).AttrOr("content", "")),
		Genre:         doc.Find(`meta[itemprop="genre"]`).AttrOr("content", ""),
		PlaybackCount: int64(extractMetaInt(doc, `meta[property="soundcloud:play_count"]`)),
		CommentCount:  extractMetaInt(doc, `meta[property="soundcloud:comments_count"]`),
		LikesCount:    extractMetaInt(doc, `meta[property="soundcloud:like_count"]`),
		DownloadCount: extractMetaInt(doc, `meta[property="soundcloud:download_count"]`),
		URL:           pageURL,
		FetchedAt:     time.Now(),
	}
	return t, u, parseComments(doc, trackID), nil
}

func parsePlaylistFallback(doc *goquery.Document, pageURL string) (*Playlist, *User, []PlaylistTrack, error) {
	u := &User{
		URL:       normalizeSCURL(doc.Find(`link[rel="author"]`).AttrOr("href", "")),
		Username:  firstPathSegment(doc.Find(`link[rel="author"]`).AttrOr("href", "")),
		FetchedAt: time.Now(),
	}
	p := &Playlist{
		Title:       textOrMeta(doc, `meta[property="twitter:title"]`, "content"),
		Description: htmlUnescape(textOrMeta(doc, `meta[property="og:description"]`, "content")),
		ArtworkURL:  textOrMeta(doc, `meta[property="og:image"]`, "content"),
		TrackCount:  extractMetaInt(doc, `meta[itemprop="numTracks"]`),
		URL:         pageURL,
		FetchedAt:   time.Now(),
	}
	urls := parsePlaylistTrackURLs(doc)
	rels := make([]PlaylistTrack, 0, len(urls))
	for i, trackURL := range urls {
		rels = append(rels, PlaylistTrack{
			TrackURL: trackURL,
			Position: i + 1,
		})
	}
	return p, u, rels, nil
}

func parsePlaylistTrackURLs(doc *goquery.Document) []string {
	var out []string
	seen := map[string]struct{}{}
	doc.Find(`section.tracklist a[itemprop="url"], section.tracklist a[href]`).Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		u := normalizeSCURL(href)
		if InferEntityType(u) != EntityTrack {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	})
	return out
}

func parseComments(doc *goquery.Document, trackID int64) []Comment {
	var out []Comment
	doc.Find("section.comments h2").Each(func(i int, h2 *goquery.Selection) {
		p := h2.NextFiltered("p")
		ts := p.NextFiltered("time")
		if p.Length() == 0 {
			return
		}
		a := h2.Find("a").First()
		out = append(out, Comment{
			CommentID: fmt.Sprintf("%d-%d", trackID, i+1),
			TrackID:   trackID,
			UserName:  strings.TrimSpace(a.Text()),
			UserURL:   normalizeSCURL(a.AttrOr("href", "")),
			Body:      strings.TrimSpace(p.Text()),
			PostedAt:  parseTime(ts.Text()),
			FetchedAt: time.Now(),
		})
	})
	return out
}

func toUser(raw apiUser, pageURL string) User {
	return User{
		UserID:             raw.ID,
		Username:           raw.Username,
		FullName:           raw.FullName,
		Description:        raw.Description,
		AvatarURL:          raw.AvatarURL,
		City:               raw.City,
		CountryCode:        raw.CountryCode,
		FollowersCount:     raw.FollowersCount,
		FollowingsCount:    raw.FollowingsCount,
		TrackCount:         raw.TrackCount,
		PlaylistCount:      raw.PlaylistCount,
		LikesCount:         raw.LikesCount,
		PlaylistLikesCount: raw.PlaylistLikesCount,
		Verified:           raw.Verified || raw.Badges.Verified,
		URL:                firstNonEmpty(normalizeSCURL(raw.PermalinkURL), pageURL),
		CreatedAt:          parseTime(raw.CreatedAt),
		FetchedAt:          time.Now(),
	}
}

func toTrack(raw apiTrack, pageURL string) Track {
	return Track{
		TrackID:       raw.ID,
		UserID:        firstNonZero(raw.UserID, raw.User.ID),
		Title:         raw.Title,
		Description:   raw.Description,
		Genre:         raw.Genre,
		TagList:       raw.TagList,
		ArtworkURL:    raw.ArtworkURL,
		WaveformURL:   raw.WaveformURL,
		LabelName:     raw.LabelName,
		License:       raw.License,
		DurationMS:    raw.Duration,
		PlaybackCount: raw.PlaybackCount,
		LikesCount:    raw.LikesCount,
		CommentCount:  raw.CommentCount,
		DownloadCount: raw.DownloadCount,
		RepostsCount:  raw.RepostsCount,
		Downloadable:  raw.Downloadable,
		Streamable:    raw.Streamable,
		ReleaseDate:   parseTime(raw.ReleaseDate),
		CreatedAt:     parseTime(raw.CreatedAt),
		URL:           firstNonEmpty(normalizeSCURL(raw.PermalinkURL), pageURL),
		FetchedAt:     time.Now(),
	}
}

func toPlaylist(raw apiPlaylist, pageURL string) Playlist {
	return Playlist{
		PlaylistID:   raw.ID,
		UserID:       firstNonZero(raw.UserID, raw.User.ID),
		Title:        raw.Title,
		Description:  raw.Description,
		ArtworkURL:   raw.ArtworkURL,
		TrackCount:   raw.TrackCount,
		DurationMS:   raw.Duration,
		LikesCount:   raw.LikesCount,
		RepostsCount: raw.RepostsCount,
		SetType:      raw.SetType,
		IsAlbum:      raw.IsAlbum,
		CreatedAt:    parseTime(raw.CreatedAt),
		PublishedAt:  parseTime(raw.PublishedAt),
		URL:          firstNonEmpty(normalizeSCURL(raw.PermalinkURL), pageURL),
		FetchedAt:    time.Now(),
	}
}

func parseCanonical(doc *goquery.Document) string {
	return doc.Find(`link[rel="canonical"]`).AttrOr("href", "")
}

func textOrMeta(doc *goquery.Document, selector, attr string) string {
	return strings.TrimSpace(doc.Find(selector).First().AttrOr(attr, ""))
}

func extractTrackIDFromDoc(doc *goquery.Document) int64 {
	for _, prefix := range []string{"soundcloud://sounds:", "soundcloud%3Atracks%253A", "soundcloud:tracks:"} {
		if id := extractNumericMeta(doc, `meta[property="al:ios:url"]`, prefix); id != 0 {
			return id
		}
		if id := extractNumericMeta(doc, `meta[property="twitter:player"]`, prefix); id != 0 {
			return id
		}
	}
	return 0
}

func extractNumericMeta(doc *goquery.Document, selector, prefix string) int64 {
	v := doc.Find(selector).AttrOr("content", "")
	idx := strings.Index(v, prefix)
	if idx < 0 {
		return 0
	}
	rest := v[idx+len(prefix):]
	n := takeDigits(rest)
	id, _ := strconv.ParseInt(n, 10, 64)
	return id
}

func extractMetaInt(doc *goquery.Document, selector string) int {
	v := doc.Find(selector).AttrOr("content", "")
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}

func parseDurationISO(raw string) int64 {
	if raw == "" {
		return 0
	}
	var h, m, s int64
	re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)
	match := re.FindStringSubmatch(raw)
	if len(match) != 4 {
		return 0
	}
	if match[1] != "" {
		h, _ = strconv.ParseInt(match[1], 10, 64)
	}
	if match[2] != "" {
		m, _ = strconv.ParseInt(match[2], 10, 64)
	}
	if match[3] != "" {
		s, _ = strconv.ParseInt(match[3], 10, 64)
	}
	return ((h*60+m)*60 + s) * 1000
}

func parseTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", time.RFC3339Nano} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func pathSegments(rawURL string) []string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	return splitPath(u.Path)
}

func splitPath(path string) []string {
	var out []string
	for _, p := range strings.Split(strings.Trim(path, "/"), "/") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstPathSegment(rawURL string) string {
	parts := pathSegments(rawURL)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func firstNonZero(vs ...int64) int64 {
	for _, v := range vs {
		if v != 0 {
			return v
		}
	}
	return 0
}

func userPtr(raw apiUser) *User {
	if raw.ID == 0 && raw.Username == "" {
		return nil
	}
	u := toUser(raw, normalizeSCURL(raw.PermalinkURL))
	return &u
}

func takeDigits(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func extractNumericSuffix(rawURL string) int64 {
	parts := pathSegments(rawURL)
	if len(parts) == 0 {
		return 0
	}
	last := parts[len(parts)-1]
	var digits strings.Builder
	for _, r := range last {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	if digits.Len() == 0 {
		return 0
	}
	id, _ := strconv.ParseInt(digits.String(), 10, 64)
	return id
}

func htmlUnescape(s string) string {
	r := strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&#x27;", "'", "&quot;", `"`)
	return r.Replace(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
