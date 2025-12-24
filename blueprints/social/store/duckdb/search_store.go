package duckdb

import (
	"context"
	"database/sql"
	"strings"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/feature/search"
)

// SearchStore implements search.Store.
type SearchStore struct {
	db *sql.DB
}

// NewSearchStore creates a new search store.
func NewSearchStore(db *sql.DB) *SearchStore {
	return &SearchStore{db: db}
}

// SearchAccounts searches for accounts.
func (s *SearchStore) SearchAccounts(ctx context.Context, query string, limit, offset int) ([]*accounts.Account, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, location, website, fields, verified, admin, suspended, private, discoverable, created_at, updated_at
		FROM accounts
		WHERE discoverable = TRUE AND suspended = FALSE
		AND (LOWER(username) LIKE $1 OR LOWER(display_name) LIKE $1)
		ORDER BY
			CASE WHEN LOWER(username) = LOWER($2) THEN 0 ELSE 1 END,
			created_at DESC
		LIMIT $3 OFFSET $4
	`, pattern, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accs []*accounts.Account
	for rows.Next() {
		a, err := s.scanAccountRow(rows)
		if err != nil {
			return nil, err
		}
		accs = append(accs, a)
	}
	return accs, rows.Err()
}

// SearchPosts searches for posts.
func (s *SearchStore) SearchPosts(ctx context.Context, query string, limit, offset int, minLikes, minReposts int, hasMedia bool) ([]*posts.Post, error) {
	pattern := "%" + strings.ToLower(query) + "%"

	q := `
		SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, edited_at, created_at, likes_count, reposts_count, replies_count, quotes_count
		FROM posts
		WHERE visibility = 'public' AND LOWER(content) LIKE $1
	`
	args := []interface{}{pattern}
	argNum := 2

	if minLikes > 0 {
		q += " AND likes_count >= $" + string(rune('0'+argNum))
		args = append(args, minLikes)
		argNum++
	}
	if minReposts > 0 {
		q += " AND reposts_count >= $" + string(rune('0'+argNum))
		args = append(args, minReposts)
		argNum++
	}
	if hasMedia {
		q += " AND EXISTS (SELECT 1 FROM media m WHERE m.post_id = posts.id)"
	}

	q += " ORDER BY created_at DESC LIMIT $" + string(rune('0'+argNum)) + " OFFSET $" + string(rune('0'+argNum+1))
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ps []*posts.Post
	for rows.Next() {
		p, err := s.scanPostRow(rows)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, rows.Err()
}

// SearchHashtags searches for hashtags.
func (s *SearchStore) SearchHashtags(ctx context.Context, query string, limit int) ([]*search.Hashtag, error) {
	pattern := strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, posts_count FROM hashtags
		WHERE LOWER(name) LIKE $1
		ORDER BY posts_count DESC, name ASC
		LIMIT $2
	`, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*search.Hashtag
	for rows.Next() {
		var t search.Hashtag
		if err := rows.Scan(&t.Name, &t.PostsCount); err != nil {
			return nil, err
		}
		tags = append(tags, &t)
	}
	return tags, rows.Err()
}

// SuggestHashtags returns hashtag suggestions.
func (s *SearchStore) SuggestHashtags(ctx context.Context, prefix string, limit int) ([]string, error) {
	pattern := strings.ToLower(prefix) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT name FROM hashtags
		WHERE LOWER(name) LIKE $1
		ORDER BY posts_count DESC
		LIMIT $2
	`, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		suggestions = append(suggestions, name)
	}
	return suggestions, rows.Err()
}

func (s *SearchStore) scanAccountRow(rows *sql.Rows) (*accounts.Account, error) {
	var a accounts.Account
	var displayName, email, bio, avatarURL, headerURL, location, website sql.NullString
	var fieldsJSON string

	err := rows.Scan(&a.ID, &a.Username, &displayName, &email, &bio, &avatarURL, &headerURL, &location, &website, &fieldsJSON, &a.Verified, &a.Admin, &a.Suspended, &a.Private, &a.Discoverable, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}

	a.DisplayName = displayName.String
	a.Email = email.String
	a.Bio = bio.String
	a.AvatarURL = avatarURL.String
	a.HeaderURL = headerURL.String
	a.Location = location.String
	a.Website = website.String

	return &a, nil
}

func (s *SearchStore) scanPostRow(rows *sql.Rows) (*posts.Post, error) {
	var p posts.Post
	var contentWarning, replyToID, threadID, quoteOfID, language sql.NullString
	var editedAt sql.NullTime

	err := rows.Scan(&p.ID, &p.AccountID, &p.Content, &contentWarning, &p.Visibility, &replyToID, &threadID, &quoteOfID, &language, &p.Sensitive, &editedAt, &p.CreatedAt, &p.LikesCount, &p.RepostsCount, &p.RepliesCount, &p.QuotesCount)
	if err != nil {
		return nil, err
	}

	p.ContentWarning = contentWarning.String
	p.ReplyToID = replyToID.String
	p.ThreadID = threadID.String
	p.QuoteOfID = quoteOfID.String
	p.Language = language.String
	if editedAt.Valid {
		p.EditedAt = &editedAt.Time
	}

	return &p, nil
}
