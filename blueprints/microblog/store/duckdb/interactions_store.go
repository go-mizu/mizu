package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/microblog/pkg/ulid"
)

// InteractionsStore implements interactions.Store using DuckDB.
type InteractionsStore struct {
	db *sql.DB
}

// NewInteractionsStore creates a new interactions store.
func NewInteractionsStore(db *sql.DB) *InteractionsStore {
	return &InteractionsStore{db: db}
}

func (s *InteractionsStore) Like(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM likes WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO likes (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, postID, time.Now())
	return err == nil, err
}

func (s *InteractionsStore) Unlike(ctx context.Context, accountID, postID string) (bool, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM likes WHERE account_id = $1 AND post_id = $2", accountID, postID)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

func (s *InteractionsStore) CheckLiked(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM likes WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	return exists, err
}

func (s *InteractionsStore) GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id FROM likes
		WHERE post_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, postID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *InteractionsStore) Repost(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM reposts WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO reposts (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, postID, time.Now())
	return err == nil, err
}

func (s *InteractionsStore) Unrepost(ctx context.Context, accountID, postID string) (bool, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM reposts WHERE account_id = $1 AND post_id = $2", accountID, postID)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

func (s *InteractionsStore) CheckReposted(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM reposts WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	return exists, err
}

func (s *InteractionsStore) GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id FROM reposts
		WHERE post_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, postID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *InteractionsStore) Bookmark(ctx context.Context, accountID, postID string) error {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM bookmarks WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO bookmarks (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, ulid.New(), accountID, postID, time.Now())
	return err
}

func (s *InteractionsStore) Unbookmark(ctx context.Context, accountID, postID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM bookmarks WHERE account_id = $1 AND post_id = $2", accountID, postID)
	return err
}

func (s *InteractionsStore) VotePoll(ctx context.Context, accountID, pollID string, choices []int) error {
	for _, choice := range choices {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO poll_votes (id, poll_id, account_id, choice, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, ulid.New(), pollID, accountID, choice, time.Now())
		if err != nil {
			return err
		}
	}

	_, _ = s.db.ExecContext(ctx, "UPDATE polls SET voters_count = voters_count + 1 WHERE id = $1", pollID)

	for _, choice := range choices {
		_, _ = s.db.ExecContext(ctx, `
			UPDATE polls
			SET options = json_set(options, '$[' || $1 || '].votes_count',
				COALESCE(json_extract(options, '$[' || $1 || '].votes_count')::INT, 0) + 1)
			WHERE id = $2
		`, choice, pollID)
	}

	return nil
}

func (s *InteractionsStore) GetPollInfo(ctx context.Context, pollID string) (*string, bool, error) {
	var expiresAt sql.NullTime
	var multiple bool
	err := s.db.QueryRowContext(ctx, "SELECT expires_at, multiple FROM polls WHERE id = $1", pollID).Scan(&expiresAt, &multiple)
	if err != nil {
		return nil, false, err
	}
	if expiresAt.Valid {
		str := expiresAt.Time.Format(time.RFC3339)
		return &str, multiple, nil
	}
	return nil, multiple, nil
}

func (s *InteractionsStore) CheckVoted(ctx context.Context, pollID, accountID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM poll_votes WHERE poll_id = $1 AND account_id = $2)", pollID, accountID).Scan(&exists)
	return exists, err
}

func (s *InteractionsStore) GetPostOwner(ctx context.Context, postID string) (string, error) {
	var ownerID string
	err := s.db.QueryRowContext(ctx, "SELECT account_id FROM posts WHERE id = $1", postID).Scan(&ownerID)
	return ownerID, err
}

func (s *InteractionsStore) IncrementLikes(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET likes_count = likes_count + 1 WHERE id = $1", postID)
	return err
}

func (s *InteractionsStore) DecrementLikes(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET likes_count = likes_count - 1 WHERE id = $1 AND likes_count > 0", postID)
	return err
}

func (s *InteractionsStore) IncrementReposts(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET reposts_count = reposts_count + 1 WHERE id = $1", postID)
	return err
}

func (s *InteractionsStore) DecrementReposts(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET reposts_count = reposts_count - 1 WHERE id = $1 AND reposts_count > 0", postID)
	return err
}

func (s *InteractionsStore) CreateNotification(ctx context.Context, accountID, actorID, notifType, postID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, account_id, type, actor_id, post_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ulid.New(), accountID, notifType, actorID, postID, time.Now())
	return err
}
