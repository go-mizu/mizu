package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/social/feature/interactions"
)

// InteractionsStore implements interactions.Store.
type InteractionsStore struct {
	db *sql.DB
}

// NewInteractionsStore creates a new interactions store.
func NewInteractionsStore(db *sql.DB) *InteractionsStore {
	return &InteractionsStore{db: db}
}

// InsertLike inserts a like.
func (s *InteractionsStore) InsertLike(ctx context.Context, l *interactions.Like) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO likes (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, l.ID, l.AccountID, l.PostID, l.CreatedAt)
	return err
}

// DeleteLike deletes a like.
func (s *InteractionsStore) DeleteLike(ctx context.Context, accountID, postID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM likes WHERE account_id = $1 AND post_id = $2", accountID, postID)
	return err
}

// ExistsLike checks if a like exists.
func (s *InteractionsStore) ExistsLike(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM likes WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	return exists, err
}

// GetLikedBy returns account IDs that liked a post.
func (s *InteractionsStore) GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id FROM likes WHERE post_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, postID, limit, offset)
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

// GetLikedPosts returns post IDs liked by an account.
func (s *InteractionsStore) GetLikedPosts(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT post_id FROM likes WHERE account_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
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

// IncrementLikesCount increments the likes count.
func (s *InteractionsStore) IncrementLikesCount(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET likes_count = likes_count + 1 WHERE id = $1", postID)
	return err
}

// DecrementLikesCount decrements the likes count.
func (s *InteractionsStore) DecrementLikesCount(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET likes_count = GREATEST(0, likes_count - 1) WHERE id = $1", postID)
	return err
}

// InsertRepost inserts a repost.
func (s *InteractionsStore) InsertRepost(ctx context.Context, r *interactions.Repost) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO reposts (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, r.ID, r.AccountID, r.PostID, r.CreatedAt)
	return err
}

// DeleteRepost deletes a repost.
func (s *InteractionsStore) DeleteRepost(ctx context.Context, accountID, postID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM reposts WHERE account_id = $1 AND post_id = $2", accountID, postID)
	return err
}

// ExistsRepost checks if a repost exists.
func (s *InteractionsStore) ExistsRepost(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM reposts WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	return exists, err
}

// GetRepostedBy returns account IDs that reposted a post.
func (s *InteractionsStore) GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id FROM reposts WHERE post_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, postID, limit, offset)
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

// IncrementRepostsCount increments the reposts count.
func (s *InteractionsStore) IncrementRepostsCount(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET reposts_count = reposts_count + 1 WHERE id = $1", postID)
	return err
}

// DecrementRepostsCount decrements the reposts count.
func (s *InteractionsStore) DecrementRepostsCount(ctx context.Context, postID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE posts SET reposts_count = GREATEST(0, reposts_count - 1) WHERE id = $1", postID)
	return err
}

// InsertBookmark inserts a bookmark.
func (s *InteractionsStore) InsertBookmark(ctx context.Context, b *interactions.Bookmark) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bookmarks (id, account_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, b.ID, b.AccountID, b.PostID, b.CreatedAt)
	return err
}

// DeleteBookmark deletes a bookmark.
func (s *InteractionsStore) DeleteBookmark(ctx context.Context, accountID, postID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM bookmarks WHERE account_id = $1 AND post_id = $2", accountID, postID)
	return err
}

// ExistsBookmark checks if a bookmark exists.
func (s *InteractionsStore) ExistsBookmark(ctx context.Context, accountID, postID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM bookmarks WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	return exists, err
}

// GetBookmarkedPosts returns post IDs bookmarked by an account.
func (s *InteractionsStore) GetBookmarkedPosts(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT post_id FROM bookmarks WHERE account_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
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

// GetPostState returns the viewer's interaction state with a post.
func (s *InteractionsStore) GetPostState(ctx context.Context, accountID, postID string) (*interactions.PostState, error) {
	state := &interactions.PostState{}

	var exists bool
	s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM likes WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	state.Liked = exists

	s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM reposts WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	state.Reposted = exists

	s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM bookmarks WHERE account_id = $1 AND post_id = $2)", accountID, postID).Scan(&exists)
	state.Bookmarked = exists

	return state, nil
}

// GetPostStates returns viewer states for multiple posts.
func (s *InteractionsStore) GetPostStates(ctx context.Context, accountID string, postIDs []string) (map[string]*interactions.PostState, error) {
	states := make(map[string]*interactions.PostState)
	for _, id := range postIDs {
		states[id] = &interactions.PostState{}
	}

	// Get likes
	rows, err := s.db.QueryContext(ctx, `
		SELECT post_id FROM likes WHERE account_id = $1 AND post_id = ANY($2)
	`, accountID, postIDs)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var postID string
			if rows.Scan(&postID) == nil {
				if state, ok := states[postID]; ok {
					state.Liked = true
				}
			}
		}
	}

	// Get reposts
	rows, err = s.db.QueryContext(ctx, `
		SELECT post_id FROM reposts WHERE account_id = $1 AND post_id = ANY($2)
	`, accountID, postIDs)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var postID string
			if rows.Scan(&postID) == nil {
				if state, ok := states[postID]; ok {
					state.Reposted = true
				}
			}
		}
	}

	// Get bookmarks
	rows, err = s.db.QueryContext(ctx, `
		SELECT post_id FROM bookmarks WHERE account_id = $1 AND post_id = ANY($2)
	`, accountID, postIDs)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var postID string
			if rows.Scan(&postID) == nil {
				if state, ok := states[postID]; ok {
					state.Bookmarked = true
				}
			}
		}
	}

	return states, nil
}
