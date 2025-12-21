package posts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/pkg/text"
	"github.com/go-mizu/blueprints/microblog/pkg/ulid"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

var (
	ErrNotFound       = errors.New("post not found")
	ErrNotAuthorized  = errors.New("not authorized")
	ErrContentTooLong = errors.New("content too long")
	ErrEmptyContent   = errors.New("content is required")

	MaxContentLength = 500
)

// Service handles post operations.
type Service struct {
	store    *duckdb.Store
	accounts *accounts.Service
}

// NewService creates a new posts service.
func NewService(store *duckdb.Store, accounts *accounts.Service) *Service {
	return &Service{store: store, accounts: accounts}
}

// Create creates a new post.
func (s *Service) Create(ctx context.Context, accountID string, in *CreateIn) (*Post, error) {
	// Validate content
	contentLen := text.CharCount(in.Content)
	if contentLen == 0 && len(in.MediaIDs) == 0 && in.Poll == nil {
		return nil, ErrEmptyContent
	}
	if contentLen > MaxContentLength {
		return nil, ErrContentTooLong
	}

	// Set defaults
	visibility := in.Visibility
	if visibility == "" {
		visibility = VisibilityPublic
	}

	id := ulid.New()
	now := time.Now()

	// Determine thread_id
	threadID := id
	if in.ReplyToID != "" {
		// Get parent's thread_id
		var parentThreadID sql.NullString
		err := s.store.QueryRow(ctx, "SELECT thread_id FROM posts WHERE id = $1", in.ReplyToID).Scan(&parentThreadID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("posts: get parent thread: %w", err)
		}
		if parentThreadID.Valid {
			threadID = parentThreadID.String
		} else {
			threadID = in.ReplyToID
		}
	}

	// Insert post
	_, err := s.store.Exec(ctx, `
		INSERT INTO posts (id, account_id, content, content_warning, visibility, reply_to_id, thread_id, quote_of_id, language, sensitive, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, id, accountID, in.Content, in.ContentWarning, visibility, nullString(in.ReplyToID), threadID, nullString(in.QuoteOfID), in.Language, in.Sensitive, now)
	if err != nil {
		return nil, fmt.Errorf("posts: insert: %w", err)
	}

	// Update parent's replies_count
	if in.ReplyToID != "" {
		_, _ = s.store.Exec(ctx, "UPDATE posts SET replies_count = replies_count + 1 WHERE id = $1", in.ReplyToID)
	}

	// Update quote's reposts_count
	if in.QuoteOfID != "" {
		_, _ = s.store.Exec(ctx, "UPDATE posts SET reposts_count = reposts_count + 1 WHERE id = $1", in.QuoteOfID)
	}

	// Extract and save hashtags
	hashtags := text.ExtractHashtags(in.Content)
	for _, tag := range hashtags {
		s.saveHashtag(ctx, id, tag)
	}

	// Extract and save mentions
	mentions := text.ExtractMentions(in.Content)
	for _, username := range mentions {
		s.saveMention(ctx, id, username)
	}

	// Create poll if provided
	if in.Poll != nil && len(in.Poll.Options) >= 2 {
		s.createPoll(ctx, id, in.Poll)
	}

	return s.GetByID(ctx, id, accountID)
}

// GetByID retrieves a post by ID.
func (s *Service) GetByID(ctx context.Context, id, viewerID string) (*Post, error) {
	post, err := s.scanPost(s.store.QueryRow(ctx, `
		SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id,
		       quote_of_id, language, sensitive, edited_at, created_at,
		       likes_count, reposts_count, replies_count
		FROM posts WHERE id = $1
	`, id))
	if err != nil {
		return nil, err
	}

	// Load account
	post.Account, _ = s.accounts.GetByID(ctx, post.AccountID)

	// Load viewer state
	if viewerID != "" {
		s.loadViewerState(ctx, post, viewerID)
	}

	// Load media
	post.Media, _ = s.getMedia(ctx, id)

	// Load poll
	post.Poll, _ = s.getPoll(ctx, id, viewerID)

	// Load hashtags
	post.Hashtags = text.ExtractHashtags(post.Content)

	// Load mentions
	post.Mentions = text.ExtractMentions(post.Content)

	return post, nil
}

func (s *Service) scanPost(row *sql.Row) (*Post, error) {
	var p Post
	var cw, replyToID, threadID, quoteOfID, language sql.NullString
	var editedAt sql.NullTime

	err := row.Scan(
		&p.ID, &p.AccountID, &p.Content, &cw, &p.Visibility, &replyToID, &threadID,
		&quoteOfID, &language, &p.Sensitive, &editedAt, &p.CreatedAt,
		&p.LikesCount, &p.RepostsCount, &p.RepliesCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("posts: scan: %w", err)
	}

	p.ContentWarning = cw.String
	p.ReplyToID = replyToID.String
	p.ThreadID = threadID.String
	p.QuoteOfID = quoteOfID.String
	p.Language = language.String
	if editedAt.Valid {
		p.EditedAt = &editedAt.Time
	}

	return &p, nil
}

// Update updates a post's content.
func (s *Service) Update(ctx context.Context, id, accountID string, in *UpdateIn) (*Post, error) {
	// Verify ownership
	var ownerID string
	err := s.store.QueryRow(ctx, "SELECT account_id FROM posts WHERE id = $1", id).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("posts: get owner: %w", err)
	}
	if ownerID != accountID {
		return nil, ErrNotAuthorized
	}

	// Get current content for history
	post, err := s.GetByID(ctx, id, "")
	if err != nil {
		return nil, err
	}

	// Save to edit history
	_, _ = s.store.Exec(ctx, `
		INSERT INTO edit_history (id, post_id, content, content_warning, sensitive, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ulid.New(), id, post.Content, post.ContentWarning, post.Sensitive, time.Now())

	// Build update query
	var sets []string
	var args []any
	argIdx := 1

	if in.Content != nil {
		if text.CharCount(*in.Content) > MaxContentLength {
			return nil, ErrContentTooLong
		}
		sets = append(sets, fmt.Sprintf("content = $%d", argIdx))
		args = append(args, *in.Content)
		argIdx++
	}
	if in.ContentWarning != nil {
		sets = append(sets, fmt.Sprintf("content_warning = $%d", argIdx))
		args = append(args, *in.ContentWarning)
		argIdx++
	}
	if in.Sensitive != nil {
		sets = append(sets, fmt.Sprintf("sensitive = $%d", argIdx))
		args = append(args, *in.Sensitive)
		argIdx++
	}

	if len(sets) == 0 {
		return s.GetByID(ctx, id, accountID)
	}

	sets = append(sets, fmt.Sprintf("edited_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE posts SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)

	_, err = s.store.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("posts: update: %w", err)
	}

	// Update hashtags if content changed
	if in.Content != nil {
		// Delete old hashtags
		_, _ = s.store.Exec(ctx, "DELETE FROM post_hashtags WHERE post_id = $1", id)
		// Add new hashtags
		for _, tag := range text.ExtractHashtags(*in.Content) {
			s.saveHashtag(ctx, id, tag)
		}

		// Delete old mentions
		_, _ = s.store.Exec(ctx, "DELETE FROM mentions WHERE post_id = $1", id)
		// Add new mentions
		for _, username := range text.ExtractMentions(*in.Content) {
			s.saveMention(ctx, id, username)
		}
	}

	return s.GetByID(ctx, id, accountID)
}

// Delete deletes a post.
func (s *Service) Delete(ctx context.Context, id, accountID string) error {
	// Verify ownership
	var ownerID string
	var replyToID sql.NullString
	err := s.store.QueryRow(ctx, "SELECT account_id, reply_to_id FROM posts WHERE id = $1", id).Scan(&ownerID, &replyToID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("posts: get owner: %w", err)
	}
	if ownerID != accountID {
		return ErrNotAuthorized
	}

	// Delete post (cascades to media, mentions, etc.)
	_, err = s.store.Exec(ctx, "DELETE FROM posts WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("posts: delete: %w", err)
	}

	// Update parent's replies_count
	if replyToID.Valid {
		_, _ = s.store.Exec(ctx, "UPDATE posts SET replies_count = replies_count - 1 WHERE id = $1 AND replies_count > 0", replyToID.String)
	}

	return nil
}

// GetThread returns the full thread context for a post.
func (s *Service) GetThread(ctx context.Context, id, viewerID string) (*ThreadContext, error) {
	post, err := s.GetByID(ctx, id, viewerID)
	if err != nil {
		return nil, err
	}

	// Get ancestors (parent chain)
	var ancestors []*Post
	currentID := post.ReplyToID
	for currentID != "" {
		ancestor, err := s.GetByID(ctx, currentID, viewerID)
		if err != nil {
			break
		}
		ancestors = append([]*Post{ancestor}, ancestors...)
		currentID = ancestor.ReplyToID
	}

	// Get descendants (all replies)
	descendants, _ := s.getDescendants(ctx, id, viewerID, 50)

	return &ThreadContext{
		Ancestors:   ancestors,
		Post:        post,
		Descendants: descendants,
	}, nil
}

func (s *Service) getDescendants(ctx context.Context, postID, viewerID string, limit int) ([]*Post, error) {
	rows, err := s.store.Query(ctx, `
		WITH RECURSIVE descendants AS (
			SELECT id, account_id, content, content_warning, visibility, reply_to_id, thread_id,
			       quote_of_id, language, sensitive, edited_at, created_at,
			       likes_count, reposts_count, replies_count, 1 as depth
			FROM posts WHERE reply_to_id = $1
			UNION ALL
			SELECT p.id, p.account_id, p.content, p.content_warning, p.visibility, p.reply_to_id, p.thread_id,
			       p.quote_of_id, p.language, p.sensitive, p.edited_at, p.created_at,
			       p.likes_count, p.reposts_count, p.replies_count, d.depth + 1
			FROM posts p
			JOIN descendants d ON p.reply_to_id = d.id
			WHERE d.depth < 10
		)
		SELECT * FROM descendants ORDER BY created_at ASC LIMIT $2
	`, postID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPosts(ctx, rows, viewerID)
}

func (s *Service) scanPosts(ctx context.Context, rows *sql.Rows, viewerID string) ([]*Post, error) {
	var posts []*Post
	for rows.Next() {
		var p Post
		var cw, replyToID, threadID, quoteOfID, language sql.NullString
		var editedAt sql.NullTime
		var depth int

		err := rows.Scan(
			&p.ID, &p.AccountID, &p.Content, &cw, &p.Visibility, &replyToID, &threadID,
			&quoteOfID, &language, &p.Sensitive, &editedAt, &p.CreatedAt,
			&p.LikesCount, &p.RepostsCount, &p.RepliesCount, &depth,
		)
		if err != nil {
			continue
		}

		p.ContentWarning = cw.String
		p.ReplyToID = replyToID.String
		p.ThreadID = threadID.String
		p.QuoteOfID = quoteOfID.String
		p.Language = language.String
		if editedAt.Valid {
			p.EditedAt = &editedAt.Time
		}

		p.Account, _ = s.accounts.GetByID(ctx, p.AccountID)
		if viewerID != "" {
			s.loadViewerState(ctx, &p, viewerID)
		}

		posts = append(posts, &p)
	}
	return posts, nil
}

func (s *Service) loadViewerState(ctx context.Context, post *Post, viewerID string) {
	var exists bool

	// Check liked
	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM likes WHERE account_id = $1 AND post_id = $2)", viewerID, post.ID).Scan(&exists)
	post.Liked = exists

	// Check reposted
	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM reposts WHERE account_id = $1 AND post_id = $2)", viewerID, post.ID).Scan(&exists)
	post.Reposted = exists

	// Check bookmarked
	_ = s.store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bookmarks WHERE account_id = $1 AND post_id = $2)", viewerID, post.ID).Scan(&exists)
	post.Bookmarked = exists
}

func (s *Service) getMedia(ctx context.Context, postID string) ([]*Media, error) {
	rows, err := s.store.Query(ctx, `
		SELECT id, type, url, preview_url, alt_text, width, height, position
		FROM media WHERE post_id = $1 ORDER BY position
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []*Media
	for rows.Next() {
		var m Media
		var previewURL, altText sql.NullString
		var width, height sql.NullInt64

		if err := rows.Scan(&m.ID, &m.Type, &m.URL, &previewURL, &altText, &width, &height, &m.Position); err != nil {
			continue
		}

		m.PreviewURL = previewURL.String
		m.AltText = altText.String
		m.Width = int(width.Int64)
		m.Height = int(height.Int64)
		media = append(media, &m)
	}

	return media, nil
}

func (s *Service) getPoll(ctx context.Context, postID, viewerID string) (*Poll, error) {
	var poll Poll
	var optionsJSON string
	var expiresAt sql.NullTime

	err := s.store.QueryRow(ctx, `
		SELECT id, options, multiple, expires_at, voters_count
		FROM polls WHERE post_id = $1
	`, postID).Scan(&poll.ID, &optionsJSON, &poll.Multiple, &expiresAt, &poll.VotersCount)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(optionsJSON), &poll.Options)

	if expiresAt.Valid {
		poll.ExpiresAt = &expiresAt.Time
		poll.Expired = time.Now().After(expiresAt.Time)
	}

	// Check if viewer voted
	if viewerID != "" {
		rows, err := s.store.Query(ctx, "SELECT choice FROM poll_votes WHERE poll_id = $1 AND account_id = $2", poll.ID, viewerID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var choice int
				if rows.Scan(&choice) == nil {
					poll.OwnVotes = append(poll.OwnVotes, choice)
					poll.Voted = true
				}
			}
		}
	}

	return &poll, nil
}

func (s *Service) createPoll(ctx context.Context, postID string, in *CreatePollIn) error {
	pollID := ulid.New()
	options := make([]PollOption, len(in.Options))
	for i, opt := range in.Options {
		options[i] = PollOption{Title: opt, VotesCount: 0}
	}
	optionsJSON, _ := json.Marshal(options)

	var expiresAt *time.Time
	if in.ExpiresIn > 0 {
		t := time.Now().Add(in.ExpiresIn)
		expiresAt = &t
	}

	_, err := s.store.Exec(ctx, `
		INSERT INTO polls (id, post_id, options, multiple, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, pollID, postID, string(optionsJSON), in.Multiple, expiresAt, time.Now())

	if err == nil {
		_, _ = s.store.Exec(ctx, "UPDATE posts SET poll_id = $1 WHERE id = $2", pollID, postID)
	}

	return err
}

func (s *Service) saveHashtag(ctx context.Context, postID, tag string) {
	// Upsert hashtag
	var hashtagID string
	err := s.store.QueryRow(ctx, "SELECT id FROM hashtags WHERE name = $1", tag).Scan(&hashtagID)
	if errors.Is(err, sql.ErrNoRows) {
		hashtagID = ulid.New()
		_, _ = s.store.Exec(ctx, `
			INSERT INTO hashtags (id, name, posts_count, last_used_at, created_at)
			VALUES ($1, $2, 1, $3, $3)
		`, hashtagID, tag, time.Now())
	} else if err == nil {
		_, _ = s.store.Exec(ctx, `
			UPDATE hashtags SET posts_count = posts_count + 1, last_used_at = $1 WHERE id = $2
		`, time.Now(), hashtagID)
	}

	// Link to post
	if hashtagID != "" {
		_, _ = s.store.Exec(ctx, `
			INSERT INTO post_hashtags (post_id, hashtag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, postID, hashtagID)
	}
}

func (s *Service) saveMention(ctx context.Context, postID, username string) {
	// Find account
	var accountID string
	err := s.store.QueryRow(ctx, "SELECT id FROM accounts WHERE LOWER(username) = LOWER($1)", username).Scan(&accountID)
	if err != nil {
		return
	}

	// Create mention
	_, _ = s.store.Exec(ctx, `
		INSERT INTO mentions (id, post_id, account_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), postID, accountID, time.Now())
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
