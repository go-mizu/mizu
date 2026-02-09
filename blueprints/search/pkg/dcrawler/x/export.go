package x

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportJSON exports tweets to a JSON file.
func ExportJSON(tweets []Tweet, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tweets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ExportCSV exports tweets to a CSV file.
func ExportCSV(tweets []Tweet, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	if err := w.Write([]string{
		"id", "username", "name", "text", "posted_at",
		"likes", "retweets", "replies", "views", "bookmarks", "quotes",
		"is_retweet", "is_reply", "is_quote",
		"reply_to_id", "reply_to_user", "quoted_id", "retweeted_id",
		"photos", "videos", "gifs", "hashtags", "mentions", "urls",
		"language", "source", "place", "is_edited",
		"permanent_url",
	}); err != nil {
		return err
	}

	for _, t := range tweets {
		if err := w.Write([]string{
			t.ID, t.Username, t.Name, t.Text, t.PostedAt.Format(time.RFC3339),
			fmt.Sprint(t.Likes), fmt.Sprint(t.Retweets), fmt.Sprint(t.Replies),
			fmt.Sprint(t.Views), fmt.Sprint(t.Bookmarks), fmt.Sprint(t.Quotes),
			fmt.Sprint(t.IsRetweet), fmt.Sprint(t.IsReply), fmt.Sprint(t.IsQuote),
			t.ReplyToID, t.ReplyToUser, t.QuotedID, t.RetweetedID,
			strings.Join(t.Photos, "|"), strings.Join(t.Videos, "|"),
			strings.Join(t.GIFs, "|"), strings.Join(t.Hashtags, "|"),
			strings.Join(t.Mentions, "|"), strings.Join(t.URLs, "|"),
			t.Language, t.Source, t.Place, fmt.Sprint(t.IsEdited),
			t.PermanentURL,
		}); err != nil {
			return err
		}
	}

	return nil
}

// rssXML types for RSS 2.0 feed generation.
type rssXML struct {
	XMLName xml.Name    `xml:"rss"`
	Version string      `xml:"version,attr"`
	Channel rssChannel  `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Author      string `xml:"author"`
}

// ExportRSS generates an RSS 2.0 feed from tweets.
func ExportRSS(tweets []Tweet, title, link, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	items := make([]rssItem, 0, len(tweets))
	for _, t := range tweets {
		desc := t.Text
		if len(t.Photos) > 0 {
			for _, p := range t.Photos {
				desc += fmt.Sprintf(`<br/><img src="%s"/>`, p)
			}
		}
		items = append(items, rssItem{
			Title:       fmt.Sprintf("@%s: %s", t.Username, truncate(t.Text, 100)),
			Link:        t.PermanentURL,
			Description: desc,
			PubDate:     t.PostedAt.Format(time.RFC1123Z),
			GUID:        t.PermanentURL,
			Author:      t.Username,
		})
	}

	rss := rssXML{
		Version: "2.0",
		Channel: rssChannel{
			Title:       title,
			Link:        link,
			Description: fmt.Sprintf("X/Twitter feed: %s", title),
			PubDate:     time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}

	data, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append([]byte(xml.Header), data...), 0o644)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
