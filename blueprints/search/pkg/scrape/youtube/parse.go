package youtube

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func NormalizeVideoURL(input string) string {
	if strings.Contains(input, "://") {
		if u, err := url.Parse(input); err == nil {
			if id := u.Query().Get("v"); id != "" {
				return BaseURL + "/watch?v=" + id
			}
			if strings.Contains(u.Host, "youtu.be") {
				id := strings.Trim(strings.TrimPrefix(u.Path, "/"), " ")
				if id != "" {
					return BaseURL + "/watch?v=" + id
				}
			}
			if parts := strings.Split(strings.Trim(u.Path, "/"), "/"); len(parts) >= 2 && parts[0] == "shorts" {
				return BaseURL + "/watch?v=" + parts[1]
			}
		}
		return input
	}
	return BaseURL + "/watch?v=" + input
}

func NormalizePlaylistURL(input string) string {
	if strings.Contains(input, "://") {
		if u, err := url.Parse(input); err == nil {
			if id := u.Query().Get("list"); id != "" {
				return BaseURL + "/playlist?list=" + id
			}
		}
		return input
	}
	return BaseURL + "/playlist?list=" + input
}

func NormalizeChannelURL(input string) string {
	if strings.Contains(input, "://") {
		u := strings.TrimSuffix(input, "/")
		if strings.Contains(u, "/videos") {
			return u
		}
		return u + "/videos"
	}
	if strings.HasPrefix(input, "@") {
		return BaseURL + "/" + input + "/videos"
	}
	if strings.HasPrefix(input, "UC") {
		return BaseURL + "/channel/" + input + "/videos"
	}
	return BaseURL + "/@" + input + "/videos"
}

func extractVideoID(input string) string {
	u := NormalizeVideoURL(input)
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("v")
}

func extractPlaylistID(input string) string {
	u := NormalizePlaylistURL(input)
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("list")
}

func extractJSONVar(html, marker string) string {
	idx := strings.Index(html, marker)
	if idx < 0 {
		return ""
	}
	start := idx + len(marker)
	for start < len(html) && (html[start] == ' ' || html[start] == '\n') {
		start++
	}
	return extractJSONObject(html, start)
}

func extractJSONCall(html, marker string) string {
	idx := strings.Index(html, marker)
	if idx < 0 {
		return ""
	}
	start := idx + len(marker)
	for start < len(html) && (html[start] == ' ' || html[start] == '\n') {
		start++
	}
	return extractJSONObject(html, start)
}

func extractJSONObject(s string, start int) string {
	if start >= len(s) || s[start] != '{' {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func extractQuotedConfig(html, key string) string {
	patterns := []string{
		`"` + key + `":"`,
		`"` + key + `": "`,
	}
	for _, pattern := range patterns {
		idx := strings.Index(html, pattern)
		if idx < 0 {
			continue
		}
		start := idx + len(pattern)
		end := start
		escaped := false
		for end < len(html) {
			ch := html[end]
			if escaped {
				escaped = false
				end++
				continue
			}
			if ch == '\\' {
				escaped = true
				end++
				continue
			}
			if ch == '"' {
				raw := html[start:end]
				decoded, err := strconv.Unquote(`"` + raw + `"`)
				if err == nil {
					return decoded
				}
				return raw
			}
			end++
		}
	}
	return ""
}

func ParseVideoPage(data *PageData, pageURL string) (*Video, []CaptionTrack, []RelatedVideo, error) {
	videoID := extractVideoID(pageURL)
	if videoID == "" {
		return nil, nil, nil, fmt.Errorf("cannot extract video id")
	}
	v := &Video{
		VideoID:   videoID,
		URL:       NormalizeVideoURL(videoID),
		EmbedURL:  BaseURL + "/embed/" + videoID,
		FetchedAt: time.Now(),
	}
	if pr, ok := data.PlayerResp.(map[string]any); ok {
		if details := mapValue(pr, "videoDetails"); details != nil {
			v.Title = stringValue(details["title"])
			v.Description = stringValue(details["shortDescription"])
			v.ChannelID = stringValue(details["channelId"])
			v.ChannelName = stringValue(details["author"])
			v.DurationSeconds = int(int64Value(details["lengthSeconds"]))
			v.ViewCount = int64Value(details["viewCount"])
			v.IsLive = boolValue(details["isLiveContent"])
			v.IsShort = boolValue(details["isLiveContent"]) == false && v.DurationSeconds > 0 && v.DurationSeconds <= 60
			if arr := stringSlice(details["keywords"]); len(arr) > 0 {
				v.Tags = arr
			}
			if thumbs := mapValue(details, "thumbnail"); thumbs != nil {
				v.ThumbnailURL = bestThumbnail(thumbs["thumbnails"])
			}
		}
		if micro := mapValue(pr, "microformat"); micro != nil {
			if pm := mapValue(micro, "playerMicroformatRenderer"); pm != nil {
				if v.Description == "" {
					v.Description = stringValue(mapValue(pm, "description")["simpleText"])
				}
				v.Category = stringValue(pm["category"])
				v.UploadDate = stringValue(pm["uploadDate"])
				if published := stringValue(pm["publishDate"]); published != "" {
					v.PublishedText = published
					v.PublishedAt = parseDate(published)
				}
				v.ChannelID = firstNonEmpty(v.ChannelID, stringValue(pm["externalChannelId"]))
				v.ChannelName = firstNonEmpty(v.ChannelName, stringValue(pm["ownerChannelName"]))
				v.LikeCount = int64Value(pm["likeCount"])
				if thumb := mapValue(pm, "thumbnail"); thumb != nil && v.ThumbnailURL == "" {
					v.ThumbnailURL = bestThumbnail(thumb["thumbnails"])
				}
			}
		}
	}
	var tracks []CaptionTrack
	if pr, ok := data.PlayerResp.(map[string]any); ok {
		if caps := mapValue(pr, "captions"); caps != nil {
			if renderer := mapValue(caps, "playerCaptionsTracklistRenderer"); renderer != nil {
				for _, item := range arrayValue(renderer["captionTracks"]) {
					m := mapValue(item, "")
					ct := CaptionTrack{
						VideoID:         v.VideoID,
						LanguageCode:    stringValue(m["languageCode"]),
						Name:            extractText(m["name"]),
						BaseURL:         stringValue(m["baseUrl"]),
						Kind:            stringValue(m["kind"]),
						IsAutoGenerated: stringValue(m["kind"]) == "asr",
						FetchedAt:       time.Now(),
					}
					if ct.LanguageCode != "" && ct.BaseURL != "" {
						tracks = append(tracks, ct)
					}
				}
			}
		}
	}

	related := parseRelatedVideos(data.InitialData, v.VideoID)
	if txt := parseCommentCountText(data.InitialData); txt != "" {
		v.CommentCount = parseCountText(txt)
	}
	if pt := parsePublishedText(data.InitialData); pt != "" && v.PublishedText == "" {
		v.PublishedText = pt
	}
	return v, tracks, related, nil
}

func ParseChannelPage(data *PageData, pageURL string) (*Channel, []Video, error) {
	c := &Channel{URL: pageURL, FetchedAt: time.Now()}
	walkJSON(data.InitialData, func(m map[string]any) {
		if r, ok := m["channelMetadataRenderer"].(map[string]any); ok {
			c.ChannelID = firstNonEmpty(c.ChannelID, stringValue(r["externalId"]))
			c.Title = firstNonEmpty(c.Title, stringValue(r["title"]))
			c.Description = firstNonEmpty(c.Description, stringValue(r["description"]))
			c.Handle = firstNonEmpty(c.Handle, strings.TrimPrefix(stringValue(r["vanityChannelUrl"]), BaseURL+"/"))
			if c.URL == "" {
				c.URL = stringValue(r["channelUrl"])
			}
			if thumbs := mapValue(r, "avatar"); thumbs != nil {
				c.AvatarURL = bestThumbnail(thumbs["thumbnails"])
			}
		}
		if r, ok := m["pageHeaderViewModel"].(map[string]any); ok {
			if banner := mapValue(r, "banner"); banner != nil {
				c.BannerURL = bestThumbnail(mapValue(banner, "image")["sources"])
			}
		}
		if r, ok := m["videoCountText"].(map[string]any); ok && c.VideosText == "" {
			c.VideosText = extractText(r)
		}
		if r, ok := m["subscriberCountText"].(map[string]any); ok && c.SubscribersText == "" {
			c.SubscribersText = extractText(r)
		}
	})
	if c.ChannelID != "" && strings.HasPrefix(c.ChannelID, "UC") {
		c.UploadsPlaylistID = "UU" + c.ChannelID[2:]
	}
	videos := parseVideosFromTree(data.InitialData)
	for i := range videos {
		if videos[i].ChannelID == "" {
			videos[i].ChannelID = c.ChannelID
		}
		if videos[i].ChannelName == "" {
			videos[i].ChannelName = c.Title
		}
	}
	if c.ChannelID == "" && c.Title == "" {
		return nil, nil, fmt.Errorf("channel metadata not found")
	}
	return c, dedupeVideos(videos), nil
}

func ParsePlaylistPage(data *PageData, pageURL string) (*Playlist, []PlaylistVideo, []Video, error) {
	playlistID := extractPlaylistID(pageURL)
	if playlistID == "" {
		return nil, nil, nil, fmt.Errorf("cannot extract playlist id")
	}
	p := &Playlist{
		PlaylistID: playlistID,
		URL:        NormalizePlaylistURL(playlistID),
		FetchedAt:  time.Now(),
	}
	walkJSON(data.InitialData, func(m map[string]any) {
		if r, ok := m["playlistHeaderRenderer"].(map[string]any); ok {
			p.Title = firstNonEmpty(p.Title, extractText(r["title"]))
			p.Description = firstNonEmpty(p.Description, extractText(r["descriptionText"]))
			p.ChannelName = firstNonEmpty(p.ChannelName, extractText(r["ownerText"]))
			p.ViewCountText = firstNonEmpty(p.ViewCountText, extractText(r["viewCountText"]))
			p.LastUpdatedText = firstNonEmpty(p.LastUpdatedText, extractText(r["lastUpdatedText"]))
			p.VideoCount = int(parseCountText(extractText(r["numVideosText"])))
		}
		if r, ok := m["playlistSidebarPrimaryInfoRenderer"].(map[string]any); ok {
			p.Title = firstNonEmpty(p.Title, extractText(r["title"]))
		}
		if r, ok := m["playlistSidebarSecondaryInfoRenderer"].(map[string]any); ok {
			p.ChannelName = firstNonEmpty(p.ChannelName, extractText(r["videoOwner"]))
		}
	})
	videos, edges := parsePlaylistVideos(data.InitialData, playlistID)
	if p.Title == "" && len(videos) == 0 {
		return nil, nil, nil, fmt.Errorf("playlist metadata not found")
	}
	return p, edges, dedupeVideos(videos), nil
}

func ParseSearchPage(data *PageData, query string) ([]SearchResult, []Video, []Channel, []Playlist, error) {
	var (
		results   []SearchResult
		videos    []Video
		channels  []Channel
		playlists []Playlist
	)
	walkJSON(data.InitialData, func(m map[string]any) {
		if r, ok := m["videoRenderer"].(map[string]any); ok {
			v := parseVideoRenderer(r)
			if v.VideoID != "" {
				videos = append(videos, v)
				results = append(results, SearchResult{EntityType: EntityVideo, ID: v.VideoID, Title: v.Title, URL: v.URL})
			}
		}
		if r, ok := m["channelRenderer"].(map[string]any); ok {
			c := Channel{
				ChannelID:       stringValue(r["channelId"]),
				Title:           extractText(r["title"]),
				Description:     extractText(r["descriptionSnippet"]),
				SubscribersText: extractText(r["subscriberCountText"]),
				URL:             joinURL(endpointURL(r["navigationEndpoint"])),
				FetchedAt:       time.Now(),
			}
			if c.ChannelID != "" {
				channels = append(channels, c)
				results = append(results, SearchResult{EntityType: EntityChannel, ID: c.ChannelID, Title: c.Title, URL: c.URL})
			}
		}
		if r, ok := m["playlistRenderer"].(map[string]any); ok {
			p := Playlist{
				PlaylistID:  stringValue(r["playlistId"]),
				Title:       extractText(r["title"]),
				ChannelName: extractText(r["longBylineText"]),
				VideoCount:  int(parseCountText(extractText(r["videoCountText"]))),
				URL:         joinURL(endpointURL(r["navigationEndpoint"])),
				FetchedAt:   time.Now(),
			}
			if p.PlaylistID != "" {
				playlists = append(playlists, p)
				results = append(results, SearchResult{EntityType: EntityPlaylist, ID: p.PlaylistID, Title: p.Title, URL: p.URL})
			}
		}
	})
	if len(results) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("no search results found for %q", query)
	}
	return results, dedupeVideos(videos), dedupeChannels(channels), dedupePlaylists(playlists), nil
}

func parseVideoRenderer(r map[string]any) Video {
	v := Video{
		VideoID:       stringValue(r["videoId"]),
		Title:         extractText(r["title"]),
		Description:   extractText(r["descriptionSnippet"]),
		DurationText:  extractText(r["lengthText"]),
		PublishedText: extractText(r["publishedTimeText"]),
		ChannelName:   extractText(r["ownerText"]),
		ThumbnailURL:  bestThumbnail(mapValue(r, "thumbnail")["thumbnails"]),
		FetchedAt:     time.Now(),
	}
	if v.VideoID != "" {
		v.URL = BaseURL + "/watch?v=" + v.VideoID
		v.EmbedURL = BaseURL + "/embed/" + v.VideoID
	}
	v.ViewCount = parseCountText(extractText(r["viewCountText"]))
	v.DurationSeconds = parseDurationSeconds(v.DurationText)
	if nav := mapValue(r, "navigationEndpoint"); nav != nil {
		if watch := mapValue(nav, "watchEndpoint"); watch != nil && v.VideoID == "" {
			v.VideoID = stringValue(watch["videoId"])
		}
	}
	return v
}

func parseVideosFromTree(root any) []Video {
	var out []Video
	walkJSON(root, func(m map[string]any) {
		if r, ok := m["videoRenderer"].(map[string]any); ok {
			v := parseVideoRenderer(r)
			if v.VideoID != "" {
				out = append(out, v)
			}
		}
		if r, ok := m["gridVideoRenderer"].(map[string]any); ok {
			v := parseVideoRenderer(r)
			if v.VideoID != "" {
				out = append(out, v)
			}
		}
	})
	return out
}

func parsePlaylistVideos(root any, playlistID string) ([]Video, []PlaylistVideo) {
	var (
		videos []Video
		edges  []PlaylistVideo
		seen   = map[string]struct{}{}
		pos    = 0
	)
	walkJSON(root, func(m map[string]any) {
		if r, ok := m["playlistVideoRenderer"].(map[string]any); ok {
			videoID := stringValue(r["videoId"])
			if videoID == "" {
				return
			}
			if _, ok := seen[videoID]; ok {
				return
			}
			seen[videoID] = struct{}{}
			pos++
			v := Video{
				VideoID:         videoID,
				Title:           extractText(r["title"]),
				ChannelName:     extractText(r["shortBylineText"]),
				DurationText:    extractText(r["lengthText"]),
				ThumbnailURL:    bestThumbnail(mapValue(r, "thumbnail")["thumbnails"]),
				URL:             BaseURL + "/watch?v=" + videoID,
				EmbedURL:        BaseURL + "/embed/" + videoID,
				FetchedAt:       time.Now(),
				DurationSeconds: parseDurationSeconds(extractText(r["lengthText"])),
			}
			videos = append(videos, v)
			edges = append(edges, PlaylistVideo{PlaylistID: playlistID, VideoID: videoID, Position: pos})
		}
	})
	return videos, edges
}

func parseRelatedVideos(root any, videoID string) []RelatedVideo {
	var out []RelatedVideo
	seen := map[string]struct{}{}
	pos := 0
	for _, v := range dedupeVideos(parseVideosFromTree(root)) {
		if v.VideoID == "" || v.VideoID == videoID {
			continue
		}
		if _, ok := seen[v.VideoID]; ok {
			continue
		}
		seen[v.VideoID] = struct{}{}
		pos++
		out = append(out, RelatedVideo{VideoID: videoID, RelatedVideoID: v.VideoID, Position: pos})
	}
	return out
}

func parseCommentCountText(root any) string {
	var out string
	walkJSON(root, func(m map[string]any) {
		if out != "" {
			return
		}
		if r, ok := m["commentsEntryPointHeaderRenderer"].(map[string]any); ok {
			out = extractText(r["commentCount"])
		}
	})
	return out
}

func parsePublishedText(root any) string {
	var out string
	walkJSON(root, func(m map[string]any) {
		if out != "" {
			return
		}
		if r, ok := m["dateText"].(map[string]any); ok {
			out = extractText(r)
		}
	})
	return out
}

func walkJSON(v any, fn func(map[string]any)) {
	switch x := v.(type) {
	case map[string]any:
		fn(x)
		for _, val := range x {
			walkJSON(val, fn)
		}
	case []any:
		for _, val := range x {
			walkJSON(val, fn)
		}
	}
}

func mapValue(v any, key string) map[string]any {
	if key == "" {
		if m, ok := v.(map[string]any); ok {
			return m
		}
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		if child, ok := m[key].(map[string]any); ok {
			return child
		}
	}
	return nil
}

func arrayValue(v any) []any {
	if arr, ok := v.([]any); ok {
		return arr
	}
	return nil
}

func stringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case fmt.Stringer:
		return x.String()
	default:
		return ""
	}
}

func int64Value(v any) int64 {
	switch x := v.(type) {
	case string:
		n, _ := strconv.ParseInt(strings.ReplaceAll(x, ",", ""), 10, 64)
		return n
	case float64:
		return int64(x)
	case int64:
		return x
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

func bestThumbnail(v any) string {
	arr := arrayValue(v)
	if len(arr) == 0 {
		return ""
	}
	best := ""
	for _, item := range arr {
		if m := mapValue(item, ""); m != nil {
			if url := stringValue(m["url"]); url != "" {
				best = url
			}
		}
	}
	return best
}

func extractText(v any) string {
	if v == nil {
		return ""
	}
	if m, ok := v.(map[string]any); ok {
		if s := stringValue(m["simpleText"]); s != "" {
			return cleanWhitespace(s)
		}
		if runs, ok := m["runs"].([]any); ok {
			var parts []string
			for _, item := range runs {
				if rm, ok := item.(map[string]any); ok {
					if txt := stringValue(rm["text"]); txt != "" {
						parts = append(parts, txt)
					}
				}
			}
			return cleanWhitespace(strings.Join(parts, ""))
		}
		if content := stringValue(m["content"]); content != "" {
			return cleanWhitespace(content)
		}
	}
	if s, ok := v.(string); ok {
		return cleanWhitespace(s)
	}
	return ""
}

func endpointURL(v any) string {
	m := mapValue(v, "")
	if m == nil {
		return ""
	}
	if wm := mapValue(m, "commandMetadata"); wm != nil {
		if web := mapValue(wm, "webCommandMetadata"); web != nil {
			return stringValue(web["url"])
		}
	}
	return ""
}

func joinURL(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return BaseURL + path
}

func parseCountText(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(strings.ReplaceAll(s, ",", "")))
	if s == "" {
		return 0
	}
	for _, suffix := range []string{" views", " view", " subscribers", " subscriber", " videos", " video", " comments", " comment"} {
		s = strings.TrimSuffix(s, suffix)
	}
	mult := float64(1)
	switch {
	case strings.HasSuffix(s, "k"):
		mult = 1_000
		s = strings.TrimSuffix(s, "k")
	case strings.HasSuffix(s, "m"):
		mult = 1_000_000
		s = strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "b"):
		mult = 1_000_000_000
		s = strings.TrimSuffix(s, "b")
	}
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return int64(f * mult)
}

func parseDurationSeconds(s string) int {
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	total := 0
	for _, part := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

func parseDate(s string) time.Time {
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func cleanWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\u00a0", " ")), " ")
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func stringSlice(v any) []string {
	if arr, ok := v.([]any); ok {
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s := stringValue(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func dedupeVideos(items []Video) []Video {
	seen := map[string]Video{}
	order := make([]string, 0, len(items))
	for _, item := range items {
		if item.VideoID == "" {
			continue
		}
		if _, ok := seen[item.VideoID]; !ok {
			order = append(order, item.VideoID)
		}
		prev := seen[item.VideoID]
		seen[item.VideoID] = mergeVideo(prev, item)
	}
	out := make([]Video, 0, len(order))
	for _, id := range order {
		out = append(out, seen[id])
	}
	return out
}

func dedupeChannels(items []Channel) []Channel {
	seen := map[string]Channel{}
	order := []string{}
	for _, item := range items {
		if item.ChannelID == "" {
			continue
		}
		if _, ok := seen[item.ChannelID]; !ok {
			order = append(order, item.ChannelID)
		}
		prev := seen[item.ChannelID]
		if prev.Title == "" {
			seen[item.ChannelID] = item
		}
	}
	out := make([]Channel, 0, len(order))
	for _, id := range order {
		out = append(out, seen[id])
	}
	return out
}

func dedupePlaylists(items []Playlist) []Playlist {
	seen := map[string]Playlist{}
	order := []string{}
	for _, item := range items {
		if item.PlaylistID == "" {
			continue
		}
		if _, ok := seen[item.PlaylistID]; !ok {
			order = append(order, item.PlaylistID)
		}
		prev := seen[item.PlaylistID]
		if prev.Title == "" {
			seen[item.PlaylistID] = item
		}
	}
	out := make([]Playlist, 0, len(order))
	for _, id := range order {
		out = append(out, seen[id])
	}
	return out
}

func mergeVideo(a, b Video) Video {
	if a.VideoID == "" {
		return b
	}
	if a.Title == "" {
		a.Title = b.Title
	}
	if a.Description == "" {
		a.Description = b.Description
	}
	if a.ChannelID == "" {
		a.ChannelID = b.ChannelID
	}
	if a.ChannelName == "" {
		a.ChannelName = b.ChannelName
	}
	if a.DurationSeconds == 0 {
		a.DurationSeconds = b.DurationSeconds
	}
	if a.DurationText == "" {
		a.DurationText = b.DurationText
	}
	if a.ViewCount == 0 {
		a.ViewCount = b.ViewCount
	}
	if a.ThumbnailURL == "" {
		a.ThumbnailURL = b.ThumbnailURL
	}
	if a.URL == "" {
		a.URL = b.URL
	}
	if a.EmbedURL == "" {
		a.EmbedURL = b.EmbedURL
	}
	return a
}
