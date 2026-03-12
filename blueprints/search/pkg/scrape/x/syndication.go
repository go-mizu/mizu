package x

// syndication.go — fetch individual tweets via the Twitter syndication/embed API.
//
// Twitter's embed widget uses cdn.syndication.twimg.com/tweet-result which:
//   - Requires no authentication (no auth_token/ct0)
//   - Has much more generous rate limits than the GraphQL API
//   - Returns a subset of tweet data (no replies, no full thread)
//
// Token formula (from Twitter's embed JS bundle):
//   token = Math.round(parseInt(id) / 1e15 * Math.PI).toString(36)

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const syndicationBaseURL = "https://cdn.syndication.twimg.com/tweet-result"

// syndicationToken computes the required token parameter from a tweet ID.
// Formula mirrors Twitter's embed JS: Math.round(parseInt(id) / 1e15 * Math.PI).toString(36)
func syndicationToken(id string) string {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "0"
	}
	v := math.Round(float64(n) / 1e15 * math.Pi)
	return strconv.FormatInt(int64(v), 36)
}

// syndicationTweet is the raw JSON response from the syndication API.
type syndicationTweet struct {
	IDStr          string `json:"id_str"`
	Text           string `json:"text"`
	FullText       string `json:"full_text"`
	CreatedAt      string `json:"created_at"`
	ConvIDStr      string `json:"conversation_id_str"`
	Lang           string `json:"lang"`
	PossiblySens   bool   `json:"possibly_sensitive"`
	FavoriteCount  int    `json:"favorite_count"`
	RetweetCount   int    `json:"retweet_count"`
	ReplyCount     int    `json:"reply_count"`
	QuoteCount     int    `json:"quote_count"`
	BookmarkCount  int    `json:"bookmark_count"`
	User           struct {
		IDStr              string `json:"id_str"`
		Name               string `json:"name"`
		ScreenName         string `json:"screen_name"`
		ProfileImageURL    string `json:"profile_image_url_https"`
		Verified           bool   `json:"verified"`
		IsBlueVerified     bool   `json:"is_blue_verified"`
	} `json:"user"`
	Entities struct {
		Hashtags     []struct{ Text string `json:"text"` } `json:"hashtags"`
		Urls         []struct {
			URL         string `json:"url"`
			ExpandedURL string `json:"expanded_url"`
		} `json:"urls"`
		UserMentions []struct{ ScreenName string `json:"screen_name"` } `json:"user_mentions"`
	} `json:"entities"`
	ExtendedEntities struct {
		Media []struct {
			Type      string `json:"type"` // "photo", "video", "animated_gif"
			MediaURL  string `json:"media_url_https"`
			VideoInfo *struct {
				Variants []struct {
					ContentType string `json:"content_type"`
					URL         string `json:"url"`
					Bitrate     int    `json:"bitrate"`
				} `json:"variants"`
			} `json:"video_info"`
		} `json:"media"`
	} `json:"extended_entities"`
	InReplyToStatusIDStr   string `json:"in_reply_to_status_id_str"`
	InReplyToScreenName    string `json:"in_reply_to_screen_name"`
	RetweetedStatusIDStr   string `json:"retweeted_status_id_str"`
	QuotedStatusIDStr      string `json:"quoted_status_id_str"`
	Source                 string `json:"source"`
}

// parseSyndicationTweet converts the raw syndication response to a Tweet.
func parseSyndicationTweet(raw *syndicationTweet) *Tweet {
	t := &Tweet{
		ID:             raw.IDStr,
		ConversationID: raw.ConvIDStr,
		Username:       raw.User.ScreenName,
		UserID:         raw.User.IDStr,
		Name:           raw.User.Name,
		Likes:          raw.FavoriteCount,
		Retweets:       raw.RetweetCount,
		Replies:        raw.ReplyCount,
		Quotes:         raw.QuoteCount,
		Bookmarks:      raw.BookmarkCount,
		Language:       raw.Lang,
		Sensitive:      raw.PossiblySens,
		ReplyToID:      raw.InReplyToStatusIDStr,
		ReplyToUser:    raw.InReplyToScreenName,
		RetweetedID:    raw.RetweetedStatusIDStr,
		QuotedID:       raw.QuotedStatusIDStr,
		FetchedAt:      time.Now(),
	}

	// Prefer full_text over text
	if raw.FullText != "" {
		t.Text = raw.FullText
	} else {
		t.Text = raw.Text
	}

	// Parse source app name from HTML anchor (e.g. <a href="...">Twitter Web App</a>)
	if raw.Source != "" {
		start := strings.Index(raw.Source, ">")
		end := strings.LastIndex(raw.Source, "<")
		if start >= 0 && end > start {
			t.Source = raw.Source[start+1 : end]
		}
	}

	// Parse created_at — syndication API returns ISO 8601 ("2006-01-02T15:04:05.000Z")
	if raw.CreatedAt != "" {
		if pt, err := time.Parse(time.RFC3339Nano, raw.CreatedAt); err == nil {
			t.PostedAt = pt
		} else if pt, err := time.Parse(time.RFC3339, raw.CreatedAt); err == nil {
			t.PostedAt = pt
		} else if pt, err := time.Parse(time.RubyDate, raw.CreatedAt); err == nil {
			t.PostedAt = pt // fallback for legacy format
		}
	}

	t.PermanentURL = "https://x.com/" + t.Username + "/status/" + t.ID
	t.IsReply = t.ReplyToID != ""
	t.IsRetweet = t.RetweetedID != ""
	t.IsQuote = t.QuotedID != ""

	// Hashtags
	for _, h := range raw.Entities.Hashtags {
		t.Hashtags = append(t.Hashtags, h.Text)
	}
	// Mentions
	for _, m := range raw.Entities.UserMentions {
		t.Mentions = append(t.Mentions, m.ScreenName)
	}
	// URLs (expanded)
	for _, u := range raw.Entities.Urls {
		if u.ExpandedURL != "" {
			t.URLs = append(t.URLs, u.ExpandedURL)
		}
	}

	// Media
	for _, m := range raw.ExtendedEntities.Media {
		switch m.Type {
		case "photo":
			t.Photos = append(t.Photos, m.MediaURL)
		case "animated_gif":
			t.GIFs = append(t.GIFs, m.MediaURL)
		case "video":
			// Pick highest-bitrate mp4
			best := ""
			bestBitrate := 0
			for _, v := range m.VideoInfo.Variants {
				if v.ContentType == "video/mp4" && v.Bitrate >= bestBitrate {
					best = v.URL
					bestBitrate = v.Bitrate
				}
			}
			if best != "" {
				t.Videos = append(t.Videos, best)
			}
		}
	}

	return t
}

// GetTweetSyndication fetches a single tweet via the syndication/embed API.
// No authentication required — uses the same endpoint as Twitter's embed widget.
// Returns less data than the GraphQL API (no replies, no full thread context).
func GetTweetSyndication(id string) (*Tweet, error) {
	token := syndicationToken(id)
	url := fmt.Sprintf("%s?id=%s&lang=en&token=%s", syndicationBaseURL, id, token)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("syndication request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://platform.twitter.com/")
	req.Header.Set("Origin", "https://platform.twitter.com")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("syndication fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("syndication read: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("tweet %s not found", id)
	}
	if resp.StatusCode != 200 {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("syndication HTTP %d: %s", resp.StatusCode, snippet)
	}

	var raw syndicationTweet
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("syndication parse: %w", err)
	}
	if raw.IDStr == "" {
		return nil, fmt.Errorf("tweet %s: empty response from syndication API", id)
	}

	return parseSyndicationTweet(&raw), nil
}
