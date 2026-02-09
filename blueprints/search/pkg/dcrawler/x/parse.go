package x

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// JSON parsing helpers for navigating map[string]any from GraphQL responses.
// Ported from Nitter's src/parser.nim and src/parserutils.nim.

// dig navigates nested maps by key path, returning nil if any key is missing.
func dig(m map[string]any, keys ...string) any {
	var cur any = m
	for _, k := range keys {
		switch v := cur.(type) {
		case map[string]any:
			cur = v[k]
		default:
			return nil
		}
	}
	return cur
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func asSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func asStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asInt(v any) int {
	switch v := v.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

func asBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// parseTwitterTime parses Twitter's time format: "Mon Jan 02 15:04:05 +0000 2006"
func parseTwitterTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("Mon Jan 02 15:04:05 +0000 2006", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// parseCreatedAt extracts created_at from legacy, trying both string and ms formats.
// ConversationTimeline uses created_at_ms (epoch ms), other endpoints use created_at (string).
func parseCreatedAt(legacy map[string]any) time.Time {
	if s := asStr(legacy["created_at"]); s != "" {
		return parseTwitterTime(s)
	}
	// Fallback: created_at_ms (millisecond epoch)
	switch v := legacy["created_at_ms"].(type) {
	case float64:
		return time.UnixMilli(int64(v))
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.UnixMilli(n)
		}
	}
	return time.Time{}
}

// ── User parsing ────────────────────────────────────────

// parseGraphUser extracts a Profile from a GraphQL user result node.
// Handles both data.user_result.result and data.user_results.result paths.
func parseGraphUser(node map[string]any) *Profile {
	// Try multiple paths for user result
	user := asMap(dig(node, "user_result", "result"))
	if user == nil {
		user = asMap(dig(node, "user_results", "result"))
	}
	if user == nil {
		// The node itself may be the user result (has rest_id, or core+legacy)
		if asStr(node["rest_id"]) != "" || (asMap(node["core"]) != nil && asMap(node["legacy"]) != nil) {
			user = node
		}
	}
	if user == nil {
		return nil
	}

	legacy := asMap(dig(user, "legacy"))
	if legacy == nil {
		// Fallback: newer API format without legacy
		return parseGraphUserDirect(user)
	}

	restID := asStr(dig(user, "rest_id"))
	p := &Profile{
		ID:             restID,
		FollowersCount: asInt(legacy["followers_count"]),
		FollowingCount: asInt(legacy["friends_count"]),
		TweetsCount:    asInt(legacy["statuses_count"]),
		LikesCount:     asInt(legacy["favourites_count"]),
		MediaCount:     asInt(legacy["media_count"]),
		ListedCount:    asInt(legacy["listed_count"]),
		FetchedAt:      time.Now(),
	}

	// Identity fields: try legacy first, then top-level core/avatar/location/privacy.
	// Twitter moved screen_name, name, created_at, location out of legacy.
	p.Username = asStr(legacy["screen_name"])
	if p.Username == "" {
		p.Username = asStr(dig(user, "core", "screen_name"))
	}
	p.Name = asStr(legacy["name"])
	if p.Name == "" {
		p.Name = asStr(dig(user, "core", "name"))
	}
	p.Joined = parseTwitterTime(asStr(legacy["created_at"]))
	if p.Joined.IsZero() {
		p.Joined = parseTwitterTime(asStr(dig(user, "core", "created_at")))
	}

	// Biography: try legacy, then profile_bio
	p.Biography = asStr(legacy["description"])
	if p.Biography == "" {
		p.Biography = asStr(dig(user, "profile_bio", "description"))
	}

	// Location: try legacy, then top-level location object
	p.Location = asStr(legacy["location"])
	if p.Location == "" {
		p.Location = asStr(dig(user, "location", "location"))
	}

	// Privacy: try legacy, then top-level privacy object
	p.IsPrivate = asBool(legacy["protected"])
	if !p.IsPrivate {
		p.IsPrivate = asBool(dig(user, "privacy", "protected"))
	}

	// Avatar: try legacy, then top-level avatar object
	pic := asStr(legacy["profile_image_url_https"])
	if pic == "" {
		pic = asStr(dig(user, "avatar", "image_url"))
	}
	p.Avatar = strings.Replace(pic, "_normal", "", 1)

	// Banner
	p.Banner = asStr(legacy["profile_banner_url"])

	// Website from entities
	if urls := asSlice(dig(legacy, "entities", "url", "urls")); len(urls) > 0 {
		if first := asMap(urls[0]); first != nil {
			p.Website = asStr(first["expanded_url"])
		}
	}

	// Verified status
	if asBool(user["is_blue_verified"]) {
		p.IsBlueVerified = true
	}
	if asBool(dig(user, "verification", "verified")) {
		p.IsVerified = true
	}
	if asStr(legacy["verified_type"]) != "" || asBool(legacy["verified"]) {
		p.IsVerified = true
	}

	// URL (t.co short link)
	p.URL = asStr(legacy["url"])

	// Pinned tweets
	if pins := asSlice(legacy["pinned_tweet_ids_str"]); len(pins) > 0 {
		for _, pin := range pins {
			if id := asStr(pin); id != "" {
				p.PinnedTweetIDs = append(p.PinnedTweetIDs, id)
			}
		}
	}

	// Professional info
	if prof := asMap(user["professional"]); prof != nil {
		p.ProfessionalType = asStr(prof["professional_type"])
		if cats := asSlice(prof["category"]); len(cats) > 0 {
			if cat := asMap(cats[0]); cat != nil {
				p.ProfessionalCategory = asStr(cat["name"])
			}
		}
	}

	// DM permission
	if asBool(legacy["can_dm"]) {
		p.CanDM = true
	}

	// Default profile/avatar
	p.DefaultProfile = asBool(legacy["default_profile"])
	p.DefaultAvatar = asBool(legacy["default_profile_image"])

	// Description URLs (expanded links in bio)
	if descURLs := asSlice(dig(legacy, "entities", "description", "urls")); len(descURLs) > 0 {
		for _, u := range descURLs {
			if um := asMap(u); um != nil {
				expanded := asStr(um["expanded_url"])
				if expanded != "" {
					p.DescriptionURLs = append(p.DescriptionURLs, expanded)
				}
			}
		}
	}

	return p
}

// parseGraphUserDirect handles newer API format where user fields are at top level.
func parseGraphUserDirect(user map[string]any) *Profile {
	p := &Profile{
		ID:        asStr(user["rest_id"]),
		Username:  asStr(dig(user, "core", "screen_name")),
		Name:      asStr(dig(user, "core", "name")),
		FetchedAt: time.Now(),
	}
	pic := asStr(dig(user, "avatar", "image_url"))
	p.Avatar = strings.Replace(pic, "_normal", "", 1)
	if asBool(user["is_blue_verified"]) {
		p.IsBlueVerified = true
	}
	return p
}

// parseGraphUserFromCore extracts user info from a tweet's core field.
func parseGraphUserFromCore(core map[string]any) (username, userID, name string) {
	if core == nil {
		return
	}
	// Try core.user_results.result or core.user_result.result
	user := asMap(dig(core, "user_results", "result"))
	if user == nil {
		user = asMap(dig(core, "user_result", "result"))
	}
	if user == nil {
		// New format: screen_name/name directly on core
		username = asStr(core["screen_name"])
		name = asStr(core["name"])
		return
	}
	userID = asStr(user["rest_id"])
	// Try legacy first (old format)
	legacy := asMap(user["legacy"])
	if legacy != nil {
		username = asStr(legacy["screen_name"])
		name = asStr(legacy["name"])
	}
	// Fallback to new format where fields moved to user.core
	if username == "" {
		username = asStr(dig(user, "core", "screen_name"))
	}
	if username == "" {
		username = asStr(user["screen_name"])
	}
	if name == "" {
		name = asStr(dig(user, "core", "name"))
	}
	if name == "" {
		name = asStr(user["name"])
	}
	return
}

// parseFollowUser extracts a FollowUser from a GraphQL user node.
func parseFollowUser(node map[string]any) *FollowUser {
	p := parseGraphUser(node)
	if p == nil {
		return nil
	}
	return &FollowUser{
		ID:             p.ID,
		Username:       p.Username,
		Name:           p.Name,
		Biography:      p.Biography,
		FollowersCount: p.FollowersCount,
		FollowingCount: p.FollowingCount,
		IsVerified:     p.IsVerified || p.IsBlueVerified,
		IsPrivate:      p.IsPrivate,
	}
}

// ── Tweet parsing ───────────────────────────────────────

// parseGraphTweet extracts a Tweet from a GraphQL tweet result node.
func parseGraphTweet(node map[string]any) *Tweet {
	if node == nil {
		return nil
	}

	typeName := asStr(node["__typename"])
	switch typeName {
	case "TweetUnavailable", "TweetTombstone", "TweetPreviewDisplay":
		return nil
	case "TweetWithVisibilityResults":
		return parseGraphTweet(asMap(node["tweet"]))
	}

	legacy := asMap(node["legacy"])
	if legacy == nil {
		return nil
	}

	restID := asStr(node["rest_id"])
	username, userID, name := parseGraphUserFromCore(asMap(node["core"]))

	t := &Tweet{
		ID:             restID,
		ConversationID: asStr(legacy["conversation_id_str"]),
		Text:           asStr(legacy["full_text"]),
		Username:       username,
		UserID:         userID,
		Name:           name,
		IsReply:        asStr(legacy["in_reply_to_status_id_str"]) != "",
		IsPin:          asBool(legacy["is_pinned"]),
		Likes:          asInt(legacy["favorite_count"]),
		Retweets:       asInt(legacy["retweet_count"]),
		Replies:        asInt(legacy["reply_count"]),
		Bookmarks:      asInt(legacy["bookmark_count"]),
		Quotes:         asInt(legacy["quote_count"]),
		Sensitive:      asBool(legacy["possibly_sensitive"]),
		Language:       asStr(legacy["lang"]),
		PostedAt:       parseCreatedAt(legacy),
		FetchedAt:      time.Now(),
	}

	// Permanent URL
	if username != "" {
		t.PermanentURL = fmt.Sprintf("https://x.com/%s/status/%s", username, restID)
	}

	// Views
	if viewCount := asStr(dig(node, "views", "count")); viewCount != "" {
		t.Views, _ = strconv.Atoi(viewCount)
	}

	// Source (client app name) — extract from HTML like <a href="...">Twitter Web App</a>
	if src := asStr(node["source"]); src != "" {
		t.Source = extractSourceName(src)
	} else if src := asStr(legacy["source"]); src != "" {
		t.Source = extractSourceName(src)
	}

	// Place / Geo
	if placeNode := asMap(legacy["place"]); placeNode != nil {
		t.Place = asStr(placeNode["full_name"])
	}

	// Edit info
	if editCtrl := asMap(node["edit_control"]); editCtrl != nil {
		if edits := asSlice(editCtrl["edit_tweet_ids"]); len(edits) > 1 {
			t.IsEdited = true
		}
	}

	// Reply info
	if replyTo := asStr(legacy["in_reply_to_status_id_str"]); replyTo != "" {
		t.ReplyToID = replyTo
	}
	t.ReplyToUser = asStr(legacy["in_reply_to_screen_name"])

	// Retweet
	if rt := asMap(dig(legacy, "retweeted_status_result", "result")); rt != nil {
		t.IsRetweet = true
		if rtLegacy := asMap(rt["legacy"]); rtLegacy != nil {
			t.RetweetedID = asStr(rt["rest_id"])
		}
	}
	if rt := asMap(dig(node, "retweeted_status_result", "result")); rt != nil {
		t.IsRetweet = true
		t.RetweetedID = asStr(rt["rest_id"])
	}

	// Quote
	if asBool(legacy["is_quote_status"]) {
		t.IsQuote = true
		t.QuotedID = asStr(legacy["quoted_status_id_str"])
	}

	// Media - photos, videos, GIFs from extended_entities
	parseMediaFromLegacy(legacy, t)

	// Also try newer media_entities path
	parseMediaFromEntities(node, t)

	// Hashtags
	if entities := asMap(legacy["entities"]); entities != nil {
		for _, h := range asSlice(entities["hashtags"]) {
			if hm := asMap(h); hm != nil {
				if tag := asStr(hm["text"]); tag != "" {
					t.Hashtags = append(t.Hashtags, tag)
				}
			}
		}
		// Mentions
		for _, m := range asSlice(entities["user_mentions"]) {
			if mm := asMap(m); mm != nil {
				if un := asStr(mm["screen_name"]); un != "" {
					t.Mentions = append(t.Mentions, "@"+un)
				}
			}
		}
		// URLs
		for _, u := range asSlice(entities["urls"]) {
			if um := asMap(u); um != nil {
				expanded := asStr(um["expanded_url"])
				if expanded == "" {
					expanded = asStr(um["url"])
				}
				if expanded != "" {
					t.URLs = append(t.URLs, expanded)
				}
			}
		}
	}

	// Note tweet (long tweets)
	if noteTweet := asMap(dig(node, "note_tweet", "note_tweet_results", "result")); noteTweet != nil {
		if noteText := asStr(noteTweet["text"]); noteText != "" {
			t.Text = noteText
		}
	}

	return t
}

// parseMediaFromLegacy extracts media from legacy.extended_entities.media
func parseMediaFromLegacy(legacy map[string]any, t *Tweet) {
	media := asSlice(dig(legacy, "extended_entities", "media"))
	if media == nil {
		return
	}
	for _, m := range media {
		mm := asMap(m)
		if mm == nil {
			continue
		}
		mediaType := asStr(mm["type"])
		switch mediaType {
		case "photo":
			if url := asStr(mm["media_url_https"]); url != "" {
				t.Photos = append(t.Photos, url)
			}
		case "video":
			variants := asSlice(dig(mm, "video_info", "variants"))
			bestURL := bestVideoVariant(variants)
			if bestURL != "" {
				t.Videos = append(t.Videos, bestURL)
			}
		case "animated_gif":
			variants := asSlice(dig(mm, "video_info", "variants"))
			if len(variants) > 0 {
				if first := asMap(variants[0]); first != nil {
					if url := asStr(first["url"]); url != "" {
						t.GIFs = append(t.GIFs, url)
					}
				}
			}
		}
	}
}

// parseMediaFromEntities extracts media from the newer media_entities path.
func parseMediaFromEntities(node map[string]any, t *Tweet) {
	mediaEntities := asSlice(node["media_entities"])
	if mediaEntities == nil {
		return
	}
	for _, me := range mediaEntities {
		mem := asMap(me)
		if mem == nil {
			continue
		}
		mediaInfo := asMap(dig(mem, "media_results", "result", "media_info"))
		if mediaInfo == nil {
			continue
		}
		typeName := asStr(mediaInfo["__typename"])
		switch typeName {
		case "ApiImage":
			if url := asStr(mediaInfo["original_img_url"]); url != "" {
				t.Photos = append(t.Photos, url)
			}
		case "ApiVideo":
			variants := asSlice(mediaInfo["variants"])
			bestURL := bestVideoVariant(variants)
			if bestURL != "" {
				t.Videos = append(t.Videos, bestURL)
			}
		case "ApiGif":
			variants := asSlice(mediaInfo["variants"])
			if len(variants) > 0 {
				if first := asMap(variants[0]); first != nil {
					if url := asStr(first["url"]); url != "" {
						t.GIFs = append(t.GIFs, url)
					}
				}
			}
		}
	}
}

// extractSourceName extracts the app name from Twitter's source HTML like
// `<a href="...">Twitter Web App</a>`.
func extractSourceName(src string) string {
	if i := strings.Index(src, ">"); i >= 0 {
		rest := src[i+1:]
		if j := strings.Index(rest, "<"); j >= 0 {
			return rest[:j]
		}
		return rest
	}
	return src
}

// bestVideoVariant picks the highest bitrate mp4 variant.
func bestVideoVariant(variants []any) string {
	var bestURL string
	var bestBitrate int
	for _, v := range variants {
		vm := asMap(v)
		if vm == nil {
			continue
		}
		ct := asStr(vm["content_type"])
		if ct != "video/mp4" {
			continue
		}
		bitrate := asInt(vm["bitrate"])
		if bitrate > bestBitrate || bestURL == "" {
			bestBitrate = bitrate
			bestURL = asStr(vm["url"])
		}
	}
	return bestURL
}

// ── Timeline parsing ────────────────────────────────────

// timelineResult holds tweets + bottom cursor from a timeline response.
type timelineResult struct {
	Tweets []Tweet
	Cursor string
}

// parseTimeline extracts tweets and bottom cursor from a GraphQL timeline response.
// Handles UserTweets, UserTweetsAndReplies, UserMedia, Home, ForYou, Bookmarks.
func parseTimeline(data map[string]any) timelineResult {
	var result timelineResult

	instructions := findInstructions(data)
	if instructions == nil {
		return result
	}

	for _, inst := range instructions {
		im := asMap(inst)
		if im == nil {
			continue
		}

		// Handle moduleItems (used in UserMedia)
		if moduleItems := asSlice(im["moduleItems"]); moduleItems != nil {
			for _, item := range moduleItems {
				tweet := extractTweetFromItem(asMap(item), "item")
				if tweet != nil {
					result.Tweets = append(result.Tweets, *tweet)
				}
			}
			continue
		}

		entries := asSlice(im["entries"])
		if entries == nil {
			continue
		}

		for _, e := range entries {
			em := asMap(e)
			if em == nil {
				continue
			}
			entryID := getEntryID(em)

			if strings.HasPrefix(entryID, "tweet") || strings.HasPrefix(entryID, "profile-grid") {
				tweets := extractTweetsFromEntry(em)
				result.Tweets = append(result.Tweets, tweets...)
			} else if strings.Contains(entryID, "-conversation-") || strings.HasPrefix(entryID, "homeConversation") {
				// Conversation thread
				tweets := extractTweetsFromConversationEntry(em)
				result.Tweets = append(result.Tweets, tweets...)
			} else if strings.HasPrefix(entryID, "cursor-bottom") {
				result.Cursor = asStr(dig(em, "content", "value"))
			}
		}
	}

	return result
}

// parseSearchTweets extracts tweets from a search response.
func parseSearchTweets(data map[string]any) timelineResult {
	var result timelineResult

	instructions := findSearchInstructions(data)
	if instructions == nil {
		return result
	}

	for _, inst := range instructions {
		im := asMap(inst)
		if im == nil {
			continue
		}
		typeName := asStr(im["type"])
		if typeName == "" {
			typeName = asStr(im["__typename"])
		}

		if typeName == "TimelineAddEntries" {
			for _, e := range asSlice(im["entries"]) {
				em := asMap(e)
				if em == nil {
					continue
				}
				entryID := getEntryID(em)
				if strings.HasPrefix(entryID, "tweet") {
					if tweet := getTweetResult(em); tweet != nil {
						t := parseGraphTweet(tweet)
						if t != nil {
							result.Tweets = append(result.Tweets, *t)
						}
					}
				} else if strings.HasPrefix(entryID, "cursor-bottom") {
					result.Cursor = asStr(dig(em, "content", "value"))
				}
			}
		} else if typeName == "TimelineReplaceEntry" {
			entryToReplace := asStr(im["entry_id_to_replace"])
			if strings.HasPrefix(entryToReplace, "cursor-bottom") {
				result.Cursor = asStr(dig(im, "entry", "content", "value"))
			}
		}
	}

	return result
}

// parseSearchUsers extracts users from a search response.
func parseSearchUsers(data map[string]any) ([]FollowUser, string) {
	var users []FollowUser
	var cursor string

	instructions := findSearchInstructions(data)
	if instructions == nil {
		return users, cursor
	}

	for _, inst := range instructions {
		im := asMap(inst)
		if im == nil {
			continue
		}
		typeName := asStr(im["type"])
		if typeName == "" {
			typeName = asStr(im["__typename"])
		}

		if typeName == "TimelineAddEntries" {
			for _, e := range asSlice(im["entries"]) {
				em := asMap(e)
				if em == nil {
					continue
				}
				entryID := getEntryID(em)
				if strings.HasPrefix(entryID, "user") {
					itemContent := asMap(dig(em, "content", "itemContent"))
					if itemContent == nil {
						continue
					}
					u := parseFollowUser(itemContent)
					if u != nil {
						users = append(users, *u)
					}
				} else if strings.HasPrefix(entryID, "cursor-bottom") {
					cursor = asStr(dig(em, "content", "value"))
				}
			}
		}
	}

	return users, cursor
}

// parseFollowList extracts users from a followers/following response.
func parseFollowList(data map[string]any) ([]FollowUser, string) {
	var users []FollowUser
	var cursor string

	instructions := findInstructions(data)
	if instructions == nil {
		return users, cursor
	}

	for _, inst := range instructions {
		im := asMap(inst)
		if im == nil {
			continue
		}
		entries := asSlice(im["entries"])
		if entries == nil {
			continue
		}
		for _, e := range entries {
			em := asMap(e)
			if em == nil {
				continue
			}
			entryID := getEntryID(em)
			if strings.HasPrefix(entryID, "user") {
				itemContent := asMap(dig(em, "content", "itemContent"))
				if itemContent == nil {
					continue
				}
				u := parseFollowUser(itemContent)
				if u != nil {
					users = append(users, *u)
				}
			} else if strings.HasPrefix(entryID, "cursor-bottom") {
				cursor = asStr(dig(em, "content", "value"))
			}
		}
	}

	return users, cursor
}

// parseConversation extracts the main tweet and replies from a ConversationTimeline response.
func parseConversation(data map[string]any, tweetID string) (*Tweet, []Tweet, string) {
	var mainTweet *Tweet
	var replies []Tweet
	var cursor string

	instructions := findConversationInstructions(data)
	if instructions == nil {
		return nil, nil, ""
	}

	for _, inst := range instructions {
		im := asMap(inst)
		if im == nil {
			continue
		}
		typeName := asStr(im["type"])
		if typeName == "" {
			typeName = asStr(im["__typename"])
		}
		if typeName != "TimelineAddEntries" {
			continue
		}

		for _, e := range asSlice(im["entries"]) {
			em := asMap(e)
			if em == nil {
				continue
			}
			entryID := getEntryID(em)

			if strings.HasPrefix(entryID, "tweet") {
				tweetResult := getTweetResult(em)
				if tweetResult != nil {
					t := parseGraphTweet(tweetResult)
					if t != nil {
						if t.ID == tweetID {
							mainTweet = t
						} else {
							replies = append(replies, *t)
						}
					}
				}
			} else if strings.HasPrefix(entryID, "conversationthread") {
				// Thread replies
				items := asSlice(dig(em, "content", "items"))
				for _, item := range items {
					itemMap := asMap(item)
					if itemMap == nil {
						continue
					}
					itemEntryID := getEntryID(itemMap)
					if strings.Contains(itemEntryID, "cursor-showmore") {
						// cursor for more replies
						val := asStr(dig(itemMap, "item", "content", "value"))
						if val == "" {
							val = asStr(dig(itemMap, "item", "itemContent", "value"))
						}
						if cursor == "" {
							cursor = val
						}
					} else if strings.Contains(itemEntryID, "tweet") {
						tr := extractTweetFromItem(itemMap, "item")
						if tr != nil {
							replies = append(replies, *tr)
						}
					}
				}
			} else if strings.HasPrefix(entryID, "cursor-bottom") {
				val := asStr(dig(em, "content", "value"))
				if val == "" {
					val = asStr(dig(em, "content", "content", "value"))
				}
				if cursor == "" {
					cursor = val
				}
			}
		}
	}

	return mainTweet, replies, cursor
}

// ── Helpers ─────────────────────────────────────────────

// findInstructions finds instructions array from various GraphQL response shapes.
func findInstructions(data map[string]any) []any {
	// Try multiple paths (from Nitter parser.nim parseGraphTimeline)
	paths := [][]string{
		{"data", "user", "result", "timeline", "timeline", "instructions"},
		{"data", "user_result", "result", "timeline_response", "timeline", "instructions"},
		{"data", "list", "timeline_response", "timeline", "instructions"},
		{"data", "timeline_response", "timeline", "instructions"},
		{"data", "home", "home_timeline_urt", "instructions"},
		{"data", "bookmark_timeline_v2", "timeline", "instructions"},
		{"data", "bookmark_search_timeline", "timeline", "instructions"},
	}
	for _, path := range paths {
		if v := asSlice(dig(data, path...)); v != nil {
			return v
		}
	}
	return nil
}

// findSearchInstructions finds instructions from search response.
func findSearchInstructions(data map[string]any) []any {
	paths := [][]string{
		{"data", "search_by_raw_query", "search_timeline", "timeline", "instructions"},
		{"data", "search", "timeline_response", "timeline", "instructions"},
	}
	for _, path := range paths {
		if v := asSlice(dig(data, path...)); v != nil {
			return v
		}
	}
	return nil
}

// findConversationInstructions finds instructions from conversation response.
func findConversationInstructions(data map[string]any) []any {
	paths := [][]string{
		{"data", "threaded_conversation_with_injections_v2", "instructions"},
		{"data", "timelineResponse", "instructions"},
		{"data", "timeline_response", "instructions"},
	}
	for _, path := range paths {
		if v := asSlice(dig(data, path...)); v != nil {
			return v
		}
	}
	return nil
}

// getEntryID extracts entryId from an entry node.
func getEntryID(entry map[string]any) string {
	if id := asStr(entry["entryId"]); id != "" {
		return id
	}
	return asStr(entry["entry_id"])
}

// getTweetResult navigates to the tweet result node within an entry.
func getTweetResult(entry map[string]any) map[string]any {
	// Try content.itemContent.tweet_results.result
	if r := asMap(dig(entry, "content", "itemContent", "tweet_results", "result")); r != nil {
		return r
	}
	// Try content.content.tweetResult.result
	if r := asMap(dig(entry, "content", "content", "tweetResult", "result")); r != nil {
		return r
	}
	// Try content.content.tweet_results.result
	if r := asMap(dig(entry, "content", "content", "tweet_results", "result")); r != nil {
		return r
	}
	return nil
}

// extractTweetFromItem extracts a tweet from an item node (used in threads/modules).
func extractTweetFromItem(item map[string]any, prefix string) *Tweet {
	if item == nil {
		return nil
	}
	// Try item.{prefix}.content.tweet_results.result
	if r := asMap(dig(item, prefix, "content", "tweet_results", "result")); r != nil {
		return parseGraphTweet(r)
	}
	// Try item.{prefix}.itemContent.tweet_results.result
	if r := asMap(dig(item, prefix, "itemContent", "tweet_results", "result")); r != nil {
		return parseGraphTweet(r)
	}
	return nil
}

// extractTweetsFromEntry extracts all tweets from a timeline entry.
func extractTweetsFromEntry(entry map[string]any) []Tweet {
	var tweets []Tweet

	// Direct tweet
	if r := getTweetResult(entry); r != nil {
		t := parseGraphTweet(r)
		if t != nil {
			tweets = append(tweets, *t)
		}
		return tweets
	}

	// Multiple items in entry (conversation thread)
	items := asSlice(dig(entry, "content", "items"))
	for _, item := range items {
		t := extractTweetFromItem(asMap(item), "item")
		if t != nil {
			tweets = append(tweets, *t)
		}
	}

	return tweets
}

// extractTweetsFromConversationEntry extracts tweets from a conversation thread entry.
func extractTweetsFromConversationEntry(entry map[string]any) []Tweet {
	var tweets []Tweet
	items := asSlice(dig(entry, "content", "items"))
	for _, item := range items {
		t := extractTweetFromItem(asMap(item), "item")
		if t != nil {
			tweets = append(tweets, *t)
		}
	}
	return tweets
}

// ── List parsing ────────────────────────────────────────

// parseGraphList extracts a List from a GraphQL list result response.
func parseGraphList(data map[string]any) *List {
	// Try multiple paths for list result
	node := asMap(dig(data, "data", "list"))
	if node == nil {
		node = asMap(dig(data, "data", "list_by_rest_id", "result"))
	}
	if node == nil {
		node = asMap(dig(data, "data", "list_by_slug", "result"))
	}
	if node == nil {
		return nil
	}

	l := &List{
		ID:          asStr(node["id_str"]),
		Name:        asStr(node["name"]),
		Description: asStr(node["description"]),
		MemberCount: asInt(node["member_count"]),
	}

	if l.ID == "" {
		l.ID = asStr(node["rest_id"])
	}

	// Banner
	if banner := asMap(node["custom_banner_media"]); banner != nil {
		if mediaInfo := asMap(banner["media_info"]); mediaInfo != nil {
			if origImg := asStr(mediaInfo["original_img_url"]); origImg != "" {
				l.Banner = origImg
			}
		}
	}
	if l.Banner == "" {
		l.Banner = asStr(node["default_banner_media_url"])
	}

	// Owner
	if userResults := asMap(dig(node, "user_results", "result")); userResults != nil {
		l.OwnerID = asStr(userResults["rest_id"])
		if legacy := asMap(userResults["legacy"]); legacy != nil {
			l.OwnerName = asStr(legacy["screen_name"])
		}
	}

	return l
}

// findListInstructions finds instructions from a list timeline response.
func findListInstructions(data map[string]any) []any {
	paths := [][]string{
		{"data", "list", "tweets_timeline", "timeline", "instructions"},
		{"data", "list", "timeline_response", "timeline", "instructions"},
	}
	for _, path := range paths {
		if v := asSlice(dig(data, path...)); v != nil {
			return v
		}
	}
	return nil
}

// parseListTimeline extracts tweets from a list timeline response.
func parseListTimeline(data map[string]any) timelineResult {
	var result timelineResult

	instructions := findListInstructions(data)
	if instructions == nil {
		// Fall back to generic instructions finder
		instructions = findInstructions(data)
	}
	if instructions == nil {
		return result
	}

	for _, inst := range instructions {
		im := asMap(inst)
		if im == nil {
			continue
		}

		entries := asSlice(im["entries"])
		if entries == nil {
			continue
		}

		for _, e := range entries {
			em := asMap(e)
			if em == nil {
				continue
			}
			entryID := getEntryID(em)

			if strings.HasPrefix(entryID, "tweet") || strings.HasPrefix(entryID, "list-") {
				tweets := extractTweetsFromEntry(em)
				result.Tweets = append(result.Tweets, tweets...)
			} else if strings.Contains(entryID, "-conversation-") {
				tweets := extractTweetsFromConversationEntry(em)
				result.Tweets = append(result.Tweets, tweets...)
			} else if strings.HasPrefix(entryID, "cursor-bottom") {
				result.Cursor = asStr(dig(em, "content", "value"))
			}
		}
	}

	return result
}

// parseListMembers extracts users from a list members response.
func parseListMembers(data map[string]any) ([]FollowUser, string) {
	var users []FollowUser
	var cursor string

	paths := [][]string{
		{"data", "list", "members_timeline", "timeline", "instructions"},
		{"data", "list", "timeline_response", "timeline", "instructions"},
	}

	var instructions []any
	for _, path := range paths {
		if v := asSlice(dig(data, path...)); v != nil {
			instructions = v
			break
		}
	}
	if instructions == nil {
		// Fall back to generic
		return parseFollowList(data)
	}

	for _, inst := range instructions {
		im := asMap(inst)
		if im == nil {
			continue
		}
		entries := asSlice(im["entries"])
		if entries == nil {
			continue
		}
		for _, e := range entries {
			em := asMap(e)
			if em == nil {
				continue
			}
			entryID := getEntryID(em)
			if strings.HasPrefix(entryID, "user") || strings.HasPrefix(entryID, "list-user") {
				itemContent := asMap(dig(em, "content", "itemContent"))
				if itemContent == nil {
					continue
				}
				u := parseFollowUser(itemContent)
				if u != nil {
					users = append(users, *u)
				}
			} else if strings.HasPrefix(entryID, "cursor-bottom") {
				cursor = asStr(dig(em, "content", "value"))
			}
		}
	}

	return users, cursor
}

// parseUserResult extracts a Profile from a UserByScreenName response.
func parseUserResult(data map[string]any) *Profile {
	node := asMap(dig(data, "data", "user", "result"))
	if node == nil {
		node = asMap(dig(data, "data", "user_result", "result"))
	}
	if node == nil {
		return nil
	}
	// Reuse the same logic as parseGraphUser, which handles both old and new API formats.
	return parseGraphUser(node)
}
