package posts

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/pkg/text"
	"github.com/go-mizu/blueprints/microblog/pkg/ulid"
)

var (
	ErrNotFound       = errors.New("post not found")
	ErrNotAuthorized  = errors.New("not authorized")
	ErrContentTooLong = errors.New("content too long")
	ErrEmptyContent   = errors.New("content is required")

	MaxContentLength = 500
)

// Service handles post operations.
// Implements API interface.
type Service struct {
	store    Store
	accounts accounts.API
}

// NewService creates a new posts service.
func NewService(store Store, accounts accounts.API) *Service {
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
		parentThreadID, err := s.store.GetThreadID(ctx, in.ReplyToID)
		if err == nil && parentThreadID != "" {
			threadID = parentThreadID
		} else {
			threadID = in.ReplyToID
		}
	}

	post := &Post{
		ID:             id,
		AccountID:      accountID,
		Content:        in.Content,
		ContentWarning: in.ContentWarning,
		Visibility:     visibility,
		ReplyToID:      in.ReplyToID,
		ThreadID:       threadID,
		QuoteOfID:      in.QuoteOfID,
		Language:       in.Language,
		Sensitive:      in.Sensitive,
		CreatedAt:      now,
	}

	if err := s.store.Insert(ctx, post); err != nil {
		return nil, err
	}

	// Update parent's replies_count
	if in.ReplyToID != "" {
		_ = s.store.IncrementReplies(ctx, in.ReplyToID)
	}

	// Update quote's reposts_count
	if in.QuoteOfID != "" {
		_ = s.store.IncrementReposts(ctx, in.QuoteOfID)
	}

	// Extract and save hashtags
	hashtags := text.ExtractHashtags(in.Content)
	for _, tag := range hashtags {
		_ = s.store.SaveHashtag(ctx, id, tag)
	}

	// Extract and save mentions
	mentions := text.ExtractMentions(in.Content)
	for _, username := range mentions {
		_ = s.store.SaveMention(ctx, id, username)
	}

	// Create poll if provided
	if in.Poll != nil && len(in.Poll.Options) >= 2 {
		_ = s.store.CreatePoll(ctx, id, in.Poll)
	}

	return s.GetByID(ctx, id, accountID)
}

// GetByID retrieves a post by ID.
func (s *Service) GetByID(ctx context.Context, id, viewerID string) (*Post, error) {
	post, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Load account
	post.Account, _ = s.accounts.GetByID(ctx, post.AccountID)

	// Load viewer state
	if viewerID != "" {
		s.loadViewerState(ctx, post, viewerID)
	}

	// Load media
	post.Media, _ = s.store.GetMedia(ctx, id)

	// Load poll
	poll, err := s.store.GetPoll(ctx, id)
	if err == nil && poll != nil {
		post.Poll = poll
		if viewerID != "" {
			choices, _ := s.store.GetVoterChoices(ctx, poll.ID, viewerID)
			if len(choices) > 0 {
				poll.Voted = true
				poll.OwnVotes = choices
			}
		}
		if poll.ExpiresAt != nil {
			poll.Expired = time.Now().After(*poll.ExpiresAt)
		}
	}

	// Load hashtags
	post.Hashtags = text.ExtractHashtags(post.Content)

	// Load mentions
	post.Mentions = text.ExtractMentions(post.Content)

	return post, nil
}

// Update updates a post's content.
func (s *Service) Update(ctx context.Context, id, accountID string, in *UpdateIn) (*Post, error) {
	// Verify ownership
	ownerID, _, err := s.store.GetOwner(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if ownerID != accountID {
		return nil, ErrNotAuthorized
	}

	// Get current content for history
	post, err := s.GetByID(ctx, id, "")
	if err != nil {
		return nil, err
	}

	// Validate new content length
	if in.Content != nil && text.CharCount(*in.Content) > MaxContentLength {
		return nil, ErrContentTooLong
	}

	// Save to edit history
	_ = s.store.SaveEditHistory(ctx, id, post.Content, post.ContentWarning, post.Sensitive)

	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}

	// Update hashtags if content changed
	if in.Content != nil {
		_ = s.store.DeleteHashtags(ctx, id)
		for _, tag := range text.ExtractHashtags(*in.Content) {
			_ = s.store.SaveHashtag(ctx, id, tag)
		}

		_ = s.store.DeleteMentions(ctx, id)
		for _, username := range text.ExtractMentions(*in.Content) {
			_ = s.store.SaveMention(ctx, id, username)
		}
	}

	return s.GetByID(ctx, id, accountID)
}

// Delete deletes a post.
func (s *Service) Delete(ctx context.Context, id, accountID string) error {
	ownerID, replyToID, err := s.store.GetOwner(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if ownerID != accountID {
		return ErrNotAuthorized
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}

	// Update parent's replies_count
	if replyToID != "" {
		_ = s.store.DecrementReplies(ctx, replyToID)
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
	rawPosts, err := s.store.GetDescendants(ctx, postID, limit)
	if err != nil {
		return nil, err
	}

	// Enrich posts
	for _, p := range rawPosts {
		p.Account, _ = s.accounts.GetByID(ctx, p.AccountID)
		if viewerID != "" {
			s.loadViewerState(ctx, p, viewerID)
		}
	}

	return rawPosts, nil
}

func (s *Service) loadViewerState(ctx context.Context, post *Post, viewerID string) {
	post.Liked, _ = s.store.CheckLiked(ctx, viewerID, post.ID)
	post.Reposted, _ = s.store.CheckReposted(ctx, viewerID, post.ID)
	post.Bookmarked, _ = s.store.CheckBookmarked(ctx, viewerID, post.ID)
	post.IsOwner = viewerID == post.AccountID
}
