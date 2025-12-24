package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/pkg/ulid"
)

// PostsStore implements posts.Store.
type PostsStore struct {
	db *sql.DB
}

// NewPostsStore creates a new posts store.
func NewPostsStore(db *sql.DB) *PostsStore {
	return &PostsStore{db: db}
}

// Insert inserts a new post.
func (s *PostsStore) Insert(ctx context.Context, p *posts.Post) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO posts (id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, created_at, likes_count, reposts_count, replies_count, quotes_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, p.ID, p.AccountID, p.Content, p.ContentWarning, p.Visibility, nullString(p.ReplyToID), nullString(p.ThreadID), nullString(p.QuoteOfID), p.Language, p.Sensitive, p.CreatedAt, p.LikesCount, p.RepostsCount, p.RepliesCount, p.QuotesCount)
	return err
}

// GetByID retrieves a post by ID.
func (s *PostsStore) GetByID(ctx context.Context, id string) (*posts.Post, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, edited_at, created_at, likes_count, reposts_count, replies_count, quotes_count
		FROM posts WHERE id = $1
	`, id)
	return s.scanPost(row)
}

// GetByIDs retrieves multiple posts by IDs.
func (s *PostsStore) GetByIDs(ctx context.Context, ids []string) ([]*posts.Post, error) {
	if len(ids) == 0 {
		return []*posts.Post{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, edited_at, created_at, likes_count, reposts_count, replies_count, quotes_count
		FROM posts WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := s.db.QueryContext(ctx, query, args...)
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

// Update updates a post.
func (s *PostsStore) Update(ctx context.Context, id string, in *posts.UpdateIn) error {
	sets := []string{"edited_at = $1"}
	args := []interface{}{time.Now()}
	argNum := 2

	if in.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", argNum))
		args = append(args, *in.Content)
		argNum++
	}
	if in.ContentWarning != nil {
		sets = append(sets, fmt.Sprintf("content_warning = $%d", argNum))
		args = append(args, *in.ContentWarning)
		argNum++
	}
	if in.Sensitive != nil {
		sets = append(sets, fmt.Sprintf("sensitive = $%d", argNum))
		args = append(args, *in.Sensitive)
		argNum++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE posts SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a post.
func (s *PostsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM posts WHERE id = $1", id)
	return err
}

// List lists posts with options.
func (s *PostsStore) List(ctx context.Context, opts posts.ListOpts) ([]*posts.Post, error) {
	query := `
		SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, edited_at, created_at, likes_count, reposts_count, replies_count, quotes_count
		FROM posts WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if opts.AccountID != "" {
		query += fmt.Sprintf(" AND account_id = $%d", argNum)
		args = append(args, opts.AccountID)
		argNum++
	}

	if opts.ExcludeReplies {
		query += " AND reply_to_id IS NULL"
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, opts.Limit)
		argNum++
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
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

// GetReplies gets replies to a post.
func (s *PostsStore) GetReplies(ctx context.Context, postID string, limit, offset int) ([]*posts.Post, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, edited_at, created_at, likes_count, reposts_count, replies_count, quotes_count
		FROM posts WHERE reply_to_id = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3
	`, postID, limit, offset)
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

// GetAncestors gets ancestor posts in a thread.
func (s *PostsStore) GetAncestors(ctx context.Context, postID string) ([]*posts.Post, error) {
	var ancestors []*posts.Post
	currentID := postID

	for {
		post, err := s.GetByID(ctx, currentID)
		if err != nil {
			break
		}
		if post.ReplyToID == "" {
			break
		}
		parent, err := s.GetByID(ctx, post.ReplyToID)
		if err != nil {
			break
		}
		ancestors = append([]*posts.Post{parent}, ancestors...)
		currentID = parent.ID
	}

	return ancestors, nil
}

// GetDescendants gets descendant posts in a thread.
func (s *PostsStore) GetDescendants(ctx context.Context, postID string, limit int) ([]*posts.Post, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, edited_at, created_at, likes_count, reposts_count, replies_count, quotes_count
		FROM posts WHERE thread_id = $1 OR reply_to_id = $1 ORDER BY created_at ASC LIMIT $2
	`, postID, limit)
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

// IncrementRepliesCount increments the replies count.
func (s *PostsStore) IncrementRepliesCount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET replies_count = replies_count + 1 WHERE id = $1", id)
	return err
}

// DecrementRepliesCount decrements the replies count.
func (s *PostsStore) DecrementRepliesCount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET replies_count = GREATEST(0, replies_count - 1) WHERE id = $1", id)
	return err
}

// IncrementQuotesCount increments the quotes count.
func (s *PostsStore) IncrementQuotesCount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET quotes_count = quotes_count + 1 WHERE id = $1", id)
	return err
}

// InsertMedia inserts a media attachment.
func (s *PostsStore) InsertMedia(ctx context.Context, m *posts.Media) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO media (id, post_id, type, url, preview_url, alt_text, width, height, position, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, m.ID, m.PostID, m.Type, m.URL, m.PreviewURL, m.AltText, m.Width, m.Height, m.Position, time.Now())
	return err
}

// GetMediaByPostID retrieves media for a post.
func (s *PostsStore) GetMediaByPostID(ctx context.Context, postID string) ([]*posts.Media, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, post_id, type, url, preview_url, alt_text, width, height, position
		FROM media WHERE post_id = $1 ORDER BY position
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []*posts.Media
	for rows.Next() {
		var m posts.Media
		var previewURL, altText sql.NullString
		err := rows.Scan(&m.ID, &m.PostID, &m.Type, &m.URL, &previewURL, &altText, &m.Width, &m.Height, &m.Position)
		if err != nil {
			return nil, err
		}
		m.PreviewURL = previewURL.String
		m.AltText = altText.String
		media = append(media, &m)
	}
	return media, rows.Err()
}

// DeleteMediaByPostID deletes media for a post.
func (s *PostsStore) DeleteMediaByPostID(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM media WHERE post_id = $1", postID)
	return err
}

// UpsertHashtag upserts a hashtag and returns its ID.
func (s *PostsStore) UpsertHashtag(ctx context.Context, name string) (string, error) {
	name = strings.ToLower(name)

	var id string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM hashtags WHERE name = $1", name).Scan(&id)
	if err == nil {
		_, _ = s.db.ExecContext(ctx, "UPDATE hashtags SET posts_count = posts_count + 1, last_used_at = $1 WHERE id = $2", time.Now(), id)
		return id, nil
	}

	id = ulid.New()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO hashtags (id, name, posts_count, last_used_at, created_at)
		VALUES ($1, $2, 1, $3, $3)
	`, id, name, time.Now())
	return id, err
}

// LinkPostHashtag links a post to a hashtag.
func (s *PostsStore) LinkPostHashtag(ctx context.Context, postID, hashtagID string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO post_hashtags (post_id, hashtag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", postID, hashtagID)
	return err
}

// GetHashtagsByPostID gets hashtags for a post.
func (s *PostsStore) GetHashtagsByPostID(ctx context.Context, postID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT h.name FROM hashtags h
		JOIN post_hashtags ph ON h.id = ph.hashtag_id
		WHERE ph.post_id = $1
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, rows.Err()
}

// InsertMention inserts a mention.
func (s *PostsStore) InsertMention(ctx context.Context, postID, accountID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mentions (id, post_id, account_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), postID, accountID, time.Now())
	return err
}

// GetMentionsByPostID gets account IDs mentioned in a post.
func (s *PostsStore) GetMentionsByPostID(ctx context.Context, postID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT account_id FROM mentions WHERE post_id = $1", postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// InsertEditHistory inserts an edit history record.
func (s *PostsStore) InsertEditHistory(ctx context.Context, postID, content, contentWarning string, sensitive bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO edit_history (id, post_id, content, content_warning, sensitive, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ulid.New(), postID, content, contentWarning, sensitive, time.Now())
	return err
}

func (s *PostsStore) scanPost(row *sql.Row) (*posts.Post, error) {
	var p posts.Post
	var contentWarning, replyToID, threadID, quoteOfID, language sql.NullString
	var editedAt sql.NullTime

	err := row.Scan(&p.ID, &p.AccountID, &p.Content, &contentWarning, &p.Visibility, &replyToID, &threadID, &quoteOfID, &language, &p.Sensitive, &editedAt, &p.CreatedAt, &p.LikesCount, &p.RepostsCount, &p.RepliesCount, &p.QuotesCount)
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

func (s *PostsStore) scanPostRow(rows *sql.Rows) (*posts.Post, error) {
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

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
