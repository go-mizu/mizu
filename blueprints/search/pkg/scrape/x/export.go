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
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
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
		var desc strings.Builder
		desc.WriteString(t.Text)
		if len(t.Photos) > 0 {
			for _, p := range t.Photos {
				desc.WriteString(fmt.Sprintf(`<br/><img src="%s"/>`, p))
			}
		}
		items = append(items, rssItem{
			Title:       fmt.Sprintf("@%s: %s", t.Username, truncate(t.Text, 100)),
			Link:        t.PermanentURL,
			Description: desc.String(),
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

// ExportMarkdown exports a tweet thread to a markdown file.
func ExportMarkdown(thread []Tweet, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(TweetThreadToMarkdown(thread)), 0o644)
}

// TweetThreadToMarkdown renders a tweet thread (root + self-replies) as a markdown document.
func TweetThreadToMarkdown(thread []Tweet) string {
	if len(thread) == 0 {
		return ""
	}

	root := thread[0]
	var sb strings.Builder

	// Header: use note tweet title if available, else author
	if root.Title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", root.Title))
		if root.Name != "" {
			sb.WriteString(fmt.Sprintf("**%s** (@%s)\n\n", root.Name, root.Username))
		} else {
			sb.WriteString(fmt.Sprintf("@%s\n\n", root.Username))
		}
	} else if root.Name != "" {
		sb.WriteString(fmt.Sprintf("# %s (@%s)\n\n", root.Name, root.Username))
	} else {
		sb.WriteString(fmt.Sprintf("# @%s\n\n", root.Username))
	}

	sb.WriteString(fmt.Sprintf("*%s*\n\n", root.PostedAt.UTC().Format("2006-01-02 15:04 UTC")))

	sb.WriteString(fmt.Sprintf("👍 %s · 🔄 %s · 💬 %s",
		fmtNum(root.Likes), fmtNum(root.Retweets), fmtNum(root.Replies)))
	if root.Views > 0 {
		sb.WriteString(fmt.Sprintf(" · 👁 %s", fmtNum(root.Views)))
	}
	sb.WriteString("\n\n---\n\n")

	// Thread body
	for i, t := range thread {
		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		// Render body: preserve paragraphs (double newlines → markdown paragraphs)
		body := strings.TrimSpace(t.Text)
		sb.WriteString(body)
		sb.WriteString("\n")
		for _, p := range t.Photos {
			sb.WriteString(fmt.Sprintf("\n![image](%s)\n", p))
		}
		for j, v := range t.Videos {
			sb.WriteString(fmt.Sprintf("\n[Video %d](%s)\n", j+1, v))
		}
	}

	sb.WriteString("\n\n---\n\n")
	if root.PermanentURL != "" {
		sb.WriteString(fmt.Sprintf("*Source: [%s](%s)*\n", root.PermanentURL, root.PermanentURL))
	}

	return sb.String()
}

// ExtractThread returns the root tweet followed by the author's self-replies in order.
func ExtractThread(root Tweet, replies []Tweet) []Tweet {
	thread := []Tweet{root}
	for _, r := range replies {
		if r.Username == root.Username {
			thread = append(thread, r)
		}
	}
	return thread
}

func fmtNum(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprint(n)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
