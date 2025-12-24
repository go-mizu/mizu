package interactions

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/social/pkg/ulid"
)

// Service implements the interactions API.
type Service struct {
	store Store
}

// NewService creates a new interactions service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Like likes a post.
func (s *Service) Like(ctx context.Context, accountID, postID string) error {
	exists, err := s.store.ExistsLike(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyLiked
	}

	like := &Like{
		ID:        ulid.New(),
		AccountID: accountID,
		PostID:    postID,
		CreatedAt: time.Now(),
	}

	if err := s.store.InsertLike(ctx, like); err != nil {
		return err
	}

	return s.store.IncrementLikesCount(ctx, postID)
}

// Unlike unlikes a post.
func (s *Service) Unlike(ctx context.Context, accountID, postID string) error {
	exists, err := s.store.ExistsLike(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotLiked
	}

	if err := s.store.DeleteLike(ctx, accountID, postID); err != nil {
		return err
	}

	return s.store.DecrementLikesCount(ctx, postID)
}

// GetLikedBy returns account IDs that liked a post.
func (s *Service) GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	return s.store.GetLikedBy(ctx, postID, limit, offset)
}

// GetLikedPosts returns post IDs liked by an account.
func (s *Service) GetLikedPosts(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	return s.store.GetLikedPosts(ctx, accountID, limit, offset)
}

// Repost reposts a post.
func (s *Service) Repost(ctx context.Context, accountID, postID string) error {
	exists, err := s.store.ExistsRepost(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyReposted
	}

	repost := &Repost{
		ID:        ulid.New(),
		AccountID: accountID,
		PostID:    postID,
		CreatedAt: time.Now(),
	}

	if err := s.store.InsertRepost(ctx, repost); err != nil {
		return err
	}

	return s.store.IncrementRepostsCount(ctx, postID)
}

// Unrepost removes a repost.
func (s *Service) Unrepost(ctx context.Context, accountID, postID string) error {
	exists, err := s.store.ExistsRepost(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotReposted
	}

	if err := s.store.DeleteRepost(ctx, accountID, postID); err != nil {
		return err
	}

	return s.store.DecrementRepostsCount(ctx, postID)
}

// GetRepostedBy returns account IDs that reposted a post.
func (s *Service) GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	return s.store.GetRepostedBy(ctx, postID, limit, offset)
}

// Bookmark bookmarks a post.
func (s *Service) Bookmark(ctx context.Context, accountID, postID string) error {
	exists, err := s.store.ExistsBookmark(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyBookmarked
	}

	bookmark := &Bookmark{
		ID:        ulid.New(),
		AccountID: accountID,
		PostID:    postID,
		CreatedAt: time.Now(),
	}

	return s.store.InsertBookmark(ctx, bookmark)
}

// Unbookmark removes a bookmark.
func (s *Service) Unbookmark(ctx context.Context, accountID, postID string) error {
	exists, err := s.store.ExistsBookmark(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotBookmarked
	}

	return s.store.DeleteBookmark(ctx, accountID, postID)
}

// GetBookmarkedPosts returns post IDs bookmarked by an account.
func (s *Service) GetBookmarkedPosts(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	return s.store.GetBookmarkedPosts(ctx, accountID, limit, offset)
}

// GetPostState returns the viewer's interaction state with a post.
func (s *Service) GetPostState(ctx context.Context, accountID, postID string) (*PostState, error) {
	return s.store.GetPostState(ctx, accountID, postID)
}

// GetPostStates returns viewer states for multiple posts.
func (s *Service) GetPostStates(ctx context.Context, accountID string, postIDs []string) (map[string]*PostState, error) {
	return s.store.GetPostStates(ctx, accountID, postIDs)
}
