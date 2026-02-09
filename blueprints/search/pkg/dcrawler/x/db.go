package x

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing X/Twitter data.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens or creates a DuckDB database at the given path.
func OpenDB(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb %s: %w", path, err)
	}

	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

func (d *DB) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tweets (
			id              VARCHAR PRIMARY KEY,
			conversation_id VARCHAR,
			text            VARCHAR,
			html            VARCHAR,
			username        VARCHAR NOT NULL,
			user_id         VARCHAR,
			name            VARCHAR,
			permanent_url   VARCHAR,
			is_retweet      BOOLEAN DEFAULT FALSE,
			is_reply        BOOLEAN DEFAULT FALSE,
			is_quote        BOOLEAN DEFAULT FALSE,
			is_pin          BOOLEAN DEFAULT FALSE,
			reply_to_id     VARCHAR,
			reply_to_user   VARCHAR,
			quoted_id       VARCHAR,
			retweeted_id    VARCHAR,
			likes           INTEGER DEFAULT 0,
			retweets        INTEGER DEFAULT 0,
			replies         INTEGER DEFAULT 0,
			views           INTEGER DEFAULT 0,
			bookmarks       INTEGER DEFAULT 0,
			quotes          INTEGER DEFAULT 0,
			photos          VARCHAR,
			videos          VARCHAR,
			gifs            VARCHAR,
			hashtags        VARCHAR,
			mentions        VARCHAR,
			urls            VARCHAR,
			sensitive       BOOLEAN DEFAULT FALSE,
			language        VARCHAR,
			source          VARCHAR,
			place           VARCHAR,
			is_edited       BOOLEAN DEFAULT FALSE,
			posted_at       TIMESTAMP,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id                     VARCHAR PRIMARY KEY,
			username               VARCHAR NOT NULL,
			name                   VARCHAR,
			biography              VARCHAR,
			avatar                 VARCHAR,
			banner                 VARCHAR,
			location               VARCHAR,
			website                VARCHAR,
			url                    VARCHAR,
			joined                 TIMESTAMP,
			birthday               VARCHAR,
			followers_count        INTEGER DEFAULT 0,
			following_count        INTEGER DEFAULT 0,
			tweets_count           INTEGER DEFAULT 0,
			likes_count            INTEGER DEFAULT 0,
			media_count            INTEGER DEFAULT 0,
			listed_count           INTEGER DEFAULT 0,
			is_private             BOOLEAN DEFAULT FALSE,
			is_verified            BOOLEAN DEFAULT FALSE,
			is_blue_verified       BOOLEAN DEFAULT FALSE,
			pinned_tweet_ids       VARCHAR,
			professional_type      VARCHAR,
			professional_category  VARCHAR,
			can_dm                 BOOLEAN DEFAULT FALSE,
			default_profile        BOOLEAN DEFAULT FALSE,
			default_avatar         BOOLEAN DEFAULT FALSE,
			description_urls       VARCHAR,
			fetched_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}

	// Migrate existing databases: add new columns if missing.
	migrations := []string{
		"ALTER TABLE tweets ADD COLUMN IF NOT EXISTS reply_to_user VARCHAR",
		"ALTER TABLE tweets ADD COLUMN IF NOT EXISTS bookmarks INTEGER DEFAULT 0",
		"ALTER TABLE tweets ADD COLUMN IF NOT EXISTS quotes INTEGER DEFAULT 0",
		"ALTER TABLE tweets ADD COLUMN IF NOT EXISTS language VARCHAR",
		"ALTER TABLE tweets ADD COLUMN IF NOT EXISTS source VARCHAR",
		"ALTER TABLE tweets ADD COLUMN IF NOT EXISTS place VARCHAR",
		"ALTER TABLE tweets ADD COLUMN IF NOT EXISTS is_edited BOOLEAN DEFAULT FALSE",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS url VARCHAR",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS pinned_tweet_ids VARCHAR",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS professional_type VARCHAR",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS professional_category VARCHAR",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS can_dm BOOLEAN DEFAULT FALSE",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS default_profile BOOLEAN DEFAULT FALSE",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS default_avatar BOOLEAN DEFAULT FALSE",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS description_urls VARCHAR",
	}
	for _, m := range migrations {
		d.db.Exec(m) // ignore errors (column may already exist)
	}

	return nil
}

// tweetColumns is the canonical column list for tweet queries.
const tweetColumns = `id, conversation_id, text, html, username, user_id, name, permanent_url,
	is_retweet, is_reply, is_quote, is_pin,
	reply_to_id, reply_to_user, quoted_id, retweeted_id,
	likes, retweets, replies, views, bookmarks, quotes,
	photos, videos, gifs, hashtags, mentions, urls,
	sensitive, language, source, place, is_edited,
	posted_at, fetched_at`

// InsertTweets bulk-inserts tweets into the database.
func (d *DB) InsertTweets(tweets []Tweet) error {
	if len(tweets) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const batchSize = 500
	const nCols = 35
	placeholders := "(" + strings.Repeat("?,", nCols-1) + "?)"

	for i := 0; i < len(tweets); i += batchSize {
		end := min(i+batchSize, len(tweets))
		batch := tweets[i:end]

		var sb strings.Builder
		sb.WriteString("INSERT OR REPLACE INTO tweets (" + tweetColumns + ") VALUES ")

		args := make([]any, 0, len(batch)*nCols)
		for j, t := range batch {
			if j > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(placeholders)

			args = append(args,
				t.ID, nullStr(t.ConversationID), t.Text, nullStr(t.HTML),
				t.Username, nullStr(t.UserID), nullStr(t.Name), nullStr(t.PermanentURL),
				t.IsRetweet, t.IsReply, t.IsQuote, t.IsPin,
				nullStr(t.ReplyToID), nullStr(t.ReplyToUser), nullStr(t.QuotedID), nullStr(t.RetweetedID),
				t.Likes, t.Retweets, t.Replies, t.Views, t.Bookmarks, t.Quotes,
				jsonArray(t.Photos), jsonArray(t.Videos), jsonArray(t.GIFs),
				jsonArray(t.Hashtags), jsonArray(t.Mentions), jsonArray(t.URLs),
				t.Sensitive, nullStr(t.Language), nullStr(t.Source), nullStr(t.Place), t.IsEdited,
				t.PostedAt, t.FetchedAt,
			)
		}

		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("insert tweets batch: %w", err)
		}
	}

	return tx.Commit()
}

// InsertUser inserts or updates a user profile.
func (d *DB) InsertUser(p *Profile) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO users (
		id, username, name, biography, avatar, banner, location, website, url,
		joined, birthday, followers_count, following_count, tweets_count,
		likes_count, media_count, listed_count, is_private, is_verified, is_blue_verified,
		pinned_tweet_ids, professional_type, professional_category,
		can_dm, default_profile, default_avatar, description_urls, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Username, p.Name, p.Biography, p.Avatar, p.Banner,
		p.Location, p.Website, nullStr(p.URL), nullTime(p.Joined), p.Birthday,
		p.FollowersCount, p.FollowingCount, p.TweetsCount,
		p.LikesCount, p.MediaCount, p.ListedCount,
		p.IsPrivate, p.IsVerified, p.IsBlueVerified,
		jsonArray(p.PinnedTweetIDs), nullStr(p.ProfessionalType), nullStr(p.ProfessionalCategory),
		p.CanDM, p.DefaultProfile, p.DefaultAvatar, jsonArray(p.DescriptionURLs),
		p.FetchedAt,
	)
	return err
}

// InsertFollowUsers bulk-inserts follow users.
func (d *DB) InsertFollowUsers(users []FollowUser) error {
	if len(users) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const batchSize = 500
	for i := 0; i < len(users); i += batchSize {
		end := min(i+batchSize, len(users))
		batch := users[i:end]

		var sb strings.Builder
		sb.WriteString(`INSERT OR REPLACE INTO users (id, username, name, biography, followers_count, following_count, is_private, is_verified, fetched_at) VALUES `)

		args := make([]any, 0, len(batch)*9)
		for j, u := range batch {
			if j > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("(?,?,?,?,?,?,?,?,?)")
			args = append(args, u.ID, u.Username, u.Name, u.Biography,
				u.FollowersCount, u.FollowingCount, u.IsPrivate, u.IsVerified,
				time.Now())
		}

		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("insert users batch: %w", err)
		}
	}

	return tx.Commit()
}

// DBStats holds database statistics.
type DBStats struct {
	Tweets int64
	Users  int64
	DBSize int64
}

// GetStats returns database statistics.
func (d *DB) GetStats() (DBStats, error) {
	var stats DBStats

	row := d.db.QueryRow("SELECT COUNT(*) FROM tweets")
	row.Scan(&stats.Tweets)

	row = d.db.QueryRow("SELECT COUNT(*) FROM users")
	row.Scan(&stats.Users)

	if fi, err := os.Stat(d.path); err == nil {
		stats.DBSize = fi.Size()
	}

	return stats, nil
}

// TopTweets returns the top N tweets by likes.
func (d *DB) TopTweets(limit int) ([]Tweet, error) {
	return d.queryTweets("SELECT "+tweetColumns+" FROM tweets ORDER BY likes DESC LIMIT ?", limit)
}

// AllTweets returns all tweets ordered by posted_at desc.
func (d *DB) AllTweets() ([]Tweet, error) {
	return d.queryTweets("SELECT "+tweetColumns+" FROM tweets ORDER BY posted_at DESC", -1)
}

func (d *DB) queryTweets(query string, args ...any) ([]Tweet, error) {
	var queryArgs []any
	for _, a := range args {
		if n, ok := a.(int); ok && n < 0 {
			continue
		}
		queryArgs = append(queryArgs, a)
	}
	rows, err := d.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tweets []Tweet
	for rows.Next() {
		var t Tweet
		var convID, html, userID, name, permURL sql.NullString
		var replyTo, replyToUser, quotedID, retweetedID sql.NullString
		var photosJSON, videosJSON, gifsJSON, hashtagsJSON, mentionsJSON, urlsJSON sql.NullString
		var lang, source, place sql.NullString
		err := rows.Scan(
			&t.ID, &convID, &t.Text, &html,
			&t.Username, &userID, &name, &permURL,
			&t.IsRetweet, &t.IsReply, &t.IsQuote, &t.IsPin,
			&replyTo, &replyToUser, &quotedID, &retweetedID,
			&t.Likes, &t.Retweets, &t.Replies, &t.Views, &t.Bookmarks, &t.Quotes,
			&photosJSON, &videosJSON, &gifsJSON, &hashtagsJSON, &mentionsJSON, &urlsJSON,
			&t.Sensitive, &lang, &source, &place, &t.IsEdited,
			&t.PostedAt, &t.FetchedAt,
		)
		if err != nil {
			return tweets, err
		}
		t.ConversationID = convID.String
		t.HTML = html.String
		t.UserID = userID.String
		t.Name = name.String
		t.PermanentURL = permURL.String
		t.ReplyToID = replyTo.String
		t.ReplyToUser = replyToUser.String
		t.QuotedID = quotedID.String
		t.RetweetedID = retweetedID.String
		t.Language = lang.String
		t.Source = source.String
		t.Place = place.String
		t.Photos = parseJSONArray(photosJSON.String)
		t.Videos = parseJSONArray(videosJSON.String)
		t.GIFs = parseJSONArray(gifsJSON.String)
		t.Hashtags = parseJSONArray(hashtagsJSON.String)
		t.Mentions = parseJSONArray(mentionsJSON.String)
		t.URLs = parseJSONArray(urlsJSON.String)
		tweets = append(tweets, t)
	}
	return tweets, rows.Err()
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func jsonArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(arr)
	return string(data)
}

func parseJSONArray(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var arr []string
	json.Unmarshal([]byte(s), &arr)
	return arr
}
